package delivery

import "testing"

func TestNormalizeCatalogPlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		wantOS   string
		wantArch string
		wantErr  bool
	}{
		{name: "linux target id", platform: "linux-x64", wantOS: "linux", wantArch: "amd64"},
		{name: "darwin arm64 target id", platform: "darwin-arm64", wantOS: "mac", wantArch: "arm64"},
		{name: "win target id", platform: "win-x64", wantOS: "windows", wantArch: "amd64"},
		{name: "windows os key is idempotent", platform: "windows", wantOS: "windows"},
		{name: "mac os key is idempotent", platform: "mac", wantOS: "mac"},
		{name: "linux os key is idempotent", platform: "linux", wantOS: "linux"},
		{name: "rejects empty", platform: "  ", wantErr: true},
		{name: "rejects unknown os", platform: "freebsd-x64", wantErr: true},
		{name: "rejects unknown arch", platform: "linux-mips", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osPlatform, arch, err := NormalizeCatalogPlatform(tt.platform)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeCatalogPlatform(%q) = (%q, %q), want error", tt.platform, osPlatform, arch)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeCatalogPlatform(%q) error: %v", tt.platform, err)
			}
			if osPlatform != tt.wantOS || arch != tt.wantArch {
				t.Fatalf("NormalizeCatalogPlatform(%q) = (%q, %q), want (%q, %q)", tt.platform, osPlatform, arch, tt.wantOS, tt.wantArch)
			}
		})
	}
}

func TestCatalogArtifactMetadataTagsMachine(t *testing.T) {
	metadata := map[string]interface{}{"release_id": "release-1", "platform": "linux-x64"}
	result := catalogArtifactMetadata(metadata, "linux-x64", "linux", "amd64")
	if result["release_id"] != "release-1" {
		t.Fatalf("release identity metadata was dropped: %+v", result)
	}
	if result["platform"] != "linux" {
		t.Fatalf("platform = %v, want canonical linux", result["platform"])
	}
	if result["target_id"] != "linux-x64" {
		t.Fatalf("target_id = %v, want linux-x64", result["target_id"])
	}
	if result["architecture"] != "amd64" {
		t.Fatalf("architecture = %v, want amd64", result["architecture"])
	}

	withoutArch := catalogArtifactMetadata(map[string]interface{}{"architecture": "stale"}, "mac", "mac", "")
	if _, exists := withoutArch["architecture"]; exists {
		t.Fatalf("unspecified architecture must not inherit a stale value: %+v", withoutArch)
	}
}
