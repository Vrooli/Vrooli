package mocks

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// FakeToolProvider is a reusable fake for the tool discovery seam.
//
// The fake intentionally does not import toolregistry, which lets tests inside
// package toolregistry use it without creating an import cycle.
type FakeToolProvider struct {
	NameValue       string
	ToolsValue      []*toolspb.ToolDefinition
	CategoriesValue []*toolspb.ToolCategory
}

func NewFakeToolProvider(name string) *FakeToolProvider {
	if name == "" {
		name = "fake-provider"
	}
	return &FakeToolProvider{NameValue: name}
}

func (f *FakeToolProvider) Name() string {
	return f.NameValue
}

func (f *FakeToolProvider) Tools(context.Context) []*toolspb.ToolDefinition {
	return append([]*toolspb.ToolDefinition(nil), f.ToolsValue...)
}

func (f *FakeToolProvider) Categories(context.Context) []*toolspb.ToolCategory {
	return append([]*toolspb.ToolCategory(nil), f.CategoriesValue...)
}
