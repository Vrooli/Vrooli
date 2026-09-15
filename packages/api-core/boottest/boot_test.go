package boottest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/vrooli/api-core/storage"
)

func TestHealthRequiresExpectedJSONIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"healthy", `{"status":"healthy","service":"fixture-api"}`, true},
		{"other listener", `{"status":"healthy","service":"other-api"}`, false},
		{"misleading text", `{"status":"unhealthy","service":"fixture-api","note":"healthy"}`, false},
		{"invalid JSON", `"status" "healthy"`, false},
		{"trailing JSON", `{"status":"healthy","service":"fixture-api"}{}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, tc.body) }))
			defer s.Close()
			err := health(t.Context(), s.Client(), s.URL, "fixture-api")
			if (err == nil) != tc.valid {
				t.Fatalf("health error = %v; valid=%v", err, tc.valid)
			}
		})
	}
}

func TestHealthHonorsRequestDeadline(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer s.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	if err := health(ctx, s.Client(), s.URL, "fixture-api"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline, got %v", err)
	}
}

func TestStorageIsolatedFromInheritedLifecycle(t *testing.T) {
	root := t.TempDir()
	env := environment([]string{"API_PORT=123", "SCENARIO_DATA_DIR=/live", "VROOLI_STORAGE_ROOT=/live", "VROOLI_STORAGE_NAMESPACE=fixture", "SQLITE_PATH=/live/live.db"}, []string{"API_PORT=456", "VROOLI_STORAGE_ROOT=/override", "BRIDGE_BOOTSTRAP_SCRIPT=/bootstrap"}, root, 789)
	values := map[string]string{}
	for _, entry := range env {
		k, v, _ := strings.Cut(entry, "=")
		values[k] = v
	}
	path, err := storage.SQLitePath(storage.SQLiteConfig{Scenario: "fixture", EnvGet: func(k string) string { return values[k] }})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
		t.Fatalf("database escaped test root: %s", path)
	}
	if values["API_PORT"] != "789" || values["BRIDGE_BOOTSTRAP_SCRIPT"] != "/bootstrap" {
		t.Fatalf("lost owned environment: %v", values)
	}
}

func TestBinaryLifecycle(t *testing.T) {
	for _, tc := range []struct{ mode, want string }{
		{"clean", ""},
		{"early-exit", "API exited before health verification"},
		{"wrong-service", "service=\"other-api\""},
		{"bad-shutdown", "did not exit cleanly"},
		{"hang-shutdown", "API shutdown deadline"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			if runtime.GOOS == "windows" && (tc.mode == "bad-shutdown" || tc.mode == "hang-shutdown") {
				t.Skip("Unix signal contract")
			}
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			addr := ln.Addr().String()
			ln.Close()
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(binary, "-test.run=^TestBootProcess$")
			cmd.Env = append(os.Environ(), "BOOT_TEST_MODE="+tc.mode, "BOOT_TEST_ADDR="+addr)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
			defer cancel()
			err = check(ctx, cmd, "http://"+addr+"/health", "fixture-api", 2*time.Second, 2*time.Second)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "fixture diagnostic") {
				t.Fatalf("expected %q and child diagnostics; got %v", tc.want, err)
			}
			if cmd.ProcessState == nil {
				t.Fatal("child was not reaped")
			}
		})
	}
}

// TestBootProcess is a bounded subprocess fixture, never a scenario binary.
func TestBootProcess(t *testing.T) {
	mode := os.Getenv("BOOT_TEST_MODE")
	if mode == "" {
		return
	}
	fmt.Fprintln(os.Stderr, "fixture diagnostic")
	if mode == "early-exit" {
		os.Exit(7)
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)
	server := &http.Server{Addr: os.Getenv("BOOT_TEST_ADDR"), Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service := "fixture-api"
		if mode == "wrong-service" {
			service = "other-api"
		}
		fmt.Fprintf(w, `{"status":"healthy","service":%q}`, service)
	})}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			os.Exit(8)
		}
	}()
	<-sig
	if mode == "bad-shutdown" {
		os.Exit(3)
	}
	if mode == "hang-shutdown" {
		select {}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		os.Exit(9)
	}
	os.Exit(0)
}

func TestDiagnosticsKeepBoundedTail(t *testing.T) {
	var tail logTail
	fmt.Fprint(&tail, strings.Repeat("x", 40<<10))
	fmt.Fprint(&tail, "last failure")
	if got := tail.String(); len(got) != 32<<10 || !strings.HasSuffix(got, "last failure") {
		t.Fatalf("tail length=%d", len(got))
	}
}
