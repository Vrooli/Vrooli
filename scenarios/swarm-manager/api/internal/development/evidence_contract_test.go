package development

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestResolverRegistryRequiresRegisteredVersionedIdentity(t *testing.T) {
	now := time.Now().UTC()
	resolver := &ownerEvidence{revision: "product", value: Evidence{OutcomeID: "tail", Source: "testgenie.audio@v1/linux-amd64-chromium", ResolverID: "testgenie.audio", ReceiptSchema: "v1", ReceiptID: "receipt", Digest: "target", ExecutionID: "run", SubjectRevision: "product", Cohort: "linux-amd64-chromium", Passed: true, Status: "passed", ObservedAt: now, FreshUntil: now.Add(time.Hour)}}
	registry := NewResolverRegistry()
	if err := registry.Register("testgenie.audio@v1", resolver); err != nil {
		t.Fatal(err)
	}
	got, err := registry.Resolve(context.Background(), "testgenie.audio@v1/linux-amd64-chromium", "receipt")
	if err != nil || got.ResolverID != "testgenie.audio" || got.Cohort != "linux-amd64-chromium" {
		t.Fatalf("resolved=%+v err=%v", got, err)
	}
	if _, err := registry.Resolve(context.Background(), "missing.owner@v1/linux-amd64-chromium", "receipt"); !errors.Is(err, ErrEvidenceResolverUnavailable) {
		t.Fatalf("unregistered resolver err=%v", err)
	}
}

func TestResolverRegistrySelectsSubjectOwnerIndependentOfRegistrationOrder(t *testing.T) {
	first := &ownerEvidence{revision: "wrong-owner"}
	second := &ownerEvidence{revision: "audio-owner"}
	registry := NewResolverRegistry()
	if err := registry.Register("generic.first@v1", first); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterForScenario("audio.owner@v1", "audio-tools", second); err != nil {
		t.Fatal(err)
	}
	if got, err := registry.CurrentRevision(context.Background(), "audio-tools"); err != nil || got != "audio-owner" {
		t.Fatalf("owner revision=%q err=%v", got, err)
	}
	if _, err := registry.CurrentRevision(context.Background(), "unmapped"); !errors.Is(err, ErrEvidenceResolverUnavailable) {
		t.Fatalf("unmapped scenario selected a resolver: %v", err)
	}
}
