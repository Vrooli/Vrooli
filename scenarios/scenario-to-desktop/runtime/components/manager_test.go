package components

import (
	"context"
	component "github.com/vrooli/vrooli/packages/desktop-component"
	"testing"
)

func TestActivationOfOldRequestCannotDiscardNewerFallback(t *testing.T) {
	m := New(t.TempDir(), "0.0.1", nil)
	row := Installation{Signed: component.SignedEntry{Entry: component.Entry{ID: "audio-tools", SHA256: "newest"}}, Generation: "newest", State: Installed, Previous: &Generation{Generation: "last-healthy"}}
	if err := m.save(map[string]Installation{"audio-tools": row}); err != nil {
		t.Fatal(err)
	}
	if err := m.CompleteActivation(context.Background(), "audio-tools", "superseded"); err == nil {
		t.Fatal("superseded activation discarded the retained healthy version")
	}
	rows, err := m.Installed(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Previous == nil || rows[0].Previous.Generation != "last-healthy" {
		t.Fatalf("fallback lost before matching readiness: %+v", rows)
	}
	if err := m.CompleteActivation(context.Background(), "audio-tools", "newest"); err != nil {
		t.Fatal(err)
	}
	rows, err = m.Installed(context.Background())
	if err != nil || rows[0].Previous != nil {
		t.Fatalf("healthy activation did not commit: %+v %v", rows, err)
	}
}
