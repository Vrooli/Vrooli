package resources

import (
	"context"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/binaryfetch"
	platform "github.com/vrooli/platform-go"
)

func testOpenRGBAppDir(t *testing.T) (string, string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "AppDir")
	for _, dir := range []string{"usr/bin", "usr/lib"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, entry := range []struct{ path, content string }{{openRGBAppDirPayload, "synthetic ELF"}, {"usr/lib/library.so", "synthetic library"}} {
		if err := os.WriteFile(filepath.Join(root, entry.path), []byte(entry.content), 0o555); err != nil {
			t.Fatal(err)
		}
	}
	tree, err := binaryfetch.TreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	return root, tree, updateTestDigest("synthetic ELF")
}

func TestOpenRGBAppDirVerificationBindsTheCompleteClosure(t *testing.T) {
	ctx := context.Background()
	root, tree, payloadSHA := testOpenRGBAppDir(t)
	seal := func(context.Context, string) error { return nil } // This fixture exercises content proof only.
	if _, err := verifyOpenRGBAppDir(ctx, root, tree, payloadSHA, seal); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root, "usr/lib/library.so"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "usr/lib/library.so"), []byte("changed dependency"), 0o444); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyOpenRGBAppDir(ctx, root, tree, payloadSHA, seal); err == nil {
		t.Fatal("the expected ELF digest concealed a changed dependency")
	}
	if _, err := verifyOpenRGBAppDir(ctx, root+":other", tree, payloadSHA, seal); err == nil {
		t.Fatal("a library-search path separator was accepted")
	}
	root, _, payloadSHA = testOpenRGBAppDir(t)
	if err := os.Symlink("/etc/passwd", filepath.Join(root, "usr/lib/outside.so")); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyOpenRGBAppDir(ctx, root, tree, payloadSHA, seal); err == nil {
		t.Fatal("the closure followed an external symlink")
	}
}

func TestOpenRGBStagedServingProofRequiresTheExactRootImageAndFrozenInputs(t *testing.T) {
	ctx := context.Background()
	x := newOpenRGBExecFixture(t)
	if _, err := x.store.Prepare(ctx, x.fence); err != nil {
		t.Fatal(err)
	}
	if err := x.run(); err != nil {
		t.Fatal(err)
	}
	f, err := x.store.read()
	if err != nil {
		t.Fatal(err)
	}
	c := x.update.Rollback
	base, err := renderManagedUpdateInvocation(c, x.fence.Intent.Runtime, x.update.ID, x.fence.Intent.InstanceID, "private-token")
	if err != nil {
		t.Fatal(err)
	}
	o, err := newOpenRGBServingObservation(f, x.fence.Intent, c, base, x.ref, f.OpenRGBWitness.Incarnation)
	if err != nil {
		t.Fatal(err)
	}
	root, tree, payloadSHA := testOpenRGBAppDir(t)
	info, err := verifyOpenRGBAppDir(ctx, root, tree, payloadSHA, func(context.Context, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	stage := openRGBAppImageStage{envelope: c.Artifact, appDir: root, payloadSHA256: payloadSHA, payload: info}
	want, err := stage.servingInvocation(base, x.fence.Intent.Runtime.Port)
	if err != nil {
		t.Fatal(err)
	}
	live := openRGBAppImageServingProcess{image: info, imageSHA256: payloadSHA, args: append([]string{want.Path}, want.Args...), env: want.Env, dir: root, config: c.ConfigBytes}
	contextFor := func(ref platform.ServiceRef, incarnation string) (string, error) {
		if ref != x.ref || incarnation != f.OpenRGBWitness.Incarnation {
			t.Fatal("serving proof accepted another native root")
		}
		return f.OpenRGBWitness.ContextSHA256, nil
	}
	listener := func(endpoint netip.AddrPort, pid int) error {
		if pid != x.ref.RootPID || int(endpoint.Port()) != x.fence.Intent.Runtime.Port || endpoint.Addr().String() != "127.0.0.1" {
			t.Fatal("serving proof used an unrelated listener")
		}
		return nil
	}
	if err := o.verifyAppImageStage(ctx, stage, live, contextFor, listener); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*openRGBAppImageServingProcess){
		func(p *openRGBAppImageServingProcess) { p.imageSHA256 = c.Artifact.SHA256 },
		func(p *openRGBAppImageServingProcess) {
			p.env = append(append([]string(nil), p.env...), "LD_PRELOAD=/other")
		},
		func(p *openRGBAppImageServingProcess) { p.dir = filepath.Dir(root) },
		func(p *openRGBAppImageServingProcess) { p.config = []byte(`{}`) },
	} {
		changed := live
		mutate(&changed)
		if err := o.verifyAppImageStage(ctx, stage, changed, contextFor, listener); err == nil {
			t.Fatal("changed staged process evidence was accepted")
		}
	}
	other := filepath.Join(t.TempDir(), "same-bytes")
	if err := os.WriteFile(other, []byte("synthetic ELF"), 0o555); err != nil {
		t.Fatal(err)
	}
	changed := live
	changed.image, err = os.Stat(other)
	if err != nil {
		t.Fatal(err)
	}
	if err := o.verifyAppImageStage(ctx, stage, changed, contextFor, listener); err == nil {
		t.Fatal("known bytes in an unrelated inode were treated as the owned executable")
	}
	file, err := stage.openPayload()
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	if err := os.Rename(other, want.Path); err != nil {
		t.Fatal(err)
	}
	if file, err := stage.openPayload(); err == nil {
		_ = file.Close()
		t.Fatal("a replaced payload inode was opened for exec")
	}
}

func TestOpenRGBServingConfigPermitsOnlyDisabledDetectorPruning(t *testing.T) {
	config := func(detectors map[string]bool) []byte {
		data, err := json.Marshal(map[string]any{"Client": map[string]any{"clients": []any{}}, "Server": map[string]any{"all_controllers": false}, "Detectors": map[string]any{"detectors": detectors}, "QMKOpenRGBDevices": map[string]any{"devices": []any{}}, "QMKVialRGBDevices": map[string]any{"devices": []any{}}})
		if err != nil {
			t.Fatal(err)
		}
		var root map[string]json.RawMessage
		if err := json.Unmarshal(data, &root); err != nil {
			t.Fatal(err)
		}
		if err := freezeOpenRGBLocalSettings(root); err != nil {
			t.Fatal(err)
		}
		data, err = json.Marshal(root)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	frozen := config(map[string]bool{"selected": true, "unsupported": false})
	if err := verifyOpenRGBServingDetectors(frozen, config(map[string]bool{"selected": true})); err != nil {
		t.Fatal(err)
	}
	for _, detectors := range []map[string]bool{{"selected": false}, {"unknown": false, "selected": true}, {"selected": true, "unsupported": true}} {
		if err := verifyOpenRGBServingDetectors(frozen, config(detectors)); err == nil {
			t.Fatalf("changed detector selection accepted: %+v", detectors)
		}
	}
}

func TestOpenRGBUpdateExecUsesStagedImageAfterConfinementAndBeforeClaim(t *testing.T) {
	x := newOpenRGBExecFixture(t)
	if _, err := x.store.Prepare(context.Background(), x.fence); err != nil {
		t.Fatal(err)
	}
	root, _, _ := testOpenRGBAppDir(t)
	payload := filepath.Join(root, openRGBAppDirPayload)
	confined, prepared := false, false
	x.owner.openRGB.confineExecThread = func() (func(), error) { confined = true; return func() {}, nil }
	x.owner.openRGBImage = func(_ context.Context, artifact ManagedServiceArtifact, base managedUpdateInvocation, _ int) (openRGBAppImageStage, managedUpdateInvocation, *os.File, error) {
		if !confined || artifact.SHA256 != x.update.Rollback.Artifact.SHA256 || base.Path != artifact.Path {
			t.Fatal("staged image preparation escaped final confinement or its envelope")
		}
		prepared = true
		base.Path, base.Dir = payload, root
		base.Env = append(base.Env, "LD_LIBRARY_PATH="+filepath.Join(root, "usr/lib"))
		file, err := os.Open(payload)
		return openRGBAppImageStage{}, base, file, err
	}
	observe := x.owner.openRGBObserve
	x.owner.openRGBObserve = func(ctx context.Context, contract ManagedUpdateContract, invocation managedUpdateInvocation, f managedUpdateExecFence, ref platform.ServiceRef, incarnation string) (openRGBExecWitness, error) {
		if !prepared || invocation.Path != contract.Artifact.Path {
			t.Fatal("the witness did not bind the envelope after staged verification")
		}
		return observe(ctx, contract, invocation, f, ref, incarnation)
	}
	x.owner.chdir = func(dir string) error {
		if dir != root {
			t.Fatal("exec did not use the sealed AppDir working directory")
		}
		return nil
	}
	x.owner.replace = func(file *os.File, args, env []string) error {
		if args[0] != payload || managedServiceEnvValues(env)["LD_LIBRARY_PATH"] != filepath.Join(root, "usr/lib") {
			t.Fatal("exec did not use the staged payload invocation")
		}
		info, err := file.Stat()
		want, statErr := os.Stat(payload)
		if err != nil || statErr != nil || !os.SameFile(info, want) {
			t.Fatal("exec received the envelope instead of its verified payload FD")
		}
		x.replaced++
		return nil
	}
	if err := x.run(); err != nil || x.replaced != 1 {
		t.Fatalf("staged update execution failed: %v", err)
	}
	f, err := x.store.read()
	if err != nil || f.OpenRGBWitness == nil || f.OpenRGBWitness.Policy.ArtifactSHA256 != x.update.Rollback.Artifact.SHA256 {
		t.Fatalf("staged execution changed the durable envelope identity: %v", err)
	}
}
