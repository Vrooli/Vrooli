package testing

import (
	"errors"
	"strings"
	stdtesting "testing"
	"time"

	clitest "prompt-manager/cli/internal/testutil"
)

func TestCommandsRegistersTestingCommand(t *stdtesting.T) {
	group := Commands(nil)
	if group.Title != "Testing" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "test" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsRunAndHistory(t *stdtesting.T) {
	text := usageText()
	if !strings.Contains(text, "run, execute") || !strings.Contains(text, "history, results") {
		t.Fatalf("usage text missing testing subcommands: %s", text)
	}
}

func TestRunRequiresSkillIDBeforeAPI(t *stdtesting.T) {
	ctx := clitest.NewContext(t)

	err := route(ctx, []string{"run"})
	if err == nil || !strings.Contains(err.Error(), "usage: test run <skill-id>") {
		t.Fatalf("expected run usage error, got %v", err)
	}
	ctx.RequireNoRequests()
}

func TestRunSendsRoleVariablesAndPrintsResponse(t *stdtesting.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/skills/skill-1/test", TestResponse{
		TestID:       "test-1",
		Role:         "chat.small",
		Response:     "Generated answer",
		ResponseTime: 42,
		TokenCount:   128,
		TestedAt:     time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})

	stdout, _, err := clitest.Output(t, func() error {
		return route(ctx, []string{
			"run", "skill-1",
			"--role", "chat.small",
			"--vars", "topic=planning, tone=direct",
			"--max-tokens", "250",
			"--temperature", "0.2",
		})
	})
	if err != nil {
		t.Fatalf("run test: %v", err)
	}
	if !strings.Contains(stdout, "Testing skill skill-1 with chat.small") || !strings.Contains(stdout, "Generated answer") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}

	req := ctx.LastRequest()
	if req.Method != "POST" || req.Path != "/skills/skill-1/test" {
		t.Fatalf("unexpected request: %+v", req)
	}
	payload, ok := req.Payload.(TestRequest)
	if !ok {
		t.Fatalf("unexpected payload type %T", req.Payload)
	}
	if payload.Role != "chat.small" || payload.Variables["topic"] != "planning" || payload.Variables["tone"] != "direct" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.MaxTokens == nil || *payload.MaxTokens != 250 || payload.Temperature == nil || *payload.Temperature != 0.2 {
		t.Fatalf("unexpected generation controls: %+v", payload)
	}
}

func TestHistorySurfacesAPIError(t *stdtesting.T) {
	ctx := clitest.NewContext(t)
	ctx.Fail("GET", "/skills/skill-1/test-history", errors.New("history unavailable"))

	err := route(ctx, []string{"history", "skill-1"})
	if err == nil || !strings.Contains(err.Error(), "failed to get test history: history unavailable") {
		t.Fatalf("expected history API error, got %v", err)
	}
}
