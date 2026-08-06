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
	h := NewConnectHandler(Deps{InferenceRoles: []string{"classify.fast", "extract.structured"}})
	resp, err := h.ListProviderRoles(context.Background(), connect.NewRequest(&inventoryv1.ListProviderRolesRequest{Provider: "ai-gateway"}))
	if err != nil {
		t.Fatalf("ListProviderRoles() error = %v", err)
	}
	if len(resp.Msg.GetRoles()) != 2 {
		t.Fatalf("roles = %+v, want two gateway roles", resp.Msg.GetRoles())
	}
	if resp.Msg.GetRoles()[0].GetRole() != "classify.fast" || resp.Msg.GetRoles()[1].GetRole() != "extract.structured" {
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
