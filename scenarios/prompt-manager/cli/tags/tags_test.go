package tags

import (
	"errors"
	"strings"
	"testing"

	clitest "prompt-manager/cli/internal/testutil"
)

func TestCommandsRegistersTagCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Tags" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "tag" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsCreate(t *testing.T) {
	if !strings.Contains(usageText(), "create, add") {
		t.Fatalf("usage text missing create command: %s", usageText())
	}
}

func TestListPrintsTagsFromAPI(t *testing.T) {
	color := "#AA00AA"
	description := "Routing"
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/tags", []Tag{
		{ID: "tag-1", Name: "planning", Color: &color, Description: &description},
	})

	stdout, _, err := clitest.Output(t, func() error {
		return route(ctx, []string{"list"})
	})
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if !strings.Contains(stdout, "planning (#AA00AA) - Routing [tag-1]") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}
	req := ctx.LastRequest()
	if req.Method != "GET" || req.Path != "/tags" {
		t.Fatalf("unexpected request: %+v", req)
	}
}

func TestCreateRequiresNameBeforeAPI(t *testing.T) {
	ctx := clitest.NewContext(t)

	err := route(ctx, []string{"create"})
	if err == nil || !strings.Contains(err.Error(), "usage: tag create <name>") {
		t.Fatalf("expected create usage error, got %v", err)
	}
	ctx.RequireNoRequests()
}

func TestCreateSendsOptionalFields(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/tags", Tag{ID: "tag-7", Name: "review"})

	stdout, _, err := clitest.Output(t, func() error {
		return route(ctx, []string{"create", "review", "--color", "#123456", "--description", "Review work"})
	})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if !strings.Contains(stdout, "Created tag: review [tag-7]") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}

	req := ctx.LastRequest()
	if req.Method != "POST" || req.Path != "/tags" {
		t.Fatalf("unexpected request: %+v", req)
	}
	payload, ok := req.Payload.(CreateTagRequest)
	if !ok {
		t.Fatalf("unexpected payload type %T", req.Payload)
	}
	if payload.Name != "review" || payload.Color == nil || *payload.Color != "#123456" || payload.Description == nil || *payload.Description != "Review work" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestListSurfacesAPIError(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Fail("GET", "/tags", errors.New("api offline"))

	err := route(ctx, []string{"list"})
	if err == nil || !strings.Contains(err.Error(), "failed to list tags: api offline") {
		t.Fatalf("expected list API error, got %v", err)
	}
}
