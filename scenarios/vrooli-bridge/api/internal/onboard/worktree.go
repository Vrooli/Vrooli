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
// onboards without a commit. Required offline source inputs, such as the Proto
// vendor snapshots, are explicitly unignored and checked by the source-closure
// validator below. The .git directory is deliberately NOT shipped; the node
// builds from a plain checkout and its provenance is the base HEAD + digest the
// op records, not an on-node git history.
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

// offlineProtoSourceRoots are source inputs to the repository's offline Buf
// workspace. Unlike dist/ and node_modules/, these files cannot be regenerated
// without contacting BSR, so a working-tree ship must include them.
var offlineProtoSourceRoots = []string{
	"packages/proto/vendor/googleapis",
	"packages/proto/vendor/protovalidate",
}

// bridgeSourceClosureFiles are small control-plane contracts that every plain
// working-tree shipment must carry. They are not generated build output and
// cannot be reconstructed safely on a node: repo-contract.json is the shared
// authority used by the node-side CLI to resolve its runtime-home paths during
// cleanup and maintenance.
var bridgeSourceClosureFiles = []string{
	".vrooli/repo-contract.json",
}

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
	// Keep the contract explicit even if a repository-local Git exclude or a
	// future enumeration change would otherwise omit a tracked control-plane
	// file. The validator below fails before SSH transfer if it is unavailable.
	files = appendRequiredSourceFiles(root, files, bridgeSourceClosureFiles)
	if err := validateWorkingTreeSourceClosure(root, files); err != nil {
		return WorkingTreeSnapshot{}, err
	}

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

// validateWorkingTreeSourceClosure fails before any SSH transfer when the
// working-tree enumeration omits required offline Proto inputs. This catches a
// global-ignore regression or an incomplete local vendor snapshot at the
// control plane, where the remediation is clear, instead of producing a late
// remote buf/validate/validate.proto error.
func validateWorkingTreeSourceClosure(root string, files []string) error {
	listed := make(map[string]struct{}, len(files))
	for _, rel := range files {
		listed[filepath.ToSlash(rel)] = struct{}{}
	}
	for _, rel := range bridgeSourceClosureFiles {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(abs); err != nil {
			return fmt.Errorf("working-tree source is incomplete: required Bridge contract %s is unavailable: %w", rel, err)
		}
		if _, ok := listed[rel]; !ok {
			return fmt.Errorf("working-tree source is incomplete: required Bridge contract %s was not enumerated", rel)
		}
	}

	protoRoot := filepath.Join(root, "packages", "proto")
	if _, err := os.Stat(filepath.Join(protoRoot, "buf.yaml")); os.IsNotExist(err) {
		// A non-Vrooli repository is allowed through this generic source seam;
		// the production Bridge control plane always has packages/proto/buf.yaml.
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect Proto workspace: %w", err)
	}

	var missing []string
	for _, relRoot := range offlineProtoSourceRoots {
		absRoot := filepath.Join(root, filepath.FromSlash(relRoot))
		info, err := os.Stat(absRoot)
		if os.IsNotExist(err) {
			missing = append(missing, relRoot+"/")
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect required Proto source %s: %w", relRoot, err)
		}
		if !info.IsDir() {
			missing = append(missing, relRoot+"/ (directory required)")
			continue
		}
		err = filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if _, ok := listed[filepath.ToSlash(rel)]; !ok {
				missing = append(missing, filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("inspect required Proto source %s: %w", relRoot, err)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	const maxReported = 8
	detail := missing
	if len(detail) > maxReported {
		detail = append(append([]string(nil), detail[:maxReported]...), fmt.Sprintf("... and %d more", len(missing)-maxReported))
	}
	return fmt.Errorf("working-tree source is incomplete: required offline Proto inputs were not enumerated: %s; restore packages/proto/vendor/googleapis and packages/proto/vendor/protovalidate (run 'cd packages/proto && make refresh-vendor' only when a BSR refresh is intended)", strings.Join(detail, ", "))
}

func appendRequiredSourceFiles(root string, files, required []string) []string {
	seen := make(map[string]struct{}, len(files)+len(required))
	for _, rel := range files {
		seen[filepath.ToSlash(rel)] = struct{}{}
	}
	for _, rel := range required {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" {
			continue
		}
		if _, ok := seen[rel]; ok {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
			files = append(files, rel)
			seen[rel] = struct{}{}
		}
	}
	return dedupeSorted(files)
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
