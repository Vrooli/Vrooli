package main

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// seamTreeFile is one regular file a seam scan inspects.
type seamTreeFile struct {
	path string // treeRoot-joined path for reading
	rel  string // slash-separated path relative to the scanned tree
}

// gitSourceListing returns Git's NUL-separated view of the source tree:
// tracked plus untracked-and-not-ignored files. It fails outside a work tree.
type gitSourceListing func(treeRoot string) ([]byte, error)

func gitLsFiles(treeRoot string) ([]byte, error) {
	return exec.Command("git", "-C", treeRoot, "ls-files", "--cached", "--others", "--exclude-standard", "-z").Output() // #nosec G204 -- fixed executable and arguments; treeRoot is the scanned tree.
}

// seamTreeFiles lists the files a seam scan judges: the files the repository
// treats as source. Build output, virtualenvs, packaged desktop bundles and
// staging trees are residue of a local build, and counting them made every
// observation depend on what happened to be built. In a work tree Git's ignore
// rules decide; elsewhere a walk skips the known build-output directories.
//
// cli-core's EnumerateInputs is not reused: its walk also skips .vrooli, data
// and tmp, which hold files seams must see.
func seamTreeFiles(treeRoot string) ([]seamTreeFile, error) {
	return seamTreeFilesWith(treeRoot, gitLsFiles)
}

func seamTreeFilesWith(treeRoot string, listGit gitSourceListing) ([]seamTreeFile, error) {
	var rels []string
	if out, err := listGit(treeRoot); err == nil {
		for _, rel := range bytes.Split(out, []byte{0}) {
			if len(rel) > 0 {
				rels = append(rels, filepath.ToSlash(string(rel)))
			}
		}
	} else {
		walked, walkErr := walkSeamTree(treeRoot)
		if walkErr != nil {
			return nil, walkErr
		}
		rels = walked
	}
	sort.Strings(rels)
	files := make([]seamTreeFile, 0, len(rels))
	for _, rel := range rels {
		if seamDependencyTree(rel) {
			continue
		}
		path := filepath.Join(treeRoot, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			// A tracked file deleted from the work tree, or a symlink, is not
			// a file in the tree being judged.
			continue
		}
		files = append(files, seamTreeFile{path: path, rel: rel})
	}
	return files, nil
}

func walkSeamTree(treeRoot string) ([]string, error) {
	var rels []string
	err := filepath.WalkDir(treeRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path != treeRoot && vanishedDuringScan(walkErr) {
				return nil
			}
			return walkErr
		}
		if path == treeRoot {
			return nil
		}
		rel, err := filepath.Rel(treeRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if seamDependencyTree(entry.Name()) || seamBuildOutputDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		rels = append(rels, rel)
		return nil
	})
	return rels, err
}

// seamBuildOutputDir reports a directory the repository ignores as build
// output: the walk fallback's stand-in for .gitignore, so a checkout without
// Git reaches the same verdict.
func seamBuildOutputDir(rel string) bool {
	switch path.Base(rel) {
	case ".venv", "dist-electron", "bin":
		return true
	case "bundle":
		// The packaged payload of a generated desktop shell; a Go package
		// named bundle elsewhere is source.
		return path.Base(path.Dir(rel)) == "electron"
	default:
		return false
	}
}

// seamDependencyTree reports a path inside an installed or vendored
// dependency tree, which no seam governs.
func seamDependencyTree(rel string) bool {
	for _, segment := range strings.Split(rel, "/") {
		switch segment {
		case ".git", "node_modules", "vendor":
			return true
		}
	}
	return false
}

// forEachSeamFile visits every inventoried file.
func forEachSeamFile(files []seamTreeFile, visit func(path, rel string) error) error {
	for _, file := range files {
		if err := visit(file.path, file.rel); err != nil {
			return err
		}
	}
	return nil
}

// forEachSeamGoFile visits every inventoried Go source file.
func forEachSeamGoFile(files []seamTreeFile, visit func(path, rel string) error) error {
	for _, file := range files {
		if !strings.HasSuffix(file.rel, ".go") {
			continue
		}
		if err := visit(file.path, file.rel); err != nil {
			return err
		}
	}
	return nil
}
