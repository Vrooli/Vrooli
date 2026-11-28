package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestNewAppConfiguresResourceApp(t *testing.T) {
	app, err := newApp()
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	if app == nil {
		t.Fatal("newApp() returned nil app")
	}
	if app.CLI == nil {
		t.Fatal("newApp() returned nil CLI")
	}
	if app.StaleChecker == nil {
		t.Fatal("newApp() returned nil stale checker")
	}
	if app.StaleChecker.SourceContextPath != ".." {
		t.Fatalf("SourceContextPath = %q, want %q", app.StaleChecker.SourceContextPath, "..")
	}
	if app.StaleChecker.ManifestSourcePath != "resource.json" {
		t.Fatalf("ManifestSourcePath = %q, want %q", app.StaleChecker.ManifestSourcePath, "resource.json")
	}
	if len(app.StaleChecker.FreshnessInputs) < 3 {
		t.Fatalf("FreshnessInputs len = %d, want at least the 3 required inputs", len(app.StaleChecker.FreshnessInputs))
	}
	if got, want := app.StaleChecker.FreshnessInputs[0], "cli/**"; got != want {
		t.Fatalf("FreshnessInputs[0] = %q, want %q", got, want)
	}
	if got, want := app.StaleChecker.FreshnessInputs[1], "resource.json"; got != want {
		t.Fatalf("FreshnessInputs[1] = %q, want %q", got, want)
	}
	if got, want := app.StaleChecker.FreshnessInputs[2], "../../packages/cli-core"; got != want {
		t.Fatalf("FreshnessInputs[2] = %q, want %q", got, want)
	}
}

func TestPolicyCommandRegistered(t *testing.T) {
	app, err := newApp()
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	output := captureStdout(t, func() {
		if err := app.CLI.Run([]string{"help"}); err != nil {
			t.Fatalf("help: %v", err)
		}
	})
	if !strings.Contains(output, "policy") {
		t.Fatalf("help output missing policy command:\n%s", output)
	}
}

func TestAuthorizeOllamaWithWakeRequiresInstalledAllowedArtifactBeforeStart(t *testing.T) {
	tests := []struct {
		name         string
		diskErr      error
		authorizeErr error
		wantWake     bool
		wantErr      bool
	}{
		{name: "uninstalled request", diskErr: errors.New("manifest missing"), wantErr: true},
		{name: "denied installed request", authorizeErr: errors.New("rights refused"), wantErr: true},
		{name: "eligible installed request", wantWake: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			liveCalls := 0
			wakeCalls := 0
			authorizeCalls := 0
			err := authorizeOllamaWithWake(context.Background(), "model:8b",
				func(context.Context, string) (string, error) {
					liveCalls++
					if liveCalls == 1 || tt.wantErr {
						return "", errors.New("Ollama unavailable")
					}
					return "sha256:live", nil
				},
				func(string) (string, error) {
					return "sha256:disk", tt.diskErr
				},
				func(_, digest string) error {
					authorizeCalls++
					if digest != "sha256:disk" && digest != "sha256:live" {
						t.Fatalf("authorized digest = %q", digest)
					}
					return tt.authorizeErr
				},
				func(context.Context) error {
					wakeCalls++
					return nil
				})
			if (err != nil) != tt.wantErr {
				t.Fatalf("authorizeOllamaWithWake error = %v, wantErr=%t", err, tt.wantErr)
			}
			if (wakeCalls == 1) != tt.wantWake {
				t.Fatalf("managed wake calls = %d, wantWake=%t", wakeCalls, tt.wantWake)
			}
			if tt.diskErr != nil && authorizeCalls != 0 {
				t.Fatalf("authorization ran for an uninstalled model: %d calls", authorizeCalls)
			}
			if tt.authorizeErr != nil && wakeCalls != 0 {
				t.Fatalf("managed wake ran before rights refusal: %d calls", wakeCalls)
			}
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return buf.String()
}
