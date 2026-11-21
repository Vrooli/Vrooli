//go:build integration

package main

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	updatehttp "landing-page-business-suite-api/handlers/delivery"
	"landing-page-business-suite-api/internal/delivery"
)

type desktopJourneyBundle struct{}

func (desktopJourneyBundle) BundleKey() string { return "desktop-journey" }

// This fixture runs the production LPBS delivery handlers, PostgreSQL store and
// S3 provider against isolated PostgreSQL and digest-verified native MinIO. No live channel
// or operator storage is used. The desktop journey owns its process lifetime.
func TestDesktopUpdateJourneyServer(t *testing.T) {
	artifactPath := os.Getenv("J10_NEXT_APPIMAGE")
	if artifactPath == "" {
		t.Skip("J10_NEXT_APPIMAGE selects an actual generated artifact")
	}
	db := setupTestDB(t)
	endpoint, access, secret := setupDesktopJourneyMinIO(t)
	ctx := t.Context()
	const bundle = "desktop-journey"
	const bucket = "desktop-journey-downloads"
	createTestBucket(t, ctx, endpoint, access, secret, bucket)
	service := delivery.NewService(db, integrationS3Provider(access, secret))
	provider, region, pathStyle := "s3", "us-east-1", true
	bucketName := bucket
	if _, err := service.SaveSettings(ctx, bundle, delivery.StorageSettingsUpdate{Provider: &provider, Bucket: &bucketName, Region: &region, Endpoint: &endpoint, ForcePathStyle: &pathStyle}); err != nil {
		t.Fatal(err)
	}
	catalog := delivery.NewCatalogService(db)
	if _, err := catalog.UpsertApp(delivery.App{BundleKey: bundle, AppKey: "web-console", Name: "Aquila journey"}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	h256, h512 := sha256.New(), sha512.New()
	if _, err := io.Copy(io.MultiWriter(h256, h512), f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	upload, err := service.PresignUpload(ctx, bundle, delivery.PresignUploadRequest{Filename: filepath.Base(artifactPath), ContentType: "application/octet-stream", AppKey: "web-console", Platform: "linux-x64", ReleaseVersion: "0.0.2"})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, upload.UploadURL, f)
	if err != nil {
		t.Fatal(err)
	}
	request.ContentLength = stat.Size()
	for key, value := range upload.RequiredHeaders {
		request.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("real MinIO upload HTTP %d", response.StatusCode)
	}
	artifact, err := service.CommitArtifact(ctx, bundle, delivery.CommitArtifactRequest{Bucket: upload.Bucket, ObjectKey: upload.ObjectKey, OriginalFilename: filepath.Base(artifactPath), ContentType: "application/octet-stream", AppKey: "web-console", Platform: "linux-x64", ReleaseVersion: "0.0.2", SHA256: hex.EncodeToString(h256.Sum(nil)), SHA512: base64.StdEncoding.EncodeToString(h512.Sum(nil))})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.PromoteChannel(ctx, delivery.ChannelPromotionRequest{BundleKey: bundle, AppKey: "web-console", VariantKey: "default", ArtifactIDs: map[string]int64{"linux-x64": artifact.ID}}); err != nil {
		t.Fatal(err)
	}
	deps := updateDependencies(desktopJourneyBundle{}, db)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/updates/{app_key}/{channel}/{file}", updatehttp.RequireUpdateAPIKey(deps, catalog)(updatehttp.UpdateFile(deps, catalog, service))).Methods("GET")
	done := make(chan struct{})
	var once sync.Once
	router.HandleFunc("/__journey/stop", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); once.Do(func() { close(done) }) }).Methods("POST")
	router.HandleFunc("/__journey/evidence", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), `SELECT event_type,COUNT(*) FROM delivery_events WHERE bundle_key=$1 GROUP BY event_type`, bundle)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		counts := map[string]int{}
		for rows.Next() {
			var key string
			var count int
			if err := rows.Scan(&key, &count); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			counts[key] = count
		}
		json.NewEncoder(w).Encode(counts)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:39565")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	})
	if ready := os.Getenv("J10_SERVER_READY"); ready != "" {
		if err := os.WriteFile(ready, []byte("ready\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-done:
	case <-time.After(15 * time.Minute):
		t.Fatal("desktop journey did not stop its LPBS server")
	}
}

// Use the same staged native artifact and ephemeral lifecycle as the resource
// owner's managed-service integration tests. Verify the resource-owned digest
// before launch; never install or reconfigure an operator's MinIO instance.
func setupDesktopJourneyMinIO(t *testing.T) (string, string, string) {
	t.Helper()
	var spec struct {
		ManagedService struct {
			Artifact struct {
				Version string
				SHA     map[string]string `json:"sha256_by_platform"`
			}
		} `json:"managed_service"`
	}
	data, err := os.ReadFile("../../../resources/minio/resource.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(home, ".vrooli", "artifacts", "minio", spec.ManagedService.Artifact.Version, "minio_linux_amd64")
	f, err := os.Open(binary)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	_, err = io.Copy(digest, f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(digest.Sum(nil)) != spec.ManagedService.Artifact.SHA["linux-amd64"] {
		t.Fatal("staged MinIO does not match the resource-owned digest")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	private := t.TempDir()
	log, err := os.Create(filepath.Join(private, "minio.log"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "server", "--address", address, filepath.Join(private, "data"), "--console-address", "127.0.0.1:0")
	command.Env = append(os.Environ(), "MINIO_ROOT_USER=minioadmin", "MINIO_ROOT_PASSWORD=minioadmin", "MINIO_BROWSER=off")
	command.Stdout = log
	command.Stderr = log
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = command.Process.Kill(); _ = command.Wait(); log.Close() })
	endpoint := "http://" + address
	client := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(endpoint + "/minio/health/ready")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == 200 {
				return endpoint, "minioadmin", "minioadmin"
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	output, _ := os.ReadFile(log.Name())
	t.Fatal(fmt.Sprintf("isolated MinIO did not become ready: %s", output))
	return "", "", ""
}
