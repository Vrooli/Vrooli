package installgateway

import (
	"context"
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
			name:       "host tool has no full closure probe",
			resolution: Resolution{SurfaceRoot: root, PackageManager: "npm", HostTool: true, Profile: InstallProfile{LifecycleMode: "scripts-disabled-by-default"}},
			wantReason: "no bounded package size probe",
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

func TestPlanDiskAdmissionRefusesNonemptyNpmDependencyClosure(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"fixture","dependencies":{"old":"1.0.0"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(`{"lockfileVersion":3,"packages":{"":{"dependencies":{"old":"1.0.0"}},"node_modules/old":{"version":"1.0.0"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanDiskAdmission(context.Background(), Resolution{
		SurfaceRoot: root, ManifestPath: filepath.Join(root, "package.json"), PackageManager: "npm",
		PackageName: "new-leaf", Argv: []string{"npm", "install", "--ignore-scripts", "new-leaf@1.2.3"},
		Profile: InstallProfile{LifecycleMode: "scripts-disabled-by-default"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.Unsupported, "already declares dependencies") || len(plan.Paths) != 0 {
		t.Fatalf("plan = %+v, want fail-closed for existing npm dependency closure", plan)
	}
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
