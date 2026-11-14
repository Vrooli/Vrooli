//go:build linux

package hostwireless

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func durableIndexedFixture(t *testing.T, profile string) *indexedActivation {
	t.Helper()
	plan, baseline, signed, cert := indexedFixture(t, profile)
	indexed, err := newIndexedActivation(context.Background(), baseline, plan, fixtureIndexes)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = indexed.close() })
	if _, err = indexed.stage(context.Background(), signed, cert); err != nil {
		t.Fatal(err)
	}
	// Keep only public cert bytes in the fixture root; no key enters the owner.
	if err = indexed.stock.root.WriteFile("fixture-public.der", cert.Raw, 0600); err != nil {
		t.Fatal(err)
	}
	return indexed
}

func durableTargetFixture(t *testing.T, indexed *indexedActivation) (*durableWireless, *CopiedActivation) {
	t.Helper()
	ctx := context.Background()
	target, err := newCopiedActivation(ctx, indexed.stock.root.FS(), indexed.plan, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = target.Close() })
	if err = cloneABITree(ctx, indexed.stock.root.FS(), target.root); err != nil {
		t.Fatal(err)
	}
	owner, err := openDurableWireless(ctx, target.root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = owner.close() })
	return owner, target
}

func beginDurableFixture(t *testing.T, owner *durableWireless, indexed *indexedActivation) {
	t.Helper()
	certBytes, err := os.ReadFile(path.Join(indexed.stock.root.Name(), "fixture-public.der"))
	if err != nil {
		t.Fatal(err)
	}
	// Fixture certificate is outside the observed stock manifest.
	if err = os.Remove(path.Join(indexed.stock.root.Name(), "fixture-public.der")); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = indexed.stock.root.WriteFile("fixture-public.der", certBytes, 0600) }()
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err = owner.begin(context.Background(), indexed, cert); err != nil {
		t.Fatal(err)
	}
}

func TestDurableWirelessReopensEveryTargetPublicationWindow(t *testing.T) {
	for _, profile := range []string{NobleProfile, NobleBluetoothProfile} {
		t.Run(profile, func(t *testing.T) {
			indexed := durableIndexedFixture(t, profile)
			for _, point := range []string{"before-rename:", "after-rename:", "after-sync:"} {
				for index := 0; index < len(preparationProfiles[profile].modules)+len(depmodIndexes); index++ {
					t.Run(fmt.Sprintf("%s%d", point, index), func(t *testing.T) {
						owner, target := durableTargetFixture(t, indexed)
						beginDurableFixture(t, owner, indexed)
						wanted := point + owner.intent.Files[index].Path
						stopped := false
						owner.fault = func(event string) error {
							if event == wanted {
								stopped = true
								return errors.New("simulated process loss")
							}
							return nil
						}
						if err := owner.apply(context.Background()); err == nil || !stopped {
							t.Fatalf("fault was not exercised: %v", err)
						}
						owner.close()
						reopened, err := openDurableWireless(context.Background(), target.root)
						if err != nil {
							t.Fatal(err)
						}
						defer reopened.close()
						if err = reopened.apply(context.Background()); err != nil {
							t.Fatal(err)
						}
						if err = reopened.rollback(context.Background()); err != nil {
							t.Fatal(err)
						}
						for _, file := range reopened.intent.Files {
							got, err := reopened.identity(file.Path)
							if err != nil || got != file.Before {
								t.Fatalf("original not restored: %s %+v %v", file.Path, got, err)
							}
						}
						if err = reopened.rollback(context.Background()); err != nil {
							t.Fatal("rollback not idempotent", err)
						}
					})
				}
			}
		})
	}
}

func TestDurableWirelessPreservesForeignDriftAndRetainsRecoveryBytes(t *testing.T) {
	indexed := durableIndexedFixture(t, NobleBluetoothProfile)
	owner, target := durableTargetFixture(t, indexed)
	beginDurableFixture(t, owner, indexed)
	if err := owner.apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	file := owner.intent.Files[len(owner.intent.Files)-1].Path
	if err := target.root.WriteFile(file, []byte("foreign administrator bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := owner.rollback(context.Background()); err == nil {
		t.Fatal("foreign bytes overwritten")
	}
	owner.close()
	if _, err := openDurableWireless(context.Background(), target.root); err == nil {
		t.Fatal("foreign reopen accepted")
	}
	data, err := target.root.ReadFile(file)
	if err != nil || string(data) != "foreign administrator bytes" {
		t.Fatal("foreign bytes lost")
	}
	if _, err = target.root.Stat(owner.statePath("old-2")); err != nil {
		t.Fatal("undo bytes removed on Close")
	}
}

func TestDurableWirelessCrossProfileLockAndCancellationRollback(t *testing.T) {
	indexed := durableIndexedFixture(t, NobleBluetoothProfile)
	owner, target := durableTargetFixture(t, indexed)
	beginDurableFixture(t, owner, indexed)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if second, err := openDurableWireless(canceled, target.root); err == nil {
		second.close()
		t.Fatal("concurrent owner obtained lock")
	}
	ctx, stop := context.WithCancel(context.Background())
	owner.fault = func(event string) error {
		if strings.HasPrefix(event, "after-rename:"+owner.intent.Plan.OverlayDirectory) {
			stop()
		}
		return nil
	}
	if err := owner.apply(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled result, got %v", err)
	}
	if owner.phase != "rolled_back" {
		t.Fatalf("cancellation did not compensate: %s", owner.phase)
	}
	for _, file := range owner.intent.Files {
		got, err := owner.identity(file.Path)
		if err != nil || got != file.Before {
			t.Fatalf("cancellation rollback differs: %s", file.Path)
		}
	}
}

func TestDurableWirelessRejectsInternalSymlinkAndSupersedingLineage(t *testing.T) {
	indexed := durableIndexedFixture(t, NobleBluetoothProfile)
	owner, target := durableTargetFixture(t, indexed)
	beginDurableFixture(t, owner, indexed)
	if err := owner.apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	file := owner.intent.Files[0].Path
	if err := target.root.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := target.root.Symlink(strings.TrimPrefix(owner.intent.Plan.Originals[0].Path, "/"), file); err != nil {
		t.Fatal(err)
	}
	if err := owner.rollback(context.Background()); err == nil {
		t.Fatal("in-root symlink accepted")
	}
	if _, err := target.root.Lstat(file); !errors.Is(err, fs.ErrNotExist) && err != nil {
		t.Fatal(err)
	}
}

func TestDurableWirelessRollbackPreservesLateForeignDrift(t *testing.T) {
	for _, index := range []int{0, 3} {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			indexed := durableIndexedFixture(t, NobleBluetoothProfile)
			owner, target := durableTargetFixture(t, indexed)
			beginDurableFixture(t, owner, indexed)
			if err := owner.apply(context.Background()); err != nil {
				t.Fatal(err)
			}
			name := owner.intent.Files[index].Path
			owner.fault = func(event string) error {
				if event == "after-rename:"+owner.statePath("phase") {
					return target.root.WriteFile(name, []byte("late foreign identity"), 0644)
				}
				return nil
			}
			if err := owner.rollback(context.Background()); err == nil {
				t.Fatal("late foreign drift overwritten")
			}
			data, err := target.root.ReadFile(name)
			if err != nil || string(data) != "late foreign identity" {
				t.Fatal("foreign identity lost")
			}
		})
	}
}

func TestDurableWirelessAuthenticatesRetainedSignatureOnReopen(t *testing.T) {
	indexed := durableIndexedFixture(t, NobleBluetoothProfile)
	owner, target := durableTargetFixture(t, indexed)
	beginDurableFixture(t, owner, indexed)
	if err := target.root.WriteFile(owner.statePath("certificate.der"), []byte("foreign public certificate"), 0600); err != nil {
		t.Fatal(err)
	}
	owner.close()
	if recovered, err := openDurableWireless(context.Background(), target.root); err == nil {
		recovered.close()
		t.Fatal("tampered retained signature evidence accepted")
	}
}

func TestDurableWirelessPreparationInterruptionHasNoTargetEffects(t *testing.T) {
	indexed := durableIndexedFixture(t, NobleBluetoothProfile)
	for _, suffix := range []string{"intent.json", "phase", "active.json"} {
		for _, prefix := range []string{"before-write:", "before-file-sync:", "after-file-sync:", "before-rename:", "after-rename:", "after-sync:"} {
			t.Run(prefix+suffix, func(t *testing.T) {
				owner, target := durableTargetFixture(t, indexed)
				certBytes, _ := indexed.stock.root.ReadFile("fixture-public.der")
				_ = indexed.stock.root.Remove("fixture-public.der")
				defer indexed.stock.root.WriteFile("fixture-public.der", certBytes, 0600)
				cert, err := x509.ParseCertificate(certBytes)
				if err != nil {
					t.Fatal(err)
				}
				fired := false
				owner.fault = func(event string) error {
					if strings.HasPrefix(event, prefix) && strings.HasSuffix(event, "/"+suffix) {
						fired = true
						return errors.New("preparation interrupted")
					}
					return nil
				}
				if err = owner.begin(context.Background(), indexed, cert); err == nil || !fired {
					t.Fatal("preparation fault not exercised", err)
				}
				owner.close()
				reopened, err := openDurableWireless(context.Background(), target.root)
				if err != nil {
					t.Fatal(err)
				}
				defer reopened.close()
				for name, want := range indexed.stockTree {
					got, err := target.root.Lstat(name)
					if err != nil || got.Mode() != want.Mode {
						t.Fatalf("stock changed before active intent: %s", name)
					}
				}
				if reopened.intent.ID != "" {
					if err = reopened.rollback(context.Background()); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestDurableWirelessResumesAmbiguousDirectoryCreationDurably(t *testing.T) {
	indexed := durableIndexedFixture(t, NobleBluetoothProfile)
	owner, target := durableTargetFixture(t, indexed)
	beginDurableFixture(t, owner, indexed)
	directory := path.Dir(owner.intent.Plan.OverlayDirectory)
	owner.fault = func(event string) error {
		if event == "after-mkdir:"+directory {
			return errors.New("parent fsync interrupted")
		}
		return nil
	}
	if err := owner.apply(context.Background()); err == nil {
		t.Fatal("directory interruption not exercised")
	}
	owner.close()
	reopened, err := openDurableWireless(context.Background(), target.root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	syncedABI := false
	reopened.fault = func(event string) error {
		if event == "after-directory-sync:"+abiTree {
			syncedABI = true
		}
		return nil
	}
	if err = reopened.apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !syncedABI {
		t.Fatal("reused updates directory parent was not durably synced")
	}
	if err = reopened.rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestDurableWirelessStartupResyncsAmbiguousJournalAncestors(t *testing.T) {
	for _, directory := range []string{"var", "var/lib", "var/lib/vrooli", wirelessJournalDir} {
		t.Run(directory, func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			fd, err := root.Open(".")
			if err != nil {
				t.Fatal(err)
			}
			owner := &durableWireless{root: root, fd: fd}
			owner.fault = func(event string) error {
				if event == "after-mkdir:"+directory {
					return errors.New("startup parent sync interrupted")
				}
				return nil
			}
			if err = owner.makeStateDirs(); err == nil {
				t.Fatal("startup interruption not exercised")
			}
			fd.Close()
			owner.fd, err = root.Open(".")
			if err != nil {
				t.Fatal(err)
			}
			defer owner.fd.Close()
			rootSynced := false
			owner.fault = func(event string) error {
				if event == "after-directory-sync:." {
					rootSynced = true
				}
				return nil
			}
			if err = owner.makeStateDirs(); err != nil {
				t.Fatal(err)
			}
			if !rootSynced {
				t.Fatal("journal ancestor link not durably resynced")
			}
			opened, err := owner.open(wirelessJournalDir, unix.O_RDONLY|unix.O_DIRECTORY, 0)
			if err != nil {
				t.Fatal(err)
			}
			opened.Close()
		})
	}
}

func TestDurableWirelessReopensInterruptedRollback(t *testing.T) {
	indexed := durableIndexedFixture(t, NobleBluetoothProfile)
	for _, event := range []string{"before-rename:" + abiTree + "/modules.dep", "after-rename:" + abiTree + "/modules.dep", "after-sync:" + abiTree + "/modules.dep", "before-unlink:" + overlayDirectory(NobleBluetoothProfile) + "/btmtk.ko", "after-unlink:" + overlayDirectory(NobleBluetoothProfile) + "/btmtk.ko"} {
		t.Run(event, func(t *testing.T) {
			owner, target := durableTargetFixture(t, indexed)
			beginDurableFixture(t, owner, indexed)
			if err := owner.apply(context.Background()); err != nil {
				t.Fatal(err)
			}
			fired := false
			owner.fault = func(point string) error {
				if point == event {
					fired = true
					return errors.New("rollback interrupted")
				}
				return nil
			}
			if err := owner.rollback(context.Background()); err == nil || !fired {
				t.Fatal("rollback interruption not exercised", err)
			}
			owner.close()
			reopened, err := openDurableWireless(context.Background(), target.root)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.close()
			if err = reopened.apply(context.Background()); err == nil {
				t.Fatal("interrupted rollback accepted apply")
			}
			if err = reopened.rollback(context.Background()); err != nil {
				t.Fatal(err)
			}
			for _, file := range reopened.intent.Files {
				got, err := reopened.identity(file.Path)
				if err != nil || got != file.Before {
					t.Fatal("rollback identity differs", file.Path, err)
				}
			}
		})
	}
}

func TestDurableWirelessNativeFirmwareRestoresPresenceAndAbsence(t *testing.T) {
	if os.Getenv("VROOLI_WIRELESS_DURABLE_FIRMWARE_TEST") != "1" {
		t.Skip("explicit governed local artifact read required")
	}
	indexed := durableIndexedFixture(t, NobleBluetoothProfile)
	images, err := observeFirmwarePreflight(context.Background(), NobleBluetoothProfile)
	if err != nil {
		t.Fatal(err)
	}
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			owner, target := durableTargetFixture(t, indexed)
			name := durableFirmwarePins(NobleBluetoothProfile)[0].Path
			if existing {
				if err = target.root.MkdirAll(path.Dir(name), 0700); err != nil {
					t.Fatal(err)
				}
				if err = target.root.WriteFile(name, []byte("original local firmware"), 0640); err != nil {
					t.Fatal(err)
				}
			}
			certBytes, _ := indexed.stock.root.ReadFile("fixture-public.der")
			_ = indexed.stock.root.Remove("fixture-public.der")
			defer indexed.stock.root.WriteFile("fixture-public.der", certBytes, 0600)
			cert, err := x509.ParseCertificate(certBytes)
			if err != nil {
				t.Fatal(err)
			}
			if err = owner.begin(context.Background(), indexed, cert, images...); err != nil {
				t.Fatal(err)
			}
			if err = owner.apply(context.Background()); err != nil {
				t.Fatal(err)
			}
			owner.close()
			reopened, err := openDurableWireless(context.Background(), target.root)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.close()
			if err = reopened.rollback(context.Background()); err != nil {
				t.Fatal(err)
			}
			got, err := reopened.identity(name)
			if err != nil {
				t.Fatal(err)
			}
			if existing {
				if got.SHA256 != digest([]byte("original local firmware")) || got.Mode != 0640 {
					t.Fatal("original firmware metadata differs", got)
				}
			} else if !got.Absent {
				t.Fatal("original absence not restored")
			}
		})
	}
}

// A fresh indexed closure includes the already committed peer profile. The
// successor must never replace combined indexes with stock-only peer selection.
func TestDurableWirelessCommittedProfilesCoexistAndRollbackInOrder(t *testing.T) {
	ctx := context.Background()
	var wifiFirmware, btFirmware []FirmwarePreflight
	if os.Getenv("VROOLI_WIRELESS_DURABLE_FIRMWARE_TEST") == "1" {
		var err error
		wifiFirmware, err = observeFirmwarePreflight(ctx, NobleProfile)
		if err != nil {
			t.Fatal(err)
		}
		btFirmware, err = observeFirmwarePreflight(ctx, NobleBluetoothProfile)
		if err != nil {
			t.Fatal(err)
		}
	}
	wifi, baseline, wifiSigned, wifiCert := indexedFixture(t, NobleProfile)
	bt, _, btSigned, btCert := indexedFixture(t, NobleBluetoothProfile)
	combined := dependencyFixture(wifi, false) + dependencyFixture(bt, false)
	currentDependencies := combined
	identity := identityFixture(baseline, "/"+abiTree+"/modules.dep", combined)
	for i, file := range wifi.Preserved {
		if file.Path == identity.Path {
			wifi.Preserved[i] = identity
		}
	}
	indexer := func(ctx context.Context, copied *CopiedActivation, plan ActivationPlan) error {
		updated := currentDependencies
		for i, original := range plan.Originals {
			updated = strings.ReplaceAll(updated, strings.TrimPrefix(original.Path, "/"+abiTree+"/"), strings.TrimPrefix(plan.OverlayDirectory, abiTree+"/")+"/"+plan.Candidates[i].Path)
		}
		for _, name := range depmodIndexes {
			content := "combined " + name
			if name == "modules.dep" {
				content = updated
			}
			if err := copied.root.WriteFile(path.Join(abiTree, name), []byte(content), 0600); err != nil {
				return err
			}
		}
		return ctx.Err()
	}
	wifiIndexed, err := newIndexedActivation(ctx, baseline, wifi, indexer)
	if err != nil {
		t.Fatal(err)
	}
	defer wifiIndexed.close()
	if _, err = wifiIndexed.stage(ctx, wifiSigned, wifiCert); err != nil {
		t.Fatal(err)
	}
	owner, target := durableTargetFixture(t, wifiIndexed)
	if err = owner.begin(ctx, wifiIndexed, wifiCert, wifiFirmware...); err != nil {
		t.Fatal(err)
	}
	if err = owner.apply(ctx); err != nil {
		t.Fatal(err)
	}
	wifiID := owner.intent.ID
	currentData, err := owner.read(abiTree+"/modules.dep", maxArtifactBytes)
	if err != nil {
		t.Fatal(err)
	}
	currentDependencies = string(currentData)
	for i, file := range bt.Preserved {
		got, err := owner.identity(strings.TrimPrefix(file.Path, "/"))
		if err != nil || got.Absent {
			t.Fatal("missing combined baseline", file.Path, err)
		}
		bt.Preserved[i].SHA256 = got.SHA256
	}
	btIndexed, err := newIndexedActivation(ctx, target.root.FS(), bt, indexer)
	if err != nil {
		t.Fatal(err)
	}
	defer btIndexed.close()
	if _, err = btIndexed.stage(ctx, btSigned, btCert); err != nil {
		t.Fatal(err)
	}
	if err = owner.begin(ctx, btIndexed, btCert, btFirmware...); err != nil {
		t.Fatal(err)
	}
	if owner.intent.Previous != wifiID || !owner.intent.PreviousPublished {
		t.Fatal("committed profile lineage lost")
	}
	if err = owner.apply(ctx); err != nil {
		t.Fatal(err)
	}
	for _, plan := range []ActivationPlan{wifi, bt} {
		for _, candidate := range plan.Candidates {
			if _, err = target.root.Stat(path.Join(plan.OverlayDirectory, candidate.Path)); err != nil {
				t.Fatal("peer overlay lost", err)
			}
		}
	}
	if err = owner.rollback(ctx); err != nil {
		t.Fatal(err)
	}
	owner.close()
	parent, err := openDurableWireless(ctx, target.root)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.close()
	if parent.intent.ID != wifiID || parent.phase != "published" {
		t.Fatal("committed parent not restored")
	}
	if err = parent.rollback(ctx); err != nil {
		t.Fatal(err)
	}
	for _, plan := range []ActivationPlan{wifi, bt} {
		if _, err = target.root.Stat(plan.OverlayDirectory); !errors.Is(err, fs.ErrNotExist) {
			t.Fatal("profile overlay not restored to absent", err)
		}
	}
	for _, pin := range append(durableFirmwarePins(NobleProfile), durableFirmwarePins(NobleBluetoothProfile)...) {
		got, err := parent.identity(pin.Path)
		if err != nil || !got.Absent {
			t.Fatal("firmware original absence not restored", pin.Path, err)
		}
	}
}

func TestDurableWirelessRejectsWritableJournalAncestorsAndTransaction(t *testing.T) {
	indexed := durableIndexedFixture(t, NobleBluetoothProfile)
	for _, directory := range []string{"var", "var/lib", "var/lib/vrooli", wirelessJournalDir, "transaction"} {
		t.Run(directory, func(t *testing.T) {
			owner, target := durableTargetFixture(t, indexed)
			beginDurableFixture(t, owner, indexed)
			name := directory
			if directory == "transaction" {
				name = path.Join(wirelessJournalDir, owner.intent.ID)
			}
			if err := target.root.Chmod(name, 0777); err != nil {
				t.Fatal(err)
			}
			owner.close()
			if reopened, err := openDurableWireless(context.Background(), target.root); err == nil {
				reopened.close()
				t.Fatal("untrusted directory admitted", name)
			}
		})
	}
}

func TestDurableWirelessRefusesLiveRootIncludingAlias(t *testing.T) {
	alias := path.Join(t.TempDir(), "host-root")
	if err := os.Symlink("/", alias); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"/", alias} {
		root, err := os.OpenRoot(name)
		if err != nil {
			t.Fatal(err)
		}
		owner, err := openDurableWireless(context.Background(), root)
		root.Close()
		if err == nil {
			owner.close()
			t.Fatal("live root accepted", name)
		}
	}
}
