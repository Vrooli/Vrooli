package pathhygiene

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

func newHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "path_hygiene", Handler: "path_hygiene"})
}

func req(manual bool) hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name: "path_hygiene", Kind: hostreqspec.KindSafeguard, Required: false, Manual: manual,
	}
}

var linuxHost = pathHygieneLinuxHost

func pathHygieneLinuxHost() hostreqkit.Host { return hostreqkit.Host{OS: "linux"} }
func winHost() hostreqkit.Host              { return hostreqkit.Host{OS: "windows"} }

// withHome points the handler at a temporary home and a fixed PATH, and
// restores the package seams afterwards.
func withHome(t *testing.T, pathEnv string, files map[string]string) string {
	t.Helper()
	home := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(home, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	origHome, origPath := homeDirFn, pathEnvFn
	homeDirFn = func() (string, error) { return home, nil }
	pathEnvFn = func() string { return pathEnv }
	t.Cleanup(func() { homeDirFn, pathEnvFn = origHome, origPath })
	return home
}

func notesContain(status hostreqkit.ItemStatus, substr string) bool {
	for _, note := range status.Notes {
		if strings.Contains(note, substr) {
			return true
		}
	}
	return false
}

func TestInspectReportsLegacyLinesAsPending(t *testing.T) {
	withHome(t, "/usr/bin", map[string]string{".bashrc": legacyFixture})

	status := newHandler().Inspect(linuxHost(), req(false))
	if status.Applied {
		t.Fatal("Applied = true, but .bashrc still holds unguarded PATH lines")
	}
	if !notesContain(status, "unguarded Vrooli PATH line") {
		t.Errorf("no note named the unguarded lines; notes = %v", status.Notes)
	}
}

func TestInspectReportsAppliedOnceTheBlockIsCurrent(t *testing.T) {
	managed, _ := Rewrite(legacyFixture)
	withHome(t, "/usr/bin", map[string]string{".bashrc": managed})

	status := newHandler().Inspect(linuxHost(), req(false))
	if !status.Applied {
		t.Fatalf("Applied = false on an already-managed file; notes = %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionAlreadyPresent)
	}
}

func TestApplyRewritesFilesAndIsIdempotent(t *testing.T) {
	home := withHome(t, "/usr/bin", map[string]string{
		".bashrc":  legacyFixture,
		".profile": "# profile\nexport PATH=\"$HOME/.vrooli/bin:$PATH\"\n",
	})
	h := newHandler()

	status, err := h.Apply(linuxHost(), h.Inspect(linuxHost(), req(false)), hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !status.Applied || status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("Apply did not report success: applied=%v state=%q notes=%v", status.Applied, status.ExecutionState, status.Notes)
	}

	for _, name := range []string{".bashrc", ".profile"} {
		data, err := os.ReadFile(filepath.Join(home, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if n := strings.Count(string(data), BeginMarker); n != 1 {
			t.Errorf("%s has %d managed blocks, want 1", name, n)
		}
	}

	// Second Inspect must see nothing left to do — otherwise `vrooli setup`
	// would rewrite these files on every run.
	if again := h.Inspect(linuxHost(), req(false)); !again.Applied {
		t.Errorf("Applied = false immediately after Apply; notes = %v", again.Notes)
	}
}

// A startup file is read by every new shell; a rewrite that lost its
// permissions would be a live hazard, so mode must survive.
func TestApplyPreservesFileMode(t *testing.T) {
	home := withHome(t, "/usr/bin", map[string]string{".bashrc": legacyFixture})
	path := filepath.Join(home, ".bashrc")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	h := newHandler()
	if _, err := h.Apply(linuxHost(), h.Inspect(linuxHost(), req(false)), hostreqkit.EnsureOptions{}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 600", info.Mode().Perm())
	}
}

func TestApplyDryRunWritesNothing(t *testing.T) {
	home := withHome(t, "/usr/bin", map[string]string{".bashrc": legacyFixture})
	h := newHandler()

	status, err := h.Apply(linuxHost(), h.Inspect(linuxHost(), req(false)), hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if status.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Errorf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionWouldApply)
	}
	data, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != legacyFixture {
		t.Error("dry run modified the file")
	}
}

// Only files that already exist are managed: creating a ~/.zshrc on a
// bash-only host would make zsh start reading a file it never had.
func TestApplyDoesNotCreateStartupFilesThatDoNotExist(t *testing.T) {
	home := withHome(t, "/usr/bin", map[string]string{".bashrc": legacyFixture})
	h := newHandler()
	if _, err := h.Apply(linuxHost(), h.Inspect(linuxHost(), req(false)), hostreqkit.EnsureOptions{}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	for _, name := range []string{".zshrc", ".profile"} {
		if _, err := os.Stat(filepath.Join(home, name)); err == nil {
			t.Errorf("%s was created but did not exist before", name)
		}
	}
}

// A host with no startup files at all still needs one, or the CLI is not on
// any interactive PATH. ~/.profile is the POSIX-standard choice.
func TestApplyCreatesProfileWhenNoStartupFileExists(t *testing.T) {
	home := withHome(t, "/usr/bin", nil)
	h := newHandler()
	if _, err := h.Apply(linuxHost(), h.Inspect(linuxHost(), req(false)), hostreqkit.EnsureOptions{}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".profile"))
	if err != nil {
		t.Fatalf("expected ~/.profile to be created: %v", err)
	}
	if !strings.Contains(string(data), BeginMarker) {
		t.Error("created ~/.profile has no managed block")
	}
	if _, err := os.Stat(filepath.Join(home, ".bashrc")); err == nil {
		t.Error(".bashrc was created too; only ~/.profile should be")
	}
}

// Windows user PATH lives in the registry, not in shell startup files.
func TestInspectIsUnsupportedOnWindows(t *testing.T) {
	withHome(t, "/usr/bin", nil)
	status := newHandler().Inspect(winHost(), req(false))
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Errorf("SupportClass = %q, want %q", status.SupportClass, hostreqkit.SupportUnsupported)
	}
}

// Reported, never removed: a competing binary belongs to whoever installed
// it. The note is what turns a silent wrong-binary run into a decision.
func TestInspectReportsAShadowingBinaryWithoutRemovingIt(t *testing.T) {
	home := t.TempDir()
	goBin := filepath.Join(home, "go", "bin")
	vrooliBin := filepath.Join(home, ".vrooli", "bin")
	for _, dir := range []string{goBin, vrooliBin} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	stale := filepath.Join(goBin, "vrooli")
	if err := os.WriteFile(stale, []byte(shelltest.POSIXShebang()+""), 0o755); err != nil {
		t.Fatalf("write stale binary: %v", err)
	}
	managed, _ := Rewrite("")
	if err := os.WriteFile(filepath.Join(home, ".profile"), []byte(managed), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	origHome, origPath := homeDirFn, pathEnvFn
	homeDirFn = func() (string, error) { return home, nil }
	pathEnvFn = func() string { return goBin + string(os.PathListSeparator) + vrooliBin }
	t.Cleanup(func() { homeDirFn, pathEnvFn = origHome, origPath })

	status := newHandler().Inspect(linuxHost(), req(false))
	if !notesContain(status, "another vrooli binary precedes") {
		t.Fatalf("no shadowing note; notes = %v", status.Notes)
	}
	if _, err := os.Stat(stale); err != nil {
		t.Error("the competing binary was removed; it must only be reported")
	}
}

func TestInspectReportsDuplicatePathEntries(t *testing.T) {
	withHome(t, "/usr/bin:/usr/bin:/bin", map[string]string{".profile": mustManaged()})
	status := newHandler().Inspect(linuxHost(), req(false))
	if !notesContain(status, "unique") {
		t.Errorf("no duplicate-entry note; notes = %v", status.Notes)
	}
}

func mustManaged() string {
	out, _ := Rewrite("")
	return out
}
