package toolregistry

import (
	"context"
	"testing"
)

func TestCreateSandboxToolProjectRootDescription(t *testing.T) {
	tool := NewSandboxToolProvider().createSandboxTool()
	projectRoot, ok := tool.Parameters.Properties["project_root"]
	if !ok {
		t.Fatal("expected project_root parameter")
	}
	if projectRoot.Description != "Root path of the project. Defaults to the repo-contract-resolved repository root when not specified." {
		t.Fatalf("unexpected project_root description: %q", projectRoot.Description)
	}
}

func TestSandboxToolProviderReturnsLifecycleTools(t *testing.T) {
	tools := NewSandboxToolProvider().Tools(context.Background())
	if len(tools) == 0 {
		t.Fatal("expected sandbox lifecycle tools")
	}
}
