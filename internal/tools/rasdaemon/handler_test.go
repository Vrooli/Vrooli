//nolint:goconst // test data deliberately reuses stable command fixtures.
package rasdaemon

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

var stubAll = rasdaemonStubAll

func rasdaemonStubAll(t *testing.T) (cmds *[]capturedCommand, units map[string]struct{ Enabled, Active bool }, restore func()) {
	t.Helper()
	origRun := hostreqkit.RunCommandFn
	origCombined := hostreqkit.CombinedOutputFn
	origLookPath := hostreqkit.LookPathFn
	origElevationFacts := hostreqkit.ElevationFactsFn
	origUnit := UnitStateFn

	captured := []capturedCommand{}
	unitMap := map[string]struct{ Enabled, Active bool }{}
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Platform: "linux", Elevated: true, CanElevate: true, Mechanism: "test"}
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		return nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		return nil, nil
	}
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" || name == "rasdaemon" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
	UnitStateFn = func(unit string) (bool, bool) {
		s := unitMap[unit]
		return s.Enabled, s.Active
	}

	return &captured, unitMap, func() {
		hostreqkit.RunCommandFn = origRun
		hostreqkit.CombinedOutputFn = origCombined
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ElevationFactsFn = origElevationFacts
		UnitStateFn = origUnit
	}
}

func newHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.ToolManifest{
		Name:           "rasdaemon",
		Handler:        "rasdaemon",
		Commands:       []string{"rasdaemon"},
		VersionArgs:    []string{"--version"},
		DefaultPackage: "rasdaemon",
	})
}

func aptHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSystemd: true}
}

func req(manual bool) hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name: "rasdaemon", Kind: hostreqspec.KindTool, Required: true, Manual: manual,
	}
}

func TestInspectNonLinux(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	st := newHandler().Inspect(hostreqkit.Host{OS: "darwin"}, req(false))
	if st.SupportClass != hostreqkit.SupportUnsupported {
		t.Errorf("SupportClass = %q", st.SupportClass)
	}
}

func TestInspectNoSystemd(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	host := hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSystemd: false}
	st := newHandler().Inspect(host, req(false))
	if st.SupportClass != hostreqkit.SupportUnsupported {
		t.Errorf("SupportClass = %q", st.SupportClass)
	}
}

func TestInspectManual(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	st := newHandler().Inspect(aptHost(), req(true))
	if st.SupportClass != hostreqkit.SupportManualOnly {
		t.Errorf("SupportClass = %q", st.SupportClass)
	}
}

func TestInspectNotInstalled(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	st := newHandler().Inspect(aptHost(), req(false))
	if st.Installed {
		t.Error("expected Installed=false")
	}
	if st.PackageName != "rasdaemon" {
		t.Errorf("PackageName = %q", st.PackageName)
	}
	if !st.InstallSupported {
		t.Error("InstallSupported should be true")
	}
}

func TestInspectInstalledAndActive(t *testing.T) {
	_, units, restore := stubAll(t)
	defer restore()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "rasdaemon" {
			return "/usr/sbin/rasdaemon", nil
		}
		if name == "sudo" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
	units[ServiceName] = struct{ Enabled, Active bool }{Enabled: true, Active: true}

	st := newHandler().Inspect(aptHost(), req(false))
	if !st.Installed {
		t.Error("expected Installed=true")
	}
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
}

func TestInspectInstalledServiceStopped(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "rasdaemon" {
			return "/usr/sbin/rasdaemon", nil
		}
		if name == "sudo" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
	// units empty → not enabled, not active

	st := newHandler().Inspect(aptHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Errorf("ExecutionState = %q, want pending (installed but service not running)", st.ExecutionState)
	}
}

func checkApplyInstallsAndEnablesRasdaemon(t *testing.T) {
	cmds, units, restore := stubAll(t)
	defer restore()

	// Initially not installed.
	st := newHandler().Inspect(aptHost(), req(false))

	// Simulate install completing → binary on PATH afterwards, unit started.
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		*cmds = append(*cmds, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		if name == "apt-get" {
			hostreqkit.LookPathFn = func(n string) (string, error) {
				if n == "rasdaemon" {
					return "/usr/sbin/rasdaemon", nil
				}
				if n == "sudo" {
					return "", fs.ErrNotExist
				}
				return "/usr/bin/" + n, nil
			}
		}
		if name == "systemctl" && len(args) >= 1 && args[0] == "enable" {
			units[ServiceName] = struct{ Enabled, Active bool }{Enabled: true, Active: true}
		}
		return nil
	}

	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !out.Installed {
		t.Errorf("Installed = false; status=%+v", out)
	}
	if out.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	sawSystemctl := false
	for _, c := range *cmds {
		if c.Name == "systemctl" && len(c.Args) >= 2 && c.Args[0] == "enable" {
			sawSystemctl = true
		}
	}
	if !sawSystemctl {
		t.Errorf("systemctl enable --now not called: %v", *cmds)
	}
}

func TestApplyDryRunInstalledNotActive(t *testing.T) {
	cmds, _, restore := stubAll(t)
	defer restore()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "rasdaemon" {
			return "/usr/sbin/rasdaemon", nil
		}
		if name == "sudo" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}

	st := newHandler().Inspect(aptHost(), req(false))
	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	for _, c := range *cmds {
		if c.Name == "systemctl" && len(c.Args) > 0 && c.Args[0] != "is-enabled" && c.Args[0] != "is-active" {
			t.Errorf("DryRun ran systemctl mutation: %v", c)
		}
	}
}

func TestApplyAlreadyPresentShortCircuits(t *testing.T) {
	cmds, _, restore := stubAll(t)
	defer restore()
	st := hostreqkit.ItemStatus{
		SupportClass:   hostreqkit.SupportSupported,
		ExecutionState: hostreqkit.ExecutionAlreadyPresent,
		Installed:      true,
	}
	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	if len(*cmds) != 0 {
		t.Errorf("short-circuit ran commands: %v", *cmds)
	}
}

func TestApplySystemctlFailureSurfaced(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "rasdaemon" {
			return "/usr/sbin/rasdaemon", nil
		}
		if name == "sudo" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name == "systemctl" {
			return errors.New("synthetic systemctl failure")
		}
		return nil
	}

	st := newHandler().Inspect(aptHost(), req(false))
	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionFailed {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	if !strings.Contains(strings.Join(out.Notes, " | "), "synthetic systemctl failure") {
		t.Errorf("notes missing root cause: %v", out.Notes)
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
