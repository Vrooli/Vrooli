package workflows

import (
	"testing"

	"github.com/stretchr/testify/require"
	actions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	workflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/proto"
)

func TestPromotionPreservesAcceptanceContract(t *testing.T) { // [REQ:BAS-AGENT-REUSE]
	before := &workflows.WorkflowDefinitionV2{Nodes: []*workflows.WorkflowNodeV2{{Id: "verify", Action: &actions.ActionDefinition{Type: actions.ActionType_ACTION_TYPE_ASSERT}}}}
	require.NoError(t, validatePromotionChecks(nil, before))
	require.Error(t, validatePromotionChecks(nil, &workflows.WorkflowDefinitionV2{}))
	after := proto.Clone(before).(*workflows.WorkflowDefinitionV2)
	after.Nodes[0].Id = "replacement"
	require.ErrorContains(t, validatePromotionChecks(before, after), "preserve assertion")
	after = proto.Clone(before).(*workflows.WorkflowDefinitionV2)
	after.Metadata = &workflows.WorkflowMetadataV2{Name: proto.String("changed")}
	require.ErrorContains(t, validatePromotionChecks(before, after), "metadata")
	after = proto.Clone(before).(*workflows.WorkflowDefinitionV2)
	after.Nodes = append(after.Nodes, &workflows.WorkflowNodeV2{Id: "repaired-navigation", Action: &actions.ActionDefinition{Type: actions.ActionType_ACTION_TYPE_NAVIGATE}})
	require.Error(t, validatePromotionChecks(before, after))
}
