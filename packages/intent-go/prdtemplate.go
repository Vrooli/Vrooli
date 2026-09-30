package intent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// This file is the canonical PRD template contract: the section model, the
// PRD document extractor, and the pure template checks. It absorbs the
// V2 template engine absorbed from the retired PRD control-tower scenario
// (prd_template_validator_v2.go) — the heading strings (emoji + en-dash
// included) are the parse contract every conformant PRD.md matches
// byte-for-byte, and the wizard derives its question model from the same
// definitions so scaffolder and validator can never disagree.

// ContentRule is one content-level expectation inside a section.
type ContentRule struct {
	// Kind: "contains" (substring), "checklist" (regex `- \[[ x]\]`), or
	// "regex" (arbitrary pattern).
	Kind        string
	Pattern     string
	Description string
	// Required escalates a miss from warning to error.
	Required bool
}

// TemplateSection is one canonical section (## level 2, ### level 3).
type TemplateSection struct {
	Title       string
	Level       int
	Required    bool
	Description string
	Subsections []TemplateSection
	Rules       []ContentRule
}

// DefaultPRDTemplate returns the canonical v2 template. Heading strings are
// exact (emoji, en-dash) — they are the machine contract, not decoration.
func DefaultPRDTemplate() []TemplateSection {
	return []TemplateSection{
		{
			Title: "🎯 Overview", Level: 2, Required: true,
			Description: "Purpose, users, deployment surfaces",
			Rules: []ContentRule{
				{Kind: "contains", Pattern: "Purpose", Description: "Outline the permanent capability", Required: true},
			},
		},
		{
			Title: "🎯 Operational Targets", Level: 2, Required: true,
			Description: "Outcome checklists grouped into tiers",
			Subsections: []TemplateSection{
				{
					Title: "🔴 P0 – Must ship for viability", Level: 3, Required: true,
					Description: "Non-negotiable launch targets",
					Rules: []ContentRule{
						{Kind: "checklist", Pattern: `- \[[ x]\]`, Description: "Use checklist format for each target", Required: true},
					},
				},
				{
					Title: "🟠 P1 – Should have post-launch", Level: 3, Required: true,
					Description: "Important enhancements",
					Rules: []ContentRule{
						{Kind: "checklist", Pattern: `- \[[ x]\]`, Description: "Use checklist format", Required: false},
					},
				},
				{
					Title: "🟢 P2 – Future / expansion", Level: 3, Required: true,
					Description: "Aspirational follow-ups",
					Rules: []ContentRule{
						{Kind: "checklist", Pattern: `- \[[ x]\]`, Description: "Use checklist format", Required: false},
					},
				},
			},
		},
		{
			Title: "🧱 Tech Direction Snapshot", Level: 2, Required: true,
			Description: "Preferred stacks, data, integrations, non-goals",
			Rules: []ContentRule{
				{Kind: "contains", Pattern: "Preferred", Description: "List preferred stacks or approaches", Required: true},
			},
		},
		{
			Title: "🤝 Dependencies & Launch Plan", Level: 2, Required: true,
			Description: "Resources, scenario dependencies, risks, sequencing",
		},
		{
			Title: "🎨 UX & Branding", Level: 2, Required: true,
			Description: "Look/feel, accessibility, voice",
			Rules: []ContentRule{
				{Kind: "contains", Pattern: "Accessibility", Description: "State accessibility expectations", Required: false},
			},
		},
		{
			Title: "📎 Appendix", Level: 2, Required: false,
			Description: "Optional references",
		},
	}
}

// PRDSection is one extracted heading with its body content.
type PRDSection struct {
	Title   string
	Level   int
	Content string
	Line    int
}

// OperationalTarget is one structured OT checklist line.
type OperationalTarget struct {
	// RawID is the ID token exactly as written.
	RawID string
	// ID is the canonicalized form (upper-cased, zero-padded) when the raw
	// token is recognizably an OT id; equal to RawID otherwise.
	ID string
	// Tier is "P0" | "P1" | "P2" (from the section the line sits under).
	Tier    string
	Checked bool
	Title   string
	// Description is the third pipe field, when present.
	Description string
	Line        int
}

// PRDDocument is the extracted shape of one PRD.md.
type PRDDocument struct {
	Path     string
	Present  bool
	Sections []PRDSection
	Targets  []OperationalTarget
}

var (
	sectionHeadingPattern = regexp.MustCompile(`^(#{2,3})\s+(.+)$`)
	otLinePattern         = regexp.MustCompile(`^-\s\[( |x|X)\]\s*(\S+)\s*(?:\|(.*))?$`)
	canonicalOTIDPattern  = regexp.MustCompile(`^OT-P[0-2]-\d{3}$`)
	looseOTIDPattern      = regexp.MustCompile(`(?i)^OT-P([0-2])-0*(\d{1,3})$`)
	emojiPattern          = regexp.MustCompile(`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{FE00}-\x{FE0F}\x{1F900}-\x{1F9FF}\x{1F1E0}-\x{1F1FF}]`)
	whitespacePattern     = regexp.MustCompile(`\s+`)
	tierPattern           = regexp.MustCompile(`\bP([0-2])\b`)
)

// NormalizeSectionTitle strips emoji and collapses whitespace so headings
// compare on their textual identity (matches the legacy validator's
// normalization; a missing emoji still matches, a renamed section does not).
func NormalizeSectionTitle(title string) string {
	normalized := emojiPattern.ReplaceAllString(title, "")
	normalized = strings.TrimSpace(normalized)
	normalized = whitespacePattern.ReplaceAllString(normalized, " ")
	return strings.ToLower(normalized)
}

// ExtractPRDDocument parses PRD.md into sections and structured operational
// targets. A missing file yields Present=false with no error — presence is
// the caller's finding to make.
func ExtractPRDDocument(scenarioRoot string) (PRDDocument, error) {
	path := filepath.Join(scenarioRoot, "PRD.md")
	doc := PRDDocument{Path: "PRD.md"}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return doc, nil
		}
		return doc, err
	}
	doc.Present = true

	lines := strings.Split(string(data), "\n")
	var current *PRDSection
	var content strings.Builder
	currentTier := ""

	flush := func() {
		if current != nil {
			current.Content = strings.TrimRight(content.String(), "\n")
			doc.Sections = append(doc.Sections, *current)
			content.Reset()
		}
	}

	for i, line := range lines {
		if m := sectionHeadingPattern.FindStringSubmatch(line); len(m) == 3 {
			flush()
			current = &PRDSection{
				Title: strings.TrimSpace(m[2]),
				Level: len(m[1]),
				Line:  i + 1,
			}
			if tier := tierPattern.FindStringSubmatch(current.Title); len(tier) == 2 && current.Level == 3 {
				currentTier = "P" + tier[1]
			} else if current.Level == 2 {
				currentTier = ""
			}
			continue
		}
		if current != nil {
			content.WriteString(line)
			content.WriteString("\n")
		}
		if m := otLinePattern.FindStringSubmatch(strings.TrimSpace(line)); len(m) == 4 {
			rawID := strings.TrimSpace(m[2])
			if !strings.HasPrefix(strings.ToUpper(rawID), "OT-") {
				continue
			}
			ot := OperationalTarget{
				RawID:   rawID,
				ID:      CanonicalOTID(rawID),
				Tier:    currentTier,
				Checked: strings.EqualFold(m[1], "x"),
				Line:    i + 1,
			}
			if rest := m[3]; rest != "" {
				fields := strings.SplitN(rest, "|", 2)
				ot.Title = strings.TrimSpace(fields[0])
				if len(fields) == 2 {
					ot.Description = strings.TrimSpace(fields[1])
				}
			}
			doc.Targets = append(doc.Targets, ot)
		}
	}
	flush()
	return doc, nil
}

// CanonicalOTID normalizes a recognizable OT id token to the canonical
// `OT-P<tier>-NNN` form (upper case, 3-digit zero padding). Unrecognizable
// tokens return unchanged.
func CanonicalOTID(raw string) string {
	m := looseOTIDPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(m) != 3 {
		return strings.TrimSpace(raw)
	}
	return fmt.Sprintf("OT-P%s-%03d", m[1], atoiSafe(m[2]))
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
}

// sectionKey builds the identity a section matches under.
func sectionKey(level int, title string) string {
	return fmt.Sprintf("%d:%s", level, NormalizeSectionTitle(title))
}

func sectionIndex(doc PRDDocument) map[string]PRDSection {
	out := make(map[string]PRDSection, len(doc.Sections))
	for _, s := range doc.Sections {
		out[sectionKey(s.Level, s.Title)] = s
	}
	return out
}

// CheckTemplateSections reports required sections/subsections missing from
// the document (code prd_template_sections, error).
func CheckTemplateSections(doc PRDDocument, template []TemplateSection) []Finding {
	if !doc.Present {
		return nil
	}
	found := sectionIndex(doc)
	var findings []Finding
	var walk func(parent string, sections []TemplateSection)
	walk = func(parent string, sections []TemplateSection) {
		for _, s := range sections {
			_, ok := found[sectionKey(s.Level, s.Title)]
			label := s.Title
			if parent != "" {
				label = parent + " > " + s.Title
			}
			if s.Required && !ok {
				findings = append(findings, Finding{
					Code:       CodeTemplateSections,
					Severity:   "error",
					Message:    fmt.Sprintf("Required section '%s' is missing", label),
					Suggestion: fmt.Sprintf("Add the heading: %s %s", strings.Repeat("#", s.Level), s.Title),
					Locations:  []string{doc.Path},
					ClaimID:    NormalizeSectionTitle(s.Title),
					Provenance: "prd-template",
				})
			}
			if ok || parent == "" {
				// Subsections of a present parent are checked; subsections of a
				// missing top-level parent are also reported (the parent finding
				// alone would hide how much is absent).
				walk(label, s.Subsections)
			}
		}
	}
	walk("", template)
	return findings
}

// CheckUnexpectedSections reports headings outside the canonical vocabulary
// (code prd_template_unexpected_sections, info).
func CheckUnexpectedSections(doc PRDDocument, template []TemplateSection) []Finding {
	if !doc.Present {
		return nil
	}
	valid := make(map[string]struct{})
	var collect func(sections []TemplateSection)
	collect = func(sections []TemplateSection) {
		for _, s := range sections {
			valid[sectionKey(s.Level, s.Title)] = struct{}{}
			collect(s.Subsections)
		}
	}
	collect(template)

	var unexpected []PRDSection
	for _, s := range doc.Sections {
		if _, ok := valid[sectionKey(s.Level, s.Title)]; !ok {
			unexpected = append(unexpected, s)
		}
	}
	sort.Slice(unexpected, func(i, j int) bool { return unexpected[i].Line < unexpected[j].Line })

	findings := make([]Finding, 0, len(unexpected))
	for _, s := range unexpected {
		findings = append(findings, Finding{
			Code:       CodeTemplateUnexpectedSections,
			Severity:   "info",
			Message:    fmt.Sprintf("Heading '%s' is not part of the canonical PRD template", s.Title),
			Suggestion: "Rename it to the canonical heading it stands for, or move the content under 📎 Appendix.",
			Locations:  []string{fmt.Sprintf("%s:%d", doc.Path, s.Line)},
			ClaimID:    NormalizeSectionTitle(s.Title),
			Provenance: "prd-template",
		})
	}
	return findings
}

// CheckTemplateContent reports content-level issues inside present sections
// (code prd_template_content; severity per rule: required misses are errors,
// optional misses and empty sections are warnings).
func CheckTemplateContent(doc PRDDocument, template []TemplateSection) []Finding {
	if !doc.Present {
		return nil
	}
	found := sectionIndex(doc)
	var findings []Finding
	var walk func(sections []TemplateSection)
	walk = func(sections []TemplateSection) {
		for _, s := range sections {
			section, ok := found[sectionKey(s.Level, s.Title)]
			if ok {
				findings = append(findings, checkSectionContent(doc.Path, s, section)...)
			}
			walk(s.Subsections)
		}
	}
	walk(template)
	return findings
}

func checkSectionContent(path string, spec TemplateSection, section PRDSection) []Finding {
	var findings []Finding
	content := strings.TrimSpace(section.Content)
	if content == "" {
		// Parent sections holding structured subsections legitimately have no
		// direct body (the parser attributes text to the ### subsections).
		if len(spec.Subsections) > 0 {
			return nil
		}
		return []Finding{{
			Code:       CodeTemplateContent,
			Severity:   "warning",
			Message:    fmt.Sprintf("Section '%s' appears empty (empty_section)", spec.Title),
			Suggestion: "Add descriptive content.",
			Locations:  []string{fmt.Sprintf("%s:%d", path, section.Line)},
			ClaimID:    NormalizeSectionTitle(spec.Title),
			Provenance: "prd-template",
		}}
	}
	for _, rule := range spec.Rules {
		miss := false
		issueType := "missing_content"
		switch rule.Kind {
		case "contains":
			miss = !strings.Contains(section.Content, rule.Pattern)
		case "checklist":
			miss = !regexp.MustCompile(rule.Pattern).MatchString(section.Content)
			issueType = "invalid_checklist"
		case "regex":
			miss = !regexp.MustCompile(rule.Pattern).MatchString(section.Content)
		}
		if !miss {
			continue
		}
		severity := "warning"
		if rule.Required {
			severity = "error"
		}
		findings = append(findings, Finding{
			Code:       CodeTemplateContent,
			Severity:   severity,
			Message:    fmt.Sprintf("Section '%s': %s (%s)", spec.Title, rule.Description, issueType),
			Suggestion: contentSuggestion(rule, issueType),
			Locations:  []string{fmt.Sprintf("%s:%d", path, section.Line)},
			ClaimID:    NormalizeSectionTitle(spec.Title),
			Provenance: "prd-template",
		})
	}
	return findings
}

func contentSuggestion(rule ContentRule, issueType string) string {
	if issueType == "invalid_checklist" {
		return "Use checklist format: - [ ] OT-P0-001 | Title | one-line description"
	}
	return fmt.Sprintf("Include: %s", rule.Pattern)
}

// CheckOTIDFormat reports operational-target lines whose ID deviates from
// the canonical `OT-P<tier>-NNN` form, including tier/section mismatches
// (code prd_ot_id_format, warning; deterministically fixable).
func CheckOTIDFormat(doc PRDDocument) []Finding {
	if !doc.Present {
		return nil
	}
	var findings []Finding
	for _, ot := range doc.Targets {
		var problem string
		switch {
		case !canonicalOTIDPattern.MatchString(ot.RawID):
			if ot.ID != ot.RawID {
				problem = fmt.Sprintf("ID %q is not in canonical form (want %q)", ot.RawID, ot.ID)
			} else {
				problem = fmt.Sprintf("ID %q does not match the canonical OT-P<tier>-NNN format", ot.RawID)
			}
		case ot.Tier != "" && !strings.HasPrefix(ot.ID, "OT-"+ot.Tier+"-"):
			problem = fmt.Sprintf("ID %q sits under the %s tier section but declares a different tier", ot.RawID, ot.Tier)
		}
		if problem == "" {
			continue
		}
		findings = append(findings, Finding{
			Code:       CodeOTIDFormat,
			Severity:   "warning",
			Message:    "Operational target " + problem,
			Suggestion: "Normalize the ID to OT-P<tier>-NNN (zero-padded); update matching prd_ref values in requirements/.",
			Locations:  []string{fmt.Sprintf("%s:%d", doc.Path, ot.Line)},
			ClaimID:    ot.ID,
			Provenance: "prd-template",
		})
	}
	return findings
}
