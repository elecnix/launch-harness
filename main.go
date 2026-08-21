// Command harness-launch launches any coding-agent harness against a
// user-provided provider URL.
//
// Usage:
//
//	harness-launch                          list supported integrations
//	harness-launch <integration> [flags] [-- <extra args>]
//
// The provider URL comes from --url or $HARNESS_PROVIDER_URL.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/elecnix/launch-harness/internal/envconfig"
	"github.com/spf13/cobra"

	launcher "github.com/elecnix/launch-harness/internal/launcher"
)

var (
	version    = "dev"
	model      string
	url        string
	yes        bool
	restore    bool
	configOnly bool
)

func main() {
	root := &cobra.Command{
		Use:   "harness-launch [INTEGRATION] [-- EXTRA_ARGS...]",
		Short: "Launch coding-agent harnesses against your provider",
		Long: `Launch a coding-agent harness (Claude Code, Codex, OpenCode, Pi, ...) configured
to use your own inference provider endpoint.

The provider URL is resolved from (in order):
  1. the --url flag
  2. the $HARNESS_PROVIDER_URL environment variable
  3. the default ` + envconfig.DefaultURL + `

Examples:
  harness-launch                              # list integrations
  harness-launch claude --model gpt-oss:20b   # launch Claude Code
  harness-launch codex --url http://localhost:8080 -- --sandbox workspace-write
  HARNESS_PROVIDER_URL=http://localhost:1234 harness-launch opencode`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		Version:       version,
		RunE: func(cmd *cobra.Command, args []string) error {
			envconfig.SetBaseURL(url)
			if len(args) == 0 {
				return listIntegrations()
			}
			return launch(args[0], args[1:])
		},
	}

	flags := root.Flags()
	flags.StringVarP(&model, "model", "m", "", "model to launch with")
	flags.StringVarP(&url, "url", "u", "", "provider base URL (or $"+envconfig.EnvVar+")")
	flags.BoolVarP(&yes, "yes", "y", false, "auto-approve confirmation prompts")
	flags.BoolVar(&restore, "restore", false, "restore the integration's original configuration and exit")
	flags.BoolVar(&configOnly, "config", false, "write the integration configuration without launching")

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%serror: %v%s\n", launcher.AnsiRed, err, launcher.AnsiReset)
		os.Exit(1)
	}
}

func listIntegrations() error {
	specs := launcher.ListVisibleIntegrationSpecs()
	fmt.Fprintf(os.Stderr, "Supported integrations:\n\n")
	for _, spec := range specs {
		line := fmt.Sprintf("  %-14s %s", spec.Name, spec.Description)
		if len(spec.Aliases) > 0 {
			line += fmt.Sprintf(" (aliases: %s)", strings.Join(spec.Aliases, ", "))
		}
		fmt.Fprintln(os.Stderr, line)
	}
	fmt.Fprintf(os.Stderr, "\nProvider URL: %s (override with --url or $%s)\n",
		envconfig.BaseURL(), envconfig.EnvVar)
	return nil
}

func launch(name string, extraArgs []string) error {
	spec, err := launcher.LookupIntegrationSpec(name)
	if err != nil {
		return fmt.Errorf("%w\nRun `harness-launch` to see available integrations", err)
	}

	runner := spec.Runner

	// Restore mode: undo configuration changes and exit.
	if restore {
		r, ok := runner.(interface{ Restore() error })
		if !ok {
			return fmt.Errorf("%s does not support restore", spec.Name)
		}
		if err := r.Restore(); err != nil {
			return err
		}
		if m, ok := runner.(interface{ RestoreSuccessMessage() string }); ok {
			fmt.Fprintf(os.Stderr, "%s\n", m.RestoreSuccessMessage())
		} else {
			fmt.Fprintf(os.Stderr, "%s configuration restored.\n", spec.Name)
		}
		return nil
	}

	if yes {
		defer launcher.WithLaunchConfirmPolicy(launcher.LaunchConfirmPolicyYes())()
	}

	if err := launcher.EnsureIntegrationInstalled(spec.Name, runner); err != nil {
		return err
	}

	var models []launcher.LaunchModel
	if model != "" {
		models = []launcher.LaunchModel{{Name: model}}
	}

	// Editor integrations persist their configuration via Edit before launch.
	if editor, ok := runner.(launcher.Editor); ok && (model != "" || configOnly) {
		if err := editor.Edit(models); err != nil {
			return fmt.Errorf("failed to configure %s: %w", spec.Name, err)
		}
		if configOnly {
			fmt.Fprintf(os.Stderr, "%s configuration updated.%s\n", launcher.AnsiGreen, launcher.AnsiReset)
			return nil
		}
	} else if configOnly {
		return fmt.Errorf("%s does not support standalone configuration", spec.Name)
	}

	fmt.Fprintf(os.Stderr, "%sLaunching %s%s%s...\n", launcher.AnsiBold, launcher.AnsiGreen, runner.String(), launcher.AnsiReset)
	return runIntegration(runner, model, models, extraArgs)
}

func runIntegration(runner launcher.Runner, modelName string, models []launcher.LaunchModel, args []string) error {
	if len(models) == 0 && modelName != "" {
		models = []launcher.LaunchModel{{Name: modelName}}
	}
	return runner.Run(modelName, models, args)
}
