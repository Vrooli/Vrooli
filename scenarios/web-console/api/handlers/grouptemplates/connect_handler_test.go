package grouptemplates

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	grouptemplatesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/grouptemplates"

	gtdomain "web-console/internal/grouptemplates"
)

func newHandler(t *testing.T) *connectHandler {
	t.Helper()
	return NewConnectHandler(Deps{Service: gtdomain.NewMemStore()})
}

// TestTemplateRPCsRoundTripAThreeRoleTemplate uses three roles on purpose: a
// transport that quietly assumed a pair would pass a two-role test.
func TestTemplateRPCsRoundTripAThreeRoleTemplate(t *testing.T) {
	ctx := context.Background()
	h := newHandler(t)

	resp, err := h.UpsertTemplate(ctx, connect.NewRequest(&grouptemplatesv1.UpsertTemplateRequest{
		Name:  "Plan, build, critique",
		Color: "#22d3ee",
		Roles: []*grouptemplatesv1.TemplateRole{
			{Label: "Planner", Command: "claude", StartMode: gtdomain.StartModeEager},
			{Label: "Implementer", Command: "codex --yolo", IncomingPrompt: "Implement the plan at {{payload}}", StartMode: gtdomain.StartModeWaiting},
			{Label: "Critic", Command: "opencode", IncomingPrompt: "Critique {{payload}}", StartMode: gtdomain.StartModeWaiting},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	created := resp.Msg.GetTemplate()
	if len(created.GetRoles()) != 3 {
		t.Fatalf("upserted roles = %d, want 3", len(created.GetRoles()))
	}
	if created.GetRoles()[1].GetIncomingPrompt() != "Implement the plan at {{payload}}" {
		t.Fatalf("incoming prompt lost in transport: %q", created.GetRoles()[1].GetIncomingPrompt())
	}

	listed, err := h.ListTemplates(ctx, connect.NewRequest(&grouptemplatesv1.ListTemplatesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Msg.GetTemplates()) != 1 {
		t.Fatalf("listed = %d, want 1", len(listed.Msg.GetTemplates()))
	}

	if _, err := h.DeleteTemplate(ctx, connect.NewRequest(&grouptemplatesv1.DeleteTemplateRequest{Id: created.GetId()})); err != nil {
		t.Fatal(err)
	}
	// Idempotent: deleting a missing id is the state the caller asked for.
	if _, err := h.DeleteTemplate(ctx, connect.NewRequest(&grouptemplatesv1.DeleteTemplateRequest{Id: created.GetId()})); err != nil {
		t.Fatalf("second delete should succeed: %v", err)
	}
}

// TestUnknownStartModeIsInvalidArgument keeps a caller mistake from reading as
// a server fault, which is what would send an operator to the logs instead of
// to the field they mistyped.
func TestUnknownStartModeIsInvalidArgument(t *testing.T) {
	_, err := newHandler(t).UpsertTemplate(context.Background(), connect.NewRequest(&grouptemplatesv1.UpsertTemplateRequest{
		Name:  "Bad",
		Roles: []*grouptemplatesv1.TemplateRole{{Label: "Planner", StartMode: "immediately"}},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unknown start mode code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestBlankTemplateNameIsInvalidArgument(t *testing.T) {
	_, err := newHandler(t).UpsertTemplate(context.Background(), connect.NewRequest(&grouptemplatesv1.UpsertTemplateRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("blank name code = %v, want invalid_argument", connect.CodeOf(err))
	}
}
