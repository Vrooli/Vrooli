package integrations

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDecisionAskSpecCarriesContextAndUsesProducerNoteFallback(t *testing.T) {
	base := map[string]any{
		"question":    "Apply this change?",
		"options":     []map[string]string{{"key": "apply", "label": "Apply"}, {"key": "reject", "label": "Reject"}},
		"recommended": "apply",
		"big_picture": "Purpose and impact",
		"note":        "Requester evidence",
	}
	encoded, err := json.Marshal(base)
	require.NoError(t, err)
	spec, err := DecisionAskSpec("operator", "agent-manager", "agent_manager.run.decision_requested.v1", "event-1", encoded, time.Now().UTC())
	require.NoError(t, err)
	require.Equal(t, "Purpose and impact", spec.BigPicture)
	require.Equal(t, "Requester evidence", spec.TechnicalDetails)

	base["technical_details"] = "Explicit evidence"
	encoded, err = json.Marshal(base)
	require.NoError(t, err)
	spec, err = DecisionAskSpec("operator", "agent-manager", "agent_manager.run.decision_requested.v1", "event-2", encoded, time.Now().UTC())
	require.NoError(t, err)
	require.Equal(t, "Explicit evidence", spec.TechnicalDetails, "explicit details take precedence over the legacy note")
}

func TestDecisionAskSpecRejectsInteractiveReviewOnlyPayload(t *testing.T) {
	encoded, err := json.Marshal(map[string]any{
		"subject":  "Authorization requested",
		"question": "Review this request in the web console?",
		"options":  []map[string]string{{"key": "review", "label": "Review in web console"}},
	})
	require.NoError(t, err)
	_, err = DecisionAskSpecWithDefaults("operator", "agent-manager", "agent_manager.interactive_authorization.decision_requested.v1", "event-interactive", encoded, time.Now().UTC(), "private", 24*time.Hour)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least two options")
}
