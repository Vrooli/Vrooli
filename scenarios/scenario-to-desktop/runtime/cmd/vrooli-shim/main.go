// Command vrooli-shim is the small, platform-native `vrooli` entry point
// shipped inside a desktop bundle. It deliberately knows only how to locate
// the bundle runtime and forward requests to its authenticated control API;
// credential storage remains behind the supervisor boundary.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/config"
	runtimemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/peerlookup"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/toolprofiles"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type bundleInfo = runtimemanifest.Manifest

func main() {
	bundleRoot, err := locateBundleRoot()
	if err != nil {
		fail(err)
	}
	manifest, err := loadBundle(filepath.Join(bundleRoot, "bundle.json"))
	if err != nil {
		fail(err)
	}
	if handled, err := runOfflineToolProfile(os.Args[1:], bundleRoot, manifest, os.Stdout); handled {
		if err != nil {
			fail(err)
		}
		return
	}
	if handled, err := runPeerScenarioPort(os.Args[1:], manifest, os.Stdout); handled {
		if err != nil {
			fail(err)
		}
		return
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
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return "", fmt.Errorf("resolve shim executable path: %w", err)
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
	if err := runtimemanifest.ValidateToolProfiles(manifest.ToolProfiles); err != nil {
		return nil, fmt.Errorf("validate bundle tool profiles: %w", err)
	}
	return &manifest, nil
}

func runOfflineToolProfile(args []string, bundleRoot string, bundle *bundleInfo, output io.Writer) (bool, error) {
	return runOfflineToolProfileWithResolver(args, bundleRoot, bundle, output, toolprofiles.Resolve)
}

const rendererProfileQueryTimeout = 20 * time.Second

func runOfflineToolProfileWithResolver(args []string, bundleRoot string, bundle *bundleInfo, output io.Writer, resolve func(context.Context, string, string, []runtimemanifest.ToolProfilePackage) (proto.Message, error)) (bool, error) {
	if len(args) == 0 || args[0] != "host" {
		return false, nil
	}
	if len(args) < 2 || args[1] != "renderer-profile" {
		return false, nil
	}
	if len(args) != 4 || args[3] != "--json" {
		return true, errors.New("usage: vrooli host renderer-profile <blender|ffmpeg-video> --json")
	}
	ctx, cancel := context.WithTimeout(context.Background(), rendererProfileQueryTimeout)
	defer cancel()
	profile, err := resolve(ctx, bundleRoot, args[2], bundle.ToolProfiles)
	if err != nil {
		return true, err
	}
	data, err := protojson.Marshal(profile)
	if err != nil {
		return true, fmt.Errorf("encode renderer profile: %w", err)
	}
	if _, err := output.Write(append(data, '\n')); err != nil {
		return true, fmt.Errorf("write renderer profile: %w", err)
	}
	return true, nil
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

func translateScenarioPort(bundle *bundleInfo, args []string) ([]string, error) {
	if len(args) != 2 {
		return nil, errors.New("usage: vrooli scenario port <scenario> <API_PORT|UI_PORT>")
	}
	service, port, err := peerlookup.Local(bundle, args[0], args[1])
	if err != nil {
		return nil, err
	}
	return []string{"port", "--service", service, "--port-name", port}, nil
}

func runPeerScenarioPort(args []string, bundle *bundleInfo, output io.Writer) (bool, error) {
	if len(args) < 2 || args[0] != "scenario" || args[1] != "port" {
		return false, nil
	}
	if len(args) < 4 || len(args) > 5 {
		return true, errors.New("usage: vrooli scenario port <scenario> <API_PORT|UI_PORT> [--json]")
	}
	scenario, portName := strings.TrimSpace(args[2]), strings.TrimSpace(args[3])
	jsonOutput := len(args) == 5 && args[4] == "--json"
	if len(args) == 5 && !jsonOutput {
		return true, errors.New("usage: vrooli scenario port <scenario> <API_PORT|UI_PORT> [--json]")
	}
	if !peerlookup.ValidScenarioName(scenario) {
		return true, errors.New("invalid scenario name")
	}
	if portName != "API_PORT" && portName != "UI_PORT" {
		return true, fmt.Errorf("unsupported port name: %s", portName)
	}
	if bundle != nil && scenario == bundle.App.Scenario {
		return false, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return true, err
	}
	port, err := peerlookup.External(bundle, home, scenario, portName)
	if err == nil && port > 0 {
		if jsonOutput {
			return true, writeScenarioPortJSON(output, &cliv1.ScenarioPortSingle{Success: true, Scenario: scenario, PortName: portName, Port: int32(port)})
		}
		_, err = fmt.Fprintln(output, port)
		return true, err
	}
	if peerlookup.HasEmbedded(bundle, scenario) {
		return false, nil
	}
	message := fmt.Sprintf("no running runtime ports found for scenario %q", scenario)
	if jsonOutput {
		return true, writeScenarioPortJSON(output, &cliv1.ScenarioPortSingle{Success: false, Scenario: scenario, PortName: portName, Error: message})
	}
	return true, errors.New(message)
}

func writeScenarioPortJSON(output io.Writer, response *cliv1.ScenarioPortSingle) error {
	data, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(response)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, string(data))
	return err
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
