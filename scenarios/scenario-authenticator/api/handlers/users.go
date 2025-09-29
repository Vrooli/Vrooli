package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/models"
	"scenario-authenticator/utils"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// GetUsersHandler lists users with optional filters
func GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	role := r.URL.Query().Get("role")
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Build query
	query := `SELECT id, email, username, roles, email_verified, created_at, last_login 
	          FROM users WHERE deleted_at IS NULL`
	args := []interface{}{}

	if role != "" {
		query += " AND roles::jsonb ? $1"
		args = append(args, role)
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	// Execute query
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		utils.SendError(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Parse results
	users := []models.User{}
	for rows.Next() {
		var user models.User
		var rolesJSON string
		var usernameNullable sql.NullString
		err := rows.Scan(&user.ID, &user.Email, &usernameNullable, &rolesJSON,
			&user.EmailVerified, &user.CreatedAt, &user.LastLogin)
		if err != nil {
			continue
		}

		// Handle nullable username
		if usernameNullable.Valid {
			user.Username = usernameNullable.String
		}

		// Parse roles
		if err := json.Unmarshal([]byte(rolesJSON), &user.Roles); err != nil {
			user.Roles = []string{"user"}
		}

		users = append(users, user)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL"
	if role != "" {
		countQuery += " AND roles::jsonb ? $1"
		db.DB.QueryRow(countQuery, role).Scan(&total)
	} else {
		db.DB.QueryRow(countQuery).Scan(&total)
	}

	utils.SendJSON(w, map[string]interface{}{
		"users": users,
		"total": total,
	}, http.StatusOK)
}

// GetUserHandler gets a specific user by ID
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	var user models.User
	var rolesJSON string
	var usernameNullable sql.NullString
	err := db.DB.QueryRow(`
		SELECT id, email, username, roles, email_verified, created_at, last_login
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	).Scan(&user.ID, &user.Email, &usernameNullable, &rolesJSON,
		&user.EmailVerified, &user.CreatedAt, &user.LastLogin)

	// Handle nullable username
	if usernameNullable.Valid {
		user.Username = usernameNullable.String
	}

	if err != nil {
		if err == sql.ErrNoRows {
			utils.SendError(w, "User not found", http.StatusNotFound)
		} else {
			utils.SendError(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Parse roles
	if err := json.Unmarshal([]byte(rolesJSON), &user.Roles); err != nil {
		user.Roles = []string{"user"}
	}

	utils.SendJSON(w, user, http.StatusOK)
}

// UpdateUserHandler updates user information
func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}

	userID := chi.URLParam(r, "id")
	if userID == "" {
		utils.SendError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Username != nil {
		trimmed := strings.TrimSpace(*req.Username)
		if trimmed == "" {
			setClauses = append(setClauses, "username = NULL")
		} else {
			setClauses = append(setClauses, fmt.Sprintf("username = $%d", argPos))
			args = append(args, trimmed)
			argPos++
		}
	}

	if req.EmailVerified != nil {
		setClauses = append(setClauses, fmt.Sprintf("email_verified = $%d", argPos))
		args = append(args, *req.EmailVerified)
		argPos++
	}

	if req.Roles != nil {
		normalizedRoles := normalizeRoles(*req.Roles)
		if len(normalizedRoles) == 0 {
			utils.SendError(w, "Roles cannot be empty", http.StatusBadRequest)
			return
		}

		rolesJSON, err := json.Marshal(normalizedRoles)
		if err != nil {
			log.Printf("Failed to marshal roles for user %s: %v", userID, err)
			utils.SendError(w, "Failed to process roles", http.StatusInternalServerError)
			return
		}

		setClauses = append(setClauses, fmt.Sprintf("roles = $%d::jsonb", argPos))
		args = append(args, string(rolesJSON))
		argPos++
	}

	if len(setClauses) == 0 {
		utils.SendError(w, "No valid fields provided for update", http.StatusBadRequest)
		return
	}

	query := fmt.Sprintf("UPDATE users SET %s, updated_at = CURRENT_TIMESTAMP WHERE id = $%d", strings.Join(setClauses, ", "), argPos)
	args = append(args, userID)

	result, err := db.DB.Exec(query, args...)
	if err != nil {
		log.Printf("Failed to update user %s: %v", userID, err)
		utils.SendError(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Failed to read rows affected for user %s update: %v", userID, err)
	}

	if rowsAffected == 0 {
		utils.SendError(w, "User not found", http.StatusNotFound)
		return
	}

	var user models.User
	var rolesJSON string
	var usernameNullable sql.NullString
	err = db.DB.QueryRow(`
		SELECT id, email, username, roles, email_verified, created_at, last_login
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	).Scan(&user.ID, &user.Email, &usernameNullable, &rolesJSON, &user.EmailVerified, &user.CreatedAt, &user.LastLogin)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.SendError(w, "User not found", http.StatusNotFound)
		} else {
			log.Printf("Failed to fetch updated user %s: %v", userID, err)
			utils.SendError(w, "Failed to fetch updated user", http.StatusInternalServerError)
		}
		return
	}

	if usernameNullable.Valid {
		user.Username = usernameNullable.String
	}

	if err := json.Unmarshal([]byte(rolesJSON), &user.Roles); err != nil {
		user.Roles = []string{"user"}
	}

	metadata := map[string]interface{}{}
	if req.Roles != nil {
		metadata["roles"] = user.Roles
	}
	if req.Username != nil {
		metadata["username"] = user.Username
	}
	if req.EmailVerified != nil {
		metadata["email_verified"] = user.EmailVerified
	}

	if len(metadata) > 0 {
		if claimsVal := r.Context().Value("claims"); claimsVal != nil {
			if claims, ok := claimsVal.(*models.Claims); ok {
				logAuthEvent(claims.UserID, "user.updated", auth.GetClientIP(r), r.Header.Get("User-Agent"), true, metadata)
			}
		}
	}

	utils.SendJSON(w, map[string]interface{}{
		"success": true,
		"user":    user,
	}, http.StatusOK)
}

// DeleteUserHandler soft deletes a user
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}

	userID := chi.URLParam(r, "id")
	if userID == "" {
		utils.SendError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	var actingUserID string
	if claimsVal := r.Context().Value("claims"); claimsVal != nil {
		if claims, ok := claimsVal.(*models.Claims); ok {
			actingUserID = claims.UserID
			if claims.UserID == userID {
				utils.SendError(w, "You cannot delete your own account", http.StatusBadRequest)
				return
			}
		}
	}

	var (
		email     string
		deletedAt sql.NullTime
	)

	err := db.DB.QueryRow(
		"SELECT email, deleted_at FROM users WHERE id = $1",
		userID,
	).Scan(&email, &deletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.SendError(w, "User not found", http.StatusNotFound)
		} else {
			log.Printf("Failed to lookup user %s for deletion: %v", userID, err)
			utils.SendError(w, "Failed to delete user", http.StatusInternalServerError)
		}
		return
	}

	if deletedAt.Valid {
		utils.SendError(w, "User already deleted", http.StatusGone)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("Failed to open transaction for user delete %s: %v", userID, err)
		utils.SendError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()

	if _, err = tx.Exec("UPDATE users SET deleted_at = $1, updated_at = $1 WHERE id = $2", now, userID); err != nil {
		log.Printf("Failed to soft delete user %s: %v", userID, err)
		utils.SendError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	if _, err = tx.Exec("DELETE FROM user_roles WHERE user_id = $1", userID); err != nil {
		log.Printf("Failed to remove user %s role mappings: %v", userID, err)
		utils.SendError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	if _, err = tx.Exec("UPDATE api_keys SET revoked_at = $1, updated_at = $1 WHERE user_id = $2 AND revoked_at IS NULL", now, userID); err != nil {
		log.Printf("Failed to revoke API keys for user %s: %v", userID, err)
		utils.SendError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	if _, err = tx.Exec("UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL", now, userID); err != nil {
		log.Printf("Failed to revoke refresh tokens for user %s: %v", userID, err)
		utils.SendError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	if tableExists(tx, "application_users") {
		if _, err = tx.Exec("UPDATE application_users SET is_active = FALSE WHERE user_id = $1", userID); err != nil {
			log.Printf("Failed to deactivate application memberships for user %s: %v", userID, err)
			utils.SendError(w, "Failed to delete user", http.StatusInternalServerError)
			return
		}
	}

	if tableExists(tx, "application_sessions") {
		if _, err = tx.Exec("UPDATE application_sessions SET ended_at = $1 WHERE user_id = $2 AND ended_at IS NULL", now, userID); err != nil {
			log.Printf("Failed to end application sessions for user %s: %v", userID, err)
			utils.SendError(w, "Failed to delete user", http.StatusInternalServerError)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit delete for user %s: %v", userID, err)
		utils.SendError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	// Clear any active cache-backed sessions
	auth.ClearUserSessions(userID)

	metadata := map[string]interface{}{
		"target_user_id":    userID,
		"target_user_email": email,
	}
	if actingUserID != "" {
		logAuthEvent(actingUserID, "user.deleted", auth.GetClientIP(r), r.Header.Get("User-Agent"), true, metadata)
	}

	utils.SendJSON(w, map[string]interface{}{
		"success": true,
		"message": "User deleted",
	}, http.StatusOK)
}

func tableExists(tx *sql.Tx, table string) bool {
	var exists bool
	if err := tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = $1
		)
	`, table).Scan(&exists); err != nil {
		log.Printf("Failed to check table existence for %s: %v", table, err)
		return false
	}
	return exists
}

// normalizeRoles cleans role names, ensures uniqueness, and enforces baseline access
func normalizeRoles(input []string) []string {
	roleSet := map[string]struct{}{}
	for _, rawRole := range input {
		trimmed := strings.TrimSpace(strings.ToLower(rawRole))
		if trimmed != "" {
			roleSet[trimmed] = struct{}{}
		}
	}

	// Always ensure standard user role is present for baseline access
	roleSet["user"] = struct{}{}

	normalized := make([]string, 0, len(roleSet))
	for role := range roleSet {
		normalized = append(normalized, role)
	}

	sort.Strings(normalized)
	return normalized
}
