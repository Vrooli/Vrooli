package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

// PermissionLevel represents the level of access required
type PermissionLevel int

const (
	PermissionRead PermissionLevel = iota
	PermissionWrite
)

// CheckGraphPermission checks if a user has the required permission level for a graph
func CheckGraphPermission(db *sql.DB, graphID string, userID string, level PermissionLevel) (bool, error) {
	var createdBy string
	var permissionsJSON sql.NullString

	err := db.QueryRow(`
		SELECT created_by, permissions
		FROM graphs
		WHERE id = $1
	`, graphID).Scan(&createdBy, &permissionsJSON)

	if err == sql.ErrNoRows {
		return false, fmt.Errorf("graph not found")
	}
	if err != nil {
		return false, err
	}

	// Creator always has full access
	if createdBy == userID {
		log.Printf("[PERMISSION CHECK] Graph %s: User %s is creator, granting access", graphID, userID)
		return true, nil
	}

	// Parse permissions
	var permissions GraphPermissions
	if permissionsJSON.Valid && permissionsJSON.String != "" {
		if err := json.Unmarshal([]byte(permissionsJSON.String), &permissions); err != nil {
			log.Printf("Error parsing permissions for graph %s: %v", graphID, err)
			// If permissions can't be parsed, default to private
			return false, nil
		}
		log.Printf("[PERMISSION CHECK] Graph %s: Permissions loaded from DB: %+v", graphID, permissions)
	} else {
		// No permissions set - default to private (only creator has access)
		permissions.Public = false
		log.Printf("[PERMISSION CHECK] Graph %s: No permissions set, defaulting to PRIVATE (creator=%s, user=%s)", graphID, createdBy, userID)
	}

	// Check read permissions
	if level == PermissionRead {
		// Public graphs are readable by anyone
		if permissions.Public {
			log.Printf("[PERMISSION CHECK] Graph %s: Graph is PUBLIC, granting read access to %s", graphID, userID)
			return true, nil
		}

		// Check if user is in allowed users list
		for _, allowedUser := range permissions.AllowedUsers {
			if allowedUser == userID {
				log.Printf("[PERMISSION CHECK] Graph %s: User %s found in AllowedUsers, granting read access", graphID, userID)
				return true, nil
			}
		}

		// Check if user is in editors list (editors can also read)
		for _, editor := range permissions.Editors {
			if editor == userID {
				log.Printf("[PERMISSION CHECK] Graph %s: User %s found in Editors, granting read access", graphID, userID)
				return true, nil
			}
		}

		log.Printf("[PERMISSION CHECK] Graph %s: DENYING read access to %s (not creator, not public, not in allowed/editors)", graphID, userID)
		return false, nil
	}

	// Check write permissions
	if level == PermissionWrite {
		// Check if user is in editors list
		for _, editor := range permissions.Editors {
			if editor == userID {
				return true, nil
			}
		}

		return false, nil
	}

	return false, nil
}

// GetGraphOwner retrieves the creator/owner of a graph
func GetGraphOwner(db *sql.DB, graphID string) (string, error) {
	var createdBy string
	err := db.QueryRow(`
		SELECT created_by
		FROM graphs
		WHERE id = $1
	`, graphID).Scan(&createdBy)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("graph not found")
	}
	if err != nil {
		return "", err
	}

	return createdBy, nil
}

// GetGraphPermissions retrieves the permissions for a graph
func GetGraphPermissions(db *sql.DB, graphID string) (*GraphPermissions, error) {
	var permissionsJSON sql.NullString

	err := db.QueryRow(`
		SELECT permissions
		FROM graphs
		WHERE id = $1
	`, graphID).Scan(&permissionsJSON)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("graph not found")
	}
	if err != nil {
		return nil, err
	}

	var permissions GraphPermissions
	if permissionsJSON.Valid && permissionsJSON.String != "" {
		if err := json.Unmarshal([]byte(permissionsJSON.String), &permissions); err != nil {
			return nil, err
		}
	} else {
		// Default permissions
		permissions.Public = true
	}

	return &permissions, nil
}

// UpdateGraphPermissions updates the permissions for a graph
func UpdateGraphPermissions(db *sql.DB, graphID string, userID string, permissions *GraphPermissions) error {
	// First check if user is the owner
	owner, err := GetGraphOwner(db, graphID)
	if err != nil {
		return err
	}

	if owner != userID {
		return fmt.Errorf("only the graph owner can update permissions")
	}

	permissionsJSON, err := json.Marshal(permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	_, err = db.Exec(`
		UPDATE graphs
		SET permissions = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, permissionsJSON, graphID)

	return err
}
