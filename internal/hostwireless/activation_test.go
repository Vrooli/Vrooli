package hostwireless

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	mt7927source "github.com/vrooli/vrooli/internal/tools/mt7927-source"
)

func copiedActivationFixture(t *testing.T) (ActivationPlan, fstest.MapFS, fstest.MapFS) {
	t.Helper()
	const recovery = "7.0.0-29-generic"
	moduleTree := "/lib/modules/" + nobleABI + "/"
	plan := ActivationPlan{
		Profile: NobleBluetoothProfile, KernelRelease: nobleABI,
		BundleReceiptSHA256: digest([]byte("exact prepared receipt")),
		OverlayDirectory:    overlayDirectory(NobleBluetoothProfile),
		RecoveryCandidates:  []string{recovery}, RecoveryRelease: recovery,
		WiredInterfaces: []string{"enp1s0"}, WiredDrivers: map[string]string{"enp1s0": "atlantic"},
		NVIDIALoadedModules: []string{"nvidia", "nvidia_modeset"},
	}
	baseline, candidate := fstest.MapFS{}, fstest.MapFS{}
	for _, name := range []string{"btmtk", "btusb"} {
		original := moduleTree + "kernel/drivers/bluetooth/" + name + ".ko.zst"
		plan.Originals = append(plan.Originals, identityFixture(baseline, original, "stock "+name))
		plan.Candidates = append(plan.Candidates, identityFixture(candidate, name+".ko", "candidate "+name))
	}
	for _, name := range append(bootFiles(nobleABI), "/boot/vmlinuz-"+recovery, "/boot/initrd.img-"+recovery,
		"/proc/net/route", moduleTree+"kernel/drivers/net/ethernet/atlantic.ko.zst",
		moduleTree+"kernel/nvidia/nvidia.ko", moduleTree+"kernel/nvidia/nvidia-modeset.ko", moduleTree+"modules.dep") {
		plan.Preserved = append(plan.Preserved, identityFixture(baseline, name, "preserve "+name))
	}
	return plan, baseline, candidate
}

func identityFixture(tree fstest.MapFS, name, contents string) FileIdentity {
	data := []byte(contents)
	tree[strings.TrimPrefix(name, "/")] = &fstest.MapFile{Data: data}
	return FileIdentity{Path: name, SHA256: digest(data)}
}

func TestCopiedActivationStagesAndRestoresExactScope(t *testing.T) {
	ctx := context.Background()
	plan, baseline, candidate := copiedActivationFixture(t)
	tx, err := NewCopiedActivation(ctx, baseline, plan)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tx.Close(); err != nil {
			t.Error(err)
		}
	})
	assertAbsent := func() {
		t.Helper()
		if _, err := tx.root.Stat(plan.OverlayDirectory); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("overlay unexpectedly present: %v", err)
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := tx.Stage(cancelled, candidate); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled stage error %v", err)
	}
	assertAbsent()
	midCopy, stop := context.WithCancel(ctx)
	defer stop()
	if err := tx.Stage(midCopy, cancelDuringOpen{FS: candidate, target: "btusb.ko", cancel: stop}); !errors.Is(err, context.Canceled) {
		t.Fatalf("mid-copy cancellation result %v", err)
	}
	assertAbsent()
	badCandidate := fstest.MapFS{"btmtk.ko": candidate["btmtk.ko"], "btusb.ko": {Data: []byte("changed bytes")}}
	if err := tx.Stage(ctx, badCandidate); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("unexpected partial stage result %v", err)
	}
	assertAbsent()
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := tx.Stage(ctx, candidate); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range plan.Candidates {
		if err := verifyFile(ctx, tx.root.FS(), path.Join(plan.OverlayDirectory, file.Path), file.SHA256); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := tx.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
		assertAbsent()
	}
	if _, err := tx.root.Stat(path.Dir(plan.OverlayDirectory)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("transaction-created updates parent remains: %v", err)
	}
	for name, original := range baseline {
		got, err := tx.root.ReadFile(name)
		if err != nil || string(got) != string(original.Data) {
			t.Fatalf("changed protected file %s: %v", name, err)
		}
	}
}

type cancelDuringOpen struct {
	fs.FS
	target string
	cancel context.CancelFunc
}

func (f cancelDuringOpen) Open(name string) (fs.File, error) {
	if name == f.target {
		f.cancel()
	}
	return f.FS.Open(name)
}

func TestCopiedActivationRefusesForeignOverlayAndBootDrift(t *testing.T) {
	ctx := context.Background()
	plan, baseline, candidate := copiedActivationFixture(t)
	tx, err := NewCopiedActivation(ctx, baseline, plan)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Close() })
	if err := tx.Stage(ctx, candidate); err != nil {
		t.Fatal(err)
	}
	target := path.Join(plan.OverlayDirectory, "btusb.ko")
	if err := tx.root.WriteFile(target, []byte("foreign overlay"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(ctx); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("foreign overlay rollback: %v", err)
	}
	if got, err := tx.root.ReadFile(target); err != nil || string(got) != "foreign overlay" {
		t.Fatalf("foreign bytes changed: %s %v", got, err)
	}
	if err := tx.root.WriteFile(target, candidate["btusb.ko"].Data, 0600); err != nil {
		t.Fatal(err)
	}
	const boot = "boot/grub/grubenv"
	if err := tx.root.WriteFile(boot, []byte("drift"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tx.Stage(ctx, candidate); err == nil || !strings.Contains(err.Error(), "protected baseline changed") {
		t.Fatalf("baseline drift retry: %v", err)
	}
	if err := tx.Rollback(ctx); err == nil || !strings.Contains(err.Error(), "protected baseline changed") {
		t.Fatalf("baseline drift rollback: %v", err)
	}
	if _, err := tx.root.Stat(target); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("own exact overlay not removed during rollback: %v", err)
	}
}

func TestCopiedActivationRequiresIndependentBootAndProviders(t *testing.T) {
	ctx := context.Background()
	for _, missing := range []string{"/boot/initrd.img-7.0.0-29-generic", "/proc/net/route", "/lib/modules/" + nobleABI + "/kernel/nvidia/nvidia-modeset.ko"} {
		plan, baseline, _ := copiedActivationFixture(t)
		plan.Preserved = slices.DeleteFunc(plan.Preserved, func(file FileIdentity) bool { return file.Path == missing })
		tx, err := NewCopiedActivation(ctx, baseline, plan)
		if err == nil {
			_ = tx.Close()
			t.Fatalf("accepted missing protected identity %s", missing)
		}
	}
}

func TestCopiedActivationPreservesExistingUpdates(t *testing.T) {
	ctx := context.Background()
	plan, baseline, candidate := copiedActivationFixture(t)
	foreign := path.Join(path.Dir(plan.OverlayDirectory), "foreign.ko")
	plan.Preserved = append(plan.Preserved, identityFixture(baseline, foreign, "unrelated overlay"))
	tx, err := NewCopiedActivation(ctx, baseline, plan)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Close() })
	if err := tx.Stage(ctx, candidate); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if content, err := tx.root.ReadFile(foreign); err != nil || string(content) != "unrelated overlay" {
		t.Fatalf("existing updates changed: %q %v", content, err)
	}
}

func TestVerifyPreparedBundleRejectsChangedBytesAndMetadata(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	const generation = "abcdef"
	spec := preparationProfiles[NobleBluetoothProfile]
	stage := filepath.Join(root, "mt7927-noble-source", "pinned")
	temp := filepath.Join(root, "temporary")
	destination := filepath.Join(root, "host-wireless", NobleBluetoothProfile, generation)
	for _, dir := range []string{stage, temp, filepath.Dir(destination)} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	const source = "exact source manifest"
	writeFixture(t, filepath.Join(stage, mt7927source.Manifest), source)
	for _, name := range spec.modules {
		writeFixture(t, filepath.Join(temp, name+".ko"), name)
	}
	expected := Receipt{
		SchemaVersion: 1, Profile: NobleBluetoothProfile, KernelRelease: nobleABI, SourceDirectory: stage,
		SourceSHA256: digest([]byte(source)), PatchSHA256: digest(spec.patch), BuildRecipeSHA256: digest(spec.recipe),
		Modaliases: []string{"observed"}, CopiedTreeRollback: true, StockABI: ABIProof{StockExports: 14, Imports: 198},
		CandidateABI: ABIProof{StockExports: 14, Imports: 199},
	}
	if _, err := publishCandidate(ctx, expected, temp, destination); err != nil {
		t.Fatal(err)
	}
	files, receiptHash, err := verifyPreparedBundle(ctx, expected, spec, generation)
	if err != nil || len(files) != len(spec.modules) || !validSHA(receiptHash) {
		t.Fatalf("files %+v receipt %q error %v", files, receiptHash, err)
	}
	wrong := expected
	wrong.KernelRelease = "7.0.0-42-generic"
	if _, _, err := verifyPreparedBundle(ctx, wrong, spec, generation); err == nil {
		t.Fatal("accepted receipt for another kernel")
	}
	if _, _, err := verifyPreparedBundle(ctx, expected, spec, "../abcdef"); err == nil {
		t.Fatal("accepted generation traversal")
	}
	writeFixture(t, filepath.Join(destination, "btmtk.ko"), "changed")
	if _, _, err := verifyPreparedBundle(ctx, expected, spec, generation); err == nil {
		t.Fatal("accepted changed candidate bytes")
	}
}
