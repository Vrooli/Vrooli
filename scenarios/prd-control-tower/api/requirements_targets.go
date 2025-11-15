package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var checkboxPattern = regexp.MustCompile(`^- \[(?i:x| )\]`)

func extractOperationalTargets(entityType, entityName string) ([]OperationalTarget, error) {
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		return nil, err
	}
	prdPath := filepath.Join(vrooliRoot, entityType+"s", entityName, "PRD.md")
	content, err := os.ReadFile(prdPath)
	if err != nil {
		return nil, err
	}
	return parseOperationalTargets(string(content), entityType, entityName), nil
}

func parseOperationalTargets(content, entityType, entityName string) []OperationalTarget {
	lines := strings.Split(content, "\n")
	var targets []OperationalTarget
	inFunctional := false
	currentCategory := ""
	currentCriticality := ""

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "### ") {
			if line == "### Functional Requirements" {
				inFunctional = true
				continue
			}
			if inFunctional {
				break
			}
		}
		if !inFunctional || line == "" {
			continue
		}

		if strings.HasPrefix(line, "- **") && strings.HasSuffix(line, "**") {
			category := strings.TrimSuffix(strings.TrimPrefix(line, "- **"), "**")
			label, critic := parseCategoryLabel(category)
			currentCategory = label
			currentCriticality = critic
			continue
		}

		if !checkboxPattern.MatchString(line) {
			continue
		}

		title, notes := splitTargetLine(line)
		status := "pending"
		if strings.HasPrefix(strings.ToLower(line), "- [x]") {
			status = "complete"
		}
		id := slugify(fmt.Sprintf("%s-%s-%s", entityName, currentCategory, title))
		targets = append(targets, OperationalTarget{
			ID:          id,
			EntityType:  entityType,
			EntityName:  entityName,
			Category:    currentCategory,
			Criticality: currentCriticality,
			Title:       title,
			Notes:       notes,
			Status:      status,
			Path:        fmt.Sprintf("Functional Requirements > %s > %s", currentCategory, title),
		})
	}

	return targets
}

func parseCategoryLabel(input string) (string, string) {
	openParen := strings.Index(input, "(")
	if openParen == -1 {
		return strings.TrimSpace(input), ""
	}
	label := strings.TrimSpace(input[:openParen])
	criticality := strings.Trim(strings.TrimSpace(input[openParen+1:]), ")")
	return label, criticality
}

func splitTargetLine(line string) (string, string) {
	// Skip "- [x] " or "- [ ] " (6 characters)
	withoutCheckbox := strings.TrimSpace(line[6:])
	parts := strings.SplitN(withoutCheckbox, "_(", 2)
	title := strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		notes := strings.TrimSuffix(parts[1], ")_")
		return title, strings.TrimSpace(notes)
	}
	return title, ""
}

func slugify(value string) string {
	lower := strings.ToLower(value)
	var builder strings.Builder
	prevHyphen := false
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			prevHyphen = false
			continue
		}
		if r == ' ' || r == '-' || r == '_' {
			if !prevHyphen {
				builder.WriteRune('-')
				prevHyphen = true
			}
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return fmt.Sprintf("target-%d", time.Now().Unix())
	}
	return result
}

func linkTargetsAndRequirements(targets []OperationalTarget, groups []RequirementGroup) ([]OperationalTarget, []RequirementRecord) {
	reqList := flattenRequirements(groups)
	targetsByID := make(map[string]*OperationalTarget)
	for i := range targets {
		targetsByID[targets[i].ID] = &targets[i]
	}

	for i := range reqList {
		req := &reqList[i]
		targetMatches := findMatchingTargets(req, targets)
		if len(targetMatches) == 0 {
			continue
		}
		req.LinkedTargets = append(req.LinkedTargets, targetMatches...)
		for _, targetID := range targetMatches {
			if target, ok := targetsByID[targetID]; ok {
				target.LinkedRequirements = appendUnique(target.LinkedRequirements, req.ID)
			}
		}
	}

	var unmatched []RequirementRecord
	for _, req := range reqList {
		if len(req.LinkedTargets) == 0 {
			unmatched = append(unmatched, req)
		}
	}

	sort.Slice(unmatched, func(i, j int) bool {
		return unmatched[i].ID < unmatched[j].ID
	})

	return targets, unmatched
}

func flattenRequirements(groups []RequirementGroup) []RequirementRecord {
	var result []RequirementRecord
	var walk func(group RequirementGroup)
	walk = func(group RequirementGroup) {
		result = append(result, group.Requirements...)
		for _, child := range group.Children {
			walk(child)
		}
	}
	for _, group := range groups {
		walk(group)
	}
	return result
}

func findMatchingTargets(req *RequirementRecord, targets []OperationalTarget) []string {
	if req.PRDRef == "" || req.Title == "" {
		return nil
	}
	reqSection := normalizeText(lastSegment(req.PRDRef))
	reqCategory := normalizeText(secondSegment(req.PRDRef))
	var matches []string
	for _, target := range targets {
		titleKey := normalizeText(target.Title)
		categoryKey := normalizeText(target.Category)
		if reqCategory != "" && categoryKey != "" && !strings.Contains(categoryKey, reqCategory) && !strings.Contains(reqCategory, categoryKey) {
			continue
		}
		if strings.Contains(titleKey, reqSection) || strings.Contains(reqSection, titleKey) {
			matches = appendUnique(matches, target.ID)
		}
	}
	return matches
}

func normalizeText(value string) string {
	value = strings.ToLower(value)
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", "/", "", ".", "", "(", "", ")", "")
	return replacer.Replace(value)
}

func lastSegment(path string) string {
	parts := strings.Split(path, ">")
	return strings.TrimSpace(parts[len(parts)-1])
}

func secondSegment(path string) string {
	parts := strings.Split(path, ">")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func appendUnique(list []string, candidate string) []string {
	for _, existing := range list {
		if existing == candidate {
			return list
		}
	}
	return append(list, candidate)
}
