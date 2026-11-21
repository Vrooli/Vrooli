package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
)

func TestMainDispatchesInspectionWorkerBeforeAPIStartup(t *testing.T) {
	args := []string{"inspect-worker", "--manifest", "/tmp/attempt/inspection-manifest.json"}
	called := ""
	handled, err := dispatchOwnedWorker(args,
		func([]string) error {
			called = "render"
			return nil
		},
		func(got []string) error {
			called = "inspection"
			if len(got) != len(args) || got[0] != args[0] || got[2] != args[2] {
				t.Fatalf("inspection worker args = %#v, want %#v", got, args)
			}
			return nil
		},
		func([]string) error { return nil },
	)
	if err != nil || !handled || called != "inspection" {
		t.Fatalf("inspection dispatch = (handled %v, err %v, called %q)", handled, err, called)
	}
}

func TestMainDispatchKeepsRenderWorkerAndAPIPathsDistinct(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "render worker", args: []string{"render-worker", "--manifest", "/tmp/attempt/render-manifest.json"}, want: "render"},
		{name: "video worker", args: []string{"video-render-worker", "--manifest", "/tmp/attempt/video-manifest.json"}, want: "video"},
		{name: "physical worker", args: []string{"physical-render-worker", "--manifest", "/tmp/attempt/physical-render-manifest.json"}, want: "physical"},
		{name: "ordinary API", args: []string{"--port", "19349"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := ""
			handled, err := dispatchOwnedWorker(tc.args,
				func([]string) error { called = "render"; return nil },
				func([]string) error { called = "inspection"; return nil },
				func([]string) error { called = "video"; return nil },
				func([]string) error { called = "physical"; return nil },
			)
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if handled || called != "" {
					t.Fatalf("ordinary API invocation dispatched as %q", called)
				}
				return
			}
			if !handled || called != tc.want {
				t.Fatalf("dispatch = (handled %v, called %q), want %q", handled, called, tc.want)
			}
		})
	}
}

func TestMainDispatchReturnsVideoWorkerError(t *testing.T) {
	wantErr := errors.New("video worker error")
	handled, err := dispatchOwnedWorker([]string{"video-render-worker", "--bad"},
		func([]string) error { return nil },
		func([]string) error { return nil },
		func([]string) error { return wantErr },
	)
	if !handled || !errors.Is(err, wantErr) {
		t.Fatalf("video worker error dispatch = (handled %v, err %v)", handled, err)
	}
}

func TestMainDispatchReturnsInspectionWorkerError(t *testing.T) {
	wantErr := errors.New("inspection worker error")
	handled, err := dispatchOwnedWorker([]string{"inspect-worker", "--bad"},
		func([]string) error { return nil },
		func([]string) error { return wantErr },
		func([]string) error { return nil },
	)
	if !handled || !errors.Is(err, wantErr) {
		t.Fatalf("inspection error dispatch = (handled %v, err %v)", handled, err)
	}
}

func TestMainWorkerSubcommandsDispatchBeforeLifecyclePreflight(t *testing.T) {
	const childEnv = "DEVICE_STUDIO_MAIN_DISPATCH_CHILD"
	if mode := os.Getenv(childEnv); mode != "" {
		args := map[string][]string{
			"inspection": {"inspect-worker", "--bad"},
			"render":     {"render-worker", "--bad"},
			"video":      {"video-render-worker", "--bad"},
			"physical":   {"physical-render-worker", "--bad"},
		}
		os.Args = append([]string{"device-studio-api"}, args[mode]...)
		main()
		t.Fatal("main returned instead of exiting with the worker parser error")
	}

	for _, tc := range []struct {
		name string
		mode string
		want string
	}{
		{name: "inspection", mode: "inspection", want: "inspect-worker requires exactly --manifest"},
		{name: "render", mode: "render", want: "render-worker requires exactly --manifest"},
		{name: "video", mode: "video", want: "video-render-worker requires exactly --manifest"},
		{name: "physical", mode: "physical", want: "physical-render-worker requires exactly --manifest"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMainWorkerSubcommandsDispatchBeforeLifecyclePreflight$")
			cmd.Env = append(os.Environ(), childEnv+"="+tc.mode)
			output, err := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("worker dispatch subprocess exceeded timeout: %s", output)
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("subprocess exit = %v, output=%s", err, output)
			}
			if !strings.Contains(string(output), tc.want) {
				t.Fatalf("output %q does not contain parser error %q", output, tc.want)
			}
			if strings.Contains(string(output), "must be run through the Vrooli lifecycle system") {
				t.Fatalf("worker invocation reached lifecycle preflight: %s", output)
			}
		})
	}
}

func TestTestPoolInitializerInstallsRenderJobSchema(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	configureTestPoolSchemas(db)
	if err := db.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), "lease-render-schema", time.Minute); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRowContext(database.WithTestMode(ctx), "SELECT count(*) FROM render_jobs").Scan(&count); err != nil {
		t.Fatalf("leased render_jobs schema was not initialized: %v", err)
	}
}
