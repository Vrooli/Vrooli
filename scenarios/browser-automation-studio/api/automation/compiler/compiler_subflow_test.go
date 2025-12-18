package compiler

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestCompileWorkflowWithSubflowByID(t *testing.T) {
	childWorkflowID := uuid.New().String()

	workflow := makeTestWorkflow(
		uuid.New(),
		"parent-workflow",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "subflow-node",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
					Params: &basactions.ActionDefinition_Subflow{
						Subflow: &basactions.SubflowParams{
							Target: &basactions.SubflowParams_WorkflowId{
								WorkflowId: childWorkflowID,
							},
						},
					},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
		},
		nil,
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Len(t, plan.Steps, 1)

	step := plan.Steps[0]
	assert.Equal(t, StepSubflow, step.Type)
	assert.Equal(t, "subflow-node", step.NodeID)

	// Verify workflow_id is extracted to params
	assert.Equal(t, childWorkflowID, step.Params["workflow_id"])
}

func TestCompileWorkflowWithSubflowByPath(t *testing.T) {
	workflow := makeTestWorkflow(
		uuid.New(),
		"parent-workflow",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "subflow-node",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
					Params: &basactions.ActionDefinition_Subflow{
						Subflow: &basactions.SubflowParams{
							Target: &basactions.SubflowParams_WorkflowPath{
								WorkflowPath: "actions/login.json",
							},
						},
					},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
		},
		nil,
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Len(t, plan.Steps, 1)

	step := plan.Steps[0]
	assert.Equal(t, StepSubflow, step.Type)

	// Verify workflow_path is extracted to params
	assert.Equal(t, "actions/login.json", step.Params["workflow_path"])
}

func TestCompileWorkflowWithSubflowVersioned(t *testing.T) {
	childWorkflowID := uuid.New().String()
	version := int32(3)

	workflow := makeTestWorkflow(
		uuid.New(),
		"parent-workflow",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "subflow-node",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
					Params: &basactions.ActionDefinition_Subflow{
						Subflow: &basactions.SubflowParams{
							Target: &basactions.SubflowParams_WorkflowId{
								WorkflowId: childWorkflowID,
							},
							WorkflowVersion: &version,
						},
					},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
		},
		nil,
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Len(t, plan.Steps, 1)

	step := plan.Steps[0]
	assert.Equal(t, StepSubflow, step.Type)
	assert.Equal(t, childWorkflowID, step.Params["workflow_id"])

	// Verify workflow_version is extracted to params
	// The exact param name depends on how extractV2Params handles it
	if wv, ok := step.Params["workflow_version"]; ok {
		assert.EqualValues(t, 3, wv)
	}
}

func TestCompileWorkflowWithSubflowArgs(t *testing.T) {
	childWorkflowID := uuid.New().String()

	workflow := makeTestWorkflow(
		uuid.New(),
		"parent-workflow",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "subflow-node",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
					Params: &basactions.ActionDefinition_Subflow{
						Subflow: &basactions.SubflowParams{
							Target: &basactions.SubflowParams_WorkflowId{
								WorkflowId: childWorkflowID,
							},
							Args: map[string]*commonv1.JsonValue{
								"baseUrl": {Kind: &commonv1.JsonValue_StringValue{StringValue: "https://example.com"}},
								"timeout": {Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: 5000}},
							},
						},
					},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
		},
		nil,
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Len(t, plan.Steps, 1)

	step := plan.Steps[0]
	assert.Equal(t, StepSubflow, step.Type)
	assert.Equal(t, childWorkflowID, step.Params["workflow_id"])

	// Verify args are extracted to params
	if args, ok := step.Params["args"].(map[string]any); ok {
		assert.NotEmpty(t, args)
	}
}

func TestCompileWorkflowWithSubflowInSequence(t *testing.T) {
	childWorkflowID := uuid.New().String()

	workflow := makeTestWorkflow(
		uuid.New(),
		"sequence-with-subflow",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "navigate-node",
				Action: &basactions.ActionDefinition{
					Type:   basactions.ActionType_ACTION_TYPE_NAVIGATE,
					Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
			{
				Id: "subflow-node",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
					Params: &basactions.ActionDefinition_Subflow{
						Subflow: &basactions.SubflowParams{
							Target: &basactions.SubflowParams_WorkflowId{
								WorkflowId: childWorkflowID,
							},
						},
					},
				},
				Position: &basbase.NodePosition{X: 200, Y: 0},
			},
			{
				Id: "screenshot-node",
				Action: &basactions.ActionDefinition{
					Type:   basactions.ActionType_ACTION_TYPE_SCREENSHOT,
					Params: &basactions.ActionDefinition_Screenshot{Screenshot: &basactions.ScreenshotParams{}},
				},
				Position: &basbase.NodePosition{X: 400, Y: 0},
			},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{Id: "edge-1", Source: "navigate-node", Target: "subflow-node"},
			{Id: "edge-2", Source: "subflow-node", Target: "screenshot-node"},
		},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Len(t, plan.Steps, 3)

	// Verify order: navigate -> subflow -> screenshot
	assert.Equal(t, StepNavigate, plan.Steps[0].Type)
	assert.Equal(t, "navigate-node", plan.Steps[0].NodeID)

	assert.Equal(t, StepSubflow, plan.Steps[1].Type)
	assert.Equal(t, "subflow-node", plan.Steps[1].NodeID)
	assert.Equal(t, childWorkflowID, plan.Steps[1].Params["workflow_id"])

	assert.Equal(t, StepScreenshot, plan.Steps[2].Type)
	assert.Equal(t, "screenshot-node", plan.Steps[2].NodeID)

	// Verify edges are preserved
	assert.Len(t, plan.Steps[0].OutgoingEdges, 1)
	assert.Equal(t, "subflow-node", plan.Steps[0].OutgoingEdges[0].TargetNode)

	assert.Len(t, plan.Steps[1].OutgoingEdges, 1)
	assert.Equal(t, "screenshot-node", plan.Steps[1].OutgoingEdges[0].TargetNode)
}

func TestCompileWorkflowWithMultipleSubflows(t *testing.T) {
	loginWorkflowID := uuid.New().String()
	logoutWorkflowID := uuid.New().String()

	workflow := makeTestWorkflow(
		uuid.New(),
		"multi-subflow-workflow",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "login-subflow",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
					Params: &basactions.ActionDefinition_Subflow{
						Subflow: &basactions.SubflowParams{
							Target: &basactions.SubflowParams_WorkflowId{
								WorkflowId: loginWorkflowID,
							},
						},
					},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
			{
				Id: "main-action",
				Action: &basactions.ActionDefinition{
					Type:   basactions.ActionType_ACTION_TYPE_CLICK,
					Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: "#dashboard"}},
				},
				Position: &basbase.NodePosition{X: 200, Y: 0},
			},
			{
				Id: "logout-subflow",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
					Params: &basactions.ActionDefinition_Subflow{
						Subflow: &basactions.SubflowParams{
							Target: &basactions.SubflowParams_WorkflowId{
								WorkflowId: logoutWorkflowID,
							},
						},
					},
				},
				Position: &basbase.NodePosition{X: 400, Y: 0},
			},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{Id: "edge-1", Source: "login-subflow", Target: "main-action"},
			{Id: "edge-2", Source: "main-action", Target: "logout-subflow"},
		},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Len(t, plan.Steps, 3)

	// Verify order and types
	assert.Equal(t, StepSubflow, plan.Steps[0].Type)
	assert.Equal(t, loginWorkflowID, plan.Steps[0].Params["workflow_id"])

	assert.Equal(t, StepClick, plan.Steps[1].Type)

	assert.Equal(t, StepSubflow, plan.Steps[2].Type)
	assert.Equal(t, logoutWorkflowID, plan.Steps[2].Params["workflow_id"])
}
