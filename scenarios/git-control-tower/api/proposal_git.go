package main

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // git blob object IDs are SHA-1 in SHA-1 repositories
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"git-control-tower/internal/proposals"
)

// emptyTreeOID is git's well-known empty tree, used as the base of an unborn
// branch.
const emptyTreeOID = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// proposalGit adapts the git binary to the proposals seam. Reads use
// --no-optional-locks and literal pathspecs; only Stage and RestoreIndex
// write, and only ApplyProposal reaches them under the repository lock and a
// consumed human intent.
type proposalGit struct {
	gitPath string
	repoDir string
	status  func(ctx context.Context) (*RepoStatus, error)
}

func newProposalGit(git GitRunner, repoDir string) *proposalGit {
	return &proposalGit{gitPath: "git", repoDir: repoDir, status: func(ctx context.Context) (*RepoStatus, error) {
		return readRepoStatusSnapshot(ctx, git, repoDir)
	}}
}

var _ proposals.Git = (*proposalGit)(nil)

// proposalGitFor builds the proposals seam for one repository. Transport
// tests substitute proposals.MemoryGit; production uses the git binary.
var proposalGitFor = func(git GitRunner, repoDir string) proposals.Git { return newProposalGit(git, repoDir) }

func (g *proposalGit) read(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	full := append([]string{"--no-optional-locks", "--literal-pathspecs", "-C", g.repoDir}, args...)
	cmd := exec.CommandContext(ctx, g.gitPath, full...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("git %s failed: %w (%s)", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func (g *proposalGit) write(ctx context.Context, stdin []byte, args ...string) error {
	build := func() *exec.Cmd {
		full := append([]string{"--literal-pathspecs", "-C", g.repoDir}, args...)
		cmd := exec.CommandContext(ctx, g.gitPath, full...)
		cmd.Stdin = bytes.NewReader(stdin)
		return cmd
	}
	out, err := execWithIndexLockRetry(ctx, build, runCombined)
	if err != nil {
		return fmt.Errorf("git %s failed: %w (%s)", args[0], err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (g *proposalGit) Head(ctx context.Context) (string, error) {
	out, err := g.read(ctx, nil, "rev-parse", "--verify", "--quiet", "HEAD^{commit}")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil // unborn branch
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *proposalGit) Branch(ctx context.Context) (string, error) {
	out, err := g.read(ctx, nil, "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil // detached HEAD
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *proposalGit) Status(ctx context.Context) ([]proposals.StatusEntry, error) {
	status, err := g.status(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var entries []proposals.StatusEntry
	add := func(path, xy string) {
		if _, ok := seen[path]; ok || path == "" {
			return
		}
		seen[path] = struct{}{}
		entries = append(entries, proposals.StatusEntry{Path: path, XY: xy})
	}
	for path, xy := range status.Files.Statuses {
		add(path, xy)
	}
	for _, path := range status.Files.Untracked {
		add(path, "??")
	}
	// A staged rename leaves its origin deleted in the index.
	for _, origin := range status.Files.Renames {
		add(origin, "D.")
	}
	return entries, nil
}

// Hash returns git's blob ID (with the path's clean filters) and the SHA-256
// of the working-tree bytes. Symlinks hash their target text, as git stores
// them; directories and submodules are marked, never read.
func (g *proposalGit) Hash(ctx context.Context, paths []string) (map[string]proposals.Content, error) {
	result := make(map[string]proposals.Content, len(paths))
	var regular []string
	for _, path := range paths {
		full := filepath.Join(g.repoDir, filepath.FromSlash(path))
		info, err := os.Lstat(full)
		switch {
		case errors.Is(err, os.ErrNotExist):
			result[path] = proposals.Content{Deleted: true}
		case err != nil:
			return nil, err
		case info.IsDir():
			result[path] = proposals.Content{BlobID: proposals.DirectoryBlob}
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(full)
			if err != nil {
				return nil, err
			}
			result[path] = proposals.Content{BlobID: rawBlobID([]byte(target)), SHA256: sha256Hex([]byte(target))}
		case strings.ContainsAny(path, "\n\r"):
			data, err := os.ReadFile(full)
			if err != nil {
				return nil, err
			}
			result[path] = proposals.Content{BlobID: rawBlobID(data), SHA256: sha256Hex(data)}
		default:
			digest, err := sha256File(full)
			if err != nil {
				return nil, err
			}
			result[path] = proposals.Content{SHA256: digest}
			regular = append(regular, path)
		}
	}
	for start := 0; start < len(regular); start += 1000 {
		batch := regular[start:min(start+1000, len(regular))]
		out, err := g.read(ctx, []byte(strings.Join(batch, "\n")+"\n"), "hash-object", "--stdin-paths")
		if err != nil {
			return nil, err
		}
		ids := strings.Fields(string(out))
		if len(ids) != len(batch) {
			return nil, fmt.Errorf("git hash-object returned %d IDs for %d paths", len(ids), len(batch))
		}
		for i, path := range batch {
			content := result[path]
			content.BlobID = ids[i]
			result[path] = content
		}
	}
	return result, nil
}

func (g *proposalGit) IndexEntries(ctx context.Context, paths []string) (map[string]proposals.IndexEntry, error) {
	result := map[string]proposals.IndexEntry{}
	for start := 0; start < len(paths); start += 500 {
		batch := paths[start:min(start+500, len(paths))]
		out, err := g.read(ctx, nil, append([]string{"ls-files", "-s", "-z", "--"}, batch...)...)
		if err != nil {
			return nil, err
		}
		for _, record := range strings.Split(string(out), "\x00") {
			meta, path, ok := strings.Cut(record, "\t")
			if !ok {
				continue
			}
			fields := strings.Fields(meta)
			if len(fields) != 3 || fields[2] != "0" {
				continue
			}
			result[path] = proposals.IndexEntry{Mode: fields[0], BlobID: fields[1]}
		}
	}
	return result, nil
}

func (g *proposalGit) StagedPaths(ctx context.Context) ([]string, error) {
	out, err := g.read(ctx, nil, "diff", "--cached", "--name-only", "--no-renames", "-z")
	if err != nil {
		return nil, err
	}
	return splitNul(out), nil
}

func (g *proposalGit) ChangedPaths(ctx context.Context, from, to string, paths []string) ([]string, error) {
	if from == "" {
		from = emptyTreeOID
	}
	if to == "" {
		to = emptyTreeOID
	}
	var changed []string
	for start := 0; start < len(paths); start += 500 {
		batch := paths[start:min(start+500, len(paths))]
		out, err := g.read(ctx, nil, append([]string{"diff", "--name-only", "--no-renames", "-z", from, to, "--"}, batch...)...)
		if err != nil {
			return nil, err
		}
		changed = append(changed, splitNul(out)...)
	}
	return changed, nil
}

func (g *proposalGit) Ignored(ctx context.Context, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	out, err := g.read(ctx, []byte(strings.Join(paths, "\x00")+"\x00"), "check-ignore", "-z", "--stdin")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil, nil // nothing ignored
		}
		return nil, err
	}
	return splitNul(out), nil
}

func (g *proposalGit) ObjectExists(ctx context.Context, oid string) (bool, error) {
	_, err := g.read(ctx, nil, "cat-file", "-e", oid+"^{commit}")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Stage adds exactly the given paths, deletions included.
func (g *proposalGit) Stage(ctx context.Context, paths []string) error {
	return g.write(ctx, []byte(strings.Join(paths, "\x00")), "add", "--all", "--pathspec-from-file=-", "--pathspec-file-nul")
}

// RestoreIndex writes back the prior stage-0 entries of the given paths and
// removes entries that did not exist before.
func (g *proposalGit) RestoreIndex(ctx context.Context, paths []string, prior map[string]proposals.IndexEntry) error {
	var input strings.Builder
	for _, path := range paths {
		if entry, ok := prior[path]; ok {
			input.WriteString(entry.Mode + " " + entry.BlobID + "\t" + path + "\x00")
			continue
		}
		input.WriteString("0 " + strings.Repeat("0", 40) + "\t" + path + "\x00")
	}
	return g.write(ctx, []byte(input.String()), "update-index", "-z", "--index-info")
}

func (g *proposalGit) CommitFacts(ctx context.Context, oid string) (proposals.CommitFacts, error) {
	out, err := g.read(ctx, nil, "diff-tree", "-r", "--root", "--no-commit-id", "--no-renames", "-z", oid)
	if err != nil {
		return proposals.CommitFacts{}, err
	}
	facts := proposals.CommitFacts{Blobs: map[string]string{}}
	records := strings.Split(string(out), "\x00")
	for i := 0; i+1 < len(records); i += 2 {
		fields := strings.Fields(strings.TrimPrefix(records[i], ":"))
		if len(fields) < 5 {
			continue
		}
		blob := fields[3]
		if strings.HasPrefix(fields[4], "D") {
			blob = proposals.DeletedBlob
		}
		facts.Blobs[records[i+1]] = blob
	}
	message, err := g.read(ctx, nil, "log", "-1", "--format=%B", oid)
	if err != nil {
		return proposals.CommitFacts{}, err
	}
	facts.Message = strings.TrimRight(string(message), "\n")
	return facts, nil
}

func splitNul(out []byte) []string {
	var values []string
	for _, value := range strings.Split(string(out), "\x00") {
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func rawBlobID(data []byte) string {
	sum := sha1.Sum(append([]byte(fmt.Sprintf("blob %d\x00", len(data))), data...)) //nolint:gosec // git object ID
	return hex.EncodeToString(sum[:])
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
