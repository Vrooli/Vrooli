package mcelog

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type capturedCommand struct {
	Name string
	Args []string
}

type unitState struct {
	Enabled, Active, Masked bool
}

func stubAll(t *testing.T) (cmds *[]capturedCommand, units map[string]*unitState, restore func()) {
	t.Helper()
	origRun := hostreqkit.RunCommandFn
	origCombined := hostreqkit.CombinedOutputFn
	origLookPath := hostreqkit.LookPathFn
	origUnit := UnitStateFn

	captured := []capturedCommand{}
	unitMap := map[string]*unitState{}

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
		UnitStateFn = origUnit
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
	cases := []struct {
		name string
		sc   hostreqkit.SupportClass
		want hostreqkit.ExecutionState
	}{
		{"unsupported", hostreqkit.SupportUnsupported, hostreqkit.ExecutionUnsupported},
		{"not_applicable", hostreqkit.SupportNotApplicable, hostreqkit.ExecutionNotApplicable},
		{"manual_only", hostreqkit.SupportManualOnly, hostreqkit.ExecutionManualActionRequired},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmds, _, restore := stubAll(t)
			defer restore()
			st := hostreqkit.ItemStatus{SupportClass: c.sc}
			out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{})
			if err != nil {
				t.Fatalf("Apply: %v", err)
			}
			if out.ExecutionState != c.want {
				t.Errorf("ExecutionState = %q, want %q", out.ExecutionState, c.want)
			}
			if len(*cmds) != 0 {
				t.Errorf("commands ran: %v", *cmds)
			}
		})
	}
}

func TestNameAndKind(t *testing.T) {
	h := newHandler()
	if h.Name() != "mcelog" {
		t.Errorf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindTool {
		t.Errorf("Kind = %q", h.Kind())
	}
}
