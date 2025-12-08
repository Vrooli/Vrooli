package requirements

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"test-genie/internal/requirements/discovery"
	"test-genie/internal/requirements/parsing"
)

type prdLintResult struct {
	Status                      string           `json:"status"`
	TargetsWithoutRequirements  []targetCoverage `json:"targets_without_requirements,omitempty"`
	RequirementsWithoutTargets  []reqCoverage    `json:"requirements_without_targets,omitempty"`
	MissingPRD                  bool             `json:"missing_prd,omitempty"`
	DiscoveredTargets           []string         `json:"discovered_targets,omitempty"`
	DiscoveredRequirementPRDRef []string         `json:"requirement_prd_refs,omitempty"`
}

type targetCoverage struct {
	TargetID string `json:"target_id"`
	Source   string `json:"source"`
}

type reqCoverage struct {
	RequirementID string `json:"requirement_id"`
	File          string `json:"file"`
	Reason        string `json:"reason"`
}

func runLintPRD(args []string) error {
	fs := flag.NewFlagSet("requirements lint-prd", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output JSON")
	dirFlag, scenarioFlag := parseCommonFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	dir, err := resolveDir(*dirFlag)
	if err != nil {
		return err
	}
	if err := ensureDir(dir); err != nil {
		return err
	}

	result, err := lintPRD(dir)
	if err != nil {
		return err
	}

	if *jsonOut {
		fmt.Println(toJSON(result))
		if result.Status == "ok" {
			return nil
		}
		return fmt.Errorf("lint-prd issues detected")
	}

	if result.Status == "missing_prd" {
		return fmt.Errorf("❌ PRD.md not found – cannot verify requirements mapping")
	}
	if result.Status == "ok" {
		fmt.Printf("✅ PRD ↔ requirements mapping healthy for '%s'\n", scenarioNameFromDir(dir, *scenarioFlag))
		return nil
	}

	fmt.Println("❌ PRD ↔ requirements mismatch detected.")
	for _, t := range result.TargetsWithoutRequirements {
		fmt.Printf("  • Target without requirements: %s (%s)\n", t.TargetID, t.Source)
	}
	for _, r := range result.RequirementsWithoutTargets {
		fmt.Printf("  • Requirement missing PRD target: %s (%s) [%s]\n", r.RequirementID, r.File, r.Reason)
	}
	return fmt.Errorf("lint-prd issues detected")
}

func lintPRD(dir string) (prdLintResult, error) {
	ctx := context.Background()
	discoverer := discovery.NewDefault()
	files, err := discoverer.Discover(ctx, dir)
	if err != nil {
		return prdLintResult{Status: "error"}, fmt.Errorf("discover requirements: %w", err)
	}
	if len(files) == 0 {
		return prdLintResult{Status: "missing_requirements"}, nil
	}

	parser := parsing.NewDefault()
	index, err := parser.ParseAll(ctx, files)
	if err != nil {
		return prdLintResult{Status: "error"}, fmt.Errorf("parse requirements: %w", err)
	}

	prdRefs := make(map[string][]string) // target -> reqIDs
	reqFiles := make(map[string]string)
	for _, module := range index.Modules {
		for _, req := range module.Requirements {
			if req.PRDRef == "" {
				continue
			}
			prd := strings.ToUpper(req.PRDRef)
			prdRefs[prd] = append(prdRefs[prd], req.ID)
			reqFiles[req.ID] = module.FilePath
		}
	}

	targets, err := collectTargetsFromPRD(dir)
	if err != nil {
		return prdLintResult{Status: "missing_prd", MissingPRD: true}, nil
	}

	targetSet := make(map[string]struct{})
	for _, t := range targets {
		targetSet[t] = struct{}{}
	}

	result := prdLintResult{
		Status:                      "ok",
		DiscoveredTargets:           targets,
		DiscoveredRequirementPRDRef: sortedKeys(prdRefs),
	}

	for t := range targetSet {
		if _, ok := prdRefs[t]; !ok {
			result.TargetsWithoutRequirements = append(result.TargetsWithoutRequirements, targetCoverage{
				TargetID: t,
				Source:   "PRD.md",
			})
		}
	}

	for prdRef, reqIDs := range prdRefs {
		if _, ok := targetSet[prdRef]; !ok {
			for _, reqID := range reqIDs {
				result.RequirementsWithoutTargets = append(result.RequirementsWithoutTargets, reqCoverage{
					RequirementID: reqID,
					File:          relativize(reqFiles[reqID], dir),
					Reason:        fmt.Sprintf("PRD target %s not found", prdRef),
				})
			}
		}
	}

	if len(result.TargetsWithoutRequirements) > 0 || len(result.RequirementsWithoutTargets) > 0 {
		result.Status = "issues"
	}

	return result, nil
}

func collectTargetsFromPRD(dir string) ([]string, error) {
	prdPath := filepath.Join(dir, "PRD.md")
	data, err := os.ReadFile(prdPath)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`OT-[Pp][0-2]-\d{3}`)
	matches := re.FindAllString(string(data), -1)
	unique := make(map[string]struct{})
	for _, m := range matches {
		unique[strings.ToUpper(m)] = struct{}{}
	}
	result := make([]string, 0, len(unique))
	for m := range unique {
		result = append(result, m)
	}
	sort.Strings(result)
	return result, nil
}

func relativize(path, base string) string {
	if path == "" {
		return ""
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
