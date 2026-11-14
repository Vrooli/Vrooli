package uxcapture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validBundle() Bundle {
	lenses := make(map[LensName]Lens, len(lensOrder))
	for _, name := range lensOrder {
		lenses[name] = Lens{Status: LensAbsent, Reason: "not measured by this test"}
	}
	lenses[LensPixels] = Lens{Status: LensPresent}
	return Bundle{
		SchemaVersion: SchemaVersion,
		ID:            "bndl-01",
		CapturedAt:    "2026-11-14T05:00:00Z",
		Source:        Source{Kind: SourceURL, URL: "https://example.org/"},
		Session:       Session{ID: "session-01"},
		Viewport:      Viewport{Width: 1440, Height: 900},
		Lenses:        lenses,
		Privacy:       Privacy{RedactionStatus: "complete", External: true, Ephemeral: true},
		NextCommands:  []string{"inspect the outline", "revise the selected region"},
	}
}

func TestBundleValidateRequiresEveryLensAndReasonForMissingLenses(t *testing.T) {
	bundle := validBundle()
	delete(bundle.Lenses, LensIdentity)
	if err := bundle.Validate(); err == nil || !strings.Contains(err.Error(), "lens L4 must be explicitly") {
		t.Fatalf("expected missing-lens error, got %v", err)
	}

	bundle = validBundle()
	bundle.Lenses[LensIdentity] = Lens{Status: LensPartial}
	if err := bundle.Validate(); err == nil || !strings.Contains(err.Error(), "requires a reason") {
		t.Fatalf("expected partial-lens reason error, got %v", err)
	}
}

func TestBundleValidateRequiresRedactionAndSafeArtifacts(t *testing.T) {
	bundle := validBundle()
	bundle.Privacy.RedactionStatus = "incomplete"
	if err := bundle.Validate(); err == nil || !strings.Contains(err.Error(), "redaction status must be complete or partial") {
		t.Fatalf("expected redaction error, got %v", err)
	}

	bundle = validBundle()
	bundle.Artifacts = []Artifact{{Kind: "screenshot", Path: "../secret.png", SizeBytes: 10, SHA256: strings.Repeat("a", 64)}}
	if err := bundle.Validate(); err == nil || !strings.Contains(err.Error(), "safe relative path") {
		t.Fatalf("expected unsafe path error, got %v", err)
	}

	bundle.Artifacts = []Artifact{{Kind: "screenshot", Path: "pixels/desktop.png", SizeBytes: 10, SHA256: strings.Repeat("z", 64)}}
	if err := bundle.Validate(); err == nil || !strings.Contains(err.Error(), "invalid SHA-256") {
		t.Fatalf("expected digest error, got %v", err)
	}
}

func TestBundleValidateRejectsCredentialedAndNonHTTPURLSources(t *testing.T) {
	for _, raw := range []string{"ftp://example.org/file", "https://user:secret@example.org/private"} {
		bundle := validBundle()
		bundle.Source.URL = raw
		if err := bundle.Validate(); err == nil {
			t.Errorf("expected URL rejection for %q", raw)
		}
	}
}

func TestBundleValidateRequiresExternalCaptureToBeEphemeral(t *testing.T) {
	bundle := validBundle()
	bundle.Privacy.Ephemeral = false
	if err := bundle.Validate(); err == nil || !strings.Contains(err.Error(), "marked ephemeral") {
		t.Fatalf("expected external retention error, got %v", err)
	}
}

func TestBundleValidateRejectsDuplicateArtifactPaths(t *testing.T) {
	bundle := validBundle()
	digest := Digest([]byte("x"))
	bundle.Artifacts = []Artifact{
		{Kind: "pixels", Path: "pixels/desktop.png", SizeBytes: 1, SHA256: digest},
		{Kind: "pixels-copy", Path: "pixels/desktop.png", SizeBytes: 1, SHA256: digest},
	}
	if err := bundle.Validate(); err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("expected duplicate artifact path error, got %v", err)
	}
}

func TestWriteBundleEnforcesRegionStyleSampleLimit(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "capture")
	payload := []byte(strings.Repeat("x", 8*1024+1))
	bundle := validBundle()
	bundle.Artifacts = []Artifact{{Kind: string(ArtifactRegionStyle), Path: "style/region-sample.json", SizeBytes: int64(len(payload)), SHA256: Digest(payload)}}
	if err := WriteBundleWithArtifacts(dir, bundle, []ArtifactPayload{{Path: "style/region-sample.json", Content: payload}}); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected region style sample limit error, got %v", err)
	}
}

func TestWriteBundlePersistsPrivateManifestAndFidelityREADME(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "capture")
	content := []byte("redacted pixels")
	bundle := validBundle()
	bundle.Artifacts = []Artifact{{Kind: "screenshot", Path: "pixels/desktop.png", MediaType: "image/png", SizeBytes: int64(len(content)), SHA256: Digest(content)}}
	if err := WriteBundleWithArtifacts(dir, bundle, []ArtifactPayload{{Path: "pixels/desktop.png", Content: content}}); err != nil {
		t.Fatalf("WriteBundleWithArtifacts: %v", err)
	}

	for _, name := range []string{"manifest.json", "README.md"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s permissions = %o, want 600", name, info.Mode().Perm())
		}
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("bundle directory permissions = %o, want 700", dirInfo.Mode().Perm())
	}
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(readme)
	for _, want := range []string{"ux-capture.v1", "L0 | present", "L1 | absent", "remove the bundle after use", "inspect the outline"} {
		if !strings.Contains(text, want) {
			t.Errorf("README missing %q", want)
		}
	}
	if err := WriteBundleWithArtifacts(dir, bundle, []ArtifactPayload{{Path: "pixels/desktop.png", Content: content}}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected overwrite refusal, got %v", err)
	}
}

func TestWriteBundleWithArtifactsPublishesMatchingPayloads(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "capture")
	payload := []byte("redacted screenshot bytes")
	bundle := validBundle()
	bundle.Artifacts = []Artifact{{
		Kind: "screenshot", Path: "pixels/desktop.png", MediaType: "image/png",
		SizeBytes: int64(len(payload)), SHA256: Digest(payload),
	}}
	if err := WriteBundleWithArtifacts(dir, bundle, []ArtifactPayload{{Path: "pixels/desktop.png", Content: payload}}); err != nil {
		t.Fatalf("WriteBundleWithArtifacts: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "pixels", "desktop.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("artifact bytes = %q, want %q", got, payload)
	}
	info, err := os.Stat(filepath.Join(dir, "pixels", "desktop.png"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("artifact permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestDigestIsStableSHA256(t *testing.T) {
	if got, want := Digest([]byte("pixels")), "6ec9c2b0eb14010746c8bce8939303b382344b296206612eb8a907a37b2b2f37"; got != want {
		t.Fatalf("Digest = %s, want %s", got, want)
	}
}
