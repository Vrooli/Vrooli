package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// OAuthConfig holds OAuth provider configurations
type OAuthConfig struct {
	Google *oauth2.Config
	GitHub *oauth2.Config
}

// OAuthUser represents user data from OAuth providers
type OAuthUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Provider      string `json:"provider"`
	EmailVerified bool   `json:"email_verified"`
}

// InitOAuthConfig initializes OAuth configurations
func InitOAuthConfig() *OAuthConfig {
	config := &OAuthConfig{}

	// Google OAuth2 configuration
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if googleClientID != "" && googleClientSecret != "" {
		config.Google = &oauth2.Config{
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
			RedirectURL:  getRedirectURL("google"),
			Scopes:       []string{"email", "profile"},
			Endpoint:     google.Endpoint,
		}
	}

	// GitHub OAuth2 configuration
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	if githubClientID != "" && githubClientSecret != "" {
		config.GitHub = &oauth2.Config{
			ClientID:     githubClientID,
			ClientSecret: githubClientSecret,
			RedirectURL:  getRedirectURL("github"),
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		}
	}

	return config
}

// GetAuthURL generates the OAuth authorization URL
func GetAuthURL(config *oauth2.Config, state string) string {
	return config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// ExchangeCode exchanges authorization code for token
func ExchangeCode(config *oauth2.Config, code string) (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return config.Exchange(ctx, code)
}

// FetchGoogleUser fetches user info from Google
func FetchGoogleUser(token *oauth2.Token) (*OAuthUser, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Google user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var googleUser struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		VerifiedEmail bool   `json:"verified_email"`
	}

	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, fmt.Errorf("failed to parse Google user info: %w", err)
	}

	return &OAuthUser{
		ID:            googleUser.ID,
		Email:         googleUser.Email,
		Name:          googleUser.Name,
		Picture:       googleUser.Picture,
		Provider:      "google",
		EmailVerified: googleUser.VerifiedEmail,
	}, nil
}

// FetchGitHubUser fetches user info from GitHub
func FetchGitHubUser(token *oauth2.Token) (*OAuthUser, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Fetch user info
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var githubUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.Unmarshal(body, &githubUser); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub user info: %w", err)
	}

	// If email is not public, fetch from emails endpoint
	if githubUser.Email == "" {
		githubUser.Email, _ = fetchGitHubPrimaryEmail(token.AccessToken, client)
	}

	return &OAuthUser{
		ID:            fmt.Sprintf("%d", githubUser.ID),
		Email:         githubUser.Email,
		Name:          githubUser.Name,
		Picture:       githubUser.AvatarURL,
		Provider:      "github",
		EmailVerified: true, // GitHub emails are verified
	}, nil
}

// fetchGitHubPrimaryEmail fetches the primary email from GitHub
func fetchGitHubPrimaryEmail(accessToken string, client *http.Client) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	return "", fmt.Errorf("no primary email found")
}

// GenerateStateToken generates a secure state token for OAuth
func GenerateStateToken() (string, error) {
	return GenerateSecureToken(32)
}

// ValidateState validates the OAuth state parameter
func ValidateState(state string) bool {
	// In production, store state tokens in Redis with expiration
	// For now, just check if it's not empty and has reasonable length
	return state != "" && len(state) >= 32
}

// getRedirectURL returns the OAuth redirect URL for the provider
func getRedirectURL(provider string) string {
	baseURL := os.Getenv("AUTH_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8105"
	}
	return fmt.Sprintf("%s/api/v1/auth/oauth/%s/callback", baseURL, provider)
}
