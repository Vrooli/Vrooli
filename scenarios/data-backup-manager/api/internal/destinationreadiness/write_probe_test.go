package destinationreadiness

import (
	"context"
	"os"
	"testing"
)

func TestFileWriteProberProvesReadBackAndLeavesNoArtifacts(t *testing.T) {
	dir := t.TempDir()
	got, err := NewFileWriteProber().Probe(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Ran || !got.Passed || got.Bytes != writeProbeBytes || got.Error != "" {
		t.Fatalf("probe = %+v, want a passed bounded probe", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("probe left artifacts: %v", entries)
	}
}

func TestFileWriteProberFailsCleanlyOnMissingDirectory(t *testing.T) {
	got, err := NewFileWriteProber().Probe(context.Background(), "/definitely/not/a/destination")
	if err != nil {
		t.Fatalf("probe should report failure in the result, not as a transport error: %v", err)
	}
	if !got.Ran || got.Passed || got.Error == "" {
		t.Fatalf("probe = %+v, want a failed probe with an error", got)
	}
}
