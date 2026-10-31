package docschema

import (
	"strings"
	"testing"
)

const validPerfAuditFrontmatter = `---
date: 2026-05-03
scenario: swarm-manager
interactions:
  - command-post-sidebar-resize
  - backlog-tab-scroll
traces:
  before: /tmp/swarm-manager/perf/trace.before.json
  after: /tmp/swarm-manager/perf/trace.after.json
status: fixed
related_skill_run: scenario-performance-audit
---

# Audit body

## Per-component aggregation

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---|---|---|---|
| BacklogTab | 60 | 114.0 | 1900 | 8000 |
`

func TestValidateFrontmatter_Valid(t *testing.T) {
	issues := ValidateFrontmatter(validPerfAuditFrontmatter, PerfAuditFrontmatterSchema)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d: %+v", len(issues), issues)
	}
}

func TestValidateFrontmatter_MissingFrontmatter(t *testing.T) {
	issues := ValidateFrontmatter("# just a body, no frontmatter\n", PerfAuditFrontmatterSchema)
	if len(issues) == 0 {
		t.Fatal("expected missing-frontmatter issue")
	}
	if issues[0].Code != "perf-audit:missing-frontmatter" {
		t.Fatalf("expected missing-frontmatter, got %q", issues[0].Code)
	}
}

func TestValidateFrontmatter_MissingRequiredKey(t *testing.T) {
	content := strings.Replace(validPerfAuditFrontmatter, "scenario: swarm-manager\n", "", 1)
	issues := ValidateFrontmatter(content, PerfAuditFrontmatterSchema)
	found := false
	for _, i := range issues {
		if i.Code == "perf-audit:missing-key" && i.Field == "scenario" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected missing-key issue for scenario, got %+v", issues)
	}
}

func TestValidateFrontmatter_InvalidEnum(t *testing.T) {
	content := strings.Replace(validPerfAuditFrontmatter, "status: fixed", "status: pending", 1)
	issues := ValidateFrontmatter(content, PerfAuditFrontmatterSchema)
	found := false
	for _, i := range issues {
		if i.Code == "perf-audit:invalid-enum" && i.Field == "status" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid-enum for status, got %+v", issues)
	}
}

func TestValidateFrontmatter_InvalidDate(t *testing.T) {
	content := strings.Replace(validPerfAuditFrontmatter, "date: 2026-05-03", "date: not-a-date", 1)
	issues := ValidateFrontmatter(content, PerfAuditFrontmatterSchema)
	found := false
	for _, i := range issues {
		if i.Code == "perf-audit:invalid-date" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid-date, got %+v", issues)
	}
}

func TestValidateFrontmatter_EmptyList(t *testing.T) {
	content := strings.Replace(
		validPerfAuditFrontmatter,
		"interactions:\n  - command-post-sidebar-resize\n  - backlog-tab-scroll\n",
		"interactions:\n",
		1,
	)
	issues := ValidateFrontmatter(content, PerfAuditFrontmatterSchema)
	found := false
	for _, i := range issues {
		if i.Code == "perf-audit:empty-list" && i.Field == "interactions" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected empty-list for interactions, got %+v", issues)
	}
}

func TestExtractFrontmatter_NoFrontmatter(t *testing.T) {
	fm, body := extractFrontmatter("# heading\n\nsome content\n")
	if fm.present {
		t.Fatal("expected fm.present=false")
	}
	if body == "" {
		t.Fatal("expected body to be returned unchanged")
	}
}

func TestExtractFrontmatter_StripsBOM(t *testing.T) {
	content := "\xef\xbb\xbf---\ndate: 2026-05-03\nscenario: x\nstatus: open\ninteractions:\n  - a\n---\n\nbody\n"
	fm, _ := extractFrontmatter(content)
	if !fm.present {
		t.Fatal("expected fm.present=true after BOM strip")
	}
	if fm.scalars["date"] != "2026-05-03" {
		t.Fatalf("expected date=2026-05-03, got %q", fm.scalars["date"])
	}
}

func TestPerfAuditFilenamePattern(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"2026-05-03-sidebar-resize.md", true},
		{"2026-05-03-MixedCase.md", false},      // uppercase rejected
		{"05-03-2026-sidebar-resize.md", false}, // wrong date order
		{"README.md", false},
		{"2026-05-03.md", false},              // missing slug
		{"2026-05-03-graph-perf.v2.md", true}, // dotted suffix allowed
	}
	for _, tc := range cases {
		if got := perfAuditFilenamePattern.MatchString(tc.name); got != tc.want {
			t.Errorf("pattern.MatchString(%q)=%v, want %v", tc.name, got, tc.want)
		}
	}
}
