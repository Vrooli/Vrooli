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
	if target.Unsupported != "" || target.Executable != "" || manifest.ManagedService.Acquisition.EffectiveKind(target) != "url" || !binaryfetch.UsesOnlyBuildTimeFacts(target) {
		return Resolution{}, fmt.Errorf("%s has no pinned release on this platform", name)
	}
	if digest, err := hex.DecodeString(target.SHA256); err != nil || len(digest) != 32 {
		return Resolution{}, fmt.Errorf("%s source requires a valid SHA-256", name)
	}
	artifact, err := manifest.ManagedService.Artifact.ForCurrentPlatform()
	if err != nil {
		return Resolution{}, err
	}
	if artifact.SHA256 == "" {
		return Resolution{}, fmt.Errorf("%s requires the exact platform artifact digest", name)
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
	if stage.DownloadBytes <= 0 || stage.DownloadBytes > DiskJobPlanningCeilingBytes/3 {
		return DiskAdmissionPlan{Unsupported: "resource staging byte bound exceeds install capacity contract"}, nil
	}
	// Exact raw downloads stage and publish beside the canonical artifact.
	// Reserve the candidate, retained/pending bytes and metadata on that volume.
	plan := DiskAdmissionPlan{Manager: r.PackageManager, ProbeBytes: stage.DownloadBytes, Paths: []DiskPathRequirement{{Path: filepath.Dir(stage.Path), Role: "verified-resource-artifact", PeakBytes: 3*stage.DownloadBytes + (1 << 20)}}}
	for _, path := range stage.MetadataDirectories {
		if !filepath.IsAbs(path) {
			return DiskAdmissionPlan{}, fmt.Errorf("resource stage metadata path is not absolute")
		}
		plan.Paths = append(plan.Paths, DiskPathRequirement{Path: path, Role: "resource-stage-metadata", PeakBytes: 1 << 20})
	}
	return plan, nil
}
