package main

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// TestDemoSmokeWorkflowJSON_RoundTripsEnums pins the regression that
// previously bit us: building the seed workflow as map[string]any →
// json.Marshal → REST body lost ChangeSource and wait-event enum
// strings during snake-case JSON encoding. The protojson path used by
// ensureWorkflow must keep those enums alive.
func TestDemoSmokeWorkflowJSON_RoundTripsEnums(t *testing.T) {
	flow := &workflowsv1.WorkflowDefinitionV2{}
	if err := protojson.Unmarshal([]byte(demoSmokeWorkflowJSON), flow); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(flow.GetNodes()) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(flow.GetNodes()))
	}

	nav := flow.GetNodes()[0].GetAction()
	if nav == nil || nav.GetType() != basactions.ActionType_ACTION_TYPE_NAVIGATE {
		t.Fatalf("first node must be NAVIGATE, got %v", nav.GetType())
	}
	navParams := nav.GetNavigate()
	if navParams == nil {
		t.Fatal("navigate params missing — enum/oneof routing broke")
	}
	if got := navParams.GetUrl(); got != "https://example.com/" {
		t.Errorf("navigate url lost: %q", got)
	}
	if got := navParams.GetDestinationType(); got != basactions.NavigateDestinationType_NAVIGATE_DESTINATION_TYPE_URL {
		t.Errorf("destination_type enum lost: %v", got)
	}
	if got := navParams.GetWaitUntil(); got != basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_LOAD {
		t.Errorf("wait_until enum lost: %v", got)
	}

	assert := flow.GetNodes()[1].GetAction()
	if assert == nil || assert.GetType() != basactions.ActionType_ACTION_TYPE_ASSERT {
		t.Fatalf("second node must be ASSERT, got %v", assert.GetType())
	}
	if a := assert.GetAssert(); a == nil || a.GetSelector() != "body" {
		t.Errorf("assert selector lost: %+v", a)
	}

	// Re-encode and ensure no DiscardUnknown warnings sneak in (a
	// silent decode is the failure mode that hid enums last time).
	if out, err := protojson.Marshal(flow); err != nil {
		t.Errorf("re-marshal: %v", err)
	} else if !strings.Contains(string(out), "NAVIGATE_WAIT_EVENT_LOAD") {
		t.Errorf("round-trip dropped enum value: %s", out)
	}
}
