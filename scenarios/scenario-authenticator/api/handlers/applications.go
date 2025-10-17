package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/models"
	"scenario-authenticator/utils"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	scenarioNamePattern  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	allowedScenarioTypes = map[string]struct{}{
		"api":      {},
		"ui":       {},
		"workflow": {},
		"hybrid":   {},
	}
)

// Application represents a scenario/app that uses this auth service
type Application struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	DisplayName    string     `json:"display_name"`
	Description    string     `json:"description,omitempty"`
	ScenarioType   string     `json:"scenario_type"`
	AllowedOrigins []string   `json:"allowed_origins"`
	RedirectURIs   []string   `json:"redirect_uris"`
	Permissions    []string   `json:"permissions"`
	RateLimit      int        `json:"rate_limit"`
	MaxUsers       *int       `json:"max_users,omitempty"`
	IsActive       bool       `json:"is_active"`
	LastAccessed   *time.Time `json:"last_accessed,omitempty"`
	TotalUsers     int        `json:"total_users"`
	TotalAuths     int        `json:"total_authentications"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func normalizeScenarioType(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "api", nil
	}

	tokens := strings.Split(trimmed, ",")
	set := make(map[string]struct{})
	for _, token := range tokens {
		normalized := strings.TrimSpace(strings.ToLower(token))
		if normalized == "" {
			continue
		}
		if _, ok := allowedScenarioTypes[normalized]; !ok {
			return "", fmt.Errorf("unsupported scenario type: %s", normalized)
		}
		set[normalized] = struct{}{}
	}

	if len(set) == 0 {
		return "api", nil
	}

	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}

	if len(values) == 1 {
		return values[0], nil
	}

	sort.Strings(values)
	return strings.Join(values, ","), nil
}

// ApplicationStats represents application statistics
type ApplicationStats struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	DisplayName      string     `json:"display_name"`
	IsActive         bool       `json:"is_active"`
	TotalUsers       int        `json:"total_users"`
	ActiveSessions   int        `json:"active_sessions"`
	TotalEventsToday int        `json:"total_events_today"`
	RateLimit        int        `json:"rate_limit"`
	LastAccessed     *time.Time `json:"last_accessed"`
	CreatedAt        time.Time  `json:"created_at"`
}

// RegisterAppRequest represents app registration payload
type RegisterAppRequest struct {
	Name           string   `json:"name"`
	DisplayName    string   `json:"display_name"`
	Description    string   `json:"description"`
	ScenarioType   string   `json:"scenario_type"`
	AllowedOrigins []string `json:"allowed_origins"`
	RedirectURIs   []string `json:"redirect_uris"`
	RateLimit      *int     `json:"rate_limit"`
	MaxUsers       *int     `json:"max_users"`
}

// AppCredentials represents API credentials for an application
type AppCredentials struct {
	ApplicationID string `json:"application_id"`
	APIKey        string `json:"api_key"`
	APISecret     string `json:"api_secret"`
}

// GetApplicationsHandler lists all registered applications
func GetApplicationsHandler(w http.ResponseWriter, r *http.Request) {
	showStats := r.URL.Query().Get("stats") == "true"

	var rows *sql.Rows
	var err error

	if showStats {
		// Get application statistics
		rows, err = db.DB.Query(`
			SELECT 
				a.id, a.name, a.display_name, a.is_active,
				COALESCE(COUNT(DISTINCT au.user_id), 0) as total_users,
				COALESCE(COUNT(DISTINCT CASE WHEN as2.ended_at IS NULL THEN as2.id END), 0) as active_sessions,
				COALESCE(COUNT(DISTINCT CASE WHEN al.created_at >= CURRENT_DATE THEN al.id END), 0) as total_events_today,
				a.rate_limit, a.last_accessed, a.created_at
			FROM applications a
			LEFT JOIN application_users au ON a.id = au.application_id AND au.is_active = true
			LEFT JOIN application_sessions as2 ON a.id = as2.application_id AND as2.ended_at IS NULL
			LEFT JOIN audit_logs al ON a.id = al.application_id
			GROUP BY a.id, a.name, a.display_name, a.is_active, a.rate_limit, a.last_accessed, a.created_at
			ORDER BY a.created_at DESC
		`)
	} else {
		// Get basic application info
		rows, err = db.DB.Query(`
			SELECT 
				id, name, display_name, description, scenario_type,
				allowed_origins, redirect_uris, permissions, rate_limit,
				max_users, is_active, last_accessed, total_users,
				total_authentications, created_at, updated_at
			FROM applications
			ORDER BY created_at DESC
		`)
	}

	if err != nil {
		log.Printf("Failed to fetch applications: %v", err)
		utils.SendError(w, "Failed to fetch applications", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	if showStats {
		var apps []ApplicationStats
		for rows.Next() {
			var app ApplicationStats
			var lastAccessed sql.NullString

			err := rows.Scan(
				&app.ID, &app.Name, &app.DisplayName, &app.IsActive,
				&app.TotalUsers, &app.ActiveSessions, &app.TotalEventsToday,
				&app.RateLimit, &lastAccessed, &app.CreatedAt,
			)
			if err != nil {
				log.Printf("Failed to scan application stats: %v", err)
				continue
			}

			if lastAccessed.Valid {
				parsed, err := time.Parse("2006-01-02 15:04:05", lastAccessed.String)
				if err == nil {
					app.LastAccessed = &parsed
				}
			}

			apps = append(apps, app)
		}

		utils.SendJSON(w, map[string]interface{}{
			"applications": apps,
			"total":        len(apps),
		}, http.StatusOK)
	} else {
		var apps []Application
		for rows.Next() {
			var app Application
			var originsJSON, urisJSON, permsJSON, description, maxUsers sql.NullString
			var lastAccessed sql.NullString

			err := rows.Scan(
				&app.ID, &app.Name, &app.DisplayName, &description, &app.ScenarioType,
				&originsJSON, &urisJSON, &permsJSON, &app.RateLimit,
				&maxUsers, &app.IsActive, &lastAccessed, &app.TotalUsers,
				&app.TotalAuths, &app.CreatedAt, &app.UpdatedAt,
			)
			if err != nil {
				log.Printf("Failed to scan application: %v", err)
				continue
			}

			if description.Valid {
				app.Description = description.String
			}

			if maxUsers.Valid {
				if val, err := strconv.Atoi(maxUsers.String); err == nil {
					app.MaxUsers = &val
				}
			}

			if lastAccessed.Valid {
				parsed, err := time.Parse("2006-01-02 15:04:05", lastAccessed.String)
				if err == nil {
					app.LastAccessed = &parsed
				}
			}

			// Parse JSON arrays
			if originsJSON.Valid && originsJSON.String != "" {
				json.Unmarshal([]byte(originsJSON.String), &app.AllowedOrigins)
			}
			if urisJSON.Valid && urisJSON.String != "" {
				json.Unmarshal([]byte(urisJSON.String), &app.RedirectURIs)
			}
			if permsJSON.Valid && permsJSON.String != "" {
				json.Unmarshal([]byte(permsJSON.String), &app.Permissions)
			}

			apps = append(apps, app)
		}

		utils.SendJSON(w, map[string]interface{}{
			"applications": apps,
			"total":        len(apps),
		}, http.StatusOK)
	}
}

// RegisterApplicationHandler registers a new application
func RegisterApplicationHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.DisplayName == "" {
		utils.SendError(w, "Name and display name are required", http.StatusBadRequest)
		return
	}

	// Validate scenario name format matches CLI scenarios
	if !scenarioNamePattern.MatchString(req.Name) {
		utils.SendError(w, "Application name must use lowercase letters, numbers, or hyphens", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.RateLimit == nil {
		defaultRate := 1000
		req.RateLimit = &defaultRate
	}

	normalizedType, err := normalizeScenarioType(req.ScenarioType)
	if err != nil {
		utils.SendError(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.ScenarioType = normalizedType

	if req.AllowedOrigins == nil {
		req.AllowedOrigins = []string{}
	}

	if req.RedirectURIs == nil {
		req.RedirectURIs = []string{}
	}

	// Check if application name already exists
	var existingID string
	err = db.DB.QueryRow("SELECT id FROM applications WHERE name = $1", req.Name).Scan(&existingID)
	if err == nil {
		utils.SendError(w, "Application name already exists", http.StatusConflict)
		return
	} else if err != sql.ErrNoRows {
		log.Printf("Database error checking application: %v", err)
		utils.SendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Generate API credentials
	apiKey := "ak_" + utils.GenerateSecureToken(32)
	apiSecret := "as_" + utils.GenerateSecureToken(32)

	// Hash the credentials
	apiKeyHash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash API key: %v", err)
		utils.SendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	apiSecretHash, err := bcrypt.GenerateFromPassword([]byte(apiSecret), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash API secret: %v", err)
		utils.SendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Marshal JSON fields
	originsJSON, _ := json.Marshal(req.AllowedOrigins)
	urisJSON, _ := json.Marshal(req.RedirectURIs)
	permsJSON, _ := json.Marshal([]string{"auth:validate", "user:read"})

	// Get current user ID from token (for created_by)
	var createdBy *string
	if claims, ok := r.Context().Value("claims").(*models.Claims); ok {
		createdBy = &claims.UserID
	}

	// Insert application
	appID := uuid.New().String()
	_, err = db.DB.Exec(`
		INSERT INTO applications (
			id, name, display_name, description, scenario_type,
			api_key_hash, api_secret_hash, allowed_origins, redirect_uris,
			permissions, rate_limit, max_users, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, appID, req.Name, req.DisplayName, req.Description, req.ScenarioType,
		string(apiKeyHash), string(apiSecretHash), string(originsJSON),
		string(urisJSON), string(permsJSON), *req.RateLimit, req.MaxUsers, createdBy)

	if err != nil {
		log.Printf("Failed to insert application: %v", err)
		utils.SendError(w, "Failed to register application", http.StatusInternalServerError)
		return
	}

	// Return credentials
	credentials := AppCredentials{
		ApplicationID: appID,
		APIKey:        apiKey,
		APISecret:     apiSecret,
	}

	// Log the registration
	actorID := ""
	if createdBy != nil {
		actorID = *createdBy
	}
	logAuthEvent(actorID, "application.registered", auth.GetClientIP(r), r.Header.Get("User-Agent"), true,
		map[string]interface{}{
			"application_id":   appID,
			"application_name": req.Name,
		})

	utils.SendJSON(w, credentials, http.StatusCreated)
}

// GetApplicationHandler gets a specific application
func GetApplicationHandler(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")

	var app Application
	var originsJSON, urisJSON, permsJSON, description, maxUsers sql.NullString
	var lastAccessed sql.NullString

	err := db.DB.QueryRow(`
		SELECT 
			id, name, display_name, description, scenario_type,
			allowed_origins, redirect_uris, permissions, rate_limit,
			max_users, is_active, last_accessed, total_users,
			total_authentications, created_at, updated_at
		FROM applications
		WHERE id = $1
	`, appID).Scan(
		&app.ID, &app.Name, &app.DisplayName, &description, &app.ScenarioType,
		&originsJSON, &urisJSON, &permsJSON, &app.RateLimit,
		&maxUsers, &app.IsActive, &lastAccessed, &app.TotalUsers,
		&app.TotalAuths, &app.CreatedAt, &app.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.SendError(w, "Application not found", http.StatusNotFound)
		} else {
			log.Printf("Database error fetching application: %v", err)
			utils.SendError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Parse optional fields
	if description.Valid {
		app.Description = description.String
	}

	if maxUsers.Valid {
		if val, err := strconv.Atoi(maxUsers.String); err == nil {
			app.MaxUsers = &val
		}
	}

	if lastAccessed.Valid {
		parsed, err := time.Parse("2006-01-02 15:04:05", lastAccessed.String)
		if err == nil {
			app.LastAccessed = &parsed
		}
	}

	// Parse JSON arrays
	if originsJSON.Valid && originsJSON.String != "" {
		json.Unmarshal([]byte(originsJSON.String), &app.AllowedOrigins)
	}
	if urisJSON.Valid && urisJSON.String != "" {
		json.Unmarshal([]byte(urisJSON.String), &app.RedirectURIs)
	}
	if permsJSON.Valid && permsJSON.String != "" {
		json.Unmarshal([]byte(permsJSON.String), &app.Permissions)
	}

	utils.SendJSON(w, app, http.StatusOK)
}

// UpdateApplicationHandler updates an application
func UpdateApplicationHandler(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")

	var req RegisterAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Marshal JSON fields
	originsJSON, _ := json.Marshal(req.AllowedOrigins)
	urisJSON, _ := json.Marshal(req.RedirectURIs)

	// Update application
	_, err := db.DB.Exec(`
		UPDATE applications 
		SET display_name = $1, description = $2, scenario_type = $3,
			allowed_origins = $4, redirect_uris = $5, rate_limit = $6,
			max_users = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8
	`, req.DisplayName, req.Description, req.ScenarioType,
		string(originsJSON), string(urisJSON), req.RateLimit, req.MaxUsers, appID)

	if err != nil {
		log.Printf("Failed to update application: %v", err)
		utils.SendError(w, "Failed to update application", http.StatusInternalServerError)
		return
	}

	utils.SendJSON(w, map[string]interface{}{
		"success": true,
		"message": "Application updated successfully",
	}, http.StatusOK)
}

// DeleteApplicationHandler deactivates an application
func DeleteApplicationHandler(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")

	// Deactivate instead of delete to preserve audit trail
	_, err := db.DB.Exec(`
		UPDATE applications 
		SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, appID)

	if err != nil {
		log.Printf("Failed to deactivate application: %v", err)
		utils.SendError(w, "Failed to deactivate application", http.StatusInternalServerError)
		return
	}

	// Log the deactivation
	logAuthEvent("", "application.deactivated", "", r.Header.Get("User-Agent"), true,
		map[string]interface{}{
			"application_id": appID,
		})

	utils.SendJSON(w, map[string]interface{}{
		"success": true,
		"message": "Application deactivated successfully",
	}, http.StatusOK)
}

// GenerateIntegrationCodeHandler generates integration code for a scenario
func GenerateIntegrationCodeHandler(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	integrationType := r.URL.Query().Get("type") // 'api', 'ui', 'workflow'

	if integrationType == "" {
		integrationType = "api"
	}

	// Get application details
	var app Application
	err := db.DB.QueryRow(`
		SELECT name, display_name, scenario_type 
		FROM applications 
		WHERE id = $1 AND is_active = true
	`, appID).Scan(&app.Name, &app.DisplayName, &app.ScenarioType)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.SendError(w, "Application not found", http.StatusNotFound)
		} else {
			utils.SendError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Generate integration code based on type
	var code string
	switch integrationType {
	case "api":
		code = generateAPIIntegrationCode(app.Name)
	case "ui":
		code = generateUIIntegrationCode(app.Name)
	case "workflow":
		code = generateWorkflowIntegrationCode(app.Name)
	default:
		utils.SendError(w, "Invalid integration type", http.StatusBadRequest)
		return
	}

	utils.SendJSON(w, map[string]interface{}{
		"application_name": app.Name,
		"integration_type": integrationType,
		"code":             code,
	}, http.StatusOK)
}

// Helper functions for code generation
func generateAPIIntegrationCode(appName string) string {
	return `// Add to your Go API main.go

import (
    "github.com/vrooli/auth-client-go"
)

// Initialize auth client
authClient := auth.NewClient("http://localhost:15000", "YOUR_API_KEY_HERE")

// Add to your router
router.Use(authClient.ValidateToken)

// In protected routes, access user context:
func protectedHandler(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(string)
    userEmail := r.Context().Value("user_email").(string)
    userRoles := r.Context().Value("user_roles").([]string)
    
    // Your protected logic here
}`
}

func generateUIIntegrationCode(appName string) string {
	return `<!-- Add to your HTML head -->
<script src="http://localhost:35000/auth-client.js"></script>

<script>
// Initialize auth
const auth = new VrooliAuth('YOUR_API_KEY_HERE');

// Check if user is authenticated
async function requireAuth() {
    const user = await auth.getUser();
    if (!user) {
        // Redirect to login
        window.location.href = 'http://localhost:35000/login?redirect=' + 
            encodeURIComponent(window.location.href);
        return;
    }
    
    // User is authenticated
    console.log('User:', user);
}

// Call on page load
requireAuth();
</script>`
}

func generateWorkflowIntegrationCode(appName string) string {
	return `{
  "name": "Auth Validation",
  "nodes": [
    {
      "parameters": {
        "url": "http://localhost:15000/api/v1/auth/validate",
        "headers": {
          "Authorization": "Bearer {{$json['token']}}"
        }
      },
      "name": "Validate Token",
      "type": "n8n-nodes-base.httpRequest"
    }
  ]
}`
}
