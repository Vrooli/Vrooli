package stagedmessages

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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
type GitSnapshotter struct{}

func gitRead(ctx context.Context, dir string, args ...string) ([]byte, error) {
	all := append([]string{"--no-optional-locks", "--literal-pathspecs", "-C", dir, "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null"}, args...)
	cmd := exec.CommandContext(ctx, "git", all...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
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
	s.CapturedAt = time.Now().UTC().Format(time.RFC3339Nano)
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
