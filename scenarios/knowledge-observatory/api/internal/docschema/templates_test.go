package docschema

import (
	"strings"
	"testing"
)

func TestListTemplateDocTypes(t *testing.T) {
	types := ListTemplateDocTypes()
	if len(types) != 11 {
		t.Fatalf("expected 11 template doc types, got %d: %v", len(types), types)
	}

	// Verify sorted order.
	for i := 1; i < len(types); i++ {
		if types[i] < types[i-1] {
			t.Fatalf("list not sorted: %s before %s", types[i-1], types[i])
		}
	}
}

func TestTemplateForDocType(t *testing.T) {
	for _, dt := range ListTemplateDocTypes() {
		content, err := TemplateForDocType(dt)
		if err != nil {
			t.Fatalf("TemplateForDocType(%s) error: %v", dt, err)
		}
		if content == "" {
			t.Fatalf("TemplateForDocType(%s) returned empty content", dt)
		}
		// Templates either lead with a heading (the docs/internal pattern) or
		// with a `---` frontmatter block (the docs/perf pattern). Reject any
		// other lead.
		if !strings.HasPrefix(content, "#") && !strings.HasPrefix(content, "---") {
			t.Fatalf("TemplateForDocType(%s) content should start with a heading or frontmatter, got: %.50s", dt, content)
		}
	}
}

func TestTemplateForDocType_Unknown(t *testing.T) {
	_, err := TemplateForDocType(DocTypeReadme)
	if err == nil {
		t.Fatal("expected error for doc type without template")
	}
}

func TestHasTemplate(t *testing.T) {
	if !HasTemplate(DocTypeSeams) {
		t.Fatal("expected seams to have a template")
	}
	if HasTemplate(DocTypeReadme) {
		t.Fatal("expected readme to not have a template")
	}
	if HasTemplate(DocTypeManifest) {
		t.Fatal("expected manifest to not have a template")
	}
}

func TestTemplateFilename(t *testing.T) {
	tests := []struct {
		dt       DocType
		expected string
	}{
		{DocTypeSeams, "SEAMS.md"},
		{DocTypeProblems, "PROBLEMS.md"},
		{DocTypeProgress, "PROGRESS.md"},
		{DocTypeErrorSemantics, "ERROR-SEMANTICS.md"},
		{DocTypeExperienceAudit, "EXPERIENCE-AUDIT.md"},
		{DocTypeReadme, "README.md"}, // Fallback from ExpectedPath.
	}
	for _, tc := range tests {
		got := tc.dt.TemplateFilename()
		if got != tc.expected {
			t.Fatalf("TemplateFilename(%s) = %q, want %q", tc.dt, got, tc.expected)
		}
	}
}

func TestTemplatePurpose(t *testing.T) {
	for _, dt := range ListTemplateDocTypes() {
		purpose := TemplatePurpose(dt)
		if purpose == "" {
			t.Fatalf("TemplatePurpose(%s) returned empty string", dt)
		}
	}
	if p := TemplatePurpose(DocTypeReadme); p != "" {
		t.Fatalf("expected empty purpose for readme, got %q", p)
	}
}

func TestExpectedDocTypes(t *testing.T) {
	// Every template type should map to a known placement contract: either an
	// ExpectedPath (single-file types) or an ExpectedDir + FilenamePattern
	// (directory-pattern types like perf-audit).
	for _, dt := range ListTemplateDocTypes() {
		ep := dt.ExpectedPath()
		if ep != "" {
			if !strings.Contains(ep, "internal/") {
				t.Fatalf("fixed-path template doc type %s expected path %q should be in docs/internal/", dt, ep)
			}
			continue
		}
		// No ExpectedPath: must declare ExpectedDir + FilenamePattern.
		if dt.ExpectedDir() == "" {
			t.Fatalf("template doc type %s has neither ExpectedPath nor ExpectedDir", dt)
		}
		if dt.FilenamePattern() == nil {
			t.Fatalf("template doc type %s has ExpectedDir but no FilenamePattern", dt)
		}
	}
}
