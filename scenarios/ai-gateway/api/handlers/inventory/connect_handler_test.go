package inventory

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory"

	"ai-gateway/internal/providers"
	providermocks "ai-gateway/internal/providers/mocks"
)

func TestListProviderRoles_ReturnsResourceInventory(t *testing.T) { // [REQ:AIGW-INVENTORY-ROLES]
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-ollama policy roles --json":     {Stdout: `{"roles":[{"schema_version":"2026-06-10","role":"embedding.default","required_capabilities":["embedding"]}]}`},
		"resource-openrouter policy roles --json": {Stdout: `{"roles":[{"schema_version":"2026-06-30","role":"chat.default","required_capabilities":["chat"]}]}`},
	}}
	handler := NewConnectHandler(Deps{Runner: runner})

	resp, err := handler.ListProviderRoles(context.Background(), connect.NewRequest(&inventoryv1.ListProviderRolesRequest{}))
	if err != nil {
		t.Fatalf("ListProviderRoles() error = %v", err)
	}
	if got, want := len(resp.Msg.GetRoles()), 2; got != want {
		t.Fatalf("roles len = %d, want %d", got, want)
	}
	if resp.Msg.GetRoles()[0].GetProvider() != "ollama" || resp.Msg.GetRoles()[0].GetRole() != "embedding.default" {
		t.Fatalf("first role = %+v", resp.Msg.GetRoles()[0])
	}
}

func TestListProviderRoles_IncludesGatewayInferenceRoles(t *testing.T) {
	h := NewConnectHandler(Deps{InferenceRoles: []string{"classify.fast", "extract.structured", "judge.default"}})
	resp, err := h.ListProviderRoles(context.Background(), connect.NewRequest(&inventoryv1.ListProviderRolesRequest{Provider: "ai-gateway"}))
	if err != nil {
		t.Fatalf("ListProviderRoles() error = %v", err)
	}
	if len(resp.Msg.GetRoles()) != 3 {
		t.Fatalf("roles = %+v, want three gateway roles", resp.Msg.GetRoles())
	}
	if resp.Msg.GetRoles()[0].GetRole() != "classify.fast" || resp.Msg.GetRoles()[1].GetRole() != "extract.structured" || resp.Msg.GetRoles()[2].GetRole() != "judge.default" {
		t.Fatalf("roles = %+v, want sorted inference roles", resp.Msg.GetRoles())
	}
}

func TestSmokeProvider_ReturnsTypedFailure(t *testing.T) { // [REQ:AIGW-INVENTORY-SMOKE]
	runner := &providermocks.FakeRunner{}
	handler := NewConnectHandler(Deps{Runner: runner})

	resp, err := handler.SmokeProvider(context.Background(), connect.NewRequest(&inventoryv1.SmokeProviderRequest{Provider: "ollama"}))
	if err != nil {
		t.Fatalf("SmokeProvider() error = %v", err)
	}
	if resp.Msg.GetStatus() != "unavailable" || resp.Msg.GetCode() != "missing_fixture" {
		t.Fatalf("smoke = %+v", resp.Msg)
	}
}

// Contract C5 reaches callers here: a local decision role's declared limits and
// placement are projected so a caller sizes digests and batches from data, and
// a role that declares none projects none.
func TestListProviderRoles_ExposesDecisionLimitsAndPlacement(t *testing.T) {
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-ollaya policy roles --json": {Stdout: `{"roles":[
			{"schema_version":"2026-10-30","role":"decide.local.kev","model":"kev:4b","locality":"local","required_capabilities":["decide"],
			 "limits":{"context_tokens":8192,"max_options":255,"max_questions":256},
			 "placement":{"device":"gpu","vram_bytes":6442450944,"keep_alive":"2m"}},
			{"schema_version":"2026-10-30","role":"decide.jev.edenai","model":"typesafe/jev-latest","locality":"remote","required_capabilities":["decide"]}]}`},
	}}
	handler := NewConnectHandler(Deps{Runner: runner})
	resp, err := handler.ListProviderRoles(context.Background(), connect.NewRequest(&inventoryv1.ListProviderRolesRequest{Provider: "ollaya"}))
	if err != nil {
		t.Fatalf("ListProviderRoles() error = %v", err)
	}
	byRole := map[string]*inventoryv1.ProviderRole{}
	for _, role := range resp.Msg.GetRoles() {
		byRole[role.GetRole()] = role
	}
	kev := byRole["decide.local.kev"]
	if kev.GetModel() != "kev:4b" || kev.GetLimits().GetContextTokens() != 8192 || kev.GetLimits().GetMaxOptions() != 255 || kev.GetLimits().GetMaxQuestions() != 256 ||
		kev.GetPlacement().GetDevice() != "gpu" || kev.GetPlacement().GetVramBytes() != 6442450944 {
		t.Fatalf("decide.local.kev = %+v, want the declared model, limits and placement", kev)
	}
	if hosted := byRole["decide.jev.edenai"]; hosted.GetLimits() != nil || hosted.GetPlacement() != nil {
		t.Fatalf("hosted role = %+v, want no local limits or placement", hosted)
	}
}
