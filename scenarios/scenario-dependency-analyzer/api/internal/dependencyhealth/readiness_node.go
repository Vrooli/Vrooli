package dependencyhealth

import (
	"path/filepath"
	"strings"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

func checkNodeSurface(surface *healthv1.DependencyHealthSurface) []*healthv1.DependencyHealthFinding {
	root := surface.GetRootPath()
	if !fileExists(filepath.Join(root, "package.json")) {
		return []*healthv1.DependencyHealthFinding{readinessFinding("node."+surfaceID(surface)+".missing-package-json", "WARNING", "JavaScript surface has no package.json", "A discovered JavaScript/TypeScript surface does not have a package.json at its root.", "Expose the correct package root through Code Facts or add package.json for this surface.", surface, "dependency.node.package_json_present", "missing package.json", "package.json at the JS/TS surface root")}
	}
	lockfiles := detectLockfiles(root)
	var findings []*healthv1.DependencyHealthFinding
	if len(lockfiles) == 0 {
		findings = append(findings, readinessFinding("node."+surfaceID(surface)+".missing-lockfile", "ERROR", "JavaScript lockfile missing", "A JavaScript/TypeScript dependency surface has package.json but no supported lockfile.", "Commit the correct lockfile for this package manager, usually pnpm-lock.yaml for Vrooli scenario surfaces.", surface, "dependency.node.lockfile_present", "no supported lockfile", "one lockfile: pnpm-lock.yaml, package-lock.json, or yarn.lock"))
	}
	if len(lockfiles) > 1 {
		findings = append(findings, readinessFinding("node."+surfaceID(surface)+".multiple-lockfiles", "ERROR", "Conflicting JavaScript lockfiles", "A JavaScript/TypeScript dependency surface has multiple package-manager lockfiles.", "Keep exactly one lockfile for the intended package manager and remove stale lockfiles.", surface, "dependency.node.single_lockfile", strings.Join(lockfiles, ", "), "exactly one package-manager lockfile"))
	}
	if !dirExists(filepath.Join(root, "node_modules")) {
		findings = append(findings, readinessFinding("node."+surfaceID(surface)+".node-modules-missing", "WARNING", "JavaScript install state is missing locally", "node_modules is absent for this JavaScript/TypeScript surface. This is local readiness, not dependency declaration drift.", "Install dependencies in the reported workspace without changing dependency declarations, for example `pnpm install --ignore-workspace` when pnpm is the intended manager.", surface, "dependency.node.install_state", "node_modules missing", "local install state present when local execution needs it"))
	}
	return findings
}

func detectLockfiles(root string) []string {
	var out []string
	for _, name := range []string{"pnpm-lock.yaml", "package-lock.json", "yarn.lock"} {
		if fileExists(filepath.Join(root, name)) {
			out = append(out, name)
		}
	}
	return out
}

func readinessFinding(id, severity, title, description, remediation string, surface *healthv1.DependencyHealthSurface, ruleID, observed, expected string) *healthv1.DependencyHealthFinding {
	return &healthv1.DependencyHealthFinding{
		Id:           "readiness." + slug(id),
		Severity:     severity,
		SourceDomain: "readiness",
		Title:        title,
		Description:  description,
		Remediation:  remediation,
		FilePath:     relScenarioPath(surface.GetRootPath()),
		SurfaceId:    surface.GetId(),
		RuleId:       ruleID,
		Observed:     observed,
		Expected:     expected,
	}
}
