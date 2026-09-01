package hygiene

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tidinessprovider"
	"github.com/vrooli/vrooli/internal/tuning"
)

const (
	serviceParameterA = 4

	// statusUntracked is the gitStatuses class for a file git neither tracks
	// nor ignores.
	statusUntracked = "untracked"
)

type Service struct {
	Root                      string
	Home                      string
	PlanReconciler            PlanReconciler
	DependencyFreshnessRunner DependencyFreshnessRunner
	StructureProvider         Provider
	TidinessProvider          Provider
}

func (s Service) Run(req Request) (Report, error) {
	root := filepath.Clean(strings.TrimSpace(s.Root))
	if root == "" || root == "." {
		return Report{}, fmt.Errorf("repo root is required")
	}
	if req.FailOn == "" {
		req.FailOn = SeverityError
	}
	if !req.IncludeContract && !req.IncludePlans && !req.IncludeDrift && !req.IncludeFreshness && !req.IncludeTidiness {
		req.IncludeContract = true
		req.IncludePlans = true
		req.IncludeDrift = true
		req.IncludeFreshness = true
		req.IncludeTidiness = true
	}
	report := Report{Root: root, Success: true}

	if req.IncludeContract {
		registry := NewRegistry(s.structureProvider(), s.layoutProvider())
		if err := registry.Run(context.Background(), req, &report, structureProviderID, layoutProviderID); err != nil {
			return Report{}, err
		}
	}

	if req.IncludeTidiness {
		registry := NewRegistry(s.tidinessProvider())
		if err := registry.Run(context.Background(), req, &report, tidinessProviderID); err != nil {
			return Report{}, err
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
		ctx, cancel := context.WithTimeout(context.Background(), tuning.HealthCheckTimeout())
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

func (s Service) tidinessProvider() Provider {
	if s.TidinessProvider != nil {
		return s.TidinessProvider
	}
	return tidinessProvider{root: s.Root, client: tidinessprovider.NewDefault()}
}

func contractFinding(name, message string) Finding {
	return Finding{
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
	cmd := shell.NewCommand("git", "status", "--porcelain=v1", "--untracked-files=all")
	cmd.Dir = root
	data, err := cmd.Output()
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if len(line) < serviceParameterA {
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
				class = statusUntracked
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

// artifactResidueExtensions are file types that are machine output rather than
// authored prose. Inside a documentation tree a file with one of these
// extensions was generated by a run, not written by anyone.
var artifactResidueExtensions = map[string]struct{}{
	".csv": {}, ".html": {}, ".json": {}, ".log": {},
	".out": {}, ".png": {}, ".tsv": {}, ".txt": {},
}

// artifactResidueDirs name the directories plan execution creates to hold
// captures. The docs-tree test in isArtifactResiduePath has to come first:
// outside a docs tree these same names are routinely real source, because Go
// packages and generated protobuf namespaces are also called "evidence".
var artifactResidueDirs = map[string]struct{}{
	"captures": {}, "evidence": {}, "phases": {}, "reports": {},
}

// artifactResidueRoots are repository-root directories that only ever hold
// generated captures, so anything untracked beneath them is residue whatever
// its file type.
var artifactResidueRoots = []string{".artifacts/", "coverage/", "evidence/"}

// DetectArtifactResidue reports untracked, unignored files that look like
// generated run evidence parked in a source tree. These are invisible to every
// other gate: no manifest lists them, no ignore rule covers them, and no Go
// code constructs their paths, so nothing else in the repository can see them.
func DetectArtifactResidue(root string) []string {
	var residue []string
	for rel, status := range gitStatuses(root) {
		if status != statusUntracked {
			continue
		}
		if isArtifactResiduePath(rel) {
			residue = append(residue, filepath.ToSlash(rel))
		}
	}
	sort.Strings(residue)
	return residue
}

func isArtifactResiduePath(rel string) bool {
	rel = filepath.ToSlash(rel)
	for _, prefix := range artifactResidueRoots {
		if strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	if !isInsideDocsTree(rel) {
		return false
	}
	if _, ok := artifactResidueExtensions[strings.ToLower(filepath.Ext(rel))]; ok {
		return true
	}
	// A capture directory inside a docs tree makes even a .md file residue:
	// docs/reports/ and docs/**/evidence/ hold per-phase write-ups that no
	// manifest registers.
	for _, segment := range strings.Split(filepath.ToSlash(filepath.Dir(rel)), "/") {
		if _, ok := artifactResidueDirs[segment]; ok {
			return true
		}
		if strings.HasPrefix(segment, "phase") {
			return true
		}
	}
	return false
}

func isInsideDocsTree(rel string) bool {
	return strings.HasPrefix(rel, "docs/") || strings.Contains(rel, "/docs/")
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
