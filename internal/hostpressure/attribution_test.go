package hostpressure

import (
	"runtime"
	"testing"
	"time"
)

func stormSnapshot(children int) PressureSnapshot {
	list := []Process{{PID: 1, PPID: 0, Name: "init"}, {PID: 100, PPID: 1, Name: "claude"}, {PID: 200, PPID: 1, Name: "bash"}, {PID: 201, PPID: 200, Name: "sleep"}}
	for i := 0; i < children; i++ {
		list = append(list, Process{PID: int64(1000 + i), PPID: 100, Name: "sleep"})
	}
	return PressureSnapshot{CapturedAt: time.Unix(10, 0), Processes: list}
}

func TestAttributionRanksParentsByChildCount(t *testing.T) {
	if runtime.GOOS != attributionOS {
		t.Skip("attribution is read on linux only")
	}
	got := Attribution(stormSnapshot(300), nil)
	if got.State != Read || len(got.ByChildren) == 0 {
		t.Fatalf("attribution = %+v", got)
	}
	top := got.ByChildren[0]
	if top.PID != 100 || top.Name != "claude" || top.Children != 300 {
		t.Fatalf("top parent = %+v, want pid 100 claude with 300 children", top)
	}
	if got.ByChildren[1].PID != 1 || got.ByChildren[1].Children != 2 {
		t.Fatalf("second parent = %+v", got.ByChildren[1])
	}
	if len(got.ByChildren) > attributionTop {
		t.Fatalf("ranking returned %d parents, cap is %d", len(got.ByChildren), attributionTop)
	}
}

func TestAttributionReportsDeltaAgainstPrevious(t *testing.T) {
	if runtime.GOOS != attributionOS {
		t.Skip("attribution is read on linux only")
	}
	previous := stormSnapshot(20)
	got := Attribution(stormSnapshot(300), &previous)
	if got.ByDelta[0].PID != 100 || got.ByDelta[0].Delta != 280 {
		t.Fatalf("delta leader = %+v, want pid 100 +280", got.ByDelta[0])
	}
	for _, parent := range got.ByChildren {
		if parent.PID == 100 && parent.Delta != 280 {
			t.Fatalf("child-count ranking lost the delta: %+v", parent)
		}
	}
	if no := Attribution(stormSnapshot(300), nil); no.ByDelta != nil {
		t.Fatalf("without a previous snapshot there is no delta ranking: %+v", no.ByDelta)
	}
}

func TestAttributionIsUnreadOffLinux(t *testing.T) {
	got := attributionFor("darwin", stormSnapshot(300), nil)
	if got.State != Unread || got.Reason == "" || got.ByChildren != nil {
		t.Fatalf("off-linux attribution must be unread with a reason, got %+v", got)
	}
	empty := attributionFor(attributionOS, PressureSnapshot{}, nil)
	if empty.State != Unread {
		t.Fatalf("a snapshot without a process list must be unread, got %+v", empty)
	}
}

func TestAttributionCompletesQuickly(t *testing.T) {
	if runtime.GOOS != attributionOS {
		t.Skip("attribution is read on linux only")
	}
	snapshot := stormSnapshot(5000)
	previous := stormSnapshot(4000)
	start := time.Now()
	got := Attribution(snapshot, &previous)
	elapsed := time.Since(start)
	t.Logf("attribution over %d processes took %s", len(snapshot.Processes), elapsed)
	if got.State != Read || elapsed > 200*time.Millisecond {
		t.Fatalf("attribution over 5000 processes took %s", elapsed)
	}
}
