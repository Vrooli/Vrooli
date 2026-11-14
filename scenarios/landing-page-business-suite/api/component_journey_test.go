//go:build integration

package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"github.com/gorilla/mux"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	component "github.com/vrooli/vrooli/packages/desktop-component"
	deliveryhttp "landing-page-business-suite-api/handlers/delivery"
	"landing-page-business-suite-api/internal/delivery"
)

type componentJourneyBundle string

func (b componentJourneyBundle) BundleKey() string { return string(b) }

// Real delivery handlers, PostgreSQL and native MinIO; no fake catalog or
// component download server stands in for LPBS.
func TestDesktopComponentJourneyServer(t *testing.T) {
	root := os.Getenv("J11_RELEASE_ROOT")
	if root == "" {
		t.Skip("J11_RELEASE_ROOT selects a real isolated component release")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	base := "http://" + listener.Addr().String()
	raw, err := os.ReadFile(filepath.Join(root, "entry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var signed component.SignedEntry
	if err = json.Unmarshal(raw, &signed); err != nil {
		t.Fatal(err)
	}
	// A fixture-owned release key signs the actual port-0 address. The real
	// catalog, redirect, PostgreSQL artifact and MinIO object remain production
	// code; only release authority and local addresses are isolated test inputs.
	keyPEM, err := os.ReadFile(filepath.Join(root, "release.key"))
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		t.Fatal("fixture release key is not PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	signed.Entry.ArtifactURL = base + "/api/v1/components?id=" + signed.Entry.ID
	fullApp := os.Getenv("J11_FULL_AUDIO_APPIMAGE")
	if fullApp != "" && signed.Entry.Owner == "audio-tools" {
		signed.Entry.StandaloneURL = base + "/api/v1/updates/audio-tools-full/stable/" + filepath.Base(fullApp)
	}
	canonical, err := signed.Entry.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	signed.Signature.Signature = base64.StdEncoding.EncodeToString(signature)
	publicPEM, err := os.ReadFile(filepath.Join(root, "release.pub"))
	if err != nil {
		t.Fatal(err)
	}
	if err = signed.Verify(string(publicPEM)); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.MarshalIndent(signed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "entry.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(filepath.Join(root, "host-input.json"))
	if err != nil {
		t.Fatal(err)
	}
	var host map[string]any
	if err = json.Unmarshal(raw, &host); err != nil {
		t.Fatal(err)
	}
	host["components"] = component.Source{CatalogURL: base + "/api/v1/components", PublicKeyPEM: string(publicPEM)}
	encoded, err = json.MarshalIndent(host, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "host-input.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	entry := signed.Entry
	db := setupTestDB(t)
	endpoint, access, secret := setupDesktopJourneyMinIO(t)
	ctx := t.Context()
	const bundleKey = "component-journey"
	const bucket = "component-journey-downloads"
	createTestBucket(t, ctx, endpoint, access, secret, bucket)
	service := delivery.NewService(db, integrationS3Provider(access, secret))
	provider, region, pathStyle := "s3", "us-east-1", true
	if _, err = service.SaveSettings(ctx, bundleKey, delivery.StorageSettingsUpdate{Provider: &provider, Bucket: func() *string { s := bucket; return &s }(), Region: &region, Endpoint: &endpoint, ForcePathStyle: &pathStyle}); err != nil {
		t.Fatal(err)
	}
	catalog := delivery.NewCatalogService(db)
	if _, err = catalog.UpsertApp(delivery.App{BundleKey: bundleKey, AppKey: entry.ID, Name: entry.Owner + " component", Metadata: map[string]any{"hidden": true, "kind": "component"}}); err != nil {
		t.Fatal(err)
	}
	upload, err := service.PresignUpload(ctx, bundleKey, delivery.PresignUploadRequest{Filename: entry.ID + ".tar.gz", ContentType: "application/gzip", AppKey: entry.ID, Platform: "linux-x64", ReleaseVersion: entry.Version})
	if err != nil {
		t.Fatal(err)
	}
	archive, err := os.Open(filepath.Join(root, entry.ID+".tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, upload.UploadURL, archive)
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = entry.Size
	for key, value := range upload.RequiredHeaders {
		req.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("MinIO upload HTTP %d", response.StatusCode)
	}
	artifact, err := service.CommitArtifact(ctx, bundleKey, delivery.CommitArtifactRequest{Bucket: bucket, ObjectKey: upload.ObjectKey, OriginalFilename: entry.ID + ".tar.gz", ContentType: "application/gzip", AppKey: entry.ID, Platform: "linux-x64", ReleaseVersion: entry.Version, SHA256: entry.SHA256})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = catalog.UpsertAsset(ctx, delivery.Asset{BundleKey: bundleKey, AppKey: entry.ID, Platform: "linux-x64", ArtifactSource: "managed", ArtifactID: &artifact.ID, ReleaseVersion: entry.Version, Checksum: entry.SHA256, Metadata: map[string]any{"kind": "component", "hidden": true, "component": signed}}); err != nil {
		t.Fatal(err)
	}
	// The release must survive the actual PostgreSQL JSONB and catalog codec
	// boundary before any host is built. Signatures cover facts, not JSON object
	// member order introduced by a storage engine.
	assets, err := catalog.ComponentAssets(ctx, bundleKey)
	if err != nil || len(assets) != 1 {
		t.Fatalf("real component catalog read: count=%d error=%v", len(assets), err)
	}
	stored, err := delivery.ComponentEntry(assets[0])
	if err != nil {
		t.Fatal(err)
	}
	storedCanonical, err := stored.Entry.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	beforeRaw, _ := json.Marshal(signed.Entry)
	afterRaw, _ := json.Marshal(stored.Entry)
	var beforeFacts, afterFacts any
	_ = json.Unmarshal(beforeRaw, &beforeFacts)
	_ = json.Unmarshal(afterRaw, &afterFacts)
	t.Logf("PostgreSQL component roundtrip: facts_equal=%t signature_equal=%t canonical_before=%x canonical_after=%x", reflect.DeepEqual(beforeFacts, afterFacts), signed.Signature == stored.Signature, sha256.Sum256(canonical), sha256.Sum256(storedCanonical))
	if err = stored.Verify(string(publicPEM)); err != nil {
		t.Fatalf("component release signature after real PostgreSQL JSONB/catalog roundtrip: %v", err)
	}
	if os.Getenv("J11_VERIFY_CATALOG_ONLY") == "1" {
		return
	}
	router := http.NewServeMux()
	router.HandleFunc("/api/v1/components", deliveryhttp.Components(func() string { return bundleKey }, catalog, service))
	var expectedArtifacts atomic.Int64
	expectedArtifacts.Store(1)
	if fullApp != "" && signed.Entry.Owner == "audio-tools" {
		file, err := os.Open(fullApp)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		stat, err := file.Stat()
		if err != nil {
			t.Fatal(err)
		}
		h256, h512 := sha256.New(), sha512.New()
		if _, err = io.Copy(io.MultiWriter(h256, h512), file); err != nil {
			t.Fatal(err)
		}
		if _, err = file.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		if _, err = catalog.UpsertApp(delivery.App{BundleKey: bundleKey, AppKey: "audio-tools-full", Name: "Audio Tools", Metadata: map[string]any{"kind": "desktop"}}); err != nil {
			t.Fatal(err)
		}
		upload, err := service.PresignUpload(ctx, bundleKey, delivery.PresignUploadRequest{Filename: filepath.Base(fullApp), ContentType: "application/octet-stream", AppKey: "audio-tools-full", Platform: "linux-x64", ReleaseVersion: entry.Version})
		if err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, upload.UploadURL, file)
		if err != nil {
			t.Fatal(err)
		}
		req.ContentLength = stat.Size()
		for key, value := range upload.RequiredHeaders {
			req.Header.Set(key, value)
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("full app MinIO upload HTTP%d", response.StatusCode)
		}
		artifact, err := service.CommitArtifact(ctx, bundleKey, delivery.CommitArtifactRequest{Bucket: upload.Bucket, ObjectKey: upload.ObjectKey, OriginalFilename: filepath.Base(fullApp), ContentType: "application/octet-stream", AppKey: "audio-tools-full", Platform: "linux-x64", ReleaseVersion: entry.Version, SHA256: hex.EncodeToString(h256.Sum(nil)), SHA512: base64.StdEncoding.EncodeToString(h512.Sum(nil))})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = catalog.PromoteChannel(ctx, delivery.ChannelPromotionRequest{BundleKey: bundleKey, AppKey: "audio-tools-full", VariantKey: "default", ArtifactIDs: map[string]int64{"linux-x64": artifact.ID}}); err != nil {
			t.Fatal(err)
		}
		deps := updateDependencies(componentJourneyBundle(bundleKey), db)
		updates := mux.NewRouter()
		updates.HandleFunc("/api/v1/updates/{app_key}/{channel}/{file}", deliveryhttp.RequireUpdateAPIKey(deps, catalog)(deliveryhttp.UpdateFile(deps, catalog, service))).Methods("GET")
		router.Handle("/api/v1/updates/", updates)
		expectedArtifacts.Add(1)
	}
	// Publish a deliberately invalid newer artifact through the actual catalog.
	// The original MinIO bytes remain available; delivery must refuse the new
	// checksum, and the desktop must retain its previously verified generation.
	router.HandleFunc("/__journey/broken-update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		next := signed
		next.Entry.Version = "0.0.2"
		next.Entry.SHA256 = strings.Repeat("0", 64)
		var nextManifest map[string]any
		_ = json.Unmarshal(next.Entry.Manifest, &nextManifest)
		nextManifest["app"].(map[string]any)["version"] = next.Entry.Version
		next.Entry.Manifest, _ = json.Marshal(nextManifest)
		canonical, err := next.Entry.CanonicalBytes()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		sum := sha256.Sum256(canonical)
		signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		next.Signature.Signature = base64.StdEncoding.EncodeToString(signature)
		_, err = catalog.UpsertAsset(r.Context(), delivery.Asset{BundleKey: bundleKey, AppKey: entry.ID, Platform: "linux-x64", ArtifactSource: "managed", ArtifactID: &artifact.ID, ReleaseVersion: next.Entry.Version, Checksum: next.Entry.SHA256, Metadata: map[string]any{"kind": "component", "hidden": true, "component": next}})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	// Deliver a correctly packaged newer owner release whose real API cannot
	// start. This crosses the actual MinIO, artifact persistence, signed catalog
	// and desktop verification boundaries before exercising activation rollback.
	router.HandleFunc("/__journey/startup-failure-update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		variant := os.Getenv("J11_STARTUP_FAILURE_RELEASE_ROOT")
		if variant == "" {
			http.Error(w, "actual startup-failing owner release required", http.StatusBadRequest)
			return
		}
		raw, err := os.ReadFile(filepath.Join(variant, "entry.json"))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		next := signed
		if err = json.Unmarshal(raw, &next.Entry); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if next.Entry.ID != entry.ID || next.Entry.Owner != entry.Owner || next.Entry.Version != "0.0.3" {
			http.Error(w, "unexpected startup-failing owner identity", http.StatusBadRequest)
			return
		}
		next.Entry.ArtifactURL = base + "/api/v1/components?id=" + next.Entry.ID
		next.Entry.StandaloneURL = signed.Entry.StandaloneURL
		canonical, err := next.Entry.CanonicalBytes()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		sum := sha256.Sum256(canonical)
		signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		next.Signature.Signature = base64.StdEncoding.EncodeToString(signature)
		if err = next.Verify(string(publicPEM)); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		file, err := os.Open(filepath.Join(variant, next.Entry.ID+".tar.gz"))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer file.Close()
		upload, err := service.PresignUpload(r.Context(), bundleKey, delivery.PresignUploadRequest{Filename: next.Entry.ID + ".tar.gz", ContentType: "application/gzip", AppKey: next.Entry.ID, Platform: "linux-x64", ReleaseVersion: next.Entry.Version})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		req, err := http.NewRequestWithContext(r.Context(), http.MethodPut, upload.UploadURL, file)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		req.ContentLength = next.Entry.Size
		for key, value := range upload.RequiredHeaders {
			req.Header.Set(key, value)
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			http.Error(w, "startup-failing archive MinIO upload failed", http.StatusBadGateway)
			return
		}
		candidate, err := service.CommitArtifact(r.Context(), bundleKey, delivery.CommitArtifactRequest{Bucket: upload.Bucket, ObjectKey: upload.ObjectKey, OriginalFilename: next.Entry.ID + ".tar.gz", ContentType: "application/gzip", AppKey: next.Entry.ID, Platform: "linux-x64", ReleaseVersion: next.Entry.Version, SHA256: next.Entry.SHA256})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		expectedArtifacts.Add(1)
		if _, err = catalog.UpsertAsset(r.Context(), delivery.Asset{BundleKey: bundleKey, AppKey: next.Entry.ID, Platform: "linux-x64", ArtifactSource: "managed", ArtifactID: &candidate.ID, ReleaseVersion: next.Entry.Version, Checksum: next.Entry.SHA256, Metadata: map[string]any{"kind": "component", "hidden": true, "component": next}}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	done := make(chan struct{})
	var once sync.Once
	router.HandleFunc("/__journey/stop", func(w http.ResponseWriter, r *http.Request) { once.Do(func() { close(done) }); w.WriteHeader(200) })
	server := &http.Server{Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	})
	if ready := os.Getenv("J11_SERVER_READY"); ready != "" {
		if err = os.WriteFile(ready, []byte(base), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-done:
	case <-time.After(18 * time.Minute):
		t.Fatal("component journey server deadline")
	}
	rows, err := db.QueryContext(ctx, `SELECT COUNT(*) FROM download_artifacts WHERE bundle_key=$1`, bundleKey)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var count int
	if rows.Next() {
		_ = rows.Scan(&count)
	}
	if int64(count) != expectedArtifacts.Load() {
		t.Fatalf("real artifact persistence count %d", count)
	}
	_, _ = io.WriteString(os.Stdout, "real LPBS component artifact persistence confirmed\n")
}
