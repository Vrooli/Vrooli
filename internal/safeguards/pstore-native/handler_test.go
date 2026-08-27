package pstorenative

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

type stat struct {
	pstoreActive   bool
	pstoreBackends []string
	efivars        bool
	erst           bool
}

var stubAll = pstoreNativeStubAll

func pstoreNativeStubAll(t *testing.T) (cmds *[]capturedCommand, files map[string]string, state *stat, restore func()) {
	t.Helper()
	origRead := hostreqkit.ReadFileFn
	origRun := hostreqkit.RunCommandFn
	origLookPath := hostreqkit.LookPathFn
	origWriteTemp := hostreqkit.WriteTempFileFn
	origModuleStat := modules.StatFn
	origStat := StatFn
	origPstore := PstoreActiveFn
	origRoot := hostreqkit.RunningAsRootFn
	hostreqkit.RunningAsRootFn = func() bool { return true }

	captured := []capturedCommand{}
	fileContents := map[string]string{}
	tempContents := map[string]string{}
	s := &stat{}

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
		path := "/tmp/vrooli-pstore-native-test-" + strings.Repeat("a", tempCounter)
		tempContents[path] = content
		return path, nil
	}
	modules.StatFn = func(path string) (os.FileInfo, error) {
		// Default: nothing loaded.
		return nil, fs.ErrNotExist
	}
	StatFn = func(path string) (os.FileInfo, error) {
		switch path {
		case EFIVarsDir:
			if s.efivars {
				return nil, nil
			}
		case ERSTACPIPath:
			if s.erst {
				return nil, nil
			}
		}
		return nil, fs.ErrNotExist
	}
	PstoreActiveFn = func() (bool, []string) {
		return s.pstoreActive, append([]string(nil), s.pstoreBackends...)
	}

	return &captured, fileContents, s, func() {
		hostreqkit.ReadFileFn = origRead
		hostreqkit.RunCommandFn = origRun
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.RunningAsRootFn = origRoot
		hostreqkit.WriteTempFileFn = origWriteTemp
		modules.StatFn = origModuleStat
		StatFn = origStat
		PstoreActiveFn = origPstore
	}
}

func newHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "pstore_native", Handler: "pstore_native"})
}

var linuxHost = pstoreNativeLinuxHost

func pstoreNativeLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSysctl: true, SupportsSystemd: true}
}

func req(manual bool) hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name: "pstore_native", Kind: hostreqspec.KindSafeguard, Required: true, Manual: manual,
	}
}

func TestBackendFromEntryParsesKnownForms(t *testing.T) {
	cases := map[string]string{
		"efi-44":          "efi",
		"erst-1024":       "erst",
		"dmesg-ramoops-0": "dmesg-ramoops",
		"console-erst-1":  "console-erst",
		"plain":           "",
		"-42":             "",
	}
	for in, want := range cases {
		if got := backendFromEntry(in); got != want {
			t.Errorf("backendFromEntry(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInspectAlreadyActive(t *testing.T) {
	_, _, s, restore := stubAll(t)
	defer restore()
	s.pstoreActive = true
	s.pstoreBackends = []string{"efi"}

	st := newHandler().Inspect(linuxHost(), req(false))
	if !st.Applied {
		t.Errorf("expected Applied=true; got %+v", st)
	}
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
}

func TestInspectNoBackendsAvailableIsNotApplicable(t *testing.T) {
	_, _, _, restore := stubAll(t)
	defer restore()
	// no efivars, no erst, no active pstore → NotApplicable
	st := newHandler().Inspect(linuxHost(), req(false))
	if st.SupportClass != hostreqkit.SupportNotApplicable {
		t.Errorf("SupportClass = %q", st.SupportClass)
	}
	if st.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
	notes := strings.Join(st.Notes, " | ")
	if !strings.Contains(notes, "pstore_ramoops") {
		t.Errorf("note should redirect to ramoops fallback: %q", notes)
	}
}

func TestInspectEFIAvailableSetsPending(t *testing.T) {
	_, _, s, restore := stubAll(t)
	defer restore()
	s.efivars = true

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Errorf("ExecutionState = %q, want pending", st.ExecutionState)
	}
	if !strings.Contains(strings.Join(st.Notes, " | "), "efi_pstore") {
		t.Error("note should mention efi_pstore candidate")
	}
}

func TestApplyHappyPathEFI(t *testing.T) {
	cmds, _, s, restore := stubAll(t)
	defer restore()
	s.efivars = true

	st := newHandler().Inspect(linuxHost(), req(false))

	// After modprobe, pretend pstore registers an efi backend.
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		*cmds = append(*cmds, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		if name == "modprobe" {
			s.pstoreActive = true
			s.pstoreBackends = []string{"efi"}
		}
		return nil
	}

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
	sawModprobe := false
	for _, c := range *cmds {
		if c.Name == "modprobe" && len(c.Args) > 0 && c.Args[0] == "efi_pstore" {
			sawModprobe = true
		}
	}
	if !sawModprobe {
		t.Errorf("expected modprobe efi_pstore; commands=%v", *cmds)
	}
}

func TestApplyTriesAllAvailableCandidates(t *testing.T) {
	cmds, _, s, restore := stubAll(t)
	defer restore()
	s.efivars = true
	s.erst = true

	st := newHandler().Inspect(linuxHost(), req(false))
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		*cmds = append(*cmds, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		// Don't activate pstore — simulate "modprobe ran but kernel didn't register".
		return nil
	}

	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionFailed {
		t.Errorf("ExecutionState = %q, want failed (no backend registered)", out.ExecutionState)
	}
	// Must have tried both modprobes.
	mods := map[string]bool{}
	for _, c := range *cmds {
		if c.Name == "modprobe" && len(c.Args) > 0 {
			mods[c.Args[0]] = true
		}
	}
	if !mods["efi_pstore"] || !mods["erst"] {
		t.Errorf("expected both modprobes; got %v", mods)
	}
}

func TestApplyAtBootFailureNonFatal(t *testing.T) {
	cmds, _, s, restore := stubAll(t)
	defer restore()
	s.efivars = true

	// install fails (at-boot persistence) but modprobe succeeds and activates pstore.
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		*cmds = append(*cmds, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		if name == "install" {
			return errors.New("synthetic install failure")
		}
		if name == "modprobe" {
			s.pstoreActive = true
			s.pstoreBackends = []string{"efi"}
		}
		return nil
	}

	st := newHandler().Inspect(linuxHost(), req(false))
	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !out.Applied {
		t.Errorf("at-boot persistence failure should not prevent live activation; status=%+v", out)
	}
	notes := strings.Join(out.Notes, " | ")
	if !strings.Contains(notes, "at-boot persistence") {
		t.Errorf("note should record the at-boot failure: %q", notes)
	}
}
