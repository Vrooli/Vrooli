//go:build linux

package hostwireless

import (
	"context"
	"crypto/x509"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func indexedFixture(t *testing.T, profile string) (ActivationPlan, fstest.MapFS, fstest.MapFS, *x509.Certificate) {
	t.Helper()
	plan, baseline, candidate := copiedActivationFixture(t)
	if profile != plan.Profile {
		plan.Profile, plan.Originals, plan.Candidates = profile, nil, nil
		plan.OverlayDirectory = overlayDirectory(profile)
		candidate = fstest.MapFS{}
		for _, module := range preparationProfiles[profile].modules {
			name := "/" + abiTree + "/kernel/" + module + ".ko.zst"
			plan.Originals = append(plan.Originals, identityFixture(baseline, name, "stock "+module))
			plan.Candidates = append(plan.Candidates, identityFixture(candidate, module+".ko", "candidate "+module))
		}
	}
	plan.Bundle, plan.Modaliases = "test-generation", []string{"fixture"}
	symbols := identityFixture(baseline, "/"+abiTree+"/build/Module.symvers", "reviewed kernel symbols")
	plan.Preserved = append(plan.Preserved, symbols)
	plan.KernelSymbolsSHA256 = symbols.SHA256
	for _, name := range depmodIndexes {
		file := "/" + abiTree + "/" + name
		identity := identityFixture(baseline, file, "stock "+name)
		if name == "modules.dep" {
			identity = identityFixture(baseline, file, dependencyFixture(plan, false))
		}
		found := false
		for i, old := range plan.Preserved {
			if old.Path == file {
				plan.Preserved[i], found = identity, true
			}
		}
		if !found {
			plan.Preserved = append(plan.Preserved, identity)
		}
	}
	dir := t.TempDir()
	certFile, key := testCertificate(t, dir, "indexed")
	cert, err := readPublicCertificate(certFile)
	if err != nil {
		t.Fatal(err)
	}
	signed := fstest.MapFS{}
	for _, original := range plan.Candidates {
		payload := candidate[original.Path].Data
		signature := signFixture(t, dir, certFile, key, payload)
		signed[original.Path] = &fstest.MapFile{Data: appendModuleSignature(payload, signature)}
	}
	return plan, baseline, signed, cert
}

func dependencyFixture(plan ActivationPlan, candidate bool) string {
	var lines []string
	for i, file := range plan.Originals {
		name := strings.TrimPrefix(file.Path, "/"+abiTree+"/")
		if candidate {
			name = strings.TrimPrefix(plan.OverlayDirectory, abiTree+"/") + "/" + plan.Candidates[i].Path
		}
		lines = append(lines, name+":")
	}
	return strings.Join(lines, "\n") + "\n"
}

func fixtureIndexes(ctx context.Context, copied *CopiedActivation, plan ActivationPlan) error {
	for _, name := range depmodIndexes {
		data := []byte("candidate " + name)
		if name == "modules.dep" {
			data = []byte(dependencyFixture(plan, true))
		}
		if err := copied.root.WriteFile(path.Join(abiTree, name), data, 0600); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func TestSignedIndexedGenerationStagesAndRestoresBothProfiles(t *testing.T) {
	for _, profile := range []string{NobleProfile, NobleBluetoothProfile} {
		t.Run(profile, func(t *testing.T) {
			ctx := context.Background()
			plan, baseline, signed, cert := indexedFixture(t, profile)
			tx, err := newIndexedActivation(ctx, baseline, plan, fixtureIndexes)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = tx.close() })
			stock := maps.Clone(tx.stockTree)
			for i := 0; i < 2; i++ {
				modules, err := tx.stage(ctx, signed, cert)
				if err != nil || len(modules) != len(plan.Candidates) {
					t.Fatalf("stage %+v %v", modules, err)
				}
			}
			for _, module := range tx.modules {
				if module.UnsignedSHA256 == module.SignedSHA256 {
					t.Fatal("signature was not bound to the changed copy")
				}
			}
			if err := tx.verifySelected(ctx); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				if err := tx.rollback(ctx); err != nil {
					t.Fatal(err)
				}
			}
			after, err := snapshotPrivateTree(ctx, tx.stock.root)
			if err != nil || !maps.Equal(stock, after) || tx.candidate != nil {
				t.Fatalf("rollback changed stock generation: %v", err)
			}
			if _, err := tx.stage(ctx, signed, cert); err != nil {
				t.Fatalf("stage after rollback failed: %v", err)
			}
			if err := tx.rollback(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSignedIndexFailureCannotPublishPartialGeneration(t *testing.T) {
	for _, failure := range []string{"partial", "missing-index", "untracked-index", "protected-drift", "protected-mode", "cancelled"} {
		t.Run(failure, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			plan, baseline, signed, cert := indexedFixture(t, NobleBluetoothProfile)
			index := func(ctx context.Context, root *CopiedActivation, plan ActivationPlan) error {
				if err := root.root.WriteFile(abiTree+"/modules.dep", []byte("partial"), 0600); err != nil {
					return err
				}
				switch failure {
				case "partial":
					return errors.New("injected depmod interruption")
				case "missing-index":
					return nil
				case "protected-drift":
					if err := fixtureIndexes(ctx, root, plan); err != nil {
						return err
					}
					return root.root.WriteFile("boot/grub/grubenv", []byte("drift"), 0600)
				case "untracked-index":
					if err := fixtureIndexes(ctx, root, plan); err != nil {
						return err
					}
					return root.root.WriteFile(abiTree+"/modules.unexpected", []byte("unknown index"), 0600)
				case "protected-mode":
					if err := fixtureIndexes(ctx, root, plan); err != nil {
						return err
					}
					return root.root.Chmod("boot/grub/grubenv", 0644)
				default:
					if err := fixtureIndexes(ctx, root, plan); err != nil {
						return err
					}
					cancel()
					return nil
				}
			}
			tx, err := newIndexedActivation(ctx, baseline, plan, index)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = tx.close() })
			if _, err := tx.stage(ctx, signed, cert); err == nil {
				t.Fatal("published a failed index generation")
			}
			if tx.candidate != nil {
				t.Fatal("failed generation retained as selected")
			}
			if err := tx.verifySelected(context.Background()); err != nil {
				t.Fatal(err)
			}
			tx.index = fixtureIndexes
			if _, err := tx.stage(context.Background(), signed, cert); err != nil {
				t.Fatalf("retry failed: %v", err)
			}
			if err := tx.candidate.root.WriteFile(abiTree+"/modules.dep.bin", []byte("foreign"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := tx.rollback(context.Background()); err == nil || !strings.Contains(err.Error(), "generation drifted") {
				t.Fatalf("rollback accepted ambiguous index identity: %v", err)
			}
		})
	}
}

func TestSignedIndexedStageRechecksBytesAfterAuthentication(t *testing.T) {
	ctx := context.Background()
	plan, baseline, signed, cert := indexedFixture(t, NobleBluetoothProfile)
	tx, err := newIndexedActivation(ctx, baseline, plan, fixtureIndexes)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.close() })
	mutating := &changeSignedCopy{FS: signed, source: signed, target: plan.Candidates[0].Path}
	if _, err := tx.stage(ctx, mutating, cert); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("accepted changed bytes after authentication: %v", err)
	}
	if tx.candidate != nil {
		t.Fatal("changed signed copy was published")
	}
	if err := tx.verifySelected(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestIndexedActivationRejectsKernelAndSelectorDrift(t *testing.T) {
	ctx := context.Background()
	plan, baseline, signed, cert := indexedFixture(t, NobleBluetoothProfile)
	wrong := plan
	wrong.KernelSymbolsSHA256 = digest([]byte("another header tree"))
	if tx, err := newIndexedActivation(ctx, baseline, wrong, fixtureIndexes); err == nil {
		_ = tx.close()
		t.Fatal("accepted a mismatched prepared kernel symbol identity")
	}
	tx, err := newIndexedActivation(ctx, baseline, plan, fixtureIndexes)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.close() })
	if err := os.Remove(filepath.Join(tx.dir, "current")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("unknown-generation", filepath.Join(tx.dir, "current")); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.stage(ctx, signed, cert); err == nil || !strings.Contains(err.Error(), "selector drifted") {
		t.Fatalf("ambiguous selector stage: %v", err)
	}
	if err := tx.rollback(ctx); err == nil || !strings.Contains(err.Error(), "selector drifted") {
		t.Fatalf("ambiguous selector rollback: %v", err)
	}
}

type changeSignedCopy struct {
	fs.FS
	source fstest.MapFS
	target string
	opens  int
}

func (f *changeSignedCopy) Open(name string) (fs.File, error) {
	if name == f.target {
		f.opens++
		if f.opens == 2 {
			f.source[name].Data[0] ^= 1
		}
	}
	return f.FS.Open(name)
}
