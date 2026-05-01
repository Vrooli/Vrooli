package mocks

import (
	"context"
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

func TestFakeToolProvider_DefaultName(t *testing.T) {
	provider := NewFakeToolProvider("")

	if provider.Name() != "fake-provider" {
		t.Fatalf("expected default provider name, got %q", provider.Name())
	}
}

func TestFakeToolProvider_ReturnsCopies(t *testing.T) {
	provider := NewFakeToolProvider("tools")
	provider.ToolsValue = []*toolspb.ToolDefinition{{Name: "first"}}
	provider.CategoriesValue = []*toolspb.ToolCategory{{Id: "cat"}}

	tools := provider.Tools(context.Background())
	categories := provider.Categories(context.Background())
	tools[0] = &toolspb.ToolDefinition{Name: "changed"}
	categories[0] = &toolspb.ToolCategory{Id: "changed"}

	if provider.ToolsValue[0].Name != "first" {
		t.Fatalf("expected stored tools slice to be protected from caller mutation")
	}
	if provider.CategoriesValue[0].Id != "cat" {
		t.Fatalf("expected stored categories slice to be protected from caller mutation")
	}
}
