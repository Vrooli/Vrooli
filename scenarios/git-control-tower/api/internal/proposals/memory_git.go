package proposals

import (
	"context"
	"crypto/sha1" //nolint:gosec // models git's SHA-1 blob IDs; not a security use
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// MemoryGit is an in-memory model of HEAD, the index and the working tree.
// It implements Git with git's semantics for the operations proposals use,
// so tests exercise staging and commits without running git (GCT test
// fixtures never mutate a real repository). Production never constructs it.
type MemoryGit struct {
	commits    map[string]memoryCommit
	head       string
	index      map[string]string // path -> blob
	worktree   map[string]string // path -> content
	ignored    map[string]bool
	stageCalls [][]string
	afterStage func()
	stageErr   error
	nextOID    int
}

type memoryCommit struct {
	parent  string
	tree    map[string]string
	message string
}

func memoryBlobID(content string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("blob %d\x00%s", len(content), content))) //nolint:gosec // git blob ID model
	return hex.EncodeToString(sum[:])
}

func NewMemoryGit(files map[string]string) *MemoryGit {
	r := &MemoryGit{commits: map[string]memoryCommit{}, index: map[string]string{}, worktree: map[string]string{}, ignored: map[string]bool{}}
	tree := map[string]string{}
	for path, content := range files {
		tree[path] = memoryBlobID(content)
		r.index[path] = memoryBlobID(content)
		r.worktree[path] = content
	}
	r.head = r.addCommit("", tree, "initial")
	return r
}

func (r *MemoryGit) addCommit(parent string, tree map[string]string, message string) string {
	r.nextOID++
	oid := fmt.Sprintf("%040x", r.nextOID)
	r.commits[oid] = memoryCommit{parent: parent, tree: tree, message: message}
	return oid
}

func (r *MemoryGit) headTree() map[string]string { return r.commits[r.head].tree }

// Write changes a working-tree file; Delete removes it.
func (r *MemoryGit) Write(path, content string) { r.worktree[path] = content }
func (r *MemoryGit) Delete(path string)         { delete(r.worktree, path) }

// StageDirect models a human `git add` outside the proposal flow.
func (r *MemoryGit) StageDirect(path string) {
	if content, ok := r.worktree[path]; ok {
		r.index[path] = memoryBlobID(content)
	} else {
		delete(r.index, path)
	}
}

// CommitOutside models a commit made elsewhere (another session or a hand
// commit) from the given files, moving HEAD.
func (r *MemoryGit) CommitOutside(files map[string]string) {
	tree := map[string]string{}
	for path, blob := range r.headTree() {
		tree[path] = blob
	}
	for path, content := range files {
		tree[path] = memoryBlobID(content)
		r.index[path] = memoryBlobID(content)
		r.worktree[path] = content
	}
	r.head = r.addCommit(r.head, tree, "outside")
}

// CommitIndex is a CommitFunc that commits the index.
func (r *MemoryGit) CommitIndex(_ context.Context, message string) (CommitOutcome, error) {
	tree := map[string]string{}
	for path, blob := range r.index {
		tree[path] = blob
	}
	r.head = r.addCommit(r.head, tree, message)
	return CommitOutcome{OID: r.head, Precommit: &PrecommitOutcome{Status: "passed", Summary: "Precommit checks passed"}}, nil
}

func (r *MemoryGit) Head(context.Context) (string, error)   { return r.head, nil }
func (r *MemoryGit) Branch(context.Context) (string, error) { return "main", nil }

func (r *MemoryGit) Status(context.Context) ([]StatusEntry, error) {
	paths := map[string]struct{}{}
	for _, set := range []map[string]string{r.headTree(), r.index, r.worktree} {
		for path := range set {
			paths[path] = struct{}{}
		}
	}
	var entries []StatusEntry
	for path := range paths {
		headBlob, inHead := r.headTree()[path]
		indexBlob, inIndex := r.index[path]
		content, inTree := r.worktree[path]
		if !inHead && !inIndex {
			if inTree && !r.ignored[path] {
				entries = append(entries, StatusEntry{Path: path, XY: "??"})
			}
			continue
		}
		x, y := byte('.'), byte('.')
		switch {
		case inIndex && !inHead:
			x = 'A'
		case !inIndex && inHead:
			x = 'D'
		case indexBlob != headBlob:
			x = 'M'
		}
		switch {
		case inIndex && !inTree:
			y = 'D'
		case inIndex && memoryBlobID(content) != indexBlob:
			y = 'M'
		}
		if x != '.' || y != '.' {
			entries = append(entries, StatusEntry{Path: path, XY: string([]byte{x, y})})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func (r *MemoryGit) Hash(_ context.Context, paths []string) (map[string]Content, error) {
	result := map[string]Content{}
	for _, path := range paths {
		content, ok := r.worktree[path]
		if !ok {
			result[path] = Content{Deleted: true}
			continue
		}
		sum := sha256.Sum256([]byte(content))
		result[path] = Content{BlobID: memoryBlobID(content), SHA256: hex.EncodeToString(sum[:])}
	}
	return result, nil
}

func (r *MemoryGit) IndexEntries(_ context.Context, paths []string) (map[string]IndexEntry, error) {
	result := map[string]IndexEntry{}
	for _, path := range paths {
		if blob, ok := r.index[path]; ok {
			result[path] = IndexEntry{Mode: "100644", BlobID: blob}
		}
	}
	return result, nil
}

func (r *MemoryGit) StagedPaths(context.Context) ([]string, error) {
	var staged []string
	for path, blob := range r.index {
		if r.headTree()[path] != blob {
			staged = append(staged, path)
		}
	}
	for path := range r.headTree() {
		if _, ok := r.index[path]; !ok {
			staged = append(staged, path)
		}
	}
	sort.Strings(staged)
	return staged, nil
}

func (r *MemoryGit) ChangedPaths(_ context.Context, from, to string, paths []string) ([]string, error) {
	var changed []string
	for _, path := range paths {
		if r.commits[from].tree[path] != r.commits[to].tree[path] {
			changed = append(changed, path)
		}
	}
	return changed, nil
}

func (r *MemoryGit) Ignored(_ context.Context, paths []string) ([]string, error) {
	var ignored []string
	for _, path := range paths {
		if r.ignored[path] {
			ignored = append(ignored, path)
		}
	}
	return ignored, nil
}

func (r *MemoryGit) ObjectExists(_ context.Context, oid string) (bool, error) {
	for id := range r.commits {
		if strings.HasPrefix(id, oid) {
			return true, nil
		}
	}
	return false, nil
}

func (r *MemoryGit) Stage(_ context.Context, paths []string) error {
	r.stageCalls = append(r.stageCalls, append([]string(nil), paths...))
	if r.stageErr != nil {
		return r.stageErr
	}
	for _, path := range paths {
		r.StageDirect(path)
	}
	if r.afterStage != nil {
		r.afterStage()
	}
	return nil
}

func (r *MemoryGit) RestoreIndex(_ context.Context, paths []string, prior map[string]IndexEntry) error {
	for _, path := range paths {
		if entry, ok := prior[path]; ok {
			r.index[path] = entry.BlobID
		} else {
			delete(r.index, path)
		}
	}
	return nil
}

func (r *MemoryGit) CommitFacts(_ context.Context, oid string) (CommitFacts, error) {
	commit, ok := r.commits[oid]
	if !ok {
		return CommitFacts{}, fmt.Errorf("unknown commit %s", oid)
	}
	parent := r.commits[commit.parent].tree
	blobs := map[string]string{}
	for path, blob := range commit.tree {
		if parent[path] != blob {
			blobs[path] = blob
		}
	}
	for path := range parent {
		if _, ok := commit.tree[path]; !ok {
			blobs[path] = DeletedBlob
		}
	}
	return CommitFacts{Blobs: blobs, Message: commit.message}, nil
}

// HeadOID returns the current HEAD commit.
func (r *MemoryGit) HeadOID() string { return r.head }

// CommitCount returns the number of commits, the initial one included.
func (r *MemoryGit) CommitCount() int { return len(r.commits) }

// StageCalls returns the path lists passed to Stage, in call order.
func (r *MemoryGit) StageCalls() [][]string { return append([][]string(nil), r.stageCalls...) }
