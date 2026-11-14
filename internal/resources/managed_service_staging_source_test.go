package resources

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/operatorstate"
)

func TestManagedStagingSourceSwitchCannotEvaluateAdoptedExecutable(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("host source fixture is linux-amd64")
	}
	controller, manifest, writer, host, marker := managedSourceRaceFixture(t)
	checked, resume := make(chan struct{}), make(chan struct{})
	var reads atomic.Int32
	statePath, err := writer.Path(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	controller.operatorState = operatorstate.New(operatorstate.Config{RepoRoot: controller.Root, StatePath: func(ctx context.Context) (string, error) {
		if reads.Add(1) == 2 {
			close(checked)
			select {
			case <-resume:
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		return statePath, nil
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := controller.StageManagedServiceArtifact(ctx, manifest.Name, manifest.ManagedService.Artifact.Version)
		done <- err
	}()
	select {
	case <-checked:
	case err := <-done:
		t.Fatalf("staging stopped before source recheck: %v", err)
	case <-ctx.Done():
		t.Fatal("managed staging did not reach source recheck")
	}
	switchRaceSourceToHost(t, writer, manifest.Name, host)
	close(resume)
	if err := <-done; err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("source switch did not invalidate managed staging: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("managed staging evaluated adopted host executable: %v", err)
	}
	document, err := writer.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if document.Resources[manifest.Name].Source.Mode != "host" {
		t.Fatal("managed staging reset the concurrent source choice")
	}
}

func TestManagedStagingSourceSwitchDuringDownloadCannotClaimHostArtifact(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("host source fixture is linux-amd64")
	}
	controller, manifest, writer, host, marker := managedSourceRaceFixture(t)
	body, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	downloading, resume := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(downloading)
		select {
		case <-resume:
			_, _ = w.Write(body)
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	manifest.ManagedService.Acquisition.Targets[0].URL = server.URL
	writeResourceManifest(t, controller.Root, manifest.Name, manifest)
	hostBefore, err := os.ReadFile(host)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := controller.StageManagedServiceArtifact(ctx, manifest.Name, manifest.ManagedService.Artifact.Version)
		done <- err
	}()
	select {
	case <-downloading:
	case err := <-done:
		t.Fatalf("staging stopped before download: %v", err)
	case <-ctx.Done():
		t.Fatal("managed staging did not reach the fixture download")
	}
	switchRaceSourceToHost(t, writer, manifest.Name, host)
	close(resume)
	if err := <-done; err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("changed source produced a successful managed claim: %v", err)
	}
	after, err := os.ReadFile(host)
	if err != nil || string(after) != string(hostBefore) {
		t.Fatal("managed staging changed the adopted host file")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("managed staging evaluated adopted host executable: %v", err)
	}
	document, err := writer.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if document.Resources[manifest.Name].Source.Mode != "host" {
		t.Fatal("managed staging reset concurrent source selection")
	}
}

func managedSourceRaceFixture(t *testing.T) (*Controller, ResourceManifest, *operatorstate.Service, string, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "runtime"))
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", filepath.Join(root, "artifacts"))
	body, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	manifest := managedServiceTestManifest(fmt.Sprintf("%x", sha256.Sum256(body)))
	manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: "https://example.invalid/no-acquisition-expected", SHA256: manifest.ManagedService.Artifact.SHA256, Layout: "file", Archive: "none"}}}
	manifest.ManagedService.HostToolExecutable = "fixture-openrgb"
	manifest.ManagedService.HostToolVersion = manifest.ManagedService.Artifact.Version
	manifest.ManagedService.SourceSelectionRequired = true
	manifest.ManagedService.ProviderPolicy.AllowedModes = manifest.ManagedService.ProviderPolicy.AllowedModes[:1]
	manifest.ManagedService.ProviderPolicy.ExternalAccessCapabilities = nil
	writeResourceManifest(t, root, manifest.Name, manifest)
	copyOperatorStateSchemaForTest(t, root)
	writer := operatorstate.New(operatorstate.Config{RepoRoot: root, StateRoot: filepath.Join(root, "operator")})
	patch := fmt.Sprintf(`{"resources":{"%s":{"source":{"mode":"managed"}}}}`, manifest.Name)
	if _, err := writer.Apply(context.Background(), []byte(patch)); err != nil {
		t.Fatal(err)
	}
	controller := NewController(root, filepath.Join(root, "home"))
	controller.operatorState = writer
	marker := filepath.Join(root, "host-was-evaluated")
	t.Setenv("VROOLI_MANAGED_SOURCE_RACE_MARKER", marker)
	host := filepath.Join(root, "adopted-openrgb")
	script := "#!/bin/sh\nprintf touched > \"$VROOLI_MANAGED_SOURCE_RACE_MARKER\"\nprintf 'OpenRGB " + manifest.ManagedService.HostToolVersion + "\\n'\n"
	if err := os.WriteFile(host, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return controller, manifest, writer, host, marker
}

func switchRaceSourceToHost(t *testing.T, writer *operatorstate.Service, name, host string) {
	t.Helper()
	digest, err := managedFileSHA256(host)
	if err != nil {
		t.Fatal(err)
	}
	patch := fmt.Sprintf(`{"resources":{"%s":{"source":{"mode":"host","executable":%q,"version":"1.0.0","sha256":"%s"}}}}`, name, host, digest)
	if _, err := writer.Apply(context.Background(), []byte(patch)); err != nil {
		t.Fatal(err)
	}
}
