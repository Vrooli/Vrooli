package sessionprofile

import (
	"strings"
	"testing"

	"github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

// =============================================================================
// ValidateBrowserProfile Tests
// =============================================================================

func TestValidateBrowserProfile_NilProfile(t *testing.T) {
	t.Parallel()

	err := ValidateBrowserProfile(nil)
	if err != nil {
		t.Errorf("expected nil error for nil profile, got: %v", err)
	}
}

func TestValidateBrowserProfile_EmptyProfile(t *testing.T) {
	t.Parallel()

	err := ValidateBrowserProfile(&persistence.BrowserProfile{})
	if err != nil {
		t.Errorf("expected nil error for empty profile, got: %v", err)
	}
}

func TestValidateBrowserProfile_Preset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		preset    string
		wantErr   bool
		errSubstr string
		category  string
	}{
		// Valid presets
		{"empty_preset", "", false, "", "valid"},
		{"stealth_preset", "stealth", false, "", "valid"},
		{"balanced_preset", "balanced", false, "", "valid"},
		{"fast_preset", "fast", false, "", "valid"},
		{"none_preset", "none", false, "", "valid"},

		// Invalid presets
		{"invalid_preset", "invalid", true, "invalid preset", "error"},
		{"case_sensitive", "Stealth", true, "invalid preset", "error"},
		{"numeric_preset", "123", true, "invalid preset", "error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{Preset: tc.preset}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for preset %q", tc.preset)
					return
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("expected error containing %q, got: %v", tc.errSubstr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error for preset %q: %v", tc.preset, err)
			}
		})
	}
}

func TestValidateBrowserProfile_Fingerprint_ViewportWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		width     int
		wantErr   bool
		errSubstr string
		category  string
	}{
		// Boundary values
		{"below_min", -1, true, "viewport_width", "boundary"},
		{"at_min", 0, false, "", "boundary"},
		{"above_min", 1, false, "", "boundary"},
		{"below_max", 7679, false, "", "boundary"},
		{"at_max", 7680, false, "", "boundary"},
		{"above_max", 7681, true, "viewport_width", "boundary"},

		// Common values
		{"common_1920", 1920, false, "", "valid"},
		{"common_1280", 1280, false, "", "valid"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{
				Fingerprint: &persistence.FingerprintSettings{
					ViewportWidth: tc.width,
				},
			}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for width %d", tc.width)
					return
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("expected error containing %q, got: %v", tc.errSubstr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error for width %d: %v", tc.width, err)
			}
		})
	}
}

func TestValidateBrowserProfile_Fingerprint_ViewportHeight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		height    int
		wantErr   bool
		errSubstr string
	}{
		// Boundary values
		{"below_min", -1, true, "viewport_height"},
		{"at_min", 0, false, ""},
		{"at_max", 4320, false, ""},
		{"above_max", 4321, true, "viewport_height"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{
				Fingerprint: &persistence.FingerprintSettings{
					ViewportHeight: tc.height,
				},
			}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
					return
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("expected error containing %q, got: %v", tc.errSubstr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateBrowserProfile_Fingerprint_DeviceScaleFactor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		factor  float64
		wantErr bool
	}{
		{"below_min", -0.1, true},
		{"at_min", 0, false},
		{"common_1", 1, false},
		{"common_2", 2, false},
		{"at_max", 5, false},
		{"above_max", 5.1, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{
				Fingerprint: &persistence.FingerprintSettings{
					DeviceScaleFactor: tc.factor,
				},
			}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr && err == nil {
				t.Error("expected error")
			} else if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateBrowserProfile_Fingerprint_Geolocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		lat       float64
		lon       float64
		wantErr   bool
		errSubstr string
	}{
		// Valid coordinates
		{"origin", 0, 0, false, ""},
		{"nyc", 40.7128, -74.0060, false, ""},
		{"sydney", -33.8688, 151.2093, false, ""},

		// Latitude boundaries
		{"lat_at_min", -90, 0, false, ""},
		{"lat_below_min", -90.1, 0, true, "latitude"},
		{"lat_at_max", 90, 0, false, ""},
		{"lat_above_max", 90.1, 0, true, "latitude"},

		// Longitude boundaries
		{"lon_at_min", 0, -180, false, ""},
		{"lon_below_min", 0, -180.1, true, "longitude"},
		{"lon_at_max", 0, 180, false, ""},
		{"lon_above_max", 0, 180.1, true, "longitude"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{
				Fingerprint: &persistence.FingerprintSettings{
					Latitude:  tc.lat,
					Longitude: tc.lon,
				},
			}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
					return
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("expected error containing %q, got: %v", tc.errSubstr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateBrowserProfile_Fingerprint_ColorScheme(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		scheme  string
		wantErr bool
	}{
		{"empty", "", false},
		{"light", "light", false},
		{"dark", "dark", false},
		{"no_preference", "no-preference", false},
		{"invalid", "auto", true},
		{"case_sensitive", "Light", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{
				Fingerprint: &persistence.FingerprintSettings{
					ColorScheme: tc.scheme,
				},
			}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr && err == nil {
				t.Error("expected error")
			} else if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateBrowserProfile_Behavior_TypingDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		min       int
		max       int
		wantErr   bool
		errSubstr string
	}{
		// Valid ranges
		{"zero_both", 0, 0, false, ""},
		{"normal_range", 50, 150, false, ""},
		{"same_values", 100, 100, false, ""},

		// Boundary tests
		{"max_at_limit", 0, 5000, false, ""},
		{"max_over_limit", 0, 5001, true, "typing_delay_max"},
		{"min_negative", -1, 100, true, "typing_delay_min"},

		// Min > Max (only invalid when max > 0)
		{"min_exceeds_max", 200, 100, true, "cannot exceed"},
		{"min_exceeds_zero_max", 100, 0, false, ""}, // Zero max means "no max"
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{
				Behavior: &persistence.BehaviorSettings{
					TypingDelayMin: tc.min,
					TypingDelayMax: tc.max,
				},
			}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
					return
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("expected error containing %q, got: %v", tc.errSubstr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateBrowserProfile_Behavior_TypingPasteThreshold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		threshold int
		wantErr   bool
	}{
		{"always_paste", -1, false},  // -1 = always paste
		{"always_type", 0, false},    // 0 = always type
		{"threshold_100", 100, false},
		{"at_max", 10000, false},
		{"below_min", -2, true},
		{"above_max", 10001, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{
				Behavior: &persistence.BehaviorSettings{
					TypingPasteThreshold: tc.threshold,
				},
			}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr && err == nil {
				t.Error("expected error")
			} else if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateBrowserProfile_Behavior_MouseMovementStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		style   string
		wantErr bool
	}{
		{"empty", "", false},
		{"linear", "linear", false},
		{"bezier", "bezier", false},
		{"natural", "natural", false},
		{"invalid", "random", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{
				Behavior: &persistence.BehaviorSettings{
					MouseMovementStyle: tc.style,
				},
			}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr && err == nil {
				t.Error("expected error")
			} else if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateBrowserProfile_Behavior_ScrollStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		style   string
		wantErr bool
	}{
		{"empty", "", false},
		{"smooth", "smooth", false},
		{"stepped", "stepped", false},
		{"invalid", "instant", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{
				Behavior: &persistence.BehaviorSettings{
					ScrollStyle: tc.style,
				},
			}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr && err == nil {
				t.Error("expected error")
			} else if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateBrowserProfile_Behavior_MicroPauseFrequency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		freq    float64
		wantErr bool
	}{
		{"at_min", 0, false},
		{"mid_range", 0.5, false},
		{"at_max", 1, false},
		{"below_min", -0.1, true},
		{"above_max", 1.1, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{
				Behavior: &persistence.BehaviorSettings{
					MicroPauseFrequency: tc.freq,
				},
			}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr && err == nil {
				t.Error("expected error")
			} else if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateBrowserProfile_AntiDetection_AdBlockingMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{"empty", "", false},
		{"none", "none", false},
		{"ads_only", "ads_only", false},
		{"ads_and_tracking", "ads_and_tracking", false},
		{"invalid", "full", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{
				AntiDetection: &persistence.AntiDetectionSettings{
					AdBlockingMode: tc.mode,
				},
			}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr && err == nil {
				t.Error("expected error")
			} else if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateBrowserProfile_Proxy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		proxy     *persistence.ProxySettings
		wantErr   bool
		errSubstr string
	}{
		{
			"nil_proxy",
			nil,
			false,
			"",
		},
		{
			"disabled_with_server",
			&persistence.ProxySettings{Enabled: false, Server: "http://proxy.example.com"},
			false,
			"",
		},
		{
			"enabled_no_server",
			&persistence.ProxySettings{Enabled: true, Server: ""},
			true,
			"proxy server is required",
		},
		{
			"enabled_http_proxy",
			&persistence.ProxySettings{Enabled: true, Server: "http://proxy.example.com:8080"},
			false,
			"",
		},
		{
			"enabled_https_proxy",
			&persistence.ProxySettings{Enabled: true, Server: "https://proxy.example.com:8080"},
			false,
			"",
		},
		{
			"enabled_socks5_proxy",
			&persistence.ProxySettings{Enabled: true, Server: "socks5://proxy.example.com:1080"},
			false,
			"",
		},
		{
			"invalid_protocol",
			&persistence.ProxySettings{Enabled: true, Server: "ftp://proxy.example.com"},
			true,
			"must start with http://",
		},
		{
			"no_protocol",
			&persistence.ProxySettings{Enabled: true, Server: "proxy.example.com:8080"},
			true,
			"must start with http://",
		},
		{
			"password_without_username",
			&persistence.ProxySettings{Enabled: true, Server: "http://proxy.example.com", Password: "secret"},
			true,
			"username is required when password is set",
		},
		{
			"username_with_password",
			&persistence.ProxySettings{Enabled: true, Server: "http://proxy.example.com", Username: "user", Password: "secret"},
			false,
			"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{Proxy: tc.proxy}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
					return
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("expected error containing %q, got: %v", tc.errSubstr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateBrowserProfile_ExtraHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		headers   map[string]string
		wantErr   bool
		errSubstr string
	}{
		{
			"nil_headers",
			nil,
			false,
			"",
		},
		{
			"empty_headers",
			map[string]string{},
			false,
			"",
		},
		{
			"valid_custom_header",
			map[string]string{"X-Custom": "value"},
			false,
			"",
		},
		{
			"blocked_host",
			map[string]string{"Host": "evil.com"},
			true,
			"cannot be set",
		},
		{
			"blocked_host_lowercase",
			map[string]string{"host": "evil.com"},
			true,
			"cannot be set",
		},
		{
			"blocked_content_length",
			map[string]string{"Content-Length": "100"},
			true,
			"cannot be set",
		},
		{
			"blocked_cookie",
			map[string]string{"Cookie": "session=abc"},
			true,
			"cannot be set",
		},
		{
			"multiple_with_one_blocked",
			map[string]string{"X-Valid": "ok", "Host": "bad"},
			true,
			"cannot be set",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bp := &persistence.BrowserProfile{ExtraHeaders: tc.headers}
			err := ValidateBrowserProfile(bp)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
					return
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("expected error containing %q, got: %v", tc.errSubstr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// =============================================================================
// ValidateHistorySettings Tests
// =============================================================================

func TestValidateHistorySettings_Nil(t *testing.T) {
	t.Parallel()

	err := ValidateHistorySettings(nil)
	if err != nil {
		t.Errorf("expected nil error for nil settings, got: %v", err)
	}
}

func TestValidateHistorySettings_MaxEntries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		entries   int
		wantErr   bool
		errSubstr string
	}{
		{"below_min", -1, true, "max_entries"},
		{"at_min", 0, false, ""},
		{"common_value", 100, false, ""},
		{"at_max", 10000, false, ""},
		{"above_max", 10001, true, "max_entries"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			settings := &persistence.HistorySettings{MaxEntries: tc.entries}
			err := ValidateHistorySettings(settings)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
					return
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("expected error containing %q, got: %v", tc.errSubstr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateHistorySettings_RetentionDays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		days      int
		wantErr   bool
		errSubstr string
	}{
		{"below_min", -1, true, "retention_days"},
		{"at_min", 0, false, ""},
		{"one_year", 365, false, ""},
		{"at_max_10_years", 3650, false, ""},
		{"above_max", 3651, true, "retention_days"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			settings := &persistence.HistorySettings{RetentionDays: tc.days}
			err := ValidateHistorySettings(settings)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
					return
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("expected error containing %q, got: %v", tc.errSubstr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
