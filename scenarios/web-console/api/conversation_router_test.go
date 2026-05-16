package main

import (
	"testing"
)

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
