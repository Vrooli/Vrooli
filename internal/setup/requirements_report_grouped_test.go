//nolint:goconst // test data deliberately reuses stable filesystem fixtures.
package setup

import (
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

func sampleMixedReport() vrooliruntime.Report {
	prov := []hostreqspec.Provenance{{Kind: "root", Name: "vrooli", Source: ".vrooli/service.json"}}
	return vrooliruntime.Report{
		Environment: "development",
		Host:        vrooliruntime.Host{OS: "linux", PackageManager: "apt-get"},
		Tools: []vrooliruntime.ToolStatus{
			{Name: "git", Kind: hostreqspec.KindTool, Required: true, ExecutionState: vrooliruntime.ExecutionAlreadyPresent, Provenance: prov},
			{Name: "docker", Kind: hostreqspec.KindTool, Required: true, ExecutionState: vrooliruntime.ExecutionAlreadyPresent, Provenance: prov},
			{
				Name: "kdump-tools", Kind: hostreqspec.KindTool, Required: true,
				ExecutionState: vrooliruntime.ExecutionFailed,
				Notes:          []string{"debconf-set-selections failed: exit status 1", "Package kdump-tools not configured"},
				Provenance:     prov,
			},
			{
				Name: "mcelog", Kind: hostreqspec.KindTool, Required: true,
				ExecutionState: vrooliruntime.ExecutionAlreadyPresent,
				Notes:          []string{"superseded by rasdaemon (no mcelog package available on this distribution)"},
				Provenance:     prov,
			},
		},
		Safeguards: []vrooliruntime.SafeguardStatus{
			{Name: "crashkernel_reserve", Kind: hostreqspec.KindSafeguard, Required: false, ExecutionState: vrooliruntime.ExecutionPending, Notes: []string{"crashkernel pending: will add crashkernel=512M-:256M"}, Provenance: prov},
			{Name: "edac_modules", Kind: hostreqspec.KindSafeguard, Required: false, ExecutionState: vrooliruntime.ExecutionNotApplicable, Provenance: prov},
			{Name: "nat_protection", Kind: hostreqspec.KindSafeguard, Required: false, ExecutionState: vrooliruntime.ExecutionApplied, Provenance: prov},
		},
	}
}

func TestRenderGroupedSeparatesGroups(t *testing.T) {
	var sb strings.Builder
	report := sampleMixedReport()
	renderGrouped(&sb, report)

	out := sb.String()
	failedIdx := strings.Index(out, "Failed (1):")
	pendingIdx := strings.Index(out, "Needs operator input")
	appliedIdx := strings.Index(out, "Applied (1):")
	alreadyIdx := strings.Index(out, "Already present (3): docker, git, mcelog")
	notApplIdx := strings.Index(out, "Not applicable (1):")
	deltaIdx := strings.Index(out, "Δ installed=0  applied=1  failed=1  pending=1  unchanged=4")

	for label, idx := range map[string]int{
		"failed":         failedIdx,
		"pending":        pendingIdx,
		"applied":        appliedIdx,
		"alreadyPresent": alreadyIdx,
		"notApplicable":  notApplIdx,
		"delta":          deltaIdx,
	} {
		if idx < 0 {
			t.Fatalf("missing %s section in output:\n%s", label, out)
		}
	}

	if !(failedIdx < pendingIdx && pendingIdx < appliedIdx && appliedIdx < alreadyIdx && alreadyIdx < notApplIdx && notApplIdx < deltaIdx) {
		t.Fatalf("group ordering wrong; got order failed=%d pending=%d applied=%d alreadyPresent=%d notApplicable=%d delta=%d in:\n%s",
			failedIdx, pendingIdx, appliedIdx, alreadyIdx, notApplIdx, deltaIdx, out)
	}

	if !strings.Contains(out, "Run 'vrooli setup explain <name>'") {
		t.Errorf("output missing explain hint:\n%s", out)
	}
}

func TestRenderGroupedFailureAttachesDetail(t *testing.T) {
	var sb strings.Builder
	renderGrouped(&sb, sampleMixedReport())
	out := sb.String()

	// First note is shown as the headline; earlier notes go in the indented detail.
	if !strings.Contains(out, "✗ kdump-tools") {
		t.Fatalf("missing failure marker:\n%s", out)
	}
	// Last note is the headline.
	if !strings.Contains(out, "Package kdump-tools not configured") {
		t.Fatalf("missing failure headline:\n%s", out)
	}
	// Earlier notes appear in the indented block.
	if !strings.Contains(out, "       debconf-set-selections failed: exit status 1") {
		t.Fatalf("indented detail missing earlier note:\n%s", out)
	}
}

func TestRenderGroupedAllPresentCollapses(t *testing.T) {
	prov := []hostreqspec.Provenance{{Kind: "root", Name: "vrooli", Source: ".vrooli/service.json"}}
	report := vrooliruntime.Report{
		Environment: "development",
		Host:        vrooliruntime.Host{OS: "linux", PackageManager: "apt-get"},
		Tools: []vrooliruntime.ToolStatus{
			{Name: "git", Kind: hostreqspec.KindTool, Required: true, ExecutionState: vrooliruntime.ExecutionAlreadyPresent, Provenance: prov},
			{Name: "jq", Kind: hostreqspec.KindTool, Required: true, ExecutionState: vrooliruntime.ExecutionAlreadyPresent, Provenance: prov},
		},
	}
	var sb strings.Builder
	renderGrouped(&sb, report)
	out := sb.String()

	// Should be a single collapsed line plus the delta + hint.
	if !strings.Contains(out, "Already present (2): git, jq") {
		t.Fatalf("expected collapsed already-present line:\n%s", out)
	}
	for _, forbidden := range []string{"Failed", "Needs sudo", "Needs operator input", "Optional", "Applied (", "Not applicable"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("unexpected group %q present in all-present render:\n%s", forbidden, out)
		}
	}
}

func TestRenderGroupedSplitsByBlockingReason(t *testing.T) {
	prov := []hostreqspec.Provenance{{Kind: "root", Name: "vrooli", Source: ".vrooli/service.json"}}
	report := vrooliruntime.Report{
		Environment: "development",
		Host:        vrooliruntime.Host{OS: "linux", PackageManager: "apt-get"},
		Tools: []vrooliruntime.ToolStatus{
			{
				Name:           "kdump-tools",
				Kind:           hostreqspec.KindTool,
				Required:       true,
				ExecutionState: vrooliruntime.ExecutionFailed,
				BlockingReason: hostreqkit.BlockingNeedsSudo,
				Notes:          []string{"debconf-set-selections failed: sudo skipped"},
				Provenance:     prov,
			},
		},
		Safeguards: []vrooliruntime.SafeguardStatus{
			{
				Name:           "tcp_tuning",
				Kind:           hostreqspec.KindSafeguard,
				Required:       false,
				ExecutionState: vrooliruntime.ExecutionPending,
				BlockingReason: hostreqkit.BlockingOptionalSkipped,
				Provenance:     prov,
			},
			{
				Name:           "crashkernel_reserve",
				Kind:           hostreqspec.KindSafeguard,
				Required:       false,
				ExecutionState: vrooliruntime.ExecutionPending,
				BlockingReason: hostreqkit.BlockingOptionalSkipped,
				Provenance:     prov,
			},
		},
	}
	var sb strings.Builder
	renderGrouped(&sb, report)
	out := sb.String()

	// Action block lists both follow-up commands.
	if !strings.Contains(out, "To finish setup:") {
		t.Fatalf("missing action block:\n%s", out)
	}
	// The privilege action names the in-flow command. `sudo vrooli setup` was
	// never the in-flow answer: it leaves the whole run as root and is outside
	// the setup-and-onboarding flow the operator is being guided through.
	if !strings.Contains(out, "setup --sudo-mode=ask") {
		t.Fatalf("action block missing the in-flow privilege command:\n%s", out)
	}
	if strings.Contains(out, "sudo vrooli setup") {
		t.Fatalf("action block still instructs an out-of-flow sudo invocation:\n%s", out)
	}
	if !strings.Contains(out, "setup --include-optional") {
		t.Fatalf("action block missing include-optional command:\n%s", out)
	}
	// The privilege group has the kdump-tools item, NOT the generic Failed group.
	if !strings.Contains(out, "Needs privilege") {
		t.Fatalf("missing privilege group:\n%s", out)
	}
	if strings.Contains(out, "Failed (1)") {
		t.Fatalf("kdump-tools should be in Needs sudo, not Failed:\n%s", out)
	}
	// Optional group is collapsed: one line listing both names.
	if !strings.Contains(out, "Optional — opt in") || !strings.Contains(out, "crashkernel_reserve, tcp_tuning") {
		t.Fatalf("missing or wrong Optional group:\n%s", out)
	}
}

func TestFindItemByNameSearchesBothKinds(t *testing.T) {
	report := sampleMixedReport()
	if item, ok := findItemByName(report, "DOCKER"); !ok || item.Name != "docker" {
		t.Fatalf("findItemByName(DOCKER) = (%+v, %v)", item, ok)
	}
	if item, ok := findItemByName(report, "edac_modules"); !ok || item.Kind != hostreqspec.KindSafeguard {
		t.Fatalf("findItemByName(edac_modules) = (%+v, %v)", item, ok)
	}
	if _, ok := findItemByName(report, "nope"); ok {
		t.Fatalf("findItemByName(nope) should be missing")
	}
}

func TestRenderVerboseModeKeepsBlocks(t *testing.T) {
	var sb strings.Builder
	report := sampleMixedReport()
	renderSetupRequirementOverview(&sb, report, true, renderModeVerbose)
	out := sb.String()
	for _, expected := range []string{
		"[INFO]    Tools:",
		"[INFO]    Safeguards:",
		"git [required] already_present",
		"declared by root:vrooli (.vrooli/service.json)",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("verbose output missing %q:\n%s", expected, out)
		}
	}
}

func TestActionBlockUsesCurrentExecutable(t *testing.T) {
	origExe := vrooliExecutableFn
	defer func() { vrooliExecutableFn = origExe }()
	exePath := "/opt/vrooli/bin/vrooli"
	lookups := 0
	vrooliExecutableFn = func() (string, error) { lookups++; return exePath, nil }

	prov := []hostreqspec.Provenance{{Kind: "root", Name: "vrooli", Source: ".vrooli/service.json"}}
	report := vrooliruntime.Report{
		Environment: "development",
		Host:        vrooliruntime.Host{OS: "linux", PackageManager: "apt-get"},
		Tools: []vrooliruntime.ToolStatus{
			{
				Name: "kdump-tools", Kind: hostreqspec.KindTool, Required: true,
				ExecutionState: vrooliruntime.ExecutionFailed,
				BlockingReason: hostreqkit.BlockingNeedsSudo,
				Notes:          []string{"sudo skipped"},
				Provenance:     prov,
			},
		},
		Safeguards: []vrooliruntime.SafeguardStatus{
			{
				Name: "tcp_tuning", Kind: hostreqspec.KindSafeguard, Required: false,
				ExecutionState: vrooliruntime.ExecutionPending,
				BlockingReason: hostreqkit.BlockingOptionalSkipped,
				Provenance:     prov,
			},
		},
	}

	var sb strings.Builder
	renderGrouped(&sb, report)
	out := sb.String()

	if !strings.Contains(out, exePath+" setup --sudo-mode=ask") {
		t.Fatalf("action block should contain the current executable path:\n%s", out)
	}
	if !strings.Contains(out, exePath+" setup --include-optional") {
		t.Fatalf("action block should contain the current executable path:\n%s", out)
	}
	if !strings.Contains(out, "Needs privilege — re-run with `"+exePath+" setup --sudo-mode=ask`") {
		t.Fatalf("privilege group should use the current executable path:\n%s", out)
	}
	if lookups != 1 {
		t.Fatalf("executable lookups = %d, want one per rendered report", lookups)
	}
}

func TestVrooliInvocationFallsBackWhenExecutableLookupFails(t *testing.T) {
	origExe := vrooliExecutableFn
	defer func() { vrooliExecutableFn = origExe }()
	for _, result := range []struct {
		name string
		exe  string
		err  error
	}{
		{name: "error", err: errors.New("lookup failed")},
		{name: "empty path"},
	} {
		t.Run(result.name, func(t *testing.T) {
			vrooliExecutableFn = func() (string, error) { return result.exe, result.err }
			if got := vrooliInvocation(); got != setupOwnerVrooli {
				t.Fatalf("vrooliInvocation = %q, want fallback %q", got, setupOwnerVrooli)
			}
		})
	}
}
