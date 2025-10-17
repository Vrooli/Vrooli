package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ProfileManager handles all profile-related database operations
type ProfileManager struct {
	db     *sql.DB
	logger *Logger
}

// NewProfileManager creates a new profile manager
func NewProfileManager(db *sql.DB, logger *Logger) *ProfileManager {
	return &ProfileManager{
		db:     db,
		logger: logger,
	}
}

// ListProfiles returns all profiles from the database
func (pm *ProfileManager) ListProfiles() ([]Profile, error) {
	query := `
		SELECT id, name, display_name, description, metadata, resources, scenarios, 
		       auto_browser, environment_vars, idle_shutdown_minutes, dependencies, 
		       status, created_at, updated_at
		FROM profiles 
		ORDER BY created_at DESC
	`
	
	rows, err := pm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}
	defer rows.Close()
	
	var profiles []Profile
	for rows.Next() {
		var p Profile
		var metadataJSON, resourcesJSON, scenariosJSON, autoBrowserJSON, envVarsJSON, dependenciesJSON []byte
		var idleShutdown sql.NullInt64
		
		err := rows.Scan(
			&p.ID, &p.Name, &p.DisplayName, &p.Description,
			&metadataJSON, &resourcesJSON, &scenariosJSON, &autoBrowserJSON,
			&envVarsJSON, &idleShutdown, &dependenciesJSON,
			&p.Status, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			pm.logger.Error("Failed to scan profile row", err)
			continue
		}
		
		// Parse JSON fields
		json.Unmarshal(metadataJSON, &p.Metadata)
		json.Unmarshal(resourcesJSON, &p.Resources)
		json.Unmarshal(scenariosJSON, &p.Scenarios)
		json.Unmarshal(autoBrowserJSON, &p.AutoBrowser)
		json.Unmarshal(envVarsJSON, &p.EnvironmentVars)
		json.Unmarshal(dependenciesJSON, &p.Dependencies)
		
		if idleShutdown.Valid {
			idle := int(idleShutdown.Int64)
			p.IdleShutdown = &idle
		}
		
		profiles = append(profiles, p)
	}
	
	return profiles, nil
}

// GetProfile returns a specific profile by name
func (pm *ProfileManager) GetProfile(name string) (*Profile, error) {
	query := `
		SELECT id, name, display_name, description, metadata, resources, scenarios, 
		       auto_browser, environment_vars, idle_shutdown_minutes, dependencies, 
		       status, created_at, updated_at
		FROM profiles 
		WHERE name = $1
	`
	
	row := pm.db.QueryRow(query, name)
	
	var p Profile
	var metadataJSON, resourcesJSON, scenariosJSON, autoBrowserJSON, envVarsJSON, dependenciesJSON []byte
	var idleShutdown sql.NullInt64
	
	err := row.Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.Description,
		&metadataJSON, &resourcesJSON, &scenariosJSON, &autoBrowserJSON,
		&envVarsJSON, &idleShutdown, &dependenciesJSON,
		&p.Status, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	
	// Parse JSON fields
	json.Unmarshal(metadataJSON, &p.Metadata)
	json.Unmarshal(resourcesJSON, &p.Resources)
	json.Unmarshal(scenariosJSON, &p.Scenarios)
	json.Unmarshal(autoBrowserJSON, &p.AutoBrowser)
	json.Unmarshal(envVarsJSON, &p.EnvironmentVars)
	json.Unmarshal(dependenciesJSON, &p.Dependencies)
	
	if idleShutdown.Valid {
		idle := int(idleShutdown.Int64)
		p.IdleShutdown = &idle
	}
	
	return &p, nil
}

// CreateProfile creates a new profile
func (pm *ProfileManager) CreateProfile(profileData map[string]interface{}) (*Profile, error) {
	// Extract and validate required fields
	name, ok := profileData["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("profile name is required")
	}
	
	displayName, _ := profileData["display_name"].(string)
	if displayName == "" {
		displayName = name
	}
	
	description, _ := profileData["description"].(string)
	
	// Generate UUID for new profile
	profileID := uuid.New().String()
	
	// Extract arrays and objects
	resources, _ := profileData["resources"].([]interface{})
	scenarios, _ := profileData["scenarios"].([]interface{})
	autoBrowser, _ := profileData["auto_browser"].([]interface{})
	dependencies, _ := profileData["dependencies"].([]interface{})
	envVars, _ := profileData["environment_vars"].(map[string]interface{})
	metadata, _ := profileData["metadata"].(map[string]interface{})
	
	// Convert to JSON
	resourcesJSON, _ := json.Marshal(resources)
	scenariosJSON, _ := json.Marshal(scenarios)
	autoBrowserJSON, _ := json.Marshal(autoBrowser)
	dependenciesJSON, _ := json.Marshal(dependencies)
	envVarsJSON, _ := json.Marshal(envVars)
	metadataJSON, _ := json.Marshal(metadata)
	
	// Handle idle shutdown
	var idleShutdown *int
	if idle, ok := profileData["idle_shutdown_minutes"].(float64); ok {
		idleVal := int(idle)
		idleShutdown = &idleVal
	}
	
	now := time.Now().UTC()
	
	query := `
		INSERT INTO profiles (
			id, name, display_name, description, metadata, resources, scenarios,
			auto_browser, environment_vars, idle_shutdown_minutes, dependencies,
			status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`
	
	_, err := pm.db.Exec(
		query, profileID, name, displayName, description, metadataJSON,
		resourcesJSON, scenariosJSON, autoBrowserJSON, envVarsJSON,
		idleShutdown, dependenciesJSON, "inactive", now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return nil, fmt.Errorf("profile with name '%s' already exists", name)
		}
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}
	
	pm.logger.Info(fmt.Sprintf("Created profile: %s (%s)", name, profileID))
	
	// Return the created profile
	return pm.GetProfile(name)
}

// UpdateProfile updates an existing profile
func (pm *ProfileManager) UpdateProfile(name string, updates map[string]interface{}) (*Profile, error) {
	// Check if profile exists
	existingProfile, err := pm.GetProfile(name)
	if err != nil {
		return nil, err
	}
	
	// Build dynamic update query
	var setParts []string
	var args []interface{}
	argIndex := 1
	
	for key, value := range updates {
		switch key {
		case "display_name", "description", "status":
			setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argIndex))
			args = append(args, value)
			argIndex++
		case "resources", "scenarios", "auto_browser", "dependencies", "metadata", "environment_vars":
			jsonValue, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal %s: %w", key, err)
			}
			setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argIndex))
			args = append(args, jsonValue)
			argIndex++
		case "idle_shutdown_minutes":
			if value == nil {
				setParts = append(setParts, fmt.Sprintf("%s = NULL", key))
			} else {
				setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argIndex))
				args = append(args, value)
				argIndex++
			}
		}
	}
	
	if len(setParts) == 0 {
		return existingProfile, nil // No updates to apply
	}
	
	// Add updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now().UTC())
	argIndex++
	
	// Add WHERE clause
	args = append(args, existingProfile.ID)
	
	query := fmt.Sprintf(
		"UPDATE profiles SET %s WHERE id = $%d",
		strings.Join(setParts, ", "),
		argIndex,
	)
	
	_, err = pm.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}
	
	pm.logger.Info(fmt.Sprintf("Updated profile: %s", name))
	
	// Return updated profile
	return pm.GetProfile(name)
}

// DeleteProfile deletes a profile
func (pm *ProfileManager) DeleteProfile(name string) error {
	// Check if profile exists and is not active
	profile, err := pm.GetProfile(name)
	if err != nil {
		return err
	}
	
	if profile.Status == "active" {
		return fmt.Errorf("cannot delete active profile '%s'. Deactivate it first", name)
	}
	
	// Delete the profile
	_, err = pm.db.Exec("DELETE FROM profiles WHERE id = $1", profile.ID)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}
	
	pm.logger.Info(fmt.Sprintf("Deleted profile: %s", name))
	return nil
}

// GetActiveProfile returns the currently active profile
func (pm *ProfileManager) GetActiveProfile() (*Profile, error) {
	query := `
		SELECT ap.profile_id, p.name, p.display_name, p.description, p.metadata, 
		       p.resources, p.scenarios, p.auto_browser, p.environment_vars, 
		       p.idle_shutdown_minutes, p.dependencies, p.status, 
		       p.created_at, p.updated_at, ap.activated_at
		FROM active_profile ap
		LEFT JOIN profiles p ON ap.profile_id = p.id
		WHERE ap.id = 1 AND ap.profile_id IS NOT NULL
	`
	
	row := pm.db.QueryRow(query)
	
	var p Profile
	var metadataJSON, resourcesJSON, scenariosJSON, autoBrowserJSON, envVarsJSON, dependenciesJSON []byte
	var idleShutdown sql.NullInt64
	var activatedAt sql.NullTime
	var profileID sql.NullString
	
	err := row.Scan(
		&profileID, &p.Name, &p.DisplayName, &p.Description,
		&metadataJSON, &resourcesJSON, &scenariosJSON, &autoBrowserJSON,
		&envVarsJSON, &idleShutdown, &dependenciesJSON,
		&p.Status, &p.CreatedAt, &p.UpdatedAt, &activatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No active profile
		}
		return nil, fmt.Errorf("failed to get active profile: %w", err)
	}
	
	if !profileID.Valid {
		return nil, nil // No active profile
	}
	
	p.ID = profileID.String
	
	// Parse JSON fields
	json.Unmarshal(metadataJSON, &p.Metadata)
	json.Unmarshal(resourcesJSON, &p.Resources)
	json.Unmarshal(scenariosJSON, &p.Scenarios)
	json.Unmarshal(autoBrowserJSON, &p.AutoBrowser)
	json.Unmarshal(envVarsJSON, &p.EnvironmentVars)
	json.Unmarshal(dependenciesJSON, &p.Dependencies)
	
	if idleShutdown.Valid {
		idle := int(idleShutdown.Int64)
		p.IdleShutdown = &idle
	}
	
	return &p, nil
}

// SetActiveProfile sets a profile as active
func (pm *ProfileManager) SetActiveProfile(profileID string) error {
	now := time.Now().UTC()
	
	// Update the active profile record
	_, err := pm.db.Exec(`
		INSERT INTO active_profile (id, profile_id, activated_at)
		VALUES (1, $1, $2)
		ON CONFLICT (id) DO UPDATE SET 
			profile_id = EXCLUDED.profile_id,
			activated_at = EXCLUDED.activated_at
	`, profileID, now)
	
	if err != nil {
		return fmt.Errorf("failed to set active profile: %w", err)
	}
	
	// Update profile status to active
	_, err = pm.db.Exec(`
		UPDATE profiles SET status = 'active', updated_at = $1 WHERE id = $2
	`, now, profileID)
	
	if err != nil {
		return fmt.Errorf("failed to update profile status: %w", err)
	}
	
	return nil
}

// ClearActiveProfile clears the active profile
func (pm *ProfileManager) ClearActiveProfile() error {
	// Get current active profile to update its status
	activeProfile, err := pm.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("failed to get active profile: %w", err)
	}
	
	if activeProfile != nil {
		// Update profile status to inactive
		_, err = pm.db.Exec(`
			UPDATE profiles SET status = 'inactive', updated_at = $1 WHERE id = $2
		`, time.Now().UTC(), activeProfile.ID)
		
		if err != nil {
			return fmt.Errorf("failed to update profile status: %w", err)
		}
	}
	
	// Clear active profile record
	_, err = pm.db.Exec(`
		UPDATE active_profile SET profile_id = NULL, activated_at = NULL WHERE id = 1
	`)
	
	if err != nil {
		return fmt.Errorf("failed to clear active profile: %w", err)
	}
	
	return nil
}