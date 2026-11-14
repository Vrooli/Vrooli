package runtimeprofile

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/process"
)

func TestSelectRecordRefusesMissingAndAmbiguousComponents(t *testing.T) {
	for _, records := range [][]process.Record{nil, {{Step: "start-ui"}}, {{Step: "start-api", Status: "exited_verified"}}, {{Step: "start-api"}, {Step: "start-api"}}} {
		if _, err := selectRecord(records, "start-api"); err == nil {
			t.Fatalf("accepted ambiguous/missing records: %+v", records)
		}
	}
	got, err := selectRecord([]process.Record{{Step: "start-ui", PID: 1}, {Step: "start-api", PID: 2}}, "start-api")
	if err != nil || got.PID != 2 {
		t.Fatalf("selection=%+v err=%v", got, err)
	}
}

func TestCaptureNeverOverwritesExistingArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.pprof")
	if err := os.WriteFile(path, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(context.Background(), t.TempDir(), Request{Scenario: "web-console", Step: "start-api", Seconds: 1, Output: path}); err == nil {
		t.Fatal("existing output accepted")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "preserve" {
		t.Fatalf("existing artifact changed: %v", err)
	}
}
