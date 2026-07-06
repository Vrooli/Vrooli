package hygiene

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
)

type Service struct {
	Root                      string
	Home                      string
	PlanReconciler            PlanReconciler
	DependencyFreshnessRunner DependencyFreshnessRunner
}

func (s Service) Run(req Request) (Report, error) {
	root := filepath.Clean(strings.TrimSpace(s.Root))
	if root == "" || root == "." {
		return Report{}, fmt.Errorf("repo root is required")
	}
	if req.FailOn == "" {
		req.FailOn = SeverityError
	}
	if !req.IncludeContract && !req.IncludePlans && !req.IncludeDrift && !req.IncludePnpmConfig && !req.IncludeFreshness {
		req.IncludeContract = true
		req.IncludePlans = true
		req.IncludeDrift = true
		req.IncludePnpmConfig = true
		req.IncludeFreshness = true
	}
	report := Report{Root: root, Success: true}

	if req.IncludeContract {
		contract, err := contractapp.Validate(root)
		if err != nil {
			report.addFinding(Finding{
				Severity:   SeverityError,
				Code:       "contract_validation_error",
				Message:    err.Error(),
				Fixability: FixabilityManual,
				NextActions: []Action{{
					Code:    "inspect_contract_validation_error",
					Message: "Inspect the contract validation error and rerun hygiene after correcting it.",
					Command: "vrooli contract validate --json",
				}},
			})
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
				report.addFinding(contractFinding(check.Name, check.Message))
			}
			if !contract.Schema.Passed {
				report.addFinding(Finding{
					Severity:   SeverityError,
					Code:       "repo_contract_schema",
					Message:    contract.Schema.Message,
					Why:        "The repository contract schema is the machine-readable source for expected repository boundaries.",
					Fixability: FixabilityManual,
					NextActions: []Action{{
						Code:    "inspect_repo_contract_schema",
						Message: "Inspect .vrooli/repo-contract.json and correct the schema violation.",
						Command: "vrooli contract validate --json",
					}},
				})
			}
		}
	}

	if req.IncludePlans {
		registry := NewRegistry(s.planProvider())
		if err := registry.Run(context.Background(), req, &report, plansProviderID); err != nil {
			return Report{}, err
		}
	}

	if req.IncludeDrift {
		registry := NewRegistry(s.sdaFreshnessProvider())
		if err := registry.Run(context.Background(), req, &report, dependencyFreshnessProviderID); err != nil {
			return Report{}, err
		}
	}

	if req.IncludePnpmConfig {
		s.checkPnpmConfig(&report, req.FixSafe)
		s.checkScenarioPnpm(&report)
	}

	if req.IncludeFreshness {
		s.checkTestFreshness(&report)
	}

	report.finish(req.FailOn)
	return report, nil
}

func (s Service) planProvider() Provider {
	reconciler := s.PlanReconciler
	var reconcileErr error
	if reconciler == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if resolved, err := NewDefaultPlanReconciler(ctx); err == nil {
			reconciler = resolved
		} else {
			reconcileErr = err
		}
	}
	return plansProvider{root: s.Root, reconciler: reconciler, unavailableReason: errorString(reconcileErr)}
}

func (s Service) sdaFreshnessProvider() Provider {
	return sdaFreshnessProvider{root: s.Root, runner: s.DependencyFreshnessRunner}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func contractFinding(name, message string) Finding {
	finding := Finding{
		Severity:   SeverityError,
		Code:       "repo_contract_" + name,
		Message:    message,
		Locations:  extractLocations(message),
		Fixability: FixabilityManual,
		NextActions: []Action{{
			Code:    "inspect_repo_contract_check",
			Message: "Inspect the failing repository contract check and rerun hygiene after correcting it.",
			Command: "vrooli contract validate --json",
		}},
	}
	switch name {
	case "project_config_surface":
		finding.Why = "The .vrooli/ directory is reserved for approved repo metadata and local build output so generated runtime state does not drift into commits."
		finding.Fixability = FixabilityGuided
		finding.NextActions = []Action{
			{
				Code:       "inspect_project_config_surface",
				Message:    "Inspect the unapproved .vrooli/ entry and remove it if it is generated local state.",
				Command:    "ls -la .vrooli",
				Fixability: FixabilityGuided,
			},
			{
				Code:       "remove_generated_vrooli_state",
				Message:    "If .vrooli/logs or .vrooli/state are generated local state, remove them.",
				Command:    "rm -rf .vrooli/logs .vrooli/state",
				Fixability: FixabilityGuided,
			},
		}
	case "personal_absolute_paths":
		finding.Why = "Committed personal home paths make tests machine-specific and can leak local environment details."
		finding.NextActions = []Action{{
			Code:       "remove_personal_absolute_paths",
			Message:    "Replace personal absolute paths with temp dirs, fixture-relative paths, or home-independent examples.",
			Fixability: FixabilityManual,
		}}
	case "ollama_gateway_only":
		finding.Path = firstPathBefore(message, " contains ")
		finding.Why = "Ollama access should go through the resource-ollama gateway so local resources remain centrally managed and portable."
		finding.NextActions = []Action{{
			Code:       "use_ollama_gateway",
			Message:    "Route Ollama calls through the resource-ollama gateway instead of direct /api/generate calls.",
			Fixability: FixabilityManual,
		}}
	}
	return finding
}

var locationPattern = regexp.MustCompile(`[A-Za-z0-9_./-]+:\d+`)

func extractLocations(message string) []string {
	matches := locationPattern.FindAllString(message, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	locations := make([]string, 0, len(matches))
	for _, match := range matches {
		if seen[match] {
			continue
		}
		seen[match] = true
		locations = append(locations, match)
	}
	return locations
}

func firstPathBefore(message, marker string) string {
	index := strings.Index(message, marker)
	if index < 0 {
		return ""
	}
	prefix := strings.TrimSpace(message[:index])
	fields := strings.Fields(prefix)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
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
		candidates = append(candidates, PlanCandidate{Path: rel, Status: status, Reason: "plan-like file in scratch or plan source location"})
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

func (r *Report) addFinding(finding Finding) {
	r.Findings = append(r.Findings, finding)
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
