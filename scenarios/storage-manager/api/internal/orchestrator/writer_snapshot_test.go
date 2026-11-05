package orchestrator

import (
	"context"
	"testing"

	"storage-manager/internal/providers"
)

// The writers view must report what the sender measured: the root's size, the
// growth over the reported window, and the mount's real used bytes.
func TestReportPressurePersistsTruthfulWriterAndMountSnapshots(t *testing.T) {
	t.Parallel()
	store, done := newStore(t)
	defer done()
	registry, err := providers.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(registry, store, nil)
	const gib = int64(1 << 30)
	signal := PressureSignal{
		SourceScenario: "system-monitor", Partition: "/", Band: BandWarning,
		UsedPercent: 80, AvailableBytes: 200 * gib, FillRateBytesPerHour: 4 * gib,
		HotWriters: []HotWriter{{Root: "/home/u/.cache/go-build", CurrentBytes: 49 * gib, BytesPerHour: 6 * gib, WindowSeconds: 1800}},
	}
	if _, err := svc.ReportPressure(context.Background(), signal); err != nil {
		t.Fatalf("ReportPressure: %v", err)
	}
	rows, err := store.ListWriterSnapshots(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	byRoot := map[string]WriterSnapshot{}
	for _, row := range rows {
		byRoot[row.Root] = row
	}
	writer := byRoot["/home/u/.cache/go-build"]
	if writer.Bytes != 49*gib || writer.BytesPerHour != 6*gib || writer.DeltaHours != 0.5 || writer.DeltaBytes != 3*gib {
		t.Fatalf("writer snapshot = %#v, want 49 GiB, 3 GiB over 0.5h at 6 GiB/h", writer)
	}
	// 80% used with 200 GiB available is a 1000 GiB disk holding 800 GiB.
	if mount := byRoot["/"]; mount.Bytes != 800*gib {
		t.Fatalf("mount used bytes = %d, want %d", mount.Bytes, 800*gib)
	}
}
