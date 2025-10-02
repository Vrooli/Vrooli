package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/models"
	"scenario-authenticator/utils"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var oauthConfig *auth.OAuthConfig

// InitOAuth initializes OAuth configuration
func InitOAuth() {
	oauthConfig = auth.InitOAuthConfig()

	if oauthConfig.Google != nil {
		log.Println("[scenario-authenticator/oauth] ✅ Google OAuth configured")
	} else {
		log.Println("[scenario-authenticator/oauth] ⚠️  Google OAuth not configured (missing GOOGLE_CLIENT_ID or GOOGLE_CLIENT_SECRET)")
	}

	if oauthConfig.GitHub != nil {
		log.Println("[scenario-authenticator/oauth] ✅ GitHub OAuth configured")
	} else {
		log.Println("[scenario-authenticator/oauth] ⚠️  GitHub OAuth not configured (missing GITHUB_CLIENT_ID or GITHUB_CLIENT_SECRET)")
	}
}

// OAuthLoginHandler initiates OAuth login
func OAuthLoginHandler(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		utils.SendError(w, "Provider is required", http.StatusBadRequest)
		return
	}

	var config *oauth2.Config
	switch provider {
	case "google":
		if oauthConfig.Google == nil {
			utils.SendError(w, "Google OAuth not configured", http.StatusBadRequest)
			return
		}
		config = oauthConfig.Google
	case "github":
		if oauthConfig.GitHub == nil {
			utils.SendError(w, "GitHub OAuth not configured", http.StatusBadRequest)
			return
		}
		config = oauthConfig.GitHub
	default:
		utils.SendError(w, "Invalid provider", http.StatusBadRequest)
		return
	}

	// Generate state token
	state, err := auth.GenerateStateToken()
	if err != nil {
		utils.SendError(w, "Failed to generate state token", http.StatusInternalServerError)
		return
	}

	// Store state in Redis with expiration (5 minutes)
	auth.StoreOAuthState(state, provider, 5*time.Minute)

	// Get authorization URL
	authURL := auth.GetAuthURL(config, state)

	// Return the authorization URL
	response := map[string]string{
		"auth_url": authURL,
		"provider": provider,
	}
	utils.SendJSON(w, response, http.StatusOK)
}

// OAuthCallbackHandler handles OAuth callback
func OAuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Path[len("/api/v1/auth/oauth/"):]
	provider = provider[:len(provider)-len("/callback")]

	// Validate state
	state := r.URL.Query().Get("state")
	if !auth.ValidateOAuthState(state, provider) {
		utils.SendError(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		utils.SendError(w, "Authorization code not provided", http.StatusBadRequest)
		return
	}

	var config *oauth2.Config
	switch provider {
	case "google":
		config = oauthConfig.Google
	case "github":
		config = oauthConfig.GitHub
	default:
		utils.SendError(w, "Invalid provider", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := auth.ExchangeCode(config, code)
	if err != nil {
		log.Printf("Failed to exchange OAuth code: %v", err)
		utils.SendError(w, "Failed to exchange authorization code", http.StatusInternalServerError)
		return
	}

	// Fetch user info
	var oauthUser *auth.OAuthUser
	switch provider {
	case "google":
		oauthUser, err = auth.FetchGoogleUser(token)
	case "github":
		oauthUser, err = auth.FetchGitHubUser(token)
	}

	if err != nil {
		log.Printf("Failed to fetch OAuth user info: %v", err)
		utils.SendError(w, "Failed to fetch user information", http.StatusInternalServerError)
		return
	}

	// Check if user exists or create new one
	user, err := findOrCreateOAuthUser(oauthUser)
	if err != nil {
		log.Printf("Failed to process OAuth user: %v", err)
		utils.SendError(w, "Failed to process user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	jwtToken, err := auth.GenerateToken(user)
	if err != nil {
		log.Printf("Failed to generate JWT token: %v", err)
		utils.SendError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Generate refresh token
	refreshToken := auth.GenerateRefreshToken()
	auth.StoreRefreshToken(user.ID, refreshToken)

	// Store session
	if err := auth.StoreSession(user.ID, jwtToken, refreshToken, r); err != nil {
		log.Printf("Failed to store session: %v", err)
	}

	// Log auth event
	logAuthEvent(user.ID, "user.oauth.login", auth.GetClientIP(r), r.Header.Get("User-Agent"), true,
		map[string]interface{}{"provider": provider})

	// Send response
	response := models.AuthResponse{
		Success:      true,
		Token:        jwtToken,
		RefreshToken: refreshToken,
		User:         user,
	}

	utils.SendJSON(w, response, http.StatusOK)
}

// GetOAuthProvidersHandler returns available OAuth providers
func GetOAuthProvidersHandler(w http.ResponseWriter, r *http.Request) {
	providers := []map[string]interface{}{}

	if oauthConfig.Google != nil {
		providers = append(providers, map[string]interface{}{
			"name":    "google",
			"display": "Google",
			"icon":    "google",
			"enabled": true,
		})
	}

	if oauthConfig.GitHub != nil {
		providers = append(providers, map[string]interface{}{
			"name":    "github",
			"display": "GitHub",
			"icon":    "github",
			"enabled": true,
		})
	}

	response := map[string]interface{}{
		"providers": providers,
	}

	utils.SendJSON(w, response, http.StatusOK)
}

// findOrCreateOAuthUser finds existing user or creates new one from OAuth data
func findOrCreateOAuthUser(oauthUser *auth.OAuthUser) (*models.User, error) {
	var user models.User
	var rolesJSON string
	var usernameNullable sql.NullString
	var oauthData sql.NullString

	// Try to find existing user by email or OAuth provider ID
	err := db.DB.QueryRow(`
		SELECT id, email, username, roles, email_verified, created_at, oauth_providers
		FROM users
		WHERE (email = $1 OR oauth_providers->$2->>'id' = $3) AND deleted_at IS NULL`,
		oauthUser.Email, oauthUser.Provider, oauthUser.ID,
	).Scan(&user.ID, &user.Email, &usernameNullable, &rolesJSON,
		&user.EmailVerified, &user.CreatedAt, &oauthData)

	if err == sql.ErrNoRows {
		// Create new user
		userID := uuid.New().String()

		// Create OAuth providers data
		oauthProviders := map[string]interface{}{
			oauthUser.Provider: map[string]interface{}{
				"id":      oauthUser.ID,
				"name":    oauthUser.Name,
				"picture": oauthUser.Picture,
			},
		}
		oauthJSON, _ := json.Marshal(oauthProviders)

		// Default roles
		roles := []string{"user"}
		rolesJSONData, _ := json.Marshal(roles)

		// Insert new user
		err = db.DB.QueryRow(`
			INSERT INTO users (id, email, username, password_hash, roles, email_verified, oauth_providers, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, email, username, roles, email_verified, created_at`,
			userID, oauthUser.Email, oauthUser.Name,
			"oauth_user_no_password", // OAuth users don't have passwords
			string(rolesJSONData), oauthUser.EmailVerified, string(oauthJSON), time.Now(),
		).Scan(&user.ID, &user.Email, &usernameNullable, &rolesJSON,
			&user.EmailVerified, &user.CreatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to create OAuth user: %w", err)
		}

		logAuthEvent(userID, "user.oauth.registered", "", "", true,
			map[string]interface{}{"provider": oauthUser.Provider})
	} else if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	} else {
		// Update existing user's OAuth data if needed
		var existingOAuth map[string]interface{}
		if oauthData.Valid {
			json.Unmarshal([]byte(oauthData.String), &existingOAuth)
		} else {
			existingOAuth = make(map[string]interface{})
		}

		// Add or update this provider's data
		existingOAuth[oauthUser.Provider] = map[string]interface{}{
			"id":      oauthUser.ID,
			"name":    oauthUser.Name,
			"picture": oauthUser.Picture,
		}
		updatedOAuthJSON, _ := json.Marshal(existingOAuth)

		// Update user's OAuth providers
		_, err = db.DB.Exec(`
			UPDATE users
			SET oauth_providers = $1, last_login = $2
			WHERE id = $3`,
			string(updatedOAuthJSON), time.Now(), user.ID,
		)
		if err != nil {
			log.Printf("Failed to update OAuth providers: %v", err)
		}
	}

	// Handle nullable username
	if usernameNullable.Valid {
		user.Username = usernameNullable.String
	}

	// Parse roles
	if err := json.Unmarshal([]byte(rolesJSON), &user.Roles); err != nil {
		user.Roles = []string{"user"}
	}

	return &user, nil
}
