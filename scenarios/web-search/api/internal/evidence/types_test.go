package evidence

import "testing"

func TestValidateSpanRejectsMidRuneOffsets(t *testing.T) {
	content := []byte("a🙂b")
	if err := validateSpan(content, 2, len(content)); err == nil {
		t.Fatal("expected start inside rune to fail")
	}
	if err := validateSpan(content, 1, 2); err == nil {
		t.Fatal("expected end inside rune to fail")
	}
}

func TestContentHashStable(t *testing.T) {
	if ContentHash([]byte("same")) != ContentHash([]byte("same")) {
		t.Fatal("hash must be stable")
	}
}
