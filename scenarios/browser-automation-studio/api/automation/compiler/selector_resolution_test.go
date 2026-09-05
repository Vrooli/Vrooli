package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// writeSelectorManifest writes a scenario-shaped selector manifest under root.
func writeSelectorManifest(t *testing.T, root string, selectors map[string]string) {
	t.Helper()
	entries := map[string]any{}
	for key, sel := range selectors {
		entries[key] = map[string]any{"selector": sel}
	}
	content, err := json.Marshal(map[string]any{"selectors": entries})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	path := filepath.Join(root, "ui", "src", "consts", "selectors.manifest.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func makeSelectorClickWorkflow(selectorRef string) *basworkflows.WorkflowDefinitionV2 {
	return &basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "navigate",
				Action: &basactions.ActionDefinition{
					Type:   basactions.ActionType_ACTION_TYPE_NAVIGATE,
					Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}},
				},
			},
			{
				Id: "click-target",
				Action: &basactions.ActionDefinition{
					Type:   basactions.ActionType_ACTION_TYPE_CLICK,
					Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: selectorRef}},
				},
			},
		},
		Edges: []*basworkflows.WorkflowEdgeV2{
			{Id: "edge-1", Source: "navigate", Target: "click-target"},
		},
	}
}

func TestCompileResolvesSelectorReferenceFromManifestRoot(t *testing.T) {
	root := t.TempDir()
	writeSelectorManifest(t, root, map[string]string{
		"dictationStudio.recordStart": "[data-testid=\"dictation-record-start\"]",
	})

	workflow := makeTestWorkflow(uuid.New(), "selector-flow", makeSelectorClickWorkflow("@selector/dictationStudio.recordStart").Nodes, makeSelectorClickWorkflow("@selector/dictationStudio.recordStart").Edges)

	plan, err := CompileWorkflowWithOptions(workflow, &CompileOptions{SelectorManifestRoot: root})
	if err != nil {
		t.Fatalf("CompileWorkflowWithOptions() error = %v", err)
	}
	if got, want := plan.Steps[1].Action.GetClick().GetSelector(), "[data-testid=\"dictation-record-start\"]"; got != want {
		t.Fatalf("resolved selector = %#v, want %q", got, want)
	}
}

// The workflow-health contract passes project_root = <scenario>/bas; the
// manifest loader must climb to the scenario root.
func TestCompileResolvesSelectorReferenceFromBasSubdirRoot(t *testing.T) {
	root := t.TempDir()
	writeSelectorManifest(t, root, map[string]string{
		"dictationStudio.recordStart": "[data-testid=\"dictation-record-start\"]",
	})
	basRoot := filepath.Join(root, "bas")
	if err := os.MkdirAll(basRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	workflow := makeTestWorkflow(uuid.New(), "selector-flow", makeSelectorClickWorkflow("@selector/dictationStudio.recordStart").Nodes, makeSelectorClickWorkflow("@selector/dictationStudio.recordStart").Edges)

	plan, err := CompileWorkflowWithOptions(workflow, &CompileOptions{SelectorManifestRoot: basRoot})
	if err != nil {
		t.Fatalf("CompileWorkflowWithOptions() error = %v", err)
	}
	if got, want := plan.Steps[1].Action.GetClick().GetSelector(), "[data-testid=\"dictation-record-start\"]"; got != want {
		t.Fatalf("resolved selector = %#v, want %q", got, want)
	}
}

func TestCompileResolvesZeroArgumentDynamicSelectorReference(t *testing.T) {
	root := t.TempDir()
	manifest := map[string]any{
		"selectors": map[string]any{},
		"dynamicSelectors": map[string]any{
			"preview.openDialog": map[string]any{
				"selectorPattern": "[role=dialog]",
				"params":          []any{},
			},
		},
	}
	content, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	path := filepath.Join(root, "ui", "src", "consts", "selectors.manifest.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	workflow := makeTestWorkflow(uuid.New(), "selector-flow", makeSelectorClickWorkflow("@selector/preview.openDialog").Nodes, makeSelectorClickWorkflow("@selector/preview.openDialog").Edges)
	plan, err := CompileWorkflowWithOptions(workflow, &CompileOptions{SelectorManifestRoot: root})
	if err != nil {
		t.Fatalf("CompileWorkflowWithOptions() error = %v", err)
	}
	if got, want := plan.Steps[1].Action.GetClick().GetSelector(), "[role=dialog]"; got != want {
		t.Fatalf("resolved selector = %#v, want %q", got, want)
	}
}

func TestCompileResolvesParameterizedDynamicSelectorReference(t *testing.T) {
	root := t.TempDir()
	manifest := map[string]any{
		"selectors": map[string]any{},
		"dynamicSelectors": map[string]any{
			"projects.cardById": map[string]any{
				"selectorPattern": "[data-project-id=\"${id}\"]",
				"params": []any{map[string]any{
					"name": "id",
					"type": "string",
				}},
			},
		},
	}
	content, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	path := filepath.Join(root, "ui", "src", "consts", "selectors.manifest.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	workflow := makeTestWorkflow(uuid.New(), "selector-flow", makeSelectorClickWorkflow("@selector/projects.cardById(id=\"${@params/projectId}\")").Nodes, makeSelectorClickWorkflow("@selector/projects.cardById(id=\"${@params/projectId}\")").Edges)
	plan, err := CompileWorkflowWithOptions(workflow, &CompileOptions{SelectorManifestRoot: root})
	if err != nil {
		t.Fatalf("CompileWorkflowWithOptions() error = %v", err)
	}
	if got, want := plan.Steps[1].Action.GetClick().GetSelector(), "[data-project-id=\"${@params/projectId}\"]"; got != want {
		t.Fatalf("resolved selector = %#v, want %q", got, want)
	}
}

func TestCompilePreservesCSSPseudoClassAfterSelectorReference(t *testing.T) {
	root := t.TempDir()
	writeSelectorManifest(t, root, map[string]string{
		"projects.card": "[data-testid=\"project-card\"]",
	})

	workflow := makeTestWorkflow(uuid.New(), "selector-flow", makeSelectorClickWorkflow("@selector/projects.card:not([data-dedupe=\"restored\"])").Nodes, makeSelectorClickWorkflow("@selector/projects.card:not([data-dedupe=\"restored\"])").Edges)
	plan, err := CompileWorkflowWithOptions(workflow, &CompileOptions{SelectorManifestRoot: root})
	if err != nil {
		t.Fatalf("CompileWorkflowWithOptions() error = %v", err)
	}
	if got, want := plan.Steps[1].Action.GetClick().GetSelector(), "[data-testid=\"project-card\"]:not([data-dedupe=\"restored\"])"; got != want {
		t.Fatalf("resolved selector = %#v, want %q", got, want)
	}
}

func TestCompileResolvesSelectorReferenceInsideEvaluateExpression(t *testing.T) {
	root := t.TempDir()
	writeSelectorManifest(t, root, map[string]string{
		"projects.card": "[data-testid=\"project-card\"]",
	})
	workflow := &basworkflows.WorkflowDefinitionV2{Nodes: []*basworkflows.WorkflowNodeV2{{
		Id: "evaluate-project-cards",
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_EVALUATE,
			Params: &basactions.ActionDefinition_Evaluate{Evaluate: &basactions.EvaluateParams{
				Expression: "document.querySelectorAll('@selector/projects.card').length",
			}},
		},
	}}}

	plan, err := CompileWorkflowWithOptions(makeTestWorkflow(uuid.New(), "expression-selector", workflow.Nodes, nil), &CompileOptions{SelectorManifestRoot: root})
	if err != nil {
		t.Fatalf("CompileWorkflowWithOptions() error = %v", err)
	}
	if got, want := plan.Steps[0].Action.GetEvaluate().GetExpression(), "document.querySelectorAll('[data-testid=\"project-card\"]').length"; got != want {
		t.Fatalf("resolved expression = %#v, want %q", got, want)
	}
}

// An unresolved @selector/ reference must fail compilation with a diagnostic
// naming the token, instead of forwarding the literal token to the driver
// (which used to surface as an opaque runtime "Selector not found: unknown").
func TestCompileFailsOnUnresolvedSelectorReference(t *testing.T) {
	root := t.TempDir()
	writeSelectorManifest(t, root, map[string]string{
		"dictationStudio.someOtherKey": "[data-testid=\"other\"]",
	})

	workflow := makeTestWorkflow(uuid.New(), "selector-flow", makeSelectorClickWorkflow("@selector/dictationStudio.recordStart").Nodes, makeSelectorClickWorkflow("@selector/dictationStudio.recordStart").Edges)

	_, err := CompileWorkflowWithOptions(workflow, &CompileOptions{SelectorManifestRoot: root})
	if err == nil {
		t.Fatal("CompileWorkflowWithOptions() error = nil, want unresolved-selector error")
	}
	if !strings.Contains(err.Error(), "@selector/dictationStudio.recordStart") {
		t.Fatalf("error %q does not name the unresolved token", err.Error())
	}
	if !strings.Contains(err.Error(), "click-target") {
		t.Fatalf("error %q does not name the failing node", err.Error())
	}
}
