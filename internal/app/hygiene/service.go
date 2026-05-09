package hygiene

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	planapp "github.com/vrooli/vrooli/internal/app/plans"
)

type Service struct {
	Root string
	Home string
}

func (s Service) Run(req Request) (Report, error) {
	root := filepath.Clean(strings.TrimSpace(s.Root))
	if root == "" || root == "." {
		return Report{}, fmt.Errorf("repo root is required")
	}
	if req.FailOn == "" {
		req.FailOn = SeverityError
	}
	report := Report{Root: root, Success: true}

	contract, err := contractapp.Validate(root)
	if err != nil {
		report.addFinding(SeverityError, "contract_validation_error", "", err.Error())
		report.addCheck("repo_contract", false, SeverityError, err.Error())
	} else {
		report.Contract = contract
		severity := SeverityInfo
		message := "passed"
		if !contract.Success {
			severity = SeverityError
			message = "repo contract validation failed"
		}
		report.addCheck("repo_contract", contract.Success, severity, message)
		for _, check := range contract.Report.Checks {
			if check.Passed {
				continue
			}
			report.addFinding(SeverityError, "repo_contract_"+check.Name, "", check.Message)
		}
		if !contract.Schema.Passed {
			report.addFinding(SeverityError, "repo_contract_schema", "", contract.Schema.Message)
		}
	}

	candidates, err := DetectPlanCandidates(root)
	if err != nil {
		report.addCheck("plan_candidates", false, SeverityError, err.Error())
		report.addFinding(SeverityError, "plan_candidate_scan", "", err.Error())
	} else {
		report.PlanCandidates = candidates
		message := "no likely scratch plans found"
		severity := SeverityInfo
		if len(candidates) > 0 {
			message = fmt.Sprintf("%d likely scratch plan candidates found", len(candidates))
			severity = SeverityWarning
			report.addFinding(SeverityWarning, "plan_candidates", "", message)
			report.Actions = append(report.Actions, Action{
				Code:    "import_plan_candidates",
				Message: "Import likely scratch plans into user-scoped plan storage.",
				Command: "vrooli hygiene --fix-safe --plans",
			})
		}
		report.addCheck("plan_candidates", true, severity, message)
	}

	if req.FixSafe && req.Plans && len(report.PlanCandidates) > 0 {
		fixes, err := s.importPlanCandidates(report.PlanCandidates)
		if err != nil {
			report.addFinding(SeverityError, "plan_candidate_import", "", err.Error())
			report.addCheck("plan_candidate_import", false, SeverityError, err.Error())
		} else {
			report.FixesApplied = fixes
			report.addCheck("plan_candidate_import", true, SeverityInfo, fmt.Sprintf("imported %d plan candidates", len(fixes)))
		}
	}

	report.finish(req.FailOn)
	return report, nil
}

func (s Service) importPlanCandidates(candidates []PlanCandidate) ([]PlanFix, error) {
	service := planapp.Service{Root: s.Root, Home: s.Home}
	fixes := make([]PlanFix, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Status != "untracked" {
			continue
		}
		out, err := service.Import(planapp.ImportRequest{Path: filepath.Join(s.Root, filepath.FromSlash(candidate.Path)), Repo: s.Root})
		if err != nil {
			return fixes, err
		}
		fixes = append(fixes, PlanFix{Source: candidate.Path, Plan: out.Plan})
	}
	return fixes, nil
}

func DetectPlanCandidates(root string) ([]PlanCandidate, error) {
	statuses := gitStatuses(root)
	var candidates []PlanCandidate
	for rel, status := range statuses {
		if !isPlanCandidatePath(rel) {
			continue
		}
		if isAllowedPlanPath(rel) {
			continue
		}
		candidates = append(candidates, PlanCandidate{Path: rel, Status: status, Reason: "plan-like file in scratch or legacy plan location"})
	}
	return candidates, nil
}

func gitStatuses(root string) map[string]string {
	out := map[string]string{}
	cmd := exec.Command("git", "status", "--porcelain=v1", "--untracked-files=all")
	cmd.Dir = root
	data, err := cmd.Output()
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if len(line) < 4 {
				continue
			}
			status := strings.TrimSpace(line[:2])
			path := strings.TrimSpace(line[3:])
			if strings.Contains(path, " -> ") {
				parts := strings.Split(path, " -> ")
				path = parts[len(parts)-1]
			}
			path = strings.Trim(path, `"`)
			class := "modified"
			if status == "??" {
				class = "untracked"
			} else if strings.Contains(status, "D") {
				class = "deleted"
			}
			out[filepath.ToSlash(path)] = class
		}
		return out
	}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr == nil {
			out[filepath.ToSlash(rel)] = ""
		}
		return nil
	})
	return out
}

func isPlanCandidatePath(rel string) bool {
	rel = filepath.ToSlash(rel)
	if !strings.EqualFold(filepath.Ext(rel), ".md") {
		return false
	}
	return strings.HasPrefix(rel, "docs/plans/") ||
		strings.HasPrefix(rel, "plans/") ||
		strings.HasPrefix(rel, "docs/scratch/") ||
		strings.HasPrefix(rel, "scratch/") ||
		(strings.HasPrefix(rel, "scenarios/") && strings.Contains(rel, "/docs/plans/")) ||
		strings.HasPrefix(rel, ".claude/") ||
		strings.HasPrefix(rel, ".codex/")
}

func isAllowedPlanPath(rel string) bool {
	rel = filepath.ToSlash(rel)
	if strings.Contains(rel, "/backlog/") && strings.HasSuffix(rel, "/plan.md") {
		return true
	}
	if strings.Contains(rel, "/backlog/") && strings.HasSuffix(rel, "/conclusion.md") {
		return true
	}
	if strings.Contains(rel, "/pricing") || strings.Contains(rel, "/product") {
		return true
	}
	return false
}

func (r *Report) addCheck(name string, passed bool, severity Severity, message string) {
	r.Checks = append(r.Checks, Check{Name: name, Passed: passed, Severity: severity, Message: message})
}

func (r *Report) addFinding(severity Severity, code, path, message string) {
	r.Findings = append(r.Findings, Finding{Severity: severity, Code: code, Path: path, Message: message})
}

func (r *Report) finish(failOn Severity) {
	for _, finding := range r.Findings {
		switch finding.Severity {
		case SeverityError:
			r.BlockingFailures++
		case SeverityWarning:
			r.Warnings++
		}
	}
	r.Success = true
	switch failOn {
	case SeverityWarning:
		r.Success = r.BlockingFailures == 0 && r.Warnings == 0
	case SeverityError, "":
		r.Success = r.BlockingFailures == 0
	default:
		r.Success = r.BlockingFailures == 0
	}
}
