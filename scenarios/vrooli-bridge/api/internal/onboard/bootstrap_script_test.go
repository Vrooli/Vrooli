package onboard

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runBootstrapFunctions sources bootstrap.sh without running an onboarding and
// runs script against it, with HOME pointed at home.
func runBootstrapFunctions(t *testing.T, home, script string, env ...string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "bootstrap", "bootstrap.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "-c", `BOOTSTRAP_SOURCE_ONLY=1 source "$BOOTSTRAP_PATH"; `+script)
	cmd.Env = append(os.Environ(), append([]string{
		"HOME=" + home, "BOOTSTRAP_PATH=" + path,
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
	}, env...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bootstrap functions failed: %v\n%s", err, out)
	}
	return string(out)
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// Only this run's artifact directory survives, and the install record forgets
// the ones removed (minimouse had 130 of them, 6.8 GB).
func TestBootstrapPrunesEarlierArtifactDirectories(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".local", "lib", "vrooli-bridge", "bootstrap")
	for _, name := range []string{"artifacts-old", "artifacts-current"} {
		writeShipFile(t, filepath.Join(root, name, "vrooli"), "bin", 0o755)
	}
	record := filepath.Join(home, ".vrooli", "state", "install-record.json")
	writeShipFile(t, record, `{"version":1,"entries":[`+
		`{"scope":"agent","kind":"binary","path":"`+filepath.Join(root, "artifacts-old", "vrooli-bridge")+`"},`+
		`{"scope":"agent","kind":"binary","path":"`+filepath.Join(root, "artifacts-current", "vrooli-bridge")+`"},`+
		`{"scope":"runtime","kind":"directory","path":"`+filepath.Join(home, "vrooli")+`"}]}`, 0o600)

	runBootstrapFunctions(t, home, `PREBUILT_MODE=1; VROOLI_BIN_OVERRIDE="$HOME/.local/lib/vrooli-bridge/bootstrap/artifacts-current/vrooli"; step_prune_artifacts`)

	if _, err := os.Stat(filepath.Join(root, "artifacts-old")); !os.IsNotExist(err) {
		t.Fatalf("earlier artifact directory survived: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "artifacts-current", "vrooli")); err != nil {
		t.Fatalf("this run's artifacts were removed: %v", err)
	}
	got := readFile(t, record)
	if strings.Contains(got, "artifacts-old") || !strings.Contains(got, "artifacts-current") || !strings.Contains(got, `"runtime"`) {
		t.Fatalf("install record = %s", got)
	}
}

// A shipped tree gains a git base without any shipped file changing, so Bridge
// can later provision it to a pushed revision.
func TestBootstrapGivesAShippedTreeAGitBase(t *testing.T) {
	origin := t.TempDir()
	git(t, origin, "init", "-q", "-b", "main")
	writeShipFile(t, filepath.Join(origin, "a.txt"), "committed", 0o644)
	git(t, origin, "add", ".")
	git(t, origin, "commit", "-q", "-m", "base")
	sha := git(t, origin, "rev-parse", "HEAD")

	for _, tc := range []struct{ name, revision string }{
		{"base revision is fetchable", sha},
		{"base revision was never pushed", "0123456789abcdef0123456789abcdef01234567"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			tree := filepath.Join(home, "vrooli")
			writeShipFile(t, filepath.Join(tree, "a.txt"), "shipped edit", 0o644)
			// The script assigns these from its own defaults when sourced, so
			// set them after sourcing, as its flag parser would.
			out := runBootstrapFunctions(t, home, `REPO_URL="file://`+origin+`"; REVISION="`+tc.revision+`"; GIT_BASE_BRANCH=main; ensure_git_base "$HOME/vrooli"; printf 'NOTE=%s\n' "$GIT_BASE_NOTE"`)
			if got := git(t, tree, "rev-parse", "HEAD"); got != sha {
				t.Fatalf("HEAD = %s, want %s (%s)", got, sha, out)
			}
			if got := readFile(t, filepath.Join(tree, "a.txt")); got != "shipped edit" {
				t.Fatalf("a shipped file changed: %q", got)
			}
			if !strings.Contains(out, "NOTE=git base ") {
				t.Fatalf("note = %s", out)
			}
		})
	}
}
