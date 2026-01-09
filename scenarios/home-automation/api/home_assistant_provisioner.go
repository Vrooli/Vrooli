package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type homeAssistantAuthRegistry struct {
	Data struct {
		RefreshTokens []struct {
			TokenType  string   `json:"token_type"`
			Token      string   `json:"token"`
			UserID     string   `json:"user_id"`
			ClientID   *string  `json:"client_id"`
			ClientName *string  `json:"client_name"`
			ExpireAt   *float64 `json:"expire_at"`
		} `json:"refresh_tokens"`
	} `json:"data"`
}

func (app *App) handleProvisionHomeAssistant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		BaseURL      string `json:"base_url"`
		ClientName   string `json:"client_name"`
		LifespanDays int    `json:"lifespan_days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	cfg := app.getHomeAssistantConfig()
	baseURL := strings.TrimSpace(req.BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(cfg.BaseURL)
	}
	if baseURL == "" {
		baseURL = "http://localhost:8123"
	}

	clientName := strings.TrimSpace(req.ClientName)
	if clientName == "" {
		clientName = "Home Automation Intelligence"
	}

	lifespan := req.LifespanDays
	if lifespan <= 0 {
		lifespan = 365
	}
	if lifespan > 3650 {
		lifespan = 3650
	}

	token, err := autoProvisionHomeAssistantToken(ctx, baseURL, clientName, lifespan)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	working, _, diag, err := app.updateHomeAssistantConfigWithToken(ctx, baseURL, token, "long_lived", "", nil)
	if err != nil {
		log.Printf("Failed to persist Home Assistant config: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to persist configuration")
		return
	}

	if diag.Status != "healthy" {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to verify Home Assistant connection: %s", strings.TrimSpace(diag.Error)))
		return
	}

	response := buildHomeAssistantResponse(working, diag)
	response["auto_provisioned"] = true
	response["saved"] = true
	response["message"] = "Home Assistant connection established automatically"

	writeJSON(w, http.StatusOK, response)
}

func autoProvisionHomeAssistantToken(ctx context.Context, baseURL, clientName string, lifespanDays int) (string, error) {
	registry, err := loadHomeAssistantAuthRegistry(ctx)
	if err != nil {
		return "", err
	}

	refreshToken, err := selectHomeAssistantRefreshToken(registry)
	if err != nil {
		return "", err
	}

	accessToken, _, err := requestHomeAssistantAccessToken(ctx, baseURL, refreshToken)
	if err != nil {
		return "", err
	}

	longLived, err := createHomeAssistantLongLivedToken(ctx, baseURL, accessToken, clientName, lifespanDays)
	if err != nil {
		return "", err
	}

	return longLived, nil
}

func (app *App) updateHomeAssistantConfigWithToken(ctx context.Context, baseURL, token, tokenType, refreshToken string, expiresAt *time.Time) (HomeAssistantConfig, *HomeAssistantClient, HomeAssistantDiagnostics, error) {
	working := app.getHomeAssistantConfig()
	updateReq := homeAssistantConfigRequest{
		BaseURL: baseURL,
		Token:   &token,
	}

	if err := applyRequestToConfig(&working, updateReq); err != nil {
		return HomeAssistantConfig{}, nil, HomeAssistantDiagnostics{}, err
	}

	working.MockMode = false
	if strings.TrimSpace(tokenType) == "" {
		tokenType = "long_lived"
	}
	working.TokenType = tokenType
	working.RefreshToken = strings.TrimSpace(refreshToken)
	if expiresAt != nil {
		expires := expiresAt.UTC()
		working.AccessTokenExpiresAt = &expires
	} else {
		working.AccessTokenExpiresAt = nil
	}
	working.Token = strings.TrimSpace(token)
	working.TokenConfigured = strings.TrimSpace(working.Token) != "" || strings.TrimSpace(working.RefreshToken) != ""
	client := buildHomeAssistantClient(working)
	diag := evaluateHomeAssistant(ctx, working, client, true)

	now := time.Now().UTC()
	checkedAt := now
	if diag.CheckedAt != nil {
		checkedAt = diag.CheckedAt.UTC()
	}
	diag.CheckedAt = &checkedAt
	working.LastCheckedAt = &checkedAt
	working.LastStatus = diag.Status
	working.LastMessage = diag.Message
	working.LastError = diag.Error
	working.UpdatedAt = &now
	working.TokenConfigured = strings.TrimSpace(working.Token) != "" || strings.TrimSpace(working.RefreshToken) != ""

	if err := saveHomeAssistantConfig(ctx, app.DB, working); err != nil {
		return HomeAssistantConfig{}, client, diag, err
	}

	app.setHomeAssistantConfig(working)
	if app.DeviceController != nil {
		app.DeviceController.SetHomeAssistantClient(client)
	}

	diag.Target = strings.TrimSpace(working.BaseURL)
	diag.MockMode = working.MockMode

	return working, client, diag, nil
}

func loadHomeAssistantAuthRegistry(ctx context.Context) (homeAssistantAuthRegistry, error) {
	container := strings.TrimSpace(os.Getenv("HOME_ASSISTANT_CONTAINER_NAME"))
	if container == "" {
		container = "home-assistant"
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "docker", "exec", container, "cat", "/config/.storage/auth")
	output, err := cmd.Output()
	if err != nil {
		return homeAssistantAuthRegistry{}, fmt.Errorf("failed to read Home Assistant auth registry: %w", err)
	}

	var registry homeAssistantAuthRegistry
	if err := json.Unmarshal(output, &registry); err != nil {
		return homeAssistantAuthRegistry{}, fmt.Errorf("failed to parse Home Assistant auth registry: %w", err)
	}

	return registry, nil
}

func selectHomeAssistantRefreshToken(registry homeAssistantAuthRegistry) (string, error) {
	preferNormal := ""
	fallback := ""

	for _, token := range registry.Data.RefreshTokens {
		trimmed := strings.TrimSpace(token.Token)
		if trimmed == "" {
			continue
		}

		// Skip expired tokens when expiration is provided
		if token.ExpireAt != nil {
			expiry := time.Unix(int64(*token.ExpireAt), 0)
			if time.Now().After(expiry) {
				continue
			}
		}

		if token.TokenType == "normal" {
			if preferNormal == "" {
				preferNormal = trimmed
			}
		} else if fallback == "" {
			fallback = trimmed
		}
	}

	candidate := preferNormal
	if candidate == "" {
		candidate = fallback
	}

	if candidate == "" {
		return "", errors.New("no Home Assistant refresh tokens available; complete Home Assistant onboarding and try again")
	}

	return candidate, nil
}

func requestHomeAssistantAccessToken(ctx context.Context, baseURL, refreshToken string) (string, int, error) {
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmedBase == "" {
		trimmedBase = "http://localhost:8123"
	}

	form := url.Values{}
	form.Set("client_id", trimmedBase+"/")
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, trimmedBase+"/auth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create Home Assistant token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to request Home Assistant access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("Home Assistant token endpoint returned %s", resp.Status)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", 0, fmt.Errorf("failed to parse Home Assistant access token response: %w", err)
	}

	if strings.TrimSpace(payload.AccessToken) == "" {
		return "", 0, errors.New("Home Assistant access token response did not include a token")
	}

	return payload.AccessToken, payload.ExpiresIn, nil
}

func createHomeAssistantLongLivedToken(ctx context.Context, baseURL, accessToken, clientName string, lifespanDays int) (string, error) {
	wsURL, err := buildHomeAssistantWebSocketURL(baseURL)
	if err != nil {
		return "", err
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Home Assistant websocket: %w", err)
	}
	defer conn.Close()

	var hello map[string]interface{}
	if err := conn.ReadJSON(&hello); err != nil {
		return "", fmt.Errorf("failed to read Home Assistant websocket handshake: %w", err)
	}

	if helloType, _ := hello["type"].(string); helloType != "auth_required" {
		return "", fmt.Errorf("unexpected Home Assistant websocket handshake response: %v", hello)
	}

	if err := conn.WriteJSON(map[string]interface{}{
		"type":         "auth",
		"access_token": accessToken,
	}); err != nil {
		return "", fmt.Errorf("failed to authenticate with Home Assistant websocket: %w", err)
	}

	var authResp map[string]interface{}
	if err := conn.ReadJSON(&authResp); err != nil {
		return "", fmt.Errorf("failed to read Home Assistant websocket auth response: %w", err)
	}

	if authType, _ := authResp["type"].(string); authType != "auth_ok" {
		if msg, _ := authResp["message"].(string); msg != "" {
			return "", fmt.Errorf("Home Assistant websocket authentication failed: %s", msg)
		}
		return "", fmt.Errorf("Home Assistant websocket authentication failed: %v", authResp)
	}

	request := map[string]interface{}{
		"id":          time.Now().UnixNano(),
		"type":        "auth/long_lived_access_token",
		"client_name": clientName,
		"lifespan":    lifespanDays,
	}

	if err := conn.WriteJSON(request); err != nil {
		return "", fmt.Errorf("failed to request Home Assistant long-lived token: %w", err)
	}

	var result map[string]interface{}
	if err := conn.ReadJSON(&result); err != nil {
		return "", fmt.Errorf("failed to read Home Assistant long-lived token response: %w", err)
	}

	if success, _ := result["success"].(bool); !success {
		if errMsg, _ := result["error"].(map[string]interface{}); errMsg != nil {
			if message, _ := errMsg["message"].(string); message != "" {
				return "", fmt.Errorf("Home Assistant refused to create token: %s", message)
			}
		}
		return "", fmt.Errorf("Home Assistant refused to create token: %v", result)
	}

	token, _ := result["result"].(string)
	if strings.TrimSpace(token) == "" {
		return "", errors.New("Home Assistant did not return a long-lived token")
	}

	return token, nil
}

func buildHomeAssistantWebSocketURL(baseURL string) (string, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		trimmed = "http://localhost:8123"
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid Home Assistant base URL: %w", err)
	}

	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	default:
		return "", fmt.Errorf("unsupported Home Assistant scheme: %s", parsed.Scheme)
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/websocket"
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed.String(), nil
}

func (app *App) shouldAutoProvisionHomeAssistant(cfg HomeAssistantConfig, diag *HomeAssistantDiagnostics) (bool, string) {
	if cfg.MockMode {
		return false, ""
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return false, ""
	}

	if !cfg.TokenConfigured || strings.TrimSpace(cfg.Token) == "" {
		return true, "missing-token"
	}

	if diag != nil && strings.ToLower(strings.TrimSpace(diag.Status)) == "degraded" {
		errorLower := strings.ToLower(diag.Error)
		if strings.Contains(errorLower, "401") || strings.Contains(errorLower, "unauthorized") || strings.Contains(errorLower, "forbidden") {
			return true, "unauthorized"
		}

		messageLower := strings.ToLower(diag.Message)
		if strings.Contains(messageLower, "token") && strings.Contains(messageLower, "not configured") {
			return true, "missing-token"
		}
	}

	return false, ""
}

func (app *App) attemptHomeAssistantAutoProvision(ctx context.Context, cfg HomeAssistantConfig, reason string) (HomeAssistantDiagnostics, bool) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "http://localhost:8123"
	}

	app.homeAssistantProvisionLock.Lock()
	defer app.homeAssistantProvisionLock.Unlock()

	if !app.lastHomeAssistantProvisionAttempt.IsZero() && reason != "startup" {
		delta := time.Since(app.lastHomeAssistantProvisionAttempt)
		if delta < 30*time.Second {
			return HomeAssistantDiagnostics{}, false
		}
	}

	app.lastHomeAssistantProvisionAttempt = time.Now()

	token, err := autoProvisionHomeAssistantToken(ctx, baseURL, "Home Automation Intelligence", 3650)
	if err == nil {
		working, _, diag, errUpdate := app.updateHomeAssistantConfigWithToken(ctx, baseURL, token, "long_lived", "", nil)
		if errUpdate != nil {
			log.Printf("⚠️  Failed to persist auto-provisioned Home Assistant token (%s): %v", reason, errUpdate)
			return diag, false
		}

		if strings.ToLower(diag.Status) != "healthy" {
			log.Printf("⚠️  Auto-provision verification degraded (%s): status=%s error=%s", reason, diag.Status, strings.TrimSpace(diag.Error))
			return diag, true
		}

		log.Printf("✅ Home Assistant auto-provision succeeded (%s) for %s", reason, strings.TrimSpace(working.BaseURL))
		app.lastHomeAssistantProvisionAttempt = time.Time{}
		return diag, true
	}

	log.Printf("⚠️  Home Assistant long-lived provisioning failed (%s): %v", reason, err)

	registry, regErr := loadHomeAssistantAuthRegistry(ctx)
	if regErr != nil {
		log.Printf("⚠️  Unable to load Home Assistant auth registry for refresh-token fallback (%s): %v", reason, regErr)
		return HomeAssistantDiagnostics{}, false
	}

	refreshToken, err := selectHomeAssistantRefreshToken(registry)
	if err != nil {
		log.Printf("⚠️  No usable Home Assistant refresh token found (%s): %v", reason, err)
		return HomeAssistantDiagnostics{}, false
	}

	accessToken, expiresIn, err := requestHomeAssistantAccessToken(ctx, baseURL, refreshToken)
	if err != nil {
		log.Printf("⚠️  Failed to exchange Home Assistant refresh token (%s): %v", reason, err)
		return HomeAssistantDiagnostics{}, false
	}

	var expiresAt *time.Time
	if expiresIn > 0 {
		exp := time.Now().Add(time.Duration(expiresIn) * time.Second).UTC()
		expiresAt = &exp
	}

	working, _, diag, errUpdate := app.updateHomeAssistantConfigWithToken(ctx, baseURL, accessToken, "refresh", refreshToken, expiresAt)
	if errUpdate != nil {
		log.Printf("⚠️  Failed to persist Home Assistant refresh-token configuration (%s): %v", reason, errUpdate)
		return diag, false
	}

	if strings.ToLower(diag.Status) != "healthy" {
		log.Printf("⚠️  Home Assistant refresh-token verification degraded (%s): status=%s error=%s", reason, diag.Status, strings.TrimSpace(diag.Error))
		return diag, true
	}

	log.Printf("✅ Home Assistant refresh-token provisioning succeeded (%s) for %s", reason, strings.TrimSpace(working.BaseURL))
	app.lastHomeAssistantProvisionAttempt = time.Time{}
	return diag, true
}
