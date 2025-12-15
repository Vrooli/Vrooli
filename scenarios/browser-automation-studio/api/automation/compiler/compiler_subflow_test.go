package compiler

import (
	"testing"
)

func TestCompileWorkflowWithInlineSubflow(t *testing.T) {
	// Subflow actions with inline workflowDefinition require V1-format node structure
	// until V2 subflow proto support is added
	t.Skip("Subflow actions require V1-format node structure - skip until V2 subflow proto support is added")
}
