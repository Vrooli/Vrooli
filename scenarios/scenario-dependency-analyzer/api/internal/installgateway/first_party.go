package installgateway

import (
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/gomodreconcile"
)

// loadGoTopology reads the repository's Go module layout; tests replace it.
var loadGoTopology = gomodreconcile.LoadTopology

// FirstParty is a request the gateway has proved to name a workspace package
// of this repository, by the repository's own layout and not by the caller's
// say-so.
type FirstParty struct {
	// Path is the package directory relative to the repository root, in slash
	// form (for example "packages/file-viewer").
	Path string
}

// ProveFirstParty reports whether an npm or Go dependency is a first-party
// workspace package: a real, non-linked directory beneath the repository's
// packages directory that carries the requested identity.
//
//   - npm: the version is a repository-relative file: specifier (resolved from
//     baseRoot, the directory that declares it) and the package.json at the
//     target carries the same name and an exact semantic version.
//   - Go: the package path lies in a module whose go.mod lives beneath
//     packages/ in the repository topology.
//
// It never errors: a request it cannot prove is simply not first-party, and the
// caller applies ordinary governance. Scenario modules, mutable tarballs, paths
// outside packages/ and name mismatches are therefore never admitted here.
func ProveFirstParty(repositoryRoot, baseRoot, ecosystem, packageName, version string) (FirstParty, bool) {
	if repositoryRoot == "" || packageName == "" {
		return FirstParty{}, false
	}
	var dir string
	switch CanonicalEcosystem(ecosystem) {
	case "npm":
		verified, err := verifyLocalPnpmPackage(repositoryRoot, baseRoot, packageName, strings.TrimSpace(version))
		if err != nil {
			return FirstParty{}, false
		}
		dir = verified
	case "go":
		owner, ok := goPackageOwnerDir(repositoryRoot, packageName)
		if !ok {
			return FirstParty{}, false
		}
		dir = owner
	default:
		return FirstParty{}, false
	}
	rel, ok := relativeWithin(repositoryRoot, "packages", dir)
	if !ok {
		return FirstParty{}, false
	}
	return FirstParty{Path: filepath.ToSlash(filepath.Join("packages", rel))}, true
}

// goPackageOwnerDir returns the directory of the longest in-repo module that
// contains packageName, and whether it lies in the real packages directory.
func goPackageOwnerDir(repositoryRoot, packageName string) (string, bool) {
	topo, err := loadGoTopology(repositoryRoot)
	if err != nil {
		return "", false
	}
	owner := ""
	for modulePath := range topo {
		if (packageName == modulePath || strings.HasPrefix(packageName, modulePath+"/")) && len(modulePath) > len(owner) {
			owner = modulePath
		}
	}
	if owner == "" || verifyWithinRoot(repositoryRoot, topo[owner]) != nil {
		return "", false
	}
	return topo[owner], true
}

// relativeWithin returns target relative to repositoryRoot/subdir when target
// is strictly inside it.
func relativeWithin(repositoryRoot, subdir, target string) (string, bool) {
	base := filepath.Join(repositoryRoot, subdir)
	if resolved, err := filepath.EvalSymlinks(base); err == nil {
		base = resolved
	}
	if resolved, err := filepath.EvalSymlinks(target); err == nil {
		target = resolved
	}
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}
