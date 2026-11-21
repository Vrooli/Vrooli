package procmetrics

import (
	"testing"
	"time"
)

func TestProcessTreeWindowAveragesCPUAndUsesFinalPSSPoint(t *testing.T) {
	start := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	monitor := NewDefaultMonitorWithTree(nil, nil, nil, nil)
	monitor.treeWindowPoints = []treeWindowPoint{
		{at: start, cpuAvailable: true, cpuPercent: 2, memoryAvailable: true, pssBytes: 100, ussBytes: 80},
		{at: start.Add(time.Second), cpuAvailable: true, cpuPercent: 4, memoryAvailable: true, pssBytes: 120, ussBytes: 90},
		{at: start.Add(2 * time.Second), cpuAvailable: true, cpuPercent: 6, memoryAvailable: true, pssBytes: 140, ussBytes: 110},
	}
	got := monitor.Window(start, start.Add(2*time.Second))
	if !got.Available || got.Samples != 3 || got.AverageCPUPercent != 4 || got.PSSBytes != 140 || got.USSBytes != 110 {
		t.Fatalf("window = %+v", got)
	}
}

func TestProcessTreeWindowRejectsSparseCoverage(t *testing.T) {
	start := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	monitor := NewDefaultMonitorWithTree(nil, nil, nil, nil)
	monitor.treeWindowPoints = []treeWindowPoint{{
		at: start.Add(time.Second), cpuAvailable: true, memoryAvailable: true,
	}}
	if got := monitor.Window(start, start.Add(time.Minute)); got.Available || got.Reason == "" {
		t.Fatalf("sparse window = %+v, want unavailable reason", got)
	}
}

func TestProcessTreeSnapshotAtReturnsNearestMemoryObservation(t *testing.T) {
	at := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	monitor := NewDefaultMonitorWithTree(nil, nil, nil, nil)
	monitor.treeWindowPoints = []treeWindowPoint{{
		at: at.Add(time.Second), memoryAvailable: true,
		roles: map[ProcessRole]RoleSummary{RoleElectronMain: {Role: RoleElectronMain, ProcessCount: 1, PSSBytes: 100}},
	}}
	report, ok := monitor.SnapshotAt(at)
	if !ok || report.Roles[RoleElectronMain].PSSBytes != 100 {
		t.Fatalf("snapshot = %+v, available=%v", report, ok)
	}
	report.Roles[RoleElectronMain] = RoleSummary{}
	if monitor.treeWindowPoints[0].roles[RoleElectronMain].PSSBytes != 100 {
		t.Fatal("snapshot shares mutable role map with monitor")
	}
}
