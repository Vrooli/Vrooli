package types

import (
	"testing"
)

func TestValidationResult_Merge(t *testing.T) {
	t.Run("merge nil is no-op", func(t *testing.T) {
		result := &ValidationResult{
			Valid:     true,
			Errors:    []ValidationError{{Code: "E1"}},
			Warnings:  []ValidationWarning{{Code: "W1"}},
			Platforms: map[string]PlatformValidation{},
		}
		result.Merge(nil)

		if !result.Valid {
			t.Error("Valid should remain true after merging nil")
		}
		if len(result.Errors) != 1 {
			t.Errorf("Errors count = %d, want 1", len(result.Errors))
		}
	})

	t.Run("merge valid into valid stays valid", func(t *testing.T) {
		a := &ValidationResult{
			Valid:     true,
			Errors:    []ValidationError{},
			Warnings:  []ValidationWarning{{Code: "W1"}},
			Platforms: map[string]PlatformValidation{},
		}
		b := &ValidationResult{
			Valid:     true,
			Errors:    []ValidationError{},
			Warnings:  []ValidationWarning{{Code: "W2"}},
			Platforms: map[string]PlatformValidation{},
		}
		a.Merge(b)

		if !a.Valid {
			t.Error("merging two valid results should stay valid")
		}
		if len(a.Warnings) != 2 {
			t.Errorf("Warnings count = %d, want 2", len(a.Warnings))
		}
	})

	t.Run("merge invalid into valid becomes invalid", func(t *testing.T) {
		a := &ValidationResult{
			Valid:     true,
			Errors:    []ValidationError{},
			Warnings:  []ValidationWarning{},
			Platforms: map[string]PlatformValidation{},
		}
		b := &ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{Code: "E1", Message: "something broke"},
				{Code: "E2", Message: "also broke"},
			},
			Warnings:  []ValidationWarning{},
			Platforms: map[string]PlatformValidation{},
		}
		a.Merge(b)

		if a.Valid {
			t.Error("merging invalid result should make receiver invalid")
		}
		if len(a.Errors) != 2 {
			t.Errorf("Errors count = %d, want 2", len(a.Errors))
		}
	})

	t.Run("merge preserves existing invalidity", func(t *testing.T) {
		a := &ValidationResult{
			Valid:     false,
			Errors:    []ValidationError{{Code: "E1"}},
			Warnings:  []ValidationWarning{},
			Platforms: map[string]PlatformValidation{},
		}
		b := &ValidationResult{
			Valid:     true,
			Errors:    []ValidationError{},
			Warnings:  []ValidationWarning{},
			Platforms: map[string]PlatformValidation{},
		}
		a.Merge(b)

		if a.Valid {
			t.Error("merging valid into invalid should stay invalid")
		}
		if len(a.Errors) != 1 {
			t.Errorf("Errors count = %d, want 1", len(a.Errors))
		}
	})

	t.Run("merge combines platform validations for new platform", func(t *testing.T) {
		a := &ValidationResult{
			Valid:    true,
			Errors:   []ValidationError{},
			Warnings: []ValidationWarning{},
			Platforms: map[string]PlatformValidation{
				"windows": {
					Errors:   []string{"WE1"},
					Warnings: []string{},
				},
			},
		}
		b := &ValidationResult{
			Valid:    true,
			Errors:   []ValidationError{},
			Warnings: []ValidationWarning{},
			Platforms: map[string]PlatformValidation{
				"linux": {
					Errors:   []string{"LE1"},
					Warnings: []string{"LW1"},
				},
			},
		}
		a.Merge(b)

		if len(a.Platforms) != 2 {
			t.Fatalf("Platforms count = %d, want 2", len(a.Platforms))
		}
		if len(a.Platforms["linux"].Errors) != 1 {
			t.Errorf("linux errors = %d, want 1", len(a.Platforms["linux"].Errors))
		}
		if len(a.Platforms["windows"].Errors) != 1 {
			t.Errorf("windows errors = %d, want 1", len(a.Platforms["windows"].Errors))
		}
	})

	t.Run("merge combines platform validations for same platform", func(t *testing.T) {
		a := &ValidationResult{
			Valid:    true,
			Errors:   []ValidationError{},
			Warnings: []ValidationWarning{},
			Platforms: map[string]PlatformValidation{
				"windows": {
					Errors:   []string{"WE1"},
					Warnings: []string{"WW1"},
				},
			},
		}
		b := &ValidationResult{
			Valid:    true,
			Errors:   []ValidationError{},
			Warnings: []ValidationWarning{},
			Platforms: map[string]PlatformValidation{
				"windows": {
					Errors:   []string{"WE2"},
					Warnings: []string{"WW2", "WW3"},
				},
			},
		}
		a.Merge(b)

		win := a.Platforms["windows"]
		if len(win.Errors) != 2 {
			t.Errorf("windows errors = %d, want 2", len(win.Errors))
		}
		if len(win.Warnings) != 3 {
			t.Errorf("windows warnings = %d, want 3", len(win.Warnings))
		}
	})

	t.Run("merge into nil platforms map initializes it", func(t *testing.T) {
		a := &ValidationResult{
			Valid:     true,
			Errors:    []ValidationError{},
			Warnings:  []ValidationWarning{},
			Platforms: nil,
		}
		b := &ValidationResult{
			Valid:    true,
			Errors:   []ValidationError{},
			Warnings: []ValidationWarning{},
			Platforms: map[string]PlatformValidation{
				"macos": {
					Errors:   []string{},
					Warnings: []string{"MW1"},
				},
			},
		}
		a.Merge(b)

		if a.Platforms == nil {
			t.Fatal("Platforms should be initialized after merge")
		}
		if len(a.Platforms["macos"].Warnings) != 1 {
			t.Errorf("macos warnings = %d, want 1", len(a.Platforms["macos"].Warnings))
		}
	})
}

func TestConstants(t *testing.T) {
	t.Run("platform constants are distinct", func(t *testing.T) {
		platforms := []string{PlatformWindows, PlatformMacOS, PlatformLinux}
		seen := make(map[string]bool)
		for _, p := range platforms {
			if p == "" {
				t.Error("platform constant should not be empty")
			}
			if seen[p] {
				t.Errorf("duplicate platform constant: %q", p)
			}
			seen[p] = true
		}
	})

	t.Run("certificate source constants are distinct", func(t *testing.T) {
		sources := []string{CertSourceFile, CertSourceStore, CertSourceAzureKeyVault, CertSourceAWSKMS}
		seen := make(map[string]bool)
		for _, s := range sources {
			if s == "" {
				t.Error("cert source constant should not be empty")
			}
			if seen[s] {
				t.Errorf("duplicate cert source constant: %q", s)
			}
			seen[s] = true
		}
	})

	t.Run("signing algorithm constants are distinct", func(t *testing.T) {
		algos := []string{SignAlgorithmSHA256, SignAlgorithmSHA384, SignAlgorithmSHA512}
		seen := make(map[string]bool)
		for _, a := range algos {
			if a == "" {
				t.Error("algo constant should not be empty")
			}
			if seen[a] {
				t.Errorf("duplicate algo constant: %q", a)
			}
			seen[a] = true
		}
	})

	t.Run("default timestamp servers are valid URLs", func(t *testing.T) {
		servers := []string{DefaultTimestampServerDigiCert, DefaultTimestampServerSectigo, DefaultTimestampServerGlobalSign}
		for _, s := range servers {
			if s == "" {
				t.Error("timestamp server should not be empty")
			}
			if len(s) < 8 { // "http://x"
				t.Errorf("timestamp server %q looks too short for a URL", s)
			}
		}
	})
}

func TestSigningConfig_Defaults(t *testing.T) {
	t.Run("zero value has expected defaults", func(t *testing.T) {
		cfg := SigningConfig{}
		if cfg.SchemaVersion != "" {
			t.Errorf("zero SchemaVersion = %q, want empty", cfg.SchemaVersion)
		}
		if cfg.Windows != nil {
			t.Error("zero Windows should be nil")
		}
		if cfg.MacOS != nil {
			t.Error("zero MacOS should be nil")
		}
		if cfg.Linux != nil {
			t.Error("zero Linux should be nil")
		}
	})
}

func TestElectronBuilderSigningConfig(t *testing.T) {
	t.Run("struct fields are independently settable", func(t *testing.T) {
		notarize := NotarizeConfig{TeamID: "TEAM123"}
		cfg := ElectronBuilderSigningConfig{
			Win: &ElectronBuilderWinSigning{
				CertificateFile:       "/path/to/cert.pfx",
				CertificatePassword:   "secret",
				SigningHashAlgorithms: []string{"sha256", "sha512"},
				TimeStampServer:       DefaultTimestampServerDigiCert,
			},
			Mac: &ElectronBuilderMacSigning{
				Identity: "Developer ID",
				Notarize: notarize,
			},
		}

		if cfg.Win.CertificateFile != "/path/to/cert.pfx" {
			t.Errorf("Win.CertificateFile = %q, want %q", cfg.Win.CertificateFile, "/path/to/cert.pfx")
		}
		if len(cfg.Win.SigningHashAlgorithms) != 2 {
			t.Errorf("Win.SigningHashAlgorithms len = %d, want 2", len(cfg.Win.SigningHashAlgorithms))
		}
		if cfg.Mac.Identity != "Developer ID" {
			t.Errorf("Mac.Identity = %q, want %q", cfg.Mac.Identity, "Developer ID")
		}
		nc, ok := cfg.Mac.Notarize.(NotarizeConfig)
		if !ok {
			t.Fatal("Notarize should be NotarizeConfig type")
		}
		if nc.TeamID != "TEAM123" {
			t.Errorf("Notarize.TeamID = %q, want %q", nc.TeamID, "TEAM123")
		}
	})
}
