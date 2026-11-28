package wizard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	intent "intent-go"
)

// FileDiff is one artifact the scaffold would write.
type FileDiff struct {
	// Path is target-scenario-relative (e.g. "PRD.md").
	Path   string
	Before string
	After  string
}

// tierMeta orders the OT tiers and names their requirement modules.
var tierMeta = []struct {
	id, questionID, moduleSlug string
}{
	{"P0", "targets_p0", "01-must-ship"},
	{"P1", "targets_p1", "02-post-launch"},
	{"P2", "targets_p2", "03-future"},
}

var prefixPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]+$`)

// Scaffold renders the artifacts the current answers produce, as diffs
// against the target tree. Pure read: nothing is written.
func (e *Engine) Scaffold(s Session) ([]FileDiff, []string, error) {
	blocking := Remaining(s)
	prd, err := renderPRD(s)
	if err != nil {
		return nil, blocking, err
	}
	files := []FileDiff{{Path: "PRD.md", After: prd}}
	files = append(files, FileDiff{Path: "requirements/index.json", After: renderIndex(s)})
	files = append(files, FileDiff{Path: "requirements/README.md", After: requirementsReadme})
	for _, tier := range tierMeta {
		a, ok := s.Answers[tier.questionID]
		if !ok || len(a.Targets) == 0 {
			continue
		}
		module, err := renderModule(s, tier.id, a.Targets)
		if err != nil {
			return nil, blocking, err
		}
		files = append(files, FileDiff{Path: "requirements/" + tier.moduleSlug + "/module.json", After: module})
	}
	for i := range files {
		if data, err := os.ReadFile(filepath.Join(s.TargetDir, filepath.FromSlash(files[i].Path))); err == nil {
			files[i].Before = string(data)
		}
	}
	return files, blocking, nil
}

// Apply writes the scaffold. It refuses while required questions are
// open — output validates clean by construction or not at all.
func (e *Engine) Apply(s Session) ([]string, error) {
	files, blocking, err := e.Scaffold(s)
	if err != nil {
		return nil, err
	}
	if len(blocking) > 0 {
		return nil, fmt.Errorf("cannot apply: required questions unanswered: %s", strings.Join(blocking, ", "))
	}
	var written []string
	for _, f := range files {
		path := filepath.Join(s.TargetDir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return written, err
		}
		if err := os.WriteFile(path, []byte(f.After), 0o644); err != nil {
			return written, err
		}
		written = append(written, f.Path)
	}
	return written, nil
}

func renderPRD(s Session) (string, error) {
	var b strings.Builder
	b.WriteString("# Product Requirements Document (PRD)\n\n")
	b.WriteString("> **Template Version**: 2.0\n")
	b.WriteString("> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`\n")
	b.WriteString("> **Validation**: Enforced by `business-health` (`validate scenario " + s.Scenario + "`)\n")
	b.WriteString("> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)\n")

	for _, section := range intent.DefaultPRDTemplate() {
		key := intent.NormalizeSectionTitle(section.Title)
		if key == "operational targets" {
			b.WriteString("\n## " + section.Title + "\n")
			b.WriteString("\n> Checkboxes auto-update from requirements sync; do not hand-edit them.\n")
			for _, tier := range tierMeta {
				sub := subsectionFor(section, tier.id)
				b.WriteString("\n### " + sub + "\n")
				targets := s.Answers[tier.questionID].Targets
				if len(targets) == 0 {
					b.WriteString("\n_None yet._\n")
					continue
				}
				b.WriteString("\n")
				for i, t := range targets {
					b.WriteString(fmt.Sprintf("- [ ] OT-%s-%03d | %s | %s\n", tier.id, i+1, strings.TrimSpace(t.Title), oneLine(t.Description)))
				}
			}
			continue
		}
		answer, ok := s.Answers["section_"+slugify(key)]
		if !ok && !section.Required {
			continue // optional section (Appendix) omitted when unanswered
		}
		b.WriteString("\n## " + section.Title + "\n\n")
		text := strings.TrimSpace(answer.Text)
		if text == "" {
			text = "TODO: fill in " + section.Description + "."
		}
		b.WriteString(text + "\n")
	}
	return b.String(), nil
}

func subsectionFor(section intent.TemplateSection, tier string) string {
	for _, sub := range section.Subsections {
		if strings.Contains(sub.Title, tier) {
			return sub.Title
		}
	}
	return tier
}

func renderIndex(s Session) string {
	imports := []string{}
	for _, tier := range tierMeta {
		if a, ok := s.Answers[tier.questionID]; ok && len(a.Targets) > 0 {
			imports = append(imports, tier.moduleSlug+"/module.json")
		}
	}
	index := map[string]any{
		"_metadata": map[string]any{
			"description":       "Parent registry linking operational targets to technical requirements",
			"auto_sync_enabled": true,
			"schema_version":    "1.0.0",
		},
		"imports": imports,
	}
	return mustJSON(index)
}

func renderModule(s Session, tier string, targets []OTAnswer) (string, error) {
	prefix, err := requirementPrefix(s)
	if err != nil {
		return "", err
	}
	reqs := make([]map[string]any, 0, len(targets))
	for i, t := range targets {
		otID := fmt.Sprintf("OT-%s-%03d", tier, i+1)
		reqs = append(reqs, map[string]any{
			"id":          fmt.Sprintf("%s-%s-%03d", prefix, tier, i+1),
			"title":       strings.TrimSpace(t.Title),
			"status":      "planned",
			"prd_ref":     otID,
			"criticality": tier,
			"description": strings.TrimSpace(t.Description),
			"validation": []map[string]any{{
				"type":   "manual",
				"status": "planned",
				"notes":  "Scaffolded stub: replace with a test-typed validation (ref + [REQ:" + fmt.Sprintf("%s-%s-%03d", prefix, tier, i+1) + "] tag) once the behavior exists.",
			}},
		})
	}
	module := map[string]any{
		"_metadata": map[string]any{
			"module":         tier + "-targets",
			"description":    "Requirement stubs scaffolded from the " + tier + " operational targets.",
			"priority":       tier,
			"schema_version": "1.0.0",
		},
		"requirements": reqs,
	}
	return mustJSON(module), nil
}

// requirementPrefix uses the answered prefix or derives one from the
// scenario slug (uppercased alphanumerics, letter-led, max 8 chars).
func requirementPrefix(s Session) (string, error) {
	if a, ok := s.Answers["requirement_prefix"]; ok && strings.TrimSpace(a.Text) != "" {
		p := strings.ToUpper(strings.TrimSpace(a.Text))
		if !prefixPattern.MatchString(p) {
			return "", fmt.Errorf("requirement prefix %q must match %s", p, prefixPattern)
		}
		return p, nil
	}
	var b strings.Builder
	for _, r := range strings.ToUpper(s.Scenario) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9' && b.Len() > 0) {
			b.WriteRune(r)
		}
		if b.Len() == 8 {
			break
		}
	}
	if b.Len() < 2 {
		return "", fmt.Errorf("cannot derive a requirement prefix from scenario %q; answer requirement_prefix", s.Scenario)
	}
	return b.String(), nil
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return "TODO: describe the outcome."
	}
	return s
}

func mustJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err) // static shapes; cannot fail
	}
	return string(data) + "\n"
}

const requirementsReadme = `# Requirements

Requirement modules live here, one folder per group of operational
targets. Every requirement links back to a PRD operational target via
` + "`prd_ref`" + ` and carries at least one validation entry pointing at its
proof.

- Statuses are earned, not asserted: auto-sync updates them from
  ` + "`[REQ:ID]`" + `-tagged test results on comprehensive suite runs.
- Replace scaffolded manual validation stubs with test-typed entries
  (a ` + "`ref`" + ` to the test file plus the ` + "`[REQ:ID]`" + ` tag) as behavior lands.
- Validate with ` + "`business-health validate scenario <scenario>`" + `; inspect
  traceability with ` + "`business-health matrix show <scenario>`" + `.
`
