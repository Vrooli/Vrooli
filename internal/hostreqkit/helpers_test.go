package hostreqkit

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	helperTestSudoPath = "/usr/bin/sudo"
	helperTestMutated  = "mutated"
)

func TestRunCommandsHonorContext(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts := EnsureOptions{Context: ctx}

	for name, run := range map[string]func() error{
		"inherited stdin": func() error { return RunCommandFn(executable, nil, opts) },
		"provided stdin":  func() error { return RunCommandInputFn(executable, nil, "private input", opts) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := run(); !errors.Is(err, context.Canceled) {
				t.Fatalf("command error = %v, want context cancellation", err)
			}
		})
	}
}

func stubLookups(t *testing.T) func() {
	t.Helper()
	origLookPath := LookPathFn
	origReadFile := ReadFileFn
	origCombinedOutput := CombinedOutputFn
	origRunCommand := RunCommandFn
	origWriteTemp := WriteTempFileFn
	origElevationFacts := ElevationFactsFn
	origProbePackageState := ProbePackageStateFn
	origRecordPackageInstall := RecordPackageInstallFn
	return func() {
		LookPathFn = origLookPath
		ReadFileFn = origReadFile
		CombinedOutputFn = origCombinedOutput
		RunCommandFn = origRunCommand
		WriteTempFileFn = origWriteTemp
		ElevationFactsFn = origElevationFacts
		ProbePackageStateFn = origProbePackageState
		RecordPackageInstallFn = origRecordPackageInstall
	}
}

func stubAvailableSudo(t *testing.T) func() {
	t.Helper()
	restore := stubLookups(t)
	LookPathFn = func(name string) (string, error) {
		if name == helpersSudo {
			return helperTestSudoPath, nil
		}
		return "", os.ErrNotExist
	}
	return restore
}

func stubUnavailableLookups(t *testing.T) func() {
	t.Helper()
	restore := stubLookups(t)
	LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	return restore
}

func TestBaseStatusCopiesSlices(t *testing.T) {
	req := hostreqspec.ResolvedRequirement{
		Name:     "test-tool",
		Kind:     hostreqspec.KindTool,
		Required: true,
		Reasons:  []string{"reason1"},
		Notes:    []string{"note1"},
		Provenance: []hostreqspec.Provenance{
			{Kind: "scenario", Name: "test"},
		},
	}
	status := BaseStatus(req)

	if status.Name != "test-tool" {
		t.Fatalf("Name = %q", status.Name)
	}
	if !status.Required {
		t.Fatal("Required should be true")
	}
	if status.SupportClass != SupportSupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if status.ExecutionState != ExecutionPending {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}

	// Verify slices are independent copies.
	req.Reasons[0] = helperTestMutated
	if status.Reasons[0] == helperTestMutated {
		t.Fatal("BaseStatus must copy Reasons slice")
	}
	req.Notes[0] = helperTestMutated
	if status.Notes[0] == helperTestMutated {
		t.Fatal("BaseStatus must copy Notes slice")
	}
	req.Provenance[0].Name = helperTestMutated
	if status.Provenance[0].Name == helperTestMutated {
		t.Fatal("BaseStatus must copy Provenance slice")
	}
}

func TestUnsupportedRequirementStatusSetsFields(t *testing.T) {
	req := hostreqspec.ResolvedRequirement{
		Name: "missing",
		Kind: hostreqspec.KindTool,
	}
	status := UnsupportedRequirementStatus(req, "no handler")
	if status.SupportClass != SupportUnsupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if status.ExecutionState != ExecutionUnsupported {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
	if len(status.Notes) != 1 || status.Notes[0] != "no handler" {
		t.Fatalf("Notes = %v", status.Notes)
	}
}

func TestUnsupportedRequirementStatusSkipsBlankNote(t *testing.T) {
	req := hostreqspec.ResolvedRequirement{Name: "x"}
	status := UnsupportedRequirementStatus(req, "   ")
	if len(status.Notes) != 0 {
		t.Fatalf("expected no notes for blank note, got %v", status.Notes)
	}
}

func TestFailureStatusCopiesNotesAndAppendsCause(t *testing.T) {
	status := ItemStatus{ExecutionState: ExecutionPending, Notes: []string{"inspect"}}
	got := FailureStatus(status, errors.New("apply failed"))
	if got.ExecutionState != ExecutionFailed {
		t.Fatalf("ExecutionState = %q, want failed", got.ExecutionState)
	}
	if strings.Join(got.Notes, "|") != "inspect|apply failed" {
		t.Fatalf("Notes = %v, want prior notes and cause", got.Notes)
	}
	got.Notes[0] = helperTestMutated
	if status.Notes[0] != "inspect" {
		t.Fatalf("FailureStatus mutated input notes: %v", status.Notes)
	}
}

func TestResolveCommandFindsFirstAvailable(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "node" {
			return "/usr/bin/node", nil
		}
		return "", os.ErrNotExist
	}

	cmd, found := ResolveCommand([]string{"nodejs", "node"})
	if !found || cmd != "node" {
		t.Fatalf("ResolveCommand = %q, %v", cmd, found)
	}
}

func TestResolveCommandReturnsFalseWhenNoneAvailable(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }

	cmd, found := ResolveCommand([]string{"nonexistent"})
	if found || cmd != "" {
		t.Fatalf("ResolveCommand = %q, %v", cmd, found)
	}
}

func TestResolveCommandSkipsBlankCandidates(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "real" {
			return "/usr/bin/real", nil
		}
		return "", os.ErrNotExist
	}

	cmd, found := ResolveCommand([]string{"", "  ", "real"})
	if !found || cmd != "real" {
		t.Fatalf("ResolveCommand = %q, %v", cmd, found)
	}
}

func TestCommandAvailable(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "git" {
			return "/usr/bin/git", nil
		}
		return "", os.ErrNotExist
	}

	if !CommandAvailable("git") {
		t.Fatal("git should be available")
	}
	if CommandAvailable("missing") {
		t.Fatal("missing should not be available")
	}
}

func TestDetectFirstAvailable(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "dnf" {
			return "/usr/bin/dnf", nil
		}
		return "", os.ErrNotExist
	}

	if got := DetectFirstAvailable([]string{"apt-get", "dnf", "brew"}); got != "dnf" {
		t.Fatalf("DetectFirstAvailable = %q, want dnf", got)
	}
	if got := DetectFirstAvailable([]string{"nope"}); got != "" {
		t.Fatalf("DetectFirstAvailable = %q, want empty", got)
	}
}

func TestReadVersionReturnsFirstLine(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("git version 2.43.0\nextra line\n"), nil
	}

	version := ReadVersion("git", []string{"--version"})
	if version != "git version 2.43.0" {
		t.Fatalf("ReadVersion = %q", version)
	}
}

func TestReadVersionReturnsEmptyOnError(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return nil, os.ErrNotExist
	}

	if got := ReadVersion("git", []string{"--version"}); got != "" {
		t.Fatalf("ReadVersion = %q, want empty", got)
	}
}

func TestReadVersionReturnsEmptyForEmptyInput(t *testing.T) {
	if got := ReadVersion("", []string{"--version"}); got != "" {
		t.Fatalf("ReadVersion with empty command = %q", got)
	}
	if got := ReadVersion("git", nil); got != "" {
		t.Fatalf("ReadVersion with nil args = %q", got)
	}
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello\nworld", "hello"},
		{"  single  ", "single"},
		{"", ""},
		{"\n\n", ""},
		{"one\ntwo\nthree", "one"},
	}
	for _, tt := range tests {
		if got := FirstLine(tt.input); got != tt.want {
			t.Errorf("FirstLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFileContentMatches(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	ReadFileFn = func(path string) ([]byte, error) {
		if path == "/etc/test.conf" {
			return []byte("expected content"), nil
		}
		return nil, os.ErrNotExist
	}

	if !FileContentMatches("/etc/test.conf", "expected content") {
		t.Fatal("should match")
	}
	if FileContentMatches("/etc/test.conf", "different") {
		t.Fatal("should not match different content")
	}
	if FileContentMatches("/etc/missing.conf", "anything") {
		t.Fatal("should not match missing file")
	}
}

func TestRunInstallCommandDelegates(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	var called bool
	RunCommandFn = func(name string, args []string, opts EnsureOptions) error {
		called = true
		if name != helpersAptGet {
			t.Fatalf("command = %q", name)
		}
		return nil
	}

	if err := RunInstallCommand("apt-get", []string{"install", "-y", "jq"}, EnsureOptions{}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("RunCommandFn not called")
	}
}

func TestRunInstallCommandWithProvenanceRecordsProbeOutcomes(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	var recorded []PackageInstallRecord
	RecordPackageInstallFn = func(record PackageInstallRecord) error {
		recorded = append(recorded, record)
		return nil
	}
	LookPathFn = func(name string) (string, error) {
		if name == "brew" {
			return "/opt/homebrew/bin/brew", nil
		}
		return "", os.ErrNotExist
	}
	RunCommandFn = func(string, []string, EnsureOptions) error { return nil }
	CombinedOutputFn = func(_ string, args ...string) ([]byte, error) {
		switch args[len(args)-1] {
		case "absent":
			return []byte("Error: No such keg: absent\n"), errors.New("exit 1")
		case "present":
			return []byte("present 1.2.3\n"), nil
		default:
			return []byte("permission denied\n"), errors.New("exit 1")
		}
	}
	for _, name := range []string{"absent", "present", "probe-failed"} {
		if err := RunInstallCommandWithProvenance("brew", []string{"install", name}, EnsureOptions{}, InstallProvenanceRequest{PackageManager: "brew", PackageName: name}); err != nil {
			t.Fatalf("record %s: %v", name, err)
		}
	}
	if len(recorded) != 3 {
		t.Fatalf("recorded %d packages, want 3", len(recorded))
	}
	if recorded[0].ObservedBefore != PackageAbsent || recorded[0].Action != PackageInstalled {
		t.Fatalf("absent record = %#v", recorded[0])
	}
	if recorded[1].ObservedBefore != PackagePresent || recorded[1].Action != PackageAdopted {
		t.Fatalf("present record = %#v", recorded[1])
	}
	if recorded[2].ObservedBefore != PackageUnknown {
		t.Fatalf("failed probe record = %#v", recorded[2])
	}
}

func TestRunPrivilegedCommandAppliesSudo(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == helpersSudo {
			return helperTestSudoPath, nil
		}
		return "", os.ErrNotExist
	}

	var gotName string
	var gotArgs []string
	RunCommandFn = func(name string, args []string, opts EnsureOptions) error {
		gotName = name
		gotArgs = args
		return nil
	}

	if err := RunPrivilegedCommand("ask", "sysctl", []string{"-p", "/etc/sysctl.conf"}, EnsureOptions{}); err != nil {
		t.Fatal(err)
	}
	if gotName != helpersSudo {
		t.Fatalf("command = %q, want sudo", gotName)
	}
	if len(gotArgs) < 2 || gotArgs[0] != "sysctl" {
		t.Fatalf("args = %v", gotArgs)
	}
}

type setupPackageInstallerStub struct {
	manager     string
	packageName string
	calls       int
}

func (s *setupPackageInstallerStub) InstallPackage(manager, packageName string) error {
	s.manager, s.packageName = manager, packageName
	s.calls++
	return nil
}

func (*setupPackageInstallerStub) InstallAptRepository(string, []byte, string) error { return nil }

func (*setupPackageInstallerStub) InstallOnboardingApplyGrant([]string, []string) error {
	return nil
}

func (*setupPackageInstallerStub) InstallAppArmorProfile(string) error { return nil }

func TestRunPackageInstallUsesSetupSessionOnlyForSudoCommand(t *testing.T) {
	installer := &setupPackageInstallerStub{}
	runCalls := 0
	original := RunCommandFn
	RunCommandFn = func(string, []string, EnsureOptions) error { runCalls++; return nil }
	t.Cleanup(func() { RunCommandFn = original })
	opts := EnsureOptions{SetupPrivilegeSession: installer}
	request := InstallProvenanceRequest{PackageManager: "apt", PackageName: "git"}
	if err := runPackageInstallCommand("sudo", []string{"apt-get", "install", "-y", "git"}, opts, request); err != nil {
		t.Fatal(err)
	}
	if installer.calls != 1 || installer.manager != "apt" || installer.packageName != "git" || runCalls != 0 {
		t.Fatalf("setup session routing: installer=%+v runCalls=%d", installer, runCalls)
	}
	if err := runPackageInstallCommand("apt-get", []string{"install", "-y", "git"}, opts, request); err != nil {
		t.Fatal(err)
	}
	if installer.calls != 1 || runCalls != 1 {
		t.Fatalf("nonprivileged command unexpectedly used setup session: installer=%+v runCalls=%d", installer, runCalls)
	}
}

func TestRunInstallCommandCannotBypassSetupSessionWithoutProvenance(t *testing.T) {
	installer := &setupPackageInstallerStub{}
	originalRecord, originalProbe, originalRun := RecordPackageInstallFn, ProbePackageStateFn, RunCommandFn
	RecordPackageInstallFn = nil
	ProbePackageStateFn = func(string, string) (PackageObservation, string, error) {
		return PackageAbsent, "", nil
	}
	runCalls := 0
	RunCommandFn = func(string, []string, EnsureOptions) error {
		runCalls++
		return nil
	}
	t.Cleanup(func() {
		RecordPackageInstallFn, ProbePackageStateFn, RunCommandFn = originalRecord, originalProbe, originalRun
	})
	opts := EnsureOptions{SetupPrivilegeSession: installer}
	args := []string{"apt-get", "install", "-y", "git"}
	for _, call := range []func() error{
		func() error { return RunInstallCommand("sudo", args, opts) },
		func() error { return RunInstallCommandWithProvenance("sudo", args, opts, InstallProvenanceRequest{}) },
	} {
		err := call()
		if err == nil || !strings.Contains(err.Error(), "provenance recorder is unavailable") {
			t.Fatalf("missing provenance error = %v", err)
		}
	}
	if installer.calls != 2 || installer.manager != "apt-get" || installer.packageName != "git" || runCalls != 0 {
		t.Fatalf("install escaped session: installer=%+v runCalls=%d", installer, runCalls)
	}
}

func TestRunPrivilegedCommandWithSudoError(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == helpersSudo {
			return helperTestSudoPath, nil
		}
		return "", os.ErrNotExist
	}

	RunCommandFn = func(string, []string, EnsureOptions) error { return nil }

	err := RunPrivilegedCommand("skip", "sysctl", []string{"-p"}, EnsureOptions{})
	if err == nil || !strings.Contains(err.Error(), "skip") {
		t.Fatalf("expected skip error, got %v", err)
	}
}

func TestEnsureManagedDirDryRunSkips(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	var called bool
	RunCommandFn = func(string, []string, EnsureOptions) error {
		called = true
		return nil
	}

	err := EnsureManagedDir("/etc/test.d", EnsureOptions{SudoMode: "ask", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("RunCommandFn should not be called during dry-run")
	}
}

func TestEnsureManagedDirRunsMkdir(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == helpersSudo {
			return helperTestSudoPath, nil
		}
		return "", os.ErrNotExist
	}

	var gotName string
	var gotArgs []string
	RunCommandFn = func(name string, args []string, opts EnsureOptions) error {
		gotName = name
		gotArgs = args
		return nil
	}

	err := EnsureManagedDir("/etc/sysctl.d", EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if gotName != helpersSudo {
		t.Fatalf("command = %q, want sudo", gotName)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "mkdir -p /etc/sysctl.d") {
		t.Fatalf("args = %q, want mkdir -p /etc/sysctl.d", joined)
	}
}

func TestEnsureManagedDirMkdirFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	RunCommandFn = func(string, []string, EnsureOptions) error {
		return os.ErrPermission
	}

	err := EnsureManagedDir("/etc/test.d", EnsureOptions{SudoMode: "ask"})
	if err == nil {
		t.Fatal("expected error from mkdir failure")
	}
}

func TestInstallManagedContentDryRunSkips(t *testing.T) {
	assertManagedInstallDryRunSkips(t, func(opts EnsureOptions) error {
		return InstallManagedContent("/etc/test.conf", "content", opts)
	})
}

func TestInstallManagedFileUsesExplicitModes(t *testing.T) {
	for _, tc := range []struct {
		name, path, mode string
		install          func(string, string, EnsureOptions) error
	}{
		{"content", "/etc/test.conf", "644", InstallManagedContent},
		{"executable", "/usr/local/bin/foo", "755", InstallManagedExecutable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			restore := stubLookups(t)
			defer restore()
			LookPathFn = func(name string) (string, error) {
				if name == helpersSudo {
					return helperTestSudoPath, nil
				}
				return "", os.ErrNotExist
			}
			var gotArgs []string
			RunCommandFn = func(_ string, args []string, _ EnsureOptions) error {
				gotArgs = args
				return nil
			}

			if err := tc.install(tc.path, "managed contents", EnsureOptions{SudoMode: "ask"}); err != nil {
				t.Fatal(err)
			}
			joined := strings.Join(gotArgs, " ")
			if !strings.Contains(joined, "install -m "+tc.mode) || !strings.Contains(joined, tc.path) {
				t.Fatalf("install args = %q, want mode %s and target %s", joined, tc.mode, tc.path)
			}
		})
	}
}

func TestInstallManagedContentCommandFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	RunCommandFn = func(string, []string, EnsureOptions) error {
		return os.ErrPermission
	}

	err := InstallManagedContent("/etc/test.conf", "content", EnsureOptions{SudoMode: "ask"})
	if err == nil {
		t.Fatal("expected error from install failure")
	}
}

func TestInstallManagedExecutableDryRunSkips(t *testing.T) {
	assertManagedInstallDryRunSkips(t, func(opts EnsureOptions) error {
		return InstallManagedExecutable("/usr/local/bin/foo", "content", opts)
	})
}

func assertManagedInstallDryRunSkips(t *testing.T, install func(EnsureOptions) error) {
	t.Helper()
	restore := stubLookups(t)
	defer restore()

	var called bool
	RunCommandFn = func(string, []string, EnsureOptions) error {
		called = true
		return nil
	}

	err := install(EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("RunCommandFn should not be called during dry-run")
	}
}

func TestValidateSetupSupported(t *testing.T) {
	h := Host{OS: "linux", SupportsSetup: true}
	if err := h.ValidateSetup(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateSetupUnsupported(t *testing.T) {
	assertUnsupportedHostOperation(t, "darwin", "setup", Host{OS: "darwin"}.ValidateSetup)
}

func TestValidateSetupUnsupportedWithNotes(t *testing.T) {
	h := Host{OS: "darwin", SupportsSetup: false, Notes: []string{"macOS support is experimental", "use Homebrew"}}
	err := h.ValidateSetup()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "macOS support is experimental") {
		t.Fatalf("error should include notes: %v", err)
	}
	if !strings.Contains(err.Error(), "use Homebrew") {
		t.Fatalf("error should include all notes: %v", err)
	}
}

func TestValidateSetupUnsupportedEmptyOS(t *testing.T) {
	h := Host{SupportsSetup: false}
	err := h.ValidateSetup()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "this platform") {
		t.Fatalf("empty OS should say 'this platform': %v", err)
	}
}

func TestValidateDevelopSupported(t *testing.T) {
	h := Host{OS: "linux", SupportsDevelop: true}
	if err := h.ValidateDevelop(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateDevelopUnsupported(t *testing.T) {
	assertUnsupportedHostOperation(t, "windows", "develop", Host{OS: "windows"}.ValidateDevelop)
}

func assertUnsupportedHostOperation(t *testing.T, osName, operation string, validate func() error) {
	t.Helper()
	err := validate()
	if err == nil {
		t.Fatalf("expected error for unsupported %s", operation)
	}
	if !strings.Contains(err.Error(), operation) {
		t.Fatalf("error should mention %s: %v", operation, err)
	}
	if !strings.Contains(err.Error(), osName) {
		t.Fatalf("error should mention OS: %v", err)
	}
}

func TestValidateDevelopUnsupportedWithNotes(t *testing.T) {
	h := Host{OS: "other", SupportsDevelop: false, Notes: []string{"unknown platform detected"}}
	err := h.ValidateDevelop()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown platform detected") {
		t.Fatalf("error should include notes: %v", err)
	}
}

func TestUnsupportedErrorWrapsErrUnsupportedPlatform(t *testing.T) {
	h := Host{OS: "freebsd", SupportsSetup: false}
	err := h.ValidateSetup()
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("error should wrap ErrUnsupportedPlatform: %v", err)
	}
}

func TestWriterOrDiscard(t *testing.T) {
	w := writerOrDiscard(nil)
	if w == nil {
		t.Fatal("should not return nil")
	}
	w = writerOrDiscard(os.Stdout)
	if w != os.Stdout {
		t.Fatal("should return the provided writer")
	}
}

func TestInstallManagedContentTempFileFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	WriteTempFileFn = func(string) (string, error) {
		return "", errors.New("disk full")
	}

	err := InstallManagedContent("/etc/test.conf", "content", EnsureOptions{SudoMode: "ask"})
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("expected disk full error, got %v", err)
	}
	if !strings.Contains(err.Error(), "/etc/test.conf") {
		t.Fatalf("error should mention target path: %v", err)
	}
}

func TestRunVerificationCheckNil(t *testing.T) {
	passed, detail := RunVerificationCheck(nil)
	if !passed {
		t.Fatal("nil verification should pass")
	}
	if detail != "" {
		t.Fatalf("detail = %q, want empty", detail)
	}
}

func TestVersionMatches(t *testing.T) {
	for _, test := range []struct {
		name     string
		observed string
		expected string
		want     bool
	}{
		{name: "go version", observed: "go version go1.25.12 linux/amd64", expected: "1.25.12", want: true},
		{name: "v prefix", observed: "pnpm  v10.19.0", expected: "10.19.0", want: true},
		{name: "partial patch is rejected", observed: "go version go1.25.1 linux/amd64", expected: "1.25.12", want: false},
		{name: "other version is rejected", observed: "go version go1.25.11 linux/amd64", expected: "1.25.12", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := VersionMatches(test.observed, test.expected); got != test.want {
				t.Fatalf("VersionMatches(%q, %q) = %t, want %t", test.observed, test.expected, got, test.want)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	for _, test := range []struct {
		left  string
		right string
		want  int
	}{
		{left: "v20.11.1", right: "20.0.0", want: 1},
		{left: "10.14", right: "10.14.0", want: 0},
		{left: "node v18.19.0", right: "20.0.0", want: -1},
	} {
		if got := CompareVersions(test.left, test.right); got != test.want {
			t.Fatalf("CompareVersions(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}

func TestRunVerificationCheckFilesPass(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	ReadFileFn = func(path string) ([]byte, error) {
		return []byte("content"), nil
	}

	hc := &VerificationCheck{
		Files: []string{"/etc/sysctl.d/99-test.conf", "/etc/systemd/test.conf"},
	}
	passed, detail := RunVerificationCheck(hc)
	if !passed {
		t.Fatalf("expected pass, got detail: %s", detail)
	}
}

func TestRunVerificationCheckFilesMissing(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	ReadFileFn = func(path string) ([]byte, error) {
		if path == "/etc/exists.conf" {
			return []byte("ok"), nil
		}
		return nil, os.ErrNotExist
	}

	hc := &VerificationCheck{
		Files: []string{"/etc/exists.conf", "/etc/missing.conf"},
	}
	passed, detail := RunVerificationCheck(hc)
	if passed {
		t.Fatal("expected failure for missing file")
	}
	if !strings.Contains(detail, "/etc/missing.conf") {
		t.Fatalf("detail = %q, want mention of missing file", detail)
	}
}

func TestRunVerificationCheckCommandPass(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("ok\n"), nil
	}

	hc := &VerificationCheck{
		Command: "docker",
		Args:    []string{"info"},
	}
	passed, detail := RunVerificationCheck(hc)
	if !passed {
		t.Fatalf("expected pass, got detail: %s", detail)
	}
	if detail != "" {
		t.Fatalf("detail = %q, want empty", detail)
	}
}

func TestRunVerificationCheckCommandFail(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("Cannot connect to the Docker daemon\n"), errors.New("exit status 1")
	}

	hc := &VerificationCheck{
		Command: "docker",
		Args:    []string{"info"},
	}
	passed, detail := RunVerificationCheck(hc)
	if passed {
		t.Fatal("expected failure")
	}
	if !strings.Contains(detail, "docker") {
		t.Fatalf("detail = %q, want mention of docker", detail)
	}
	if !strings.Contains(detail, "Cannot connect") {
		t.Fatalf("detail = %q, want command output", detail)
	}
}

func TestRunVerificationCheckCommandFailNoOutput(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return nil, errors.New("command not found")
	}

	hc := &VerificationCheck{
		Command: "missing-tool",
	}
	passed, detail := RunVerificationCheck(hc)
	if passed {
		t.Fatal("expected failure")
	}
	if !strings.Contains(detail, "command not found") {
		t.Fatalf("detail = %q, want error message", detail)
	}
}

func TestRunVerificationCheckFilesAndCommand(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	ReadFileFn = func(string) ([]byte, error) {
		return []byte("ok"), nil
	}
	CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return []byte("healthy\n"), nil
	}

	hc := &VerificationCheck{
		Files:   []string{"/etc/test.conf"},
		Command: "check",
		Args:    []string{"--status"},
	}
	passed, detail := RunVerificationCheck(hc)
	if !passed {
		t.Fatalf("expected pass, got: %s", detail)
	}
}

func TestRunVerificationCheckFilesFailShortCircuitsCommand(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	ReadFileFn = func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	}
	var commandCalled bool
	CombinedOutputFn = func(string, ...string) ([]byte, error) {
		commandCalled = true
		return nil, nil
	}

	hc := &VerificationCheck{
		Files:   []string{"/etc/missing.conf"},
		Command: "check",
	}
	passed, _ := RunVerificationCheck(hc)
	if passed {
		t.Fatal("expected failure")
	}
	if commandCalled {
		t.Fatal("command should not run when file check fails")
	}
}
