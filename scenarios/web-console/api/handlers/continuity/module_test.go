package continuity

import "testing"

func TestSafeUTF8PreservesReadableTextAndReplacesMalformedBytes(t *testing.T) {
	got := safeUTF8("plan-manager\xff continuity")
	if got != "plan-manager\uFFFD continuity" {
		t.Fatalf("safeUTF8() = %q", got)
	}
}
