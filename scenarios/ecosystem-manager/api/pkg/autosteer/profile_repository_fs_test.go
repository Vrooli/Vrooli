package autosteer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileProfileRepository_CRUD(t *testing.T) {
	root := t.TempDir()
	writeMetadata(t, root, ProfileMetadataIndex{Profiles: []ProfileMetadata{}})

	repo, err := NewFileProfileRepository(root)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	profile := &AutoSteerProfile{
		Name:        "Test Profile",
		Description: "Test profile description",
		Objective: Objective{
			DimensionWeights: map[string]float64{"standards": 1.0},
			Targets:          ObjectiveTargets{MaxOpenSeverity: "warning"},
		},
		AllowedSkills: []string{"progress"},
		Budget:        Budget{MaxIterations: 3, DiminishingReturnsFloor: 0.02},
		Tags:          []string{"test", "example"},
	}

	if err := repo.CreateProfile(profile); err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	if profile.ID == "" {
		t.Fatal("expected profile ID to be set")
	}

	loaded, err := repo.GetProfile(profile.ID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if loaded.Name != profile.Name {
		t.Errorf("expected name %q, got %q", profile.Name, loaded.Name)
	}
	if loaded.Description != profile.Description {
		t.Errorf("expected description %q, got %q", profile.Description, loaded.Description)
	}

	loaded.Description = "Updated"
	loaded.Tags = []string{"updated"}
	if err := repo.UpdateProfile(profile.ID, loaded); err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}

	updated, err := repo.GetProfile(profile.ID)
	if err != nil {
		t.Fatalf("GetProfile() after update error = %v", err)
	}
	if updated.Description != "Updated" {
		t.Errorf("expected updated description, got %q", updated.Description)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "updated" {
		t.Errorf("expected updated tags, got %v", updated.Tags)
	}

	if err := repo.DeleteProfile(profile.ID); err != nil {
		t.Fatalf("DeleteProfile() error = %v", err)
	}

	if _, err := repo.GetProfile(profile.ID); err == nil {
		t.Fatal("expected error after deleting profile")
	}
}

func TestFileProfileRepository_ListAndTemplates(t *testing.T) {
	root := t.TempDir()

	template := &AutoSteerProfile{
		ID:          "template-1",
		Name:        "Template One",
		Description: "Template profile",
		Objective: Objective{
			DimensionWeights: map[string]float64{"standards": 1.0},
			Targets:          ObjectiveTargets{MaxOpenSeverity: "warning"},
		},
		AllowedSkills: []string{"progress"},
		Budget:        Budget{MaxIterations: 1, DiminishingReturnsFloor: 0.02},
		Tags:          []string{"template"},
	}
	writeProfileFile(t, root, "templates/template-one/profile.json", template)

	profile := &AutoSteerProfile{
		ID:          "profile-1",
		Name:        "Profile One",
		Description: "Regular profile",
		Objective: Objective{
			DimensionWeights: map[string]float64{"ui": 1.0},
			Targets:          ObjectiveTargets{MaxOpenSeverity: "warning"},
		},
		AllowedSkills: []string{"ux"},
		Budget:        Budget{MaxIterations: 2, DiminishingReturnsFloor: 0.02},
		Tags:          []string{"profile", "ux"},
	}
	writeProfileFile(t, root, "profiles/profile-one/profile.json", profile)

	writeMetadata(t, root, ProfileMetadataIndex{
		Profiles: []ProfileMetadata{
			{
				ID:          "template-1",
				Name:        "Template One",
				Description: "Template profile",
				Tags:        []string{"template"},
				Kind:        ProfileKindTemplate,
				File:        "templates/template-one/profile.json",
			},
			{
				ID:          "profile-1",
				Name:        "Profile One",
				Description: "Regular profile",
				Tags:        []string{"profile", "ux"},
				Kind:        ProfileKindProfile,
				File:        "profiles/profile-one/profile.json",
			},
		},
	})

	repo, err := NewFileProfileRepository(root)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	templates := repo.GetTemplates()
	if len(templates) != 1 || templates[0].ID != "template-1" {
		t.Fatalf("expected 1 template, got %v", templates)
	}

	profiles, err := repo.ListProfiles(nil)
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].ID != "profile-1" {
		t.Fatalf("expected 1 profile, got %v", profiles)
	}

	filtered, err := repo.ListProfiles([]string{"ux"})
	if err != nil {
		t.Fatalf("ListProfiles() with tags error = %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != "profile-1" {
		t.Fatalf("expected tag-filtered profile, got %v", filtered)
	}
}

func TestFileProfileRepository_ValidateMetadataMismatch(t *testing.T) {
	root := t.TempDir()

	profile := &AutoSteerProfile{
		ID:          "profile-1",
		Name:        "Profile One",
		Description: "Profile description",
		Objective: Objective{
			DimensionWeights: map[string]float64{"standards": 1.0},
			Targets:          ObjectiveTargets{MaxOpenSeverity: "warning"},
		},
		AllowedSkills: []string{"progress"},
		Budget:        Budget{MaxIterations: 1, DiminishingReturnsFloor: 0.02},
	}
	writeProfileFile(t, root, "profiles/profile-one/profile.json", profile)

	writeMetadata(t, root, ProfileMetadataIndex{
		Profiles: []ProfileMetadata{
			{
				ID:          "profile-1",
				Name:        "Different Name",
				Description: "Profile description",
				Kind:        ProfileKindProfile,
				File:        "profiles/profile-one/profile.json",
			},
		},
	})

	if _, err := NewFileProfileRepository(root); err == nil {
		t.Fatal("expected metadata mismatch error")
	}
}

func writeMetadata(t *testing.T, root string, index ProfileMetadataIndex) {
	t.Helper()
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "metadata.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}
}

func writeProfileFile(t *testing.T, root, rel string, profile *AutoSteerProfile) {
	t.Helper()
	profile.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	profile.UpdatedAt = profile.CreatedAt

	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create profile dir: %v", err)
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal profile: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}
}
