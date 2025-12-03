package smoke

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Timeout != DefaultTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultTimeout, cfg.Timeout)
	}
	if cfg.HandshakeTimeout != DefaultHandshakeTimeout {
		t.Errorf("expected handshake timeout %v, got %v", DefaultHandshakeTimeout, cfg.HandshakeTimeout)
	}
	if cfg.Viewport.Width != DefaultViewportWidth {
		t.Errorf("expected viewport width %d, got %d", DefaultViewportWidth, cfg.Viewport.Width)
	}
	if cfg.Viewport.Height != DefaultViewportHeight {
		t.Errorf("expected viewport height %d, got %d", DefaultViewportHeight, cfg.Viewport.Height)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				ScenarioName:     "test-scenario",
				ScenarioDir:      "/path/to/scenario",
				BrowserlessURL:   "http://localhost:4110",
				Timeout:          90 * time.Second,
				HandshakeTimeout: 15 * time.Second,
				Viewport:         DefaultViewport(),
			},
			wantErr: false,
		},
		{
			name: "missing scenario name",
			cfg: Config{
				ScenarioDir:      "/path/to/scenario",
				BrowserlessURL:   "http://localhost:4110",
				Timeout:          90 * time.Second,
				HandshakeTimeout: 15 * time.Second,
				Viewport:         DefaultViewport(),
			},
			wantErr: true,
		},
		{
			name: "missing scenario dir",
			cfg: Config{
				ScenarioName:     "test-scenario",
				BrowserlessURL:   "http://localhost:4110",
				Timeout:          90 * time.Second,
				HandshakeTimeout: 15 * time.Second,
				Viewport:         DefaultViewport(),
			},
			wantErr: true,
		},
		{
			name: "missing browserless URL",
			cfg: Config{
				ScenarioName:     "test-scenario",
				ScenarioDir:      "/path/to/scenario",
				Timeout:          90 * time.Second,
				HandshakeTimeout: 15 * time.Second,
				Viewport:         DefaultViewport(),
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			cfg: Config{
				ScenarioName:     "test-scenario",
				ScenarioDir:      "/path/to/scenario",
				BrowserlessURL:   "http://localhost:4110",
				Timeout:          0,
				HandshakeTimeout: 15 * time.Second,
				Viewport:         DefaultViewport(),
			},
			wantErr: true,
		},
		{
			name: "zero handshake timeout",
			cfg: Config{
				ScenarioName:     "test-scenario",
				ScenarioDir:      "/path/to/scenario",
				BrowserlessURL:   "http://localhost:4110",
				Timeout:          90 * time.Second,
				HandshakeTimeout: 0,
				Viewport:         DefaultViewport(),
			},
			wantErr: true,
		},
		{
			name: "zero viewport width",
			cfg: Config{
				ScenarioName:     "test-scenario",
				ScenarioDir:      "/path/to/scenario",
				BrowserlessURL:   "http://localhost:4110",
				Timeout:          90 * time.Second,
				HandshakeTimeout: 15 * time.Second,
				Viewport:         Viewport{Width: 0, Height: 720},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigTimeoutMs(t *testing.T) {
	cfg := Config{Timeout: 90 * time.Second}
	if got := cfg.TimeoutMs(); got != 90000 {
		t.Errorf("TimeoutMs() = %d, want 90000", got)
	}
}

func TestConfigHandshakeTimeoutMs(t *testing.T) {
	cfg := Config{HandshakeTimeout: 15 * time.Second}
	if got := cfg.HandshakeTimeoutMs(); got != 15000 {
		t.Errorf("HandshakeTimeoutMs() = %d, want 15000", got)
	}
}

func TestConfigResolveUIURL(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "explicit UIURL",
			cfg:  Config{UIURL: "http://example.com:8080"},
			want: "http://example.com:8080",
		},
		{
			name: "UIPort only",
			cfg:  Config{UIPort: 3000},
			want: "http://localhost:3000",
		},
		{
			name: "UIURL takes precedence over UIPort",
			cfg:  Config{UIURL: "http://example.com", UIPort: 3000},
			want: "http://example.com",
		},
		{
			name: "neither set",
			cfg:  Config{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.ResolveUIURL(); got != tt.want {
				t.Errorf("ResolveUIURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
