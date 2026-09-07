package securestore

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCopyStoreS3UploadsEncryptedObjectAndReceipt(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "secrets.enc.json")
	newDrillEncryptedStore(t, source, "copy-passphrase")
	want, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	var gotPath string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		if r.Method == http.MethodPut {
			gotBody = body
		} else {
			_, _ = w.Write(gotBody)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	status, err := CopyStoreS3(source, "s3://bucket/recovery", filepath.Join(root, "state", "receipt.json"), S3CopyOptions{
		Region:      serverRegion,
		Endpoint:    server.URL,
		Credentials: ObjectStoreCredentials{AccessKey: "access", SecretKey: "secret"},
		Now:         func() time.Time { return time.Date(2026, 8, 18, 20, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("CopyStoreS3: %v", err)
	}
	if gotPath != "/bucket/recovery/secrets.enc.json" {
		t.Fatalf("request path = %q", gotPath)
	}
	if !bytes.Equal(gotBody, want) {
		t.Fatal("object body differs from encrypted store")
	}
	if status.Path != "s3://bucket/recovery/secrets.enc.json" || status.Generation == "" {
		t.Fatalf("unexpected status: %+v", status)
	}
	if _, err := os.Stat(filepath.Join(root, "state", "receipt.json")); err != nil {
		t.Fatalf("receipt: %v", err)
	}
}

func TestUploadS3ArtifactReadsBackArbitraryEncryptedArtifact(t *testing.T) {
	var stored []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			stored, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write(stored)
	}))
	defer server.Close()
	want := []byte(`{"version":1,"ciphertext":"opaque"}`)
	path, checksum, err := UploadS3Artifact("s3://bucket/escrow", "recovery/credentials.bundle.json", want, S3CopyOptions{Region: serverRegion, Endpoint: server.URL, Credentials: ObjectStoreCredentials{AccessKey: "access", SecretKey: "secret"}, Now: func() time.Time { return time.Date(2026, 8, 18, 20, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatal(err)
	}
	if path != "s3://bucket/escrow/recovery/credentials.bundle.json" || checksum == "" || !bytes.Equal(stored, want) {
		t.Fatalf("path=%q checksum=%q stored=%q", path, checksum, stored)
	}
}

func TestS3ObjectURLRejectsNonS3Sink(t *testing.T) {
	if _, _, err := s3ObjectURL("/tmp/recovery", "us-east-1", ""); err == nil || !strings.Contains(err.Error(), "s3://") {
		t.Fatalf("s3ObjectURL error = %v", err)
	}
}

func TestCopyStoreS3RejectsSinkInsideRegisteredRepository(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "secrets.enc.json")
	newDrillEncryptedStore(t, source, "copy-passphrase")

	_, err := CopyStoreS3(source, "s3://bucket/recovery/root-copy", filepath.Join(root, "receipt.json"), S3CopyOptions{
		Region:          serverRegion,
		Credentials:     ObjectStoreCredentials{AccessKey: "access", SecretKey: "secret"},
		RepositorySinks: []string{"s3://bucket/recovery"},
	})
	var conflict *SinkConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("CopyStoreS3 error = %v, want SinkConflictError", err)
	}
}

func TestCopyStoreS3AllowsSiblingOrParentSink(t *testing.T) {
	for _, sink := range []string{"s3://bucket/other", "s3://bucket"} {
		if err := rejectS3RepositoryContainment(sink, []string{"s3://bucket/recovery"}); err != nil {
			t.Fatalf("rejectS3RepositoryContainment(%q): %v", sink, err)
		}
	}
}

func TestCanonicalS3HeadersHaveNoBlankLineBeforeSignedNames(t *testing.T) {
	parsed, err := url.Parse("https://objects.example/bucket/secrets.enc.json")
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPut, parsed.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Amz-Content-Sha256", "payload")
	req.Header.Set("X-Amz-Date", "20260818T200000Z")
	canonical, signed := canonicalS3Headers(req)
	if strings.HasSuffix(canonical, "\n") || strings.Contains(canonical, "\n\n") {
		t.Fatalf("canonical headers contain an extra blank line: %q", canonical)
	}
	if signed != "host;x-amz-content-sha256;x-amz-date" {
		t.Fatalf("signed headers = %q", signed)
	}
}

const serverRegion = "us-east-1"
