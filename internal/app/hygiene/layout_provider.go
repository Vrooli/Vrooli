package hygiene

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

const layoutProviderID = "repo-layout"

// layoutProvider owns repository-wide layout checks that are independent of a
// scenario's structure-health profile. The roots come from repo-contract so
// the check follows the canonical repository vocabulary instead of embedding
// another copy of the layout in the hygiene command.
type layoutProvider struct {
	root string
}

func (p layoutProvider) ID() string { return layoutProviderID }

func (p layoutProvider) Run(_ context.Context, req Request, report *Report) error {
	contract, err := repocontract.LoadDefault(p.root)
	if err != nil {
		// Structure Health reports malformed or missing contract state. Do not
		// turn the same condition into a second, less useful hygiene failure.
		return nil
	}

	layout := contract.Layout()
	undocumented, err := findUndocumentedInternalPackages(p.root)
	if err != nil {
		return err
	}
	for _, rel := range undocumented {
		report.addFinding(Finding{
			Severity:   SeverityError,
			Code:       "repo_contract_internal_package_doc",
			Path:       rel,
			Message:    fmt.Sprintf("internal package has no package documentation: %s", rel),
			Why:        "every control-plane package must state what it owns and what it does not own before it becomes a new maintenance boundary",
			Fixability: FixabilityManual,
			NextActions: []Action{{
				Code:       "document_internal_package",
				Message:    "Add a package comment beginning with the package name and describe its ownership boundary.",
				Fixability: FixabilityManual,
			}},
		})
	}
	report.addCheck("repo_layout_internal_package_docs", len(undocumented) == 0, SeverityError, fmt.Sprintf("%d internal packages lack package documentation", len(undocumented)))

	// The general contract roots cover internal/ and packages/. Test Genie is
	// also governed explicitly: its empty package directories are scenario
	// residue and must fail hygiene even though scenarios are not part of the
	// repository-wide package layout roots.
	roots := []string{layout.InternalDir, layout.PackageDir, filepath.Join("scenarios", "test-genie")}
	empty, err := findEmptyDirectories(p.root, roots)
	if err != nil {
		return err
	}

	if req.FixSafe {
		removed := 0
		for {
			candidates, scanErr := findEmptyDirectories(p.root, roots)
			if scanErr != nil {
				return scanErr
			}
			if len(candidates) == 0 {
				break
			}
			progress := false
			for _, path := range candidates {
				rel, relErr := filepath.Rel(p.root, path)
				if relErr != nil {
					return relErr
				}
				rel = filepath.ToSlash(rel)
				if removeErr := os.Remove(path); removeErr != nil {
					report.addFinding(emptyDirectoryFinding(rel, removeErr))
					continue
				}
				report.ConfigFixes = append(report.ConfigFixes, "removed empty directory: "+rel)
				removed++
				progress = true
			}
			if !progress {
				break
			}
		}
		remaining, scanErr := findEmptyDirectories(p.root, roots)
		if scanErr != nil {
			return scanErr
		}
		if len(remaining) == 0 {
			report.addCheck("repo_layout_empty_directories", true, SeverityInfo, fmt.Sprintf("removed %d empty directories without .gitkeep", removed))
		} else {
			report.addCheck("repo_layout_empty_directories", false, SeverityError, fmt.Sprintf("%d empty directories remain after safe fix", len(remaining)))
		}
		return nil
	}

	for _, path := range empty {
		rel, relErr := filepath.Rel(p.root, path)
		if relErr != nil {
			return relErr
		}
		report.addFinding(emptyDirectoryFinding(filepath.ToSlash(rel), nil))
	}

	report.addCheck("repo_layout_empty_directories", len(empty) == 0, SeverityError, fmt.Sprintf("found %d empty directories without .gitkeep", len(empty)))
	return nil
}

func findUndocumentedInternalPackages(root string) ([]string, error) {
	internalRoot := filepath.Join(root, "internal")
	documented := make(map[string]bool)
	hasGo := make(map[string]bool)
	fset := token.NewFileSet()
	err := filepath.WalkDir(internalRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != internalRoot && (entry.Name() == "vendor" || entry.Name() == "testdata" || strings.HasPrefix(entry.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return parseErr
		}
		dir := filepath.Dir(path)
		hasGo[dir] = true
		if packageComment(file) {
			documented[dir] = true
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	missing := make([]string, 0)
	for dir := range hasGo {
		if documented[dir] {
			continue
		}
		rel, relErr := filepath.Rel(root, dir)
		if relErr != nil {
			return nil, relErr
		}
		missing = append(missing, filepath.ToSlash(rel))
	}
	slices.Sort(missing)
	return missing, nil
}

func packageComment(file *ast.File) bool {
	if file.Doc == nil {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(file.Doc.Text()), "Package ")
}

func emptyDirectoryFinding(rel string, removeErr error) Finding {
	message := fmt.Sprintf("empty directory without .gitkeep: %s", rel)
	if removeErr != nil {
		message = fmt.Sprintf("empty directory without .gitkeep could not be removed: %s: %v", rel, removeErr)
	}
	return Finding{
		Severity:   SeverityError,
		Code:       "repo_contract_empty_directory",
		Path:       rel,
		Message:    message,
		Why:        "empty directories are local residue; use .gitkeep when an empty directory is deliberate",
		Fixability: FixabilityAutomatic,
		NextActions: []Action{{
			Code:       "remove_empty_directory",
			Message:    "Remove the empty directory with the safe hygiene fixer.",
			Command:    "vrooli hygiene --fix-safe --contract-only",
			Fixability: FixabilityAutomatic,
		}},
	}
}

func (s Service) layoutProvider() Provider {
	return layoutProvider{root: s.Root}
}

func findEmptyDirectories(root string, relativeRoots []string) ([]string, error) {
	var empty []string
	for _, relativeRoot := range relativeRoots {
		relativeRoot = filepath.Clean(strings.TrimSpace(relativeRoot))
		if relativeRoot == "" || relativeRoot == "." || filepath.IsAbs(relativeRoot) || relativeRoot == ".." || strings.HasPrefix(relativeRoot, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("invalid layout root %q", relativeRoot)
		}
		absoluteRoot := filepath.Join(root, relativeRoot)
		if _, err := os.Stat(absoluteRoot); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		err := filepath.WalkDir(absoluteRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() || path == absoluteRoot {
				return nil
			}
			children, readErr := os.ReadDir(path)
			if readErr != nil {
				return readErr
			}
			if len(children) == 0 {
				empty = append(empty, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	// Removing deepest paths first also makes the fixer converge when an
	// entire abandoned scaffold is empty at several nesting levels.
	sort.Slice(empty, func(i, j int) bool {
		depth := func(path string) int { return strings.Count(filepath.Clean(path), string(filepath.Separator)) }
		return depth(empty[i]) > depth(empty[j])
	})
	return empty, nil
}
