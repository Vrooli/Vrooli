package pathhygiene

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The shape observed on the development host: eight retired installers,
// each appending its own captioned PATH line, in both the append and
// prepend forms, with runs of bare lines where a caption was written once
// for several appends.
const legacyFixture = `# ~/.bashrc
export GOPATH="$HOME/go"
export PATH="$GOPATH/bin:$PATH"

# Vrooli CLI tools
export PATH="$PATH:/home/dev/.vrooli/bin"

# Vrooli CLI tools
export PATH="$PATH:/home/dev/.vrooli/bin"
export PATH="$PATH:/home/dev/.vrooli/bin"
export PATH="$HOME/.vrooli/bin:$PATH"

# Added by scenario-to-android
export PATH="$PATH:/home/dev/.vrooli/bin"

# Added by email-triage CLI installation
export PATH="$HOME/.vrooli/bin:$PATH"

# opencode
export PATH=/home/dev/.opencode/bin:$PATH
`

func TestRewriteCollapsesEveryLegacyFormIntoOneBlock(t *testing.T) {
	got, findings := Rewrite(legacyFixture)

	if findings.LegacyLines != 6 {
		t.Errorf("LegacyLines = %d, want 6", findings.LegacyLines)
	}
	if findings.CaptionLines != 4 {
		t.Errorf("CaptionLines = %d, want 4", findings.CaptionLines)
	}
	if n := strings.Count(got, BeginMarker); n != 1 {
		t.Fatalf("managed block appears %d times, want exactly 1", n)
	}
	if strings.Contains(got, ".vrooli/bin:$PATH\"") || strings.Contains(got, "$PATH:/home/dev/.vrooli/bin") {
		t.Error("a legacy unguarded PATH line survived the rewrite")
	}
	// Unrelated content must be untouched.
	for _, want := range []string{`export GOPATH="$HOME/go"`, "# opencode", `export PATH=/home/dev/.opencode/bin:$PATH`} {
		if !strings.Contains(got, want) {
			t.Errorf("rewrite dropped unrelated line %q", want)
		}
	}
}

// The whole point of the marker delimiters: running the safeguard again
// must replace, never append. Without this the safeguard would reproduce
// the very bug it exists to fix.
func TestRewriteIsIdempotent(t *testing.T) {
	once, _ := Rewrite(legacyFixture)
	twice, findings := Rewrite(once)

	if twice != once {
		t.Error("second rewrite changed the file; the managed block is not idempotent")
	}
	if !findings.AlreadyCurrent {
		t.Error("AlreadyCurrent = false on an already-managed file")
	}
	if n := strings.Count(twice, BeginMarker); n != 1 {
		t.Fatalf("managed block appears %d times after two rewrites, want 1", n)
	}
}

// Regression guard: the block's position is remembered through the removal
// pass with a sentinel. Tracking a raw index instead would drift by the
// number of legacy lines deleted above the block, silently relocating it —
// which matters because position determines which later PATH edits the
// block runs before.
func TestRewriteKeepsAnExistingBlockWhereItSits(t *testing.T) {
	input := "# top\n" +
		"export PATH=\"$PATH:/home/dev/.vrooli/bin\"\n" +
		"export PATH=\"$PATH:/home/dev/.vrooli/bin\"\n" +
		"export PATH=\"$PATH:/home/dev/.vrooli/bin\"\n" +
		BeginMarker + "\n# stale contents\n" + EndMarker + "\n" +
		"# MARKER-AFTER\n"

	got, findings := Rewrite(input)
	if !findings.ReplacedBlock {
		t.Fatal("ReplacedBlock = false, want true")
	}
	if findings.InsertedBlock {
		t.Error("InsertedBlock = true, but a block already existed")
	}
	if strings.Contains(got, "# stale contents") {
		t.Error("stale block contents survived")
	}
	lines := strings.Split(got, "\n")
	blockIdx, afterIdx := -1, -1
	for i, l := range lines {
		switch strings.TrimSpace(l) {
		case BeginMarker:
			blockIdx = i
		case "# MARKER-AFTER":
			afterIdx = i
		}
	}
	if blockIdx < 0 || afterIdx < 0 {
		t.Fatalf("expected both markers present, got block=%d after=%d", blockIdx, afterIdx)
	}
	if blockIdx > afterIdx {
		t.Error("block moved below content that was originally after it")
	}
	if strings.TrimSpace(lines[0]) != "# top" {
		t.Errorf("first line = %q, want the original %q", lines[0], "# top")
	}
}

// A comment that is not an installer caption must survive even when it sits
// directly above a removed line — the rewrite trims captions, not context.
func TestRewriteKeepsNonInstallerCommentsAboveLegacyLines(t *testing.T) {
	input := "# IMPORTANT: audited by security 2026-01-01\n" +
		"export PATH=\"$PATH:/home/dev/.vrooli/bin\"\n"

	got, findings := Rewrite(input)
	if findings.LegacyLines != 1 {
		t.Fatalf("LegacyLines = %d, want 1", findings.LegacyLines)
	}
	if findings.CaptionLines != 0 {
		t.Errorf("CaptionLines = %d, want 0 — the comment is not an installer caption", findings.CaptionLines)
	}
	if !strings.Contains(got, "audited by security") {
		t.Error("an unrelated comment was removed")
	}
}

// A begin marker with no end (a half-finished hand edit, or an interrupted
// write) must not leave a partial block that the next rewrite appends after.
func TestRewriteRepairsAnUnterminatedBlock(t *testing.T) {
	input := "# top\n" + BeginMarker + "\nsome half-written contents\n"

	got, _ := Rewrite(input)
	if n := strings.Count(got, BeginMarker); n != 1 {
		t.Fatalf("begin marker appears %d times, want 1", n)
	}
	if n := strings.Count(got, EndMarker); n != 1 {
		t.Fatalf("end marker appears %d times, want 1", n)
	}
	if strings.Contains(got, "half-written") {
		t.Error("orphaned block contents survived")
	}
}

func TestRewriteOnEmptyFileAppendsTheBlock(t *testing.T) {
	got, findings := Rewrite("")
	if !findings.InsertedBlock {
		t.Error("InsertedBlock = false on a file with no block")
	}
	if !strings.Contains(got, BeginMarker) || !strings.Contains(got, EndMarker) {
		t.Fatal("block not present after rewriting an empty file")
	}
}

// ManagedBlock is shell source embedded in a Go string: no Go test would
// catch a syntax error or a wrong expansion in it. Execute it for real
// under every available POSIX shell and assert the PATH it produces.
//
// The "moved to front" cases are the ones that matter. A presence-only
// guard passes "absent" and "already first" while still leaving a stale
// binary ahead of the canonical directory in every other case.
func TestManagedBlockProducesOneLeadingEntryUnderPOSIXShells(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".vrooli", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}
	blockPath := filepath.Join(t.TempDir(), "block.sh")
	if err := os.WriteFile(blockPath, []byte(ManagedBlock+"\n"), 0o644); err != nil {
		t.Fatalf("write block: %v", err)
	}

	cases := []struct{ name, input, want string }{
		{"absent", "/usr/bin:/bin", bin + ":/usr/bin:/bin"},
		{"already first", bin + ":/usr/bin", bin + ":/usr/bin"},
		{"in middle moves to front", "/usr/bin:" + bin + ":/bin", bin + ":/usr/bin:/bin"},
		{"at end moves to front", "/usr/bin:/bin:" + bin, bin + ":/usr/bin:/bin"},
		{"duplicates collapse to one", bin + ":/usr/bin:" + bin + ":/bin:" + bin, bin + ":/usr/bin:/bin"},
		{"adjacent duplicates", "/usr/bin:" + bin + ":" + bin + ":/bin", bin + ":/usr/bin:/bin"},
		{"sole entry", bin, bin},
	}

	shells := availableShells(t)
	for _, shell := range shells {
		for _, tc := range cases {
			t.Run(shell+"/"+tc.name, func(t *testing.T) {
				cmd := exec.Command(shell, "-c", `PATH="$1"; . "$2"; printf '%s' "$PATH"`, "_", tc.input, blockPath)
				cmd.Env = append(os.Environ(), "HOME="+home)
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("%s failed: %v (output %q)", shell, err, out)
				}
				if string(out) != tc.want {
					t.Errorf("PATH = %q, want %q", out, tc.want)
				}
			})
		}
	}
}

// With no ~/.vrooli/bin the block must leave PATH untouched, so a host that
// has not installed the CLI does not get a phantom entry.
func TestManagedBlockIsANoOpWithoutTheDirectory(t *testing.T) {
	blockPath := filepath.Join(t.TempDir(), "block.sh")
	if err := os.WriteFile(blockPath, []byte(ManagedBlock+"\n"), 0o644); err != nil {
		t.Fatalf("write block: %v", err)
	}
	for _, shell := range availableShells(t) {
		cmd := exec.Command(shell, "-c", `PATH="$1"; . "$2"; printf '%s' "$PATH"`, "_", "/usr/bin:/bin", blockPath)
		cmd.Env = append(os.Environ(), "HOME="+filepath.Join(t.TempDir(), "absent"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s failed: %v (output %q)", shell, err, out)
		}
		if string(out) != "/usr/bin:/bin" {
			t.Errorf("%s: PATH = %q, want it unchanged", shell, out)
		}
	}
}

func availableShells(t *testing.T) []string {
	t.Helper()
	var found []string
	for _, shell := range []string{"sh", "dash", "bash", "zsh"} {
		if path, err := exec.LookPath(shell); err == nil {
			found = append(found, path)
		}
	}
	if len(found) == 0 {
		t.Skip("no POSIX shell available to execute the managed block")
	}
	return found
}
