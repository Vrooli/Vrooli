package backlog

import (
	"context"
	"testing"

	"swarm-manager/internal/execution"
)

func newFilerService(t *testing.T) (*Service, *FileStore) {
	t.Helper()
	store := NewFileStore(t.TempDir())
	svc, err := NewService(ServiceConfig{Store: store})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc, store
}

func TestFixDiscoveryFiler_CreatesFixItems(t *testing.T) {
	svc, store := newFilerService(t)
	filer := NewFixDiscoveryFiler(svc)

	findings := []execution.DiscoveryFinding{
		{Dimension: "tests", Status: "red", Details: "3 failing"},
		{Dimension: "rules", Status: "yellow", Details: "2 warnings"},
	}
	created, err := filer.FileRemediation(context.Background(), "demo", findings)
	if err != nil {
		t.Fatalf("FileRemediation: %v", err)
	}
	if created != 2 {
		t.Fatalf("created = %d, want 2", created)
	}

	item, err := store.LoadItem(KindFix, "fix-discovery-demo-tests")
	if err != nil {
		t.Fatalf("expected fix item to exist: %v", err)
	}
	if item.Kind != KindFix {
		t.Errorf("kind = %s, want fix", item.Kind)
	}
	if len(item.AcceptanceAllow) != 1 || item.AcceptanceAllow[0] != "scenarios/demo/**" {
		t.Errorf("acceptance_allow = %v, want [scenarios/demo/**]", item.AcceptanceAllow)
	}
	if !containsTag(item.Tags, fixDiscoveryTag) {
		t.Errorf("tags = %v, want to contain %q", item.Tags, fixDiscoveryTag)
	}
	if item.CreatedBy == nil || item.CreatedBy.Source != fixDiscoveryTag {
		t.Errorf("created_by = %+v, want source %q", item.CreatedBy, fixDiscoveryTag)
	}
}

func TestFixDiscoveryFiler_Idempotent(t *testing.T) {
	svc, _ := newFilerService(t)
	filer := NewFixDiscoveryFiler(svc)

	findings := []execution.DiscoveryFinding{{Dimension: "tests", Status: "red"}}
	if _, err := filer.FileRemediation(context.Background(), "demo", findings); err != nil {
		t.Fatalf("first FileRemediation: %v", err)
	}
	// Second run with the same finding must not duplicate or error.
	created, err := filer.FileRemediation(context.Background(), "demo", findings)
	if err != nil {
		t.Fatalf("second FileRemediation: %v", err)
	}
	if created != 0 {
		t.Errorf("second run created = %d, want 0 (idempotent)", created)
	}
}

func containsTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}
