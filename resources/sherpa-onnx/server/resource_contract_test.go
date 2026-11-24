package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Keep the managed data-group contract aligned with the paths consumed by the
// server. This catches a subtle failure mode where acquisition succeeds but
// startup silently disables STT because a model was installed elsewhere.
func TestManagedStreamingProfileMatchesServerPaths(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	manifestPath := filepath.Join(filepath.Dir(source), "..", "resource.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		ManagedService struct {
			ArtifactRole string `json:"artifact_role"`
			Acquisition  struct {
				Kind    string `json:"kind"`
				Targets []struct {
					Unsupported string `json:"unsupported"`
				} `json:"targets"`
			} `json:"acquisition"`
			Environment   map[string]string `json:"environment"`
			DataArtifacts []struct {
				Name  string `json:"name"`
				Path  string `json:"path"`
				Group string `json:"group"`
			} `json:"data_artifacts"`
		} `json:"managed_service"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	// A locally built adapter and upstream model URLs do not make a managed
	// release. Keep acquisition disabled until the release owner signs and
	// publishes a target artifact that the runtime can verify.
	if manifest.ManagedService.ArtifactRole != "managed-service" || manifest.ManagedService.Acquisition.Kind != "none" {
		t.Fatalf("managed acquisition must remain gated on signed release metadata: role=%q kind=%q", manifest.ManagedService.ArtifactRole, manifest.ManagedService.Acquisition.Kind)
	}
	if len(manifest.ManagedService.Acquisition.Targets) == 0 {
		t.Fatal("each target needs an explicit unsupported reason until signed artifacts are published")
	}
	for _, target := range manifest.ManagedService.Acquisition.Targets {
		if !strings.Contains(strings.ToLower(target.Unsupported), "sign") || !strings.Contains(strings.ToLower(target.Unsupported), "publish") {
			t.Errorf("target must explain signed publication gate, got %q", target.Unsupported)
		}
	}
	const group = "streaming-zipformer-en-2023-06-26"
	want := map[string]string{
		"streaming-zipformer-en-2023-06-26":                "models/asr",
		"punctuation-ct-transformer-zh-en-2024-04-12-int8": "models/punctuation",
	}
	for name, path := range want {
		found := false
		for _, artifact := range manifest.ManagedService.DataArtifacts {
			if artifact.Name == name && artifact.Path == path && artifact.Group == group {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("managed streaming group %q must declare %s at %s", group, name, path)
		}
	}
	for key, relative := range map[string]string{
		"SHERPA_ONNX_STREAMING_MODEL_DIR":   "models/asr/sherpa-onnx-streaming-zipformer-en-2023-06-26",
		"SHERPA_ONNX_PUNCTUATION_MODEL_DIR": "models/punctuation/sherpa-onnx-punct-ct-transformer-zh-en-vocab272727-2024-04-12-int8",
	} {
		got := strings.TrimPrefix(manifest.ManagedService.Environment[key], "${RESOURCE_DATA_DIR}/")
		if got != relative {
			t.Errorf("%s resolves to %q, want RESOURCE_DATA_DIR/%s", key, got, relative)
		}
	}
}

// Audio Tools can select the local Sherpa adapter, but managed lifecycle must
// not claim Linux support until the complete native bundle has been signed
// and published. A local tree digest establishes integrity, but not release
// provenance or an address from which managed lifecycle can acquire it.
func TestLinuxManagedLifecycleRemainsGatedOnPublishedBundle(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	manifestPath := filepath.Join(filepath.Dir(source), "..", "resource.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Platforms map[string]string `json:"platforms"`
		Deployment struct {
			Profiles map[string]map[string]struct {
				Support string `json:"support"`
			} `json:"profiles"`
		} `json:"deployment"`
		ManagedService struct {
			Artifact struct {
				SHA256ByPlatform map[string]string `json:"sha256_by_platform"`
			} `json:"artifact"`
			Acquisition struct {
				Kind    string `json:"kind"`
				Targets []struct {
					When struct {
						OS   string `json:"os"`
						Arch string `json:"arch"`
					} `json:"when"`
					Unsupported string `json:"unsupported"`
				} `json:"targets"`
			} `json:"acquisition"`
		} `json:"managed_service"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}

	const platform = "linux-amd64"
	if manifest.ManagedService.Artifact.SHA256ByPlatform[platform] == "" {
		t.Fatalf("expected local bundle digest for %s so the test proves a digest alone does not enable lifecycle", platform)
	}
	if manifest.ManagedService.Acquisition.Kind != "none" {
		t.Fatalf("acquisition kind = %q, want none until signed bundle publication exists", manifest.ManagedService.Acquisition.Kind)
	}
	foundTarget := false
	for _, target := range manifest.ManagedService.Acquisition.Targets {
		if target.When.OS == "linux" && target.When.Arch == "amd64" {
			foundTarget = true
			if !strings.Contains(strings.ToLower(target.Unsupported), "signed") || !strings.Contains(strings.ToLower(target.Unsupported), "published") {
				t.Errorf("Linux amd64 unsupported reason must state signed publication gate, got %q", target.Unsupported)
			}
		}
	}
	if !foundTarget {
		t.Fatal("Linux amd64 must have an explicit unsupported acquisition target")
	}
	if got := manifest.Platforms["linux"]; got != "unsupported" {
		t.Errorf("Linux platform support = %q, want unsupported until bundle publication", got)
	}
	if got := manifest.Deployment.Profiles["desktop"]["linux"].Support; got != "unsupported" {
		t.Errorf("desktop Linux deployment support = %q, want unsupported until bundle publication", got)
	}
}

func TestManaged20MZipformerIsSeparatelyPinnedOnDemand(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	manifestPath := filepath.Join(filepath.Dir(source), "..", "resource.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		ManagedService struct {
			DataArtifacts []struct {
				Name            string `json:"name"`
				Path            string `json:"path"`
				Group           string `json:"group"`
				AcquisitionMode string `json:"acquisition_mode"`
				Acquisition     struct {
					License string `json:"license"`
					Targets []struct {
						URL     string `json:"url"`
						SHA256  string `json:"sha256"`
						Archive string `json:"archive"`
						BinPath string `json:"bin_path"`
					} `json:"targets"`
				} `json:"acquisition"`
			} `json:"data_artifacts"`
		} `json:"managed_service"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	const name = "streaming-zipformer-en-20m-2023-02-17"
	for _, artifact := range manifest.ManagedService.DataArtifacts {
		if artifact.Name != name {
			continue
		}
		if artifact.Path != "models/asr" || artifact.Group != name || artifact.AcquisitionMode != "on-demand" {
			t.Fatalf("20M Zipformer location/group/mode = %q/%q/%q", artifact.Path, artifact.Group, artifact.AcquisitionMode)
		}
		if !strings.Contains(artifact.Acquisition.License, "Apache-2.0") || len(artifact.Acquisition.Targets) != 1 {
			t.Fatalf("20M Zipformer license/targets are incomplete: license=%q targets=%d", artifact.Acquisition.License, len(artifact.Acquisition.Targets))
		}
		target := artifact.Acquisition.Targets[0]
		if target.URL != "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-streaming-zipformer-en-20M-2023-02-17.tar.bz2" ||
			target.SHA256 != "9c559283e8498d3fe95913c79ca1cb454bb26281ac2b102b41306c7d752765d9" ||
			target.Archive != "tar.bz2" || target.BinPath != "sherpa-onnx-streaming-zipformer-en-20M-2023-02-17/tokens.txt" {
			t.Fatalf("unexpected 20M Zipformer artifact target: %+v", target)
		}
		return
	}
	t.Fatalf("missing separately pinned on-demand artifact %q", name)
}
