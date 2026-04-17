package hostreqkit

import (
	"os"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func stubLookups(t *testing.T) func() {
	t.Helper()
	origLookPath := LookPathFn
	origReadFile := ReadFileFn
	origCombinedOutput := CombinedOutputFn
	origRunCommand := RunCommandFn
	return func() {
		LookPathFn = origLookPath
		ReadFileFn = origReadFile
		CombinedOutputFn = origCombinedOutput
		RunCommandFn = origRunCommand
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
