package control

import (
	"context"
	"fmt"
	"strings"

	"device-control/internal/execution"
	"device-control/strategy"
)

func (s *Service) installVerifiedAndroidArtifact(ctx context.Context, deviceID string, lease Session, adapter strategy.Strategy, step Step) (*execution.ArtifactInstallProof, error) {
	artifactRef, _ := step.Arguments["artifact_ref"].(string)
	packageName, _ := step.Arguments["package"].(string)
	versionName, _ := step.Arguments["version_name"].(string)
	proof := &execution.ArtifactInstallProof{ImmutableRef: artifactRef, PackageName: packageName}
	if adapter.ID() != "android-adb" {
		return proof, fmt.Errorf("verified Android artifact install requires the android-adb strategy")
	}
	s.mu.Lock()
	resolver := s.androidArtifactResolver
	s.mu.Unlock()
	if resolver == nil {
		return proof, fmt.Errorf("verified Android artifact producer is unavailable")
	}
	staged, err := resolver.Resolve(ctx, artifactRef, packageName, versionName)
	if err != nil {
		return proof, err
	}
	defer staged.Close()
	proof.ArtifactChecksum = staged.Descriptor.Checksum
	proof.APKChecksum = staged.Descriptor.APKChecksum
	proof.PackageName = staged.Descriptor.PackageName
	proof.VersionName = staged.Descriptor.VersionName
	proof.VersionCode = staged.Descriptor.VersionCode
	if err := ctx.Err(); err != nil {
		return proof, err
	}
	if _, err := s.sessionForLease(ctx, deviceID, lease.LeaseToken); err != nil {
		return proof, fmt.Errorf("device lease is no longer valid after artifact resolution: %w", err)
	}
	actuator, ok := adapter.(strategy.InputActuator)
	if !ok {
		return proof, &strategy.UnsupportedCapabilityError{Capability: strategy.CapAppLifecycle, Operation: "install-artifact"}
	}
	if err := actuator.Actuate(ctx, strategy.Actuation{Action: "install", Package: staged.Descriptor.PackageName, Value: staged.Path}); err != nil {
		return proof, err
	}
	proof.InstallAccepted = true
	lifecycle, ok := adapter.(strategy.AppLifecycle)
	if !ok {
		return proof, &strategy.UnsupportedCapabilityError{Capability: strategy.CapAppLifecycle, Operation: "package-state"}
	}
	state, err := lifecycle.AppLifecycle(ctx, strategy.AppLifecycleRequest{Operation: "package-state", Package: staged.Descriptor.PackageName})
	if err != nil {
		return proof, fmt.Errorf("verify installed Android package: %w", err)
	}
	proof.Installed = state.Status == strategy.StatusAvailable && state.Installed && state.Package == staged.Descriptor.PackageName
	proof.InstalledVersion = strings.TrimSpace(state.Version)
	if !proof.Installed {
		return proof, fmt.Errorf("post-install package-state did not verify %s as installed", staged.Descriptor.PackageName)
	}
	if proof.InstalledVersion == "" || proof.InstalledVersion != staged.Descriptor.VersionName {
		return proof, fmt.Errorf("installed package version %q does not match artifact version %q", proof.InstalledVersion, staged.Descriptor.VersionName)
	}
	proof.PostInstallVerified = true
	return proof, nil
}
