package agenttranscript

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// This local fixture exercises the bytes that an archive payload would carry:
// native JSONL stays byte-identical across disk serialization, prefix hashing
// remains stable when a tail is appended, and a same-length edit is detected.
// It does not send context or establish a remote-node round trip.
func TestAntigravityNativePayloadSerializationAndIntegrity(t *testing.T) {
	records := []map[string]any{
		{"step_index": 1, "type": "USER_INPUT", "source": "USER_EXPLICIT", "created_at": "2026-11-24T10:00:00Z", "content": "sanitized payload fixture"},
		{"step_index": 2, "type": "PLANNER_RESPONSE", "source": "MODEL", "created_at": "2026-11-24T10:00:01Z", "content": "sanitized response fixture"},
	}
	var payload bytes.Buffer
	for _, record := range records {
		line, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		payload.Write(line)
		payload.WriteByte('\n')
	}
	original := append([]byte(nil), payload.Bytes()...)
	originalDigest := SHA256Bytes(original)
	archive := filepath.Join(t.TempDir(), "transcript_full.jsonl")
	if err := os.WriteFile(archive, original, 0o600); err != nil {
		t.Fatal(err)
	}
	readback, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readback, original) || SHA256Bytes(readback) != originalDigest {
		t.Fatal("serialized native payload changed bytes or digest on local readback")
	}
	var messages []Message
	for _, line := range bytes.Split(bytes.TrimSpace(readback), []byte("\n")) {
		messages = append(messages, ParseMessages("antigravity", line)...)
	}
	if len(messages) != 2 || messages[0].Role != "user" || messages[1].Role != "assistant" {
		t.Fatalf("round-tripped sanitized payload decoded as %+v", messages)
	}

	prefixDigest, err := SHA256FilePrefixExact(archive, int64(len(original)))
	if err != nil || prefixDigest != originalDigest {
		t.Fatalf("initial exact prefix digest = %q, err=%v", prefixDigest, err)
	}
	withTail := append(append([]byte(nil), original...), []byte("{\"step_index\":3,\"type\":\"CHECKPOINT\"}\n")...)
	if err := os.WriteFile(archive, withTail, 0o600); err != nil {
		t.Fatal(err)
	}
	prefixDigest, err = SHA256FilePrefixExact(archive, int64(len(original)))
	if err != nil || prefixDigest != originalDigest {
		t.Fatalf("append changed verified prefix: digest=%q err=%v", prefixDigest, err)
	}
	mutated := append([]byte(nil), withTail...)
	mutated[bytes.Index(mutated, []byte("sanitized response fixture"))] ^= 1
	if err := os.WriteFile(archive, mutated, 0o600); err != nil {
		t.Fatal(err)
	}
	prefixDigest, err = SHA256FilePrefixExact(archive, int64(len(original)))
	if err != nil {
		t.Fatal(err)
	}
	if prefixDigest == originalDigest {
		t.Fatal("same-length edit escaped native payload prefix integrity check")
	}
}
