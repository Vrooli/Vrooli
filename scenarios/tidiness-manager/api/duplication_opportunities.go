package main

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// DuplicationOpportunity is a producer-owned root-cause cluster. Individual
// detector groups remain in finding evidence for diagnosis; this projection is
// the small actionable queue exposed through maturity presentation.
type DuplicationOpportunity struct {
	Key          string   `json:"key"`
	Class        string   `json:"class"`
	Boundary     string   `json:"boundary"`
	LineDebt     int      `json:"line_debt"`
	MemberGroups int      `json:"member_groups"`
	Locations    []string `json:"locations"`
}

func rankDuplicationOpportunities(findings []TidinessFinding) []DuplicationOpportunity {
	byKey := map[string]*DuplicationOpportunity{}
	for _, finding := range findings {
		if finding.RuleID != "DUPLICATED_CODE" || finding.Evidence == nil {
			continue
		}
		class, _ := finding.Evidence["duplication_class"].(string)
		if class != string(DuplicationClassOpportunity) && class != string(DuplicationClassHighLeverage) {
			continue
		}
		debt, _ := finding.Evidence["duplication_line_debt"].(int)
		locations := evidenceDuplicateLocations(finding)
		family, boundary := duplicationOpportunityFamily(locations)
		key := class + ":" + family
		opportunity := byKey[key]
		if opportunity == nil {
			opportunity = &DuplicationOpportunity{Key: family, Class: class, Boundary: boundary}
			byKey[key] = opportunity
		}
		opportunity.LineDebt += debt
		opportunity.MemberGroups++
		opportunity.Locations = append(opportunity.Locations, locations...)
	}
	result := make([]DuplicationOpportunity, 0, len(byKey))
	for _, opportunity := range byKey {
		opportunity.Locations = dedupeStrings(opportunity.Locations)
		result = append(result, *opportunity)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].LineDebt != result[j].LineDebt {
			return result[i].LineDebt > result[j].LineDebt
		}
		if result[i].Class != result[j].Class {
			return result[i].Class == string(DuplicationClassHighLeverage)
		}
		return result[i].Key < result[j].Key
	})
	return result
}

func evidenceDuplicateLocations(finding TidinessFinding) []string {
	locations, ok := finding.Evidence["locations"].([]DuplicateLocation)
	if !ok {
		return []string{finding.FilePath}
	}
	result := make([]string, 0, len(locations))
	for _, location := range locations {
		result = append(result, fmt.Sprintf("%s:%d-%d", location.Path, location.StartLine, location.EndLine))
	}
	return result
}

func duplicationOpportunityFamily(locations []string) (string, string) {
	if len(locations) == 0 {
		return "unknown", "same-package"
	}
	directories := make([]string, 0, len(locations))
	for _, location := range locations {
		file := strings.Split(location, ":")[0]
		directories = append(directories, path.Dir(file))
	}
	root := directories[0]
	for _, directory := range directories[1:] {
		for root != "." && directory != root && !strings.HasPrefix(directory, root+"/") {
			root = path.Dir(root)
		}
	}
	if root == "." || root == "" {
		root = "cross-boundary"
	}
	boundary := "same-package"
	for _, directory := range directories[1:] {
		if directory != directories[0] {
			boundary = "cross-package"
			break
		}
	}
	return root, boundary
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func applyDuplicationOpportunityPresentation(a *commonv1.MaturityAssessment, opportunities []DuplicationOpportunity) {
	if a == nil || len(opportunities) == 0 {
		return
	}
	for _, capability := range a.GetCapabilities() {
		if capability.GetId() != "duplication_control" {
			continue
		}
		for _, level := range capability.GetLevels() {
			if level.GetId() == capability.GetCurrentLevel() {
				level.NextUnlock = formatDuplicationOpportunityAction(opportunities)
				break
			}
		}
		break
	}
	a.Presentation = assessment.BuildPhasePresentation(a)
}

func formatDuplicationOpportunityAction(opportunities []DuplicationOpportunity) string {
	limit := len(opportunities)
	if limit > 5 {
		limit = 5
	}
	parts := make([]string, 0, limit)
	for _, opportunity := range opportunities[:limit] {
		parts = append(parts, fmt.Sprintf("%s (%s, %d line-debt)", opportunity.Key, opportunity.Class, opportunity.LineDebt))
	}
	return "Prioritize duplication opportunities: " + strings.Join(parts, "; ") + "."
}
