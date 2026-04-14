package control

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"testing"

	catalogpkg "github.com/vrooli/vrooli/internal/resources/catalog"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

func TestStatusRejectsDeprecatedResource(t *testing.T) {
	service := Service{
		IsDeprecatedFn:    func(name string) (bool, error) { return true, nil },
		IsBlueprintArchFn: func(name string) (bool, error) { return false, nil },
	}

	_, err := service.Status("redis", true)
	var resourceErr *vroolierr.Error
	if !errors.As(err, &resourceErr) {
		t.Fatalf("expected *vroolierr.Error, got %T", err)
	}
	if resourceErr.Code != "resource_deprecated" {
		t.Fatalf("resourceErr.Code = %q", resourceErr.Code)
	}
}

func TestRunManifestNativeReturnsDriverErrorWithoutFallback(t *testing.T) {
	service := Service{
		IsDeprecatedFn:    func(name string) (bool, error) { return false, nil },
		IsBlueprintArchFn: func(name string) (bool, error) { return false, nil },
		DiscoverOneFn: func(name string) (*catalogpkg.Resource, error) {
			return &catalogpkg.Resource{
				Name:         name,
				ManifestPath: "/repo/resources/redis/resource.json",
				ControlMode:  "manifest-native",
			}, nil
		},
		LoadManifestFn: func(path string) (manifestpkg.ResourceManifest, error) {
			return manifestpkg.ResourceManifest{Name: "redis", Driver: "external-cli", Binary: "redis", PortabilityTier: "full"}, nil
		},
		DriverRunFn: func(ctx context.Context, item catalogpkg.Resource, manifest manifestpkg.ResourceManifest, operation string, args []string, stdout, stderr io.Writer) error {
			return &vroolierr.Error{Code: ErrorCodeCommandUnavailable, Category: "Driver", Message: "driver unavailable"}
		},
		RunResourceCommandFn: func(name, operation string, args []string, stdout, stderr io.Writer) error {
			t.Fatalf("unexpected fallback command path: %s %s", name, operation)
			return nil
		},
	}

	err := service.Run("redis", []string{"start"}, io.Discard, io.Discard)
	var resourceErr *vroolierr.Error
	if !errors.As(err, &resourceErr) {
		t.Fatalf("expected *vroolierr.Error, got %T", err)
	}
	if resourceErr.Code != ErrorCodeCommandUnavailable {
		t.Fatalf("resourceErr.Code = %q", resourceErr.Code)
	}
}

func TestStatusForResourceCategorizesInvalidPayload(t *testing.T) {
	service := Service{
		CommandForResourceFn: func(name string, args ...string) (*exec.Cmd, error) {
			return exec.Command("echo"), nil
		},
		RunCommandFn: func(ctx context.Context, cmd *exec.Cmd) CommandResult {
			return CommandResult{Output: []byte("not-json")}
		},
	}

	status, err := service.StatusForResource(catalogpkg.Resource{Name: "redis", HasCLI: true}, true)
	if err != nil {
		t.Fatalf("StatusForResource: %v", err)
	}
	if status.StatusCode != StatusCodeInvalidStatusPayload {
		t.Fatalf("status.StatusCode = %q", status.StatusCode)
	}
}

func TestStartAllUsesBestEffortForDegradedStoppedResources(t *testing.T) {
	var operations []string
	service := Service{
		DiscoverFn: func() ([]catalogpkg.Resource, error) {
			return []catalogpkg.Resource{{Name: "redis", Enabled: true, HasCLI: true}}, nil
		},
		CommandForResourceFn: func(name string, args ...string) (*exec.Cmd, error) {
			return exec.Command("echo"), nil
		},
		RunCommandFn: func(ctx context.Context, cmd *exec.Cmd) CommandResult {
			return CommandResult{Output: []byte("not-json")}
		},
		IsDeprecatedFn:    func(name string) (bool, error) { return false, nil },
		IsBlueprintArchFn: func(name string) (bool, error) { return false, nil },
		DiscoverOneFn: func(name string) (*catalogpkg.Resource, error) {
			return &catalogpkg.Resource{Name: name, HasCLI: true}, nil
		},
		RunResourceCommandFn: func(name, operation string, args []string, stdout, stderr io.Writer) error {
			operations = append(operations, operation)
			return nil
		},
	}

	report, err := service.StartAll(&bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("StartAll: %v", err)
	}
	if len(report.Started) != 1 || report.Started[0].Message != "Started successfully after degraded status probe" {
		t.Fatalf("report.Started = %#v", report.Started)
	}
	if len(operations) != 1 || operations[0] != "start" {
		t.Fatalf("operations = %#v", operations)
	}
}
