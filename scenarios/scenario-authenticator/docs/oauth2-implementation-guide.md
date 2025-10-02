# OAuth2 Implementation Guide

This document provides a comprehensive roadmap for implementing OAuth2 provider support (Google, GitHub, Microsoft) in scenario-authenticator. This is a P1 requirement that will enable enterprise adoption and social login capabilities.

## Current State

✅ **Already Implemented:**
- Database schema includes `oauth_providers` JSONB field in users table
- OAuth endpoints registered in API (lines 105-110 in `api/main.go`):
  - `/api/v1/auth/oauth/providers` - List available providers
  - `/api/v1/auth/oauth/login` - Initiate OAuth flow
  - `/api/v1/auth/oauth/google/callback` - Google callback handler
  - `/api/v1/auth/oauth/github/callback` - GitHub callback handler
- OAuth configuration initialized in `handlers.InitOAuth()` function
- `golang.org/x/oauth2` library already in dependencies (go.mod line 17)

❌ **Not Yet Implemented:**
- Provider-specific OAuth configurations
- OAuth state management (CSRF protection)
- Token exchange logic
- User linking/creation from OAuth profiles
- UI components for OAuth login buttons

## Architecture Overview

```
┌──────────────┐          ┌──────────────┐          ┌──────────────┐
│   User's     │          │   Vrooli     │          │   OAuth      │
│   Browser    │          │   Scenario   │          │   Provider   │
│              │          │   Auth       │          │  (Google)    │
└──────────────┘          └──────────────┘          └──────────────┘
       │                         │                          │
       │  1. Click "Login"       │                          │
       ├────────────────────────>│                          │
       │                         │                          │
       │  2. Redirect to OAuth   │                          │
       │<────────────────────────┤                          │
       │                         │                          │
       │  3. Authenticate        │                          │
       ├─────────────────────────┼─────────────────────────>│
       │                         │                          │
       │  4. Callback with code  │                          │
       │<────────────────────────┼──────────────────────────┤
       │                         │                          │
       │  5. Exchange code       │                          │
       │                         ├─────────────────────────>│
       │                         │                          │
       │  6. Return access token │                          │
       │                         │<─────────────────────────┤
       │                         │                          │
       │  7. Get user profile    │                          │
       │                         ├─────────────────────────>│
       │                         │                          │
       │  8. Return profile data │                          │
       │                         │<─────────────────────────┤
       │                         │                          │
       │  9. Create/link user    │                          │
       │         & return JWT    │                          │
       │<────────────────────────┤                          │
       │                         │                          │
```

## Implementation Steps

### Phase 1: Provider Configuration (2-3 hours)

#### 1.1: Environment Variables

Add to `.vrooli/service.json` resources:

```json
{
  "lifecycle": {
    "develop": {
      "env": {
        "GOOGLE_CLIENT_ID": "${GOOGLE_CLIENT_ID}",
        "GOOGLE_CLIENT_SECRET": "${GOOGLE_CLIENT_SECRET}",
        "GITHUB_CLIENT_ID": "${GITHUB_CLIENT_ID}",
        "GITHUB_CLIENT_SECRET": "${GITHUB_CLIENT_SECRET}",
        "OAUTH_REDIRECT_BASE_URL": "http://localhost:${API_PORT}"
      }
    }
  }
}
```

#### 1.2: OAuth Configuration File

Create `api/auth/oauth_config.go`:

```go
package auth

import (
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

var (
	GoogleOAuthConfig *oauth2.Config
	GitHubOAuthConfig *oauth2.Config
)

func InitOAuthProviders() error {
	baseURL := os.Getenv("OAUTH_REDIRECT_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:15785"
	}

	// Google OAuth2 Configuration
	GoogleOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  fmt.Sprintf("%s/api/v1/auth/oauth/google/callback", baseURL),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	// GitHub OAuth2 Configuration
	GitHubOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectURL:  fmt.Sprintf("%s/api/v1/auth/oauth/github/callback", baseURL),
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}

	return nil
}
```

### Phase 2: State Management (1-2 hours)

Create `api/auth/oauth_state.go`:

```go
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

type OAuthState struct {
	State      string
	CreatedAt  time.Time
	Provider   string
	RedirectTo string // Optional: where to redirect after successful auth
}

var (
	// In-memory state store (use Redis in production)
	stateStore = make(map[string]*OAuthState)
	stateMutex sync.RWMutex
)

// GenerateOAuthState creates a secure random state token
func GenerateOAuthState(provider, redirectTo string) (string, error) {
	// Generate 32 random bytes
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	state := base64.URLEncoding.EncodeToString(b)

	// Store state
	stateMutex.Lock()
	defer stateMutex.Unlock()

	stateStore[state] = &OAuthState{
		State:      state,
		CreatedAt:  time.Now(),
		Provider:   provider,
		RedirectTo: redirectTo,
	}

	// Clean up old states (older than 10 minutes)
	go cleanupOldStates()

	return state, nil
}

// ValidateOAuthState checks if a state token is valid
func ValidateOAuthState(state string) (*OAuthState, error) {
	stateMutex.RLock()
	defer stateMutex.RUnlock()

	oauthState, exists := stateStore[state]
	if !exists {
		return nil, fmt.Errorf("invalid state token")
	}

	// Check if state is expired (10 minutes)
	if time.Since(oauthState.CreatedAt) > 10*time.Minute {
		return nil, fmt.Errorf("state token expired")
	}

	return oauthState, nil
}

// ConsumeOAuthState validates and removes a state token (one-time use)
func ConsumeOAuthState(state string) (*OAuthState, error) {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	oauthState, exists := stateStore[state]
	if !exists {
		return nil, fmt.Errorf("invalid state token")
	}

	if time.Since(oauthState.CreatedAt) > 10*time.Minute {
		delete(stateStore, state)
		return nil, fmt.Errorf("state token expired")
	}

	// Remove state after use (prevent replay attacks)
	delete(stateStore, state)

	return oauthState, nil
}

// cleanupOldStates removes expired states from memory
func cleanupOldStates() {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	now := time.Now()
	for state, oauthState := range stateStore {
		if now.Sub(oauthState.CreatedAt) > 10*time.Minute {
			delete(stateStore, state)
		}
	}
}
```

### Phase 3: OAuth Handlers (3-4 hours)

Update `api/handlers/oauth.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/models"
	"github.com/google/uuid"
)

// OAuthProvider represents an OAuth provider configuration
type OAuthProvider struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	LoginURL    string `json:"login_url"`
	Enabled     bool   `json:"enabled"`
}

// InitOAuth initializes OAuth providers
func InitOAuth() {
	auth.InitOAuthProviders()
}

// GetOAuthProvidersHandler returns list of available OAuth providers
func GetOAuthProvidersHandler(w http.ResponseWriter, r *http.Request) {
	providers := []OAuthProvider{
		{
			Name:        "google",
			DisplayName: "Google",
			LoginURL:    "/api/v1/auth/oauth/login?provider=google",
			Enabled:     auth.GoogleOAuthConfig.ClientID != "",
		},
		{
			Name:        "github",
			DisplayName: "GitHub",
			LoginURL:    "/api/v1/auth/oauth/login?provider=github",
			Enabled:     auth.GitHubOAuthConfig.ClientID != "",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

// OAuthLoginHandler initiates OAuth flow
func OAuthLoginHandler(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	redirectTo := r.URL.Query().Get("redirect_to")

	// Generate state token for CSRF protection
	state, err := auth.GenerateOAuthState(provider, redirectTo)
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	var authURL string
	switch provider {
	case "google":
		authURL = auth.GoogleOAuthConfig.AuthCodeURL(state)
	case "github":
		authURL = auth.GitHubOAuthConfig.AuthCodeURL(state)
	default:
		http.Error(w, "Unsupported provider", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// OAuthCallbackHandler handles OAuth callbacks
func OAuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Get state and code from query parameters
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	// Validate state (CSRF protection)
	oauthState, err := auth.ConsumeOAuthState(state)
	if err != nil {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	var token string
	var userInfo map[string]interface{}

	switch oauthState.Provider {
	case "google":
		token, userInfo, err = exchangeGoogleCode(code)
	case "github":
		token, userInfo, err = exchangeGitHubCode(code)
	default:
		http.Error(w, "Unsupported provider", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "Failed to exchange code: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create or link user account
	user, isNew, err := createOrLinkOAuthUser(oauthState.Provider, userInfo)
	if err != nil {
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	jwtToken, refreshToken, err := auth.GenerateTokenPair(user.ID, user.Email, user.Roles)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Return response
	response := models.AuthResponse{
		Success:      true,
		Token:        jwtToken,
		RefreshToken: refreshToken,
		User:         user,
		Message:      fmt.Sprintf("OAuth login successful (new: %v)", isNew),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// exchangeGoogleCode exchanges Google authorization code for user info
func exchangeGoogleCode(code string) (string, map[string]interface{}, error) {
	ctx := context.Background()

	// Exchange code for token
	token, err := auth.GoogleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		return "", nil, err
	}

	// Get user info
	client := auth.GoogleOAuthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(data, &userInfo); err != nil {
		return "", nil, err
	}

	return token.AccessToken, userInfo, nil
}

// exchangeGitHubCode exchanges GitHub authorization code for user info
func exchangeGitHubCode(code string) (string, map[string]interface{}, error) {
	ctx := context.Background()

	// Exchange code for token
	token, err := auth.GitHubOAuthConfig.Exchange(ctx, code)
	if err != nil {
		return "", nil, err
	}

	// Get user info
	client := auth.GitHubOAuthConfig.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(data, &userInfo); err != nil {
		return "", nil, err
	}

	// GitHub may not provide email in user endpoint, fetch separately
	if _, ok := userInfo["email"]; !ok {
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err == nil {
			defer emailResp.Body.Close()
			emailData, _ := ioutil.ReadAll(emailResp.Body)
			var emails []map[string]interface{}
			if err := json.Unmarshal(emailData, &emails); err == nil && len(emails) > 0 {
				// Find primary email
				for _, email := range emails {
					if primary, ok := email["primary"].(bool); ok && primary {
						userInfo["email"] = email["email"]
						break
					}
				}
			}
		}
	}

	return token.AccessToken, userInfo, nil
}

// createOrLinkOAuthUser creates a new user or links existing user with OAuth provider
func createOrLinkOAuthUser(provider string, userInfo map[string]interface{}) (*models.User, bool, error) {
	email, _ := userInfo["email"].(string)
	if email == "" {
		return nil, false, fmt.Errorf("email not provided by OAuth provider")
	}

	// Check if user already exists
	var existingUserID string
	err := db.DB.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&existingUserID)

	if err == nil {
		// User exists, link OAuth provider
		oauthData := map[string]interface{}{
			provider: userInfo,
		}
		oauthJSON, _ := json.Marshal(oauthData)

		_, err = db.DB.Exec(
			"UPDATE users SET oauth_providers = oauth_providers || $1::jsonb WHERE id = $2",
			string(oauthJSON),
			existingUserID,
		)
		if err != nil {
			return nil, false, err
		}

		// Return existing user
		var user models.User
		err = db.DB.QueryRow(
			"SELECT id, email, username, roles FROM users WHERE id = $1",
			existingUserID,
		).Scan(&user.ID, &user.Email, &user.Username, &user.Roles)

		return &user, false, err
	}

	// User doesn't exist, create new account
	userID := uuid.New().String()
	username := extractUsername(userInfo)

	oauthData := map[string]interface{}{
		provider: userInfo,
	}
	oauthJSON, _ := json.Marshal(oauthData)

	_, err = db.DB.Exec(`
		INSERT INTO users (id, email, username, password_hash, oauth_providers, email_verified)
		VALUES ($1, $2, $3, '', $4::jsonb, TRUE)
	`, userID, email, username, string(oauthJSON))

	if err != nil {
		return nil, false, err
	}

	user := &models.User{
		ID:            userID,
		Email:         email,
		Username:      username,
		Roles:         []string{"user"},
		EmailVerified: true,
	}

	return user, true, nil
}

// extractUsername extracts username from OAuth user info
func extractUsername(userInfo map[string]interface{}) string {
	// Try different username fields
	if username, ok := userInfo["login"].(string); ok {
		return username
	}
	if name, ok := userInfo["name"].(string); ok {
		return name
	}
	if email, ok := userInfo["email"].(string); ok {
		return email
	}
	return ""
}
```

### Phase 4: UI Integration (2-3 hours)

Update `ui/login.html` to add OAuth buttons:

```html
<div class="oauth-providers">
    <h3>Or sign in with</h3>
    <div id="oauth-buttons"></div>
</div>

<script>
// Fetch available OAuth providers
async function loadOAuthProviders() {
    const response = await fetch(`${AUTH_URL}/api/v1/auth/oauth/providers`);
    const providers = await response.json();

    const buttonsContainer = document.getElementById('oauth-buttons');
    buttonsContainer.innerHTML = '';

    providers.filter(p => p.enabled).forEach(provider => {
        const button = document.createElement('button');
        button.className = `oauth-button oauth-${provider.name}`;
        button.textContent = `Sign in with ${provider.display_name}`;
        button.onclick = () => window.location.href = `${AUTH_URL}${provider.login_url}`;
        buttonsContainer.appendChild(button);
    });
}

loadOAuthProviders();
</script>

<style>
.oauth-providers {
    margin-top: 20px;
    padding-top: 20px;
    border-top: 1px solid #ddd;
}

.oauth-button {
    width: 100%;
    padding: 12px;
    margin: 8px 0;
    border: 1px solid #ddd;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
}

.oauth-google {
    background: #4285f4;
    color: white;
}

.oauth-github {
    background: #24292e;
    color: white;
}
</style>
```

### Phase 5: Testing (2-3 hours)

Create `test/test-oauth.sh`:

```bash
#!/bin/bash
# OAuth Integration Test

# NOTE: This test requires real OAuth credentials
# Set these in your environment before running:
# export GOOGLE_CLIENT_ID="..."
# export GOOGLE_CLIENT_SECRET="..."

set -e

echo "OAuth Integration Test"
echo "======================"

# 1. Test provider listing
echo "1. Testing OAuth provider listing..."
PROVIDERS=$(curl -s http://localhost:15785/api/v1/auth/oauth/providers)
echo "$PROVIDERS" | jq .

# 2. Test OAuth login flow (manual)
echo ""
echo "2. Manual OAuth Test:"
echo "   Visit: http://localhost:15785/api/v1/auth/oauth/login?provider=google"
echo "   After successful login, you should receive a JWT token"

echo ""
echo "OAuth tests completed!"
```

## Security Considerations

### 1. State Token Management
- **CSRF Protection**: State tokens prevent cross-site request forgery
- **One-Time Use**: State tokens consumed after single use
- **Expiration**: 10-minute expiration for state tokens
- **Production**: Move state storage to Redis for distributed systems

### 2. Token Storage
- **Never Log**: Never log OAuth access tokens or refresh tokens
- **Secure Storage**: Store OAuth provider data in encrypted JSONB field
- **Token Rotation**: Implement refresh token rotation

### 3. User Linking
- **Email Verification**: OAuth providers mark email as verified
- **Prevent Account Takeover**: Validate email ownership before linking
- **Multiple Providers**: Allow linking multiple OAuth providers to one account

### 4. Redirect URL Validation
- **Whitelist**: Only allow redirects to known domains
- **Production**: Use HTTPS-only redirect URLs
- **Validation**: Validate redirect_to parameter

## Testing Strategy

### Unit Tests
```go
func TestGenerateOAuthState(t *testing.T) {
	state, err := auth.GenerateOAuthState("google", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, state)
}

func TestValidateOAuthState(t *testing.T) {
	state, _ := auth.GenerateOAuthState("google", "/dashboard")
	oauthState, err := auth.ValidateOAuthState(state)
	assert.NoError(t, err)
	assert.Equal(t, "google", oauthState.Provider)
}
```

### Integration Tests
1. Provider configuration validation
2. State generation and validation
3. Mock OAuth callback handling
4. User creation from OAuth profile
5. User linking with existing accounts

## Deployment Checklist

- [ ] Obtain OAuth credentials from Google Cloud Console
- [ ] Obtain OAuth credentials from GitHub Developer Settings
- [ ] Configure redirect URLs in provider consoles
- [ ] Set environment variables in production
- [ ] Update CORS settings to allow OAuth callbacks
- [ ] Test OAuth flow in staging environment
- [ ] Document OAuth setup for other developers
- [ ] Add OAuth login buttons to UI
- [ ] Implement error handling for OAuth failures
- [ ] Add logging for OAuth events

## Provider Setup Guides

### Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create new project or select existing
3. Enable Google+ API
4. Create OAuth 2.0 credentials
5. Add authorized redirect URI: `http://localhost:15785/api/v1/auth/oauth/google/callback`
6. Copy Client ID and Client Secret

### GitHub OAuth Setup

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click "New OAuth App"
3. Set callback URL: `http://localhost:15785/api/v1/auth/oauth/github/callback`
4. Copy Client ID and generate Client Secret

## Estimated Effort

- **Phase 1** (Configuration): 2-3 hours
- **Phase 2** (State Management): 1-2 hours
- **Phase 3** (OAuth Handlers): 3-4 hours
- **Phase 4** (UI Integration): 2-3 hours
- **Phase 5** (Testing): 2-3 hours
- **Documentation & Deployment**: 1-2 hours

**Total**: 11-17 hours (1.5-2 days)

## Future Enhancements

- Microsoft/Azure AD OAuth support
- LinkedIn OAuth support
- Apple Sign-In support
- OAuth token refresh
- Multi-provider account merging
- Admin UI for OAuth provider management

## Resources

- [OAuth 2.0 RFC 6749](https://tools.ietf.org/html/rfc6749)
- [Google OAuth2 Documentation](https://developers.google.com/identity/protocols/oauth2)
- [GitHub OAuth Documentation](https://docs.github.com/en/developers/apps/building-oauth-apps)
- [golang.org/x/oauth2 Package](https://pkg.go.dev/golang.org/x/oauth2)

---

**Last Updated**: 2025-10-01
**Status**: Implementation Guide
**Priority**: P1 (Should Have)
