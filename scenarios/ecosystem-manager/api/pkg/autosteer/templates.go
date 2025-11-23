package autosteer

import "github.com/google/uuid"

// GetBuiltInTemplates returns all built-in profile templates
func GetBuiltInTemplates() []*AutoSteerProfile {
	return []*AutoSteerProfile{
		getBalancedTemplate(),
		getRapidMVPTemplate(),
		getProductionReadyTemplate(),
		getRefactorTestFocusTemplate(),
		getUXExcellenceTemplate(),
	}
}

// getBalancedTemplate returns the "Balanced" template
func getBalancedTemplate() *AutoSteerProfile {
	return &AutoSteerProfile{
		ID:          uuid.New().String(),
		Name:        "Balanced",
		Description: "Well-rounded improvement across all dimensions. Suitable for most scenarios.",
		Phases: []SteerPhase{
			{
				ID:            uuid.New().String(),
				Mode:          ModeProgress,
				MaxIterations: 10,
				StopConditions: []StopCondition{
					{
						Type:     ConditionTypeCompound,
						Operator: LogicalOR,
						Conditions: []StopCondition{
							{
								Type:            ConditionTypeSimple,
								Metric:          "loops",
								CompareOperator: OpGreaterThan,
								Value:           10,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "operational_targets_percentage",
								CompareOperator: OpGreaterThanEquals,
								Value:           80,
							},
						},
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeUX,
				MaxIterations: 10,
				StopConditions: []StopCondition{
					{
						Type:     ConditionTypeCompound,
						Operator: LogicalOR,
						Conditions: []StopCondition{
							{
								Type:            ConditionTypeSimple,
								Metric:          "loops",
								CompareOperator: OpGreaterThan,
								Value:           10,
							},
							{
								Type:     ConditionTypeCompound,
								Operator: LogicalAND,
								Conditions: []StopCondition{
									{
										Type:            ConditionTypeSimple,
										Metric:          "accessibility_score",
										CompareOperator: OpGreaterThan,
										Value:           90,
									},
									{
										Type:            ConditionTypeSimple,
										Metric:          "ui_test_coverage",
										CompareOperator: OpGreaterThan,
										Value:           60,
									},
								},
							},
						},
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeRefactor,
				MaxIterations: 8,
				StopConditions: []StopCondition{
					{
						Type:     ConditionTypeCompound,
						Operator: LogicalOR,
						Conditions: []StopCondition{
							{
								Type:            ConditionTypeSimple,
								Metric:          "loops",
								CompareOperator: OpGreaterThan,
								Value:           8,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "tidiness_score",
								CompareOperator: OpGreaterThan,
								Value:           85,
							},
						},
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeTest,
				MaxIterations: 15,
				StopConditions: []StopCondition{
					{
						Type:     ConditionTypeCompound,
						Operator: LogicalOR,
						Conditions: []StopCondition{
							{
								Type:            ConditionTypeSimple,
								Metric:          "loops",
								CompareOperator: OpGreaterThan,
								Value:           15,
							},
							{
								Type:     ConditionTypeCompound,
								Operator: LogicalAND,
								Conditions: []StopCondition{
									{
										Type:            ConditionTypeSimple,
										Metric:          "unit_test_coverage",
										CompareOperator: OpGreaterThan,
										Value:           80,
									},
									{
										Type:            ConditionTypeSimple,
										Metric:          "integration_test_coverage",
										CompareOperator: OpGreaterThan,
										Value:           70,
									},
									{
										Type:            ConditionTypeSimple,
										Metric:          "flaky_tests",
										CompareOperator: OpEquals,
										Value:           0,
									},
								},
							},
						},
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
				Message:       "Build must be passing to continue",
			},
		},
		Tags: []string{"balanced", "recommended", "complete"},
	}
}

// getRapidMVPTemplate returns the "Rapid MVP" template
func getRapidMVPTemplate() *AutoSteerProfile {
	return &AutoSteerProfile{
		ID:          uuid.New().String(),
		Name:        "Rapid MVP",
		Description: "Fast iteration to minimum viable product. Focuses on getting features working quickly with basic testing.",
		Phases: []SteerPhase{
			{
				ID:            uuid.New().String(),
				Mode:          ModeProgress,
				MaxIterations: 20,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "operational_targets_percentage",
						CompareOperator: OpGreaterThanEquals,
						Value:           70,
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeTest,
				MaxIterations: 5,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "unit_test_coverage",
						CompareOperator: OpGreaterThan,
						Value:           60,
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModePolish,
				MaxIterations: 3,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThanEquals,
						Value:           3,
					},
				},
			},
		},
		Tags: []string{"rapid", "mvp", "quick"},
	}
}

// getProductionReadyTemplate returns the "Production Ready" template
func getProductionReadyTemplate() *AutoSteerProfile {
	return &AutoSteerProfile{
		ID:          uuid.New().String(),
		Name:        "Production Ready",
		Description: "Comprehensive quality assurance for production deployment. Covers all quality dimensions thoroughly.",
		Phases: []SteerPhase{
			{
				ID:            uuid.New().String(),
				Mode:          ModeProgress,
				MaxIterations: 15,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "operational_targets_percentage",
						CompareOperator: OpGreaterThanEquals,
						Value:           100,
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeUX,
				MaxIterations: 15,
				StopConditions: []StopCondition{
					{
						Type:     ConditionTypeCompound,
						Operator: LogicalAND,
						Conditions: []StopCondition{
							{
								Type:            ConditionTypeSimple,
								Metric:          "accessibility_score",
								CompareOperator: OpGreaterThan,
								Value:           95,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "ui_test_coverage",
								CompareOperator: OpGreaterThan,
								Value:           80,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "responsive_breakpoints",
								CompareOperator: OpGreaterThanEquals,
								Value:           3,
							},
						},
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeTest,
				MaxIterations: 20,
				StopConditions: []StopCondition{
					{
						Type:     ConditionTypeCompound,
						Operator: LogicalAND,
						Conditions: []StopCondition{
							{
								Type:            ConditionTypeSimple,
								Metric:          "unit_test_coverage",
								CompareOperator: OpGreaterThan,
								Value:           90,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "integration_test_coverage",
								CompareOperator: OpGreaterThan,
								Value:           80,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "ui_test_coverage",
								CompareOperator: OpGreaterThan,
								Value:           70,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "flaky_tests",
								CompareOperator: OpEquals,
								Value:           0,
							},
						},
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeRefactor,
				MaxIterations: 12,
				StopConditions: []StopCondition{
					{
						Type:     ConditionTypeCompound,
						Operator: LogicalAND,
						Conditions: []StopCondition{
							{
								Type:            ConditionTypeSimple,
								Metric:          "tidiness_score",
								CompareOperator: OpGreaterThan,
								Value:           90,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "cyclomatic_complexity_avg",
								CompareOperator: OpLessThan,
								Value:           8,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "duplication_percentage",
								CompareOperator: OpLessThan,
								Value:           3,
							},
						},
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeSecurity,
				MaxIterations: 10,
				StopConditions: []StopCondition{
					{
						Type:     ConditionTypeCompound,
						Operator: LogicalAND,
						Conditions: []StopCondition{
							{
								Type:            ConditionTypeSimple,
								Metric:          "vulnerability_count",
								CompareOperator: OpEquals,
								Value:           0,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "input_validation_coverage",
								CompareOperator: OpGreaterThan,
								Value:           95,
							},
						},
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModePerformance,
				MaxIterations: 8,
				StopConditions: []StopCondition{
					{
						Type:     ConditionTypeCompound,
						Operator: LogicalAND,
						Conditions: []StopCondition{
							{
								Type:            ConditionTypeSimple,
								Metric:          "initial_load_time_ms",
								CompareOperator: OpLessThan,
								Value:           3000,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "lcp_ms",
								CompareOperator: OpLessThan,
								Value:           2500,
							},
						},
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModePolish,
				MaxIterations: 5,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThanEquals,
						Value:           5,
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
		Tags: []string{"production", "comprehensive", "enterprise"},
	}
}

// getRefactorTestFocusTemplate returns the "Refactor & Test Focus" template
func getRefactorTestFocusTemplate() *AutoSteerProfile {
	return &AutoSteerProfile{
		ID:          uuid.New().String(),
		Name:        "Refactor & Test Focus",
		Description: "For scenarios that work but need quality improvement. Focuses on testing and code quality.",
		Phases: []SteerPhase{
			{
				ID:            uuid.New().String(),
				Mode:          ModeTest,
				MaxIterations: 20,
				StopConditions: []StopCondition{
					{
						Type:     ConditionTypeCompound,
						Operator: LogicalAND,
						Conditions: []StopCondition{
							{
								Type:            ConditionTypeSimple,
								Metric:          "unit_test_coverage",
								CompareOperator: OpGreaterThan,
								Value:           85,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "integration_test_coverage",
								CompareOperator: OpGreaterThan,
								Value:           75,
							},
						},
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeRefactor,
				MaxIterations: 15,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "tidiness_score",
						CompareOperator: OpGreaterThan,
						Value:           90,
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeTest,
				MaxIterations: 10,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "flaky_tests",
						CompareOperator: OpEquals,
						Value:           0,
					},
				},
			},
		},
		Tags: []string{"quality", "testing", "refactor"},
	}
}

// getUXExcellenceTemplate returns the "UX Excellence" template
func getUXExcellenceTemplate() *AutoSteerProfile {
	return &AutoSteerProfile{
		ID:          uuid.New().String(),
		Name:        "UX Excellence",
		Description: "Maximum focus on user experience and design. Creates delightful, accessible interfaces.",
		Phases: []SteerPhase{
			{
				ID:            uuid.New().String(),
				Mode:          ModeProgress,
				MaxIterations: 8,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "operational_targets_percentage",
						CompareOperator: OpGreaterThanEquals,
						Value:           70,
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeExplore,
				MaxIterations: 10,
				Description:   "Experiment with creative UI approaches",
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThanEquals,
						Value:           10,
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeUX,
				MaxIterations: 20,
				StopConditions: []StopCondition{
					{
						Type:     ConditionTypeCompound,
						Operator: LogicalAND,
						Conditions: []StopCondition{
							{
								Type:            ConditionTypeSimple,
								Metric:          "accessibility_score",
								CompareOperator: OpGreaterThan,
								Value:           95,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "user_flows_implemented",
								CompareOperator: OpGreaterThanEquals,
								Value:           10,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "loading_states_count",
								CompareOperator: OpGreaterThanEquals,
								Value:           5,
							},
						},
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModePerformance,
				MaxIterations: 10,
				Description:   "Optimize for perceived performance",
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "initial_load_time_ms",
						CompareOperator: OpLessThan,
						Value:           2000,
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModePolish,
				MaxIterations: 8,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThanEquals,
						Value:           8,
					},
				},
			},
			{
				ID:            uuid.New().String(),
				Mode:          ModeTest,
				MaxIterations: 10,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "ui_test_coverage",
						CompareOperator: OpGreaterThan,
						Value:           80,
					},
				},
			},
		},
		Tags: []string{"ux", "design", "accessibility", "creative"},
	}
}
