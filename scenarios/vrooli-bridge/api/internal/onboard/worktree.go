package onboard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// gitWorkingTreeSource is the production WorkingTreeSource: it enumerates the
// control plane's local working tree with git and computes a deterministic
// content digest over it. It is the single place the onboard domain shells out to
// git for a working-tree ship; the transport (tar-over-ssh) lives in the
// SSHDriver.
//
// The tree it ships is exactly `git ls-files -z -c -o --exclude-standard`:
// tracked files (committed state) PLUS modified-but-uncommitted PLUS
// untracked-but-not-gitignored files. That is the operator's actual working state
// minus ignored build junk — the whole point of the mode is that uncommitted work
// onboards without a commit. The .git directory is deliberately NOT shipped; the
// node builds from a plain checkout and its provenance is the base HEAD + digest
// the op records, not an on-node git history.
type gitWorkingTreeSource struct {
	repoDir string
	run     func(ctx context.Context, dir, name string, args ...string) ([]byte, error)
}

// NewWorkingTreeSource constructs the production WorkingTreeSource over the
// control-plane checkout root. An empty repoDir lets git discover the repository
// from the process working directory (inside the monorepo in production); main.go
// passes the same dir the cprev resolver uses (BRIDGE_CP_REPO_DIR).
func NewWorkingTreeSource(repoDir string) WorkingTreeSource {
	return &gitWorkingTreeSource{
		repoDir: strings.TrimSpace(repoDir),
		run:     execGit,
	}
}

var _ WorkingTreeSource = (*gitWorkingTreeSource)(nil)

// execGit runs one command capturing stdout; stderr rides the error.
func execGit(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), msg)
		}
		return nil, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

func (g *gitWorkingTreeSource) Snapshot(ctx context.Context) (WorkingTreeSnapshot, error) {
	// Resolve the repo root so the returned RepoDir is absolute and the file list
	// is relative to a stable base (the SSHDriver tars from RepoDir).
	rootOut, err := g.run(ctx, g.repoDir, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return WorkingTreeSnapshot{}, fmt.Errorf("resolve control-plane repo root: %w", err)
	}
	root := strings.TrimSpace(string(rootOut))
	if root == "" {
		return WorkingTreeSnapshot{}, fmt.Errorf("control-plane repo root is empty")
	}

	headOut, err := g.run(ctx, root, "git", "rev-parse", "HEAD")
	if err != nil {
		return WorkingTreeSnapshot{}, fmt.Errorf("resolve control-plane HEAD: %w", err)
	}
	base := strings.TrimSpace(string(headOut))

	// -z gives NUL-terminated paths so filenames with spaces/newlines survive; -c
	// tracked, -o untracked, --exclude-standard honours .gitignore.
	lsOut, err := g.run(ctx, root, "git", "ls-files", "-z", "-c", "-o", "--exclude-standard")
	if err != nil {
		return WorkingTreeSnapshot{}, fmt.Errorf("enumerate working tree: %w", err)
	}
	files := splitNUL(lsOut)
	// git ls-files can list the same path twice (once tracked, once as an "other")
	// only in edge cases; dedupe + sort so the digest is deterministic regardless.
	files = dedupeSorted(files)

	digest, err := digestFiles(root, files)
	if err != nil {
		return WorkingTreeSnapshot{}, err
	}

	return WorkingTreeSnapshot{
		BaseHEAD: base,
		Digest:   digest,
		RepoDir:  root,
		Files:    files,
	}, nil
}

// splitNUL splits git's NUL-terminated output into non-empty entries.
func splitNUL(b []byte) []string {
	parts := strings.Split(string(b), "\x00")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// dedupeSorted sorts and de-duplicates a path list so the content digest is
// order-independent.
func dedupeSorted(in []string) []string {
	sort.Strings(in)
	out := in[:0:0]
	var last string
	for i, p := range in {
		if i == 0 || p != last {
			out = append(out, p)
		}
		last = p
	}
	return out
}

// digestFiles computes a deterministic sha256 over the sorted file list: for each
// path, the running hash absorbs "<path>\0<kind><hex-content-hash>\n" where kind
// is 'f' for a regular (or vanished) file and 'l' for a symlink. Entries are
// hashed exactly as writeTarStream ships them — Lstat, never following symlinks:
// a regular file's hash covers its content, a symlink's covers its target string
// (git tracks the link itself, and the monorepo tracks symlinks to directories
// that os.Open would EISDIR on). The kind byte domain-separates the two so a file
// whose content spells a link target cannot collide with the link. A path that
// vanished between enumeration and hashing (a concurrent delete) is absorbed as
// an empty 'f' hash so the digest stays defined rather than erroring. It is
// content-addressed: the same working tree yields the same digest, and any
// content or link-target change changes it.
func digestFiles(root string, files []string) (string, error) {
	h := sha256.New()
	for _, rel := range files {
		fh := sha256.New()
		kind := byte('f')
		abs := filepath.Join(root, rel)
		info, err := os.Lstat(abs)
		switch {
		case os.IsNotExist(err):
			// absorbed as the empty 'f' hash below
		case err != nil:
			return "", fmt.Errorf("stat %s: %w", rel, err)
		case info.Mode()&os.ModeSymlink != 0:
			target, lerr := os.Readlink(abs)
			if lerr != nil {
				return "", fmt.Errorf("readlink %s: %w", rel, lerr)
			}
			kind = 'l'
			fh.Write([]byte(target))
		case info.Mode().IsRegular():
			f, oerr := os.Open(abs)
			if oerr == nil {
				if _, cerr := io.Copy(fh, f); cerr != nil {
					_ = f.Close()
					return "", fmt.Errorf("hash %s: %w", rel, cerr)
				}
				_ = f.Close()
			} else if !os.IsNotExist(oerr) {
				return "", fmt.Errorf("open %s: %w", rel, oerr)
			}
		}
		fmt.Fprintf(h, "%s\x00%c%s\n", rel, kind, hex.EncodeToString(fh.Sum(nil)))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
