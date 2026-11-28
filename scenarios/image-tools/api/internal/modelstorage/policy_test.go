package modelstorage

import (
	"math"
	"testing"
	"time"
)

const gib = int64(1) << 30

func testPolicy() Policy {
	p := DefaultPolicy()
	p.FloorBytes = 100 * gib
	p.ReserveBytes = 20 * gib
	p.AmpleBytes = 100 * gib
	p.MinIdle = 24 * time.Hour
	p.MaxIdle = 30 * 24 * time.Hour
	return p
}

func TestEvaluateMapsHeadroomToModeAndThreshold(t *testing.T) {
	p := testPolicy()
	cases := []struct {
		name      string
		headroom  int64
		mode      string
		unlimited bool
		threshold time.Duration
	}{
		{"ample keeps everything", 100 * gib, ModeAmple, true, 0},
		{"far above ample", 400 * gib, ModeAmple, true, 0},
		{"just below ample tolerates max idle", 100*gib - 1, ModeAdaptive, false, 30 * 24 * time.Hour},
		{"at reserve tolerates min idle", 20 * gib, ModeAdaptive, false, 24 * time.Hour},
		{"below reserve is a deficit", 20*gib - 1, ModeReserveDeficit, false, 24 * time.Hour},
		{"below the floor is a deficit", -5 * gib, ModeReserveDeficit, false, 24 * time.Hour},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := p.Evaluate(p.FloorBytes + tc.headroom)
			if got.Mode != tc.mode || got.Unlimited != tc.unlimited {
				t.Fatalf("mode=%s unlimited=%t, want %s %t", got.Mode, got.Unlimited, tc.mode, tc.unlimited)
			}
			if got.HeadroomBytes != tc.headroom {
				t.Fatalf("headroom=%d, want %d", got.HeadroomBytes, tc.headroom)
			}
			if diff := got.IdleThreshold - tc.threshold; diff > time.Minute || diff < -time.Minute {
				t.Fatalf("threshold=%s, want %s", got.IdleThreshold, tc.threshold)
			}
		})
	}
}

func TestEvaluateShrinksThresholdGeometrically(t *testing.T) {
	p := testPolicy()
	mid := p.Evaluate(p.FloorBytes + 60*gib) // half-way between reserve and ample
	want := time.Duration(float64(24*time.Hour) * math.Sqrt(30))
	if diff := mid.IdleThreshold - want; diff > time.Minute || diff < -time.Minute {
		t.Fatalf("midpoint threshold=%s, want ~%s", mid.IdleThreshold, want)
	}
	previous := time.Duration(math.MaxInt64)
	for headroom := 99 * gib; headroom >= 20*gib; headroom -= 7 * gib {
		got := p.Evaluate(p.FloorBytes + headroom).IdleThreshold
		if got > previous {
			t.Fatalf("threshold grew as headroom shrank: %s at %d GiB after %s", got, headroom/gib, previous)
		}
		previous = got
	}
}

func TestPriorityPrefersIdleAndLargeModels(t *testing.T) {
	p := testPolicy()
	if p.Priority(4*gib, 10*24*time.Hour) <= p.Priority(4*gib, 24*time.Hour) {
		t.Fatal("a longer-idle model of the same size must rank higher")
	}
	if p.Priority(4*gib, 48*time.Hour) <= p.Priority(50<<20, 48*time.Hour) {
		t.Fatal("at equal idleness the fixed re-acquisition overhead must favour freeing the larger model")
	}
	if p.Priority(0, 1000*time.Hour) != 0 {
		t.Fatal("a zero-byte model has no offload value")
	}
}

func candidate(id string, bytes int64, idle time.Duration, protected ...string) Candidate {
	return Candidate{ModelID: id, Bytes: bytes, Idle: idle, LastUseBasis: UseBasisSuccessfulJob, Protected: protected}
}

func selected(s Selection) map[string]string {
	out := map[string]string{}
	for _, c := range s.Candidates {
		if c.Selected {
			out[c.ModelID] = c.SelectionReason
		}
	}
	return out
}

func TestSelectAmpleHeadroomKeepsEverything(t *testing.T) {
	p := testPolicy()
	cands := []Candidate{candidate("old", 5*gib, 365*24*time.Hour), candidate("new", gib, time.Hour)}
	got := p.Select(cands, p.Evaluate(p.FloorBytes+150*gib), 0, nil)
	if len(selected(got)) != 0 {
		t.Fatalf("ample headroom selected %v", selected(got))
	}
}

func TestSelectAdaptiveOffloadsOnlyBeyondThreshold(t *testing.T) {
	p := testPolicy()
	pressure := p.Evaluate(p.FloorBytes + 60*gib) // ~5.5 day threshold
	cands := []Candidate{
		candidate("stale", 4*gib, 10*24*time.Hour),
		candidate("recent", 4*gib, 2*24*time.Hour),
		candidate("stale-but-kept", 4*gib, 40*24*time.Hour, ProtectKept),
	}
	got := selected(p.Select(cands, pressure, 0, nil))
	if len(got) != 1 || got["stale"] != ReasonIdle {
		t.Fatalf("selected %v, want only stale for idleness", got)
	}
}

func TestSelectRestoresReserveByPriorityAndReportsShortfall(t *testing.T) {
	p := testPolicy()
	pressure := p.Evaluate(p.FloorBytes + 12*gib) // 8 GiB below reserve
	cands := []Candidate{
		candidate("big-recent", 6*gib, 3*time.Hour),
		candidate("small-recent", 200<<20, 2*time.Hour),
		candidate("mid-idle", 3*gib, 20*time.Hour),
		candidate("in-use", 30*gib, 400*time.Hour, ProtectInUse),
	}
	s := p.Select(cands, pressure, 0, nil)
	got := selected(s)
	if got["big-recent"] != ReasonReserve || got["mid-idle"] != ReasonReserve {
		t.Fatalf("selected %v, want the two highest-value eligible models", got)
	}
	if _, ok := got["in-use"]; ok {
		t.Fatal("a protected model was selected under reserve pressure")
	}
	if s.ProjectedHeadroomBytes < p.ReserveBytes || s.ShortfallBytes != 0 {
		t.Fatalf("projected=%d shortfall=%d, want reserve restored", s.ProjectedHeadroomBytes, s.ShortfallBytes)
	}
	if _, ok := got["small-recent"]; ok {
		t.Fatal("selection kept removing after the reserve was restored")
	}

	short := p.Select([]Candidate{candidate("only", gib, time.Hour)}, p.Evaluate(p.FloorBytes+2*gib), 0, nil)
	if short.ShortfallBytes != 17*gib {
		t.Fatalf("shortfall=%d, want %d", short.ShortfallBytes, 17*gib)
	}
}

func TestSelectMakesRoomForPendingAcquisition(t *testing.T) {
	p := testPolicy()
	pressure := p.Evaluate(p.FloorBytes + 30*gib) // above reserve, below reserve+need
	s := p.Select([]Candidate{candidate("a", 8*gib, 2*time.Hour), candidate("b", 8*gib, time.Hour)}, pressure, 15*gib, nil)
	if s.Pressure.Mode != ModeReserveDeficit || s.TargetHeadroomBytes != 35*gib {
		t.Fatalf("mode=%s target=%d", s.Pressure.Mode, s.TargetHeadroomBytes)
	}
	if got := selected(s); len(got) != 1 || got["a"] != ReasonReserve {
		t.Fatalf("selected %v, want only the higher-value model", got)
	}
}

func TestSelectRequestedHonoursProtections(t *testing.T) {
	p := testPolicy()
	cands := []Candidate{candidate("dncnn", 1<<20, time.Hour), candidate("pinned", gib, 900*time.Hour, ProtectKept), candidate("other", gib, 900*time.Hour)}
	got := selected(p.Select(cands, p.Evaluate(p.FloorBytes+150*gib), 0, []string{"dncnn", "pinned"}))
	if len(got) != 1 || got["dncnn"] != ReasonRequested {
		t.Fatalf("selected %v, want only the unprotected requested model", got)
	}
}

func TestSelectPendingAcquisitionUsesReserveThreshold(t *testing.T) {
	p := testPolicy()
	// Ample headroom, but a large pending acquisition would leave less than
	// the reserve: the pass must not report "nothing idles out".
	s := p.Select([]Candidate{candidate("idle", gib, 3*24*time.Hour)}, p.Evaluate(p.FloorBytes+150*gib), 140*gib, nil)
	if s.Pressure.Mode != ModeReserveDeficit || s.Pressure.Unlimited || s.Pressure.IdleThreshold != p.MinIdle {
		t.Fatalf("pressure=%+v, want reserve deficit at the minimum idle threshold", s.Pressure)
	}
	if got := selected(s); got["idle"] != ReasonIdle {
		t.Fatalf("selected %v, want the idle model chosen for idleness", got)
	}
}
