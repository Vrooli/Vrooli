package appctx

import "testing"

func TestDecodeIgnoresNilResultAndEmptyBody(t *testing.T) {
	if err := decode([]byte(`{"ok":true}`), nil); err != nil {
		t.Fatalf("nil result should not decode: %v", err)
	}
	var result map[string]bool
	if err := decode(nil, &result); err != nil {
		t.Fatalf("empty body should not decode: %v", err)
	}
}

func TestDecodeUnmarshalsJSON(t *testing.T) {
	var result struct {
		OK bool `json:"ok"`
	}
	if err := decode([]byte(`{"ok":true}`), &result); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !result.OK {
		t.Fatal("expected decoded true value")
	}
}
