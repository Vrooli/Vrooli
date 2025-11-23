package autosteer

import (
	"strings"
	"testing"
	"time"
)

func TestPromptEnhancer_GenerateAutoSteerSection(t *testing.T) {
	enhancer := NewPromptEnhancer()

	profile := &AutoSteerProfile{
		ID:          "test-profile",
		Name:        "Test Profile",
		Description: "A test profile for testing",
		Phases: []SteerPhase{
			{
				ID:            "phase-1",
				Mode:          ModeProgress,
				MaxIterations: 10,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "operational_targets_percentage",
						CompareOperator: OpGreaterThanEquals,
						Value:           80,
					},
				},
			},
			{
				ID:            "phase-2",
				Mode:          ModeUX,
				MaxIterations: 5,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "accessibility_score",
						CompareOperator: OpGreaterThan,
						Value:           90,
					},
				},
			},
		},
		QualityGates: []QualityGate{
			{
				Name: "build_health",
				Condition: StopCondition{
					Type:            ConditionTypeSimple,
					Metric:          "build_status",
					CompareOperator: OpEquals,
					Value:           1,
				},
				FailureAction: ActionHalt,
				Message:       "Build must be passing",
			},
		},
	}

	state := &ProfileExecutionState{
		TaskID:                "task-1",
		ProfileID:             "test-profile",
		CurrentPhaseIndex:     0,
		CurrentPhaseIteration: 3,
		PhaseHistory:          []PhaseExecution{},
		Metrics: MetricsSnapshot{
			Timestamp:                    time.Now(),
			Loops:                        3,
			BuildStatus:                  1,
			OperationalTargetsPercentage: 70.0,
		},
		PhaseStartMetrics: MetricsSnapshot{
			Timestamp:                    time.Now().Add(-1 * time.Hour),
			OperationalTargetsPercentage: 50.0,
		},
	}

	evaluator := NewConditionEvaluator()

	t.Run("generate section for first phase", func(t *testing.T) {
		section := enhancer.GenerateAutoSteerSection(state, profile, evaluator)

		// Verify key sections are present
		if !strings.Contains(section, "Test Profile") {
			t.Error("Expected profile name in output")
		}
		if !strings.Contains(section, "PROGRESS") {
			t.Error("Expected current mode (PROGRESS) in output")
		}
		if !strings.Contains(section, "Phase 1 of 2") {
			t.Error("Expected phase progress in output")
		}
		if !strings.Contains(section, "Iteration: 3 of 10") {
			t.Error("Expected iteration progress in output")
		}
		if !strings.Contains(section, "Stop Conditions") {
			t.Error("Expected stop conditions section")
		}
		if !strings.Contains(section, "Quality Gates") {
			t.Error("Expected quality gates section")
		}
		if !strings.Contains(section, "build_health") {
			t.Error("Expected quality gate name")
		}
	})

	t.Run("generate section with phase history", func(t *testing.T) {
		// Add completed phase to history
		completedPhase := PhaseExecution{
			PhaseID:    "phase-1",
			Mode:       ModeProgress,
			Iterations: 10,
			StartMetrics: MetricsSnapshot{
				OperationalTargetsPercentage: 50.0,
			},
			EndMetrics: MetricsSnapshot{
				OperationalTargetsPercentage: 80.0,
			},
			StopReason: "condition_met",
		}

		stateWithHistory := &ProfileExecutionState{
			TaskID:                "task-1",
			ProfileID:             "test-profile",
			CurrentPhaseIndex:     1,
			CurrentPhaseIteration: 2,
			PhaseHistory:          []PhaseExecution{completedPhase},
			Metrics: MetricsSnapshot{
				Timestamp:   time.Now(),
				BuildStatus: 1,
				UX: &UXMetrics{
					AccessibilityScore: 85.0,
				},
			},
		}

		section := enhancer.GenerateAutoSteerSection(stateWithHistory, profile, evaluator)

		if !strings.Contains(section, "Completed Phases") {
			t.Error("Expected completed phases section")
		}
		if !strings.Contains(section, "PROGRESS") {
			t.Error("Expected completed phase mode in history")
		}
		if !strings.Contains(section, "10 iterations") {
			t.Error("Expected iteration count in history")
		}
		if !strings.Contains(section, "condition_met") {
			t.Error("Expected stop reason in history")
		}
	})

	t.Run("generate section for completed profile", func(t *testing.T) {
		completedState := &ProfileExecutionState{
			TaskID:            "task-1",
			ProfileID:         "test-profile",
			CurrentPhaseIndex: 2, // Beyond last phase
			Metrics: MetricsSnapshot{
				Timestamp:   time.Now(),
				BuildStatus: 1,
			},
		}

		section := enhancer.GenerateAutoSteerSection(completedState, profile, evaluator)

		if !strings.Contains(section, "All phases completed") {
			t.Error("Expected completion message")
		}
	})

	t.Run("return empty for nil state", func(t *testing.T) {
		section := enhancer.GenerateAutoSteerSection(nil, profile, evaluator)
		if section != "" {
			t.Error("Expected empty section for nil state")
		}
	})

	t.Run("return empty for nil profile", func(t *testing.T) {
		section := enhancer.GenerateAutoSteerSection(state, nil, evaluator)
		if section != "" {
			t.Error("Expected empty section for nil profile")
		}
	})
}

func TestPromptEnhancer_GetKeyImprovements(t *testing.T) {
	enhancer := NewPromptEnhancer()

	t.Run("extract significant improvements", func(t *testing.T) {
		phase := PhaseExecution{
			StartMetrics: MetricsSnapshot{
				OperationalTargetsPercentage: 60.0,
				UX: &UXMetrics{
					AccessibilityScore: 70.0,
					UITestCoverage:     40.0,
				},
				Test: &TestMetrics{
					UnitTestCoverage: 50.0,
				},
				Refactor: &RefactorMetrics{
					TidinessScore:           70.0,
					CyclomaticComplexityAvg: 15.0,
				},
			},
			EndMetrics: MetricsSnapshot{
				OperationalTargetsPercentage: 80.0, // +20% (significant)
				UX: &UXMetrics{
					AccessibilityScore: 85.0, // +15 (significant)
					UITestCoverage:     45.0, // +5 (not significant - threshold is 10)
				},
				Test: &TestMetrics{
					UnitTestCoverage: 75.0, // +25 (significant)
				},
				Refactor: &RefactorMetrics{
					TidinessScore:           78.0, // +8 (significant)
					CyclomaticComplexityAvg: 10.0, // -5 (significant reduction)
				},
			},
		}

		improvements := enhancer.getKeyImprovements(phase)

		// Should have 5 significant improvements
		// - operational_targets_percentage: +20
		// - accessibility_score: +15
		// - unit_test_coverage: +25
		// - tidiness_score: +8
		// - cyclomatic_complexity_avg: -5

		if len(improvements) != 5 {
			t.Errorf("Expected 5 significant improvements, got %d", len(improvements))
			for _, imp := range improvements {
				t.Logf("Improvement: %s", imp)
			}
		}

		// Verify specific improvements are present
		hasOpTargets := false
		hasAccessibility := false
		hasComplexity := false

		for _, imp := range improvements {
			if strings.Contains(imp, "Operational Targets Percentage") {
				hasOpTargets = true
			}
			if strings.Contains(imp, "Accessibility Score") {
				hasAccessibility = true
			}
			if strings.Contains(imp, "reduced") {
				hasComplexity = true
			}
		}

		if !hasOpTargets {
			t.Error("Expected operational targets improvement")
		}
		if !hasAccessibility {
			t.Error("Expected accessibility improvement")
		}
		if !hasComplexity {
			t.Error("Expected complexity reduction")
		}
	})

	t.Run("no improvements below threshold", func(t *testing.T) {
		phase := PhaseExecution{
			StartMetrics: MetricsSnapshot{
				OperationalTargetsPercentage: 70.0,
			},
			EndMetrics: MetricsSnapshot{
				OperationalTargetsPercentage: 72.0, // +2% (below 5% threshold)
			},
		}

		improvements := enhancer.getKeyImprovements(phase)

		if len(improvements) != 0 {
			t.Errorf("Expected no improvements below threshold, got %d", len(improvements))
		}
	})
}

func TestPromptEnhancer_FormatImprovement(t *testing.T) {
	enhancer := NewPromptEnhancer()

	tests := []struct {
		name         string
		metric       string
		delta        float64
		wantContains []string
	}{
		{
			name:         "positive improvement",
			metric:       "operational_targets_percentage",
			delta:        20.0,
			wantContains: []string{"Operational Targets Percentage", "+20.0"},
		},
		{
			name:         "accessibility score increase",
			metric:       "accessibility_score",
			delta:        15.0,
			wantContains: []string{"Accessibility Score", "+15.0"},
		},
		{
			name:         "complexity reduction",
			metric:       "cyclomatic_complexity_avg",
			delta:        -5.0,
			wantContains: []string{"Cyclomatic Complexity Avg", "reduced", "5.0"},
		},
		{
			name:         "complexity increase (bad)",
			metric:       "cyclomatic_complexity_avg",
			delta:        3.0,
			wantContains: []string{"Cyclomatic Complexity Avg", "increased", "3.0"},
		},
		{
			name:         "negative delta",
			metric:       "flaky_tests",
			delta:        -2.0,
			wantContains: []string{"Flaky Tests", "-2.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enhancer.formatImprovement(tt.metric, tt.delta)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("formatImprovement() = %q, want to contain %q", result, want)
				}
			}
		})
	}
}

func TestPromptEnhancer_GeneratePhaseTransitionMessage(t *testing.T) {
	enhancer := NewPromptEnhancer()

	oldPhase := SteerPhase{
		ID:            "phase-1",
		Mode:          ModeProgress,
		MaxIterations: 10,
	}

	newPhase := SteerPhase{
		ID:            "phase-2",
		Mode:          ModeUX,
		MaxIterations: 5,
	}

	message := enhancer.GeneratePhaseTransitionMessage(oldPhase, newPhase, 2, 3)

	// Verify key sections
	if !strings.Contains(message, "Phase Transition") {
		t.Error("Expected 'Phase Transition' header")
	}
	if !strings.Contains(message, "PROGRESS") {
		t.Error("Expected old phase mode (PROGRESS)")
	}
	if !strings.Contains(message, "UX") {
		t.Error("Expected new phase mode (UX)")
	}
	if !strings.Contains(message, "Phase 2 of 3") {
		t.Error("Expected phase progress")
	}
	if !strings.Contains(message, "New Phase Focus") {
		t.Error("Expected new phase focus section")
	}
}

func TestPromptEnhancer_GenerateCompletionMessage(t *testing.T) {
	enhancer := NewPromptEnhancer()

	profile := &AutoSteerProfile{
		ID:   "test-profile",
		Name: "Test Profile",
		Phases: []SteerPhase{
			{ID: "phase-1", Mode: ModeProgress, MaxIterations: 10},
			{ID: "phase-2", Mode: ModeUX, MaxIterations: 5},
		},
	}

	state := &ProfileExecutionState{
		TaskID:    "task-1",
		ProfileID: "test-profile",
		PhaseHistory: []PhaseExecution{
			{
				PhaseID:    "phase-1",
				Mode:       ModeProgress,
				Iterations: 10,
				StartMetrics: MetricsSnapshot{
					OperationalTargetsPercentage: 50.0,
				},
				EndMetrics: MetricsSnapshot{
					OperationalTargetsPercentage: 80.0,
				},
				StopReason: "condition_met",
			},
			{
				PhaseID:    "phase-2",
				Mode:       ModeUX,
				Iterations: 5,
				StartMetrics: MetricsSnapshot{
					UX: &UXMetrics{
						AccessibilityScore: 70.0,
					},
				},
				EndMetrics: MetricsSnapshot{
					UX: &UXMetrics{
						AccessibilityScore: 92.0,
					},
				},
				StopReason: "max_iterations",
			},
		},
	}

	message := enhancer.GenerateCompletionMessage(profile, state)

	// Verify key sections
	if !strings.Contains(message, "Auto Steer Profile Complete") {
		t.Error("Expected completion header")
	}
	if !strings.Contains(message, "Test Profile") {
		t.Error("Expected profile name")
	}
	if !strings.Contains(message, "2 phases") {
		t.Error("Expected phase count")
	}
	if !strings.Contains(message, "Phase Summary") {
		t.Error("Expected phase summary section")
	}
	if !strings.Contains(message, "PROGRESS") {
		t.Error("Expected first phase mode")
	}
	if !strings.Contains(message, "UX") {
		t.Error("Expected second phase mode")
	}
	if !strings.Contains(message, "condition_met") {
		t.Error("Expected first phase stop reason")
	}
	if !strings.Contains(message, "max_iterations") {
		t.Error("Expected second phase stop reason")
	}
	if !strings.Contains(message, "Next Steps") {
		t.Error("Expected next steps section")
	}

	// Verify improvements are shown
	if !strings.Contains(message, "Key Improvements") {
		t.Error("Expected key improvements section")
	}
}

func TestPromptEnhancer_EdgeCases(t *testing.T) {
	enhancer := NewPromptEnhancer()

	t.Run("state with no metrics", func(t *testing.T) {
		profile := &AutoSteerProfile{
			ID:   "test",
			Name: "Test",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					Mode:          ModeProgress,
					MaxIterations: 5,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "loops",
							CompareOperator: OpGreaterThan,
							Value:           3,
						},
					},
				},
			},
		}

		state := &ProfileExecutionState{
			TaskID:                "task-1",
			ProfileID:             "test",
			CurrentPhaseIndex:     0,
			CurrentPhaseIteration: 1,
			Metrics:               MetricsSnapshot{}, // Empty metrics
		}

		evaluator := NewConditionEvaluator()
		section := enhancer.GenerateAutoSteerSection(state, profile, evaluator)

		// Should still generate valid output
		if section == "" {
			t.Error("Expected non-empty section even with empty metrics")
		}
		if !strings.Contains(section, "PROGRESS") {
			t.Error("Expected mode in output")
		}
	})

	t.Run("profile with no quality gates", func(t *testing.T) {
		profile := &AutoSteerProfile{
			ID:   "test",
			Name: "Test",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					Mode:          ModeProgress,
					MaxIterations: 5,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "loops",
							CompareOperator: OpGreaterThan,
							Value:           3,
						},
					},
				},
			},
			QualityGates: []QualityGate{}, // No gates
		}

		state := &ProfileExecutionState{
			TaskID:                "task-1",
			ProfileID:             "test",
			CurrentPhaseIndex:     0,
			CurrentPhaseIteration: 1,
			Metrics: MetricsSnapshot{
				Timestamp: time.Now(),
			},
		}

		evaluator := NewConditionEvaluator()
		section := enhancer.GenerateAutoSteerSection(state, profile, evaluator)

		// Should generate output without quality gates section
		if section == "" {
			t.Error("Expected non-empty section")
		}
		// Quality Gates section should not be present
		if strings.Contains(section, "Quality Gates") {
			t.Error("Should not have quality gates section when none defined")
		}
	})
}
