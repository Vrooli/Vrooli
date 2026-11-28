package agenttranscript

import "testing"

func TestParseCodexRolloutSessionMeta(t *testing.T) {
	meta, ok := ParseCodexRolloutSessionMeta([]byte(`{"type":"session_meta","payload":{"id":"thread-1","session_id":"root-1","cwd":"/workspace","forked_from_id":"parent-1","thread_source":"subagent","source":{"kind":"agent"}}}`))
	if !ok {
		t.Fatal("valid session_meta was not parsed")
	}
	if meta.ID != "thread-1" || meta.SessionID != "root-1" || meta.CWD != "/workspace" || meta.ForkedFromID != "parent-1" || meta.ThreadSource != "subagent" || string(meta.Source) != `{"kind":"agent"}` {
		t.Fatalf("meta=%+v source=%s", meta, meta.Source)
	}
	if _, ok := ParseCodexRolloutSessionMeta([]byte(`{"type":"turn_context","payload":{"id":"thread-1"}}`)); ok {
		t.Fatal("non-session_meta record was accepted")
	}
	if _, ok := ParseCodexRolloutSessionMeta([]byte(`not-json`)); ok {
		t.Fatal("malformed record was accepted")
	}
}
