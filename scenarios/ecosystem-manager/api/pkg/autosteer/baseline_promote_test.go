package autosteer

import "testing"

// validBaselineProfile returns a minimal valid profile to attach a
// BaselinePromote block to.
func validBaselineProfile() *AutoSteerProfile {
	return &AutoSteerProfile{
		Name:          "p",
		AllowedSkills: []string{"progress"},
		Objective: Objective{
			DimensionWeights: map[string]float64{"operational-targets": 1},
			Targets:          ObjectiveTargets{MaxOpenSeverity: "warning"},
		},
		Budget: Budget{MaxIterations: 5},
	}
}

func TestValidateProfile_BaselinePromote(t *testing.T) { // [REQ:EM-BASE-004]
	cases := []struct {
		name      string
		bp        *BaselinePromoteObjective
		wantErr   bool
		wantMode  string // expected normalized Mode after validation (when no error)
		wantCheck bool   // expect BaselinePromoteEnabled()
	}{
		{name: "absent block is valid", bp: nil, wantErr: false},
		{name: "enabled default mode", bp: &BaselinePromoteObjective{Enabled: true}, wantMode: "", wantCheck: true},
		{name: "end_of_engagement", bp: &BaselinePromoteObjective{Enabled: true, Mode: "end_of_engagement"}, wantMode: "end_of_engagement", wantCheck: true},
		{name: "checkpoint_on_green", bp: &BaselinePromoteObjective{Enabled: true, Mode: "checkpoint_on_green", CadenceIter: 3}, wantMode: "checkpoint_on_green", wantCheck: true},
		{name: "mode normalized to lower", bp: &BaselinePromoteObjective{Mode: "  Checkpoint_On_Green "}, wantMode: "checkpoint_on_green"},
		{name: "unknown mode rejected", bp: &BaselinePromoteObjective{Enabled: true, Mode: "yolo"}, wantErr: true},
		{name: "negative cadence rejected", bp: &BaselinePromoteObjective{Enabled: true, CadenceIter: -1}, wantErr: true},
		{name: "disabled is not enabled", bp: &BaselinePromoteObjective{Enabled: false}, wantCheck: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validBaselineProfile()
			p.BaselinePromote = tc.bp
			err := ValidateProfile(p)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected validation error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
			if tc.bp != nil && p.BaselinePromote.Mode != tc.wantMode {
				t.Fatalf("Mode normalized to %q, want %q", p.BaselinePromote.Mode, tc.wantMode)
			}
			if got := p.BaselinePromoteEnabled(); got != tc.wantCheck {
				t.Fatalf("BaselinePromoteEnabled()=%v, want %v", got, tc.wantCheck)
			}
		})
	}
}

func TestBaselinePromoteHelpers_Defaults(t *testing.T) { // [REQ:EM-BASE-004]
	// nil block: disabled, default mode, zero cadence.
	p := validBaselineProfile()
	if p.BaselinePromoteEnabled() {
		t.Fatalf("nil block should be disabled")
	}
	if got := p.BaselinePromoteMode(); got != BaselinePromoteEndOfEngagement {
		t.Fatalf("nil block mode=%q, want %q", got, BaselinePromoteEndOfEngagement)
	}
	if got := p.BaselinePromoteCadence(); got != 0 {
		t.Fatalf("nil block cadence=%d, want 0", got)
	}

	// Empty mode falls back to end_of_engagement.
	p.BaselinePromote = &BaselinePromoteObjective{Enabled: true, Mode: "", CadenceIter: 4}
	if got := p.BaselinePromoteMode(); got != BaselinePromoteEndOfEngagement {
		t.Fatalf("empty mode=%q, want %q", got, BaselinePromoteEndOfEngagement)
	}
	if got := p.BaselinePromoteCadence(); got != 4 {
		t.Fatalf("cadence=%d, want 4", got)
	}

	// nil profile is safe.
	var np *AutoSteerProfile
	if np.BaselinePromoteEnabled() {
		t.Fatalf("nil profile should be disabled")
	}
	if got := np.BaselinePromoteMode(); got != BaselinePromoteEndOfEngagement {
		t.Fatalf("nil profile mode=%q", got)
	}
}
