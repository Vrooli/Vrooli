package config

import (
	"net/http"
	"testing"
)

func TestCFAccessConstructorsShareConfiguration(t *testing.T) {
	customDoer := &http.Client{}
	configs := []struct {
		name          string
		cfg           CFConfig
		doer          httpDoer
		wantDoer      httpDoer
		wantBaseURL   string
		wantAccountID string
		wantAPIToken  string
	}{
		{
			name:          "defaults",
			cfg:           CFConfig{APIToken: "token", AccountID: "account"},
			wantDoer:      http.DefaultClient,
			wantBaseURL:   "https://api.cloudflare.com/client/v4",
			wantAccountID: "account",
			wantAPIToken:  "token",
		},
		{
			name: "configured overrides",
			cfg: CFConfig{
				APIToken:  "custom-token",
				AccountID: "custom-account",
				BaseURL:   "https://cf.example.invalid/client/v4",
			},
			doer:          customDoer,
			wantDoer:      customDoer,
			wantBaseURL:   "https://cf.example.invalid/client/v4",
			wantAccountID: "custom-account",
			wantAPIToken:  "custom-token",
		},
	}

	for _, tt := range configs {
		t.Run(tt.name, func(t *testing.T) {
			clients := []struct {
				name  string
				value any
			}{
				{name: "AccessClient", value: NewCFAccessClient(tt.doer, tt.cfg)},
				{name: "AccessMetadataReader", value: NewCFAccessMetadataReader(tt.doer, tt.cfg)},
			}
			for _, entry := range clients {
				state, ok := entry.value.(*cfAccessClient)
				if !ok || state == nil {
					t.Fatalf("%s = %#v, want non-nil *cfAccessClient", entry.name, entry.value)
				}
				if state.doer != tt.wantDoer || state.baseURL != tt.wantBaseURL {
					t.Errorf("%s transport/base = (%T, %q), want (%T, %q)", entry.name, state.doer, state.baseURL, tt.wantDoer, tt.wantBaseURL)
				}
				if state.accountID != tt.wantAccountID || state.apiToken != tt.wantAPIToken {
					t.Errorf("%s credentials = (%q, %q), want (%q, %q)", entry.name, state.apiToken, state.accountID, tt.wantAPIToken, tt.wantAccountID)
				}
			}
		})
	}
}

func TestCFAccessConstructorsReturnNilWithoutEitherCredential(t *testing.T) {
	for _, cfg := range []CFConfig{{APIToken: "token"}, {AccountID: "account"}} {
		if client := NewCFAccessClient(nil, cfg); client != nil {
			t.Errorf("NewCFAccessClient(%+v) = %T, want nil", cfg, client)
		}
		if reader := NewCFAccessMetadataReader(nil, cfg); reader != nil {
			t.Errorf("NewCFAccessMetadataReader(%+v) = %T, want nil", cfg, reader)
		}
	}
}
