package hub_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"notification-hub/internal/hub"
)

// readySpec is a decision that needs no improvement: a question, two choices,
// a recommendation with a reason, one plain sentence of situation, and a packet.
func readySpec(now time.Time, key, question string) hub.AskSpec {
	return hub.AskSpec{
		Recipient:            "alice",
		Question:             question,
		Options:              []hub.AskOption{{Key: "yes", Label: "Yes, do it"}, {Key: "no", Label: "No, leave it"}},
		Recommended:          "yes",
		RecommendationReason: "It removes the repeated confirmation without losing the guard.",
		BigPicture:           "Two checks ask the same run the same question every day.",
		Packet:               "/work/" + key + "/BRIEF.md",
		Deadline:             now.Add(24 * time.Hour),
		SensitivityLabel:     "private",
		IdempotencyKey:       key,
		Source:               "agent-manager",
		SourceEventType:      "agent_manager.effort.decision_requested.v1",
	}
}

func TestReadinessIsStoredAndListed(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()

	weak := l.ask(raise{state: hub.AskPending})
	stored, err := l.service.GetAsk(ctx, weak.ID)
	require.NoError(t, err)
	require.NotEmpty(t, stored.QualityGrade, "a stored ask carries its grade")
	require.NotEmpty(t, stored.QualityReport)
	require.Contains(t, stored.QualityReport, "missing_big_picture")
	require.NotEmpty(t, stored.QualitySummary, "a list row gets one plain line")
	require.NotContains(t, stored.QualitySummary, `"findings"`, "the summary is prose, not JSON")

	page, err := l.service.ListAsksPage(ctx, "alice", hub.AskListOptions{})
	require.NoError(t, err)
	require.Equal(t, stored.QualityGrade, page.Asks[0].QualityGrade, "a list row carries the grade")
	require.Equal(t, stored.QualitySummary, page.Asks[0].QualitySummary, "a list row carries the summary")
}

func TestAWellFramedDecisionHasNoCoverFindings(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	spec := readySpec(listNow, "ready-1", "Restructure both deletion floors?")
	ask, _, err := l.service.AskWithSpec(ctx, spec)
	require.NoError(t, err)

	stored, err := l.service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.NotEqual(t, "refused", stored.QualityGrade, stored.QualityReport)
	for _, code := range []string{"no_question", "missing_big_picture", "missing_recommend_reason", "packet_missing", "no_options"} {
		require.NotContains(t, stored.QualityReport, code, "a well-framed ask has no cover finding")
	}
}

func TestBackfillGradesAnAskWrittenWithoutReadiness(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	spec := readySpec(listNow, "backfill-1", "Restructure both deletion floors?")
	ask, _, err := l.service.AskWithSpec(ctx, spec)
	require.NoError(t, err)
	_, err = l.db.ExecContext(ctx, `UPDATE asks SET quality_grade='', quality_report='' WHERE id=?`, ask.ID)
	require.NoError(t, err)

	graded, err := l.service.BackfillQuality(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, graded)

	stored, err := l.service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.NotEmpty(t, stored.QualityGrade, "the backfill stored a grade")
	require.NotEqual(t, "refused", stored.QualityGrade)
}

func TestRelatedFindsSimilarOpenDecisions(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	first, _, err := l.service.AskWithSpec(ctx, readySpec(listNow, "rel-1", "Should the agent delete the staging database?"))
	require.NoError(t, err)
	second, _, err := l.service.AskWithSpec(ctx, readySpec(listNow, "rel-2", "Should the agent be allowed to drop the staging DB?"))
	require.NoError(t, err)
	_, _, err = l.service.AskWithSpec(ctx, readySpec(listNow, "rel-3", "Which logo variant should the landing page use?"))
	require.NoError(t, err)

	stored, err := l.service.GetAsk(ctx, first.ID)
	require.NoError(t, err)
	require.NotEmpty(t, stored.Related)
	require.Equal(t, second.ID, stored.Related[0].AskID)
	require.Equal(t, "lexical", stored.Related[0].Basis)
	for _, match := range stored.Related {
		require.NotEmpty(t, match.Question)
	}
	require.NotEqual(t, first.ID, stored.Related[0].AskID, "an ask is never related to itself")
}

func TestRelatedNeverCrossesTheRecipient(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	alice, _, err := l.service.AskWithSpec(ctx, readySpec(listNow, "scope-a", "Should the agent delete the staging database?"))
	require.NoError(t, err)
	bobSpec := readySpec(listNow, "scope-b", "Should the agent delete the staging database?")
	bobSpec.Recipient = "bob"
	bob, _, err := l.service.AskWithSpec(ctx, bobSpec)
	require.NoError(t, err)

	stored, err := l.service.GetAsk(ctx, alice.ID)
	require.NoError(t, err)
	for _, match := range stored.Related {
		require.NotEqual(t, bob.ID, match.AskID, "another recipient's ask is never a suggestion")
	}
}

func TestDeclaredFamilyOutranksSimilarity(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	first := readySpec(listNow, "fam-1", "Restructure the declared-paths floor?")
	first.DecisionFamily, first.DecisionCause = "guard-friction", "The floors ask twice"
	ask, _, err := l.service.AskWithSpec(ctx, first)
	require.NoError(t, err)
	second := readySpec(listNow, "fam-2", "A differently worded question entirely?")
	second.DecisionFamily, second.DecisionCause = "guard-friction", "The floors ask twice"
	_, _, err = l.service.AskWithSpec(ctx, second)
	require.NoError(t, err)

	stored, err := l.service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.NotEmpty(t, stored.Related)
	require.Equal(t, "family", stored.Related[0].Basis)
	require.Equal(t, float64(1), stored.Related[0].Score)
}

func TestFollowthroughReportsAndProjects(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	ask := l.ask(raise{state: hub.AskPending})
	require.NoError(t, l.service.Answer(ctx, ask.ID, "re-aim", "alice"))

	empty, err := l.service.ReadFollowthrough(ctx, ask.ID)
	require.NoError(t, err)
	require.Nil(t, empty, "no report is the honest empty state")

	report, err := l.service.RecordFollowthrough(ctx, ask.ID, hub.Followthrough{
		State: "running", Owner: "scenario:agent-manager", RunRef: "run-1", BoundSeconds: 3600,
		Summary: "The guard owner is rewriting both floors.",
	})
	require.NoError(t, err)
	require.NotEmpty(t, report.ReportedAt)

	stored, err := l.service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.Followthrough)
	require.Equal(t, "running", stored.Followthrough.State)
	require.Equal(t, 3600, stored.Followthrough.BoundSeconds)

	_, err = l.service.RecordFollowthrough(ctx, ask.ID, hub.Followthrough{State: "sideways"})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	require.True(t, strings.Contains(err.Error(), "queued"), "the refusal names the states")
}
