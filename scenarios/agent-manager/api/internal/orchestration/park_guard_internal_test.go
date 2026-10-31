package orchestration

import (
	"testing"

	"agent-manager/internal/domain"
)

// TestEvaluateReparkGuard covers the no-progress re-park guard's decision table:
// the degenerate "wake → re-run the same blocking command → re-park" loop is the
// only thing it refuses; forward progress or a genuinely different await always
// proceeds (and resets the streak).
func TestEvaluateReparkGuard(t *testing.T) {
	o := &Orchestrator{}
	const key = "git-control-tower:agent-manager/am-park-resume"

	cases := []struct {
		name           string
		run            *domain.Run
		producer, akey string
		wantStreak     int
		wantRefuse     bool
	}{
		{
			name:       "first ever park (no prior await) proceeds",
			run:        &domain.Run{LastAwaitKey: "", LastWakeSeq: 0, TranscriptLastSeq: 0},
			producer:   "git-control-tower",
			akey:       "agent-manager/am-park-resume",
			wantStreak: 0,
			wantRefuse: false,
		},
		{
			name:       "same key, no progress, first re-park is tolerated (lag benefit-of-doubt)",
			run:        &domain.Run{LastAwaitKey: key, LastWakeSeq: 5, TranscriptLastSeq: 5, SameKeyParkStreak: 0},
			producer:   "git-control-tower",
			akey:       "agent-manager/am-park-resume",
			wantStreak: 1,
			wantRefuse: false,
		},
		{
			name:       "same key, no progress, second consecutive re-park is REFUSED",
			run:        &domain.Run{LastAwaitKey: key, LastWakeSeq: 5, TranscriptLastSeq: 5, SameKeyParkStreak: 1},
			producer:   "git-control-tower",
			akey:       "agent-manager/am-park-resume",
			wantStreak: 1,
			wantRefuse: true,
		},
		{
			name:       "same key WITH progress resets and proceeds",
			run:        &domain.Run{LastAwaitKey: key, LastWakeSeq: 5, TranscriptLastSeq: 9, SameKeyParkStreak: 1},
			producer:   "git-control-tower",
			akey:       "agent-manager/am-park-resume",
			wantStreak: 0,
			wantRefuse: false,
		},
		{
			name:       "different key, no progress, proceeds and resets (legit sequential await)",
			run:        &domain.Run{LastAwaitKey: key, LastWakeSeq: 5, TranscriptLastSeq: 5, SameKeyParkStreak: 1},
			producer:   "test-genie",
			akey:       "web-console/run-99",
			wantStreak: 0,
			wantRefuse: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			streak, refuse := o.evaluateReparkGuard(tc.run, tc.producer, tc.akey)
			if refuse != tc.wantRefuse {
				t.Errorf("refuse = %v, want %v", refuse, tc.wantRefuse)
			}
			if streak != tc.wantStreak {
				t.Errorf("streak = %d, want %d", streak, tc.wantStreak)
			}
		})
	}
}
