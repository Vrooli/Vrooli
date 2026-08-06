package hostreqkit

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func stubLookups(t *testing.T) func() {
	t.Helper()
	origLookPath := LookPathFn
	origReadFile := ReadFileFn
	origCombinedOutput := CombinedOutputFn
	origRunCommand := RunCommandFn
	origWriteTemp := WriteTempFileFn
	origElevationFacts := ElevationFactsFn
	return func() {
		LookPathFn = origLookPath
		ReadFileFn = origReadFile
		CombinedOutputFn = origCombinedOutput
		RunCommandFn = origRunCommand
		WriteTempFileFn = origWriteTemp
		ElevationFactsFn = origElevationFacts
	}
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
	req.Reasons[0] = "mutated"
	if status.Reasons[0] == "mutated" {
		t.Fatal("BaseStatus must copy Reasons slice")
	}
	req.Notes[0] = "mutated"
	if status.Notes[0] == "mutated" {
		t.Fatal("BaseStatus must copy Notes slice")
	}
	req.Provenance[0].Name = "mutated"
	if status.Provenance[0].Name == "mutated" {
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
		if name != "apt-get" {
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

func TestRunPrivilegedCommandAppliesSudo(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
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
	if gotName != "sudo" {
		t.Fatalf("command = %q, want sudo", gotName)
	}
	if len(gotArgs) < 2 || gotArgs[0] != "sysctl" {
		t.Fatalf("args = %v", gotArgs)
	}
}

func TestRunPrivilegedCommandWithSudoError(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
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

	err := EnsureManagedDir("/etc/test.d", "ask", EnsureOptions{DryRun: true})
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
		if name == "sudo" {
			return "/usr/bin/sudo", nil
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

	err := EnsureManagedDir("/etc/sysctl.d", "ask", EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if gotName != "sudo" {
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

	err := EnsureManagedDir("/etc/test.d", "ask", EnsureOptions{})
	if err == nil {
		t.Fatal("expected error from mkdir failure")
	}
}

func TestInstallManagedContentDryRunSkips(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	var called bool
	RunCommandFn = func(string, []string, EnsureOptions) error {
		called = true
		return nil
	}

	err := InstallManagedContent("/etc/test.conf", "content", "ask", EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("RunCommandFn should not be called during dry-run")
	}
}

func TestInstallManagedContentWritesAndInstalls(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	var gotArgs []string
	RunCommandFn = func(name string, args []string, opts EnsureOptions) error {
		gotArgs = args
		return nil
	}

	err := InstallManagedContent("/etc/test.conf", "hello world", "ask", EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "install -m 0644") {
		t.Fatalf("args = %q, want install -m 0644", joined)
	}
	if !strings.Contains(joined, "/etc/test.conf") {
		t.Fatalf("args = %q, want target path /etc/test.conf", joined)
	}
}

func TestInstallManagedContentCommandFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	RunCommandFn = func(string, []string, EnsureOptions) error {
		return os.ErrPermission
	}

	err := InstallManagedContent("/etc/test.conf", "content", "ask", EnsureOptions{})
	if err == nil {
		t.Fatal("expected error from install failure")
	}
}

func TestInstallManagedExecutableUsesMode0755(t *testing.T) {
	// Mode 0755 is the only intended difference from InstallManagedContent.
	// Lock it in so a future refactor doesn't silently flip the bit.
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	var gotArgs []string
	RunCommandFn = func(name string, args []string, opts EnsureOptions) error {
		gotArgs = args
		return nil
	}

	err := InstallManagedExecutable("/usr/local/bin/foo", "#!/bin/sh\necho hi\n", "ask", EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "install -m 0755") {
		t.Fatalf("args = %q, want install -m 0755", joined)
	}
	if !strings.Contains(joined, "/usr/local/bin/foo") {
		t.Fatalf("args = %q, want target path /usr/local/bin/foo", joined)
	}
}

func TestInstallManagedExecutableDryRunSkips(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	var called bool
	RunCommandFn = func(string, []string, EnsureOptions) error {
		called = true
		return nil
	}

	err := InstallManagedExecutable("/usr/local/bin/foo", "content", "ask", EnsureOptions{DryRun: true})
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
	h := Host{OS: "darwin", SupportsSetup: false}
	err := h.ValidateSetup()
	if err == nil {
		t.Fatal("expected error for unsupported setup")
	}
	if !strings.Contains(err.Error(), "setup") {
		t.Fatalf("error should mention setup: %v", err)
	}
	if !strings.Contains(err.Error(), "darwin") {
		t.Fatalf("error should mention OS: %v", err)
	}
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
	h := Host{OS: "windows", SupportsDevelop: false}
	err := h.ValidateDevelop()
	if err == nil {
		t.Fatal("expected error for unsupported develop")
	}
	if !strings.Contains(err.Error(), "develop") {
		t.Fatalf("error should mention develop: %v", err)
	}
	if !strings.Contains(err.Error(), "windows") {
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

	err := InstallManagedContent("/etc/test.conf", "content", "ask", EnsureOptions{})
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
