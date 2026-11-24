package installgateway

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type lockedSizerFixture struct {
	calls []frozenNPMPackage
}

func (f *lockedSizerFixture) Registry(context.Context, string, string, string) (artifactSize, error) {
	return artifactSize{}, nil
}

func (f *lockedSizerFixture) Tarball(context.Context, string, string) (artifactSize, error) {
	return artifactSize{}, nil
}

func (f *lockedSizerFixture) LockedRegistry(_ context.Context, name, version, resolved, integrity string) (artifactSize, error) {
	f.calls = append(f.calls, frozenNPMPackage{Name: name, Version: version, Resolved: resolved, Integrity: integrity})
	return artifactSize{Archive: 1000, Unpacked: 4000}, nil
}

func TestPlanNPMFrozenHostToolUsesExactLockAndFullWritePathMap(t *testing.T) {
	root := t.TempDir()
	manifest := `{"name":"fixture-tool-owner","version":"1.0.0","dependencies":{"fixture-tool":"1.2.3"}}`
	lock := `{"name":"fixture-tool-owner","version":"1.0.0","lockfileVersion":3,"packages":{"":{"name":"fixture-tool-owner","version":"1.0.0","dependencies":{"fixture-tool":"1.2.3"}},"node_modules/fixture-tool":{"version":"1.2.3","resolved":"https://registry.npmjs.org/fixture-tool/-/fixture-tool-1.2.3.tgz","integrity":"sha512-fixture"}}}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(lock), 0o600); err != nil {
		t.Fatal(err)
	}
	sizer := &lockedSizerFixture{}
	previous := newRegistrySizer
	newRegistrySizer = func() (registrySizer, error) { return sizer, nil }
	t.Cleanup(func() { newRegistrySizer = previous })

	plan, err := planNPMFrozenHostToolDisk(context.Background(), Resolution{
		HostTool: true, ExactVersion: true, SurfaceRoot: root, PackageName: "fixture-tool",
		PackageManager: "npm", Profile: InstallProfile{LifecycleMode: "scripts-disabled-by-default"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Unsupported != "" {
		t.Fatalf("plan refused: %s", plan.Unsupported)
	}
	if len(sizer.calls) != 1 || plan.ProbeBytes != 5000 || len(plan.Paths) != 3 {
		t.Fatalf("sizing result = calls %v, probe %d, paths %v", sizer.calls, plan.ProbeBytes, plan.Paths)
	}
	if plan.Paths[0].PeakBytes != 10000 || plan.Paths[1].PeakBytes != 5000 || plan.Paths[2].PeakBytes != 5000 {
		t.Fatalf("write path peaks = %+v", plan.Paths)
	}
	if len(plan.Notes) != 1 || !strings.HasPrefix(plan.Notes[0], "frozen-npm-host-tool-size-receipt=") {
		t.Fatalf("missing frozen closure receipt note: %v", plan.Notes)
	}
	var receipt frozenNPMHostToolReceipt
	if err := json.Unmarshal([]byte(strings.TrimPrefix(plan.Notes[0], "frozen-npm-host-tool-size-receipt=")), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Package != "fixture-tool" || receipt.PackageVersion != "1.2.3" || receipt.PackageCount != 1 || receipt.ProbeBytes != 5000 || !receipt.OptionalEntriesIncluded {
		t.Fatalf("receipt does not bind the exact package closure and estimate: %+v", receipt)
	}
}

func TestPlanNPMFrozenHostToolFailsClosedOnManifestLockDrift(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"fixture-tool-owner","version":"1.0.0","dependencies":{"fixture-tool":"1.2.3"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(`{"name":"fixture-tool-owner","version":"1.0.0","lockfileVersion":3,"packages":{"":{"name":"fixture-tool-owner","version":"1.0.0","dependencies":{"fixture-tool":"1.2.4"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := planNPMFrozenHostToolDisk(context.Background(), Resolution{
		HostTool: true, ExactVersion: true, SurfaceRoot: root, PackageName: "fixture-tool",
		PackageManager: "npm", Profile: InstallProfile{LifecycleMode: "scripts-disabled-by-default"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.Unsupported, "root disagree") || len(plan.Paths) != 0 {
		t.Fatalf("manifest/lock drift was not refused: %+v", plan)
	}
}
