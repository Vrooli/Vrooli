package autosteer

import "testing"

func validObjectiveProfile() *AutoSteerProfile {
	return &AutoSteerProfile{
		Name: "Valid",
		Objective: Objective{
			DimensionWeights: map[string]float64{"standards": 1.0, "tests": 1.2},
			Targets:          ObjectiveTargets{MaxOpenSeverity: "warning", OperationalTargetsPct: 90},
		},
		AllowedSkills: []string{"progress", "test"},
		Budget:        Budget{MaxIterations: 20, DiminishingReturnsFloor: 0.02, ReauditCadence: 5},
		AuditPreset:   "comprehensive",
	}
}

func TestValidateProfile_Valid(t *testing.T) {
	p := validObjectiveProfile()
	if err := ValidateProfile(p); err != nil {
		t.Fatalf("expected valid profile, got error: %v", err)
	}
}

func TestValidateProfile_ValidLadder(t *testing.T) {
	p := validObjectiveProfile()
	p.Ladder = &LadderObjective{Enabled: true, TopRung: "R4", BoostFactor: 8, StandardsMaxCount: 3}
	if err := ValidateProfile(p); err != nil {
		t.Fatalf("expected valid ladder profile, got error: %v", err)
	}
	if !p.ladderEnabled() {
		t.Error("ladderEnabled() should be true for an enabled ladder block")
	}
}

func TestValidateProfile_Invalid(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*AutoSteerProfile)
	}{
		{"nil name", func(p *AutoSteerProfile) { p.Name = "" }},
		{"empty allowed skill", func(p *AutoSteerProfile) { p.AllowedSkills = []string{"  "} }},
		{"empty denied skill", func(p *AutoSteerProfile) { p.DeniedSkills = []string{"  "} }},
		{"values nothing", func(p *AutoSteerProfile) {
			p.Objective.DimensionWeights = nil
			p.Ladder = nil
		}},
		{"unknown dimension weight", func(p *AutoSteerProfile) {
			p.Objective.DimensionWeights = map[string]float64{"not-a-dimension": 1.0}
		}},
		{"negative weight", func(p *AutoSteerProfile) {
			p.Objective.DimensionWeights = map[string]float64{"standards": -1}
		}},
		{"bad severity", func(p *AutoSteerProfile) { p.Objective.Targets.MaxOpenSeverity = "catastrophic" }},
		{"op pct out of range", func(p *AutoSteerProfile) { p.Objective.Targets.OperationalTargetsPct = 150 }},
		{"zero iterations", func(p *AutoSteerProfile) { p.Budget.MaxIterations = 0 }},
		{"negative floor", func(p *AutoSteerProfile) { p.Budget.DiminishingReturnsFloor = -0.1 }},
		{"negative cadence", func(p *AutoSteerProfile) { p.Budget.ReauditCadence = -1 }},
		{"bad ladder top_rung", func(p *AutoSteerProfile) {
			p.Ladder = &LadderObjective{Enabled: true, TopRung: "R9"}
		}},
		{"negative ladder boost", func(p *AutoSteerProfile) {
			p.Ladder = &LadderObjective{Enabled: true, BoostFactor: -1}
		}},
		{"negative ladder standards cap", func(p *AutoSteerProfile) {
			p.Ladder = &LadderObjective{Enabled: true, StandardsMaxCount: -1}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validObjectiveProfile()
			tc.mutate(p)
			if err := ValidateProfile(p); err == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestValidateProfile_EmptyAllowedSkillsDerives(t *testing.T) {
	p := validObjectiveProfile()
	p.AllowedSkills = nil
	if err := ValidateProfile(p); err != nil {
		t.Fatalf("empty allowed_skills should derive from catalog later: %v", err)
	}
}

func TestValidateProfile_NormalizesAllowedSkills(t *testing.T) {
	p := validObjectiveProfile()
	p.AllowedSkills = []string{" progress ", "test", "test", ""}
	// Empty entry should fail.
	if err := ValidateProfile(p); err == nil {
		t.Fatal("expected error for empty skill id")
	}

	p.AllowedSkills = []string{" progress ", "test", "test"}
	if err := ValidateProfile(p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.AllowedSkills) != 2 {
		t.Fatalf("expected dedup+trim to 2 skills, got %v", p.AllowedSkills)
	}
	if p.AllowedSkills[0] != "progress" {
		t.Fatalf("expected trimmed 'progress', got %q", p.AllowedSkills[0])
	}
}
