package control

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

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
			return manifestpkg.ResourceManifest{Name: "redis", Driver: "external-cli", Binary: "redis"}, nil
		},
		DriverRunFn: func(ctx context.Context, item catalogpkg.Resource, manifest manifestpkg.ResourceManifest, operation string, args []string, stdout, stderr io.Writer) error {
			return &vroolierr.Error{Code: ErrorCodeCommandUnavailable, Category: "Driver", Message: "driver unavailable"}
		},
		RunResourceCommandFn: func(_ context.Context, name, operation string, args []string, stdout, stderr io.Writer) error {
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

	status, err := service.StatusForResource(catalogpkg.Resource{Name: "redis", DeclaresCLI: true, CLIInstalled: true}, true)
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
			return []catalogpkg.Resource{{Name: "redis", Enabled: true, DeclaresCLI: true, CLIInstalled: true}}, nil
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
			return &catalogpkg.Resource{Name: name, DeclaresCLI: true, CLIInstalled: true}, nil
		},
		RunResourceCommandFn: func(_ context.Context, name, operation string, args []string, stdout, stderr io.Writer) error {
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

func TestManagedServiceOperationIncludesAuthoredStartupBudget(t *testing.T) {
	for index, test := range []struct {
		operation, driver, override, healthOverride string
		seconds                                     int
		want                                        time.Duration
	}{
		{operation: "start", driver: "managed-service", seconds: 600, want: 12 * time.Minute},
		{operation: "restart", driver: "managed-service", seconds: 600, want: 12 * time.Minute},
		{operation: "start", driver: "managed-service", want: 3 * time.Minute},
		{operation: "stop", driver: "managed-service", seconds: 600, want: 2 * time.Minute},
		{operation: "install", driver: "managed-service", seconds: 600, want: 30 * time.Minute},
		{operation: "start", driver: "external-cli", seconds: 600, want: 2 * time.Minute},
		{operation: "start", driver: "managed-service", seconds: 600, override: "7s", want: 7 * time.Second},
		{operation: "start", driver: "managed-service", seconds: 600, healthOverride: "9s", want: 2*time.Minute + 9*time.Second},
	} {
		selected := os.Getenv("VROOLI_START_BUDGET_CASE")
		if selected != "" && selected != strconv.Itoa(index) {
			continue
		}
		t.Run(test.operation+"/"+test.driver+"/"+test.override+"/"+test.healthOverride+"/"+strconv.Itoa(test.seconds)+"s", func(t *testing.T) {
			// Timing policy resolves once per process, so override contracts
			// need a fresh test process rather than mutating a cached lever.
			if selected == "" && (test.override != "" || test.healthOverride != "") {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestManagedServiceOperationIncludesAuthoredStartupBudget$")
				cmd.Env = append(os.Environ(), "VROOLI_START_BUDGET_CASE="+strconv.Itoa(index))
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("fresh policy process: %v: %s", err, output)
				}
				return
			}
			t.Setenv("VROOLI_TUNING_RESOURCE_OPERATION_TIMEOUT", test.override)
			t.Setenv("VROOLI_TUNING_MANAGED_SERVICE_CONFIGURED_TIMEOUT", test.healthOverride)
			t.Setenv("VROOLI_TUNING_RESOURCE_CONTROL_EXTENDED_TIMEOUT", "")
			service := Service{
				LoadManifestFn: func(string) (manifestpkg.ResourceManifest, error) {
					return manifestpkg.ResourceManifest{Driver: test.driver, Lifecycle: manifestpkg.ResourceLifecycle{StartTimeoutSeconds: test.seconds}}, nil
				},
				DriverRunFn: func(ctx context.Context, _ catalogpkg.Resource, _ manifestpkg.ResourceManifest, _ string, _ []string, _, _ io.Writer) error {
					deadline, ok := ctx.Deadline()
					if !ok {
						t.Fatal("operation has no deadline")
					}
					remaining := time.Until(deadline)
					if remaining > test.want || remaining < test.want-time.Second {
						t.Fatalf("operation budget=%s want %s", remaining, test.want)
					}
					return nil
				},
			}
			if err := service.runNativeResourceCommand(context.Background(), catalogpkg.Resource{Name: "fixture"}, test.operation, nil, io.Discard, io.Discard); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestManagedStartupOperationUsesDriverOwnedBudgetAndRejectsInvalidBounds(t *testing.T) {
	for _, test := range []struct {
		name      string
		startup   time.Duration
		wantError bool
	}{
		{name: "observed-collection-budget", startup: 110 * time.Second},
		{name: "invalid-zero", wantError: true},
		{name: "invalid-negative", startup: -time.Second, wantError: true},
		{name: "overflow", startup: time.Duration(1<<63 - 1), wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			service := Service{
				LoadManifestFn: func(string) (manifestpkg.ResourceManifest, error) {
					return manifestpkg.ResourceManifest{Driver: "managed-service", Lifecycle: manifestpkg.ResourceLifecycle{StartTimeoutSeconds: 90}}, nil
				},
				StartupTimeoutFn: func(manifestpkg.ResourceManifest) time.Duration { return test.startup },
				DriverRunFn: func(ctx context.Context, _ catalogpkg.Resource, _ manifestpkg.ResourceManifest, _ string, _ []string, _, _ io.Writer) error {
					called = true
					deadline, ok := ctx.Deadline()
					remaining := time.Until(deadline)
					if !ok || remaining > 230*time.Second || remaining < 229*time.Second {
						t.Fatalf("driver-owned derived budget lost: %s", remaining)
					}
					return nil
				},
			}
			err := service.runNativeResourceCommand(context.Background(), catalogpkg.Resource{Name: "fixture"}, "start", nil, io.Discard, io.Discard)
			if (err != nil) != test.wantError || called == test.wantError {
				t.Fatalf("invalid boundary reached driver or valid budget refused: called=%v err=%v", called, err)
			}
		})
	}
}

func TestRunContextCancelsNativeAndLegacyOwners(t *testing.T) {
	for _, native := range []bool{true, false} {
		t.Run(strconv.FormatBool(native), func(t *testing.T) {
			entered := make(chan struct{})
			nativeCalls, legacyCalls := 0, 0
			parent, cancel := context.WithCancel(context.Background())
			defer cancel()
			service := Service{IsDeprecatedFn: func(string) (bool, error) { return false, nil }, IsBlueprintArchFn: func(string) (bool, error) { return false, nil }, DiscoverOneFn: func(name string) (*catalogpkg.Resource, error) {
				mode := "legacy"
				if native {
					mode = "manifest-native"
				}
				path := ""
				if native {
					path = "/fixture/resource.json"
				}
				return &catalogpkg.Resource{Name: name, ControlMode: mode, ManifestPath: path}, nil
			}, LoadManifestFn: func(path string) (manifestpkg.ResourceManifest, error) {
				if path != "/fixture/resource.json" {
					t.Errorf("unexpected manifest path: %q", path)
				}
				return manifestpkg.ResourceManifest{Driver: "managed-service", Lifecycle: manifestpkg.ResourceLifecycle{StartTimeoutSeconds: 600}}, nil
			}}
			observe := func(ctx context.Context) error {
				close(entered)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Second):
					return errors.New("caller context detached")
				}
			}
			service.DriverRunFn = func(ctx context.Context, _ catalogpkg.Resource, _ manifestpkg.ResourceManifest, _ string, _ []string, _, _ io.Writer) error {
				nativeCalls++
				return observe(ctx)
			}
			service.RunResourceCommandFn = func(ctx context.Context, _, _ string, _ []string, _, _ io.Writer) error {
				legacyCalls++
				return observe(ctx)
			}
			done := make(chan error, 1)
			go func() { done <- service.RunContext(parent, "fixture", []string{"restart"}, io.Discard, io.Discard) }()
			<-entered
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("context lost: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("owner did not cancel")
			}
			wantNative, wantLegacy := 0, 1
			if native {
				wantNative, wantLegacy = 1, 0
			}
			if nativeCalls != wantNative || legacyCalls != wantLegacy {
				t.Fatalf("wrong owner selected: native=%d legacy=%d want %d/%d", nativeCalls, legacyCalls, wantNative, wantLegacy)
			}
		})
	}
}

func TestRunContextPrecancelDoesNotDiscoverOrDispatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := Service{IsDeprecatedFn: func(string) (bool, error) { t.Error("cancelled operation validated/discovered"); return false, nil }}
	if err := service.RunContext(ctx, "fixture", []string{"restart"}, io.Discard, io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancel lost: %v", err)
	}
}
