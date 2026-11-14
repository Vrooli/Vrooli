package runtimeapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	platform "github.com/vrooli/platform-go"

	"github.com/vrooli/cliresolve"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/shell"
	dependencyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	"google.golang.org/protobuf/encoding/protojson"
)

// lightingManagedDependencyArgs is intentionally a fixed owner/operation. The
// version is rechecked against the governed resource declaration before use.
// The SDA owner keeps the exact1.1 profile immutable (version and digest); a
// different artifact requires a separately admitted profile, never same-version
// replacement. Setup checks the selected digest before and after acquisition.
func lightingManagedDependencyArgs(version string) []string {
	return []string{"deps", "install", "source/openrgb@" + version, "--scenario", "openrgb", "--surface", "host-tool", "--exact", "--apply", "--json"}
}

type lightingBoundedOutput struct {
	data     []byte
	overflow bool
}

func (b *lightingBoundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := 65536 - len(b.data)
	if len(p) > remaining {
		p = p[:remaining]
		b.overflow = true
	}
	b.data = append(b.data, p...)
	return n, nil
}

func acquireManagedLighting(ctx context.Context, controller *resources.Controller, home, version string) (resources.ManagedServiceArtifact, error) {
	empty := resources.ManagedServiceArtifact{}
	nativeRoot := false
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		security, e := platform.ProcessSecurity(os.Getpid())
		if e != nil {
			return empty, e
		}
		nativeRoot = security.Elevated
	}
	return acquireManagedLightingWith(ctx, controller, home, version, lightingAcquisitionEffects{nativeRoot: nativeRoot, userInstall: installGovernedLightingAsOperator})
}

type lightingAcquisitionEffects struct {
	nativeRoot  bool
	userInstall func(context.Context, *resources.Controller, string, string) error
}

func acquireManagedLightingWith(ctx context.Context, controller *resources.Controller, home, version string, e lightingAcquisitionEffects) (resources.ManagedServiceArtifact, error) {
	empty := resources.ManagedServiceArtifact{}
	manifest, err := controller.ResourceManifest(openRGBResource)
	if err != nil {
		return empty, err
	}
	if manifest.ManagedService == nil || version == "" || manifest.ManagedService.Artifact.Version != version {
		return empty, errors.New("selected OpenRGB release changed before governed installation")
	}
	if e.nativeRoot {
		return empty, errors.New("acquire the pinned OpenRGB software with ordinary operator lighting setup before running elevated runtime lighting prepare; privileged setup never executes the user-owned dependency CLI")
	} else {
		err = e.userInstall(ctx, controller, home, version)
	}
	if err != nil {
		return empty, err
	}
	artifact, err := controller.InspectManagedServiceArtifact(ctx, openRGBResource)
	if err != nil {
		return empty, err
	}
	if artifact.Version != version {
		return empty, errors.New("installed OpenRGB release changed before verification")
	}
	return artifact, nil
}

// Acquisition belongs to the ordinary operator; native privileged publication
// never executes this user-owned dependency CLI.
func installGovernedLightingAsOperator(ctx context.Context, controller *resources.Controller, home, version string) error {
	binary, err := cliresolve.New(home).Executable("scenario-dependency-analyzer")
	if err != nil {
		return fmt.Errorf("governed dependency installation service is unavailable: %w", err)
	}
	operation, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	stdout, stderr := &lightingBoundedOutput{}, &lightingBoundedOutput{}
	cmd := shell.Command(shell.Spec{Context: operation, Name: binary, Args: lightingManagedDependencyArgs(version), Stdout: stdout, Stderr: stderr})
	cmd.WaitDelay = 5 * time.Second
	if err := cmd.Run(); err != nil {
		if operation.Err() != nil {
			return operation.Err()
		}
		return fmt.Errorf("governed OpenRGB installation failed: %w: %s", err, string(stderr.data))
	}
	if stdout.overflow {
		return errors.New("governed installation response exceeds size limit")
	}
	receipt := &dependencyv1.InstallDependencyResponse{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(stdout.data, receipt); err != nil {
		return fmt.Errorf("invalid governed installation receipt: %w", err)
	}
	if receipt.Blocked || !receipt.Installed || receipt.DryRun {
		message := receipt.Message
		if len(message) > 1024 {
			message = message[:1024]
		}
		return fmt.Errorf("governed OpenRGB installation did not admit and install the selected release (%s): %s", receipt.Verdict, message)
	}

	return nil
}
