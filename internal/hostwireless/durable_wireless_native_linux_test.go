//go:build linux

package hostwireless

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"testing"
)

// Called only by the explicit native signed-indexed fixture. Real prepared
// module bytes are signed with that fixture's ephemeral key; boot files remain
// synthetic and every publication occurs under a new private copied root.
func rehearseNativeDurableWireless(t *testing.T, indexed *indexedActivation, cert *x509.Certificate) {
	t.Helper()
	ctx := context.Background()
	owner, target := durableTargetFixture(t, indexed)
	images, err := observeFirmwarePreflight(ctx, indexed.plan.Profile)
	if err != nil {
		t.Fatal(err)
	}
	if err = owner.begin(ctx, indexed, cert, images...); err != nil {
		t.Fatal(err)
	}
	// Simulate losing the process after an individual native index rename but
	// before its parent fsync. Startup must recognize old/new identities and
	// durably finish that effect before reporting publication.
	interrupted := path.Join(abiTree, depmodIndexes[3])
	fired := false
	owner.fault = func(event string) error {
		if event == "after-rename:"+interrupted {
			fired = true
			return errors.New("native fixture process interruption")
		}
		return nil
	}
	if err = owner.apply(ctx); err == nil || !fired {
		t.Fatal("native publication interruption not exercised", err)
	}
	owner.close()
	reopened, err := openDurableWireless(ctx, target.root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	if err = reopened.apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err = compareDependencyProviders(indexed.stock.root.FS(), target.root.FS(), indexed.plan); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range indexed.plan.Candidates {
		module := candidate.Path[:len(candidate.Path)-3]
		if err = requirePrivateProvider(ctx, target.root.Name(), module, path.Join(target.root.Name(), indexed.plan.OverlayDirectory, candidate.Path)); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("durable_profile=%s generation=%s journal=%s intent_sha256=%s files=%d firmware=%d partial_index_reopen=true published=true synthetic_boot_fixture=true", indexed.plan.Profile, indexed.plan.Bundle, reopened.intent.ID, reopened.intentSHA, len(reopened.intent.Files), len(reopened.intent.Firmware))
	if directory := os.Getenv("VROOLI_WIRELESS_NATIVE_EVIDENCE_DIR"); directory != "" {
		data, err := json.Marshal(reopened.intent)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(directory, "durable-"+indexed.plan.Profile+"-intent.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err = reopened.rollback(ctx); err != nil {
		t.Fatal(err)
	}
	for _, file := range reopened.intent.Files {
		got, err := reopened.identity(file.Path)
		if err != nil || got != file.Before {
			t.Fatalf("native exact original not restored %s: %v", file.Path, err)
		}
	}
	if err = reopened.verifyProtected(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("durable_profile=%s exact_original_bytes_mode_uid_gid_restored=true all_stock_files_preserved=true live_effects=false kernel_trust_verified=false activation_eligible=false", indexed.plan.Profile)
}
