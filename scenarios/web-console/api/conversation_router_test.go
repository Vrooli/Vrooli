package main

import (
	"context"
	"errors"
	"testing"

	"web-console/integrations/audiotools"
	"web-console/internal/audioports"
)

type testSummarizer struct {
	out audioports.SummarizeOutput
	err error
}

func (s testSummarizer) Summarize(context.Context, audioports.SummarizeInput) (audioports.SummarizeOutput, error) {
	return s.out, s.err
}

// conversation_router_test.go: post-audio-tools-adoption.
//
// The pre-adoption tests in this file (TestAsyncSummarizeAndNotify_*,
// TestHandleSummarizeEvent_*) exercised a contract that has moved
// entirely into audio-tools: the in-process TTS cache + Ollama-direct
// summarizer + cache-invalidation-on-summarize sequence are all owned
// by the audio-tools scenario now (see scenarios/audio-tools/api/...).
//
// The remaining tests here lock in the conversation-routing failure modes
// that ARE still web-console's job: refusing to publish events when the
// owning terminal session is missing or unknown. Cache + summarize
// behaviour is tested end-to-end via audio-tools' own test suite.

func TestAppendConversationEvent_MissingSession(t *testing.T) {
	srv := newFakeTestServer()
	result := srv.AppendAssistant("hello", "", "test")
	if result.Appended {
		t.Error("expected failure for empty session")
	}
	if result.Code != "conversation_target_missing" {
		t.Errorf("expected code conversation_target_missing, got %s", result.Code)
	}
}

func TestAppendConversationEvent_UnknownSession(t *testing.T) {
	srv := newFakeTestServer()
	result := srv.AppendAssistant("hello", "nonexistent", "test")
	if result.Appended {
		t.Error("expected failure for unknown session")
	}
}

func TestAppendUserConversationEvent_MissingSession(t *testing.T) {
	srv := newFakeTestServer()
	result := srv.AppendUser("hello", "", "test")
	if result.Appended {
		t.Error("expected failure for empty session")
	}
}

func TestAppendUserConversationEvent_UnknownSession(t *testing.T) {
	srv := newFakeTestServer()
	result := srv.AppendUser("hello", "nonexistent", "test")
	if result.Appended {
		t.Error("expected failure for unknown session")
	}
}

func TestSummarizeHelpersAndAsyncPaths(t *testing.T) {
	if got := splitIntoSpeechParagraphs(" one \n\n\n two "); len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("paragraphs = %#v", got)
	}
	if splitIntoSpeechParagraphs(" \n\n") != nil {
		t.Fatal("blank summary should produce no paragraphs")
	}
	for _, err := range []error{audiotools.ErrTimeout, audiotools.ErrUnavailable, audiotools.ErrFailedPrecondition, errors.New("detail")} {
		if got := summarizeErrorMessage(err); got == "" {
			t.Fatalf("empty summary error for %v", err)
		}
	}
	if summarizeErrorMessage(nil) != "" {
		t.Fatal("nil summary error was not empty")
	}

	srv := newFakeTestServer()
	event := ConversationEvent{ID: "event-1", Text: "this is long enough"}
	srv.summarizer = testSummarizer{out: audioports.SummarizeOutput{Text: "first\n\nsecond", Latency: 5}}
	srv.asyncSummarizeAndNotify(event, "missing-session", SummarizeAutoPolicy{Enabled: false, CharThreshold: 1})
	srv.asyncSummarizeAndNotify(event, "missing-session", SummarizeAutoPolicy{Enabled: true, CharThreshold: 100})
	srv.asyncSummarizeAndNotify(event, "missing-session", SummarizeAutoPolicy{Enabled: true, CharThreshold: 1})
	srv.summarizer = testSummarizer{err: audiotools.ErrTimeout}
	srv.asyncSummarizeAndNotify(event, "missing-session", SummarizeAutoPolicy{Enabled: true, CharThreshold: 1})
	if result := conversationAppendFailure("code", "reason", "source", "session"); result.Appended || result.Code != "code" {
		t.Fatalf("failure result = %#v", result)
	}
}
