package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/operatorinput"
	"github.com/vrooli/vrooli/internal/runtime"
)

// SetupResultVersion is the wire contract version written by --result-file.
// Consumers must branch on this value before interpreting the payload.
const SetupResultVersion = "v1"

const (
	SetupStatusSuccess = "success"
	SetupStatusFailed  = "failed"

	SetupCategorySuccess                    = "success"
	SetupCategoryConfigurationPending       = "configuration_pending"
	SetupCategoryUnsupportedPlatform        = "unsupported_platform"
	SetupCategoryRequiredRequirementBlocked = "required_requirement_blocked"
	SetupCategoryInvalidConfiguration       = "invalid_configuration"
	SetupCategoryPartialState               = "partial_state"
	SetupCategoryTransientFailure           = "transient_failure"
)

// SetupResult is setup's machine-readable terminal contract. Human-oriented
// setup output remains on stdout/stderr; this document is deliberately written
// to a separate caller-provided path so automation never has to parse prose.
type SetupResult struct {
	Version              string   `json:"version"`
	Status               string   `json:"status"`
	Category             string   `json:"category"`
	Stage                string   `json:"stage"`
	Retryable            bool     `json:"retryable"`
	BlockedRequirements  []string `json:"blocked_requirements,omitempty"`
	Remediation          string   `json:"remediation"`
	ConfigurationPending bool     `json:"configuration_pending,omitempty"`
}

func setupTerminalResult(stage string, report runtime.Report, runErr error) SetupResult {
	result := SetupResult{Version: SetupResultVersion, Stage: stage}
	if runErr == nil {
		result.Status = SetupStatusSuccess
		result.Category = SetupCategorySuccess
		result.Stage = "complete"
		result.Remediation = "Setup completed successfully."
		if pending, err := operatorinput.Load(); err == nil && len(pending.Requests) > 0 {
			result.Category = SetupCategoryConfigurationPending
			result.ConfigurationPending = true
			result.Remediation = "Bootstrap completed. Continue in vrooli-onboarding to answer the pending operator inputs."
		}
		return result
	}

	result.Status = SetupStatusFailed
	if errors.Is(runErr, runtime.ErrUnsupportedPlatform) {
		result.Category = SetupCategoryUnsupportedPlatform
		result.Retryable = false
		result.Remediation = "This host platform is not supported by vrooli setup. Use a supported host or update vrooli when support is available."
		return result
	}

	result.BlockedRequirements = blockedRequirementNames(report)
	if len(result.BlockedRequirements) > 0 {
		result.Category = SetupCategoryRequiredRequirementBlocked
		result.Retryable = true
		result.Remediation = requirementRemediation(report)
		return result
	}

	switch stage {
	case "configuration", "project", "resolution":
		result.Category = SetupCategoryInvalidConfiguration
		result.Retryable = false
		result.Remediation = "Correct the project or setup configuration, then run setup again."
	case "filesystem", "bootstrap", "requirements", "generated-packages", "credentials", "privilege-broker", "git", "resources", "cli", "finalize":
		result.Category = SetupCategoryPartialState
		result.Retryable = true
		result.Remediation = "Setup may have completed some steps. Correct the reported condition and re-run; setup is designed to converge."
	default:
		result.Category = SetupCategoryTransientFailure
		result.Retryable = true
		result.Remediation = "Inspect the human diagnostics, correct the reported condition, and re-run setup."
	}
	return result
}

func blockedRequirementNames(report runtime.Report) []string {
	blocked := append([]string(nil), report.MissingRequired...)
	sort.Strings(blocked)
	return blocked
}

func requirementRemediation(report runtime.Report) string {
	for _, item := range append(append([]runtime.ItemStatus(nil), report.Tools...), report.Safeguards...) {
		if !item.Required || item.BlockingReason == hostreqkit.BlockingNone {
			continue
		}
		switch item.BlockingReason {
		case hostreqkit.BlockingNeedsSudo:
			return "A required operation needs privilege. Re-run with an explicit permitted sudo policy or configure passwordless sudo for noninteractive onboarding."
		case hostreqkit.BlockingNeedsReboot:
			return "A required operation needs a reboot. Reboot the host, then re-run setup."
		case hostreqkit.BlockingNeedsEnv:
			return "A required environment value is missing. Configure it, then re-run setup."
		case hostreqkit.BlockingInvalidParameter:
			return "A required safeguard parameter is invalid. Correct operator-state config, then re-run setup."
		case hostreqkit.BlockingManual:
			return "A required dependency needs the documented manual action. Complete it, then re-run setup."
		case hostreqkit.BlockingPrerequisiteMissing:
			return "A required safeguard is waiting on another host requirement. Satisfy the prerequisite named in the diagnostics, then re-run setup."
		}
	}
	if len(report.MissingRequired) > 0 {
		return "A required dependency could not be installed or verified. Inspect the human diagnostics, correct the condition, and re-run setup."
	}
	return "Inspect the human diagnostics, correct the reported condition, and re-run setup."
}

func writeSetupResult(path string, result SetupResult) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create setup result directory: %w", err)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal setup result: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".setup-result-*")
	if err != nil {
		return fmt.Errorf("create setup result: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(append(payload, '\n')); err != nil {
		tmp.Close()
		return fmt.Errorf("write setup result: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("secure setup result: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close setup result: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("publish setup result: %w", err)
	}
	return nil
}
