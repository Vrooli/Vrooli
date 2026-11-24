package installgateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/resources"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

// The closed resource owner surface admits exact committed source releases.
// It accepts no caller URL, path or service lifecycle action.
func resolveManagedServiceResource(root, name, ecosystem, pkg, version string) (Resolution, error) {
	if name != "cloudflared" && name != "openrgb" {
		return Resolution{}, fmt.Errorf("resource has no admitted managed artifact route")
	}
	path := filepath.Join(root, "resources", name, "resource.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return Resolution{}, err
	}
	manifest, err := manifestpkg.Parse(path, raw)
	if err != nil {
		return Resolution{}, err
	}
	if ecosystem != "source" || pkg != name || manifest.Name != name || manifest.Driver != "managed-service" || manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil || manifest.ManagedService.Artifact.Verification == "host-tool" || version == "" || version != manifest.ManagedService.Artifact.Version {
		return Resolution{}, fmt.Errorf("%s requires its exact resource-owned checksum-pinned source release", name)
	}
	target, err := manifest.ManagedService.Acquisition.Resolve(binaryfetch.Facts{"os": runtime.GOOS, "arch": runtime.GOARCH})
	if err != nil {
		return Resolution{}, err
	}
	kind := manifest.ManagedService.Acquisition.EffectiveKind(target)
	if target.Unsupported != "" || target.Executable != "" || (kind != "url" && kind != "composed") || !binaryfetch.UsesOnlyBuildTimeFacts(target) {
		return Resolution{}, fmt.Errorf("%s has no pinned release on this platform", name)
	}
	artifactDigest := target.SHA256
	if kind == "composed" {
		if target.Auth != nil {
			return Resolution{}, fmt.Errorf("%s composed release cannot use acquisition credentials", name)
		}
		if _, err := binaryfetch.PlanCompositionCapacity(target.Compose); err != nil {
			return Resolution{}, fmt.Errorf("%s composed release has no complete capacity bound: %w", name, err)
		}
		artifactDigest = target.ArtifactSHA256
	}
	if digest, err := hex.DecodeString(artifactDigest); err != nil || len(digest) != 32 {
		return Resolution{}, fmt.Errorf("%s source requires a valid SHA-256", name)
	}
	artifact, err := manifest.ManagedService.Artifact.ForCurrentPlatform()
	if err != nil {
		return Resolution{}, err
	}
	if artifact.SHA256 == "" {
		return Resolution{}, fmt.Errorf("%s requires the exact platform artifact digest", name)
	}
	if kind == "composed" && artifact.SHA256 != artifactDigest {
		return Resolution{}, fmt.Errorf("%s composed artifact digest differs from the exact platform artifact", name)
	}
	return Resolution{RepositoryRoot: root, SurfaceRoot: filepath.Dir(path), ManifestPath: path, PackageManager: "native-binaryfetch", PackageName: pkg, HostToolTargetVersion: version, HostManifestSHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), ExactVersion: true, Argv: []string{"native-acquire", pkg, version}, Profile: InstallProfile{Governance: "scenario-dependency-analyzer", LifecycleMode: "managed-service-artifact", FrozenLockfile: true, ScriptsDisabled: true}}, nil
}

func provisionManagedServiceArtifact(ctx context.Context, r Resolution) (string, error) {
	want, err := resolveManagedServiceResource(r.RepositoryRoot, r.PackageName, "source", r.PackageName, r.HostToolTargetVersion)
	if err != nil {
		return "", err
	}
	// Every admitted field is frozen, including the complete resource manifest.
	a, _ := json.Marshal(r)
	b, _ := json.Marshal(want)
	if string(a) != string(b) {
		return "", fmt.Errorf("managed-service artifact resolution changed before acquisition")
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", err
	}
	info, err := resources.NewController(r.RepositoryRoot, home).StageManagedServiceArtifactSnapshot(ctx, r.PackageName, r.HostToolTargetVersion, r.HostManifestSHA256)
	if err != nil {
		return "", err
	}
	return "Staged verified " + r.PackageName + " " + info.Version + " at " + info.Path, nil
}

func planManagedServiceArtifactDisk(ctx context.Context, r Resolution) (DiskAdmissionPlan, error) {
	want, err := resolveManagedServiceResource(r.RepositoryRoot, r.PackageName, "source", r.PackageName, r.HostToolTargetVersion)
	if err != nil {
		return DiskAdmissionPlan{Unsupported: err.Error()}, nil
	}
	a, _ := json.Marshal(r)
	b, _ := json.Marshal(want)
	if string(a) != string(b) {
		return DiskAdmissionPlan{Unsupported: "managed-service artifact resolution changed before disk admission"}, nil
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return DiskAdmissionPlan{}, err
	}
	stage, err := resources.NewController(r.RepositoryRoot, home).PlanManagedServiceArtifactStage(ctx, r.PackageName, r.HostToolTargetVersion, r.HostManifestSHA256)
	if err != nil {
		return DiskAdmissionPlan{Unsupported: err.Error()}, nil
	}
	if stage.DownloadBytes <= 0 || stage.PeakBytes <= 0 || stage.PeakBytes > DiskJobPlanningCeilingBytes {
		return DiskAdmissionPlan{Unsupported: "resource staging byte bound exceeds install capacity contract"}, nil
	}
	// The resource owner accounts conservatively for download, extracted tree,
	// retained/pending artifact copies, and metadata. Admission applies the
	// canonical per-job ceiling before acquisition starts.
	plan := DiskAdmissionPlan{Manager: r.PackageManager, ProbeBytes: stage.DownloadBytes, Paths: []DiskPathRequirement{{Path: filepath.Dir(stage.Path), Role: "verified-resource-artifact", PeakBytes: stage.PeakBytes}}}
	for _, path := range stage.MetadataDirectories {
		if !filepath.IsAbs(path) {
			return DiskAdmissionPlan{}, fmt.Errorf("resource stage metadata path is not absolute")
		}
		plan.Paths = append(plan.Paths, DiskPathRequirement{Path: path, Role: "resource-stage-metadata", PeakBytes: 1 << 20})
	}
	return plan, nil
}
