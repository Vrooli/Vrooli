package installgateway

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"

	mt7927source "github.com/vrooli/vrooli/internal/tools/mt7927-source"
)

// MT7927SourceLifecycle stages data in the artifact store.
const MT7927SourceLifecycle = "pinned-wireless-source"

func resolveMT7927Source(root, ecosystem, name, version string) (Resolution, error) {
	profile, ok := mt7927source.Lookup(name)
	if !ok {
		return Resolution{}, fmt.Errorf("unknown MT7927 source profile %q", name)
	}
	if ecosystem != "source" || version != profile.Version() {
		return Resolution{}, fmt.Errorf("MT7927 staging requires source/%s@%s", name, profile.Version())
	}
	manifest := profile.ManifestPath(root)
	if err := profile.VerifyManifest(manifest); err != nil {
		return Resolution{}, fmt.Errorf("verify MT7927 source profile: %w", err)
	}
	return Resolution{
		RepositoryRoot:        root,
		SurfaceRoot:           filepath.Dir(manifest),
		ManifestPath:          manifest,
		PackageName:           name,
		PackageManager:        "native-binaryfetch",
		ExactVersion:          true,
		HostToolTargetVersion: version,
		Argv:                  []string{"native-acquire", name, version},
		Profile: InstallProfile{
			Governance:      "scenario-dependency-analyzer",
			LifecycleMode:   MT7927SourceLifecycle,
			FrozenLockfile:  true,
			ScriptsDisabled: true,
		},
	}, nil
}

func validateMT7927Source(r Resolution) (mt7927source.Profile, error) {
	want, err := resolveMT7927Source(r.RepositoryRoot, "source", r.PackageName, r.HostToolTargetVersion)
	if err != nil {
		return mt7927source.Profile{}, err
	}
	if !reflect.DeepEqual(r, want) {
		return mt7927source.Profile{}, fmt.Errorf("MT7927 source staging resolution changed")
	}
	profile, _ := mt7927source.Lookup(r.PackageName)
	return profile, nil
}

func stageMT7927Source(ctx context.Context, r Resolution) (string, error) {
	profile, err := validateMT7927Source(r)
	if err != nil {
		return "", err
	}
	path, err := profile.Provision(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Verified %s staged at %s", profile.Description(), path), nil
}

func planMT7927SourceDisk(r Resolution) (DiskAdmissionPlan, error) {
	profile, err := validateMT7927Source(r)
	if err != nil {
		return DiskAdmissionPlan{}, err
	}
	path, peak, err := profile.StageBudget()
	if err != nil {
		return DiskAdmissionPlan{}, err
	}
	return DiskAdmissionPlan{
		Manager: r.PackageManager,
		Paths: []DiskPathRequirement{
			{Path: path, Role: "wireless-source-staging", PeakBytes: peak},
		},
	}, nil
}
