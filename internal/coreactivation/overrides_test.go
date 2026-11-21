package coreactivation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeOverridesAreOrderIndependentAndDetectContentDrift(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.conf"), filepath.Join(dir, "b.conf")
	for _, path := range []string{a, b} {
		if err := os.WriteFile(path, []byte("[Service]\nEnvironment=OWNER_POLICY=preserved\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	first, err := NativeOverridesDigest(ctx, a+" "+b)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NativeOverridesDigest(ctx, b+" "+a)
	if err != nil || first != second {
		t.Fatal("native listing order changed the policy binding")
	}
	if err := os.WriteFile(b, []byte("operator changed policy"), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := NativeOverridesDigest(ctx, a+" "+b)
	if err != nil || changed == first {
		t.Fatal("operator override content drift was hidden")
	}
	if _, err := NativeOverridesDigest(ctx, a+" "+a); err == nil {
		t.Fatal("duplicate override path accepted")
	}
	if err := os.Remove(b); err != nil {
		t.Fatal(err)
	}
	if _, err := NativeOverridesDigest(ctx, a+" "+b); err == nil {
		t.Fatal("missing override silently discarded")
	}
}
