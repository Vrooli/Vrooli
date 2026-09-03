package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	"github.com/vrooli/vrooli/internal/projectstate"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

func (s *setupService) runSetupStatus(root, home string, opts Options, stdout io.Writer) error {
	if err := s.deps.currentHost().ValidateSetup(); err != nil {
		return err
	}
	if _, err := s.deps.loadProject(root); err != nil {
		return err
	}
	if locator, err := projectstate.NewLocator(home, root); err == nil && !opts.JSON {
		renderActiveSetupState(stdout, locator.ActiveSetupPath(), s.deps.now())
		if locator.HasConfigurationComplete() {
			marker, markerErr := locator.ReadConfigurationComplete()
			if markerErr == nil {
				_, _ = fmt.Fprintf(stdout, "[INFO]    Configuration: complete (completed_at=%s selection_digest=%s)\n", marker.CompletedAt, marker.SelectionDigest)
			} else {
				_, _ = fmt.Fprintf(stdout, "[INFO]    Configuration: complete (.configuration-complete present; unreadable: %v)\n", markerErr)
			}
		} else {
			_, _ = fmt.Fprintln(stdout, "[INFO]    Configuration: pending (.configuration-complete absent)")
		}
	}
	phase := strings.ToLower(strings.TrimSpace(opts.Phase))
	if phase == "" {
		phase = "setup"
	}
	requirements, err := s.deps.resolveHostRequirements(root, home, hostreq.ResolveOptions{
		Environment: opts.Environment,
		When:        phase,
		Resources:   opts.Resources,
		Scenarios:   opts.Scenarios,
		Platform:    hostreq.CurrentPlatform(),
	})
	if err != nil {
		return err
	}
	if phase == "setup" {
		requirements = bootstrapAwareRequirements(requirements)
		if executable, executableErr := s.deps.osExecutable(); executableErr == nil {
			requirements = addOnboardingApplyPrivilegeRequirement(requirements, executable)
		}
	}
	report, err := s.deps.inspectRequirements(opts.Environment, requirements)
	if err != nil {
		return err
	}
	report = vrooliruntime.AnnotateInspectOnly(report, opts.IncludeOptional)
	verdict := verifySetupReadiness(root, report, nil)
	if opts.JSON {
		return writeSetupStatusJSON(stdout, phase, report, verdict)
	}
	if verdict.Source == ReadinessSourceUnavailable {
		_, _ = fmt.Fprintf(stdout, "[INFO]    Readiness: unverified (%s)\n", verdict.Reason)
	} else if len(verdict.Blockers) > 0 {
		_, _ = fmt.Fprintf(stdout, "[INFO]    Readiness: %s (unresolved: %s)\n", verdict.Status, strings.Join(verdict.Blockers, ", "))
	} else {
		_, _ = fmt.Fprintf(stdout, "[INFO]    Readiness: %s\n", verdict.Status)
	}
	_, _ = fmt.Fprintf(
		stdout,
		"[INFO]    Host requirements status (environment=%s)\n",
		displaySelection(report.Environment, displaySelection(opts.Environment, defaultEnvironment)),
	)
	mode := renderModeGrouped
	if opts.Verbose {
		mode = renderModeVerbose
	}
	renderSetupRequirementOverview(stdout, report, false, mode)
	renderPrivilegeBrokerStatus(stdout, s.deps.inspectPrivilegeBroker())
	bootRecovery, bootErr := s.deps.bootRecoveryStatus(context.Background())
	renderBootRecovery(stdout, bootRecovery, bootErr)
	return nil
}

// SetupStatusReport is the machine-readable form of `vrooli setup status`.
// Consumers (the autoheal boot-recovery readiness check) read one safeguard's
// `applied`, `execution_state`, `notes` and `evidence` fields; the shape is
// the inspection report the human rendering is built from, not a summary.
type SetupStatusReport struct {
	Version     string                  `json:"version"`
	Phase       string                  `json:"phase"`
	Environment string                  `json:"environment"`
	Readiness   SetupReadiness          `json:"readiness"`
	Tools       []hostreqkit.ToolStatus `json:"tools"`
	Safeguards  []hostreqkit.ItemStatus `json:"safeguards"`
	Missing     SetupStatusMissing      `json:"missing"`
	Host        hostreqkit.Host         `json:"host"`
}

// SetupStatusMissing lists the requirements the inspection found unmet.
type SetupStatusMissing struct {
	Required []string `json:"required"`
	Optional []string `json:"optional"`
}

// SetupStatusReportVersion is bumped when a field changes meaning; consumers
// branch on it and ignore unknown fields.
const SetupStatusReportVersion = "1"

func writeSetupStatusJSON(stdout io.Writer, phase string, report vrooliruntime.Report, verdict SetupReadiness) error {
	payload := SetupStatusReport{
		Version:     SetupStatusReportVersion,
		Phase:       phase,
		Environment: report.Environment,
		Readiness:   verdict,
		Tools:       append([]hostreqkit.ToolStatus{}, report.Tools...),
		Safeguards:  append([]hostreqkit.ItemStatus{}, report.Safeguards...),
		Missing:     SetupStatusMissing{Required: append([]string{}, report.MissingRequired...), Optional: append([]string{}, report.MissingOptional...)},
		Host:        report.Host,
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func renderPrivilegeBrokerStatus(stdout io.Writer, status privilegebroker.SetupStatus) {
	if status.Available {
		_, _ = fmt.Fprintf(stdout, "[INFO]    Privilege broker: available (%s)\n", status.SocketPath)
		return
	}
	if status.Supported {
		_, _ = fmt.Fprintf(stdout, "[INFO]    Privilege broker: unavailable — %s\n", status.Reason)
	} else {
		_, _ = fmt.Fprintf(stdout, "[INFO]    Privilege broker: unsupported — %s\n", status.Reason)
	}
	if status.Recovery != "" {
		_, _ = fmt.Fprintf(stdout, "[INFO]    Privilege broker recovery: %s\n", status.Recovery)
	}
}

// runSetupExplain prints the full per-item block for one requirement.
func (s *setupService) runSetupExplain(root, home string, opts Options, stdout io.Writer) error {
	name := strings.TrimSpace(opts.ExplainName)
	if name == "" {
		return fmt.Errorf("setup explain requires a requirement name")
	}
	if err := s.deps.currentHost().ValidateSetup(); err != nil {
		return err
	}
	if _, err := s.deps.loadProject(root); err != nil {
		return err
	}
	requirements, err := s.deps.resolveHostRequirements(root, home, hostreq.ResolveOptions{
		Environment: opts.Environment,
		When:        "setup",
		Resources:   opts.Resources,
		Scenarios:   opts.Scenarios,
		Platform:    hostreq.CurrentPlatform(),
	})
	if err != nil {
		return err
	}
	requirements = bootstrapAwareRequirements(requirements)
	if executable, executableErr := s.deps.osExecutable(); executableErr == nil {
		requirements = addOnboardingApplyPrivilegeRequirement(requirements, executable)
	}
	report, err := s.deps.inspectRequirements(opts.Environment, requirements)
	if err != nil {
		return err
	}
	item, ok := findItemByName(report, name)
	if !ok {
		return fmt.Errorf("no host requirement named %q (run 'vrooli setup status' to list)", name)
	}
	_, _ = fmt.Fprintf(stdout, "[INFO]    %s\n", item.Name)
	renderRequirementVerboseItem(stdout, item, false)
	return nil
}
