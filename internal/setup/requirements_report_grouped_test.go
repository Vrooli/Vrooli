package setup

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

// fakeFileInfo satisfies os.FileInfo for the vrooliLauncherStatFn stub. Only
// the existence (non-nil err) check matters; field values are placeholders.
type fakeFileInfo struct{}

func (fakeFileInfo) Name() string       { return "vrooli" }
func (fakeFileInfo) Size() int64        { return 0 }
func (fakeFileInfo) Mode() os.FileMode  { return 0o755 }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return false }
func (fakeFileInfo) Sys() any           { return nil }

func sampleMixedReport() vrooliruntime.Report {
	prov := []hostreqspec.Provenance{{Kind: "root", Name: "vrooli", Source: ".vrooli/service.json"}}
	return vrooliruntime.Report{
		Environment: "development",
		Host:        vrooliruntime.Host{OS: "linux", PackageManager: "apt-get"},
		Tools: []vrooliruntime.ToolStatus{
			{Name: "git", Kind: hostreq.KindTool, Required: true, ExecutionState: vrooliruntime.ExecutionAlreadyPresent, Provenance: prov},
			{Name: "docker", Kind: hostreq.KindTool, Required: true, ExecutionState: vrooliruntime.ExecutionAlreadyPresent, Provenance: prov},
			{
				Name: "kdump-tools", Kind: hostreq.KindTool, Required: true,
				ExecutionState: vrooliruntime.ExecutionFailed,
				Notes:          []string{"debconf-set-selections failed: exit status 1", "Package kdump-tools not configured"},
				Provenance:     prov,
			},
			{
				Name: "mcelog", Kind: hostreq.KindTool, Required: true,
				ExecutionState: vrooliruntime.ExecutionAlreadyPresent,
				Notes:          []string{"superseded by rasdaemon (no mcelog package available on this distribution)"},
				Provenance:     prov,
			},
		},
		Safeguards: []vrooliruntime.SafeguardStatus{
			{Name: "crashkernel_reserve", Kind: hostreq.KindSafeguard, Required: false, ExecutionState: vrooliruntime.ExecutionPending, Notes: []string{"crashkernel pending: will add crashkernel=512M-:256M"}, Provenance: prov},
			{Name: "edac_modules", Kind: hostreq.KindSafeguard, Required: false, ExecutionState: vrooliruntime.ExecutionNotApplicable, Provenance: prov},
			{Name: "nat_protection", Kind: hostreq.KindSafeguard, Required: false, ExecutionState: vrooliruntime.ExecutionApplied, Provenance: prov},
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
	notApplIdx := strings.Index(out, "Not applicable (1): edac_modules")
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
			{Name: "git", Kind: hostreq.KindTool, Required: true, ExecutionState: vrooliruntime.ExecutionAlreadyPresent, Provenance: prov},
			{Name: "jq", Kind: hostreq.KindTool, Required: true, ExecutionState: vrooliruntime.ExecutionAlreadyPresent, Provenance: prov},
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
	// Stub the launcher seam so action-block hints use the bare
	// `sudo vrooli setup` form (the steady state once the shim is
	// installed). Otherwise the tests would see the test binary's
	// own absolute path under /tmp/go-build*.
	origStat := vrooliLauncherStatFn
	defer func() { vrooliLauncherStatFn = origStat }()
	vrooliLauncherStatFn = func() (os.FileInfo, error) { return fakeFileInfo{}, nil }

	prov := []hostreqspec.Provenance{{Kind: "root", Name: "vrooli", Source: ".vrooli/service.json"}}
	report := vrooliruntime.Report{
		Environment: "development",
		Host:        vrooliruntime.Host{OS: "linux", PackageManager: "apt-get"},
		Tools: []vrooliruntime.ToolStatus{
			{
				Name:           "kdump-tools",
				Kind:           hostreq.KindTool,
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
				Kind:           hostreq.KindSafeguard,
				Required:       false,
				ExecutionState: vrooliruntime.ExecutionPending,
				BlockingReason: hostreqkit.BlockingOptionalSkipped,
				Provenance:     prov,
			},
			{
				Name:           "crashkernel_reserve",
				Kind:           hostreq.KindSafeguard,
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
	if !strings.Contains(out, "sudo vrooli setup") {
		t.Fatalf("action block missing sudo command:\n%s", out)
	}
	if !strings.Contains(out, "vrooli setup --include-optional") {
		t.Fatalf("action block missing include-optional command:\n%s", out)
	}
	// Needs sudo group has the kdump-tools item, NOT the generic Failed group.
	if !strings.Contains(out, "Needs sudo") {
		t.Fatalf("missing Needs sudo group:\n%s", out)
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
	if item, ok := findItemByName(report, "edac_modules"); !ok || item.Kind != hostreq.KindSafeguard {
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

func TestVrooliInvocationReturnsBareWhenLauncherPresent(t *testing.T) {
	origStat := vrooliLauncherStatFn
	origExe := vrooliExecutableFn
	defer func() {
		vrooliLauncherStatFn = origStat
		vrooliExecutableFn = origExe
	}()
	vrooliLauncherStatFn = func() (os.FileInfo, error) { return fakeFileInfo{}, nil }
	exePath := filepath.Join(t.TempDir(), ".vrooli", "bin", "vrooli")
	vrooliExecutableFn = func() (string, error) { return exePath, nil }

	if got := vrooliInvocation(); got != "vrooli" {
		t.Fatalf("vrooliInvocation = %q, want bare 'vrooli' when launcher is present", got)
	}
}

func TestVrooliInvocationFallsBackToAbsolutePathWhenLauncherMissing(t *testing.T) {
	origStat := vrooliLauncherStatFn
	origExe := vrooliExecutableFn
	defer func() {
		vrooliLauncherStatFn = origStat
		vrooliExecutableFn = origExe
	}()
	vrooliLauncherStatFn = func() (os.FileInfo, error) { return nil, fs.ErrNotExist }
	exePath := filepath.Join(t.TempDir(), ".vrooli", "bin", "vrooli")
	vrooliExecutableFn = func() (string, error) { return exePath, nil }

	if got := vrooliInvocation(); got != exePath {
		t.Fatalf("vrooliInvocation = %q, want absolute path fallback", got)
	}
}

func TestVrooliInvocationDegradesGracefullyWhenExecutableLookupFails(t *testing.T) {
	// If both the shim is missing AND os.Executable() fails, the bare name
	// is the least-bad fallback. The operator gets the same "command not
	// found" they had before — no regression — and the hint stays readable.
	origStat := vrooliLauncherStatFn
	origExe := vrooliExecutableFn
	defer func() {
		vrooliLauncherStatFn = origStat
		vrooliExecutableFn = origExe
	}()
	vrooliLauncherStatFn = func() (os.FileInfo, error) { return nil, fs.ErrNotExist }
	vrooliExecutableFn = func() (string, error) { return "", errors.New("os.Executable failed") }

	if got := vrooliInvocation(); got != "vrooli" {
		t.Fatalf("vrooliInvocation = %q, want bare 'vrooli' fallback", got)
	}
}

func TestActionBlockUsesAbsolutePathWhenLauncherMissing(t *testing.T) {
	// Integration: render a report with sudo + optional groups, with the
	// shim absent. Action block should suggest the absolute path; the
	// "Needs sudo" header should match.
	origStat := vrooliLauncherStatFn
	origExe := vrooliExecutableFn
	defer func() {
		vrooliLauncherStatFn = origStat
		vrooliExecutableFn = origExe
	}()
	vrooliLauncherStatFn = func() (os.FileInfo, error) { return nil, fs.ErrNotExist }
	exePath := filepath.Join(t.TempDir(), ".vrooli", "bin", "vrooli")
	vrooliExecutableFn = func() (string, error) { return exePath, nil }

	prov := []hostreqspec.Provenance{{Kind: "root", Name: "vrooli", Source: ".vrooli/service.json"}}
	report := vrooliruntime.Report{
		Environment: "development",
		Host:        vrooliruntime.Host{OS: "linux", PackageManager: "apt-get"},
		Tools: []vrooliruntime.ToolStatus{
			{
				Name: "kdump-tools", Kind: hostreq.KindTool, Required: true,
				ExecutionState: vrooliruntime.ExecutionFailed,
				BlockingReason: hostreqkit.BlockingNeedsSudo,
				Notes:          []string{"sudo skipped"},
				Provenance:     prov,
			},
		},
		Safeguards: []vrooliruntime.SafeguardStatus{
			{
				Name: "tcp_tuning", Kind: hostreq.KindSafeguard, Required: false,
				ExecutionState: vrooliruntime.ExecutionPending,
				BlockingReason: hostreqkit.BlockingOptionalSkipped,
				Provenance:     prov,
			},
		},
	}

	var sb strings.Builder
	renderGrouped(&sb, report)
	out := sb.String()

	if !strings.Contains(out, "sudo "+exePath+" setup") {
		t.Fatalf("action block should contain absolute-path sudo command:\n%s", out)
	}
	if !strings.Contains(out, exePath+" setup --include-optional") {
		t.Fatalf("action block should contain absolute-path include-optional command:\n%s", out)
	}
}
