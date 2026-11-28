package conversations_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/operatordecision"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"notification-hub/internal/hub"
	"notification-hub/internal/integrations"
)

// A scenario's decision can carry an answer form: its page's group switches
// fill a multi_choice question, and the groups the operator chose come back on
// the resolution it verifies against the Hub's own record.
func TestAScenarioDecisionCarriesAFormAndGetsTheGroupsTheOperatorChose(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	const kind = "storage-manager.cleanup.decision_requested.v1"
	facts := operatordecision.Facts{
		Question: "Delete the groups of leftover test copies you switch on?",
		Options:  []operatordecision.Option{{Key: "delete", Label: "Delete the groups I switch on"}, {Key: "keep", Label: "Keep everything"}},
		Questions: []operatordecision.Question{{
			ID: "groups", Type: operatordecision.QuestionMultiChoice, Prompt: "Which groups may go?",
			Choices:     []operatordecision.Choice{{Key: "ledger", Label: "Ledger copies"}, {Key: "hub", Label: "Hub copies"}, {Key: "treasury", Label: "Treasury copies"}},
			Recommended: []string{"ledger", "hub", "treasury"},
		}},
		SubjectKey: "cleanup:copies",
	}
	require.NoError(t, facts.Validate())
	// The payload reaches the Hub the way the event bus carries it: as a
	// protobuf Struct, then JSON.
	event := operatordecision.BuildEventTyped("storage-manager", kind, time.Now(), facts)
	wire, err := structpb.NewStruct(event.Payload)
	require.NoError(t, err, "the event bus must accept the payload")
	payload, err := protojson.Marshal(wire)
	require.NoError(t, err)
	spec, err := integrations.DecisionAskSpec("alice", "storage-manager", kind, "evt-cleanup-1", payload, time.Now().UTC())
	require.NoError(t, err)
	result, err := f.service.AskDecision(ctx, spec, "")
	require.NoError(t, err)
	for _, warning := range result.Warnings {
		require.NotEqual(t, hub.WarnQuestionInvalid, warning.Code, warning.Message)
	}
	asked, err := f.service.GetAsk(ctx, result.Ask.ID)
	require.NoError(t, err)
	require.Len(t, asked.Questions, 1)
	require.Equal(t, "groups", asked.Questions[0].ID)
	require.NotNil(t, asked.Questions[0].Recommended)
	require.Equal(t, []string{"ledger", "hub", "treasury"}, asked.Questions[0].Recommended.Selected, "the recommendation survives the bus")

	_, err = f.client.Answer(ctx, reading("owner", &v1.AnswerRequest{
		AskId: result.Ask.ID, Answer: "delete",
		Responses: []*v1.AskQuestionResponse{{QuestionId: "groups", Selected: []string{"ledger", "treasury"}}},
	}))
	require.NoError(t, err)
	record, err := f.service.GetAsk(ctx, result.Ask.ID)
	require.NoError(t, err)
	raw, err := json.Marshal(hub.AskResolvedEvent(record).Payload)
	require.NoError(t, err)
	var claimed operatordecision.DecisionResolution
	require.NoError(t, json.Unmarshal(raw, &claimed))

	response, err := f.client.GetAsk(ctx, reading("bearer:svc-storage", &v1.GetAskRequest{AskId: result.Ask.ID}))
	require.NoError(t, err)
	verified, err := operatordecision.VerifiedResolution(claimed, askreader.AskFromProto(response.Msg.GetAsk()))
	require.NoError(t, err)
	chosen, ok := verified.Response("groups")
	require.True(t, ok, "the verified resolution carries the form answers from the Hub's record")
	require.Equal(t, []string{"ledger", "treasury"}, chosen.Selected)
	require.Contains(t, verified.AnswerSummary, "Ledger copies")
}
