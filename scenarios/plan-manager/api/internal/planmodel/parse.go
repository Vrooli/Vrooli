package planmodel

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	titleRe              = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	sectionRe            = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)
	phaseRe              = regexp.MustCompile(`(?m)^###\s+Phase\s+(\d+)\s*[—:-]\s*(.+?)\s*$`)
	referenceRe          = regexp.MustCompile(`\[(CODE|REQ|DOC):\s*([^\]]+?)\]`)
	malformedReferenceRe = regexp.MustCompile(`\[(CODE|REQ|DOC)(?:\s*:)?\s*(?:\]|$)`)
	bulletKeyValueLineRe = regexp.MustCompile(`(?m)^-\s*([A-Za-z ]+):\s*(.+?)\s*$`)
)

// ParsePlanMarkdown parses a markdown plan into the structured model. It is a
// pure, deterministic adoption helper for the rendered markdown view.
func ParsePlanMarkdown(markdown string) (Plan, error) {
	if strings.TrimSpace(markdown) == "" {
		return Plan{}, ErrInvalidPlan{Reason: "empty markdown"}
	}
	var p Plan
	if m := titleRe.FindStringSubmatch(markdown); m != nil {
		p.Title = strings.TrimSpace(m[1])
	}
	if p.Title == "" {
		return Plan{}, ErrInvalidPlan{Reason: "markdown has no title heading"}
	}
	if err := validateMachineReadableMarkup(markdown); err != nil {
		return Plan{}, err
	}

	sections := extractSections(markdown)
	p.Purpose = sections["purpose"]
	p.Scope = sections["scope"]
	p.Constraints = sections["constraints"]
	p.NonGoals = firstNonEmpty(sections["non-goals"], sections["non goals"])
	p.DefinitionOfDone = firstNonEmpty(sections["definition of done"], sections["definition-of-done"])

	p.References = parseReferences(markdown)
	p.Phases = parsePhases(markdown)
	return p, nil
}

func validateMachineReadableMarkup(markdown string) error {
	if m := malformedReferenceRe.FindString(markdown); m != "" {
		return ErrInvalidPlan{Reason: "malformed reference marker " + m}
	}
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "### Phase") && !phaseRe.MatchString(trimmed) {
			return ErrInvalidPlan{Reason: "malformed phase heading " + trimmed}
		}
	}
	return nil
}

func extractSections(markdown string) map[string]string {
	out := map[string]string{}
	locs := sectionRe.FindAllStringSubmatchIndex(markdown, -1)
	for i, loc := range locs {
		heading := strings.ToLower(strings.TrimSpace(markdown[loc[2]:loc[3]]))
		bodyStart := loc[1]
		bodyEnd := len(markdown)
		if i+1 < len(locs) {
			bodyEnd = locs[i+1][0]
		}
		body := markdown[bodyStart:bodyEnd]
		if idx := phaseRe.FindStringIndex(body); idx != nil {
			body = body[:idx[0]]
		}
		out[heading] = strings.TrimSpace(body)
	}
	return out
}

func parseReferences(markdown string) []Reference {
	matches := referenceRe.FindAllStringSubmatch(markdown, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]Reference, 0, len(matches))
	for _, m := range matches {
		kind := referenceKindFromMarker(m[1])
		target := strings.TrimSpace(m[2])
		key := string(kind) + "|" + target
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Reference{Kind: kind, Target: target})
	}
	return out
}

func parsePhases(markdown string) []Phase {
	locs := phaseRe.FindAllStringSubmatchIndex(markdown, -1)
	if len(locs) == 0 {
		return nil
	}
	out := make([]Phase, 0, len(locs))
	for i, loc := range locs {
		order, _ := strconv.Atoi(markdown[loc[2]:loc[3]])
		title := strings.TrimSpace(markdown[loc[4]:loc[5]])
		bodyStart := loc[1]
		bodyEnd := len(markdown)
		if i+1 < len(locs) {
			bodyEnd = locs[i+1][0]
		}
		body := markdown[bodyStart:bodyEnd]
		ph := Phase{Order: order, Title: title, Status: PhaseStatusTodo}
		for _, kv := range bulletKeyValueLineRe.FindAllStringSubmatch(body, -1) {
			key := strings.ToLower(strings.TrimSpace(kv[1]))
			val := strings.TrimSpace(kv[2])
			switch key {
			case "intent":
				ph.Intent = val
			case "acceptance":
				ph.Acceptance = val
			case "status":
				ph.Status = phaseStatusFromLabel(val)
			}
		}
		ph.References = parseReferences(body)
		out = append(out, ph)
	}
	return out
}

func referenceKindFromMarker(marker string) ReferenceKind {
	switch strings.ToUpper(strings.TrimSpace(marker)) {
	case "REQ":
		return ReferenceReq
	case "DOC":
		return ReferenceDoc
	default:
		return ReferenceCode
	}
}

func phaseStatusFromLabel(s string) PhaseStatus {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(s), "*")) {
	case "active":
		return PhaseStatusActive
	case "done":
		return PhaseStatusDone
	case "blocked":
		return PhaseStatusBlocked
	default:
		return PhaseStatusTodo
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
