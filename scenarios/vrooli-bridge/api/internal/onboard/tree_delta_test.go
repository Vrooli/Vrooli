package onboard

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"vrooli-bridge/internal/onboard/ssh"
)

// localShellStream runs the remote ship commands in a local POSIX shell with
// HOME pointed at a temp directory, so the tests exercise the exact commands a
// node receives.
func localShellStream(home string) func(context.Context, ssh.ConnectionConfig, string, ssh.StreamOptions) (ssh.Result, error) {
	return func(ctx context.Context, _ ssh.ConnectionConfig, command string, opts ssh.StreamOptions) (ssh.Result, error) {
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Env = append(os.Environ(), "HOME="+home)
		switch {
		case opts.StdinReader != nil:
			cmd.Stdin = opts.StdinReader
		case opts.Stdin != nil:
			cmd.Stdin = bytes.NewReader(opts.Stdin)
		}
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		code := 0
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code, err = exitErr.ExitCode(), nil
		}
		for _, line := range strings.Split(stdout.String(), "\n") {
			if line != "" && opts.OnStdoutLine != nil {
				opts.OnStdoutLine(line)
			}
		}
		return ssh.Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}, err
	}
}

type memShipRecords map[string]shipRecord

func (m memShipRecords) Load(key string) (shipRecord, bool) { r, ok := m[key]; return r, ok }
func (m memShipRecords) Save(key string, r shipRecord) error {
	m[key] = r
	return nil
}

func writeShipFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func repoFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

func shipOnce(t *testing.T, d *sshDriver, repo string) (SyncResult, []string) {
	t.Helper()
	files := repoFiles(t, repo)
	digest, entries, err := digestFileEntries(repo, files)
	if err != nil {
		t.Fatal(err)
	}
	res, err := d.SyncTree(context.Background(), SyncParams{
		Conn: Conn{Host: "node", Port: 22, User: "owner"}, Platform: NodePlatform{OS: "linux", Arch: "amd64"},
		RepoDir: repo, Files: files, Entries: entries, Digest: digest,
	})
	if err != nil {
		t.Fatalf("ship: %v", err)
	}
	return res, files
}

// A ship must leave every path the node owns untouched: the in-tree scenario
// databases, and the checkout's own .git base. Before 2026-09-15 each ship
// swapped the whole directory and deleted them.
func TestSyncTreeUpdatesInPlaceAndKeepsNodeOwnedPaths(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	dest := filepath.Join(home, "vrooli")
	writeShipFile(t, filepath.Join(repo, "a.txt"), "one", 0o644)
	writeShipFile(t, filepath.Join(repo, "dir", "b.txt"), "two", 0o644)
	writeShipFile(t, filepath.Join(repo, "run.sh"), "#!/bin/sh\n", 0o755)
	writeShipFile(t, filepath.Join(dest, "scenarios", "app", "data", "app.db"), "node data", 0o600)
	writeShipFile(t, filepath.Join(dest, ".gitignore"), "scenarios/*/data/\n", 0o644)
	writeShipFile(t, filepath.Join(dest, "stale-from-older-source.go"), "package stale\n", 0o644)
	if err := exec.Command("git", "-C", dest, "init", "-q").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dest, "add", ".gitignore", "stale-from-older-source.go").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dest, "-c", "user.email=test@example.invalid", "-c", "user.name=test", "commit", "-qm", "base").Run(); err != nil {
		t.Fatal(err)
	}
	// A leftover from the retired swap design is cleared by the probe.
	writeShipFile(t, filepath.Join(home, ".vrooli.bridge-old-123", "stale"), "x", 0o644)

	d := &sshDriver{stream: localShellStream(home), shipRecords: memShipRecords{}}
	first, files := shipOnce(t, d, repo)
	if first.ResolvedDestDir != dest || first.Incremental || first.FilesTransferred != len(files) || first.FilesDeleted != 0 {
		t.Fatalf("first ship = %+v, want a full ship of %d files into %s", first, len(files), dest)
	}
	if _, err := os.Stat(filepath.Join(dest, "stale-from-older-source.go")); !os.IsNotExist(err) {
		t.Fatalf("stale non-ignored source file was not cleaned: %v", err)
	}
	if got := readFile(t, filepath.Join(dest, "dir", "b.txt")); got != "two" {
		t.Fatalf("dir/b.txt = %q", got)
	}
	if info, err := os.Stat(filepath.Join(dest, "run.sh")); err != nil || info.Mode()&0o111 == 0 {
		t.Fatalf("run.sh lost its executable bit: %v %v", info, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".vrooli.bridge-old-123")); !os.IsNotExist(err) {
		t.Fatalf("retired swap backup was not removed: %v", err)
	}

	writeShipFile(t, filepath.Join(repo, "a.txt"), "one, edited", 0o644)
	if err := os.RemoveAll(filepath.Join(repo, "dir")); err != nil {
		t.Fatal(err)
	}
	writeShipFile(t, filepath.Join(repo, "c.txt"), "three", 0o644)
	second, files := shipOnce(t, d, repo)
	if second.Incremental || second.FilesTransferred != len(files) || second.FilesDeleted != 1 {
		t.Fatalf("second ship = %+v, want a complete resend of %d files and 1 removed", second, len(files))
	}
	if got := readFile(t, filepath.Join(dest, "a.txt")); got != "one, edited" {
		t.Fatalf("a.txt = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dest, "dir")); !os.IsNotExist(err) {
		t.Fatalf("directory emptied by the removal should be gone: %v", err)
	}
	if got := readFile(t, filepath.Join(dest, "scenarios", "app", "data", "app.db")); got != "node data" {
		t.Fatalf("node-owned database changed: %q", got)
	}
	if got := readFile(t, filepath.Join(dest, ".git", "HEAD")); !strings.HasPrefix(got, "ref:") {
		t.Fatalf("node .git base changed: %q", got)
	}

	// A node whose last ship did not complete is sent everything again.
	writeShipFile(t, filepath.Join(home, ".vrooli.bridge-ship-digest"), "interrupted", 0o644)
	third, files := shipOnce(t, d, repo)
	if third.Incremental || third.FilesTransferred != len(files) {
		t.Fatalf("third ship = %+v, want a full resend after a digest mismatch", third)
	}
}

func TestPlanTreeDeltaNeverDeletesOutsideTheCheckout(t *testing.T) {
	prev := shipRecord{Digest: "d", Entries: map[string]string{
		"kept": "f-:1", "gone": "f-:2", "../escape": "f-:3", "/abs": "f-:4", "a/./b": "f-:5",
	}}
	delta := planTreeDelta([]string{"kept"}, map[string]string{"kept": "f-:1"}, prev, true, "d")
	if strings.Join(delta.Delete, ",") != "gone" {
		t.Fatalf("delete = %v, want only the in-checkout path", delta.Delete)
	}
	if !delta.Incremental || len(delta.Transfer) != 0 {
		t.Fatalf("unchanged file should not be transferred: %+v", delta)
	}
}

func TestPlanTreeDeltaRefreshesGeneratedProtoContracts(t *testing.T) {
	prev := shipRecord{Digest: "d", Entries: map[string]string{
		"packages/proto/gen/go/cli/v1/runtime.pb.go": "f-:same",
		"ordinary.txt": "f-:same",
	}}
	delta := planTreeDelta(
		[]string{"packages/proto/gen/go/cli/v1/runtime.pb.go", "ordinary.txt"},
		map[string]string{
			"packages/proto/gen/go/cli/v1/runtime.pb.go": "f-:same",
			"ordinary.txt": "f-:same",
		}, prev, true, "d",
	)
	if !delta.Incremental || len(delta.Transfer) != 1 || delta.Transfer[0] != "packages/proto/gen/go/cli/v1/runtime.pb.go" {
		t.Fatalf("delta = %+v, want only the generated proto contract re-sent", delta)
	}
}
