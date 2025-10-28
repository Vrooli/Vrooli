package main

import "testing"

func TestBuildHomeAssistantWebSocketURL(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"http", "http://localhost:8123", "ws://localhost:8123/api/websocket", false},
		{"http with path", "http://localhost:8123/root", "ws://localhost:8123/root/api/websocket", false},
		{"https", "https://example.com", "wss://example.com/api/websocket", false},
		{"invalid", "ftp://example.com", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildHomeAssistantWebSocketURL(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}

func TestSelectHomeAssistantRefreshToken(t *testing.T) {
	registry := homeAssistantAuthRegistry{}
	registry.Data.RefreshTokens = append(registry.Data.RefreshTokens,
		struct {
			TokenType  string   `json:"token_type"`
			Token      string   `json:"token"`
			UserID     string   `json:"user_id"`
			ClientID   *string  `json:"client_id"`
			ClientName *string  `json:"client_name"`
			ExpireAt   *float64 `json:"expire_at"`
		}{TokenType: "system", Token: "system-token"},
		struct {
			TokenType  string   `json:"token_type"`
			Token      string   `json:"token"`
			UserID     string   `json:"user_id"`
			ClientID   *string  `json:"client_id"`
			ClientName *string  `json:"client_name"`
			ExpireAt   *float64 `json:"expire_at"`
		}{TokenType: "normal", Token: "normal-token"},
	)

	token, err := selectHomeAssistantRefreshToken(registry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "normal-token" {
		t.Fatalf("expected normal-token, got %s", token)
	}

	registry.Data.RefreshTokens = registry.Data.RefreshTokens[:1]
	token, err = selectHomeAssistantRefreshToken(registry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "system-token" {
		t.Fatalf("expected fallback system-token, got %s", token)
	}

	registry.Data.RefreshTokens = nil
	if _, err := selectHomeAssistantRefreshToken(registry); err == nil {
		t.Fatalf("expected error when no tokens available")
	}
}
