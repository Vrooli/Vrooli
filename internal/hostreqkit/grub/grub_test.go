package grub

import (
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// stubAll replaces every package-level seam used by grub.AddCmdlineParams and
// returns a restore func. Each test composes its own behavior by overwriting
// individual *Fn vars after this call.
func stubAll(t *testing.T) (commands *[]capturedCommand, tempContents *[]string, restore func()) {
	t.Helper()
	origRead := hostreqkit.ReadFileFn
	origRun := hostreqkit.RunCommandFn
	origCombined := hostreqkit.CombinedOutputFn
	origLookPath := hostreqkit.LookPathFn
	origWriteTemp := hostreqkit.WriteTempFileFn
	origNow := NowFn
	origValidate := ValidateGrubConfigFn
	origElevation := hostreqkit.ElevationFactsFn
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Platform: "linux", Elevated: true, CanElevate: true, Mechanism: "test"}
	}

	cmds := []capturedCommand{}
	temps := []string{}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		cmds = append(cmds, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		return nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		cmds = append(cmds, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		return nil, nil
	}
	hostreqkit.LookPathFn = func(name string) (string, error) {
		// Default: nothing on PATH except bash for the validator gate.
		if name == "bash" {
			return "/bin/bash", nil
		}
		return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
	}
	hostreqkit.WriteTempFileFn = func(content string) (string, error) {
		temps = append(temps, content)
		return fmt.Sprintf("/tmp/vrooli-grub-test-%d", len(temps)), nil
	}
	NowFn = func() time.Time {
		return time.Date(2026, 5, 6, 22, 30, 0, 0, time.UTC)
	}
	// Default validator: always pass. Tests override per-case.
	ValidateGrubConfigFn = func(content string, opts hostreqkit.EnsureOptions) (bool, string) {
		return true, ""
	}

	return &cmds, &temps, func() {
		hostreqkit.ReadFileFn = origRead
		hostreqkit.RunCommandFn = origRun
		hostreqkit.CombinedOutputFn = origCombined
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.WriteTempFileFn = origWriteTemp
		NowFn = origNow
		ValidateGrubConfigFn = origValidate
		hostreqkit.ElevationFactsFn = origElevation
	}
}

type capturedCommand struct {
	Name string
	Args []string
}

func setReadFile(content string) {
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		return []byte(content), nil
	}
}

func TestParseCmdlineSimpleQuoted(t *testing.T) {
	tokens, quote, err := parseCmdline(`GRUB_CMDLINE_LINUX="quiet splash crashkernel=auto"`)
	if err != nil {
		t.Fatalf("parseCmdline: %v", err)
	}
	if quote != '"' {
		t.Errorf("quote = %q, want %q", quote, `"`)
	}
	want := []cmdlineToken{
		{Param: "quiet"},
		{Param: "splash"},
		{Param: "crashkernel", Value: "auto", HasValue: true},
	}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d (%v)", len(tokens), len(want), tokens)
	}
	for i := range want {
		if tokens[i] != want[i] {
			t.Errorf("token[%d] = %+v, want %+v", i, tokens[i], want[i])
		}
	}
}

func TestParseCmdlineEmptyValue(t *testing.T) {
	tokens, quote, err := parseCmdline(`GRUB_CMDLINE_LINUX=""`)
	if err != nil {
		t.Fatalf("parseCmdline: %v", err)
	}
	if quote != '"' {
		t.Errorf("quote = %q", quote)
	}
	if len(tokens) != 0 {
		t.Errorf("got %d tokens, want 0", len(tokens))
	}
}

func TestParseCmdlineSingleQuoted(t *testing.T) {
	_, quote, err := parseCmdline(`GRUB_CMDLINE_LINUX='quiet splash'`)
	if err != nil {
		t.Fatalf("parseCmdline: %v", err)
	}
	if quote != '\'' {
		t.Errorf("quote = %q, want '", quote)
	}
}

func TestParseCmdlineCommentedOut(t *testing.T) {
	_, _, err := parseCmdline(`# GRUB_CMDLINE_LINUX="quiet"`)
	if err == nil {
		t.Fatal("expected error for missing GRUB_CMDLINE_LINUX")
	}
}

func TestParseCmdlineMissing(t *testing.T) {
	_, _, err := parseCmdline("GRUB_DEFAULT=0\nGRUB_TIMEOUT=5\n")
	if err == nil {
		t.Fatal("expected error for missing GRUB_CMDLINE_LINUX")
	}
}

func TestParseCmdlineDuplicate(t *testing.T) {
	_, _, err := parseCmdline(`GRUB_CMDLINE_LINUX="a"
GRUB_CMDLINE_LINUX="b"
`)
	if err == nil {
		t.Fatal("expected error for duplicate GRUB_CMDLINE_LINUX")
	}
}

func TestParseCmdlineUnterminatedQuote(t *testing.T) {
	_, _, err := parseCmdline(`GRUB_CMDLINE_LINUX="quiet`)
	if err == nil {
		t.Fatal("expected error for unterminated quote")
	}
}

func TestParseCmdlineValueContainsEquals(t *testing.T) {
	tokens, _, err := parseCmdline(`GRUB_CMDLINE_LINUX="console=ttyS0,115200n8"`)
	if err != nil {
		t.Fatalf("parseCmdline: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1", len(tokens))
	}
	if tokens[0].Param != "console" || tokens[0].Value != "ttyS0,115200n8" {
		t.Errorf("token = %+v", tokens[0])
	}
}

func TestHasCmdlineParamPresent(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	setReadFile(`GRUB_CMDLINE_LINUX="quiet crashkernel=512M-:256M"` + "\n")

	present, value, err := HasCmdlineParam("/etc/default/grub", "crashkernel")
	if err != nil {
		t.Fatalf("HasCmdlineParam: %v", err)
	}
	if !present || value != "512M-:256M" {
		t.Errorf("got present=%v value=%q", present, value)
	}
}

func TestHasCmdlineParamBareFlag(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	setReadFile(`GRUB_CMDLINE_LINUX="quiet splash"` + "\n")

	present, value, err := HasCmdlineParam("/etc/default/grub", "splash")
	if err != nil {
		t.Fatalf("HasCmdlineParam: %v", err)
	}
	if !present || value != "" {
		t.Errorf("got present=%v value=%q, want present=true value=\"\"", present, value)
	}
}

func TestHasCmdlineParamAbsent(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	setReadFile(`GRUB_CMDLINE_LINUX="quiet"` + "\n")

	present, _, err := HasCmdlineParam("/etc/default/grub", "ramoops.mem_size")
	if err != nil {
		t.Fatalf("HasCmdlineParam: %v", err)
	}
	if present {
		t.Error("expected absent")
	}
}

func TestHasCmdlineParamReadError(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		return nil, fs.ErrNotExist
	}

	_, _, err := HasCmdlineParam("/etc/default/grub", "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddCmdlineParamsAppendsNew(t *testing.T) {
	cmds, temps, restore := stubAll(t)
	defer restore()
	setReadFile(`# Sample grub config
GRUB_DEFAULT=0
GRUB_CMDLINE_LINUX="quiet splash"
GRUB_TIMEOUT=5
`)

	out, err := AddCmdlineParams(
		"/etc/default/grub",
		[]CmdlineEdit{{Param: "crashkernel", Value: "512M-:256M"}},
		"ask",
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatalf("AddCmdlineParams: %v", err)
	}
	if !out.Changed {
		t.Error("expected Changed=true")
	}
	if out.NewCmdline != "quiet splash crashkernel=512M-:256M" {
		t.Errorf("NewCmdline = %q", out.NewCmdline)
	}
	wantBackup := "/etc/default/grub.vrooli-bak.20260506T223000.000000000Z"
	if out.BackupPath != wantBackup {
		t.Errorf("BackupPath = %q, want %q", out.BackupPath, wantBackup)
	}
	// Two install calls expected: one for backup, one for the new content.
	installCount := 0
	for _, c := range *cmds {
		if c.Name == "install" || (c.Name == "sudo" && len(c.Args) > 0 && c.Args[0] == "install") {
			installCount++
		}
	}
	if installCount != 2 {
		t.Errorf("install calls = %d, want 2 (backup + new). commands=%v", installCount, *cmds)
	}
	// Two tempfiles: one for backup content, one for new content.
	if len(*temps) != 2 {
		t.Errorf("tempfiles written = %d, want 2", len(*temps))
	}
	// New-content tempfile must contain the new GRUB_CMDLINE_LINUX line.
	newTemp := (*temps)[1]
	if !strings.Contains(newTemp, `GRUB_CMDLINE_LINUX="quiet splash crashkernel=512M-:256M"`) {
		t.Errorf("new-content tempfile missing expected line:\n%s", newTemp)
	}
	// Other lines preserved verbatim.
	if !strings.Contains(newTemp, "GRUB_TIMEOUT=5") || !strings.Contains(newTemp, "GRUB_DEFAULT=0") {
		t.Error("non-cmdline lines were not preserved")
	}
}

func TestAddCmdlineParamsReplacesExistingValue(t *testing.T) {
	_, temps, restore := stubAll(t)
	defer restore()
	setReadFile(`GRUB_CMDLINE_LINUX="quiet crashkernel=auto splash"` + "\n")

	out, err := AddCmdlineParams(
		"/etc/default/grub",
		[]CmdlineEdit{{Param: "crashkernel", Value: "512M-:256M"}},
		"ask",
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatalf("AddCmdlineParams: %v", err)
	}
	if !out.Changed {
		t.Error("expected Changed=true")
	}
	// Token order must be preserved (replace in place, don't append).
	if out.NewCmdline != "quiet crashkernel=512M-:256M splash" {
		t.Errorf("NewCmdline = %q (token order should be preserved)", out.NewCmdline)
	}
	if len(*temps) != 2 {
		t.Errorf("tempfiles = %d", len(*temps))
	}
}

func TestAddCmdlineParamsIdempotent(t *testing.T) {
	cmds, temps, restore := stubAll(t)
	defer restore()
	setReadFile(`GRUB_CMDLINE_LINUX="quiet crashkernel=512M-:256M"` + "\n")

	out, err := AddCmdlineParams(
		"/etc/default/grub",
		[]CmdlineEdit{{Param: "crashkernel", Value: "512M-:256M"}},
		"ask",
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatalf("AddCmdlineParams: %v", err)
	}
	if out.Changed {
		t.Error("expected Changed=false for already-applied value")
	}
	if out.BackupPath != "" {
		t.Errorf("BackupPath = %q, want empty for no-change run", out.BackupPath)
	}
	if len(*cmds) != 0 {
		t.Errorf("ran %d commands on no-op apply: %v", len(*cmds), *cmds)
	}
	if len(*temps) != 0 {
		t.Errorf("wrote %d tempfiles on no-op apply", len(*temps))
	}
}

func TestAddCmdlineParamsMixedNewAndExisting(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	setReadFile(`GRUB_CMDLINE_LINUX="quiet ramoops.mem_size=0x100000"` + "\n")

	out, err := AddCmdlineParams(
		"/etc/default/grub",
		[]CmdlineEdit{
			{Param: "ramoops.mem_size", Value: "0x100000"}, // already matches
			{Param: "ramoops.mem_address", Value: "0x70000000"},
			{Param: "ramoops.ecc", Value: "1"},
		},
		"ask",
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatalf("AddCmdlineParams: %v", err)
	}
	if !out.Changed {
		t.Error("expected Changed=true (two new params)")
	}
	if !strings.Contains(out.NewCmdline, "ramoops.mem_address=0x70000000") {
		t.Errorf("missing mem_address: %q", out.NewCmdline)
	}
	if !strings.Contains(out.NewCmdline, "ramoops.ecc=1") {
		t.Errorf("missing ecc: %q", out.NewCmdline)
	}
}

func TestAddCmdlineParamsDryRunWritesNothing(t *testing.T) {
	cmds, temps, restore := stubAll(t)
	defer restore()
	setReadFile(`GRUB_CMDLINE_LINUX="quiet"` + "\n")

	out, err := AddCmdlineParams(
		"/etc/default/grub",
		[]CmdlineEdit{{Param: "crashkernel", Value: "auto"}},
		"ask",
		hostreqkit.EnsureOptions{DryRun: true},
	)
	if err != nil {
		t.Fatalf("AddCmdlineParams: %v", err)
	}
	if !out.Changed {
		t.Error("expected Changed=true (DryRun reports what would happen)")
	}
	if out.BackupPath == "" {
		t.Error("DryRun should still surface the backup path")
	}
	if len(*cmds) != 0 {
		t.Errorf("DryRun ran commands: %v", *cmds)
	}
	if len(*temps) != 0 {
		t.Errorf("DryRun wrote tempfiles: %d", len(*temps))
	}
}

func TestAddCmdlineParamsValidationFailureBlocksWrite(t *testing.T) {
	cmds, temps, restore := stubAll(t)
	defer restore()
	setReadFile(`GRUB_CMDLINE_LINUX="quiet"` + "\n")
	ValidateGrubConfigFn = func(content string, opts hostreqkit.EnsureOptions) (bool, string) {
		return false, "synthetic syntax error"
	}

	_, err := AddCmdlineParams(
		"/etc/default/grub",
		[]CmdlineEdit{{Param: "crashkernel", Value: "auto"}},
		"ask",
		hostreqkit.EnsureOptions{},
	)
	if err == nil {
		t.Fatal("expected error from validation failure")
	}
	if !strings.Contains(err.Error(), "synthetic syntax error") {
		t.Errorf("error = %v, want it to surface validator reason", err)
	}
	if len(*cmds) != 0 {
		t.Errorf("validation failure should run no commands; got %v", *cmds)
	}
	if len(*temps) != 0 {
		t.Errorf("validation failure should write no tempfiles; got %d", len(*temps))
	}
}

func TestAddCmdlineParamsBackupFailureAbortsBeforeWrite(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	setReadFile(`GRUB_CMDLINE_LINUX="quiet"` + "\n")

	callCount := 0
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		callCount++
		// First install call (the backup) fails. The new-content install must
		// not run.
		if callCount == 1 {
			return errors.New("backup failed")
		}
		return nil
	}

	_, err := AddCmdlineParams(
		"/etc/default/grub",
		[]CmdlineEdit{{Param: "crashkernel", Value: "auto"}},
		"ask",
		hostreqkit.EnsureOptions{},
	)
	if err == nil {
		t.Fatal("expected error from backup failure")
	}
	if callCount != 1 {
		t.Errorf("RunCommandFn called %d times; want 1 (backup attempted, new-content install skipped)", callCount)
	}
}

func TestAddCmdlineParamsRejectsEmptyParam(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	setReadFile(`GRUB_CMDLINE_LINUX=""` + "\n")

	_, err := AddCmdlineParams(
		"/etc/default/grub",
		[]CmdlineEdit{{Param: "  ", Value: "x"}},
		"ask",
		hostreqkit.EnsureOptions{},
	)
	if err == nil {
		t.Fatal("expected error for empty Param")
	}
}

func TestAddCmdlineParamsNoEditsIsNoOp(t *testing.T) {
	cmds, _, restore := stubAll(t)
	defer restore()
	setReadFile(`GRUB_CMDLINE_LINUX="quiet"` + "\n")

	out, err := AddCmdlineParams(
		"/etc/default/grub",
		nil,
		"ask",
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatalf("AddCmdlineParams: %v", err)
	}
	if out.Changed {
		t.Error("expected Changed=false")
	}
	if len(*cmds) != 0 {
		t.Errorf("no-edit run executed commands: %v", *cmds)
	}
}

func TestAddCmdlineParamsDefaultsConfigPath(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path != DefaultConfigPath {
			t.Errorf("read path = %q, want default %q", path, DefaultConfigPath)
		}
		return []byte(`GRUB_CMDLINE_LINUX="quiet"` + "\n"), nil
	}

	_, err := AddCmdlineParams("", []CmdlineEdit{{Param: "x", Value: "1"}}, "ask", hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("AddCmdlineParams: %v", err)
	}
}

func TestRenderCmdlinePreservesTokenForm(t *testing.T) {
	got := renderCmdline([]cmdlineToken{
		{Param: "quiet"},
		{Param: "crashkernel", Value: "auto", HasValue: true},
		{Param: "splash"},
	})
	if got != "quiet crashkernel=auto splash" {
		t.Errorf("renderCmdline = %q", got)
	}
}

func TestSortEditsDeterministic(t *testing.T) {
	got := SortEdits([]CmdlineEdit{
		{Param: "ramoops.ecc", Value: "1"},
		{Param: "crashkernel", Value: "auto"},
		{Param: "ramoops.mem_size", Value: "0x100000"},
	})
	wantOrder := []string{"crashkernel", "ramoops.ecc", "ramoops.mem_size"}
	for i, want := range wantOrder {
		if got[i].Param != want {
			t.Errorf("SortEdits[%d].Param = %q, want %q", i, got[i].Param, want)
		}
	}
}
