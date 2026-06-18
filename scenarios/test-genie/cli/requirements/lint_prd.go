package requirements

import (
	"flag"
	"fmt"
	"path/filepath"
	"sort"

	intent "intent-go"

	"github.com/vrooli/cli-core/cliutil"
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

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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
	requirements, err := (intent.FileRequirementsExtractor{}).ExtractRequirementClaims(dir)
	if err != nil {
		return prdLintResult{Status: "missing_requirements"}, nil
	}

	prdRefs := make(map[string][]string) // target -> reqIDs
	reqFiles := make(map[string]string)
	for _, req := range requirements {
		prd := intent.RequirementPRDRef(req)
		if prd == "" {
			continue
		}
		prdRefs[prd] = append(prdRefs[prd], req.ID)
		reqFiles[req.ID] = req.Anchor
	}

	targetClaims, err := (intent.FilePRDExtractor{}).ExtractPRDClaims(dir)
	if err != nil {
		return prdLintResult{Status: "missing_prd", MissingPRD: true}, nil
	}
	targets := make([]string, 0, len(targetClaims))
	for _, target := range targetClaims {
		targets = append(targets, target.ID)
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
