package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"scenario-authenticator/db"
	"scenario-authenticator/models"
	"scenario-authenticator/utils"
	"strconv"

	"github.com/gorilla/mux"
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
		err := rows.Scan(&user.ID, &user.Email, &user.Username, &rolesJSON, 
			&user.EmailVerified, &user.CreatedAt, &user.LastLogin)
		if err != nil {
			continue
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
	vars := mux.Vars(r)
	userID := vars["id"]
	
	var user models.User
	var rolesJSON string
	err := db.DB.QueryRow(`
		SELECT id, email, username, roles, email_verified, created_at, last_login
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	).Scan(&user.ID, &user.Email, &user.Username, &rolesJSON, 
		&user.EmailVerified, &user.CreatedAt, &user.LastLogin)
	
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
	// TODO: Implement user update logic
	utils.SendError(w, "Not implemented", http.StatusNotImplemented)
}

// DeleteUserHandler soft deletes a user
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement user deletion logic
	utils.SendError(w, "Not implemented", http.StatusNotImplemented)
}