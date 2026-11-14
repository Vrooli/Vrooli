package buildinfo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentStateBuildFlag stamps the persistence source inventory into a
// governed native build: per stored named type in statecontract.CodecPackages
// its enum constants and custom codec closure; each whole-source owner's
// source digest; each constants-stamped owner's constants; each precise
// owner's stored untyped constants and time layouts; the format-version
// constants in use; and every failed source check. The executable combines
// it with its compiled shapes and facts (statecontract.Compute); a failed
// check leaves it unqualified for automatic activation, naming the check.
func PersistentStateBuildFlag(root string) (string, error) {
	stamp, err := PersistentStateCodecStamp(root)
	if err != nil {
		return "", err
	}
	return "-X github.com/vrooli/vrooli/internal/statecontract.CodecStamp=" + stamp, nil
}

// PersistentStateCodecStamp scans the codec inventory under an explicit
// canonical source root.
func PersistentStateCodecStamp(root string) (string, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return "", fmt.Errorf("persistent-state build requires an explicit canonical source root")
	}
	packages := make([]string, 0, len(statecontract.CodecPackages))
	entries := map[string]string{}
	var violations []string
	for _, dir := range statecontract.CodecPackages {
		importPath, err := sourceImportPath(root, dir)
		if err != nil {
			return "", fmt.Errorf("persistent-state codec package %s: %w", dir, err)
		}
		digests, codecViolations, err := scanCodecPackage(root, dir)
		if err != nil {
			return "", fmt.Errorf("persistent-state codec package %s: %w", dir, err)
		}
		violations = append(violations, codecViolations...)
		packages = append(packages, importPath)
		for typeName, digest := range digests {
			entries[importPath+"."+typeName] = digest
		}
	}
	sources := make(map[string]string, len(statecontract.WholeSources))
	for owner, patterns := range statecontract.WholeSources {
		digest, err := wholeSourceDigest(root, patterns)
		if err != nil {
			return "", fmt.Errorf("persistent-state whole-source owner %s: %w", owner, err)
		}
		sources[owner] = digest
	}
	consts := make(map[string]string, len(statecontract.ConstantSources))
	values := map[string]string{}
	layouts := map[string]string{}
	var versions, literals []string
	analyse := func(owner string, dirs []string, vocabulary bool) error {
		analysis, err := analyzeOwnerPackages(root, dirs, statecontract.NotStoredConstants, vocabulary)
		if err != nil {
			return fmt.Errorf("persistent-state owner %s: %w", owner, err)
		}
		for name, digest := range analysis.values {
			values[owner+"/"+name] = digest
		}
		for _, version := range analysis.versions {
			versions = append(versions, owner+"/"+version)
		}
		layouts[owner] = analysis.layoutDigest()
		violations = append(violations, analysis.violations...)
		literals = append(literals, analysis.literals...)
		return nil
	}
	for owner, dirs := range statecontract.ConstantSources {
		digest, err := packageConstantsDigest(root, dirs, statecontract.NotStoredConstants)
		if err != nil {
			return "", fmt.Errorf("persistent-state constants owner %s: %w", owner, err)
		}
		consts[owner] = digest
		if err := analyse(owner, dirs, false); err != nil {
			return "", err
		}
	}
	for owner, dirs := range statecontract.PreciseSources {
		if err := analyse(owner, dirs, true); err != nil {
			return "", err
		}
	}
	return statecontract.EncodeStamp(statecontract.Stamp{Packages: packages, Entries: entries, Sources: sources, Consts: consts, Values: values, Layouts: layouts, Versions: versions, Literals: compactStrings(sortedCopy(literals)), Violations: compactStrings(sortedCopy(violations))})
}

// sourceImportPath resolves a repository directory through its nearest
// go.mod, so nested modules (packages/platform-go) and root-module packages
// (packages/resource-deployment) both resolve without a second inventory.
func sourceImportPath(root, dir string) (string, error) {
	current := filepath.Join(root, filepath.FromSlash(dir))
	for {
		module, err := goModulePath(filepath.Join(current, "go.mod"))
		if err == nil {
			rel, err := filepath.Rel(current, filepath.Join(root, filepath.FromSlash(dir)))
			if err != nil {
				return "", err
			}
			if rel == "." {
				return module, nil
			}
			return module + "/" + filepath.ToSlash(rel), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		if current == root {
			return "", fmt.Errorf("no enclosing go.mod")
		}
		current = filepath.Dir(current)
	}
}

func goModulePath(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if module, ok := strings.CutPrefix(strings.TrimSpace(scanner.Text()), "module "); ok {
			return strings.Trim(strings.TrimSpace(module), `"`), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("%s declares no module", path)
}

func sortedCopy(items []string) []string {
	out := append([]string(nil), items...)
	sort.Strings(out)
	return out
}
