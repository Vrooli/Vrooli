package handoffrules

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	handoffrulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/handoffrules"

	hrdomain "web-console/internal/handoffrules"
)

func newHandler(t *testing.T) *connectHandler {
	t.Helper()
	return NewConnectHandler(Deps{Service: hrdomain.NewMemStore()})
}

func TestRuleRPCsRoundTrip(t *testing.T) {
	ctx := context.Background()
	h := newHandler(t)

	resp, err := h.UpsertRule(ctx, connect.NewRequest(&handoffrulesv1.UpsertRuleRequest{
		Name:     "Plan file",
		Enabled:  true,
		Source:   hrdomain.SourceFilePath,
		Pattern:  "**/.vrooli/plans/*.md",
		Surfaces: []string{"messages"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	created := resp.Msg.GetRule()
	if !created.GetEnabled() || created.GetPattern() != "**/.vrooli/plans/*.md" {
		t.Fatalf("upserted rule = %#v", created)
	}

	listed, err := h.ListRules(ctx, connect.NewRequest(&handoffrulesv1.ListRulesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Msg.GetRules()) != 1 {
		t.Fatalf("listed = %d, want 1", len(listed.Msg.GetRules()))
	}

	if _, err := h.DeleteRule(ctx, connect.NewRequest(&handoffrulesv1.DeleteRuleRequest{Id: created.GetId()})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.DeleteRule(ctx, connect.NewRequest(&handoffrulesv1.DeleteRuleRequest{Id: created.GetId()})); err != nil {
		t.Fatalf("second delete should succeed: %v", err)
	}
}

// TestInvalidRulesAreInvalidArgument covers all three validation failures at
// the transport boundary, because an operator authoring a rule sees this code
// and nothing else.
func TestInvalidRulesAreInvalidArgument(t *testing.T) {
	ctx := context.Background()
	h := newHandler(t)
	cases := map[string]*handoffrulesv1.UpsertRuleRequest{
		"blank name":     {Source: hrdomain.SourceFilePath, Pattern: "*.md"},
		"blank pattern":  {Name: "Anything", Source: hrdomain.SourceFilePath},
		"unknown source": {Name: "Anything", Source: "terminal_output", Pattern: "*.md"},
	}
	for label, req := range cases {
		if _, err := h.UpsertRule(ctx, connect.NewRequest(req)); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("%s code = %v, want invalid_argument", label, connect.CodeOf(err))
		}
	}
}
