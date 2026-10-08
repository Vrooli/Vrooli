package landing

import (
	"strings"
	"testing"

	"landing-page-business-suite-api/internal/delivery"
)

func TestSanitizeSigningNotice(t *testing.T) {
	tests := []struct {
		name    string
		raw     interface{}
		wantNil bool
		assert  func(t *testing.T, notice map[string]interface{})
	}{
		{
			name:    "non-object is dropped",
			raw:     "not a notice",
			wantNil: true,
		},
		{
			name:    "missing body disabled",
			raw:     map[string]interface{}{"title": "Signing pending"},
			wantNil: true,
		},
		{
			name:    "missing title disabled",
			raw:     map[string]interface{}{"body": "Details"},
			wantNil: true,
		},
		{
			name: "explicit suppression keeps only the flag",
			raw:  map[string]interface{}{"enabled": false, "title": "secret", "body": "secret"},
			assert: func(t *testing.T, notice map[string]interface{}) {
				if len(notice) != 1 || notice["enabled"] != false {
					t.Fatalf("suppression notice = %#v, want {\"enabled\": false}", notice)
				}
			},
		},
		{
			name: "valid notice defaults severity to info and bounds copy",
			raw: map[string]interface{}{
				"title":      "  Preview release  ",
				"body":       "This build is unsigned while we finish signing.",
				"link_url":   "https://github.com/Vrooli/Vrooli/tree/master/scenarios/web-console",
				"link_label": "Review the source",
				"unknown":    "dropped",
			},
			assert: func(t *testing.T, notice map[string]interface{}) {
				if notice["enabled"] != true || notice["severity"] != "info" {
					t.Fatalf("notice flags = %#v", notice)
				}
				if notice["title"] != "Preview release" {
					t.Fatalf("title = %v", notice["title"])
				}
				if notice["link_label"] != "Review the source" {
					t.Fatalf("link_label = %v", notice["link_label"])
				}
				if _, ok := notice["unknown"]; ok {
					t.Fatalf("unknown field forwarded: %#v", notice)
				}
			},
		},
		{
			name: "warning severity passes through",
			raw:  map[string]interface{}{"title": "Unsigned", "body": "Details", "severity": "WARNING"},
			assert: func(t *testing.T, notice map[string]interface{}) {
				if notice["severity"] != "warning" {
					t.Fatalf("severity = %v", notice["severity"])
				}
			},
		},
		{
			name: "insecure and script links are dropped",
			raw:  map[string]interface{}{"title": "Unsigned", "body": "Details", "link_url": "http://insecure.example.com", "link_label": "bad"},
			assert: func(t *testing.T, notice map[string]interface{}) {
				if _, ok := notice["link_url"]; ok {
					t.Fatalf("insecure link forwarded: %#v", notice)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			notice := sanitizeSigningNotice(test.raw)
			if test.wantNil {
				if notice != nil {
					t.Fatalf("sanitizeSigningNotice() = %#v, want nil", notice)
				}
				return
			}
			if notice == nil {
				t.Fatal("sanitizeSigningNotice() = nil, want notice")
			}
			test.assert(t, notice)
		})
	}
}

func TestSanitizeSigningNoticeTruncatesCopy(t *testing.T) {
	notice := sanitizeSigningNotice(map[string]interface{}{"title": strings.Repeat("a", 400), "body": strings.Repeat("b", 4000)})
	if notice == nil {
		t.Fatal("sanitizeSigningNotice() = nil")
	}
	if got := len([]rune(notice["title"].(string))); got != 160 {
		t.Fatalf("title length = %d, want 160", got)
	}
	if got := len([]rune(notice["body"].(string))); got != 2000 {
		t.Fatalf("body length = %d, want 2000", got)
	}
}

func TestProtoPresentationDownloadsForwardsSigningNotice(t *testing.T) {
	appNotice := map[string]interface{}{"enabled": true, "title": "Preview release", "body": "Signing is in progress.", "severity": "info", "link_label": "Review the source", "link_url": "https://github.com/Vrooli/Vrooli/tree/master/scenarios/web-console"}
	assetNotice := map[string]interface{}{"enabled": true, "title": "Windows build", "body": "The Windows installer is unsigned.", "severity": "warning"}
	apps := []delivery.App{{
		BundleKey: "business_suite", AppKey: "web-console", Name: "Aquila",
		Metadata: map[string]interface{}{"web_url": "/app/web-console", "signing_notice": appNotice, "catalog_status": "live"},
		Platforms: []delivery.Asset{
			{
				ID: 1, BundleKey: "business_suite", AppKey: "web-console", Platform: "linux",
				ReleaseVersion: "1.0.0", ArtifactURL: "https://private.example.com/secret.AppImage",
				Metadata: map[string]interface{}{"signing_notice": assetNotice},
			},
			{
				ID: 2, BundleKey: "business_suite", AppKey: "web-console", Platform: "mac",
				ReleaseVersion: "1.0.0", ArtifactURL: "https://private.example.com/secret.dmg",
				Metadata: map[string]interface{}{"signing_notice": map[string]interface{}{"enabled": false}},
			},
		},
	}}

	projected, err := ProtoPresentationDownloads(apps)
	if err != nil {
		t.Fatalf("ProtoPresentationDownloads() error = %v", err)
	}
	if len(projected) != 1 {
		t.Fatalf("projected app count = %d, want 1", len(projected))
	}
	metadata := projected[0].Metadata.AsMap()
	if metadata["web_url"] != "/app/web-console" {
		t.Fatalf("web_url = %v", metadata["web_url"])
	}
	if _, ok := metadata["catalog_status"]; ok {
		t.Fatalf("catalog_status leaked to public contract: %#v", metadata)
	}
	notice, ok := metadata["signing_notice"].(map[string]interface{})
	if !ok {
		t.Fatalf("app signing_notice missing: %#v", metadata)
	}
	if notice["title"] != "Preview release" || notice["link_url"] != "https://github.com/Vrooli/Vrooli/tree/master/scenarios/web-console" {
		t.Fatalf("app notice = %#v", notice)
	}
	if len(projected[0].Platforms) != 2 {
		t.Fatalf("platform count = %d, want 2", len(projected[0].Platforms))
	}
	linux := projected[0].Platforms[0]
	if linux.ArtifactUrl != "" {
		t.Fatalf("artifact URL leaked: %q", linux.ArtifactUrl)
	}
	linuxNotice, ok := linux.Metadata.AsMap()["signing_notice"].(map[string]interface{})
	if !ok || linuxNotice["severity"] != "warning" {
		t.Fatalf("linux notice = %#v", linux.Metadata.AsMap())
	}
	mac := projected[0].Platforms[1].Metadata.AsMap()["signing_notice"].(map[string]interface{})
	if mac["enabled"] != false || len(mac) != 1 {
		t.Fatalf("mac suppression notice = %#v", mac)
	}
}

func TestPublicOwnerSnapshotRefTracksSigningNotice(t *testing.T) {
	notice := map[string]interface{}{"signing_notice": map[string]interface{}{"enabled": true, "title": "Preview release", "body": "Signing is in progress."}}
	base := []delivery.App{{AppKey: "web-console", Platforms: []delivery.Asset{{AppKey: "web-console", Platform: "linux", ReleaseVersion: "1.0.0", Checksum: "sha256:a"}}}}
	withNotice := []delivery.App{{AppKey: "web-console", Metadata: notice, Platforms: []delivery.Asset{{AppKey: "web-console", Platform: "linux", ReleaseVersion: "1.0.0", Checksum: "sha256:a", Metadata: notice}}}}

	if publicOwnerSnapshotRef(nil, base) == publicOwnerSnapshotRef(nil, withNotice) {
		t.Fatal("signing notice did not change the public owner snapshot ref")
	}
}
