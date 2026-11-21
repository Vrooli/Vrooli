package repocontract

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

// SourceRoots returns the slash-relative layout directories that hold
// repository source code: scenarios, resources, templates, packages, commands
// and internal code. The repository docs tree is not among them.
func (l Layout) SourceRoots() []string {
	return []string{l.ScenarioDir, l.ResourceDir, l.TemplateDir, l.PackageDir, l.CommandDir, l.InternalDir}
}

// sourceWalkSkippedDirs are directory names that never hold first-party
// source: version control metadata and installed third-party dependencies.
var sourceWalkSkippedDirs = map[string]bool{".git": true, "node_modules": true, "vendor": true}

// WalkSource walks every source root with fs.WalkDir semantics, so fn may
// return fs.SkipDir to prune further. Paths passed to fn are absolute.
//
// Documentation trees are not source even when code files appear in them:
// the repository docs tree and each target owner's docs directory (for
// example scenarios/<name>/docs) retain evidence — captured bundles,
// preimages, truncated snapshots — that records what code looked like.
// Repository-wide source audits must not judge those records, so WalkSource
// never descends into them. A deeper package that happens to be named docs
// (scenarios/<name>/api/handlers/docs) is ordinary source and is walked.
func (c *Contract) WalkSource(repoRoot string, fn fs.WalkDirFunc) error {
	if c == nil {
		return &Error{Kind: ErrInvalidInput, Message: "contract is required"}
	}
	root, err := filepath.Abs(strings.TrimSpace(repoRoot))
	if err != nil || strings.TrimSpace(repoRoot) == "" {
		return &Error{Kind: ErrInvalidInput, Message: "repo root is required", Err: err}
	}
	docsDir := path.Clean(c.doc.Layout.DocsDir)
	index, err := c.NewTargetIndex(root)
	if err != nil {
		return err
	}
	for _, sourceRoot := range c.doc.Layout.SourceRoots() {
		if strings.TrimSpace(sourceRoot) == "" {
			continue
		}
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(sourceRoot)), func(current string, entry fs.DirEntry, walkErr error) error {
			if walkErr == nil && entry.IsDir() {
				if sourceWalkSkippedDirs[entry.Name()] {
					return fs.SkipDir
				}
				if entry.Name() == path.Base(docsDir) && isOwnerDocsDir(index, root, current) {
					return fs.SkipDir
				}
			}
			return fn(current, entry, walkErr)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// isOwnerDocsDir reports whether dir is the documentation directory placed
// directly in a target owner's root.
func isOwnerDocsDir(index *TargetIndex, repoRoot, dir string) bool {
	rel, err := filepath.Rel(repoRoot, dir)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	target, ok := index.Lookup(rel)
	return ok && normalizedTargetRoot(target.Root) == path.Dir(rel)
}
