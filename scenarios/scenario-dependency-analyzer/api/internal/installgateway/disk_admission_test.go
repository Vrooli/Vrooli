package installgateway

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanDiskAdmissionRefusesUnboundedInstallProfiles(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name       string
		resolution Resolution
		wantReason string
	}{
		{
			name:       "generic pip has no surface-scoped environment",
			resolution: Resolution{SurfaceRoot: root, PackageManager: "pip", Profile: InstallProfile{LifecycleMode: "native-governed"}},
			wantReason: "surface-scoped environment",
		},
		{
			name:       "host tool with incomplete frozen identity is refused",
			resolution: Resolution{SurfaceRoot: root, PackageManager: "npm", HostTool: true, Profile: InstallProfile{LifecycleMode: "scripts-disabled-by-default"}},
			wantReason: "frozen npm host-tool sizing",
		},
		{
			name:       "managed artifact owner path is not exposed",
			resolution: Resolution{SurfaceRoot: root, PackageManager: "native-binaryfetch", Profile: InstallProfile{LifecycleMode: "managed-service-artifact"}},
			wantReason: "no bounded size probe",
		},
		{
			name:       "pinned binary archive and cache are not bounded",
			resolution: Resolution{SurfaceRoot: root, PackageManager: "native-binaryfetch", Profile: InstallProfile{LifecycleMode: "pinned-release"}},
			wantReason: "no bounded size probe",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := PlanDiskAdmission(context.Background(), tc.resolution)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(plan.Unsupported, tc.wantReason) || len(plan.Paths) != 0 {
				t.Fatalf("plan = %+v, want fail-closed reason containing %q", plan, tc.wantReason)
			}
		})
	}
}

func TestProbePnpmArchiveSizeUsesBoundedRangeFallback(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/package.tgz", http.StatusTemporaryRedirect)
			return
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if got := r.Header.Get("Range"); got != "bytes=0-0" {
			t.Errorf("Range header = %q, want bytes=0-0", got)
		}
		w.Header().Set("Content-Range", "bytes 0-0/123456")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte{0})
	}))
	defer server.Close()

	size, err := probePnpmArchiveSize(context.Background(), noRedirectClient(server.Client()), server.URL+"/redirect", server.Listener.Addr().String(), func() error { return nil }, func(int) error { return nil })
	if err != nil || size != 123456 {
		t.Fatalf("bounded range probe size=%d err=%v, want size 123456", size, err)
	}
}

func TestProbePnpmArchiveSizeRefusesIgnoredRangeAndOffRegistryRedirect(t *testing.T) {
	t.Run("ignored range", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			_, _ = w.Write([]byte("full archive"))
		}))
		defer server.Close()
		_, err := probePnpmArchiveSize(context.Background(), noRedirectClient(server.Client()), server.URL+"/package.tgz", server.Listener.Addr().String(), func() error { return nil }, func(int) error { return nil })
		if err == nil || !strings.Contains(err.Error(), "did not honor") {
			t.Fatalf("ignored range error = %v, want bounded fail-closed refusal", err)
		}
	})

	t.Run("off registry redirect", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://registry.attacker.example/package.tgz", http.StatusFound)
		}))
		defer server.Close()
		_, err := probePnpmArchiveSize(context.Background(), noRedirectClient(server.Client()), server.URL+"/package.tgz", server.Listener.Addr().String(), func() error { return nil }, func(int) error { return nil })
		if err == nil || !strings.Contains(err.Error(), "redirected outside") {
			t.Fatalf("off-registry redirect error = %v, want registry refusal", err)
		}
	})
}

func TestProbePnpmArchiveContentMeasuresAndVerifiesBoundedTarball(t *testing.T) {
	archive := testPnpmArchive(t, "payload", "drawer action labels")
	digest := sha512.Sum512(archive)
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(digest[:])
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("request method = %s, want GET", r.Method)
		}
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	client := noRedirectClient(server.Client())
	got, err := probePnpmArchiveContent(context.Background(), client, server.URL+"/package.tgz", server.Listener.Addr().String(), integrity)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, archive) {
		t.Fatal("archive probe changed bytes")
	}
	size, err := measurePnpmArchive(got)
	if err != nil {
		t.Fatal(err)
	}
	if size != int64(len("drawer action labels")) {
		t.Fatalf("unpacked size = %d, want %d", size, len("drawer action labels"))
	}
	if _, err := probePnpmArchiveContent(context.Background(), client, server.URL+"/package.tgz", server.Listener.Addr().String(), "sha512-invalid"); err == nil {
		t.Fatal("archive with mismatched integrity was accepted")
	}
}

func testPnpmArchive(t *testing.T, name, content string) []byte {
	t.Helper()
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gz)
	if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

func noRedirectClient(client *http.Client) *http.Client {
	copy := *client
	copy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return &copy
}

func TestPlanDiskAdmissionBoundsApprovedLocalYtDlpWheel(t *testing.T) {
	previous := ytDlpWheelMetadataProbe
	ytDlpWheelMetadataProbe = func(context.Context) (ytDlpWheelMetadata, string, error) {
		return ytDlpWheelMetadata{URL: "https://files.pythonhosted.org/packages/yt-dlp.whl", SHA256: strings.Repeat("a", 64), Bytes: 8 << 20}, "/usr/bin/python3", nil
	}
	t.Cleanup(func() { ytDlpWheelMetadataProbe = previous })

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "requirements.txt"), []byte("# local research target\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanDiskAdmission(context.Background(), Resolution{
		SurfaceRoot: root, ManifestPath: filepath.Join(root, "requirements.txt"), PackageManager: "pip-local-wheel",
		PackageName: "yt-dlp", ExactVersion: true, Argv: []string{"pip-local-wheel", "install", "yt-dlp==2026.10.26"},
		Profile: InstallProfile{LifecycleMode: "native-governed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Unsupported != "" || plan.PythonWheelBytes != 8<<20 || plan.PythonWheelSHA256 != strings.Repeat("a", 64) || len(plan.Paths) != 2 {
		t.Fatalf("bounded local wheel plan = %+v", plan)
	}
	var total int64
	for _, path := range plan.Paths {
		if !filepath.IsAbs(path.Path) || path.PeakBytes <= 0 {
			t.Fatalf("invalid disk path requirement: %+v", path)
		}
		total += path.PeakBytes
	}
	if total > DiskJobPlanningCeilingBytes {
		t.Fatalf("plan peak %d exceeds the per-job ceiling", total)
	}
}

func TestResolveUsesSurfaceScopedYtDlpWheelOnlyForApprovedPin(t *testing.T) {
	repo := t.TempDir()
	root := filepath.Join(repo, "scenarios", "demo", "api")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "requirements.txt"), []byte("# research surface\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := Resolve(repo, "demo", "api", "pip", "yt-dlp", "2026.10.26")
	if err != nil {
		t.Fatal(err)
	}
	if r.PackageManager != "pip-local-wheel" || !r.ExactVersion || !strings.Contains(strings.Join(r.Argv, " "), "--no-deps") || !strings.Contains(r.Command(), ".sda-python-packages") {
		t.Fatalf("approved local wheel route = %+v", r)
	}

	generic, err := Resolve(repo, "demo", "api", "pip", "another-package", "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if generic.PackageManager != "pip" {
		t.Fatalf("generic pip install escaped its denied profile: %+v", generic)
	}

	unapproved, err := Resolve(repo, "demo", "api", "pip", "yt-dlp", "2026.10.27")
	if err != nil {
		t.Fatal(err)
	}
	if unapproved.PackageManager != "pip" {
		t.Fatalf("unapproved yt-dlp pin unexpectedly resolved to the local wheel route: %+v", unapproved)
	}
}

func TestPlanDiskAdmissionAdmitsTemplateDeclarationsWithALockOnlyAllowance(t *testing.T) {
	// Template installs regenerate a lockfile in a temporary scenario-shaped
	// surface and install nothing; they were refused for lack of a size probe.
	for _, manager := range []string{"pnpm", "npm"} {
		repo := t.TempDir()
		root := filepath.Join(repo, "scenarios", "scenario-to-desktop", "templates", "vanilla")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		plan, err := PlanDiskAdmission(context.Background(), Resolution{
			RepositoryRoot: repo, SurfaceRoot: root, ManifestPath: filepath.Join(root, "package.json"), PackageManager: manager,
			PackageName: "electron", TemplateVersion: "42.0.0", ExactVersion: true,
			Argv:    []string{manager, "install", "--ignore-scripts", "electron@42.0.0"},
			Profile: InstallProfile{LifecycleMode: "scripts-disabled-by-default"},
		})
		if err != nil || plan.Unsupported != "" || plan.JSStage != nil {
			t.Fatalf("%s template plan = %+v, %v", manager, plan, err)
		}
		var staging, total int64
		for _, path := range plan.Paths {
			total += path.PeakBytes
			if path.Role == "template-install-staging-and-cache" {
				staging = path.PeakBytes
			}
		}
		if staging < templateLockOnlyBytes || total > DiskJobPlanningCeilingBytes {
			t.Fatalf("%s template reservation staging=%d total=%d", manager, staging, total)
		}
	}
}
