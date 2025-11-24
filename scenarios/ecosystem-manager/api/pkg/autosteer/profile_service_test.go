package autosteer

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// Test helper to create a test database connection using testcontainers
func setupTestDB(t *testing.T) (*PostgresContainer, func()) {
	t.Helper()

	container, cleanup := SetupTestDatabase(t)
	if container == nil {
		return nil, nil
	}
	// Clean start for each test file
	_, _ = container.db.Exec("TRUNCATE auto_steer_profiles, profile_execution_state, profile_executions CASCADE")
	return container, cleanup
}

func TestProfileService_CreateProfile(t *testing.T) {
	container, cleanup := setupTestDB(t)
	if container == nil {
		return // Test skipped
	}
	defer cleanup()

	service := NewProfileService(container.db)

	t.Run("create valid profile", func(t *testing.T) {
		profile := &AutoSteerProfile{
			Name:        "Test Profile",
			Description: "A test profile",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					Mode:          ModeProgress,
					MaxIterations: 10,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "loops",
							CompareOperator: OpGreaterThan,
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
					Message:       "Build must pass",
				},
			},
			Tags: []string{"test", "balanced"},
		}

		err := service.CreateProfile(profile)
		if err != nil {
			t.Fatalf("CreateProfile() error = %v", err)
		}

		// Verify ID was generated
		if profile.ID == "" {
			t.Error("Expected ID to be generated")
		}

		// Verify timestamps were set
		if profile.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
		if profile.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}

		// Verify we can retrieve it
		retrieved, err := service.GetProfile(profile.ID)
		if err != nil {
			t.Fatalf("GetProfile() error = %v", err)
		}

		if retrieved.Name != profile.Name {
			t.Errorf("Expected name %s, got %s", profile.Name, retrieved.Name)
		}
		if len(retrieved.Phases) != 1 {
			t.Errorf("Expected 1 phase, got %d", len(retrieved.Phases))
		}
		if len(retrieved.QualityGates) != 1 {
			t.Errorf("Expected 1 quality gate, got %d", len(retrieved.QualityGates))
		}
	})

	t.Run("create profile with custom ID", func(t *testing.T) {
		customID := uuid.New().String()
		profile := &AutoSteerProfile{
			ID:          customID,
			Name:        "Custom ID Profile",
			Description: "Profile with custom ID",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
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
		}

		err := service.CreateProfile(profile)
		if err != nil {
			t.Fatalf("CreateProfile() error = %v", err)
		}

		if profile.ID != customID {
			t.Errorf("Expected ID %s, got %s", customID, profile.ID)
		}
	})

	t.Run("create profile without name - should fail", func(t *testing.T) {
		profile := &AutoSteerProfile{
			Description: "No name",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					Mode:          ModeProgress,
					MaxIterations: 10,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "loops",
							CompareOperator: OpGreaterThan,
							Value:           5,
						},
					},
				},
			},
		}

		err := service.CreateProfile(profile)
		if err == nil {
			t.Error("Expected error for profile without name")
		}
	})

	t.Run("create profile without phases - should fail", func(t *testing.T) {
		profile := &AutoSteerProfile{
			Name:        "No Phases",
			Description: "Profile without phases",
			Phases:      []SteerPhase{},
		}

		err := service.CreateProfile(profile)
		if err == nil {
			t.Error("Expected error for profile without phases")
		}
	})

	t.Run("create profile with invalid mode - should fail", func(t *testing.T) {
		profile := &AutoSteerProfile{
			Name: "Invalid Mode",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					Mode:          "invalid_mode",
					MaxIterations: 10,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "loops",
							CompareOperator: OpGreaterThan,
							Value:           5,
						},
					},
				},
			},
		}

		err := service.CreateProfile(profile)
		if err == nil {
			t.Error("Expected error for invalid mode")
		}
	})

	t.Run("create profile with zero max iterations - should fail", func(t *testing.T) {
		profile := &AutoSteerProfile{
			Name: "Zero Iterations",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					Mode:          ModeProgress,
					MaxIterations: 0,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "loops",
							CompareOperator: OpGreaterThan,
							Value:           5,
						},
					},
				},
			},
		}

		err := service.CreateProfile(profile)
		if err == nil {
			t.Error("Expected error for zero max iterations")
		}
	})
}

func TestProfileService_GetProfile(t *testing.T) {
	container, cleanup := setupTestDB(t)
	if container == nil {
		return
	}
	defer cleanup()

	service := NewProfileService(container.db)

	// Create a test profile
	profile := &AutoSteerProfile{
		Name:        "Get Test Profile",
		Description: "Profile for get tests",
		Phases: []SteerPhase{
			{
				ID:            "phase-1",
				Mode:          ModeRefactor,
				MaxIterations: 8,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "tidiness_score",
						CompareOperator: OpGreaterThan,
						Value:           85,
					},
				},
			},
		},
		Tags: []string{"refactor", "quality"},
	}

	if err := service.CreateProfile(profile); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	t.Run("get existing profile", func(t *testing.T) {
		retrieved, err := service.GetProfile(profile.ID)
		if err != nil {
			t.Fatalf("GetProfile() error = %v", err)
		}

		if retrieved.ID != profile.ID {
			t.Errorf("Expected ID %s, got %s", profile.ID, retrieved.ID)
		}
		if retrieved.Name != profile.Name {
			t.Errorf("Expected name %s, got %s", profile.Name, retrieved.Name)
		}
		if retrieved.Description != profile.Description {
			t.Errorf("Expected description %s, got %s", profile.Description, retrieved.Description)
		}
		if len(retrieved.Tags) != len(profile.Tags) {
			t.Errorf("Expected %d tags, got %d", len(profile.Tags), len(retrieved.Tags))
		}
	})

	t.Run("get non-existent profile", func(t *testing.T) {
		_, err := service.GetProfile(uuid.New().String())
		if err == nil {
			t.Error("Expected error for non-existent profile")
		}
	})
}

func TestProfileService_ListProfiles(t *testing.T) {
	container, cleanup := setupTestDB(t)
	if container == nil {
		return
	}
	defer cleanup()

	service := NewProfileService(container.db)

	// Create multiple test profiles
	profiles := []*AutoSteerProfile{
		{
			Name: "Profile A",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					Mode:          ModeProgress,
					MaxIterations: 10,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "loops",
							CompareOperator: OpGreaterThan,
							Value:           5,
						},
					},
				},
			},
			Tags: []string{"balanced", "production"},
		},
		{
			Name: "Profile B",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					Mode:          ModeTest,
					MaxIterations: 15,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "unit_test_coverage",
							CompareOperator: OpGreaterThan,
							Value:           80,
						},
					},
				},
			},
			Tags: []string{"testing", "quality"},
		},
		{
			Name: "Profile C",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					Mode:          ModeUX,
					MaxIterations: 12,
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
			Tags: []string{"ux", "production"},
		},
	}

	for _, p := range profiles {
		if err := service.CreateProfile(p); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
	}

	t.Run("list all profiles", func(t *testing.T) {
		list, err := service.ListProfiles(nil)
		if err != nil {
			t.Fatalf("ListProfiles() error = %v", err)
		}

		if len(list) != 3 {
			t.Errorf("Expected 3 profiles, got %d", len(list))
		}

		// Verify ordering (should be alphabetical by name)
		if list[0].Name != "Profile A" {
			t.Errorf("Expected first profile to be 'Profile A', got %s", list[0].Name)
		}
	})

	t.Run("list profiles with tag filter", func(t *testing.T) {
		list, err := service.ListProfiles([]string{"production"})
		if err != nil {
			t.Fatalf("ListProfiles() error = %v", err)
		}

		if len(list) != 2 {
			t.Errorf("Expected 2 profiles with 'production' tag, got %d", len(list))
		}
	})

	t.Run("list profiles with non-matching tag", func(t *testing.T) {
		list, err := service.ListProfiles([]string{"nonexistent"})
		if err != nil {
			t.Fatalf("ListProfiles() error = %v", err)
		}

		if len(list) != 0 {
			t.Errorf("Expected 0 profiles with 'nonexistent' tag, got %d", len(list))
		}
	})
}

func TestProfileService_UpdateProfile(t *testing.T) {
	container, cleanup := setupTestDB(t)
	if container == nil {
		return
	}
	defer cleanup()

	service := NewProfileService(container.db)

	// Create initial profile
	original := &AutoSteerProfile{
		Name:        "Original Profile",
		Description: "Original description",
		Phases: []SteerPhase{
			{
				ID:            "phase-1",
				Mode:          ModeProgress,
				MaxIterations: 10,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           5,
					},
				},
			},
		},
		Tags: []string{"original"},
	}

	if err := service.CreateProfile(original); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	originalCreatedAt := original.CreatedAt
	time.Sleep(10 * time.Millisecond) // Ensure timestamp difference

	t.Run("update profile successfully", func(t *testing.T) {
		updated := &AutoSteerProfile{
			Name:        "Updated Profile",
			Description: "Updated description",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					Mode:          ModeUX,
					MaxIterations: 15,
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
			Tags: []string{"updated", "ux"},
		}

		err := service.UpdateProfile(original.ID, updated)
		if err != nil {
			t.Fatalf("UpdateProfile() error = %v", err)
		}

		// Retrieve and verify
		retrieved, err := service.GetProfile(original.ID)
		if err != nil {
			t.Fatalf("GetProfile() error = %v", err)
		}

		if retrieved.Name != "Updated Profile" {
			t.Errorf("Expected name 'Updated Profile', got %s", retrieved.Name)
		}
		if retrieved.Description != "Updated description" {
			t.Errorf("Expected description 'Updated description', got %s", retrieved.Description)
		}
		if retrieved.Phases[0].Mode != ModeUX {
			t.Errorf("Expected mode %s, got %s", ModeUX, retrieved.Phases[0].Mode)
		}

		// Verify created_at preserved (allowing for DB rounding differences) and updated_at changed
		if retrieved.CreatedAt.IsZero() {
			t.Error("CreatedAt should be set")
		} else {
			diff := retrieved.CreatedAt.UTC().Sub(originalCreatedAt.UTC())
			if diff > 6*time.Hour || diff < -6*time.Hour {
				t.Errorf("CreatedAt drifted unexpectedly: original=%v retrieved=%v", originalCreatedAt, retrieved.CreatedAt)
			}
		}
		if retrieved.UpdatedAt.Before(retrieved.CreatedAt) {
			t.Errorf("UpdatedAt should be >= CreatedAt (got %v < %v)", retrieved.UpdatedAt, retrieved.CreatedAt)
		}
	})

	t.Run("update non-existent profile", func(t *testing.T) {
		nonExistent := &AutoSteerProfile{
			Name: "Non-existent",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					Mode:          ModeProgress,
					MaxIterations: 10,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "loops",
							CompareOperator: OpGreaterThan,
							Value:           5,
						},
					},
				},
			},
		}

		err := service.UpdateProfile(uuid.New().String(), nonExistent)
		if err == nil {
			t.Error("Expected error when updating non-existent profile")
		}
	})
}

func TestProfileService_DeleteProfile(t *testing.T) {
	container, cleanup := setupTestDB(t)
	if container == nil {
		return
	}
	defer cleanup()

	service := NewProfileService(container.db)

	// Create test profile
	profile := &AutoSteerProfile{
		Name: "Delete Test Profile",
		Phases: []SteerPhase{
			{
				ID:            "phase-1",
				Mode:          ModeProgress,
				MaxIterations: 10,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           5,
					},
				},
			},
		},
	}

	if err := service.CreateProfile(profile); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	t.Run("delete existing profile", func(t *testing.T) {
		err := service.DeleteProfile(profile.ID)
		if err != nil {
			t.Fatalf("DeleteProfile() error = %v", err)
		}

		// Verify it's gone
		_, err = service.GetProfile(profile.ID)
		if err == nil {
			t.Error("Expected error when getting deleted profile")
		}
	})

	t.Run("delete non-existent profile", func(t *testing.T) {
		err := service.DeleteProfile(uuid.New().String())
		if err == nil {
			t.Error("Expected error when deleting non-existent profile")
		}
	})
}

func TestProfileService_GetTemplates(t *testing.T) {
	container, cleanup := setupTestDB(t)
	if container == nil {
		return
	}
	defer cleanup()

	service := NewProfileService(container.db)

	templates := service.GetTemplates()

	if len(templates) == 0 {
		t.Error("Expected at least one built-in template")
	}

	// Verify templates have valid structure
	for _, template := range templates {
		if template.Name == "" {
			t.Error("Template should have a name")
		}
		if len(template.Phases) == 0 {
			t.Errorf("Template %s should have phases", template.Name)
		}

		// Verify all phases are valid
		for i, phase := range template.Phases {
			if !phase.Mode.IsValid() {
				t.Errorf("Template %s phase %d has invalid mode", template.Name, i)
			}
			if phase.MaxIterations <= 0 {
				t.Errorf("Template %s phase %d has invalid max iterations", template.Name, i)
			}
			if len(phase.StopConditions) == 0 {
				t.Errorf("Template %s phase %d has no stop conditions", template.Name, i)
			}
		}
	}
}

func TestProfileService_ValidateCondition(t *testing.T) {
	container, cleanup := setupTestDB(t)
	if container == nil {
		return
	}
	defer cleanup()

	service := NewProfileService(container.db)

	tests := []struct {
		name      string
		condition StopCondition
		wantErr   bool
	}{
		{
			name: "valid simple condition",
			condition: StopCondition{
				Type:            ConditionTypeSimple,
				Metric:          "loops",
				CompareOperator: OpGreaterThan,
				Value:           5,
			},
			wantErr: false,
		},
		{
			name: "simple condition without metric",
			condition: StopCondition{
				Type:            ConditionTypeSimple,
				CompareOperator: OpGreaterThan,
				Value:           5,
			},
			wantErr: true,
		},
		{
			name: "simple condition without operator",
			condition: StopCondition{
				Type:   ConditionTypeSimple,
				Metric: "loops",
				Value:  5,
			},
			wantErr: true,
		},
		{
			name: "valid compound condition",
			condition: StopCondition{
				Type:     ConditionTypeCompound,
				Operator: LogicalAND,
				Conditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           5,
					},
					{
						Type:            ConditionTypeSimple,
						Metric:          "accessibility_score",
						CompareOperator: OpGreaterThan,
						Value:           90,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "compound condition without operator",
			condition: StopCondition{
				Type: ConditionTypeCompound,
				Conditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           5,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "compound condition without sub-conditions",
			condition: StopCondition{
				Type:       ConditionTypeCompound,
				Operator:   LogicalAND,
				Conditions: []StopCondition{},
			},
			wantErr: true,
		},
		{
			name: "simple condition with unsupported metric",
			condition: StopCondition{
				Type:            ConditionTypeSimple,
				Metric:          "unknown_metric",
				CompareOperator: OpGreaterThan,
				Value:           1,
			},
			wantErr: true,
		},
		{
			name: "compound condition with invalid operator",
			condition: StopCondition{
				Type:     ConditionTypeCompound,
				Operator: "XOR",
				Conditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           1,
					},
					{
						Type:            ConditionTypeSimple,
						Metric:          "operational_targets_percentage",
						CompareOperator: OpGreaterThanEquals,
						Value:           50,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "nested compound condition",
			condition: StopCondition{
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
								Metric:          "loops",
								CompareOperator: OpGreaterThan,
								Value:           3,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "accessibility_score",
								CompareOperator: OpGreaterThan,
								Value:           90,
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateCondition(tt.condition)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCondition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
