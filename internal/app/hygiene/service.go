package hygiene

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	planapp "github.com/vrooli/vrooli/internal/app/plans"
	shareddriftapp "github.com/vrooli/vrooli/internal/app/shareddrift"
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
	if !req.IncludeContract && !req.IncludePlans && !req.IncludeDrift {
		req.IncludeContract = true
		req.IncludePlans = true
		req.IncludeDrift = true
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
		candidates, err := DetectPlanCandidates(root)
		if err != nil {
			report.addCheck("plan_candidates", false, SeverityError, err.Error())
			report.addFinding(Finding{
				Severity:   SeverityError,
				Code:       "plan_candidate_scan",
				Message:    err.Error(),
				Fixability: FixabilityManual,
				NextActions: []Action{{
					Code:    "inspect_plan_candidate_scan",
					Message: "Inspect the plan-candidate scan error and rerun hygiene.",
				}},
			})
		} else {
			report.PlanCandidates = candidates
			message := "no likely scratch plans found"
			severity := SeverityInfo
			if len(candidates) > 0 {
				message = fmt.Sprintf("%d likely scratch plan candidates found", len(candidates))
				severity = SeverityWarning
				action := Action{
					Code:       "import_plan_candidates",
					Message:    "Import untracked scratch plan candidates into user-scoped plan storage.",
					Command:    "vrooli hygiene --fix-safe --plans",
					Fixability: FixabilityAutomatic,
				}
				report.addFinding(Finding{
					Severity:   SeverityWarning,
					Code:       "plan_candidates",
					Message:    message,
					Why:        "Scratch implementation plans should live in user-scoped plan storage until intentionally promoted into the repository.",
					Fixability: FixabilityAutomatic,
					NextActions: []Action{
						action,
						{
							Code:       "review_modified_plan_candidates",
							Message:    "Review modified plan files manually before deciding whether to promote, revert, or archive them.",
							Fixability: FixabilityManual,
						},
					},
				})
				report.Actions = append(report.Actions, action)
			}
			report.addCheck("plan_candidates", true, severity, message)
		}
	}

	if req.IncludeDrift {
		drift, err := shareddriftapp.Service{Root: root}.Check(shareddriftapp.CheckRequest{
			OnlyTouched: true,
			Fix:         false,
		})
		if err != nil {
			report.addCheck("shared_drift", false, SeverityError, err.Error())
			report.addFinding(Finding{
				Severity:   SeverityError,
				Code:       "shared_drift_scan",
				Message:    err.Error(),
				Fixability: FixabilityManual,
				NextActions: []Action{{
					Code:    "inspect_shared_drift_scan",
					Message: "Inspect the shared-drift scan error and rerun hygiene after correcting it.",
					Command: "vrooli check-shared-drift --only-touched --json",
				}},
			})
		} else {
			report.SharedDrift = &drift
			if drift.Clean {
				message := "no dependent scenarios stale"
				if len(drift.Scenarios) == 0 {
					message = "no shared-package changes staged"
				}
				report.addCheck("shared_drift", true, SeverityInfo, message)
			} else {
				stale := driftStaleScenarios(drift)
				message := fmt.Sprintf("%d dependent scenarios are stale relative to shared packages", len(stale))
				report.addCheck("shared_drift", false, SeverityError, message)
				action := Action{
					Code:       "fix_shared_drift",
					Message:    "Tidy stale scenario go.mod/go.sum then stage the changes.",
					Command:    "vrooli check-shared-drift --fix --only-touched",
					Fixability: FixabilityGuided,
				}
				report.addFinding(Finding{
					Severity:    SeverityError,
					Code:        "shared_drift",
					Message:     message,
					Why:         "Stale scenario go.mod/go.sum entries cause scenarios to fail to start after a shared package changes.",
					Locations:   driftLocations(stale),
					Fixability:  FixabilityGuided,
					NextActions: []Action{action},
				})
				report.Actions = append(report.Actions, action)
			}
		}
	}

	if req.FixSafe && req.Plans && len(report.PlanCandidates) > 0 {
		fixes, err := s.importPlanCandidates(report.PlanCandidates)
		if err != nil {
			report.addFinding(Finding{
				Severity:   SeverityError,
				Code:       "plan_candidate_import",
				Message:    err.Error(),
				Fixability: FixabilityManual,
				NextActions: []Action{{
					Code:    "inspect_plan_candidate_import",
					Message: "Inspect the plan import error and retry after correcting it.",
				}},
			})
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

func driftStaleScenarios(drift shareddriftapp.Report) []shareddriftapp.ScenarioReport {
	var out []shareddriftapp.ScenarioReport
	for _, sc := range drift.Scenarios {
		if sc.Status == shareddriftapp.StatusStaleModules || sc.Status == shareddriftapp.StatusStaleBuild {
			out = append(out, sc)
		}
	}
	return out
}

func driftLocations(stale []shareddriftapp.ScenarioReport) []string {
	const maxLocations = 5
	locations := make([]string, 0, maxLocations+1)
	for i, sc := range stale {
		if i >= maxLocations {
			locations = append(locations, fmt.Sprintf("... %d more (see shared_drift summary)", len(stale)-maxLocations))
			break
		}
		locations = append(locations, sc.Path)
	}
	return locations
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
