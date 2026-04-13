package packagegov

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AuditReport struct {
	Validation ValidationReport  `json:"validation"`
	Issues     []ValidationIssue `json:"issues"`
}

func Audit(root string, filter string) (AuditReport, error) {
	validation, err := Validate(root, filter)
	if err != nil {
		return AuditReport{}, err
	}

	var issues []ValidationIssue
	for _, path := range []string{
		filepath.Join(root, "packages"),
		filepath.Join(root, "scenarios"),
		filepath.Join(root, "templates"),
		filepath.Join(root, "docs"),
		filepath.Join(root, "pnpm-workspace.yaml"),
	} {
		scanIssues, err := scanDocsDrift(path)
		if err != nil {
			return AuditReport{}, err
		}
		issues = append(issues, scanIssues...)
	}

	return AuditReport{
		Validation: validation,
		Issues:     normalizeIssues(append(validation.Issues, issues...)),
	}, nil
}

func scanDocsDrift(root string) ([]ValidationIssue, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return scanSingleFile(root)
	}

	var issues []ValidationIssue
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "dist", "build", "bundle", "bin", "coverage", "generated", "testdata", "logs", "artifacts":
				return filepath.SkipDir
			}
			if strings.HasPrefix(d.Name(), ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if filepath.Ext(path) == ".png" || filepath.Ext(path) == ".jpg" || filepath.Ext(path) == ".jpeg" || filepath.Ext(path) == ".pb.go" || filepath.Ext(path) == ".pb.ts" {
			return nil
		}
		fileIssues, err := scanSingleFile(path)
		if err != nil {
			return err
		}
		issues = append(issues, fileIssues...)
		return nil
	})
	return issues, err
}

func scanSingleFile(path string) ([]ValidationIssue, error) {
	slashPath := filepath.ToSlash(path)
	if strings.Contains(slashPath, "/docs/plans/") ||
		strings.Contains(slashPath, "/testdata/") ||
		strings.HasSuffix(slashPath, ".lock") ||
		strings.HasSuffix(slashPath, "pnpm-lock.yaml") ||
		strings.HasSuffix(slashPath, "SKILL.md") ||
		strings.HasSuffix(slashPath, "_test.go") {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return nil, nil
	}
	content := string(data)
	var issues []ValidationIssue
	if strings.Contains(content, "refresh-shared-package.sh") {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Code:     "legacy-refresh-helper-reference",
			Message:  "legacy refresh helper is still referenced",
			Path:     path,
		})
	}
	guidanceCandidate := isGovernanceGuidanceCandidate(slashPath)
	if guidanceCandidate && strings.Contains(content, "workspace:*") &&
		!strings.Contains(content, "must not use `workspace:*`") &&
		!strings.Contains(content, "Do NOT use `\"workspace:*\"`") &&
		!strings.Contains(content, "Do NOT use `workspace:*`") {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Code:     "workspace-star-guidance",
			Message:  "workspace:* guidance conflicts with scenario isolation policy",
			Path:     path,
		})
	}
	if guidanceCandidate && strings.Contains(content, "pnpm workspace") &&
		!strings.Contains(content, "do not join the root pnpm workspace") &&
		!strings.Contains(content, "do not join the pnpm workspace") &&
		!strings.Contains(content, "do not use the pnpm workspace") &&
		!strings.Contains(content, "must not use the pnpm workspace") {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Code:     "pnpm-workspace-guidance",
			Message:  "pnpm workspace guidance conflicts with scenario isolation policy",
			Path:     path,
		})
	}
	if guidanceCandidate && strings.Contains(content, "go.work") &&
		!strings.Contains(content, "GOWORK=off") &&
		!strings.Contains(content, "does not depend on a repo-wide `go.work`") &&
		!strings.Contains(content, "must not depend on a repo-wide `go.work`") &&
		!strings.Contains(content, "no dependency on repo-level go.work") {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Code:     "go-work-guidance",
			Message:  "go.work guidance conflicts with governed Go package adoption",
			Path:     path,
		})
	}
	return issues, nil
}

func isGovernanceGuidanceCandidate(path string) bool {
	path = filepath.ToSlash(path)
	if strings.HasPrefix(path, "docs/") || strings.Contains(path, "/docs/") && !strings.Contains(path, "/scenarios/") {
		return true
	}
	if strings.HasPrefix(path, "packages/") || strings.Contains(path, "/packages/") {
		return true
	}
	if strings.HasPrefix(path, "templates/") || strings.Contains(path, "/templates/") {
		return true
	}
	if (strings.HasPrefix(path, "scenarios/") || strings.Contains(path, "/scenarios/")) && strings.HasSuffix(path, "/README.md") {
		return true
	}
	return false
}
