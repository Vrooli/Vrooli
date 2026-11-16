package integrations

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/operatordecision"

	"notification-hub/internal/hub"
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

// A producer's declaration travels in the payload operatordecision.Facts builds,
// and only then does the ask carry it.
func TestDecisionAskSpecCarriesTheProducersContextRequestDeclaration(t *testing.T) {
	facts := operatordecision.Facts{
		Question: "Retry the backup?", Options: []operatordecision.Option{{Key: "retry", Label: "Retry"}, {Key: "skip", Label: "Skip"}},
		Recommended: "retry", Urgency: "normal", SubjectKey: "backup-failure:vol-1",
	}
	const eventType = "data-backup-manager.backup.decision_requested.v1"
	build := func() hub.AskSpec {
		encoded, err := json.Marshal(facts.Payload())
		require.NoError(t, err)
		spec, err := DecisionAskSpec("operator", "data-backup-manager", eventType, "event-1", encoded, time.Now().UTC())
		require.NoError(t, err)
		return spec
	}
	require.False(t, build().AcceptsContextRequests, "an undeclared decision stays undeclared")
	facts.AcceptsContextRequests = true
	require.True(t, build().AcceptsContextRequests)
}

func TestDecisionAskSpecNeedsNoRecommendation(t *testing.T) {
	encoded, err := json.Marshal(map[string]any{
		"question": "Retry the backup?",
		"options":  []map[string]string{{"key": "retry", "label": "Retry"}, {"key": "skip", "label": "Skip"}},
	})
	require.NoError(t, err)
	spec, err := DecisionAskSpec("operator", "data-backup-manager", "data-backup-manager.backup.decision_requested.v1", "event-1", encoded, time.Now().UTC())
	require.NoError(t, err)
	require.Empty(t, spec.Recommended)
}

// An event cannot say who published it, so a decision event never brings files
// into the ask, whatever its payload claims. A scenario that wants files sends
// Ask as its own verified sender. Its answer form comes along: questions name
// no file the event could vouch for, and the Hub checks them as it checks a
// --form.
func TestDecisionAskSpecIgnoresFilesAPayloadClaimsButKeepsItsForm(t *testing.T) {
	encoded, err := json.Marshal(map[string]any{
		"question": "Apply this change?",
		"options":  []map[string]string{{"key": "apply", "label": "Apply"}, {"key": "reject", "label": "Reject"}},
		"attachments": []map[string]any{
			{"name": "page.html", "kind": "file", "role": "explainer", "sha256": "abc", "source_path": "/etc/passwd"},
		},
		"questions": []map[string]any{{"id": "q1", "type": "text", "prompt": "Why?"}},
	})
	require.NoError(t, err)
	spec, err := DecisionAskSpec("operator", "research-desk", "research_desk.needs_you.decision_requested.v1", "event-1", encoded, time.Now().UTC())
	require.NoError(t, err)
	require.Empty(t, spec.Attachments)
	require.Equal(t, []hub.AskQuestion{{ID: "q1", Type: "text", Prompt: "Why?"}}, spec.Questions)
	require.False(t, spec.VerifiedProducer, "an event ask is never a verified producer's")
}

func TestDecisionAskSpecCarriesTheProducersPageAndPacket(t *testing.T) {
	encoded, err := json.Marshal(map[string]any{
		"question":       "Apply this change?",
		"options":        []map[string]string{{"key": "apply", "label": "Apply"}, {"key": "reject", "label": "Reject"}},
		"explainer_html": "<!doctype html><h1>Apply?</h1>",
		"packet":         "/home/me/.vrooli/research/topics/t/work/APPLY/BRIEF.md",
	})
	require.NoError(t, err)
	spec, err := DecisionAskSpec("operator", "storage-manager", "storage-manager.deletion-grant.decision_requested.v1", "event-1", encoded, time.Now().UTC())
	require.NoError(t, err)
	require.Equal(t, "<!doctype html><h1>Apply?</h1>", spec.ExplainerHTML)
	require.Equal(t, "/home/me/.vrooli/research/topics/t/work/APPLY/BRIEF.md", spec.Packet)
}
