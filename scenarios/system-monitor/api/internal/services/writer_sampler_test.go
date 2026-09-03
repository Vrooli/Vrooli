package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriterSamplerReportsDeltaAndRate(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "payload")
	if err := os.WriteFile(path, make([]byte, 100), 0o600); err != nil {
		t.Fatal(err)
	}
	sampler := NewWriterSampler([]GovernedRoot{{ID: "test", Root: root, Mount: "/", HotWriterBytesHour: 200}})
	start := time.Unix(100, 0)
	first := sampler.Sample(context.Background(), start)[0]
	if first.Bytes != 100 || first.BytesPerHour != 0 {
		t.Fatalf("first snapshot = %#v", first)
	}
	if first.Root != root || first.RootID != "test" {
		t.Fatalf("root identity = %q/%q, want %q/test", first.Root, first.RootID, root)
	}
	if got := sampler.Sample(context.Background(), start.Add(30*time.Second)); got != nil {
		t.Fatalf("sample inside cadence = %#v, want nil", got)
	}
	if err := os.WriteFile(path, make([]byte, 300), 0o600); err != nil {
		t.Fatal(err)
	}
	second := sampler.Sample(context.Background(), start.Add(time.Hour))[0]
	if second.DeltaBytes != 200 || second.BytesPerHour != 200 || second.Hot {
		t.Fatalf("second snapshot = %#v", second)
	}
}

func TestWriterSamplerExpandsGovernedChildren(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"b", "a"} {
		child := filepath.Join(root, name)
		if err := os.Mkdir(child, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(child, "payload"), make([]byte, 10), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	snapshots := NewWriterSampler([]GovernedRoot{{
		ID: "go-work-dirs", Root: root, Mount: "/", ExpandChildren: true,
	}}).Sample(context.Background(), time.Unix(100, 0))
	if len(snapshots) != 2 {
		t.Fatalf("expanded snapshots = %d, want 2", len(snapshots))
	}
	if snapshots[0].RootID != "go-work-dirs/a" || snapshots[1].RootID != "go-work-dirs/b" {
		t.Fatalf("expanded root IDs = %q, %q; want sorted children", snapshots[0].RootID, snapshots[1].RootID)
	}
}
