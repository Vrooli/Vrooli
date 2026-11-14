package mutation

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
)

type mutationClient struct {
	response *validationv1.RunMutationPilotResponse
	err      error
	request  *validationv1.RunMutationPilotRequest
}

func (c *mutationClient) ValidateScenario(context.Context, *connect.Request[validationv1.ValidateScenarioRequest]) (*connect.Response[validationv1.ValidateScenarioResponse], error) {
	return nil, nil
}

func (c *mutationClient) ReadTestBody(context.Context, *connect.Request[validationv1.ReadTestBodyRequest]) (*connect.Response[validationv1.ReadTestBodyResponse], error) {
	return nil, nil
}

func (c *mutationClient) RunCalibration(context.Context, *connect.Request[validationv1.RunCalibrationRequest]) (*connect.Response[validationv1.RunCalibrationResponse], error) {
	return nil, nil
}

func (c *mutationClient) RunMutationPilot(_ context.Context, request *connect.Request[validationv1.RunMutationPilotRequest]) (*connect.Response[validationv1.RunMutationPilotResponse], error) {
	c.request = request.Msg
	if c.err != nil {
		return nil, c.err
	}
	if c.response == nil {
		return nil, nil
	}
	return connect.NewResponse(c.response), nil
}

func mutationSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario"}},
		Flags: []cliapp.Flag{
			{Name: "workspace"},
			{Name: "package"},
			{Name: "operators"},
			{Name: "max-mutants"},
			{Name: "seed"},
		},
	}
}

func TestMutationPilotBuildsRequestAndRendersReceipts(t *testing.T) {
	client := &mutationClient{response: &validationv1.RunMutationPilotResponse{
		RunId:       "run-1",
		Summary:     &validationv1.MutationSummary{Generated: 2, Killed: 1, Survived: 1, KillRate: 0.5},
		Receipts:    []*validationv1.MutantReceipt{{Disposition: "killed", Operator: "boundary", File: "x.go", Line: 7, OwningTest: "TestX", Detail: "assertion failed"}},
		Limitations: []string{"bounded"},
	}}
	h := newHandlers(&cliapp.ScenarioApp{})
	h.client = client
	ctx, out := cliapptest.NewCapturedRunContext(nil, mutationSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
		Flags: map[string]string{
			"workspace":   "cli",
			"package":     "./internal/foo",
			"max-mutants": "4",
			"seed":        "seed-1",
		},
		FlagLists: map[string][]string{"operators": {"boundary", "return-constant"}},
	})

	if err := h.pilot(ctx); err != nil {
		t.Fatal(err)
	}
	if client.request.GetScenario() != "demo" || client.request.GetWorkspace() != "cli" || client.request.GetPackage() != "./internal/foo" {
		t.Fatalf("request routing = %+v", client.request)
	}
	if client.request.GetMaxMutants() != 4 || client.request.GetSeed() != "seed-1" || len(client.request.GetOperators()) != 2 {
		t.Fatalf("request options = %+v", client.request)
	}
	if got := out.String(); got == "" || !containsAll(got, "Mutation pilot run-1", "killed", "Limitation: bounded") {
		t.Fatalf("rendered output = %q", got)
	}
}

func TestMutationPilotUsesDefaultsAndRejectsInvalidInputs(t *testing.T) {
	client := &mutationClient{response: &validationv1.RunMutationPilotResponse{Summary: &validationv1.MutationSummary{}}}
	h := newHandlers(nil)
	h.client = client
	ctx, _ := cliapptest.NewCapturedRunContext(nil, mutationSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
		Flags:       map[string]string{"seed": "seed-1"},
	})
	if err := h.pilot(ctx); err != nil {
		t.Fatal(err)
	}
	if client.request.GetWorkspace() != "api" || client.request.GetPackage() != "./internal/testquality/..." || len(client.request.GetOperators()) != 3 {
		t.Fatalf("defaults = %+v", client.request)
	}

	invalid, _ := cliapptest.NewCapturedRunContext(nil, mutationSchema(), cliapptest.TestRunContextOptions{Flags: map[string]string{"max-mutants": "nope", "seed": "seed-1"}})
	if err := h.pilot(invalid); err == nil {
		t.Fatal("invalid max-mutants accepted")
	}
	missingSeed, _ := cliapptest.NewCapturedRunContext(nil, mutationSchema(), cliapptest.TestRunContextOptions{})
	if err := h.pilot(missingSeed); err == nil {
		t.Fatal("missing seed accepted")
	}
}

func TestMutationPilotSurfacesAPIAndNilResponses(t *testing.T) {
	for name, client := range map[string]*mutationClient{
		"api error":    {err: errors.New("unavailable")},
		"nil response": {},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHandlers(nil)
			h.client = client
			ctx, _ := cliapptest.NewCapturedRunContext(nil, mutationSchema(), cliapptest.TestRunContextOptions{Flags: map[string]string{"seed": "seed-1"}})
			if err := h.pilot(ctx); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestMutationHelpersAndRegister(t *testing.T) {
	if first(nil) != "" || first([]string{"  value  "}) != "value" || firstOr(nil, "fallback") != "fallback" || firstOr([]string{"value"}, "fallback") != "value" {
		t.Fatal("mutation flag helpers returned unexpected values")
	}
	manifest := []byte(`{"groups":{"mutation":{"subcommands":[{"name":"pilot","handler":"ValidationService.RunMutationPilot"}]}}}`)
	if _, err := Register(&cliapp.ScenarioApp{}, manifest); err == nil {
		t.Fatal("invalid manifest unexpectedly registered")
	}
}

func containsAll(value string, needles ...string) bool {
	for _, needle := range needles {
		if !contains(value, needle) {
			return false
		}
	}
	return true
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
