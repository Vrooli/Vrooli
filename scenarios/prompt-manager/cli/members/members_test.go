package members

import (
	"errors"
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestCommandsRegistersMemberCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Members" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "member" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsMemberLifecycle(t *testing.T) {
	if !strings.Contains(usageText(), "create, add") || !strings.Contains(usageText(), "delete, rm") {
		t.Fatalf("usage text missing lifecycle commands: %s", usageText())
	}
}

func TestListPrintsMembersFromAPI(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/members", []Member{
		{ID: "mem-1", Name: "Ava"},
		{ID: "mem-2", Name: "Noah"},
	})

	stdout, _, err := clitest.Output(t, func() error {
		return route(ctx, []string{"list"})
	})
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if !strings.Contains(stdout, "Ava [mem-1]") || !strings.Contains(stdout, "Noah [mem-2]") {
		t.Fatalf("list output missing members:\n%s", stdout)
	}
	req := ctx.LastRequest()
	if req.Method != "GET" || req.Path != "/members" {
		t.Fatalf("unexpected request: %+v", req)
	}
}

func TestCreateRequiresNameBeforeAPI(t *testing.T) {
	ctx := clitest.NewContext(t)

	err := route(ctx, []string{"create"})
	if err == nil || !strings.Contains(err.Error(), "usage: member create <name>") {
		t.Fatalf("expected create usage error, got %v", err)
	}
	ctx.RequireNoRequests()
}

func TestCreateSendsPayloadAndPrintsCreatedMember(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/members", Member{ID: "mem-7", Name: "Riley"})

	stdout, _, err := clitest.Output(t, func() error {
		return route(ctx, []string{
			"create", "Riley",
			"--body-color", "#111111",
			"--head-color", "#222222",
			"--accent-color", "#333333",
		})
	})
	if err != nil {
		t.Fatalf("create member: %v", err)
	}
	if !strings.Contains(stdout, "Created member: Riley [mem-7]") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}

	req := ctx.LastRequest()
	if req.Method != "POST" || req.Path != "/members" {
		t.Fatalf("unexpected request: %+v", req)
	}
	payload, ok := req.Payload.(CreateMemberRequest)
	if !ok {
		t.Fatalf("unexpected payload type %T", req.Payload)
	}
	if payload.Name != "Riley" || payload.BodyColor != "#111111" || payload.HeadColor != "#222222" || payload.AccentColor != "#333333" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestDeleteSurfacesAPIErrorAfterConfirmationLookup(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/members/mem-1", Member{ID: "mem-1", Name: "Ava"})
	ctx.Fail("DELETE", "/members/mem-1", errors.New("permission denied"))

	err := route(ctx, []string{"delete", "mem-1", "--force"})
	if err == nil || !strings.Contains(err.Error(), "failed to delete member: permission denied") {
		t.Fatalf("expected delete API error, got %v", err)
	}

	requests := ctx.Requests()
	if len(requests) != 2 {
		t.Fatalf("expected lookup then delete, got %+v", requests)
	}
	if requests[0].Method != "GET" || requests[1].Method != "DELETE" {
		t.Fatalf("unexpected requests: %+v", requests)
	}
}
