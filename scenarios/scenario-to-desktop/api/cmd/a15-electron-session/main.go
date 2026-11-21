// a15-electron-session is an opt-in helper for the hermetic BAS consent
// integration. It keeps target.StartElectronSession ownership in the S2D
// module while an outer Gateway fixture drives the packaged renderer over its
// authenticated loopback CDP endpoint.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"scenario-to-desktop-api/target"
)

type ready struct {
	Endpoint   string `json:"endpoint"`
	PID        int    `json:"pid"`
	Artifact   string `json:"artifact"`
	Digest     string `json:"digest"`
	Renderer   string `json:"renderer"`
	LPBSOrigin string `json:"lpbs_origin"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	artifact := strings.TrimSpace(os.Getenv("A15_APPIMAGE_PATH"))
	renderer := strings.TrimSpace(os.Getenv("A15_RENDERER_URL"))
	apiURL := strings.TrimSpace(os.Getenv("A15_API_URL"))
	lpbsURL := strings.TrimSpace(os.Getenv("A15_LPBS_URL"))
	for key, value := range map[string]string{"A15_APPIMAGE_PATH": artifact, "A15_RENDERER_URL": renderer, "A15_API_URL": apiURL, "A15_LPBS_URL": lpbsURL} {
		if value == "" {
			return fmt.Errorf("%s is required", key)
		}
	}
	file, err := os.Open(artifact)
	if err != nil {
		return fmt.Errorf("open exact AppImage: %w", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		_ = file.Close()
		return fmt.Errorf("hash AppImage: %w", err)
	}
	_ = file.Close()
	digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if expected := strings.TrimSpace(os.Getenv("A15_ARTIFACT_DIGEST")); expected != "" && expected != digest {
		return fmt.Errorf("AppImage digest mismatch: got %s, expected %s", digest, expected)
	}
	profile, err := os.MkdirTemp("", "a15-packaged-bas-")
	if err != nil {
		return fmt.Errorf("create isolated Electron profile root: %w", err)
	}
	defer os.RemoveAll(profile)
	pathValue := os.Getenv("PATH")
	if xdgDir := strings.TrimSpace(os.Getenv("A15_XDG_OPEN_DIR")); xdgDir != "" {
		pathValue = xdgDir + string(os.PathListSeparator) + pathValue
	}
	launch := target.ElectronLaunchOptions{
		UserDataDir: filepath.Join(profile, "user-data"), ContextID: "a15-packaged-consent",
		ScenarioName: "browser-automation-studio", ArtifactDigest: digest,
		TargetID: "local-linux-amd64", JourneyID: "A15-packaged-consent",
		ProfileID: "isolated-auth-fixture", IsolationLeaseID: "a15-packaged-consent-lease",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	session, err := target.StartElectronSession(ctx, target.ElectronSessionOptions{
		ArtifactPath: artifact, Launch: launch, StartupTimeout: 90 * time.Second,
		Renderer: target.RendererExpectation{URLPrefix: renderer},
		Environment: map[string]string{
			"HOME": profile, "USERPROFILE": profile,
			"PATH": pathValue,
			"XDG_CONFIG_HOME":                filepath.Join(profile, ".config"),
			"XDG_DATA_HOME":                  filepath.Join(profile, ".local", "share"),
			"XDG_STATE_HOME":                 filepath.Join(profile, ".local", "state"),
			"XDG_CACHE_HOME":                 filepath.Join(profile, ".cache"),
			"VROOLI_VALIDATION_RENDERER_URL": renderer,
			"VROOLI_VALIDATION_API_URL":      apiURL,
			"VROOLI_VALIDATION_LPBS_URL":     lpbsURL,
		},
	})
	if err != nil {
		return fmt.Errorf("start exact packaged BAS artifact: %w", err)
	}
	defer func() { _ = session.Close(context.Background()) }()
	if err := json.NewEncoder(os.Stdout).Encode(ready{
		Endpoint: session.Endpoint(), PID: session.PID(), Artifact: artifact,
		Digest: digest, Renderer: session.Target().GetRendererUrl(), LPBSOrigin: lpbsURL,
	}); err != nil {
		return fmt.Errorf("write session readiness: %w", err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	exited := make(chan error, 1)
	go func() { exited <- session.WaitForExit(ctx) }()
	select {
	case <-signals:
		return nil
	case err := <-exited:
		if err != nil {
			return fmt.Errorf("packaged BAS exited before consent journey cleanup: %w", err)
		}
		return fmt.Errorf("packaged BAS exited before consent journey cleanup")
	}
}
