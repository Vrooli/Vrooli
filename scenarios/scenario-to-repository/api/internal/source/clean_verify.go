package source

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CleanReport is the provider-neutral result of the pre-build independence
// gate. It intentionally does not execute package managers or Git hooks. A
// caller may run the exported package in a separate worker after this gate.
type CleanReport struct {
	Status               string
	ArtifactRoot         string
	DeclaredInputs       []string
	ExternalRequirements []string
	GatesPassed          []string
	Failures             []string
}

// VerifyCleanInputs proves that an admitted closure can be handed to a clean
// worker without relying on the monorepo, Git metadata, or private planning
// state. Runtime services remain explicit requirements rather than green
// skips. It performs no repository mutation and no network access.
func VerifyCleanInputs(root string, closure Closure) (CleanReport, error) {
	report := CleanReport{Status: "passed", ArtifactRoot: root}
	if root == "" {
		return CleanReport{Status: "failed", Failures: []string{"artifact root is required"}}, nil
	}
	if !filepath.IsAbs(root) {
		report.Status = "failed"
		report.Failures = append(report.Failures, "clean verification requires an explicit artifact root")
	}
	for _, file := range closure.Files {
		path := filepath.ToSlash(filepath.Clean(file.ExportPath))
		if err := validateArchivePath(path); err != nil {
			report.Failures = append(report.Failures, err.Error())
			continue
		}
		if strings.HasPrefix(path, ".git/") || path == ".git" || strings.HasPrefix(path, ".vrooli/plan-artifacts/") {
			report.Failures = append(report.Failures, fmt.Sprintf("private environment input admitted: %s", path))
			continue
		}
		report.DeclaredInputs = append(report.DeclaredInputs, path)
	}
	for _, node := range closure.Nodes {
		if node.Kind == "runtime_requirement" {
			report.ExternalRequirements = append(report.ExternalRequirements, node.ID)
		}
	}
	if len(report.Failures) > 0 {
		report.Status = "failed"
		return report, nil
	}
	report.GatesPassed = []string{"isolated-root", "declared-inputs", "no-git-mutation", "runtime-requirements-explicit"}
	return report, nil
}
