package setup

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
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
	if locator, err := projectstate.NewLocator(home, root); err == nil {
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
	report = vrooliruntime.AnnotateInspectOnly(report, opts.IncludeOptional)
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
	return nil
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
