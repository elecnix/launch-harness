package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/elecnix/launch-harness/internal/config"
	"github.com/elecnix/launch-harness/internal/envconfig"
	launcher "github.com/elecnix/launch-harness/internal/launcher"
)

func TestListIntegrationsIncludesCoreHarnesses(t *testing.T) {
	specs := launcher.ListVisibleIntegrationSpecs()
	want := map[string]bool{
		"claude": false, "codex": false, "opencode": false, "pi": false,
		"cline": false, "droid": false, "qwen": false, "copilot": false,
	}
	for _, spec := range specs {
		if _, ok := want[spec.Name]; ok {
			want[spec.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected integration %q to be registered", name)
		}
	}
}

func TestLookupIntegrationAliases(t *testing.T) {
	cases := map[string]string{
		"claude":      "claude",
		"CLAUDE":      "claude",
		"codex-app":   "chatgpt",
		"copilot-cli": "copilot",
		"clawdbot":    "openclaw",
	}
	for alias, want := range cases {
		spec, err := launcher.LookupIntegrationSpec(alias)
		if err != nil {
			t.Fatalf("alias %q: %v", alias, err)
		}
		if spec.Name != want {
			t.Errorf("alias %q resolved to %q, want %q", alias, spec.Name, want)
		}
	}
	if _, err := launcher.LookupIntegrationSpec("does-not-exist"); err == nil {
		t.Error("expected error for unknown integration")
	}
}

func TestBaseURLResolution(t *testing.T) {
	t.Setenv(envconfig.EnvVar, "")
	envconfig.SetBaseURL("")
	if got := envconfig.BaseURL(); got != envconfig.DefaultURL {
		t.Errorf("default URL = %q, want %q", got, envconfig.DefaultURL)
	}

	t.Setenv(envconfig.EnvVar, "http://localhost:1234/")
	if got := envconfig.BaseURL(); got != "http://localhost:1234" {
		t.Errorf("env URL = %q, want trailing slash trimmed", got)
	}

	envconfig.SetBaseURL("http://example.com:9999")
	if got := envconfig.BaseURL(); got != "http://example.com:9999" {
		t.Errorf("flag override = %q, want http://example.com:9999", got)
	}
	if got := envconfig.Host().Host; got != "example.com:9999" {
		t.Errorf("Host().Host = %q", got)
	}
	envconfig.SetBaseURL("")
}

func TestRestoreUnsupportedIntegration(t *testing.T) {
	spec, err := launcher.LookupIntegrationSpec("claude")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := spec.Runner.(interface{ Restore() error }); ok {
		t.Error("claude runner should not implement Restore")
	}
	spec, err = launcher.LookupIntegrationSpec("codex")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := spec.Runner.(interface{ Restore() error }); !ok {
		t.Error("codex runner should implement Restore")
	}
}

func TestConfigPersistence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := config.SaveIntegration("codex", []string{"gpt-oss:20b"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadIntegration("codex")
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil || len(cfg.Models) != 1 || cfg.Models[0] != "gpt-oss:20b" {
		t.Fatalf("unexpected persisted config: %+v", cfg)
	}
	if _, err := os.Stat(filepath.Join(home, ".launch-harness", "config.json")); err != nil {
		t.Errorf("config file missing: %v", err)
	}
}

func TestEnvVarNameIsStable(t *testing.T) {
	if envconfig.EnvVar != "HARNESS_PROVIDER_URL" {
		t.Errorf("unexpected env var %q", envconfig.EnvVar)
	}
}
