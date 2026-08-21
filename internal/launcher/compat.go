package launch

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/elecnix/launch-harness/internal/api"
	"github.com/elecnix/launch-harness/internal/config"
	"golang.org/x/term"

	"github.com/elecnix/launch-harness/internal/model"
)

// ANSI color helpers shared by the adapters.
const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiBold   = "\x1b[1m"
)

// LaunchModel is minimal model metadata passed to integration config writers.
// The server-backed inventory from Ollama's launcher is gone; models come
// from the --model flag, so most fields stay zero and adapters fall back to
// their defaults.
type LaunchModel struct {
	Name         string
	Remote       bool
	ToolCapable  bool
	Capabilities []model.Capability
	Details      struct {
		ContextLength int
		Format        string
		Family        string
		Families      []string
	}
	ContextLength   int
	MaxOutputTokens int
}

func (m LaunchModel) HasCapability(capability model.Capability) bool {
	for _, c := range m.Capabilities {
		if c == capability {
			return true
		}
	}
	return false
}

func (m LaunchModel) WithCloudLimits() LaunchModel { return m }

// fallbackLaunchModel builds a bare LaunchModel for a model name.
func fallbackLaunchModel(name string) LaunchModel {
	return LaunchModel{Name: name}
}

// findLaunchModel finds a model by name in a list.
func findLaunchModel(models []LaunchModel, name string) (LaunchModel, bool) {
	for _, m := range models {
		if m.Name == name {
			return m, true
		}
	}
	return LaunchModel{}, false
}

// isCloudModelName reports whether a name refers to an Ollama Cloud model.
// launch-harness has no cloud catalog, so this is always false.
func isCloudModelName(string) bool { return false }

// lookupCloudModelLimit always reports no limit known.
func lookupCloudModelLimit(string) (struct{ Context, Output int }, bool) {
	return struct{ Context, Output int }{}, false
}

var confirmReader = bufio.NewReader(os.Stdin)

// quotePowerShellString quotes s for safe interpolation into a PowerShell
// -Command argument.
func quotePowerShellString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// ansiGray dims text.
const ansiGray = "\033[37m"

// ErrCancelled is returned when the user cancels a prompt.
var ErrCancelled = errors.New("cancelled")

// launchModelsFromNames builds bare LaunchModels from names.
func launchModelsFromNames(names []string) []LaunchModel {
	out := make([]LaunchModel, 0, len(names))
	for _, n := range names {
		if n == "" {
			continue
		}
		out = append(out, fallbackLaunchModel(n))
	}
	return out
}

// ConfirmDefault controls which answer is highlighted in confirmation prompts.
type ConfirmDefault int

const (
	ConfirmDefaultYes ConfirmDefault = iota
	ConfirmDefaultNo
)

// ConfirmOptions customizes labels for confirmation prompts.
type ConfirmOptions struct {
	YesLabel string
	NoLabel  string
	Default  ConfirmDefault
}

// DefaultConfirmPrompt provides a TUI-based confirmation prompt.
var DefaultConfirmPrompt func(prompt string, options ConfirmOptions) (bool, error)

type launchConfirmPolicy struct {
	yes               bool
	requireYesMessage bool
}

var currentLaunchConfirmPolicy launchConfirmPolicy

func withLaunchConfirmPolicy(policy launchConfirmPolicy) func() {
	old := currentLaunchConfirmPolicy
	currentLaunchConfirmPolicy = policy
	return func() {
		currentLaunchConfirmPolicy = old
	}
}

// ConfirmPrompt is the shared confirmation gate (auto-approved by --yes).
func ConfirmPrompt(prompt string) (bool, error) {
	return ConfirmPromptWithOptions(prompt, ConfirmOptions{})
}

// ConfirmPromptWithOptions is the shared confirmation gate for launch flows
// that need custom yes/no labels.
func ConfirmPromptWithOptions(prompt string, options ConfirmOptions) (bool, error) {
	if currentLaunchConfirmPolicy.yes {
		return true, nil
	}
	if currentLaunchConfirmPolicy.requireYesMessage {
		return false, fmt.Errorf("%s requires confirmation; re-run with --yes to continue", prompt)
	}
	if DefaultConfirmPrompt != nil {
		return DefaultConfirmPrompt(prompt, options)
	}
	return plainConfirmPrompt(prompt, options)
}

func plainConfirmPrompt(prompt string, options ConfirmOptions) (bool, error) {
	defaultNo := options.Default == ConfirmDefaultNo
	suffix := "(y/N)"
	if !defaultNo {
		suffix = "(Y/n)"
	}
	fmt.Fprintf(os.Stderr, "%s %s ", prompt, suffix)
	line, err := confirmReader.ReadString('\n')
	if err != nil && line == "" {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer == "" {
		return !defaultNo, nil
	}
	return answer == "y" || answer == "yes", nil
}

var isInteractiveSession = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// dedupeModelList removes empty entries and duplicates, preserving order.
func dedupeModelList(models []string) []string {
	deduped := make([]string, 0, len(models))
	seen := make(map[string]bool, len(models))
	for _, m := range models {
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		deduped = append(deduped, m)
	}
	return deduped
}

// launchModelNames extracts non-empty model names from a LaunchModel list.
func launchModelNames(models []LaunchModel) []string {
	names := make([]string, 0, len(models))
	for _, m := range models {
		if m.Name != "" {
			names = append(names, m.Name)
		}
	}
	return names
}

// cloneLaunchModels copies a LaunchModel slice.
func cloneLaunchModels(models []LaunchModel) []LaunchModel {
	cloned := make([]LaunchModel, len(models))
	copy(cloned, models)
	return cloned
}

// launchModelMatches reports whether a candidate model name matches a target,
// tolerating the implicit :latest tag.
func launchModelMatches(candidate, name string) bool {
	if candidate == name {
		return true
	}
	return strings.TrimSuffix(candidate, ":latest") == name
}

// cloudStatusDisabled always reports "unknown" — no cloud status without a server.
func cloudStatusDisabled(context.Context, *api.Client) (disabled bool, known bool) {
	return false, false
}

// loadStoredIntegrationConfig loads the persisted config for an integration.
func loadStoredIntegrationConfig(name string) (*config.IntegrationConfig, error) {
	return config.LoadIntegration(name)
}

// LoadedContextWindow always reports 0 — it required a live Ollama server.
func LoadedContextWindow(context.Context, *api.Client, string) int { return 0 }

// pullModel is a stub: launch-harness cannot pull models without an Ollama
// server. Callers surface the error to the user.
func pullModel(context.Context, *api.Client, string, bool) error {
	return errors.New("launch-harness cannot pull models (no Ollama server)")
}

// SupportedIntegration lets an integration report platform support separately
// from installation state.
type SupportedIntegration interface {
	Supported() error
}

// ModelItem is a lightweight name/description row for listings.
type ModelItem struct {
	Name        string
	Description string
}

// Runner executes an integration with the selected model and passthrough args.
type Runner interface {
	Run(model string, models []LaunchModel, args []string) error
	String() string
}

// Editor can edit config files for integrations that support model configuration.
type Editor interface {
	Paths() []string
	Edit(models []LaunchModel) error
	Models() []string
}

// AnsiRed, AnsiGreen, AnsiReset and AnsiBold are exported ANSI helpers for
// the CLI entrypoint.
const (
	AnsiRed   = ansiRed
	AnsiGreen = ansiGreen
	AnsiReset = ansiReset
	AnsiBold  = ansiBold
)
