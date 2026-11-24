package stagedmessages

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"git-control-tower/internal/gitpaths"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

const (
	maxFileBytes  = 128 << 10
	maxTotalBytes = 4 << 20
	// Manifest identity is independent of the bounded explanatory detail sample.
	MaxManifestFiles = 4096
	maxDetailFiles   = 64
	maxPatchBytes    = 96 << 10
)

type (
	FileBytes struct{ Before, After []byte }
	Captured  struct {
		Snapshot *v1.StagedMessageSnapshot
		Bytes    map[string]FileBytes
	}
)
type Snapshotter interface {
	Capture(context.Context, string, string) (Captured, error)
	Identity(context.Context, string, string) (string, error)
}

// CaptureWorktreeSnapshot binds exactly the requested unstaged paths to their
// index and worktree contents. It deliberately does not stage or write objects.
func CaptureWorktreeSnapshot(ctx context.Context, repo, dir string, paths []string) (Captured, error) {
	return captureWorktreeSnapshot(ctx, repo, dir, paths, time.Now)
}

func captureWorktreeSnapshot(ctx context.Context, repo, dir string, paths []string, now func() time.Time) (Captured, error) {
	selected, err := normalizeWorktreePaths(paths)
	if err != nil {
		return Captured{}, err
	}
	objectFormat, err := gitObjectFormat(ctx, dir)
	if err != nil {
		return Captured{}, err
	}
	headBytes, err := gitRead(ctx, dir, "rev-parse", "--verify", "HEAD")
	if err != nil {
		headBytes = nil
	}
	snap := &v1.StagedMessageSnapshot{RepositoryId: repo, BaseHead: strings.TrimSpace(string(headBytes))}
	captured := Captured{Snapshot: snap, Bytes: map[string]FileBytes{}}
	total := 0
	for _, path := range selected {
		parts := strings.Split(filepath.FromSlash(path), string(os.PathSeparator))
		parent := dir
		for _, part := range parts[:len(parts)-1] {
			parent = filepath.Join(parent, part)
			info, e := os.Lstat(parent)
			if errors.Is(e, os.ErrNotExist) {
				break
			} // permits a tracked deletion whose parent directory was removed
			if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return Captured{}, fmt.Errorf("selected path %q has an unsafe parent", path)
			}
		}
		idx, e := gitRead(ctx, dir, "ls-files", "--stage", "-z", "--", path)
		if e != nil {
			return Captured{}, e
		}
		var oldID, oldMode string
		for _, row := range bytes.Split(idx, []byte{0}) {
			if len(row) == 0 {
				continue
			}
			meta, got, ok := strings.Cut(string(row), "\t")
			fields := strings.Fields(meta)
			if !ok || got != path || len(fields) != 3 || fields[2] != "0" {
				return Captured{}, fmt.Errorf("invalid or unmerged index entry for %q", path)
			}
			oldMode, oldID = fields[0], fields[1]
		}
		full := filepath.Join(dir, filepath.FromSlash(path))
		readLimit := int64(maxFileBytes)
		if remaining := int64(maxTotalBytes - total); remaining < readLimit {
			readLimit = remaining
		}
		after, info, e := readRegularFileBounded(full, readLimit)
		var newMode, newID, status string
		if e == nil {
			total += len(after)
			newMode = "100644"
			if info.Mode()&0111 != 0 {
				newMode = "100755"
			}
			newID, e = gitBlobID(objectFormat, after)
			if e != nil {
				return Captured{}, e
			}
		} else if !errors.Is(e, os.ErrNotExist) {
			return Captured{}, e
		}
		if oldID == "" {
			status = "A"
		} else if newID == "" {
			status = "D"
		} else {
			status = "M"
		}
		if oldID != "" && oldID == newID && oldMode == newMode {
			return Captured{}, fmt.Errorf("selected path %q has no index-to-worktree change", path)
		}
		if oldID == "" { // New paths must be visible as untracked, not ignored.
			untracked, e := gitRead(ctx, dir, "ls-files", "--others", "--exclude-standard", "--", path)
			if e != nil || strings.TrimSuffix(string(untracked), "\n") != path {
				return Captured{}, fmt.Errorf("selected path %q is not an untracked worktree file", path)
			}
		}
		f := &v1.StagedMessageFile{Path: path, OldPath: path, OldMode: oldMode, NewMode: newMode, OldBlobId: oldID, NewBlobId: newID, Status: status}
		if f.OldBlobId != "" {
			before, e := gitRead(ctx, dir, "cat-file", "blob", f.OldBlobId)
			if e != nil {
				return Captured{}, e
			}
			if len(before) > maxFileBytes || total+len(before) > maxTotalBytes {
				return Captured{}, fmt.Errorf("selected index content exceeds bounded capture limit")
			}
			total += len(before)
			captured.Bytes[path] = FileBytes{Before: before, After: after}
		} else {
			captured.Bytes[path] = FileBytes{After: after}
		}
		if bytes.IndexByte(after, 0) >= 0 || !utf8.Valid(after) {
			f.Binary = true
			f.OmissionReason = "binary content omitted"
		} else if patch, e := renderPatch(ctx, path, captured.Bytes[path].Before, after); e != nil {
			f.OmissionReason = e.Error()
		} else {
			f.Patch = patch
			if len(f.Patch) > maxPatchBytes {
				f.Patch = ""
				f.OmissionReason = "patch exceeds per-file detail bound"
			}
		}
		snap.Files = append(snap.Files, f)
	}
	encoded, _ := json.Marshal(snap)
	sum := sha256.Sum256(encoded)
	snap.Digest = "sha256:" + hex.EncodeToString(sum[:])
	snap.CapturedAt = snapshotTime(now)
	return captured, nil
}

func snapshotTime(now func() time.Time) string {
	if now == nil {
		now = time.Now
	}
	return now().UTC().Format(time.RFC3339Nano)
}

func normalizeWorktreePaths(paths []string) ([]string, error) {
	if len(paths) == 0 || len(paths) > MaxManifestFiles {
		return nil, fmt.Errorf("select between 1 and %d worktree paths", MaxManifestFiles)
	}
	selected := append([]string(nil), paths...)
	sort.Strings(selected)
	if err := gitpaths.ValidateRepositoryRelativePaths(selected); err != nil {
		return nil, fmt.Errorf("invalid selected paths: %w", err)
	}
	return selected, nil
}

func sameWorktreePaths(snapshot *v1.StagedMessageSnapshot, selected []string) bool {
	if snapshot == nil {
		return false
	}
	return sameWorktreePathSet(worktreeSnapshotPaths(snapshot), selected)
}

func worktreeSnapshotPaths(snapshot *v1.StagedMessageSnapshot) []string {
	if snapshot == nil {
		return nil
	}
	paths := make([]string, 0, len(snapshot.Files))
	for _, file := range snapshot.Files {
		paths = append(paths, file.GetPath())
	}
	return paths
}

// sameWorktreePathSet is the shared manifest/selection comparison used by
// admission and later queue-state validation. Copies keep caller ordering
// untouched while making both comparisons independent of input order.
func sameWorktreePathSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	left = append([]string(nil), left...)
	right = append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func hashBytes(ctx context.Context, dir string, data []byte) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "--no-optional-locks", "--literal-pathspecs", "-C", dir, "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null", "hash-object", "--stdin")
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitObjectFormat(ctx context.Context, dir string) (string, error) {
	format, err := gitRead(ctx, dir, "rev-parse", "--show-object-format")
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(format))
	if value != "sha1" && value != "sha256" {
		return "", fmt.Errorf("unsupported Git object format %q", value)
	}
	return value, nil
}

func gitBlobID(format string, data []byte) (string, error) {
	var digest hash.Hash
	switch format {
	case "sha1":
		digest = sha1.New()
	case "sha256":
		digest = sha256.New()
	default:
		return "", fmt.Errorf("unsupported Git object format %q", format)
	}
	_, _ = io.WriteString(digest, "blob "+strconv.Itoa(len(data))+"\x00")
	_, _ = digest.Write(data)
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// WorktreeSelectionApplicable accepts the captured unstaged state or the same
// content staged exactly as captured. It scopes identity to selected paths and
// intentionally ignores HEAD movement.
func WorktreeSelectionApplicable(ctx context.Context, dir string, snapshot *v1.StagedMessageSnapshot) (bool, error) {
	return worktreeSelectionApplicable(ctx, dir, snapshot, true)
}

// WorktreeSelectionUnstagedApplicable requires the saved index-to-worktree
// form, which is the post-unstage condition for staged-selection finalization.
func WorktreeSelectionUnstagedApplicable(ctx context.Context, dir string, snapshot *v1.StagedMessageSnapshot) (bool, error) {
	return worktreeSelectionApplicable(ctx, dir, snapshot, false)
}

// WorktreeSelectionStagedApplicable requires the saved selection's exact
// post-stage contents to be present in Git's stage-0 index entries. The
// worktree may contain later edits; those edits are not part of the commit.
func WorktreeSelectionStagedApplicable(ctx context.Context, dir string, snapshot *v1.StagedMessageSnapshot) (bool, error) {
	if snapshot == nil || len(snapshot.Files) == 0 {
		return false, nil
	}
	wanted := make(map[string]*v1.StagedMessageFile, len(snapshot.Files))
	paths := make([]string, 0, len(snapshot.Files))
	for _, file := range snapshot.Files {
		if file == nil || file.Path == "" {
			return false, nil
		}
		if _, exists := wanted[file.Path]; exists {
			return false, nil
		}
		wanted[file.Path] = file
		paths = append(paths, file.Path)
	}
	index := make(map[string]stagedIndexEntry, len(paths))
	for start := 0; start < len(paths); start += 128 {
		end := min(start+128, len(paths))
		args := []string{"ls-files", "--stage", "-z", "--"}
		for _, path := range paths[start:end] {
			args = append(args, path)
		}
		idx, err := gitRead(ctx, dir, args...)
		if err != nil {
			return false, err
		}
		for _, row := range bytes.Split(idx, []byte{0}) {
			if len(row) == 0 {
				continue
			}
			meta, path, ok := strings.Cut(string(row), "\t")
			fields := strings.Fields(meta)
			if !ok || wanted[path] == nil || len(fields) != 3 || fields[2] != "0" {
				return false, nil
			}
			if _, duplicate := index[path]; duplicate {
				return false, nil
			}
			index[path] = stagedIndexEntry{mode: fields[0], oid: fields[1], stage: fields[2]}
		}
	}
	for path, file := range wanted {
		entry, exists := index[path]
		if file.NewBlobId == "" {
			if exists {
				return false, nil
			}
		} else if !exists || entry.oid != file.NewBlobId || entry.mode != file.NewMode {
			return false, nil
		}
	}
	return true, nil
}

// WorktreeSelectionCommittedAtHead checks a single Git tree read for every
// selected path. Queue retirement therefore records success only after HEAD
// contains the immutable file versions that were reserved for commit.
func WorktreeSelectionCommittedAtHead(ctx context.Context, dir string, snapshot *v1.StagedMessageSnapshot) (bool, error) {
	if snapshot == nil || len(snapshot.Files) == 0 {
		return false, nil
	}
	wanted := make(map[string]*v1.StagedMessageFile, len(snapshot.Files))
	paths := make([]string, 0, len(snapshot.Files))
	for _, file := range snapshot.Files {
		if file == nil || file.Path == "" {
			return false, nil
		}
		wanted[file.Path] = file
		paths = append(paths, file.Path)
	}
	found := make(map[string]struct{}, len(wanted))
	for start := 0; start < len(paths); start += 128 {
		end := min(start+128, len(paths))
		args := []string{"ls-tree", "-rz", "--full-tree", "HEAD", "--"}
		for _, path := range paths[start:end] {
			args = append(args, path)
		}
		output, err := gitRead(ctx, dir, args...)
		if err != nil {
			return false, err
		}
		for _, row := range bytes.Split(output, []byte{0}) {
			if len(row) == 0 {
				continue
			}
			meta, path, ok := strings.Cut(string(row), "\t")
			fields := strings.Fields(meta)
			if !ok || len(fields) != 3 || fields[1] != "blob" {
				return false, nil
			}
			file := wanted[path]
			if file == nil || file.NewBlobId == "" || fields[0] != file.NewMode || fields[2] != file.NewBlobId {
				return false, nil
			}
			found[path] = struct{}{}
		}
	}
	for path, file := range wanted {
		_, exists := found[path]
		if file.NewBlobId == "" {
			if exists {
				return false, nil
			}
		} else if !exists {
			return false, nil
		}
	}
	return true, nil
}

func worktreeSelectionApplicable(ctx context.Context, dir string, snapshot *v1.StagedMessageSnapshot, allowStaged bool) (bool, error) {
	files, paths, wanted, valid := selectedSnapshotFiles(snapshot)
	if !valid {
		return false, nil
	}
	objectFormat, err := gitObjectFormat(ctx, dir)
	if err != nil {
		return false, err
	}
	index, valid, err := readSelectedIndex(ctx, dir, paths, wanted)
	if err != nil || !valid {
		return false, err
	}
	return selectedWorktreeFilesMatch(ctx, dir, files, index, objectFormat, allowStaged)
}

func selectedSnapshotFiles(snapshot *v1.StagedMessageSnapshot) ([]*v1.StagedMessageFile, []string, map[string]*v1.StagedMessageFile, bool) {
	if snapshot == nil || len(snapshot.Files) == 0 {
		return nil, nil, nil, false
	}
	paths := make([]string, 0, len(snapshot.Files))
	wanted := make(map[string]*v1.StagedMessageFile, len(snapshot.Files))
	for _, file := range snapshot.Files {
		if file == nil || file.Path == "" || wanted[file.Path] != nil {
			return nil, nil, nil, false
		}
		wanted[file.Path] = file
		paths = append(paths, file.Path)
	}
	return snapshot.Files, paths, wanted, true
}

func selectedWorktreeFilesMatch(ctx context.Context, dir string, files []*v1.StagedMessageFile, index map[string]stagedIndexEntry, objectFormat string, allowStaged bool) (bool, error) {
	totalWorktreeBytes := int64(0)
	for _, file := range files {
		remaining := int64(maxQueuedCaptureBytes) - totalWorktreeBytes
		if remaining < 0 {
			return false, nil
		}
		workID, workMode, bytesRead, valid, err := readSelectedWorktreeVersion(dir, file.Path, remaining, objectFormat)
		if err != nil || !valid {
			return false, err
		}
		totalWorktreeBytes += bytesRead
		indexEntry, indexExists := index[file.Path]
		if !selectionFileMatches(file, indexEntry, indexExists, workID, workMode, allowStaged) {
			return false, nil
		}
	}
	return true, nil
}

func readSelectedIndex(ctx context.Context, dir string, paths []string, wanted map[string]*v1.StagedMessageFile) (map[string]stagedIndexEntry, bool, error) {
	index := make(map[string]stagedIndexEntry, len(paths))
	for start := 0; start < len(paths); start += 128 {
		end := min(start+128, len(paths))
		args := []string{"ls-files", "--stage", "-z", "--"}
		for _, path := range paths[start:end] {
			args = append(args, path)
		}
		idx, readErr := gitRead(ctx, dir, args...)
		if readErr != nil {
			return nil, false, readErr
		}
		for _, row := range bytes.Split(idx, []byte{0}) {
			if len(row) == 0 {
				continue
			}
			if !addSelectedIndexRow(index, wanted, row) {
				return nil, false, nil
			}
		}
	}
	return index, true, nil
}

func addSelectedIndexRow(index map[string]stagedIndexEntry, wanted map[string]*v1.StagedMessageFile, row []byte) bool {
	meta, path, ok := strings.Cut(string(row), "\t")
	fields := strings.Fields(meta)
	if !ok || wanted[path] == nil || len(fields) != 3 || fields[2] != "0" {
		return false
	}
	if _, duplicate := index[path]; duplicate {
		return false
	}
	index[path] = stagedIndexEntry{mode: fields[0], oid: fields[1], stage: fields[2]}
	return true
}

func readSelectedWorktreeVersion(dir, path string, maxBytes int64, objectFormat string) (blobID, mode string, bytesRead int64, valid bool, err error) {
	if !selectedPathParentsAreSafe(dir, path) {
		return "", "", 0, false, nil
	}
	full := filepath.Join(dir, filepath.FromSlash(path))
	data, info, readErr := readRegularFileBounded(full, maxBytes)
	switch {
	case errors.Is(readErr, errBoundedReadLimit):
		return "", "", 0, false, nil
	case errors.Is(readErr, os.ErrNotExist):
		return "", "", 0, true, nil
	case readErr != nil:
		return "", "", 0, false, readErr
	}
	blobID, err = gitBlobID(objectFormat, data)
	if err != nil {
		return "", "", 0, false, err
	}
	mode = "100644"
	if info.Mode()&0111 != 0 {
		mode = "100755"
	}
	return blobID, mode, int64(len(data)), true, nil
}

func selectedPathParentsAreSafe(dir, path string) bool {
	parent := dir
	parts := strings.Split(filepath.FromSlash(path), string(os.PathSeparator))
	for _, part := range parts[:len(parts)-1] {
		parent = filepath.Join(parent, part)
		info, statErr := os.Lstat(parent)
		if errors.Is(statErr, os.ErrNotExist) {
			break
		}
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return false
		}
	}
	return true
}

func selectionFileMatches(file *v1.StagedMessageFile, indexEntry stagedIndexEntry, indexExists bool, workID, workMode string, allowStaged bool) bool {
	unstaged := !indexExists && file.OldBlobId == "" && file.OldMode == "" && workID == file.NewBlobId && workMode == file.NewMode
	if indexExists {
		unstaged = indexEntry.oid == file.OldBlobId && indexEntry.mode == file.OldMode && workID == file.NewBlobId && workMode == file.NewMode
	}
	staged := !indexExists && file.NewBlobId == "" && file.NewMode == "" && workID == "" && workMode == ""
	if indexExists {
		staged = indexEntry.oid == file.NewBlobId && indexEntry.mode == file.NewMode && workID == file.NewBlobId && workMode == file.NewMode
	}
	return unstaged || allowStaged && staged
}

// GitSnapshotter captures immutable staged content. Now controls only the
// descriptive capture timestamp; snapshot identity remains content-derived.
type GitSnapshotter struct{ Now func() time.Time }

func gitRead(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return gitReadInput(ctx, dir, nil, args...)
}

func gitReadInput(ctx context.Context, dir string, input []byte, args ...string) ([]byte, error) {
	all := append([]string{"--no-optional-locks", "--literal-pathspecs", "-C", dir, "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null"}, args...)
	cmd := exec.CommandContext(ctx, "git", all...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("git %s: %w", args[0], err)
	}
	out, readErr := io.ReadAll(io.LimitReader(stdout, (16<<20)+1))
	if len(out) > 16<<20 {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("git metadata exceeds 16 MiB bound")
	}
	err = cmd.Wait()
	if readErr != nil {
		return nil, readErr
	}
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", args[0], err)
	}
	return out, nil
}

func (GitSnapshotter) manifest(ctx context.Context, repo, dir string) (*v1.StagedMessageSnapshot, error) {
	head, err := gitRead(ctx, dir, "rev-parse", "--verify", "HEAD")
	if err != nil {
		// An unborn repository is valid; other repository errors are not empty input.
		ref, e := gitRead(ctx, dir, "symbolic-ref", "-q", "HEAD")
		if e != nil {
			return nil, fmt.Errorf("HEAD cannot be resolved: %w", err)
		}
		branch := strings.TrimSpace(string(ref))
		if !strings.HasPrefix(branch, "refs/heads/") {
			return nil, fmt.Errorf("unresolved HEAD is not an unborn branch")
		}
		// A symbolic HEAD whose branch ref exists is corruption, not an unborn repo.
		if _, e := gitRead(ctx, dir, "show-ref", "--verify", "--quiet", branch); e == nil {
			return nil, fmt.Errorf("HEAD points to an unreadable existing branch")
		} else {
			var exit *exec.ExitError
			if !errors.As(e, &exit) || exit.ExitCode() != 1 {
				return nil, fmt.Errorf("HEAD branch cannot be inspected: %w", e)
			}
		}
		head = nil
	}
	base := strings.TrimSpace(string(head))
	index, err := gitRead(ctx, dir, "ls-files", "--stage", "-z")
	if err != nil {
		return nil, err
	}
	for _, row := range bytes.Split(index, []byte{0}) {
		if len(row) == 0 {
			continue
		}
		meta, _, ok := strings.Cut(string(row), "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 {
			return nil, fmt.Errorf("invalid index entry")
		}
		if fields[2] != "0" {
			return nil, fmt.Errorf("unmerged index entries must be resolved before drafting")
		}
	}
	args := []string{"diff", "--cached", "--raw", "--no-abbrev", "--no-renames", "--no-ext-diff", "--no-textconv", "-z"}
	if base != "" {
		args = append(args, base)
	}
	args = append(args, "--")
	raw, err := gitRead(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	rows := bytes.Split(raw, []byte{0})
	s := &v1.StagedMessageSnapshot{RepositoryId: repo, BaseHead: base}
	for i := 0; i < len(rows) && len(rows[i]) > 0; i += 2 {
		if i+1 >= len(rows) {
			return nil, fmt.Errorf("incomplete index diff")
		}
		f := strings.Fields(strings.TrimPrefix(string(rows[i]), ":"))
		if len(f) != 5 {
			return nil, fmt.Errorf("invalid index diff")
		}
		path := string(rows[i+1])
		if !utf8.ValidString(path) {
			return nil, fmt.Errorf("non UTF-8 index path is unsupported")
		}
		entry := &v1.StagedMessageFile{Path: path, OldPath: path, OldMode: f[0], NewMode: f[1], OldBlobId: f[2], NewBlobId: f[3], Status: f[4]}
		if zeroOID(entry.OldBlobId) {
			entry.OldBlobId = ""
		}
		if zeroOID(entry.NewBlobId) {
			entry.NewBlobId = ""
		}
		s.Files = append(s.Files, entry)
		if len(s.Files) > MaxManifestFiles {
			return nil, fmt.Errorf("staged selection exceeds %d files; narrow selection", MaxManifestFiles)
		}
	}
	sort.Slice(s.Files, func(i, j int) bool { return s.Files[i].Path < s.Files[j].Path })
	data, _ := json.Marshal(s)
	sum := sha256.Sum256(data)
	s.Digest = "sha256:" + hex.EncodeToString(sum[:])
	return s, nil
}
func zeroOID(s string) bool { return strings.Trim(s, "0") == "" }
func (g GitSnapshotter) Identity(ctx context.Context, repo, dir string) (string, error) {
	s, e := g.manifest(ctx, repo, dir)
	if e != nil {
		return "", e
	}
	return s.Digest, nil
}

func (g GitSnapshotter) Capture(ctx context.Context, repo, dir string) (Captured, error) {
	s, err := g.manifest(ctx, repo, dir)
	if err != nil {
		return Captured{}, err
	}
	c := Captured{Snapshot: s, Bytes: map[string]FileBytes{}}
	total := 0
	pending := map[string]FileBytes{}
	cached := cachedTimes(ctx, dir)
	details := representativeDetails(s.Files)
	for i, f := range s.Files {
		if !details[i] {
			f.OmissionReason = "detail omitted by 64-file scope-balanced sample; full index identity retained"
		}
	}
	for _, i := range representativeDetailOrder(s.Files, details) {
		f := s.Files[i]
		var fb FileBytes
		if f.NewMode == "160000" || f.OldMode == "160000" {
			f.OmissionReason = "submodule content omitted"
			continue
		}
		for _, item := range []struct {
			id  string
			dst *[]byte
		}{{f.OldBlobId, &fb.Before}, {f.NewBlobId, &fb.After}} {
			if item.id == "" {
				continue
			}
			size, e := gitRead(ctx, dir, "cat-file", "-s", item.id)
			if e != nil {
				return Captured{}, e
			}
			n, e := strconv.Atoi(strings.TrimSpace(string(size)))
			if e != nil {
				return Captured{}, e
			}
			if n > maxFileBytes || total+n > maxTotalBytes-maxPatchBytes {
				f.OmissionReason = "content exceeds bounded collection limit"
				break
			}
			*item.dst, e = gitRead(ctx, dir, "cat-file", "blob", item.id)
			if e != nil {
				return Captured{}, e
			}
			total += n
		}
		f.TimingHints = timingHints(dir, f, cached[f.Path], fb.After)
		if f.OmissionReason != "" {
			continue
		}
		if bytes.IndexByte(fb.Before, 0) >= 0 || bytes.IndexByte(fb.After, 0) >= 0 || !utf8.Valid(fb.Before) || !utf8.Valid(fb.After) {
			f.Binary = true
			f.OmissionReason = "binary content omitted"
			continue
		}
		f.Patch, err = renderPatch(ctx, f.Path, fb.Before, fb.After)
		if err != nil {
			f.OmissionReason = err.Error()
			continue
		}
		if len(f.Patch) > 64<<10 {
			f.Patch = ""
			f.OmissionReason = "patch exceeds 64 KiB per-file detail bound"
			continue
		}
		f.EditCount = int32(len(EditAtoms(fb.Before, fb.After)))
		if f.EditCount == 0 && f.OldMode != f.NewMode {
			f.EditCount = 1
		}
		pending[f.Path] = fb
	}
	keep := representativePatches(s.Files)
	for i, f := range s.Files {
		if f.OmissionReason != "" {
			continue
		}
		if !keep[i] {
			f.Patch = ""
			f.OmissionReason = "complete patch omitted by scope-balanced 96 KiB aggregate detail bound"
			continue
		}
		c.Bytes[f.Path] = pending[f.Path]
	}
	after, err := g.Identity(ctx, repo, dir)
	if err != nil {
		return Captured{}, err
	}
	if after != s.Digest {
		return Captured{}, fmt.Errorf("staged selection changed during collection; collect again")
	}
	s.CapturedAt = snapshotTime(g.Now)
	return c, nil
}

// Read bytes round-robin too: a lexical first scope must not exhaust the raw
// content budget before another already-selected scope gets its first detail.
func representativeDetailOrder(files []*v1.StagedMessageFile, selected map[int]bool) []int {
	var ordered []int
	for _, bulk := range []bool{false, true} {
		groups := map[string][]int{}
		for i, f := range files {
			if selected[i] && bulkDetailPath(f.Path) == bulk {
				groups[detailScope(f.Path)] = append(groups[detailScope(f.Path)], i)
			}
		}
		var keys []string
		for key := range groups {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for round := 0; round < maxDetailFiles; round++ {
			for _, key := range keys {
				if round < len(groups[key]) {
					ordered = append(ordered, groups[key][round])
				}
			}
		}
	}
	return ordered
}

// representativePatches spends the actual aggregate budget on complete patches,
// with one small candidate per scope before further candidates from that scope.
// Unused small-file allowance is available to other scopes; no diff is clipped.
func representativePatches(files []*v1.StagedMessageFile) map[int]bool {
	groups := map[string][]int{}
	for i, f := range files {
		if f.OmissionReason == "" {
			groups[detailScope(f.Path)] = append(groups[detailScope(f.Path)], i)
		}
	}
	keys := make([]string, 0, len(groups))
	for key, indices := range groups {
		sort.SliceStable(indices, func(i, j int) bool { return len(files[indices[i]].Patch) < len(files[indices[j]].Patch) })
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := files[groups[keys[i]][0]], files[groups[keys[j]][0]]
		if bulkDetailPath(a.Path) != bulkDetailPath(b.Path) {
			return !bulkDetailPath(a.Path)
		}
		if len(a.Patch) != len(b.Patch) {
			return len(a.Patch) < len(b.Patch)
		}
		return keys[i] < keys[j]
	})
	selected, used := map[int]bool{}, 0
	for round := 0; round < maxDetailFiles; round++ {
		for _, key := range keys {
			if round >= len(groups[key]) {
				continue
			}
			i := groups[key][round]
			if used+len(files[i].Patch) <= maxPatchBytes {
				selected[i] = true
				used += len(files[i].Patch)
			}
		}
	}
	return selected
}

func detailScope(path string) string {
	parts := strings.Split(path, "/")
	scope := parts[0]
	if len(parts) > 1 && (scope == "scenarios" || scope == "packages" || scope == "resources") {
		scope += "/" + parts[1]
	}
	return scope
}

// representativeDetails allocates a round-robin quota across repository scopes,
// then samples each scope evenly. Source paths precede obvious generated/lock
// bulk. This is a bounded explanatory sample, never a claim about importance,
// completeness or authorship; every unselected identity remains in the manifest.
func representativeDetails(files []*v1.StagedMessageFile) map[int]bool {
	selected := map[int]bool{}
	for _, bulk := range []bool{false, true} {
		groups := map[string][]int{}
		for i, f := range files {
			if bulkDetailPath(f.Path) != bulk {
				continue
			}
			scope := detailScope(f.Path)
			groups[scope] = append(groups[scope], i)
		}
		keys := make([]string, 0, len(groups))
		for key := range groups {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		quotas := map[string]int{}
		remaining := maxDetailFiles - len(selected)
		if len(keys) > remaining {
			spread := make([]string, 0, remaining)
			for j := 0; j < remaining; j++ {
				pos := len(keys) / 2
				if remaining > 1 {
					pos = j * (len(keys) - 1) / (remaining - 1)
				}
				spread = append(spread, keys[pos])
			}
			keys = spread
		}
		for remaining > 0 {
			progress := false
			for _, key := range keys {
				if remaining > 0 && quotas[key] < len(groups[key]) {
					quotas[key]++
					remaining--
					progress = true
				}
			}
			if !progress {
				break
			}
		}
		for _, key := range keys {
			indices, quota := groups[key], quotas[key]
			for j := 0; j < quota; j++ {
				pos := len(indices) / 2
				if quota > 1 {
					pos = j * (len(indices) - 1) / (quota - 1)
				}
				selected[indices[pos]] = true
			}
		}
	}
	return selected
}

func bulkDetailPath(path string) bool {
	for _, segment := range strings.Split(strings.ToLower(path), "/") {
		switch segment {
		case "gen", "generated", "node_modules", "vendor", "dist", "build":
			return true
		}
	}
	name := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(name, ".lock") || strings.HasSuffix(name, ".pb.go") || strings.HasSuffix(name, "_pb.ts") || name == "go.sum" || name == "package-lock.json" || name == "pnpm-lock.yaml"
}

// Cached stat is a labeled hint, never part of content identity. --debug is a
// best-effort installed-Git diagnostic: absence or an unrecognized format is unknown.
func cachedTimes(ctx context.Context, dir string) map[string]time.Time {
	out := map[string]time.Time{}
	raw, e := gitRead(ctx, dir, "ls-files", "--debug", "-z")
	if e != nil {
		return out
	}
	rest := string(raw)
	for len(rest) > 0 {
		path, tail, ok := strings.Cut(rest, "\x00")
		if !ok {
			break
		}
		lines := strings.SplitN(tail, "\n", 6)
		if len(lines) < 6 {
			break
		}
		var sec, nsec int64
		if _, e := fmt.Sscanf(strings.TrimSpace(lines[1]), "mtime: %d:%d", &sec, &nsec); e == nil && sec > 0 {
			out[path] = time.Unix(sec, nsec).UTC()
		}
		rest = lines[5]
	}
	return out
}

func timingHints(dir string, f *v1.StagedMessageFile, cached time.Time, after []byte) []*v1.StagedMessageTimingHint {
	var out []*v1.StagedMessageTimingHint
	match := false
	full := filepath.Join(dir, filepath.FromSlash(f.Path))
	root, e := filepath.EvalSymlinks(dir)
	if e != nil {
		return out
	}
	parent, e := filepath.EvalSymlinks(filepath.Dir(full))
	if e != nil {
		return out
	}
	relative, e := filepath.Rel(root, parent)
	if e != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return out
	}
	if st, e := os.Lstat(full); e == nil {
		// Raw byte equality is conservative under line-ending/clean filters;
		// no external clean filter is executed for this optional hint.
		if st.Mode().IsRegular() && st.Size() <= maxFileBytes {
			if data, e := os.ReadFile(full); e == nil {
				match = bytes.Equal(data, after) && f.NewBlobId != ""
			}
		}
		out = append(out, &v1.StagedMessageTimingHint{Source: "filesystem_mtime", Timestamp: st.ModTime().UTC().Format(time.RFC3339Nano), ContentMatchesIndex: match, Limitation: "last filesystem modification; raw byte comparison may differ under Git filters; touches, preserved timestamps and multiple edits are possible"})
	}
	if !cached.IsZero() {
		out = append(out, &v1.StagedMessageTimingHint{Source: "cached_index_stat_mtime", Timestamp: cached.Format(time.RFC3339Nano), ContentMatchesIndex: match, Limitation: "mutable stat cache; may be refreshed without content change; partial staging may omit it"})
	}
	return out
}
