package storeunlock

import "testing"

func TestOwnsOnlyThisNodesStorePassphrase(t *testing.T) {
	for _, tc := range []struct {
		logicalID, field string
		want             bool
	}{
		{"vrooli-bridge/node-credential-store/451ea636-a80f-4080-82b7-fa65d0e3289a", "passphrase", true},
		{"vrooli-bridge/node-credential-store/451ea636", "api-key", false},
		{"vrooli/openrouter", "passphrase", false},
		{"", "passphrase", false},
	} {
		if got := Owns(tc.logicalID, tc.field); got != tc.want {
			t.Errorf("Owns(%q, %q) = %v, want %v", tc.logicalID, tc.field, got, tc.want)
		}
	}
}

// The holder keeps the passphrase for the agent's lifetime (a job-scoped
// ephemeral value is taken once) and hands out copies, so a caller can never
// mutate or retain the held bytes.
func TestHolderKeepsAndCopiesThePassphrase(t *testing.T) {
	var holder Holder
	if _, ok := holder.Get(); ok || holder.Held() {
		t.Fatal("an empty holder reported a passphrase")
	}
	holder.Put([]byte("first"))
	got, ok := holder.Get()
	if !ok || string(got) != "first" {
		t.Fatalf("Get() = %q, %v", got, ok)
	}
	got[0] = 'X'
	again, _ := holder.Get()
	if string(again) != "first" {
		t.Fatalf("a caller's copy changed the held value: %q", again)
	}
	holder.Put([]byte("rotated"))
	if value, _ := holder.Get(); string(value) != "rotated" {
		t.Fatalf("after rotation Get() = %q", value)
	}
	holder.Clear()
	if holder.Held() {
		t.Fatal("Clear left a passphrase held")
	}
}
