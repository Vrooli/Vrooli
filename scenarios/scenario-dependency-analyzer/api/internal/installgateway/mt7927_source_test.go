package installgateway

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	mt7927source "github.com/vrooli/vrooli/internal/tools/mt7927-source"
)

func TestMT7927SourceResolvesOnlyCommittedProfileAndBoundsItsWrites(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	const surface = "host-tool"
	name, version := mt7927source.Name, mt7927source.Version
	r, err := Resolve(root, name, surface, "source", name, version)
	if err != nil {
		t.Fatal(err)
	}
	if r.Profile.LifecycleMode != MT7927SourceLifecycle || !r.Profile.ScriptsDisabled || !r.Profile.FrozenLockfile || !r.ExactVersion {
		t.Fatalf("unfrozen acquisition profile: %+v", r)
	}
	if err := ApplyExactVersion(&r, version); err != nil {
		t.Fatal(err)
	}
	if err := ApplyExactVersion(&r, "main"); err == nil {
		t.Fatal("moving source ref accepted")
	}
	for _, bad := range []struct{ owner, ecosystem, name, version string }{
		{name, "source", name, "main"},
		{name, "source", "other", version},
		{"other", "source", name, version},
		{name, "system", name, version},
	} {
		if _, err := Resolve(root, bad.owner, surface, bad.ecosystem, bad.name, bad.version); err == nil {
			t.Fatalf("invalid request accepted: %+v", bad)
		}
	}
	plan, err := PlanDiskAdmission(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Unsupported != "" || len(plan.Paths) != 1 || filepath.Base(plan.Paths[0].Path) != name || plan.Paths[0].PeakBytes < 164389251 {
		t.Fatalf("incomplete disk plan: %+v", plan)
	}

	r.Argv[1] = "other"
	if _, err := (ExecInstaller{}).Install(context.Background(), r); err == nil {
		t.Fatal("changed acquisition plan accepted")
	}
}

func TestMT7927SourceRefusesStaleGatewayManifest(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "tools", mt7927source.Name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, mt7927source.Manifest), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, mt7927source.Name, "host-tool", "source", mt7927source.Name, mt7927source.Version); err == nil {
		t.Fatal("divergent on-disk manifest accepted")
	}
}

func TestMT7927NobleSourceHasSeparateGovernedIdentityAndWritePath(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	name, version := mt7927source.NobleName, mt7927source.NobleVersion
	r, err := Resolve(root, name, "host-tool", "source", name, version)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyExactVersion(&r, version); err != nil {
		t.Fatal(err)
	}
	if r.PackageName != name || r.Profile.LifecycleMode != MT7927SourceLifecycle ||
		!r.Profile.ScriptsDisabled || filepath.Base(r.ManifestPath) != "noble-source-profile.json" {
		t.Fatalf("unexpected Noble source resolution: %+v", r)
	}
	for _, bad := range []struct{ owner, packageName, version string }{
		{mt7927source.Name, name, version},
		{name, mt7927source.Name, mt7927source.Version},
		{name, name, mt7927source.Version},
	} {
		if _, err := Resolve(root, bad.owner, "host-tool", "source", bad.packageName, bad.version); err == nil {
			t.Fatalf("cross-profile request accepted: %+v", bad)
		}
	}
	plan, err := PlanDiskAdmission(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	const sourceBytes = 8259 + 254937830 + 1988323 + 74504
	if plan.Unsupported != "" || len(plan.Paths) != 1 ||
		filepath.Base(plan.Paths[0].Path) != name || plan.Paths[0].PeakBytes != sourceBytes+(1<<20) {
		t.Fatalf("unexpected Noble disk admission plan: %+v", plan)
	}
}

func TestASUSBluetoothArchiveHasClosedLocalStagingIdentity(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	name, version := mt7927source.ASUSBluetoothName, mt7927source.ASUSBluetoothVersion
	r, err := Resolve(root, name, "host-tool", "source", name, version)
	if err != nil {
		t.Fatal(err)
	}
	if r.PackageName != name || !r.Profile.ScriptsDisabled || !r.ExactVersion ||
		filepath.Base(r.ManifestPath) != "asus-bluetooth-source-profile.json" {
		t.Fatalf("unexpected ASUS staging resolution %+v", r)
	}
	for _, bad := range []struct{ owner, pkg, version string }{
		{mt7927source.NobleName, name, version}, {name, name, "latest"}, {name, mt7927source.NobleName, mt7927source.NobleVersion},
	} {
		if _, err := Resolve(root, bad.owner, "host-tool", "source", bad.pkg, bad.version); err == nil {
			t.Fatalf("accepted cross-profile ASUS request %+v", bad)
		}
	}
	plan, err := PlanDiskAdmission(context.Background(), r)
	if err != nil || plan.Unsupported != "" || len(plan.Paths) != 1 || plan.Paths[0].PeakBytes != 7142068+(1<<20) ||
		filepath.Base(plan.Paths[0].Path) != name {
		t.Fatalf("ASUS staging budget %+v error %v", plan, err)
	}
}
