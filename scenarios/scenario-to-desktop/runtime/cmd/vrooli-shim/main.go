// Command vrooli-shim is the small, platform-native `vrooli` entry point
// shipped inside a desktop bundle. It deliberately knows only how to locate
// the bundle runtime and forward requests to its authenticated control API;
// credential storage remains behind the supervisor boundary.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/config"
	runtimemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

type bundleInfo struct {
	App      runtimemanifest.App       `json:"app"`
	IPC      runtimemanifest.IPC       `json:"ipc"`
	Services []runtimemanifest.Service `json:"services"`
}

func main() {
	bundleRoot, err := locateBundleRoot()
	if err != nil {
		fail(err)
	}
	manifest, err := loadBundle(filepath.Join(bundleRoot, "bundle.json"))
	if err != nil {
		fail(err)
	}
	appData, err := resolveAppData(manifest.App.Name)
	if err != nil {
		fail(err)
	}
	port, err := resolvePort(appData, manifest.IPC.Port)
	if err != nil {
		fail(err)
	}
	tokenPath := manifest.IPC.AuthTokenRel
	if strings.TrimSpace(tokenPath) == "" {
		tokenPath = filepath.Join("runtime", "auth-token")
	}
	tokenPath = filepath.Join(appData, filepath.FromSlash(tokenPath))
	runtimectl, err := resolveRuntimeCtl(bundleRoot)
	if err != nil {
		fail(err)
	}

	args := os.Args[1:]
	if len(args) >= 3 && args[0] == "scenario" && args[1] == "port" {
		args, err = translateScenarioPort(manifest, args[2:])
		if err != nil {
			fail(err)
		}
	}
	if len(args) == 0 {
		fail(errors.New("usage: vrooli <command> [args...]"))
	}

	forwarded := []string{"--host", "127.0.0.1", "--port", strconv.Itoa(port), "--token-file", tokenPath}
	forwarded = append(forwarded, args...)
	// The shim intentionally forwards the user's CLI arguments to the
	// bundle-owned compiled runtime. exec.Command does not invoke a shell, and
	// runtimectl is resolved from the verified bundle root above.
	// #nosec G204 G702 -- this is direct execution of the bundle-owned runtimectl; no shell is involved.
	cmd := exec.Command(runtimectl, forwarded...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fail(err)
	}
}

func locateBundleRoot() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve shim executable: %w", err)
	}
	return filepath.Abs(filepath.Join(filepath.Dir(executable), ".."))
}

func loadBundle(path string) (*bundleInfo, error) {
	// The caller derives path from the shim's bundle root and appends the
	// fixed bundle manifest name; this is not an arbitrary user file import.
	// #nosec G304 -- the bundle manifest is fixed relative to the verified executable root.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bundle manifest: %w", err)
	}
	var manifest bundleInfo
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse bundle manifest: %w", err)
	}
	if strings.TrimSpace(manifest.App.Name) == "" {
		return nil, errors.New("bundle manifest has no app name")
	}
	return &manifest, nil
}

func resolveAppData(appName string) (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(base, config.SanitizeAppName(appName)), nil
}

func resolvePort(appData string, manifestPort int) (int, error) {
	// appData is the explicitly selected runtime state directory, and the
	// relative filename is fixed to the supervisor's IPC metadata file.
	// #nosec G304 -- this intentionally reads only the fixed IPC port file under appData.
	data, err := os.ReadFile(filepath.Join(appData, "runtime", "ipc_port"))
	if err == nil {
		port, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
		if parseErr == nil && port > 0 && port <= 65535 {
			return port, nil
		}
	}
	if manifestPort > 0 && manifestPort <= 65535 {
		return manifestPort, nil
	}
	return 0, fmt.Errorf("runtime IPC port is unavailable under %s", appData)
}

func resolveRuntimeCtl(bundleRoot string) (string, error) {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	runtimeDir := filepath.Join(bundleRoot, "runtime", runtime.GOOS+"-"+arch)
	name := "runtimectl"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(runtimeDir, name)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("runtimectl not found at %s: %w", path, err)
	}
	return path, nil
}

func translateScenarioPort(manifest *bundleInfo, args []string) ([]string, error) {
	if len(args) != 2 {
		return nil, errors.New("usage: vrooli scenario port <scenario> <API_PORT|UI_PORT>")
	}
	serviceType := map[string]string{"API_PORT": "api-binary", "UI_PORT": "ui-bundle"}[args[1]]
	if serviceType == "" {
		return nil, fmt.Errorf("unsupported port name: %s", args[1])
	}
	for _, service := range manifest.Services {
		if service.Type != serviceType || service.Ports == nil {
			continue
		}
		portName := "http"
		if len(service.Ports.Requested) > 0 && service.Ports.Requested[0].Name != "" {
			portName = service.Ports.Requested[0].Name
		}
		// Keep the scenario argument validated even though the desktop manifest
		// owns the actual service identity; it prevents accidental forwarding of
		// a URL-like value into the control API query string.
		if _, err := url.Parse(args[0]); err != nil {
			return nil, fmt.Errorf("invalid scenario name: %w", err)
		}
		return []string{"port", "--service", service.ID, "--port-name", portName}, nil
	}
	return nil, fmt.Errorf("no service for type %s", serviceType)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
