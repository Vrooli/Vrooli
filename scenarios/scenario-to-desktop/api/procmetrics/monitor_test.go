package procmetrics

import (
	"log/slog"
	"testing"
	"time"
)

func TestCollectTreeSampleAggregatesCurrentResourcesAndUniqueProcesses(t *testing.T) {
	monitor := NewDefaultMonitorWithTree(nil, nil, nil, slog.Default())
	monitor.tree = &sequenceTreeReader{samples: [][]ProcessInfo{
		{
			{PID: 10, Role: RoleElectronMain, CPUJiffies: 100, RSSBytes: 100, Threads: 2},
			{PID: 11, Role: RoleUnknown, CPUJiffies: 20, RSSBytes: 25, Threads: 1},
		},
		{
			{PID: 10, Role: RoleElectronMain, CPUJiffies: 120, RSSBytes: 140, Threads: 3},
			{PID: 11, Role: RoleUnknown, CPUJiffies: 30, RSSBytes: 35, Threads: 2},
		},
	}}
	monitor.report.ProcessTree = newProcessTreeReport(true)
	monitor.collectTreeSample(1)
	monitor.treeAt = time.Now().Add(-time.Second)
	monitor.collectTreeSample(1)

	main := monitor.report.ProcessTree.Roles[RoleElectronMain]
	if main.ProcessCount != 1 || main.RSSBytes != 140 || main.PeakRSSBytes != 140 || main.Threads != 3 {
		t.Fatalf("main summary = %+v; expected unique count and current resources", main)
	}
	if main.SampleCount != 2 || main.CPUPercent <= 0 || main.PeakCPU <= 0 {
		t.Fatalf("main timing summary = %+v", main)
	}
	unknown := monitor.report.ProcessTree.Roles[RoleUnknown]
	if unknown.ProcessCount != 1 || unknown.RSSBytes != 35 {
		t.Fatalf("unknown role should remain explicit and attributed: %+v", unknown)
	}
}

type sequenceTreeReader struct {
	samples [][]ProcessInfo
	index   int
}

func (r *sequenceTreeReader) ProcessTree(int) ([]ProcessInfo, error) {
	if r.index >= len(r.samples) {
		return r.samples[len(r.samples)-1], nil
	}
	sample := r.samples[r.index]
	r.index++
	return sample, nil
}
