package codesigning

import (
	"testing"
	"time"
)

func TestDefaultSigningConfig(t *testing.T) {
	config := DefaultSigningConfig()

	if config.Enabled {
		t.Error("default signing config should have enabled=false")
	}
	if config.Windows != nil {
		t.Error("default signing config should have nil windows")
	}
	if config.MacOS != nil {
		t.Error("default signing config should have nil macos")
	}
	if config.Linux != nil {
		t.Error("default signing config should have nil linux")
	}
}

func TestDefaultWindowsConfig(t *testing.T) {
	config := DefaultWindowsConfig()

	if config.CertificateSource != CertSourceFile {
		t.Errorf("expected certificate_source=%s, got %s", CertSourceFile, config.CertificateSource)
	}
	if config.TimestampServer != DefaultTimestampServerDigiCert {
		t.Errorf("expected timestamp_server=%s, got %s", DefaultTimestampServerDigiCert, config.TimestampServer)
	}
	if config.SignAlgorithm != SignAlgorithmSHA256 {
		t.Errorf("expected sign_algorithm=%s, got %s", SignAlgorithmSHA256, config.SignAlgorithm)
	}
	if config.DualSign {
		t.Error("expected dual_sign=false")
	}
}

func TestDefaultMacOSConfig(t *testing.T) {
	config := DefaultMacOSConfig()

	if !config.HardenedRuntime {
		t.Error("expected hardened_runtime=true")
	}
	if !config.Notarize {
		t.Error("expected notarize=true")
	}
}

func TestIsValidCertificateSource(t *testing.T) {
	tests := []struct {
		source string
		valid  bool
	}{
		{CertSourceFile, true},
		{CertSourceStore, true},
		{CertSourceAzureKeyVault, true},
		{CertSourceAWSKMS, true},
		{"invalid", false},
		{"", false},
		{"FILE", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			result := IsValidCertificateSource(tt.source)
			if result != tt.valid {
				t.Errorf("IsValidCertificateSource(%q) = %v, want %v", tt.source, result, tt.valid)
			}
		})
	}
}

func TestIsValidSignAlgorithm(t *testing.T) {
	tests := []struct {
		algo  string
		valid bool
	}{
		{SignAlgorithmSHA256, true},
		{SignAlgorithmSHA384, true},
		{SignAlgorithmSHA512, true},
		{"sha1", false},
		{"md5", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.algo, func(t *testing.T) {
			result := IsValidSignAlgorithm(tt.algo)
			if result != tt.valid {
				t.Errorf("IsValidSignAlgorithm(%q) = %v, want %v", tt.algo, result, tt.valid)
			}
		})
	}
}

func TestIsValidPlatform(t *testing.T) {
	tests := []struct {
		platform string
		valid    bool
	}{
		{PlatformWindows, true},
		{PlatformMacOS, true},
		{PlatformLinux, true},
		{"Windows", false}, // case-sensitive
		{"darwin", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			result := IsValidPlatform(tt.platform)
			if result != tt.valid {
				t.Errorf("IsValidPlatform(%q) = %v, want %v", tt.platform, result, tt.valid)
			}
		})
	}
}

func TestValidationResult_AddError(t *testing.T) {
	result := NewValidationResult()

	if !result.Valid {
		t.Error("new validation result should be valid")
	}

	result.AddError(ValidationError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
	})

	if result.Valid {
		t.Error("validation result should be invalid after adding error")
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestValidationResult_AddWarning(t *testing.T) {
	result := NewValidationResult()

	result.AddWarning(ValidationWarning{
		Code:    "TEST_WARNING",
		Message: "Test warning message",
	})

	if !result.Valid {
		t.Error("validation result should still be valid after adding warning")
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
}

func TestCalculateDaysToExpiry(t *testing.T) {
	tests := []struct {
		name     string
		notAfter time.Time
		now      time.Time
		expected int
	}{
		{
			name:     "30 days",
			notAfter: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
			now:      time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			expected: 30,
		},
		{
			name:     "1 day",
			notAfter: time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC),
			now:      time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "already expired",
			notAfter: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
			now:      time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			expected: -31,
		},
		{
			name:     "365 days",
			notAfter: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			now:      time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			expected: 365,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateDaysToExpiry(tt.notAfter, tt.now)
			if result != tt.expected {
				t.Errorf("CalculateDaysToExpiry() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestIsCertificateExpired(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		notAfter time.Time
		expected bool
	}{
		{"not expired", time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), false},
		{"exactly now", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), false},
		{"expired", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCertificateExpired(tt.notAfter, now)
			if result != tt.expected {
				t.Errorf("IsCertificateExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsCertificateExpiringWarning(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		notAfter time.Time
		expected bool
	}{
		{"31 days - no warning", time.Date(2025, 7, 2, 0, 0, 0, 0, time.UTC), false},
		{"30 days - warning", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), true},
		{"15 days - warning", time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC), true},
		{"7 days - warning", time.Date(2025, 6, 8, 0, 0, 0, 0, time.UTC), true},
		{"already expired", time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCertificateExpiringWarning(tt.notAfter, now)
			if result != tt.expected {
				t.Errorf("IsCertificateExpiringWarning() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsCertificateExpiringCritical(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		notAfter time.Time
		expected bool
	}{
		{"30 days - not critical", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), false},
		{"8 days - not critical", time.Date(2025, 6, 9, 0, 0, 0, 0, time.UTC), false},
		{"7 days - critical", time.Date(2025, 6, 8, 0, 0, 0, 0, time.UTC), true},
		{"3 days - critical", time.Date(2025, 6, 4, 0, 0, 0, 0, time.UTC), true},
		{"already expired", time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCertificateExpiringCritical(tt.notAfter, now)
			if result != tt.expected {
				t.Errorf("IsCertificateExpiringCritical() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// mockEnvironmentReader implements EnvironmentReader for testing.
type mockEnvironmentReader struct {
	env map[string]string
}

func (m *mockEnvironmentReader) GetEnv(key string) string {
	return m.env[key]
}

func (m *mockEnvironmentReader) LookupEnv(key string) (string, bool) {
	val, ok := m.env[key]
	return val, ok
}

func TestResolveEnvValue(t *testing.T) {
	env := &mockEnvironmentReader{
		env: map[string]string{
			"MY_PASSWORD": "secret123",
			"EMPTY_VAR":   "",
		},
	}

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"direct value", "plaintext", "plaintext"},
		{"env reference", "$MY_PASSWORD", "secret123"},
		{"missing env", "$NOT_SET", ""},
		{"empty env", "$EMPTY_VAR", ""},
		{"dollar only", "$", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveEnvValue(tt.value, env)
			if result != tt.expected {
				t.Errorf("ResolveEnvValue(%q) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestRealTimeProvider(t *testing.T) {
	provider := NewRealTimeProvider()

	before := time.Now()
	result := provider.Now()
	after := time.Now()

	if result.Before(before) || result.After(after) {
		t.Error("RealTimeProvider.Now() returned invalid time")
	}
}

func TestRealEnvironmentReader(t *testing.T) {
	reader := NewRealEnvironmentReader()

	// Test GetEnv with PATH (always set)
	path := reader.GetEnv("PATH")
	if path == "" {
		t.Error("expected PATH to be set")
	}

	// Test LookupEnv
	_, exists := reader.LookupEnv("PATH")
	if !exists {
		t.Error("expected PATH to exist")
	}

	_, exists = reader.LookupEnv("DEFINITELY_NOT_SET_12345")
	if exists {
		t.Error("expected DEFINITELY_NOT_SET_12345 to not exist")
	}
}
