package agentapp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
)

// NativeSettings binds resources/<name>/native-settings.json to the agent's
// own configuration file. The resource supplies the file location and format;
// agentharness owns ownership, conflict and state rules.
type NativeSettings struct {
	// DocumentPath overrides resources/<name>/native-settings.json (tests).
	DocumentPath string
	SettingsPath func() (string, error)
	Decode       func([]byte) (map[string]any, error)
	Encode       func(map[string]any) ([]byte, error)
}

// EnsureStep is one resource-specific self-heal action run by `ensure` after
// native settings. Steps write their own human output.
type EnsureStep struct {
	Name string
	Run  func(stdout, stderr io.Writer) error
}

// ensureCommand is the `ensure` verb that `vrooli setup` and scenario
// dependency starts call on a resource advertising supports_ensure. Every
// step runs even when an earlier one fails, so one broken step cannot hide
// the others; the command fails if any step failed.
func ensureCommand(spec Spec, binary string) cliapp.Command {
	return cliapp.Command{
		Name:        "ensure",
		Description: "Bring this agent's Vrooli-managed configuration into compliance (idempotent; user-owned values are left alone)",
		Usage:       "resource-" + spec.Name + " ensure [--json] [--config-base64 <b64>]",
		Run: func(args []string) error {
			return runEnsure(spec, binary, args, os.Stdout, os.Stderr)
		},
	}
}

type ensureStepResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type ensureResult struct {
	Resource       string                             `json:"resource"`
	Status         string                             `json:"status"`
	NativeSettings *agentharness.NativeSettingsReport `json:"native_settings,omitempty"`
	Steps          []ensureStepResult                 `json:"steps"`
	Notes          []string                           `json:"notes,omitempty"`
}

func runEnsure(spec Spec, binary string, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("ensure", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "Emit JSON")
	configB64 := fs.String("config-base64", "", "Scenario dependency config (base64 JSON); agent resources declare no dependency keys yet")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result := ensureResult{Resource: spec.Name, Status: "ok", Steps: []ensureStepResult{}}
	if note, err := dependencyConfigNote(*configB64); err != nil {
		return err
	} else if note != "" {
		result.Notes = append(result.Notes, note)
	}
	human := stdout
	if *jsonOut {
		human = stderr
	}
	var failures []string
	if spec.NativeSettings != nil {
		report, err := agentharness.EnsureNativeSettings(nativeSettingsConfig(spec, binary, human, stderr))
		step := ensureStepResult{Name: "native-settings", Status: "ok"}
		if err != nil {
			step.Status, step.Error = "failed", err.Error()
			failures = append(failures, "native-settings: "+err.Error())
		} else {
			result.NativeSettings = &report
			if len(report.Conflicts()) > 0 {
				step.Status = "conflicts"
			}
			if !*jsonOut {
				agentharness.WriteNativeSettingsReport(human, report)
			}
		}
		result.Steps = append(result.Steps, step)
	}
	for _, s := range spec.EnsureSteps {
		step := ensureStepResult{Name: s.Name, Status: "ok"}
		if err := s.Run(human, stderr); err != nil {
			step.Status, step.Error = "failed", err.Error()
			failures = append(failures, s.Name+": "+err.Error())
		}
		result.Steps = append(result.Steps, step)
	}
	if len(result.Steps) == 0 {
		result.Notes = append(result.Notes, "nothing declared to ensure for "+spec.Name)
	}
	if len(failures) > 0 {
		result.Status = "failed"
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return err
		}
	} else {
		for _, note := range result.Notes {
			fmt.Fprintln(stdout, "note: "+note)
		}
	}
	if len(failures) > 0 {
		return errors.New("ensure failed: " + strings.Join(failures, "; "))
	}
	return nil
}

// dependencyConfigNote accepts the lifecycle's --config-base64 contract. No
// agent resource declares dependency keys yet, so non-empty config is
// reported rather than silently dropped.
func dependencyConfigNote(encoded string) (string, error) {
	if strings.TrimSpace(encoded) == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("--config-base64: %w", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("--config-base64: %w", err)
	}
	if len(payload) == 0 {
		return "", nil
	}
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	return "dependency config keys are not interpreted by this resource: " + strings.Join(keys, ", "), nil
}

func nativeSettingsConfig(spec Spec, binary string, stdout, stderr io.Writer) agentharness.NativeSettingsConfig {
	return agentharness.NativeSettingsConfig{
		Runner:           spec.Name,
		DocumentPath:     spec.NativeSettings.DocumentPath,
		SettingsPath:     spec.NativeSettings.SettingsPath,
		Decode:           spec.NativeSettings.Decode,
		Encode:           spec.NativeSettings.Encode,
		InstalledVersion: func() string { return installedVersion(binary) },
		WriterVersion:    "resource-" + spec.Name + " " + Version,
		Stdout:           stdout,
		Stderr:           stderr,
	}
}

func installedVersion(binary string) string {
	if binary == "" {
		return ""
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
