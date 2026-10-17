package retention

import (
	"context"
	"io/fs"
	"testing"
	"time"
)

type sliceWalker []FileEntry

func (w sliceWalker) Walk(_ context.Context, _ string, visit func(FileEntry) error) error {
	for _, entry := range w {
		if err := visit(entry); err != nil {
			return err
		}
	}
	return nil
}

func TestSelectFilesOrdersOldestAndHonorsBounds(t *testing.T) {
	now := time.Unix(1000, 0)
	got, err := SelectFiles(context.Background(), sliceWalker{
		{Path: "/root/new", Size: 4, ModTime: now.Add(-time.Hour), Mode: fs.ModePerm},
		{Path: "/root/old", Size: 4, ModTime: now.Add(-3 * time.Hour)},
		{Path: "/root/protected.tmp", Size: 4, ModTime: now.Add(-4 * time.Hour)},
		{Path: "/root/dir", IsDir: true, ModTime: now.Add(-5 * time.Hour)},
	}, FileSelectionConfig{Root: "/root", Now: now, MinAge: 2 * time.Hour, MaxBytes: 4, ProtectedGlobs: []string{"*.tmp"}})
	if err != nil {
		t.Fatalf("SelectFiles: %v", err)
	}
	if len(got) != 1 || got[0].Path != "/root/old" {
		t.Fatalf("selected = %#v, want only oldest eligible file", got)
	}
}

func TestSelectFilesRejectsRelativeRootAndHonorsActive(t *testing.T) {
	if _, err := SelectFiles(context.Background(), sliceWalker{}, FileSelectionConfig{Root: "relative"}); err == nil {
		t.Fatal("relative root accepted")
	}
	got, err := SelectFiles(context.Background(), sliceWalker{{Path: "/root/a", Size: 1}}, FileSelectionConfig{Root: "/root", IsActive: func(string) bool { return true }})
	if err != nil || len(got) != 0 {
		t.Fatalf("active selection = %#v, %v; want empty", got, err)
	}
}
