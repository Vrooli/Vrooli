package gomodreconcile

import (
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// walkModuleGoFiles visits every Go file that `go mod tidy` considers part of
// the module rooted at moduleDir, with its imports parsed. It mirrors the go
// command's package rules: directories and files whose names begin with "." or
// "_" are ignored, as are testdata and vendor directories and nested modules
// (a directory holding its own go.mod). Tidy loads every build tag except
// "ignore", so only files excluded by an `ignore` constraint are skipped.
// node_modules is skipped as well; it never holds module source here.
func walkModuleGoFiles(moduleDir string, visit func(path string, file *ast.File)) error {
	return walkGoSourceTree(moduleDir, true, func(dir string, files []string) {
		for _, path := range files {
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly|parser.ParseComments)
			if err != nil || excludedByIgnoreTag(file) {
				continue
			}
			visit(path, file)
		}
	})
}

// walkGoSourceTree calls visit for each directory that holds Go files the go
// command would consider, passing the files. With stopAtModules it does not
// descend into nested modules.
func walkGoSourceTree(root string, stopAtModules bool, visit func(dir string, files []string)) error {
	var walk func(dir string) error
	walk = func(dir string) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if dir == root {
				return err
			}
			return nil
		}
		var files []string
		var subdirs []string
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				continue
			}
			if entry.IsDir() {
				switch name {
				case "testdata", "vendor", "node_modules":
					continue
				}
				subdirs = append(subdirs, filepath.Join(dir, name))
				continue
			}
			if strings.HasSuffix(name, ".go") {
				files = append(files, filepath.Join(dir, name))
			}
		}
		if len(files) > 0 {
			visit(dir, files)
		}
		for _, sub := range subdirs {
			if stopAtModules && isModuleRoot(sub) {
				continue
			}
			if err := walk(sub); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root)
}

func isModuleRoot(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil && !info.IsDir()
}

// excludedByIgnoreTag reports whether a file's build constraint can only be
// satisfied by the "ignore" tag being false, the one tag `go mod tidy` treats
// as unset. Every other tag counts as satisfiable either way, as in the go
// command's imports.AnyTags evaluation.
func excludedByIgnoreTag(file *ast.File) bool {
	for _, group := range file.Comments {
		if group.Pos() >= file.Package {
			break
		}
		for _, comment := range group.List {
			if !constraint.IsGoBuild(comment.Text) {
				continue
			}
			expr, err := constraint.Parse(comment.Text)
			if err != nil {
				return false
			}
			return !anyTagsEval(expr, true)
		}
	}
	return false
}

// anyTagsEval evaluates expr with every tag except "ignore" taking whichever
// value satisfies the expression (prefer), mirroring cmd/go/internal/imports.
func anyTagsEval(expr constraint.Expr, prefer bool) bool {
	switch e := expr.(type) {
	case *constraint.TagExpr:
		if e.Tag == "ignore" {
			return false
		}
		return prefer
	case *constraint.NotExpr:
		return !anyTagsEval(e.X, !prefer)
	case *constraint.AndExpr:
		return anyTagsEval(e.X, prefer) && anyTagsEval(e.Y, prefer)
	case *constraint.OrExpr:
		return anyTagsEval(e.X, prefer) || anyTagsEval(e.Y, prefer)
	}
	return true
}

// UnfencedEvidence returns the directories under root (relative, slash
// separated) that hold Go packages inside a documentation `docs/` tree (a docs
// directory holding Markdown, as opposed to a production Go package that is
// merely named docs, such as a CLI `domains/docs`) without their own module
// boundary. Go compiles such a package as part of the nearest enclosing
// module, so retained source copies silently join a production module and pull
// its imports into that module's requirements (they once made the repository
// root require nutrition-planner, workflow-health, and browser-automation-
// studio). The docs-evidence convention fences each tree with a go.mod at the
// top of its docs directory. With stopAtModules the scan covers only the module
// rooted at root; otherwise it covers the whole tree (for example a scenario
// directory, whose docs belong to an enclosing module).
func UnfencedEvidence(root string, stopAtModules bool) ([]string, error) {
	var out []string
	err := walkGoSourceTree(root, stopAtModules, func(dir string, _ []string) {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return
		}
		segments := strings.Split(filepath.ToSlash(rel), "/")
		docsAt := -1
		for i, segment := range segments {
			if segment == "docs" && isDocumentationTree(filepath.Join(root, filepath.FromSlash(strings.Join(segments[:i+1], "/")))) {
				docsAt = i
				break
			}
		}
		if docsAt < 0 {
			return
		}
		// Fenced when a module root sits at or below the docs directory.
		for i := len(segments); i > docsAt; i-- {
			if isModuleRoot(filepath.Join(root, filepath.FromSlash(strings.Join(segments[:i], "/")))) {
				return
			}
		}
		out = append(out, filepath.ToSlash(rel))
	})
	sort.Strings(out)
	return out, err
}

// isDocumentationTree reports whether dir directly holds a Markdown file.
func isDocumentationTree(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return true
		}
	}
	return false
}
