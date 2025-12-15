//go:build legacydb
// +build legacydb

package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
)

type stubPlanCompiler struct {
	called       bool
	plan         contracts.ExecutionPlan
	instructions []contracts.CompiledInstruction
	err          error
	execID       uuid.UUID
	workflowID   uuid.UUID
}

func (s *stubPlanCompiler) Compile(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	s.called = true
	s.execID = executionID
	if workflow != nil {
		s.workflowID = workflow.ID
	}
	return s.plan, s.instructions, s.err
}

func TestBuildContractsPlanWithCustomCompiler(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] invokes custom compiler with correct arguments", func(t *testing.T) {
		execID := uuid.New()
		wfID := uuid.New()
		workflow := &database.Workflow{ID: wfID}

		expectedPlan := contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     wfID,
			CreatedAt:      time.Now().UTC(),
		}
		expectedInstructions := []contracts.CompiledInstruction{
			{Index: 0, NodeID: "one", Type: "noop"},
		}

		compiler := &stubPlanCompiler{
			plan:         expectedPlan,
			instructions: expectedInstructions,
		}

		plan, instructions, err := BuildContractsPlanWithCompiler(context.Background(), execID, workflow, compiler)

		require.NoError(t, err)
		assert.True(t, compiler.called, "custom compiler should be invoked")
		assert.Equal(t, execID, compiler.execID)
		assert.Equal(t, wfID, compiler.workflowID)
		assert.Equal(t, expectedPlan.ExecutionID, plan.ExecutionID)
		assert.Equal(t, expectedPlan.WorkflowID, plan.WorkflowID)
		require.Len(t, instructions, 1)
		assert.Equal(t, "one", instructions[0].NodeID)
		assert.Equal(t, "noop", instructions[0].Type)
	})
}

func TestBuildContractsPlanWithCustomCompiler_NilCompiler(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] uses default compiler when nil is passed", func(t *testing.T) {
		execID := uuid.New()
		wfID := uuid.New()
		workflow := &database.Workflow{
			ID:   wfID,
			Name: "test-workflow",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "nav", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				},
				"edges": []any{},
			},
		}

		// Pass nil compiler - should use DefaultPlanCompiler
		plan, instructions, err := BuildContractsPlanWithCompiler(context.Background(), execID, workflow, nil)

		require.NoError(t, err)
		assert.Equal(t, execID, plan.ExecutionID)
		assert.Equal(t, wfID, plan.WorkflowID)
		require.Len(t, instructions, 1)
		assert.Equal(t, "nav", instructions[0].NodeID)
	})
}

func TestBuildContractsPlanWithCustomCompiler_ErrorPropagation(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] propagates compiler errors", func(t *testing.T) {
		execID := uuid.New()
		workflow := &database.Workflow{ID: uuid.New()}
		expectedErr := errors.New("compilation failed")

		compiler := &stubPlanCompiler{
			err: expectedErr,
		}

		plan, instructions, err := BuildContractsPlanWithCompiler(context.Background(), execID, workflow, compiler)

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.True(t, compiler.called, "compiler should still be invoked")
		assert.Empty(t, instructions)
		assert.Equal(t, uuid.Nil, plan.ExecutionID)
	})
}

func TestBuildContractsPlan(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] uses default compiler", func(t *testing.T) {
		execID := uuid.New()
		wfID := uuid.New()
		workflow := &database.Workflow{
			ID:   wfID,
			Name: "default-compiler-test",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "click", "type": "click", "data": map[string]any{"selector": "#btn"}},
				},
				"edges": []any{},
			},
		}

		plan, instructions, err := BuildContractsPlan(context.Background(), execID, workflow)

		require.NoError(t, err)
		assert.Equal(t, execID, plan.ExecutionID)
		assert.Equal(t, wfID, plan.WorkflowID)
		require.Len(t, instructions, 1)
		assert.Equal(t, "click", instructions[0].NodeID)
		assert.Equal(t, "click", instructions[0].Type)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns error for invalid workflow", func(t *testing.T) {
		execID := uuid.New()
		workflow := &database.Workflow{
			ID:             uuid.New(),
			Name:           "invalid-workflow",
			FlowDefinition: nil,
		}

		plan, instructions, err := BuildContractsPlan(context.Background(), execID, workflow)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "flow_definition")
		assert.Empty(t, instructions)
		assert.Equal(t, uuid.Nil, plan.ExecutionID)
	})
}

func TestPlanCompilerForEngine(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns contract compiler for browserless", func(t *testing.T) {
		comp := PlanCompilerForEngine("browserless")

		assert.NotNil(t, comp)
		_, ok := comp.(*ContractPlanCompiler)
		assert.True(t, ok, "browserless should use ContractPlanCompiler")
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns contract compiler for browserless-contract", func(t *testing.T) {
		comp := PlanCompilerForEngine("browserless-contract")

		assert.NotNil(t, comp)
		_, ok := comp.(*ContractPlanCompiler)
		assert.True(t, ok, "browserless-contract should use ContractPlanCompiler")
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns default compiler for unknown engine", func(t *testing.T) {
		comp := PlanCompilerForEngine("unknown-engine-xyz")

		assert.NotNil(t, comp)
		// Unknown engines should fall back to DefaultPlanCompiler
		assert.Equal(t, DefaultPlanCompiler, comp)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] handles case-insensitive engine names", func(t *testing.T) {
		comp1 := PlanCompilerForEngine("BROWSERLESS")
		comp2 := PlanCompilerForEngine("Browserless")
		comp3 := PlanCompilerForEngine("browserless")

		assert.NotNil(t, comp1)
		assert.NotNil(t, comp2)
		assert.NotNil(t, comp3)
		// All should resolve to the same compiler type
		_, ok1 := comp1.(*ContractPlanCompiler)
		_, ok2 := comp2.(*ContractPlanCompiler)
		_, ok3 := comp3.(*ContractPlanCompiler)
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.True(t, ok3)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] handles whitespace in engine names", func(t *testing.T) {
		comp := PlanCompilerForEngine("  browserless  ")

		assert.NotNil(t, comp)
		_, ok := comp.(*ContractPlanCompiler)
		assert.True(t, ok, "whitespace-padded name should still resolve")
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns default for empty engine name", func(t *testing.T) {
		comp := PlanCompilerForEngine("")

		assert.NotNil(t, comp)
		assert.Equal(t, DefaultPlanCompiler, comp)
	})
}

func TestRegisterPlanCompiler(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] registers custom compiler", func(t *testing.T) {
		customCompiler := &stubPlanCompiler{}

		// Register a custom compiler
		RegisterPlanCompiler("test-custom-engine", customCompiler)

		// Retrieve it
		retrieved := PlanCompilerForEngine("test-custom-engine")

		assert.Equal(t, customCompiler, retrieved)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] ignores empty engine name", func(t *testing.T) {
		customCompiler := &stubPlanCompiler{}

		// This should be a no-op
		RegisterPlanCompiler("", customCompiler)

		// Empty name should still return default
		comp := PlanCompilerForEngine("")
		assert.Equal(t, DefaultPlanCompiler, comp)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] ignores nil compiler", func(t *testing.T) {
		// Store what's currently registered for a test key
		RegisterPlanCompiler("test-nil-check", &stubPlanCompiler{})
		before := PlanCompilerForEngine("test-nil-check")

		// Try to register nil - should be ignored
		RegisterPlanCompiler("test-nil-check", nil)

		// Should still have the previous registration
		after := PlanCompilerForEngine("test-nil-check")
		assert.Equal(t, before, after)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] normalizes engine name case", func(t *testing.T) {
		customCompiler := &stubPlanCompiler{}

		// Register with uppercase
		RegisterPlanCompiler("TEST-CASE-ENGINE", customCompiler)

		// Should be retrievable with lowercase
		retrieved := PlanCompilerForEngine("test-case-engine")
		assert.Equal(t, customCompiler, retrieved)
	})
}

func TestDefaultPlanCompiler(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] is initialized as ContractPlanCompiler", func(t *testing.T) {
		assert.NotNil(t, DefaultPlanCompiler)
		_, ok := DefaultPlanCompiler.(*ContractPlanCompiler)
		assert.True(t, ok, "DefaultPlanCompiler should be ContractPlanCompiler")
	})
}
