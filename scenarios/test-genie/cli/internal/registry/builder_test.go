package registry

import (
	"testing"

	apiRegistry "test-genie/cli/internal/playbookregistry"
)

func TestRegistryFileNameAliasesSharedAPIConstant(t *testing.T) {
	if RegistryFileName != apiRegistry.RegistryFileName {
		t.Fatalf("expected CLI registry file name %q to match API constant %q", RegistryFileName, apiRegistry.RegistryFileName)
	}
}

func TestNewBuilderReusesSharedRegistryBuilder(t *testing.T) {
	if builder := NewBuilder(t.TempDir()); builder == nil {
		t.Fatal("expected shared registry builder instance")
	}
}
