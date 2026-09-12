// Package wizard is the deterministic contract-authoring engine: an
// interview whose question model derives from the SAME intent-go template
// definitions the validator checks (single source — scaffolder and
// validator can never disagree), rendering a PRD.md + requirements/
// skeleton that validates clean by construction.
//
// Design guarantees (plan decisions D4/D7 made structural):
//   - No LLM or network calls anywhere in this package; answers come from
//     the caller. The only outward seam is Hinter (capability dedup),
//     which defaults to a no-op and degrades silently.
//   - Dry-run by default: Preview renders diffs; Apply is the only writer
//     and demands explicit intent.
//   - Sessions persist as JSON and resume across processes.
package wizard

import (
	"fmt"
	"strings"

	intent "intent-go"
)

// QuestionKind values.
const (
	KindText      = "text"
	KindMultiline = "multiline"
	KindOTList    = "ot_list"
)

// Question is one interview step.
type Question struct {
	ID     string
	Target string
	Prompt string
	Help   string
	Kind   string
	// Required questions block Apply while unanswered/invalid.
	Required bool
	// MinEntries applies to ot_list kinds.
	MinEntries int
	// RequiredAnchors are substrings the answer must contain — the SAME
	// `contains` rules CheckTemplateContent enforces, so an answer that
	// passes the interview can never fail the validator.
	RequiredAnchors []string
}

// OTAnswer is one structured operational-target answer.
type OTAnswer struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Answer is the caller's response to one question.
type Answer struct {
	QuestionID string     `json:"question_id"`
	Text       string     `json:"text,omitempty"`
	Items      []string   `json:"items,omitempty"`
	Targets    []OTAnswer `json:"targets,omitempty"`
}

// Questions derives the interview from the canonical template. The
// question set covers exactly what the validator checks: every required
// section's content (with its anchor phrases in the help text) and the
// three OT tiers.
func Questions() []Question {
	var out []Question
	for _, section := range intent.DefaultPRDTemplate() {
		key := intent.NormalizeSectionTitle(section.Title)
		switch key {
		case "operational targets":
			for _, sub := range section.Subsections {
				tier := tierOf(sub.Title)
				out = append(out, Question{
					ID:         "targets_" + strings.ToLower(tier),
					Target:     "operational_targets_" + strings.ToLower(tier),
					Prompt:     fmt.Sprintf("List the %s operational targets (%s)", tier, sub.Description),
					Help:       "Each becomes a checklist line `- [ ] OT-" + tier + "-NNN | Title | description` and one requirement stub. At least one per tier (aspirational entries are fine for P1/P2 — the canonical template keeps every tier populated).",
					Kind:       KindOTList,
					Required:   true,
					MinEntries: 1,
				})
			}
		default:
			q := Question{
				ID:       "section_" + slugify(key),
				Target:   slugify(key),
				Prompt:   fmt.Sprintf("Content for '%s' (%s)", section.Title, section.Description),
				Kind:     KindMultiline,
				Required: section.Required,
			}
			var required, optional []string
			for _, rule := range section.Rules {
				if rule.Kind != "contains" {
					continue
				}
				if rule.Required {
					required = append(required, rule.Pattern)
				} else {
					optional = append(optional, rule.Pattern)
				}
			}
			q.RequiredAnchors = required
			if len(required)+len(optional) > 0 {
				q.Help = "Must mention: " + strings.Join(append(append([]string{}, required...), optional...), ", ") + " (the validator checks these anchors)."
			}
			out = append(out, q)
		}
	}
	out = append(out, Question{
		ID:       "requirement_prefix",
		Target:   "requirements",
		Prompt:   "Requirement ID prefix (e.g. IMG for IMG-P0-001)",
		Help:     "Uppercase letters/digits, starting with a letter. Defaults to a prefix derived from the scenario slug.",
		Kind:     KindText,
		Required: false,
	})
	return out
}

func tierOf(title string) string {
	for _, tier := range []string{"P0", "P1", "P2"} {
		if strings.Contains(title, tier) {
			return tier
		}
	}
	return "P0"
}

func slugify(s string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore && b.Len() > 0 {
				b.WriteRune('_')
				lastUnderscore = true
			}
		}
	}
	return strings.TrimSuffix(b.String(), "_")
}

// ValidateAnswer checks one answer against its question. Empty reason
// means valid.
func ValidateAnswer(q Question, a Answer) string {
	switch q.Kind {
	case KindOTList:
		if len(a.Targets) < q.MinEntries {
			return fmt.Sprintf("needs at least %d target(s)", q.MinEntries)
		}
		for i, t := range a.Targets {
			if strings.TrimSpace(t.Title) == "" {
				return fmt.Sprintf("target #%d has no title", i+1)
			}
			if strings.Contains(t.Title, "|") || strings.Contains(t.Description, "|") {
				return fmt.Sprintf("target #%d contains '|' (reserved as the OT field separator)", i+1)
			}
		}
	case KindText, KindMultiline:
		if q.Required && strings.TrimSpace(a.Text) == "" {
			return "answer is required"
		}
		for _, anchor := range q.RequiredAnchors {
			if !strings.Contains(a.Text, anchor) {
				return fmt.Sprintf("answer must mention %q (the validator's content anchor)", anchor)
			}
		}
	}
	return ""
}
