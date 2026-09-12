package experiments

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	clitest "prompt-manager/cli/internal/testutil"
)

func TestCommandsRegistersExperimentCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Experiments" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "experiment" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsLifecycleCommands(t *testing.T) {
	text := usageText()
	for _, want := range []string{"create, add", "start", "conclude"} {
		if !strings.Contains(text, want) {
			t.Fatalf("usage text missing %q: %s", want, text)
		}
	}
}

func TestListUsesSkillScopedEndpointWhenFilterProvided(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/skills/skill-1/experiments", []ExperimentResponse{{
		ID:      "exp-1",
		SkillID: "skill-1",
		Name:    "Variant test",
		Status:  "draft",
		Arms: []ExperimentArmResp{
			{VariantID: "a", Weight: 0.5},
			{VariantID: "b", Weight: 0.5},
		},
	}})

	stdout, _, err := clitest.Output(t, func() error {
		return route(ctx, []string{"list", "--skill", "skill-1"})
	})
	if err != nil {
		t.Fatalf("list experiments: %v", err)
	}
	if !strings.Contains(stdout, "Variant test") || !strings.Contains(stdout, "2 arms") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}
	req := ctx.LastRequest()
	if req.Method != "GET" || req.Path != "/skills/skill-1/experiments" {
		t.Fatalf("unexpected request: %+v", req)
	}
}

func TestCreateValidatesRequiredFieldsBeforeAPI(t *testing.T) {
	ctx := clitest.NewContext(t)

	err := route(ctx, []string{"create", "--name", "Missing skill", "--arm", "a:0.5", "--arm", "b:0.5"})
	if err == nil || !strings.Contains(err.Error(), "--skill is required") {
		t.Fatalf("expected missing skill error, got %v", err)
	}
	ctx.RequireNoRequests()
}

func TestCreateSendsArmsAndDerivedID(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/experiments", ExperimentResponse{
		ID:      "prompt-tone-test",
		SkillID: "skill-1",
		Name:    "Prompt Tone Test",
		Status:  "draft",
	})

	stdout, _, err := clitest.Output(t, func() error {
		return route(ctx, []string{
			"create",
			"--skill", "skill-1",
			"--name", "Prompt Tone Test",
			"--hypothesis", "Tone affects quality",
			"--arm", "variant-a:0.75",
			"--arm=variant-b:0.25",
		})
	})
	if err != nil {
		t.Fatalf("create experiment: %v", err)
	}
	if !strings.Contains(stdout, "Created experiment: Prompt Tone Test [prompt-tone-test]") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}

	req := ctx.LastRequest()
	if req.Method != "POST" || req.Path != "/experiments" {
		t.Fatalf("unexpected request: %+v", req)
	}
	raw, err := json.Marshal(req.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var payload struct {
		ID         string `json:"id"`
		SkillID    string `json:"skillId"`
		Name       string `json:"name"`
		Hypothesis string `json:"hypothesis"`
		Arms       []struct {
			VariantID string  `json:"variantId"`
			Weight    float64 `json:"weight"`
		} `json:"arms"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.ID != "prompt-tone-test" || payload.SkillID != "skill-1" || payload.Hypothesis != "Tone affects quality" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if len(payload.Arms) != 2 || payload.Arms[0].VariantID != "variant-a" || payload.Arms[0].Weight != 0.75 || payload.Arms[1].VariantID != "variant-b" || payload.Arms[1].Weight != 0.25 {
		t.Fatalf("unexpected arms: %+v", payload.Arms)
	}
}

func TestConcludeSurfacesAPIError(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Fail("POST", "/experiments/exp-1/conclude", errors.New("experiment is not running"))

	err := route(ctx, []string{"conclude", "exp-1", "variant-a", "--notes", "winner"})
	if err == nil || !strings.Contains(err.Error(), "failed to conclude experiment: experiment is not running") {
		t.Fatalf("expected conclude API error, got %v", err)
	}

	req := ctx.LastRequest()
	if req.Method != "POST" || req.Path != "/experiments/exp-1/conclude" {
		t.Fatalf("unexpected request: %+v", req)
	}
}
