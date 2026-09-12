package templateengine

import (
	"path/filepath"
	"testing"

	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

// templatePRD mirrors the shape of the react-vite PRD.md scaffold: placeholder
// lines the generator substitutes plus fixed prose that survives generation.
const templatePRD = `# {{SCENARIO_DISPLAY_NAME}} PRD

## Capability
[Summarize the permanent capability this scenario adds]

## Outcomes
### [Outcome title]
- Target: [define an operational target]

## Users
[Describe the primary users]
`

// generatedPRD is what a freshly generated, UNEDITED scenario ships: the template
// with placeholders substituted. It must count as zero adapted content.
const generatedPRD = `# Probe Widget PRD

## Capability
[Summarize the permanent capability this scenario adds]

## Outcomes
### [Outcome title]
- Target: [define an operational target]

## Users
[Describe the primary users]
`

// gamedPRD deletes only the text_absent marker lines without adding any real
// content — the loophole content_adapted must close.
const gamedPRD = `# Probe Widget PRD

## Capability

## Outcomes
###

## Users
[Describe the primary users]
`

// orientedPRD is genuine adaptation: real scenario-specific prose replaces the
// scaffold.
const orientedPRD = `# Probe Widget PRD

## Capability
Probe Widget catalogs telemetry probes and renders live health rollups for operators.

## Outcomes
### Probes stay green
- Target: 99.9% of probes report within a 30s freshness window.
### Operators triage fast
- Target: median time-to-acknowledge under two minutes.

## Users
On-call reliability engineers monitoring fleet probe coverage.
`

func writeTemplateAndScenario(t *testing.T, templateBody, scenarioBody string) orientationEval {
	t.Helper()
	templateRoot := t.TempDir()
	scenarioRoot := t.TempDir()
	if templateBody != "" {
		mustWrite(t, filepath.Join(templateRoot, "PRD.md"), templateBody)
	}
	if scenarioBody != "" {
		mustWrite(t, filepath.Join(scenarioRoot, "PRD.md"), scenarioBody)
	}
	return orientationEval{scenarioRoot: scenarioRoot, templateSourceRoot: templateRoot}
}

func TestTemplateNetNewContentLinesPlaceholderAware(t *testing.T) {
	ev := writeTemplateAndScenario(t, templatePRD, generatedPRD)
	got, err := templateNetNewContentLines(filepath.Join(ev.templateSourceRoot, "PRD.md"), filepath.Join(ev.scenarioRoot, "PRD.md"))
	if err != nil {
		t.Fatalf("netNew: %v", err)
	}
	if got != 0 {
		t.Fatalf("freshly generated scenario should have 0 net-new content lines, got %d", got)
	}

	ev = writeTemplateAndScenario(t, templatePRD, orientedPRD)
	got, err = templateNetNewContentLines(filepath.Join(ev.templateSourceRoot, "PRD.md"), filepath.Join(ev.scenarioRoot, "PRD.md"))
	if err != nil {
		t.Fatalf("netNew: %v", err)
	}
	if got < 3 {
		t.Fatalf("oriented scenario should have plenty of net-new content, got %d", got)
	}
}

func TestContentAdaptedThreshold(t *testing.T) {
	if got := contentAdaptedThreshold("PRD.md", 0); got != 3 {
		t.Fatalf("prose threshold = %d, want 3", got)
	}
	if got := contentAdaptedThreshold("requirements/index.json", 0); got != 1 {
		t.Fatalf("json threshold = %d, want 1", got)
	}
	if got := contentAdaptedThreshold("PRD.md", 7); got != 7 {
		t.Fatalf("override threshold = %d, want 7", got)
	}
}

func TestEvaluateContentAdaptedFailsOnBoilerplate(t *testing.T) {
	ev := writeTemplateAndScenario(t, templatePRD, generatedPRD)
	report := templatecontracts.OrientationCheckReport{Kind: "content_adapted"}
	evaluateOrientationContentAdapted(&report, ev, templatecontracts.TemplateOrientationCheck{Kind: "content_adapted", Path: "PRD.md"})
	if report.Passed || report.Skipped {
		t.Fatalf("boilerplate PRD must fail content_adapted, got %#v", report)
	}
}

func TestEvaluateContentAdaptedFailsOnMarkerDeletionGaming(t *testing.T) {
	ev := writeTemplateAndScenario(t, templatePRD, gamedPRD)
	report := templatecontracts.OrientationCheckReport{Kind: "content_adapted"}
	evaluateOrientationContentAdapted(&report, ev, templatecontracts.TemplateOrientationCheck{Kind: "content_adapted", Path: "PRD.md"})
	if report.Passed {
		t.Fatalf("deleting marker lines without adding content must not pass content_adapted, got %#v", report)
	}
}

func TestEvaluateContentAdaptedPassesOnRealContent(t *testing.T) {
	ev := writeTemplateAndScenario(t, templatePRD, orientedPRD)
	report := templatecontracts.OrientationCheckReport{Kind: "content_adapted"}
	evaluateOrientationContentAdapted(&report, ev, templatecontracts.TemplateOrientationCheck{Kind: "content_adapted", Path: "PRD.md"})
	if !report.Passed {
		t.Fatalf("genuinely oriented PRD must pass content_adapted, got %#v", report)
	}
}

func TestEvaluateContentAdaptedThinJSONRegistry(t *testing.T) {
	templateRoot := t.TempDir()
	scenarioRoot := t.TempDir()
	// Thin JSON registry oriented only by re-pointing a single import value.
	mustWrite(t, filepath.Join(templateRoot, "requirements", "index.json"), "{\n  \"imports\": [\n    \"01-foundation/module.json\"\n  ]\n}\n")
	freshEV := orientationEval{scenarioRoot: scenarioRoot, templateSourceRoot: templateRoot}

	// Fresh: identical import -> zero adapted -> fail.
	mustWrite(t, filepath.Join(scenarioRoot, "requirements", "index.json"), "{\n  \"imports\": [\n    \"01-foundation/module.json\"\n  ]\n}\n")
	fresh := templatecontracts.OrientationCheckReport{Kind: "content_adapted"}
	evaluateOrientationContentAdapted(&fresh, freshEV, templatecontracts.TemplateOrientationCheck{Kind: "content_adapted", Path: "requirements/index.json"})
	if fresh.Passed {
		t.Fatalf("fresh thin registry must fail, got %#v", fresh)
	}

	// Oriented: one real re-pointed import line -> passes at json threshold of 1.
	orientedRoot := t.TempDir()
	mustWrite(t, filepath.Join(orientedRoot, "requirements", "index.json"), "{\n  \"imports\": [\n    \"01-probe-telemetry/module.json\"\n  ]\n}\n")
	oriented := templatecontracts.OrientationCheckReport{Kind: "content_adapted"}
	evaluateOrientationContentAdapted(&oriented, orientationEval{scenarioRoot: orientedRoot, templateSourceRoot: templateRoot}, templatecontracts.TemplateOrientationCheck{Kind: "content_adapted", Path: "requirements/index.json"})
	if !oriented.Passed {
		t.Fatalf("re-pointed thin registry must pass, got %#v", oriented)
	}
}

func TestEvaluateContentAdaptedSkipsWhenTemplateUnavailable(t *testing.T) {
	// No generation template recorded.
	scenarioRoot := t.TempDir()
	mustWrite(t, filepath.Join(scenarioRoot, "PRD.md"), generatedPRD)
	report := templatecontracts.OrientationCheckReport{Kind: "content_adapted"}
	evaluateOrientationContentAdapted(&report, orientationEval{scenarioRoot: scenarioRoot}, templatecontracts.TemplateOrientationCheck{Kind: "content_adapted", Path: "PRD.md"})
	if !report.Passed || !report.Skipped {
		t.Fatalf("missing template source must skip (pass), got %#v", report)
	}

	// Template exists but does not ship this file (e.g. DESIGN.md).
	ev := writeTemplateAndScenario(t, templatePRD, generatedPRD)
	mustWrite(t, filepath.Join(ev.scenarioRoot, "DESIGN.md"), "anything\n")
	designReport := templatecontracts.OrientationCheckReport{Kind: "content_adapted"}
	evaluateOrientationContentAdapted(&designReport, ev, templatecontracts.TemplateOrientationCheck{Kind: "content_adapted", Path: "DESIGN.md"})
	if !designReport.Passed || !designReport.Skipped {
		t.Fatalf("template-less file must skip (pass), got %#v", designReport)
	}
}

func TestDeriveContentAdaptedChecks(t *testing.T) {
	ev := writeTemplateAndScenario(t, templatePRD, generatedPRD)
	// DESIGN.md is intentionally NOT shipped by the template.
	declared := []templatecontracts.TemplateOrientationCheck{
		{Kind: "file_exists", Path: "PRD.md"},
		{Kind: "text_absent", Path: "PRD.md", Text: "[Outcome title]"},
		{Kind: "text_absent", Path: "DESIGN.md", Text: "ORIENTATION-TODO"},
	}
	derived := deriveContentAdaptedChecks(declared, ev)
	if len(derived) != 1 || derived[0].Kind != "content_adapted" || derived[0].Path != "PRD.md" {
		t.Fatalf("expected one derived content_adapted for PRD.md only, got %#v", derived)
	}

	// No template source -> no derivation.
	if got := deriveContentAdaptedChecks(declared, orientationEval{scenarioRoot: ev.scenarioRoot}); got != nil {
		t.Fatalf("no template source must derive nothing, got %#v", got)
	}

	// Does not duplicate an explicitly declared content_adapted for the same path.
	withExplicit := append(declared, templatecontracts.TemplateOrientationCheck{Kind: "content_adapted", Path: "PRD.md"})
	if got := deriveContentAdaptedChecks(withExplicit, ev); len(got) != 0 {
		t.Fatalf("explicit content_adapted should suppress derivation, got %#v", got)
	}
}

func TestEvaluateOrientationStepBlocksGamingButPassesRealContent(t *testing.T) {
	step := templatecontracts.TemplateOrientationStep{
		ID: "charter",
		Checks: []templatecontracts.TemplateOrientationCheck{
			{Kind: "file_exists", Path: "PRD.md"},
			{Kind: "text_absent", Path: "PRD.md", Text: "[Summarize the permanent capability this scenario adds]"},
			{Kind: "text_absent", Path: "PRD.md", Text: "[Outcome title]"},
		},
	}
	deps := HandlerDeps[struct{}]{}

	// Gaming: marker lines deleted (text_absent passes) but no real content added.
	gamed := writeTemplateAndScenario(t, templatePRD, gamedPRD)
	if report := evaluateOrientationStep(deps, struct{}{}, gamed, step); report.Complete {
		t.Fatalf("gate must stay incomplete when markers are deleted but no content added: %#v", report.Checks)
	}

	// Real adaptation: markers gone AND real content present.
	oriented := writeTemplateAndScenario(t, templatePRD, orientedPRD)
	if report := evaluateOrientationStep(deps, struct{}{}, oriented, step); !report.Complete {
		t.Fatalf("gate must complete for genuinely oriented content: %#v", report.Checks)
	}

	// No template source (e.g. hand-built scenario): derived check skips, so the
	// step's completeness depends only on the declared checks (markers absent).
	handBuilt := orientationEval{scenarioRoot: oriented.scenarioRoot}
	if report := evaluateOrientationStep(deps, struct{}{}, handBuilt, step); !report.Complete {
		t.Fatalf("hand-built scenario without template source must not be blocked by derivation: %#v", report.Checks)
	}
}
