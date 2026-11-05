package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestExecutionTimeoutHonorsExplicitWorkflowTimeout(t *testing.T) {
	plan := contracts.ExecutionPlan{Metadata: map[string]any{"executionTimeoutMs": 3900000}}
	require.Equal(t, 3900*time.Second, executionTimeout(plan))
}

func TestExecutionTimeoutCapsExplicitWorkflowTimeout(t *testing.T) {
	plan := contracts.ExecutionPlan{Metadata: map[string]any{"executionTimeoutMs": 9000000}}
	require.Equal(t, 2*time.Hour, executionTimeout(plan))
}

func TestExecutionTimeoutFallsBackToDynamicPolicy(t *testing.T) {
	plan := contracts.ExecutionPlan{Instructions: make([]contracts.CompiledInstruction, 1)}
	require.Equal(t, 90*time.Second, executionTimeout(plan))
}
