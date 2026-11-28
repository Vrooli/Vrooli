package agentaccess

import (
	"encoding/json"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// This joins the task family exercised by Browser Use and Playwright MCP to
// the trace compiler. The adapters must not merely agree on accepted tasks;
// the common accepted actions must also produce a stable, replayable BAS draft.
func TestSharedHarnessConformanceTaskFamilyCompilesToStableDraft(t *testing.T) {
	type step struct {
		Task struct {
			Operation string `json:"operation"`
			Ref       string `json:"ref"`
			Value     string `json:"value"`
			URL       string `json:"url"`
		} `json:"task"`
	}
	var vector struct {
		SessionID string `json:"session_id"`
		Steps     []step `json:"steps"`
	}
	data, err := os.ReadFile("../../../testdata/harness-conformance/v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &vector); err != nil {
		t.Fatal(err)
	}
	if vector.SessionID == "" || len(vector.Steps) == 0 {
		t.Fatal("shared adapter vector has no session or steps")
	}

	base := time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)
	trace := Trace{ID: "shared-harness-vector", SessionID: vector.SessionID, PermissionRef: "fixture-permission"}
	sequence := uint64(0)
	currentOrigin := "https://allowed.example"
	appendEvent := func(kind, permission, effect, target string, action *actionsv1.ActionDefinition) {
		sequence++
		trace.Events = append(trace.Events, TraceEvent{
			Sequence: sequence, SessionID: vector.SessionID, Kind: kind,
			PermissionClass: permission, EffectClass: effect, URL: currentOrigin,
			TargetOrigin: target, OccurredAt: base.Add(time.Duration(sequence) * time.Millisecond),
			Instruction: instruction(action),
			Outcome:     outcome(action),
		})
	}
	for _, item := range vector.Steps {
		switch item.Task.Operation {
		case "read":
			appendEvent("observation", "observe", "none", "", nil)
		case "click":
			selector := selectorForRef(t, item.Task.Ref)
			appendEvent("action", "act", "ui", "", &actionsv1.ActionDefinition{
				Type:   actionsv1.ActionType_ACTION_TYPE_CLICK,
				Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: selector}},
			})
			appendEvent("observation", "observe", "none", "", nil)
		case "fill":
			selector := selectorForRef(t, item.Task.Ref)
			appendEvent("action", "act", "ui", "", &actionsv1.ActionDefinition{
				Type:   actionsv1.ActionType_ACTION_TYPE_INPUT,
				Params: &actionsv1.ActionDefinition_Input{Input: &actionsv1.InputParams{Selector: selector, Value: item.Task.Value}},
			})
			appendEvent("observation", "observe", "none", "", nil)
		case "navigate":
			parsed, err := url.Parse(item.Task.URL)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				t.Fatalf("shared vector navigation URL is invalid: %q", item.Task.URL)
			}
			targetOrigin := parsed.Scheme + "://" + parsed.Host
			appendEvent("action", "act", "navigation", targetOrigin, &actionsv1.ActionDefinition{
				Type:   actionsv1.ActionType_ACTION_TYPE_NAVIGATE,
				Params: &actionsv1.ActionDefinition_Navigate{Navigate: &actionsv1.NavigateParams{Url: item.Task.URL}},
			})
			currentOrigin = targetOrigin
			appendEvent("observation", "observe", "none", "", nil)
		default:
			t.Fatalf("shared adapter vector added unsupported operation %q; extend the draft contract deliberately", item.Task.Operation)
		}
	}

	request := PromoteRequest{Name: "Shared adapter task", Assertions: []*actionsv1.ActionDefinition{
		visibleAssertion(`[data-testid="continue"]`), visibleAssertion(`[data-testid="search"]`), visibleAssertion(`[data-testid="continue"]`),
	}}
	first, err := compileTraceDraft(trace, vector.SessionID, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := compileTraceDraft(trace, vector.SessionID, request)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if first.DraftDigest == "" || first.DraftDigest != second.DraftDigest || string(firstJSON) != string(secondJSON) {
		t.Fatalf("shared adapter task draft was not deterministic: %q != %q", first.DraftDigest, second.DraftDigest)
	}
	if first.ActionCount != 3 {
		t.Fatalf("compiled %d actions; shared harness task family contains click, fill, and navigate", first.ActionCount)
	}
	if first.Definition.GetNodes()[0].GetAction().GetClick() == nil || first.Definition.GetNodes()[2].GetAction().GetInput() == nil || first.Definition.GetNodes()[4].GetAction().GetNavigate() == nil {
		t.Fatalf("compiled draft changed the shared click/fill/navigate order: %v", first.Definition.GetNodes())
	}
}

func selectorForRef(t *testing.T, ref string) string {
	t.Helper()
	switch ref {
	case "e123":
		return `[data-testid="continue"]`
	case "e456":
		return `[data-testid="search"]`
	default:
		t.Fatalf("shared vector introduced unknown fixture ref %q", ref)
		return ""
	}
}

func instruction(action *actionsv1.ActionDefinition) *contracts.CompiledInstruction {
	if action == nil {
		return nil
	}
	return &contracts.CompiledInstruction{Action: action}
}

func outcome(action *actionsv1.ActionDefinition) *contracts.StepOutcome {
	if action == nil {
		return nil
	}
	return &contracts.StepOutcome{Success: true}
}
