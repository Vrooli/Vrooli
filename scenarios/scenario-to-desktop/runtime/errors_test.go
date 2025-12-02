package bundleruntime

import (
	"errors"
	"testing"
)

func TestServiceError(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		err := NewServiceError("api", "start", errors.New("connection refused"))
		want := "service api: start: connection refused"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("without underlying error", func(t *testing.T) {
		err := &ServiceError{ServiceID: "worker", Op: "health_check"}
		want := "service worker: health_check failed"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		inner := errors.New("inner error")
		err := NewServiceError("api", "start", inner)
		if !errors.Is(err, inner) {
			t.Error("Unwrap() should return inner error")
		}
	})
}

func TestSecretError(t *testing.T) {
	t.Run("with service ID", func(t *testing.T) {
		err := NewSecretError("API_KEY", "api", "missing")
		want := "secret API_KEY for service api: missing"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("without service ID", func(t *testing.T) {
		err := &SecretError{SecretID: "DB_PASSWORD", Reason: "invalid format"}
		want := "secret DB_PASSWORD: invalid format"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})
}

func TestPortError(t *testing.T) {
	t.Run("with range", func(t *testing.T) {
		err := NewPortError("api", "http", PortRange{Min: 47000, Max: 47100}, "no free ports")
		want := "port http for service api in range 47000-47100: no free ports"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("without range", func(t *testing.T) {
		err := &PortError{ServiceID: "api", PortName: "http", Reason: "not allocated"}
		want := "port http for service api: not allocated"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})
}

func TestAssetError(t *testing.T) {
	t.Run("with sizes", func(t *testing.T) {
		err := &AssetError{
			ServiceID: "browser",
			Path:      "chromium/chrome",
			Reason:    "size mismatch",
			Expected:  1000000,
			Actual:    2000000,
		}
		want := "asset chromium/chrome for service browser: size mismatch (expected 1000000 bytes, got 2000000)"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("without sizes", func(t *testing.T) {
		err := NewAssetError("browser", "chromium/chrome", "missing")
		want := "asset chromium/chrome for service browser: missing"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})
}

func TestMigrationError(t *testing.T) {
	inner := errors.New("syntax error")
	err := NewMigrationError("api", "v2", inner)
	want := "migration v2 for service api: syntax error"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}

	if !errors.Is(err, inner) {
		t.Error("Unwrap() should return inner error")
	}
}

func TestHealthCheckError(t *testing.T) {
	t.Run("with last error", func(t *testing.T) {
		err := NewHealthCheckError("api", "http", 3, errors.New("timeout"))
		want := "health check http for service api failed after 3 attempts: timeout"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("without last error", func(t *testing.T) {
		err := &HealthCheckError{ServiceID: "api", Type: "tcp", Attempts: 5}
		want := "health check tcp for service api failed after 5 attempts"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})
}

func TestIsServiceError(t *testing.T) {
	err := NewServiceError("api", "start", nil)

	if !IsServiceError(err, "api") {
		t.Error("IsServiceError() should return true for matching service")
	}

	if IsServiceError(err, "worker") {
		t.Error("IsServiceError() should return false for non-matching service")
	}

	if IsServiceError(errors.New("other error"), "api") {
		t.Error("IsServiceError() should return false for non-ServiceError")
	}
}

func TestIsSecretError(t *testing.T) {
	secretErr := NewSecretError("API_KEY", "api", "missing")
	otherErr := errors.New("other error")

	if !IsSecretError(secretErr) {
		t.Error("IsSecretError() should return true for SecretError")
	}

	if IsSecretError(otherErr) {
		t.Error("IsSecretError() should return false for non-SecretError")
	}
}

func TestIsAssetError(t *testing.T) {
	assetErr := NewAssetError("browser", "chrome", "missing")
	otherErr := errors.New("other error")

	if !IsAssetError(assetErr) {
		t.Error("IsAssetError() should return true for AssetError")
	}

	if IsAssetError(otherErr) {
		t.Error("IsAssetError() should return false for non-AssetError")
	}
}

func TestIsMigrationError(t *testing.T) {
	migErr := NewMigrationError("api", "v1", errors.New("failed"))
	otherErr := errors.New("other error")

	if !IsMigrationError(migErr) {
		t.Error("IsMigrationError() should return true for MigrationError")
	}

	if IsMigrationError(otherErr) {
		t.Error("IsMigrationError() should return false for non-MigrationError")
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrManifestRequired.Error() != "manifest is required" {
		t.Errorf("ErrManifestRequired = %q", ErrManifestRequired.Error())
	}

	if ErrCyclicDependency.Error() != "cycle detected in dependencies" {
		t.Errorf("ErrCyclicDependency = %q", ErrCyclicDependency.Error())
	}

	if ErrNoFreePorts.Error() != "no free ports in range" {
		t.Errorf("ErrNoFreePorts = %q", ErrNoFreePorts.Error())
	}

	if ErrGPURequired.Error() != "gpu required but not available" {
		t.Errorf("ErrGPURequired = %q", ErrGPURequired.Error())
	}
}
