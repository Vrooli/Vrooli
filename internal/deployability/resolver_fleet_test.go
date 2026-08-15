package deployability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

type fleetResourceManifest struct {
	Name         string                     `json:"name"`
	Bundling     Bundling                   `json:"bundling"`
	Platforms    map[string]string          `json:"platforms"`
	Requirements *fleetResourceRequirements `json:"requirements"`
}

type fleetResourceRequirements struct {
	Class      string  `json:"class"`
	Weight     float64 `json:"weight"`
	RAMMB      float64 `json:"ram_mb"`
	DiskMB     float64 `json:"disk_mb"`
	CPUCores   float64 `json:"cpu_cores"`
	GPU        bool    `json:"gpu"`
	Network    string  `json:"network"`
	Source     string  `json:"source"`
	Confidence string  `json:"confidence"`
}

func TestResolveEnumeratesTheLiveResourceFleet(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	declarations, err := loadFleetResourceDeclarations(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(declarations) == 0 {
		t.Fatal("resource fleet enumeration returned no declarations")
	}

	for _, declaration := range declarations {
		for _, tier := range []DeliveryTier{TierLocal, TierDesktop, TierMobile, TierSaaS, TierEnterprise} {
			for _, hostOS := range []HostOS{HostOSLinux, HostOSMacOS, HostOSWindows} {
				result := Resolve(ResolutionInput{
					Target: TargetDeclaration{Name: "fleet-test", Dependencies: []DependencyDeclaration{declaration}},
					Tier:   tier,
					OS:     hostOS,
				})
				if result.Verdict == VerdictUnknown {
					t.Fatalf("resource %q produced unknown for tier=%s os=%s: %#v", declaration.Name, tier, hostOS, result.Reasons)
				}
			}
		}
	}
}

func TestResourceFleetEnumerationRejectsMissingRequirements(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "temporary-missing-requirements")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"name":"temporary-missing-requirements","bundling":"vendorable","platforms":{"linux":"supported","macos":"supported","windows":"supported"}}`)
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := loadFleetResourceDeclarations(root); err == nil {
		t.Fatal("resource fleet enumeration accepted a manifest without requirements")
	}
}

func loadFleetResourceDeclarations(repoRoot string) ([]DependencyDeclaration, error) {
	paths, err := filepath.Glob(filepath.Join(repoRoot, "resources", "*", "resource.json"))
	if err != nil {
		return nil, fmt.Errorf("glob resource manifests: %w", err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no resource manifests found under %s", repoRoot)
	}

	declarations := make([]DependencyDeclaration, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var manifest fleetResourceManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if strings.TrimSpace(manifest.Name) == "" {
			return nil, fmt.Errorf("%s has no resource name", path)
		}
		if manifest.Requirements == nil {
			return nil, fmt.Errorf("%s has no requirements", path)
		}
		if strings.TrimSpace(manifest.Requirements.Class) == "" || manifest.Requirements.Weight <= 0 {
			return nil, fmt.Errorf("%s has invalid requirements", path)
		}
		if strings.TrimSpace(manifest.Requirements.Source) == "" || strings.TrimSpace(manifest.Requirements.Confidence) == "" {
			return nil, fmt.Errorf("%s has requirements without source and confidence", path)
		}
		switch manifest.Bundling {
		case BundlingVendorable, BundlingHostRequired, BundlingProhibited:
		default:
			return nil, fmt.Errorf("%s has unknown bundling mode %q", path, manifest.Bundling)
		}
		platforms := make(map[HostOS]PlatformDeclaration, len(manifest.Platforms))
		for rawOS, status := range manifest.Platforms {
			hostOS, ok := normalizeFleetHostOS(rawOS)
			if !ok {
				return nil, fmt.Errorf("%s declares unknown host OS %q", path, rawOS)
			}
			platforms[hostOS] = PlatformDeclaration{Status: status}
		}
		if len(platforms) != 3 {
			return nil, fmt.Errorf("%s must declare linux, macos, and windows", path)
		}
		declarations = append(declarations, DependencyDeclaration{
			Kind:            "resource",
			Name:            manifest.Name,
			Required:        true,
			Bundling:        manifest.Bundling,
			Present:         true,
			Artifact:        true,
			PlatformSupport: platforms,
			Requirements: &ResourceRequirements{
				Class: manifest.Requirements.Class, Weight: manifest.Requirements.Weight,
				RAMMB: manifest.Requirements.RAMMB, DiskMB: manifest.Requirements.DiskMB,
				CPUCores: manifest.Requirements.CPUCores, GPU: manifest.Requirements.GPU,
				Network: manifest.Requirements.Network, Source: manifest.Requirements.Source,
				Confidence: manifest.Requirements.Confidence,
			},
		})
	}
	return declarations, nil
}

func normalizeFleetHostOS(value string) (HostOS, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "linux":
		return HostOSLinux, true
	case "macos", "darwin":
		return HostOSMacOS, true
	case "windows", "win32":
		return HostOSWindows, true
	default:
		return "", false
	}
}
