package mcelog

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type capturedCommand struct {
	Name string
	Args []string
}

type unitState struct {
	Enabled, Active, Masked bool
}

var stubAll = mcelogStubAll

func mcelogStubAll(t *testing.T) (cmds *[]capturedCommand, units map[string]*unitState, restore func()) {
	t.Helper()
	origRun := hostreqkit.RunCommandFn
	origCombined := hostreqkit.CombinedOutputFn
	origLookPath := hostreqkit.LookPathFn
	origElevationFacts := hostreqkit.ElevationFactsFn
	origUnit := UnitStateFn
	origEDAC := EDACModuleLoadedFn

	captured := []capturedCommand{}
	unitMap := map[string]*unitState{}
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Platform: "linux", Elevated: true, CanElevate: true, Mechanism: "test"}
	}

	// Default: EDAC module loaded so existing tests don't need to know about
	// the cross-check. Tests that exercise the missing-EDAC path override.
	EDACModuleLoadedFn = func() bool { return true }

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		return nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		return nil, nil
	}
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" || name == "mcelog" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
	UnitStateFn = func(unit string) (bool, bool, bool) {
		if s, ok := unitMap[unit]; ok {
			return s.Enabled, s.Active, s.Masked
		}
		return false, false, false
	}

	return &captured, unitMap, func() {
		hostreqkit.RunCommandFn = origRun
		hostreqkit.CombinedOutputFn = origCombined
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ElevationFactsFn = origElevationFacts
		UnitStateFn = origUnit
		EDACModuleLoadedFn = origEDAC
	}
}

func newHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.ToolManifest{
		Name:           "mcelog",
		Handler:        "mcelog",
		Commands:       []string{"mcelog"},
		VersionArgs:    []string{"--version"},
		DefaultPackage: "mcelog",
	})
}

func aptHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSystemd: true}
}

func req(manual bool) hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name: "mcelog", Kind: hostreqspec.KindTool, Required: true, Manual: manual,
	}
}

func setInstalled() {
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "mcelog" {
			return "/usr/bin/mcelog", nil
		}
		if name == "sudo" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
}

func TestInspectMaskedTreatedAsAlreadyPresent(t *testing.T) {
	_, units, restore := stubAll(t)
	defer restore()
	setInstalled()
	units[ServiceName] = &unitState{Masked: true}

	st := newHandler().Inspect(aptHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q, want already_present (masked == superseded)", st.ExecutionState)
	}
	if !strings.Contains(strings.Join(st.Notes, " | "), "masked") {
		t.Errorf("note should explain masking: %v", st.Notes)
	}
}

func TestInspectInstalledAndActive(t *testing.T) {
	_, units, restore := stubAll(t)
	defer restore()
	setInstalled()
	units[ServiceName] = &unitState{Enabled: true, Active: true}

	st := newHandler().Inspect(aptHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
}

func TestInspectInstalledServiceStopped(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	setInstalled()

	st := newHandler().Inspect(aptHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
}

func TestApplyMaskingRaceTreatedAsAlreadyPresent(t *testing.T) {
	cmds, units, restore := stubAll(t)
	defer restore()
	setInstalled()

	// systemctl enable returns an error, but a re-probe shows the unit got
	// masked between Inspect and Apply (rasdaemon claimed the channel).
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		*cmds = append(*cmds, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		if name == "systemctl" && len(args) >= 1 && args[0] == "enable" {
			units[ServiceName] = &unitState{Masked: true}
			return errors.New("Failed to enable unit: Unit is masked.")
		}
		return nil
	}

	st := newHandler().Inspect(aptHost(), req(false))
	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q, want already_present (masked race)", out.ExecutionState)
	}
}

func TestApplyHappyPath(t *testing.T) {
	_, units, restore := stubAll(t)
	defer restore()

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name == "apt-get" {
			setInstalled()
		}
		if name == "systemctl" && len(args) >= 1 && args[0] == "enable" {
			units[ServiceName] = &unitState{Enabled: true, Active: true}
		}
		return nil
	}

	st := newHandler().Inspect(aptHost(), req(false))
	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
}

func TestApplyDryRunNotInstalled(t *testing.T) {
	cmds, _, restore := stubAll(t)
	defer restore()

	st := newHandler().Inspect(aptHost(), req(false))
	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	for _, c := range *cmds {
		if c.Name == "apt-get" || c.Name == "systemctl" {
			t.Errorf("DryRun ran %s: %v", c.Name, c)
		}
	}
}

func TestApplyShortCircuitsOnSupportClasses(t *testing.T) {
	hostreqkittest.RunApplyShortCircuitCases(t, func(t *testing.T, support hostreqkit.SupportClass) (hostreqkit.ItemStatus, int, error) {
		cmds, _, restore := stubAll(t)
		defer restore()
		out, err := newHandler().Apply(aptHost(), hostreqkit.ItemStatus{SupportClass: support}, hostreqkit.EnsureOptions{})
		return out, len(*cmds), err
	})
}

func TestInspectSupersedesWhenNoAptCandidateAndRasdaemonActive(t *testing.T) {
	_, units, restore := stubAll(t)
	defer restore()
	// mcelog binary not installed (default LookPathFn) and no apt candidate.
	origPkg := PackageInstallableFn
	PackageInstallableFn = func(host hostreqkit.Host, pkg string) bool { return false }
	defer func() { PackageInstallableFn = origPkg }()
	units[rasdaemonServiceName] = &unitState{Enabled: true, Active: true}

	st := newHandler().Inspect(aptHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q, want already_present (superseded)", st.ExecutionState)
	}
	joined := strings.Join(st.Notes, " | ")
	if !strings.Contains(joined, "superseded by rasdaemon") {
		t.Errorf("notes missing supersede explanation: %v", st.Notes)
	}
}

func TestInspectStillPendingWhenRasdaemonInactive(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	origPkg := PackageInstallableFn
	PackageInstallableFn = func(host hostreqkit.Host, pkg string) bool { return false }
	defer func() { PackageInstallableFn = origPkg }()
	// rasdaemon not active in unitMap.

	st := newHandler().Inspect(aptHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Errorf("ExecutionState = %q, want pending (no rasdaemon to supersede)", st.ExecutionState)
	}
}

func TestApplySupersedesOnInstallFailureWhenRasdaemonActive(t *testing.T) {
	cmds, units, restore := stubAll(t)
	defer restore()
	origPkg := PackageInstallableFn
	PackageInstallableFn = func(host hostreqkit.Host, pkg string) bool { return false }
	defer func() { PackageInstallableFn = origPkg }()
	units[rasdaemonServiceName] = &unitState{Enabled: true, Active: true}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		*cmds = append(*cmds, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		if name == "apt-get" {
			return errors.New("E: Package 'mcelog' has no installation candidate")
		}
		return nil
	}

	// Inspect already supersedes (no install needed) since the package is
	// not installable; force a "would-install" status to trigger Apply's
	// install path and the supersede-on-error branch.
	status := hostreqkit.ItemStatus{
		Name:             "mcelog",
		Installed:        false,
		InstallSupported: true,
		PackageName:      "mcelog",
		ExecutionState:   hostreqkit.ExecutionPending,
	}
	out, err := newHandler().Apply(aptHost(), status, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q, want already_present (supersede on install failure)", out.ExecutionState)
	}
	if !strings.Contains(strings.Join(out.Notes, " | "), "superseded by rasdaemon") {
		t.Errorf("notes missing supersede note: %v", out.Notes)
	}
}

func TestInspectMaskedAddsEDACNoteWhenModuleMissing(t *testing.T) {
	_, units, restore := stubAll(t)
	defer restore()
	setInstalled()
	units[ServiceName] = &unitState{Masked: true}
	EDACModuleLoadedFn = func() bool { return false }

	st := newHandler().Inspect(aptHost(), req(false))
	joined := strings.Join(st.Notes, " | ")
	if !strings.Contains(joined, "no EDAC kernel module loaded") {
		t.Errorf("expected EDAC note when module missing; got %v", st.Notes)
	}
	if !strings.Contains(joined, "edac_modules") {
		t.Errorf("expected EDAC note to point at the edac_modules safeguard; got %v", st.Notes)
	}
}

func TestInspectActiveSkipsEDACNoteWhenModulePresent(t *testing.T) {
	_, units, restore := stubAll(t)
	defer restore()
	setInstalled()
	units[ServiceName] = &unitState{Enabled: true, Active: true}
	// EDACModuleLoadedFn defaults to true via stubAll.

	st := newHandler().Inspect(aptHost(), req(false))
	joined := strings.Join(st.Notes, " | ")
	if strings.Contains(joined, "no EDAC kernel module loaded") {
		t.Errorf("EDAC note should not appear when module is loaded; got %v", st.Notes)
	}
}
