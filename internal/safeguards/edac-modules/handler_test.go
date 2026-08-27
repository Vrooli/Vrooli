package edacmodules

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/modules"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type capturedCommand struct {
	Name string
	Args []string
}

var stubAll = edacStubAll

func edacStubAll(t *testing.T) (
	cmds *[]capturedCommand,
	files map[string]string,
	cpuinfo *string,
	mcSlots *bool,
	restore func(),
) {
	t.Helper()
	origRead := hostreqkit.ReadFileFn
	origRun := hostreqkit.RunCommandFn
	origLookPath := hostreqkit.LookPathFn
	origWriteTemp := hostreqkit.WriteTempFileFn
	origStat := modules.StatFn
	origReadCPU := ReadCPUInfoFn
	origMCExists := MCSlotsExistFn
	origRoot := hostreqkit.RunningAsRootFn
	hostreqkit.RunningAsRootFn = func() bool { return true }

	captured := []capturedCommand{}
	fileContents := map[string]string{}
	tempContents := map[string]string{}
	cpu := ""
	mc := false

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if c, ok := fileContents[path]; ok {
			return []byte(c), nil
		}
		return nil, fs.ErrNotExist
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		if name == "install" && len(args) >= 4 {
			tmp := args[len(args)-2]
			dst := args[len(args)-1]
			if c, ok := tempContents[tmp]; ok {
				fileContents[dst] = c
			}
		}
		return nil
	}
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
	tempCounter := 0
	hostreqkit.WriteTempFileFn = func(content string) (string, error) {
		tempCounter++
		path := "/tmp/vrooli-edac-test-" + string(rune('a'+tempCounter%26))
		tempContents[path] = content
		return path, nil
	}
	modules.StatFn = func(path string) (os.FileInfo, error) {
		return nil, fs.ErrNotExist
	}
	ReadCPUInfoFn = func() ([]byte, error) {
		if cpu == "" {
			return nil, fs.ErrNotExist
		}
		return []byte(cpu), nil
	}
	MCSlotsExistFn = func() bool { return mc }

	return &captured, fileContents, &cpu, &mc, func() {
		hostreqkit.ReadFileFn = origRead
		hostreqkit.RunCommandFn = origRun
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.RunningAsRootFn = origRoot
		hostreqkit.WriteTempFileFn = origWriteTemp
		modules.StatFn = origStat
		ReadCPUInfoFn = origReadCPU
		MCSlotsExistFn = origMCExists
	}
}

func newHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "edac_modules", Handler: "edac_modules"})
}

var linuxHost = edacLinuxHost

func edacLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSysctl: true, SupportsSystemd: true}
}

func req(manual bool) hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name: "edac_modules", Kind: hostreqspec.KindSafeguard, Required: true, Manual: manual,
	}
}

const ryzen7000CPUInfo = `processor	: 0
vendor_id	: AuthenticAMD
cpu family	: 25
model		: 97
model name	: AMD Ryzen 9 7950X 16-Core Processor
`

const intelCPUInfo = `processor	: 0
vendor_id	: GenuineIntel
cpu family	: 6
model		: 158
`

// AMD reports "cpu family" in decimal; AMD's marketing names use hex. So
// "Family 15h" (Bulldozer / Piledriver) appears here as decimal 21.
const olderAMDCPUInfo = `processor	: 0
vendor_id	: AuthenticAMD
cpu family	: 21
`

func TestParseCPUInfoExtractsFirstStanza(t *testing.T) {
	vendor, family := parseCPUInfo(ryzen7000CPUInfo)
	if vendor != "AuthenticAMD" || family != "25" {
		t.Errorf("got vendor=%q family=%q", vendor, family)
	}
}

func TestParseCPUInfoEmpty(t *testing.T) {
	vendor, family := parseCPUInfo("")
	if vendor != "" || family != "" {
		t.Errorf("got vendor=%q family=%q", vendor, family)
	}
}

func TestInspectIntelCPUNotApplicable(t *testing.T) {
	_, _, cpu, _, restore := stubAll(t)
	defer restore()
	*cpu = intelCPUInfo
	st := newHandler().Inspect(linuxHost(), req(false))
	if st.SupportClass != hostreqkit.SupportNotApplicable {
		t.Errorf("SupportClass = %q (Intel not yet supported)", st.SupportClass)
	}
	if st.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
}

func TestInspectAMDFamily15hSupported(t *testing.T) {
	_, _, cpu, mc, restore := stubAll(t)
	defer restore()
	*cpu = olderAMDCPUInfo
	*mc = true

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.SupportClass == hostreqkit.SupportNotApplicable {
		t.Errorf("Family 15h should match drivers list; got NotApplicable. Notes=%v", st.Notes)
	}
}

func TestInspectConsumerRyzenNoMCsIsNotApplicable(t *testing.T) {
	_, _, cpu, mc, restore := stubAll(t)
	defer restore()
	*cpu = ryzen7000CPUInfo
	*mc = false // consumer Ryzen — no MCs register

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.SupportClass != hostreqkit.SupportNotApplicable {
		t.Errorf("SupportClass = %q (consumer Ryzen no-ECC should be NotApplicable)", st.SupportClass)
	}
	notes := strings.Join(st.Notes, " | ")
	if !strings.Contains(notes, "ECC not exposed by firmware") {
		t.Errorf("note missing ECC explanation: %q", notes)
	}
}

func TestInspectAMDWithMCsAndDriverLoadedIsAlreadyPresent(t *testing.T) {
	_, files, cpu, mc, restore := stubAll(t)
	defer restore()
	*cpu = ryzen7000CPUInfo
	*mc = true
	files[modules.LoadFilePath("amd64_edac")] = expectedLoadContent("amd64_edac")
	modules.StatFn = func(path string) (os.FileInfo, error) {
		if path == "/sys/module/amd64_edac" {
			return nil, nil
		}
		return nil, fs.ErrNotExist
	}

	st := newHandler().Inspect(linuxHost(), req(false))
	if !st.Applied {
		t.Errorf("expected Applied=true; status=%+v", st)
	}
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
}

func TestInspectAMDWithMCsButDriverMissingIsPending(t *testing.T) {
	_, _, cpu, mc, restore := stubAll(t)
	defer restore()
	*cpu = ryzen7000CPUInfo
	*mc = true

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.Applied {
		t.Error("expected Applied=false")
	}
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Errorf("ExecutionState = %q, want pending", st.ExecutionState)
	}
}

func TestApplyHappyPath(t *testing.T) {
	cmds, _, cpu, mc, restore := stubAll(t)
	defer restore()
	*cpu = ryzen7000CPUInfo
	*mc = true

	st := newHandler().Inspect(linuxHost(), req(false))
	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !out.Applied {
		t.Errorf("Applied = false; status=%+v", out)
	}
	if out.ExecutionState != hostreqkit.ExecutionApplied {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	sawInstall := false
	sawModprobe := false
	for _, c := range *cmds {
		if c.Name == "install" {
			sawInstall = true
		}
		if c.Name == "modprobe" {
			sawModprobe = true
		}
	}
	if !sawInstall || !sawModprobe {
		t.Errorf("missing commands: install=%v modprobe=%v (commands=%v)", sawInstall, sawModprobe, *cmds)
	}
}

func TestApplyModprobeFailureSurfaced(t *testing.T) {
	_, _, cpu, mc, restore := stubAll(t)
	defer restore()
	*cpu = ryzen7000CPUInfo
	*mc = true
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name == "modprobe" {
			return errors.New("synthetic modprobe failure")
		}
		return nil
	}

	st := newHandler().Inspect(linuxHost(), req(false))
	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionFailed {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	if !strings.Contains(strings.Join(out.Notes, " | "), "synthetic modprobe failure") {
		t.Errorf("notes missing root cause: %v", out.Notes)
	}
}

func TestApplyShortCircuitsOnSupportClasses(t *testing.T) {
	type tc struct {
		name string
		sc   hostreqkit.SupportClass
		want hostreqkit.ExecutionState
	}
	cases := []tc{
		{"unsupported", hostreqkit.SupportUnsupported, hostreqkit.ExecutionUnsupported},
		{"not_applicable", hostreqkit.SupportNotApplicable, hostreqkit.ExecutionNotApplicable},
		{"manual_only", hostreqkit.SupportManualOnly, hostreqkit.ExecutionManualActionRequired},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmds, _, _, _, restore := stubAll(t)
			defer restore()
			st := hostreqkit.ItemStatus{SupportClass: c.sc}
			out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{})
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

// A loaded driver and an observable ECC counter are two different facts. When
// the module is loaded but no memory controller registers, nothing is being
// watched — reporting that as applied would tell an operator that memory
// errors are covered when no counter exists to read.
func TestInspectDriverLoadedWithoutMemoryControllersIsNotApplied(t *testing.T) {
	_, files, cpu, mc, restore := stubAll(t)
	defer restore()
	*cpu = ryzen7000CPUInfo
	*mc = false
	files[modules.LoadFilePath("amd64_edac")] = expectedLoadContent("amd64_edac")
	modules.StatFn = func(path string) (os.FileInfo, error) {
		if path == "/sys/module/amd64_edac" {
			return nil, nil
		}
		return nil, fs.ErrNotExist
	}

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.Applied {
		t.Error("a loaded driver with no registered memory controller must not report as applied")
	}
	if st.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Errorf("ExecutionState = %q, want not_applicable", st.ExecutionState)
	}
	if st.SupportClass != hostreqkit.SupportNotApplicable {
		t.Errorf("SupportClass = %q, want not_applicable", st.SupportClass)
	}
	assertNoteFact(t, st.Notes, DriverLoadedNotePrefix+"true")
	assertNoteFact(t, st.Notes, ECCObservableNotePrefix+"false")
}

// The two facts are reported separately on every Linux outcome, so a consumer
// never has to infer one from the other.
func TestInspectReportsDriverAndECCFactsSeparately(t *testing.T) {
	for _, tc := range []struct {
		name           string
		memControllers bool
		wantDriver     string
		wantECC        string
	}{
		{"no controllers registered", false, DriverLoadedNotePrefix + "false", ECCObservableNotePrefix + "false"},
		{"controllers registered", true, DriverLoadedNotePrefix + "false", ECCObservableNotePrefix + "true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, cpu, mc, restore := stubAll(t)
			defer restore()
			*cpu = ryzen7000CPUInfo
			*mc = tc.memControllers

			st := newHandler().Inspect(linuxHost(), req(false))
			assertNoteFact(t, st.Notes, tc.wantDriver)
			assertNoteFact(t, st.Notes, tc.wantECC)
		})
	}
}

func assertNoteFact(t *testing.T, notes []string, want string) {
	t.Helper()
	for _, note := range notes {
		if strings.HasPrefix(note, want) {
			return
		}
	}
	t.Errorf("notes %v do not report %q", notes, want)
}
