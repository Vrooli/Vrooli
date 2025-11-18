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

var (
	checkboxPattern = regexp.MustCompile(`^- \[(?i:x| )\]`)
	reqLinkPattern  = regexp.MustCompile("`\\[req:([A-Z0-9,-]+)\\]`")
)

const (
	modernOperationalHeader = "## 🎯 Operational Targets"
	modernP0Heading         = "### 🔴 P0 – Must ship for viability"
	modernP1Heading         = "### 🟠 P1 – Should have post-launch"
	modernP2Heading         = "### 🟢 P2 – Future / expansion"
)

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
	targets := parseOperationalTargets(string(content), entityType, entityName)
	return targets, nil
}

func parseOperationalTargets(content, entityType, entityName string) []OperationalTarget {
	if strings.Contains(strings.ToLower(content), strings.ToLower(modernOperationalHeader)) {
		if modern := parseModernOperationalTargets(content, entityType, entityName); len(modern) > 0 {
			return modern
		}
	}
	return parseLegacyOperationalTargets(content, entityType, entityName)
}

func parseModernOperationalTargets(content, entityType, entityName string) []OperationalTarget {
	lines := strings.Split(content, "\n")
	var targets []OperationalTarget
	inSection := false
	currentCriticality := ""

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "## ") {
			if strings.EqualFold(line, modernOperationalHeader) {
				inSection = true
				continue
			}
			if inSection {
				break
			}
		}

		if !inSection || line == "" {
			continue
		}

		if strings.HasPrefix(line, "###") {
			switch {
			case strings.Contains(line, "P0"):
				currentCriticality = "P0"
			case strings.Contains(line, "P1"):
				currentCriticality = "P1"
			case strings.Contains(line, "P2"):
				currentCriticality = "P2"
			default:
				currentCriticality = ""
			}
			continue
		}

		if !strings.HasPrefix(line, "- [") {
			continue
		}

		target := parseModernTargetLine(line, entityType, entityName, currentCriticality)
		targets = append(targets, target)
	}

	return targets
}

func parseModernTargetLine(line, entityType, entityName, criticality string) OperationalTarget {
	status := "pending"
	if strings.HasPrefix(strings.ToLower(line), "- [x]") {
		status = "complete"
	}

	if len(line) < 6 {
		return OperationalTarget{}
	}
	content := strings.TrimSpace(line[6:])
	var linked []string
	if matches := reqLinkPattern.FindStringSubmatch(content); len(matches) > 1 {
		ids := strings.Split(matches[1], ",")
		for _, id := range ids {
			trimmed := strings.TrimSpace(id)
			if trimmed != "" {
				linked = append(linked, trimmed)
			}
		}
		content = strings.TrimSpace(reqLinkPattern.ReplaceAllString(content, ""))
	}

	parts := strings.SplitN(content, "|", 3)
	var id, title, description string
	if len(parts) > 0 {
		id = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		title = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		description = strings.TrimSpace(parts[2])
	}

	normalizedCriticality := strings.ToUpper(strings.TrimSpace(criticality))
	if normalizedCriticality == "" {
		normalizedCriticality = "P2"
	}

	if id == "" {
		id = slugify(fmt.Sprintf("%s-%s-%s", entityName, normalizedCriticality, title))
	}

	pathLeaf := id
	if pathLeaf == "" {
		pathLeaf = title
	}

	return OperationalTarget{
		ID:                 id,
		EntityType:         entityType,
		EntityName:         entityName,
		Category:           normalizedCriticality,
		Criticality:        normalizedCriticality,
		Title:              title,
		Notes:              description,
		Status:             status,
		Path:               fmt.Sprintf("Operational Targets > %s > %s", normalizedCriticality, pathLeaf),
		LinkedRequirements: linked,
	}
}

func parseLegacyOperationalTargets(content, entityType, entityName string) []OperationalTarget {
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

		title, notes := splitLegacyTargetLine(line)
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

func splitLegacyTargetLine(line string) (string, string) {
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
		idKey := normalizeText(target.ID)
		if reqCategory != "" && categoryKey != "" && categoryKey != reqCategory {
			continue
		}
		if reqSection != "" && idKey != "" && idKey == reqSection {
			matches = appendUnique(matches, target.ID)
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

func updateDraftOperationalTargets(draftContent string, targets []OperationalTarget) string {
	lowerContent := strings.ToLower(draftContent)
	if strings.Contains(lowerContent, strings.ToLower(modernOperationalHeader)) {
		return replaceModernTargetsSection(draftContent, serializeOperationalTargetsNew(targets))
	}
	return replaceLegacyFunctionalSection(draftContent, serializeOperationalTargetsLegacy(targets))
}

func replaceModernTargetsSection(content, replacement string) string {
	lines := strings.Split(content, "\n")
	var result []string
	replaced := false

	for i := 0; i < len(lines); i++ {
		if strings.EqualFold(strings.TrimSpace(lines[i]), modernOperationalHeader) {
			replaced = true
			result = append(result, strings.TrimRight(replacement, "\n"))

			for j := i + 1; j < len(lines); j++ {
				trimmed := strings.TrimSpace(lines[j])
				if strings.HasPrefix(trimmed, "## ") && !strings.EqualFold(trimmed, modernOperationalHeader) {
					i = j - 1
					break
				}
				if j == len(lines)-1 {
					i = j
				}
			}
			continue
		}
		result = append(result, lines[i])
	}

	if !replaced {
		if strings.TrimSpace(content) == "" {
			return strings.TrimRight(replacement, "\n") + "\n"
		}
		return strings.TrimRight(content, "\n") + "\n\n" + strings.TrimRight(replacement, "\n") + "\n"
	}

	return strings.Join(result, "\n")
}

func replaceLegacyFunctionalSection(content, replacement string) string {
	lines := strings.Split(content, "\n")
	var result []string
	replaced := false
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "### Functional Requirements" {
			replaced = true
			result = append(result, strings.TrimRight(replacement, "\n"))

			for j := i + 1; j < len(lines); j++ {
				next := strings.TrimSpace(lines[j])
				if strings.HasPrefix(next, "### ") && next != "### Functional Requirements" {
					i = j - 1
					break
				}
				if j == len(lines)-1 {
					i = j
				}
			}
			continue
		}
		result = append(result, lines[i])
	}

	if !replaced {
		return strings.TrimRight(content, "\n") + "\n\n" + strings.TrimRight(replacement, "\n") + "\n"
	}

	return strings.Join(result, "\n")
}

func serializeOperationalTargetsNew(targets []OperationalTarget) string {
	categoryGroups := map[string][]OperationalTarget{
		"P0": {},
		"P1": {},
		"P2": {},
	}
	for _, target := range targets {
		key := strings.ToUpper(strings.TrimSpace(target.Criticality))
		if key == "" {
			key = strings.ToUpper(strings.TrimSpace(target.Category))
		}
		if key == "" {
			key = "P2"
		}
		categoryGroups[key] = append(categoryGroups[key], target)
	}

	var builder strings.Builder
	builder.WriteString(modernOperationalHeader)
	builder.WriteString("\n\n")

	order := []struct {
		heading     string
		criticality string
	}{
		{modernP0Heading, "P0"},
		{modernP1Heading, "P1"},
		{modernP2Heading, "P2"},
	}

	for idx, entry := range order {
		builder.WriteString(entry.heading)
		builder.WriteString("\n")
		for _, target := range categoryGroups[entry.criticality] {
			status := " "
			if strings.EqualFold(target.Status, "complete") {
				status = "x"
			}
			id := target.ID
			if id == "" {
				id = slugify(fmt.Sprintf("%s-%s-%s", target.EntityName, entry.criticality, target.Title))
			}
			title := target.Title
			if title == "" {
				title = "Target"
			}
			description := target.Notes
			line := fmt.Sprintf("- [%s] %s | %s | %s", status, id, title, description)
			builder.WriteString(line)
			builder.WriteString("\n")
		}
		if idx < len(order)-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func serializeOperationalTargetsLegacy(targets []OperationalTarget) string {
	if len(targets) == 0 {
		return "### Functional Requirements\n"
	}

	categoryMap := make(map[string][]OperationalTarget)
	var categoryOrder []string
	for _, target := range targets {
		key := target.Category
		if _, exists := categoryMap[key]; !exists {
			categoryOrder = append(categoryOrder, key)
		}
		categoryMap[key] = append(categoryMap[key], target)
	}

	var builder strings.Builder
	builder.WriteString("### Functional Requirements\n")

	for _, category := range categoryOrder {
		items := categoryMap[category]
		if len(items) == 0 {
			continue
		}

		criticality := items[0].Criticality
		if criticality != "" {
			builder.WriteString(fmt.Sprintf("- **%s (%s)**\n", category, criticality))
		} else {
			builder.WriteString(fmt.Sprintf("- **%s**\n", category))
		}

		for _, target := range items {
			checkbox := "[ ]"
			if target.Status == "complete" {
				checkbox = "[x]"
			}

			line := fmt.Sprintf("  - %s %s", checkbox, target.Title)
			if target.Notes != "" {
				line += fmt.Sprintf(" _(%s)_", target.Notes)
			}
			if len(target.LinkedRequirements) > 0 {
				reqList := strings.Join(target.LinkedRequirements, ",")
				line += fmt.Sprintf(" `[req:%s]`", reqList)
			}
			builder.WriteString(line + "\n")
		}
	}

	return builder.String()
}
