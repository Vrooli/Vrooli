package resources

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/credentialauthority"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/securestore"
	"github.com/vrooli/vrooli/internal/testenv"
)

type gatedRoundTrip func(*http.Request) (*http.Response, error)

func (f gatedRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func gatedManifest() ResourceManifest {
	return ResourceManifest{Name: "gated-fixture", Driver: "managed-service", Credentials: manifestpkg.ResourceCredentials{Descriptors: []manifestpkg.CredentialDescriptor{{LogicalID: "vrooli/gated-fixture", Field: "token"}}}}
}
func TestAcquisitionCredentialMissingOrUndeclaredMakesNoRequests(t *testing.T) {
	for _, kind := range []string{"missing", "unavailable", "undeclared"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			old := binaryfetch.HTTPClient
			binaryfetch.HTTPClient = &http.Client{Transport: gatedRoundTrip(func(*http.Request) (*http.Response, error) { calls++; return nil, fmt.Errorf("unexpected download") })}
			t.Cleanup(func() { binaryfetch.HTTPClient = old })
			var store securestore.Store = testenv.NewCredentialStore(securestore.ErrNotFound)
			if kind == "unavailable" {
				store = securestore.Unavailable("private backend failure")
			}
			withCredentialStore(t, store)
			manifest := gatedManifest()
			if kind == "undeclared" {
				manifest.Credentials = manifestpkg.ResourceCredentials{}
			}
			_, err := resolveAcquisitionAuthorization(context.Background(), manifest, binaryfetch.AcquisitionTarget{URL: "https://origin.test/data", Auth: &binaryfetch.AcquisitionAuth{Scheme: "bearer", LogicalID: "vrooli/gated-fixture", Field: "token"}})
			if err == nil || calls != 0 {
				t.Fatalf("missing credential allowed request: %v calls%d", err, calls)
			}
		})
	}
}

func TestResourceInspectDataUsesDeclaredCredentialAndNoRuntime(t *testing.T) {
	authority := withCredentialStore(t, testenv.NewCredentialStore(securestore.ErrNotFound))
	if err := authority.Put("vrooli/gated-fixture", "token", "secret-local-test"); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	manifest := testresource.ResourceManifest("gated-fixture", testresource.WithResourceDriver("managed-service"))
	manifest.Credentials = gatedManifest().Credentials

	manifest.ManagedService.DataArtifacts = []resourcedeployment.ManagedServiceDataArtifact{{Name: "config", Path: "config.json", Acquisition: &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: "https://origin.test/data", Auth: &binaryfetch.AcquisitionAuth{Scheme: "bearer", LogicalID: "vrooli/gated-fixture", Field: "token"}}}}}}
	testresource.WriteResourceManifest(t, root, manifest.Name, manifest)
	if _, err := manifestpkg.Load(filepath.Join(root, "resources", manifest.Name, "resource.json")); err == nil {
		t.Fatal("unresolved digest admitted normal manifest/install path")
	}

	body := `{"generation":1}`
	git := fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("blob %d%c%s", len(body), 0, body))))
	calls := 0
	old := binaryfetch.HTTPClient
	binaryfetch.HTTPClient = &http.Client{Transport: gatedRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "Bearer secret-local-test" {
			t.Error("wrong credential")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	t.Cleanup(func() { binaryfetch.HTTPClient = old })
	out := filepath.Join(t.TempDir(), "config.json")
	req := ResourceDataInspection{URL: "https://origin.test/data", GitBlobSHA1: git, MaxBytes: 1024, Destination: out, LogicalID: "vrooli/gated-fixture", Field: "token"}
	receipt, err := NewController(root, t.TempDir()).InspectData(context.Background(), manifest.Name, req)
	if err != nil || receipt.Bytes != int64(len(body)) || calls != 1 {
		t.Fatalf("inspection %+v %v calls%d", receipt, err, calls)
	}
	got, _ := os.ReadFile(out)
	if string(got) != body {
		t.Fatal("wrong data")
	}
}

func TestManagedDataDownloadUsesAuthAndAcceptsTinyVerifiedJSON(t *testing.T) {
	authority := withCredentialStore(t, testenv.NewCredentialStore(securestore.ErrNotFound))
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "runtime"))
	body := []byte(`{"generation":1}`)
	digest := fmt.Sprintf("%x", sha256.Sum256(body))
	manifest := gatedManifest()
	manifest.ManagedService = &resourcedeployment.ManagedService{DataArtifacts: []resourcedeployment.ManagedServiceDataArtifact{{Name: "config", Path: "models/config.json", Acquisition: &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: "https://origin.test/data", SHA256: digest, Auth: &binaryfetch.AcquisitionAuth{Scheme: "bearer", LogicalID: "vrooli/gated-fixture", Field: "token"}}}}}}}
	calls := 0
	old := binaryfetch.HTTPClient
	binaryfetch.HTTPClient = &http.Client{Transport: gatedRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "Bearer secret-local-test" {
			t.Error("missing bearer")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}
	t.Cleanup(func() { binaryfetch.HTTPClient = old })
	controller := NewController(root, t.TempDir())

	if err := ensureManagedServiceDataArtifacts(context.Background(), controller, manifest); err == nil || calls != 0 {
		t.Fatalf("missing credential downloaded data: %v calls%d", err, calls)
	}
	if err := authority.Put("vrooli/gated-fixture", "token", "secret-local-test"); err != nil {
		t.Fatal(err)
	}
	if err := ensureManagedServiceDataArtifacts(context.Background(), controller, manifest); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "runtime/data/vrooli/resources", manifest.Name, "models/config.json")
	got, err := os.ReadFile(file)
	if err != nil || string(got) != string(body) || calls != 1 {
		t.Fatalf("wrong data %q %v calls%d", got, err, calls)
	}
	// Existing verified bytes require neither token availability nor another request.
	withCredentialStore(t, securestore.Unavailable("unavailable after acquisition"))
	if err := ensureManagedServiceDataArtifacts(context.Background(), controller, manifest); err != nil || calls != 1 {
		t.Fatalf("verified data tried reacquisition %v calls%d", err, calls)
	}
}

func TestInspectDataRequiresExactDeclaredURLBeforeCredentialResolution(t *testing.T) {
	authority := withCredentialStore(t, testenv.NewCredentialStore(securestore.ErrNotFound))
	if err := authority.Put("vrooli/gated-fixture", "token", "secret-local-test"); err != nil {
		t.Fatal(err)
	}
	resolves := 0
	original := credentialauthority.DefaultAuthority
	credentialauthority.DefaultAuthority = func() (*credentialauthority.Authority, error) { resolves++; return original() }
	t.Cleanup(func() { credentialauthority.DefaultAuthority = original })
	root := t.TempDir()
	manifest := testresource.ResourceManifest("gated-fixture", testresource.WithResourceDriver("managed-service"))
	manifest.Credentials = gatedManifest().Credentials
	auth := &binaryfetch.AcquisitionAuth{Scheme: "bearer", LogicalID: "vrooli/gated-fixture", Field: "token"}
	manifest.ManagedService.DataArtifacts = []resourcedeployment.ManagedServiceDataArtifact{{Name: "config", Path: "config.json", Acquisition: &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: "https://origin.test/data", SHA256: strings.Repeat("a", 64), Auth: auth}}}}}
	testresource.WriteResourceManifest(t, root, manifest.Name, manifest)
	calls := 0
	old := binaryfetch.HTTPClient
	binaryfetch.HTTPClient = &http.Client{Transport: gatedRoundTrip(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("valid")), Header: http.Header{}}, nil
	})}
	t.Cleanup(func() { binaryfetch.HTTPClient = old })
	git := fmt.Sprintf("%x", sha1.Sum([]byte("blob 5\x00valid")))
	for _, kind := range []string{"wrong-origin", "wrong-path", "wrong-reference"} {
		t.Run(kind, func(t *testing.T) {
			req := ResourceDataInspection{URL: "https://origin.test/data", GitBlobSHA1: git, MaxBytes: 1024, Destination: filepath.Join(t.TempDir(), "out"), LogicalID: auth.LogicalID, Field: auth.Field}
			switch kind {
			case "wrong-origin":
				req.URL = "https://evil.test/data"
			case "wrong-path":
				req.URL = "https://origin.test/other"
			case "wrong-reference":
				req.Field = "other"
			}
			if _, err := NewController(root, t.TempDir()).InspectData(context.Background(), manifest.Name, req); err == nil || resolves != 0 || calls != 0 {
				t.Fatalf("undeclared URL/ref reached credentials/network: %v resolves%d calls%d", err, resolves, calls)
			}
		})
	}
}
