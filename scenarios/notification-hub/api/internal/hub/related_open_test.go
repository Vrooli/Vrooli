package hub_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hub "notification-hub/internal/hub"
)

func TestRelatedOpenDecisionAdviceRespectsCreatorAndDoesNotMerge(t *testing.T) {
	service, _ := testService(t)
	now := time.Now().UTC()
	first := decisionSpec(now, "related-first")
	first.CreatedBy = "run:one"
	first.VerifiedProducer = true
	first.CreatedConversation = "thread:one"
	first.Question = "Should we back up private records to the external drive?"
	first.SubjectKey = "backup-records"
	one, err := service.AskDecision(context.Background(), first, "")
	require.NoError(t, err)

	second := first
	second.IdempotencyKey = "related-second"
	second.SubjectKey = "another-backup"
	second.Question = "Should we back up private records to the external disk?"
	checked, err := service.CheckAsk(context.Background(), second, hub.CheckInput{})
	require.NoError(t, err)
	require.Contains(t, findWarning(t, checked.Warnings, hub.WarnRelatedOpen).Message, one.Ask.ID)
	two, err := service.AskDecision(context.Background(), second, "")
	require.NoError(t, err)
	require.NotEqual(t, one.Ask.ID, two.Ask.ID)
	require.Contains(t, findWarning(t, two.Warnings, hub.WarnRelatedOpen).Message, one.Ask.ID)

	second.CreatedBy = "run:other"
	second.CreatedConversation = "thread:other"
	checked, err = service.CheckAsk(context.Background(), second, hub.CheckInput{})
	require.NoError(t, err)
	require.NotContains(t, warningCodes(checked.Warnings), hub.WarnRelatedOpen)
	second.CreatedConversation = "thread:one"
	checked, err = service.CheckAsk(context.Background(), second, hub.CheckInput{})
	require.NoError(t, err)
	require.Contains(t, findWarning(t, checked.Warnings, hub.WarnRelatedOpen).Message, one.Ask.ID)
	second.CreatedBy = ""
	checked, err = service.CheckAsk(context.Background(), second, hub.CheckInput{})
	require.NoError(t, err)
	require.NotContains(t, warningCodes(checked.Warnings), hub.WarnRelatedOpen)
	second.CreatedBy = "run:one"
	second.VerifiedProducer = false
	checked, err = service.CheckAsk(context.Background(), second, hub.CheckInput{})
	require.NoError(t, err)
	require.NotContains(t, warningCodes(checked.Warnings), hub.WarnRelatedOpen)
}

func TestRelatedOpenCrossConversationSignalDoesNotDiscloseAsk(t *testing.T) {
	service, _ := testService(t)
	first := decisionSpec(time.Now().UTC(), "cross-first")
	first.Source, first.SourceEventType = "", ""
	first.CreatedBy, first.CreatedConversation, first.CreatedSubject = "run:one", "thread:one", "owner-a"
	first.SubjectKey = "private-backup-key"
	first.Question = "Should we back up private records to the external drive?"
	one, err := service.AskDecision(context.Background(), first, "")
	require.NoError(t, err)
	second := first
	second.IdempotencyKey, second.SubjectKey = "cross-second", "different-key"
	second.CreatedBy, second.CreatedConversation = "run:two", "thread:two"
	second.Question = "Should we back up private records to the external disk?"
	checked, err := service.CheckAsk(context.Background(), second, hub.CheckInput{})
	require.NoError(t, err)
	message := findWarning(t, checked.Warnings, hub.WarnRelatedOpen).Message
	require.NotContains(t, message, one.Ask.ID)
	require.NotContains(t, message, first.SubjectKey)
	require.NotContains(t, message, first.Question)
	second.CreatedSubject = "owner-b"
	checked, err = service.CheckAsk(context.Background(), second, hub.CheckInput{})
	require.NoError(t, err)
	require.NotContains(t, warningCodes(checked.Warnings), hub.WarnRelatedOpen)
	second.CreatedSubject = ""
	checked, err = service.CheckAsk(context.Background(), second, hub.CheckInput{})
	require.NoError(t, err)
	require.NotContains(t, warningCodes(checked.Warnings), hub.WarnRelatedOpen)
	second.CreatedSubject = "owner-a"
	_, err = service.AskDecision(context.Background(), second, "")
	require.ErrorIs(t, err, hub.ErrRelatedDecision)
	require.NotContains(t, err.Error(), first.SubjectKey)
	require.NotContains(t, err.Error(), one.Ask.ID)
	open, err := service.ListAsks(context.Background(), "alice", true, 20)
	require.NoError(t, err)
	require.Len(t, open, 1, "the refused collision must not create another ask")
	second.AllowRelated = true
	_, err = service.AskDecision(context.Background(), second, "")
	require.NoError(t, err)
}

func TestOwnerRelatedAdviceIncludesOtherSources(t *testing.T) {
	service, _ := testService(t)
	first := decisionSpec(time.Now().UTC(), "owner-cross-source")
	first.Question = "Should we back up private records to the external drive?"
	one, err := service.AskDecision(context.Background(), first, "")
	require.NoError(t, err)
	second := first
	second.IdempotencyKey, second.Source, second.CreatedBy = "owner-check", "", "owner:alice"
	second.Question = "Should we back up private records to the external disk?"
	checked, err := service.CheckAsk(context.Background(), second, hub.CheckInput{})
	require.NoError(t, err)
	require.Contains(t, findWarning(t, checked.Warnings, hub.WarnRelatedOpen).Message, one.Ask.ID)
}

func TestConcurrentRelatedAsksDoNotBothCreate(t *testing.T) {
	service, _ := testService(t)
	first := decisionSpec(time.Now().UTC(), "concurrent-first")
	first.Source, first.SourceEventType = "", ""
	first.CreatedBy, first.CreatedConversation, first.CreatedSubject = "run:one", "thread:one", "owner-a"
	first.SubjectKey = "concurrent-first"
	first.Question = "Should we back up private records to the external drive?"
	second := first
	second.IdempotencyKey, second.SubjectKey = "concurrent-second", "concurrent-second"
	second.CreatedBy, second.CreatedConversation = "run:two", "thread:two"
	second.Question = "Should we back up private records to the external disk?"
	type result struct {
		ask hub.AskResult
		err error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	for _, spec := range []hub.AskSpec{first, second} {
		go func(spec hub.AskSpec) {
			<-start
			ask, err := service.AskDecision(context.Background(), spec, "")
			results <- result{ask, err}
		}(spec)
	}
	close(start)
	a, b := <-results, <-results
	if a.err == nil {
		require.ErrorIs(t, b.err, hub.ErrRelatedDecision)
	} else {
		require.ErrorIs(t, a.err, hub.ErrRelatedDecision)
		require.NoError(t, b.err)
	}
	// A sender who has reviewed the collision can affirm a distinct decision.
	second.AllowRelated = true
	_, err := service.AskDecision(context.Background(), second, "")
	require.NoError(t, err)
}

func TestUnkeyedRelatedAskIsRefusedBeforeCreation(t *testing.T) {
	service, _ := testService(t)
	first := decisionSpec(time.Now().UTC(), "unkeyed-first")
	first.Source, first.SourceEventType = "", ""
	first.CreatedBy, first.CreatedConversation, first.CreatedSubject = "run:one", "thread:one", "owner-a"
	first.Question = "Should we back up private records to the external drive?"
	_, err := service.AskDecision(context.Background(), first, "")
	require.NoError(t, err)
	second := first
	second.IdempotencyKey = "unkeyed-second"
	second.CreatedBy, second.CreatedConversation = "run:two", "thread:two"
	second.Question = "Should we back up private records to the external disk?"
	_, err = service.AskDecision(context.Background(), second, "")
	require.ErrorIs(t, err, hub.ErrRelatedDecision)
	open, err := service.ListAsks(context.Background(), "alice", true, 20)
	require.NoError(t, err)
	require.Len(t, open, 1)
}
