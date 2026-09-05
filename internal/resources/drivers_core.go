package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	runtimelogs "github.com/vrooli/vrooli/internal/resources/runtime/logs"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
)

const StatusCodeUnsupportedPlatform = "unsupported_platform"

const driversDockerUninstall = "uninstall"

type resourceDriver interface {
	Name() string
	Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error)
	Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error
}

var (
	lookPathCommandFn       = exec.LookPath
	runSourceBuildCommandFn = func(cmd *exec.Cmd) error { return cmd.Run() }
	gpuVerificationSleep    = sleepForGPUVerification
)

func sleepForGPUVerification(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func driverForManifest(manifest ResourceManifest) (resourceDriver, error) {
	switch manifest.Driver {
	case accelBridgeExternalCli:
		return externalCLIDriver{}, nil
	case "native-cli":
		return nativeCLIDriver{}, nil
	case accelBridgeManagedService:
		return managedServiceDriver{}, nil
	case "cloud-api":
		return cloudAPIDriver{}, nil
	default:
		return nil, fmt.Errorf("driver %q is not supported by the native resource control plane", manifest.Driver)
	}
}

func ensureSupportedPlatform(manifest ResourceManifest) error {
	support := manifest.Platforms.SupportForCurrentPlatform()
	if support == "" {
		return nil
	}
	if support == "unsupported" {
		return fmt.Errorf("resource %q is unsupported on %s", manifest.Name, manifestpkg.CurrentPlatform())
	}
	return nil
}

func statusRawWithCompanions(raw json.RawMessage, companions []CompanionStatus) json.RawMessage {
	payload := make(map[string]any)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &payload)
	}
	payload["companions"] = companions
	out, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return out
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func nextArgValue(args []string, flag string) string {
	for index, arg := range args {
		if arg == flag && index+1 < len(args) {
			return args[index+1]
		}
		if strings.HasPrefix(arg, flag+"=") {
			return strings.TrimPrefix(arg, flag+"=")
		}
	}
	return ""
}

func managedLogCandidates(controller *Controller, manifest ResourceManifest) []string {
	resolver, err := runtimestorage.NewResolver(runtimestorage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		return nil
	}
	paths, err := resolver.Resolve(runtimestorage.Options{ResourceID: manifest.Name})
	if err != nil {
		return nil
	}
	return runtimelogs.CandidatePaths(manifestpkg.ResourceManifest(manifest), paths)
}
