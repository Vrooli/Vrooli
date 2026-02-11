package xrefs

import (
	"testing"
)

// testValidIDs is a set of known skill IDs for testing.
var testValidIDs = map[string]bool{
	"screaming-architecture-audit": true,
	"e2e-testing":                  true,
	"cli-steer":                    true,
	"api-steer":                    true,
	"utils-unification":            true,
	"knowledge-observatory-tools":  true,
	"feature-scope":                true,
	"platform-scope":               true,
	"some-other-skill":             true,
}

func TestExtractRefsFromContent_CLIReadSingle(t *testing.T) {
	content := "Use this: `prompt-manager skill read screaming-architecture-audit`"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].skillID != "screaming-architecture-audit" {
		t.Errorf("expected screaming-architecture-audit, got %s", refs[0].skillID)
	}
	if refs[0].refType != RefCLIRead {
		t.Errorf("expected cli-read, got %s", refs[0].refType)
	}
	if refs[0].lineNumber != 1 {
		t.Errorf("expected line 1, got %d", refs[0].lineNumber)
	}
}

func TestExtractRefsFromContent_CLIReadPlural(t *testing.T) {
	content := "`prompt-manager skills read knowledge-observatory-tools`"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].skillID != "knowledge-observatory-tools" {
		t.Errorf("expected knowledge-observatory-tools, got %s", refs[0].skillID)
	}
}

func TestExtractRefsFromContent_CLIReadMulti(t *testing.T) {
	content := "`prompt-manager skill read cli-steer api-steer utils-unification`"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d", len(refs))
	}

	ids := make(map[string]bool)
	for _, ref := range refs {
		ids[ref.skillID] = true
		if ref.refType != RefCLIRead {
			t.Errorf("expected cli-read, got %s", ref.refType)
		}
	}
	for _, expected := range []string{"cli-steer", "api-steer", "utils-unification"} {
		if !ids[expected] {
			t.Errorf("missing expected skill ID: %s", expected)
		}
	}
}

func TestExtractRefsFromContent_BoldListed(t *testing.T) {
	content := "**screaming-architecture-audit** -- Architecture alignment tool"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].refType != RefBoldListed {
		t.Errorf("expected bold-listed, got %s", refs[0].refType)
	}
}

func TestExtractRefsFromContent_BoldListedInvalid(t *testing.T) {
	// Bold text that's not a valid skill ID
	content := "**Not A Skill** -- Some description"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs for invalid bold text, got %d", len(refs))
	}
}

func TestExtractRefsFromContent_TemplateExclusion(t *testing.T) {
	// Template variables should not match
	content := "{{SKILL}} {{TARGET}} prompt-manager skill read {{skill-id}}"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs for template vars, got %d", len(refs))
	}
}

func TestExtractRefsFromContent_PlaceholderExclusion(t *testing.T) {
	content := "prompt-manager skill read <skill-id>"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs for placeholder, got %d", len(refs))
	}
}

func TestExtractRefsFromContent_PathRefRelative(t *testing.T) {
	content := "See store/skills/packs/core/feature-scope/SKILL.md for details"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].skillID != "feature-scope" {
		t.Errorf("expected feature-scope, got %s", refs[0].skillID)
	}
	if refs[0].refType != RefPathRef {
		t.Errorf("expected path-ref, got %s", refs[0].refType)
	}
}

func TestExtractRefsFromContent_PathRefDirectory(t *testing.T) {
	content := "Located at store/skills/packs/core/platform-scope/"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].skillID != "platform-scope" {
		t.Errorf("expected platform-scope, got %s", refs[0].skillID)
	}
}

func TestExtractRefsFromContent_MixedPatterns(t *testing.T) {
	content := `# Agent Skills

- **e2e-testing** -- End to end testing skill
- Architecture: ` + "`prompt-manager skill read screaming-architecture-audit`" + `
- See store/skills/packs/core/feature-scope/SKILL.md
`
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d", len(refs))
	}

	types := make(map[ReferenceType]int)
	ids := make(map[string]bool)
	for _, ref := range refs {
		types[ref.refType]++
		ids[ref.skillID] = true
	}

	if !ids["e2e-testing"] {
		t.Error("missing e2e-testing")
	}
	if !ids["screaming-architecture-audit"] {
		t.Error("missing screaming-architecture-audit")
	}
	if !ids["feature-scope"] {
		t.Error("missing feature-scope")
	}
}

func TestExtractRefsFromContent_Deduplication(t *testing.T) {
	content := `prompt-manager skill read e2e-testing
Some text
prompt-manager skill read e2e-testing`
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref (deduplicated), got %d", len(refs))
	}
	// Should keep first occurrence
	if refs[0].lineNumber != 1 {
		t.Errorf("expected line 1 (first occurrence), got %d", refs[0].lineNumber)
	}
}

func TestExtractRefsFromContent_DifferentTypesNotDeduplicated(t *testing.T) {
	// Same skill ID referenced via different patterns should appear for each type
	content := `**e2e-testing** -- Testing skill
prompt-manager skill read e2e-testing`
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 2 {
		t.Fatalf("expected 2 refs (different types), got %d", len(refs))
	}

	types := make(map[ReferenceType]bool)
	for _, ref := range refs {
		types[ref.refType] = true
	}
	if !types[RefBoldListed] {
		t.Error("missing bold-listed type")
	}
	if !types[RefCLIRead] {
		t.Error("missing cli-read type")
	}
}

func TestExtractRefsFromContent_UnknownSkillIgnored(t *testing.T) {
	content := "prompt-manager skill read nonexistent-skill"
	refs := extractRefsFromContent(content, testValidIDs)

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs for unknown skill, got %d", len(refs))
	}
}

func TestExtractRefsFromContent_EmptyContent(t *testing.T) {
	refs := extractRefsFromContent("", testValidIDs)

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs for empty content, got %d", len(refs))
	}
}

func TestIsValidSkillToken(t *testing.T) {
	tests := []struct {
		token string
		valid bool
	}{
		{"e2e-testing", true},
		{"cli-steer", true},
		{"simple", true},
		{"a1", true},
		{"Not-Valid", false},   // uppercase
		{"123-invalid", false}, // starts with number
		{"-invalid", false},    // starts with hyphen
		{"{{SKILL}}", false},   // template
		{"<skill-id>", false},  // placeholder
		{"has spaces", false},  // spaces
		{"", false},            // empty
	}

	for _, tt := range tests {
		got := isValidSkillToken(tt.token)
		if got != tt.valid {
			t.Errorf("isValidSkillToken(%q) = %v, want %v", tt.token, got, tt.valid)
		}
	}
}
