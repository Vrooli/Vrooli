package runtimeapp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeLightingLauncherBindsPublishedBytesToInstalledHost(t *testing.T) {
	hash := strings.Repeat("a", 64)
	var events []string
	e := nativeLightingLauncherEffects{sealedDigest: func(_ context.Context, path string) (string, error) { events = append(events, path); return hash, nil }, stage: func(context.Context) (string, error) {
		events = append(events, "publication")
		return "/staged", nil
	}}
	if err := prepareNativeLightingLauncherWith(context.Background(), "/installed", e); err != nil {
		t.Fatal(err)
	}
	if strings.Join(events, ",") != "/installed,publication,/staged" {
		t.Fatalf("image lineage order %v", events)
	}
	e.sealedDigest = func(_ context.Context, path string) (string, error) {
		if path == "/staged" {
			return strings.Repeat("b", 64), nil
		}
		return hash, nil
	}
	if err := prepareNativeLightingLauncherWith(context.Background(), "/installed", e); err == nil {
		t.Fatal("published launcher differs from exact installed host")
	}
}

func TestNativeLightingLauncherRequiresTrustedSealedHostBeforePublication(t *testing.T) {
	published := false
	denied := errors.New("installed path not immutable")
	e := nativeLightingLauncherEffects{sealedDigest: func(context.Context, string) (string, error) { return "", denied }, stage: func(context.Context) (string, error) { published = true; return "/staged", nil }}
	if err := prepareNativeLightingLauncherWith(context.Background(), "/untrusted", e); !errors.Is(err, denied) || published {
		t.Fatalf("untrusted installed host triggered publication %v %v", err, published)
	}
}

func TestNativeLightingImageDigestChecksBoundsAndCancellation(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "image"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err = file.WriteString("fixture-image"); err != nil {
		t.Fatal(err)
	}
	if hash, err := nativeLightingCLIImageDigest(context.Background(), file); err != nil || len(hash) != 64 {
		t.Fatalf("digest %q %v", hash, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := nativeLightingCLIImageDigest(ctx, file); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err = file.Truncate(512<<20 + 1); err != nil {
		t.Fatal(err)
	}
	if _, err = nativeLightingCLIImageDigest(context.Background(), file); err == nil {
		t.Fatal("unbounded executable read")
	}
}
