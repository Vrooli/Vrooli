package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostpresentation"
	"github.com/vrooli/vrooli/internal/onboardinghandoff"
	"github.com/vrooli/vrooli/internal/projectstate"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioexec"
)

const (
	onboardingUrl = "url"
)

func forceSetupApplies(slug string) bool {
	if strings.ToLower(strings.TrimSpace(os.Getenv("FORCE_SETUP"))) != "true" {
		return false
	}
	target := strings.TrimSpace(os.Getenv("FORCE_SETUP_SCENARIO"))
	return target == "" || target == slug
}

type onboardingPreferences struct {
	AutoOpen   *bool  `json:"auto_open,omitempty"`
	PromptedAt string `json:"prompted_at,omitempty"`
}

func (s *setupService) runOnboardingHandoff(root, home string, opts Options, stdout, stderr io.Writer) (*OnboardingResult, error) {
	if configurationAlreadyComplete(home, root) || !onboardingScenarioExists(root) {
		return nil, nil
	}
	configPath, err := onboardingConfigPath(home)
	if err != nil {
		return nil, err
	}
	doc, prefs, err := loadOnboardingPreferences(configPath)
	if err != nil {
		return nil, err
	}
	mode, err := effectiveOnboardingMode(opts.Onboarding, prefs)
	if err != nil {
		return nil, err
	}
	if mode == onboardinghandoff.ModeNone {
		return &OnboardingResult{Decision: string(mode), Reason: "onboarding disabled by operator", PresentationKind: string(hostpresentation.KindUnknown)}, nil
	}

	capability := s.deps.detectPresentation(context.Background())
	decision, decisionErr := onboardinghandoff.Decide(capability, mode, scenarioexec.WriterSupportsStreaming(os.Stdin))
	result := &OnboardingResult{Decision: decisionAction(decision.Action), Reason: decision.Reason, PresentationKind: string(capability.Kind), ResumeCommand: decision.ResumeCommand}
	if decisionErr != nil {
		result.Reason = decisionErr.Error()
		return result, decisionErr
	}
	executable, err := s.deps.osExecutable()
	if err != nil {
		return result, err
	}
	if err := startOnboardingScenarioFn(root, executable); err != nil {
		result.Reason = "could not start onboarding: " + err.Error()
		return result, err
	}
	url, resolveErr := s.resolveOnboardingURL(executable)
	result.URL = url
	if resolveErr != nil {
		result.Decision = onboardingUrl
		result.Reason = "onboarding URL resolution timed out: " + resolveErr.Error()
		printOnboardingURL(stdout, result, 0)
		return result, nil
	}

	prefs.PromptedAt = s.deps.now().UTC().Format(time.RFC3339)
	if err := saveOnboardingPreferences(configPath, doc, prefs); err != nil {
		return result, err
	}
	port := onboardingPort(url)
	switch decision.Action {
	case "browser":
		_, _ = fmt.Fprintf(stdout, "[INFO]    Opening Vrooli onboarding at %s\n", url)
		if err := s.deps.openOnboardingURL(url); err != nil {
			_, _ = fmt.Fprintf(stderr, "[WARN]    Browser handoff failed: %v\n", err)
			result.Decision = onboardingUrl
			result.Reason = "browser handoff failed; URL handoff is available: " + err.Error()
			printOnboardingURL(stdout, result, port)
			return result, nil
		}
		result.Opened = true
	case "cli":
		cliExecutable, cliErr := s.deps.onboardingCLIExecutable()
		if cliErr != nil {
			return result, cliErr
		}
		if cliErr = s.deps.runOnboardingCLI(cliExecutable, root, os.Stdin, stdout, stderr); cliErr != nil {
			result.Reason = "interactive onboarding exited with an error: " + cliErr.Error()
			return result, cliErr
		}
	case onboardingUrl:
		printOnboardingURL(stdout, result, port)
	}
	return result, nil
}

func effectiveOnboardingMode(explicit onboardinghandoff.Mode, prefs onboardingPreferences) (onboardinghandoff.Mode, error) {
	if explicit != "" {
		return onboardinghandoff.ParseMode(string(explicit))
	}
	if value := strings.TrimSpace(strings.ToLower(os.Getenv(onboardingSkipEnv))); value == "1" || value == "true" || value == "yes" {
		return onboardinghandoff.ModeNone, nil
	}
	if prefs.AutoOpen != nil && !*prefs.AutoOpen {
		return onboardinghandoff.ModeURL, nil
	}
	return onboardinghandoff.ModeAuto, nil
}

func decisionAction(action string) string {
	if action == "" {
		return onboardingUrl
	}
	return action
}

func printOnboardingURL(stdout io.Writer, result *OnboardingResult, port int) {
	_, _ = fmt.Fprintf(stdout, "[INFO]    Onboarding is running at %s\n", result.URL)
	_, _ = fmt.Fprintf(stdout, "[INFO]    Presentation: %s (%s) — not opening a browser\n", result.PresentationKind, result.Reason)
	_, _ = fmt.Fprintf(stdout, "[ACTION]  Finish configuration here:  %s\n", result.ResumeCommand)
	_, _ = fmt.Fprintf(stdout, "[ACTION]  Or from a browser on this host: %s\n", result.URL)
	if port > 0 {
		_, _ = fmt.Fprintf(stdout, "[ACTION]  Or forward the port:          ssh -L %d:127.0.0.1:%d <user>@<host>\n", port, port)
	}
}

func onboardingPort(url string) int {
	const prefix = "http://127.0.0.1:"
	port, _ := strconv.Atoi(strings.TrimPrefix(url, prefix))
	return port
}

func onboardingScenarioExists(root string) bool {
	_, err := os.Stat(scenario.ServicePath(root, onboardingSlug))
	return err == nil
}

func configurationAlreadyComplete(home, root string) bool {
	locator, err := projectstate.NewLocator(home, root)
	return err == nil && locator.HasConfigurationComplete()
}

func onboardingConfigPath(home string) (string, error) {
	return filepath.Join(home, ".config", "vrooli", "config.json"), nil
}

func loadOnboardingPreferences(path string) (map[string]json.RawMessage, onboardingPreferences, error) {
	doc := map[string]json.RawMessage{}
	file, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return doc, onboardingPreferences{}, nil
		}
		return nil, onboardingPreferences{}, err
	}
	if len(file) == 0 {
		return doc, onboardingPreferences{}, nil
	}
	if err := json.Unmarshal(file, &doc); err != nil {
		return nil, onboardingPreferences{}, err
	}
	var prefs onboardingPreferences
	if raw, ok := doc["onboarding"]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &prefs); err != nil {
			return nil, onboardingPreferences{}, err
		}
	}
	return doc, prefs, nil
}

func saveOnboardingPreferences(path string, doc map[string]json.RawMessage, prefs onboardingPreferences) error {
	if doc == nil {
		doc = map[string]json.RawMessage{}
	}
	raw, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	doc["onboarding"] = raw
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), tuning.PermDir); err != nil {
		return err
	}
	return os.WriteFile(path, data, tuning.PermFile)
}

func startOnboardingScenario(root, executable string) error {
	if invokingUIDFn() != 0 || strings.TrimSpace(os.Getenv("SUDO_USER")) == "" {
		return scenarioexec.LaunchDetachedScenario(executable, root, rootcli.GlobalOptions{}, os.Environ(), "start", onboardingSlug)
	}
	return launchOnboardingAsOperatorFn(root, executable)
}

func launchOnboardingAsOperator(root, executable string) error {
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	ctx, cancel := context.WithTimeout(context.Background(), tuning.SetupOperationTimeout())
	defer cancel()
	return platform.RunAsInvokingUserInSession(ctx, executable,
		[]string{"scenario", "start", onboardingSlug},
		platform.IdentityCommandOptions{Dir: root, Stdin: devNull, Stdout: devNull, Stderr: devNull})
}

func (s *setupService) resolveOnboardingURL(executable string) (string, error) {
	deadline := s.deps.now().Add(tuning.SetupOperationTimeout())
	for {
		ctx, cancel := context.WithTimeout(context.Background(), tuning.HealthCheckTimeout())
		output, err := s.deps.onboardingPortCommandRunner(ctx, executable, "scenario", "port", onboardingSlug, "UI_PORT")
		cancel()
		text := strings.TrimSpace(string(output))
		if err == nil {
			port, parseErr := strconv.Atoi(text)
			if parseErr == nil && port > 0 {
				return fmt.Sprintf("http://127.0.0.1:%d", port), nil
			}
		}
		if s.deps.now().After(deadline) {
			if text == "" {
				text = "port could not be resolved before timeout"
			}
			return "", fmt.Errorf("onboarding UI not ready: %s", text)
		}
		time.Sleep(tuning.SetupProgressPollInterval())
	}
}

func markComplete(home, root string) error {
	locator, err := projectstate.NewLocator(home, root)
	if err != nil {
		return err
	}
	if _, err := config.EnsureOwnedDir(locator.SetupStateDir()); err != nil {
		return err
	}

	payload := map[string]any{
		"setup_version": "2.0.0",
		"completed_at":  time.Now().Format(time.RFC3339),
		"phase":         "bootstrap_complete",
		"project_key":   locator.ProjectKey(),
		"root":          locator.Root(),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return config.WriteOwnedFile(locator.BootstrapCompletePath(), data, tuning.PermFile)
}

// runSetupStatus runs an inspection-only pass and prints the grouped overview.
// No mutating operations, safe to run without sudo.
