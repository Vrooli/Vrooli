package aisearch

import (
	"prompt-manager/skills"
	"prompt-manager/store"
	"strings"
	"testing"
)

func TestComposePayloadHash_Deterministic(t *testing.T) {
	p := map[string]interface{}{
		"name":        "Foo",
		"description": "bar",
		"tags":        []string{"a", "b"},
	}
	h1 := composePayloadHash("text", p)
	h2 := composePayloadHash("text", p)
	if h1 != h2 {
		t.Fatalf("hash not deterministic: %q vs %q", h1, h2)
	}
}

func TestComposePayloadHash_PrefixIsSha256(t *testing.T) {
	h := composePayloadHash("hello", map[string]interface{}{"a": 1})
	if !strings.HasPrefix(h, "sha256:") {
		t.Fatalf("expected sha256: prefix, got %q", h)
	}
	// 8-byte hex == 16 chars, plus prefix.
	if len(h) != len("sha256:")+16 {
		t.Fatalf("expected hash length %d, got %d (%q)", len("sha256:")+16, len(h), h)
	}
}

func TestComposePayloadHash_TextChangeDetected(t *testing.T) {
	p := map[string]interface{}{"name": "Foo"}
	h1 := composePayloadHash("v1", p)
	h2 := composePayloadHash("v2", p)
	if h1 == h2 {
		t.Fatalf("expected different hashes for different text, got %q twice", h1)
	}
}

func TestComposePayloadHash_PayloadFieldChangeDetected(t *testing.T) {
	p1 := map[string]interface{}{"enabled": true}
	p2 := map[string]interface{}{"enabled": false}
	h1 := composePayloadHash("text", p1)
	h2 := composePayloadHash("text", p2)
	if h1 == h2 {
		t.Fatalf("expected different hashes for different payload, got %q twice", h1)
	}
}

func TestComposePayloadHash_ExistingHashFieldIgnored(t *testing.T) {
	p1 := map[string]interface{}{"name": "Foo"}
	p2 := map[string]interface{}{"name": "Foo", payloadHashKey: "sha256:deadbeef"}
	h1 := composePayloadHash("text", p1)
	h2 := composePayloadHash("text", p2)
	if h1 != h2 {
		t.Fatalf("expected hash to ignore payload_hash field; got %q vs %q", h1, h2)
	}
}

func TestComposePayloadHash_KeyOrderStable(t *testing.T) {
	// Map iteration order is randomized; canonicalJSON must produce stable output.
	p1 := map[string]interface{}{"a": 1, "b": 2, "c": 3}
	p2 := map[string]interface{}{"c": 3, "b": 2, "a": 1}
	h1 := composePayloadHash("t", p1)
	h2 := composePayloadHash("t", p2)
	if h1 != h2 {
		t.Fatalf("expected hash to be insensitive to map insertion order; got %q vs %q", h1, h2)
	}
}

func TestBuildSkillPayload_IncludesPayloadHash(t *testing.T) {
	m := &skills.Metadata{ID: "s1", Name: "Foo", Description: "bar", Tags: []string{"x"}, Modes: []string{"local"}}
	p := buildSkillPayload(m, "local", "embed-text")
	hash, ok := p[payloadHashKey].(string)
	if !ok || hash == "" {
		t.Fatalf("expected payload_hash string, got %#v", p[payloadHashKey])
	}
	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("expected sha256: prefix, got %q", hash)
	}
	if p["skill_id"] != "s1" {
		t.Errorf("skill_id missing/wrong: %#v", p["skill_id"])
	}
}

func TestBuildAgentPayload_IncludesPayloadHash(t *testing.T) {
	a := &store.Agent{ID: "a1", DisplayName: "Agent", Description: "desc", Status: "active", Tags: []string{"t"}}
	p := buildAgentPayload(a, "embed-text")
	if _, ok := p[payloadHashKey].(string); !ok {
		t.Fatalf("expected payload_hash, got %#v", p)
	}
}

func TestBuildTeamPayload_IncludesPayloadHash(t *testing.T) {
	tm := &store.Team{ID: "t1", DisplayName: "Team", Mission: "ship", Enabled: true}
	p := buildTeamPayload(tm, 3, "embed-text")
	if _, ok := p[payloadHashKey].(string); !ok {
		t.Fatalf("expected payload_hash, got %#v", p)
	}
	if p["member_count"] != 3 {
		t.Errorf("member_count = %#v, want 3", p["member_count"])
	}
}

func TestBuildTopicPayload_IncludesPayloadHash(t *testing.T) {
	parent := "parent-id"
	tp := &store.Topic{ID: "t1", Name: "Topic", Description: "d", Skills: []string{"s1"}, ParentTopicID: &parent}
	p := buildTopicPayload(tp, "embed-text")
	if _, ok := p[payloadHashKey].(string); !ok {
		t.Fatalf("expected payload_hash, got %#v", p)
	}
	if p["parent_topic_id"] != parent {
		t.Errorf("parent_topic_id = %#v, want %q", p["parent_topic_id"], parent)
	}
}

func TestBuildActionPayload_IncludesPayloadHash(t *testing.T) {
	a := &store.Action{ID: "a1", Name: "Run", Description: "d", Status: "ready", Tags: []string{"x"}}
	p := buildActionPayload(a, "embed-text")
	if _, ok := p[payloadHashKey].(string); !ok {
		t.Fatalf("expected payload_hash, got %#v", p)
	}
}
