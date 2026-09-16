package delivery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestDefaultDownloadSeedPreservesExactFallbackCatalog(t *testing.T) {
	apps, err := DefaultDownloadSeed()
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 19 {
		t.Fatalf("download seed contains %d apps, want 19", len(apps))
	}

	keys := make([]string, 0, len(apps))
	for _, app := range apps {
		if app.BundleKey != "business_suite" {
			t.Fatalf("app %q has bundle %q", app.AppKey, app.BundleKey)
		}
		keys = append(keys, app.AppKey)
	}
	wantKeys := []string{
		"browser-automation-studio", "landing-page-business-suite", "web-console", "audio-tools",
		"git-control-tower", "agent-manager", "swarm-manager", "prompt-manager", "app-library",
		"scenario-to-plugin", "vrooli-onboarding", "vrooli-bridge", "notification-hub", "device-control",
		"switchboard", "treasury", "system-monitor", "portal", "vrooli-memory",
	}
	if got, want := keys, wantKeys; len(got) != len(want) {
		t.Fatalf("app key count = %d, want %d", len(got), len(want))
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("app key %d = %q, want %q", i, got[i], want[i])
			}
		}
	}

	bas := apps[0]
	if bas.AppKey != "browser-automation-studio" || bas.Metadata["enabled"] != false || bas.Metadata["catalog_status"] != "planned" {
		t.Fatalf("BAS availability changed: %#v", bas)
	}
	if len(bas.Platforms) != 3 {
		t.Fatalf("BAS platform count = %d, want 3", len(bas.Platforms))
	}
	for _, platform := range bas.Platforms {
		if !platform.RequiresEntitlement {
			t.Fatalf("BAS platform %q lost entitlement gating", platform.Platform)
		}
	}
	if len(apps[1].Platforms) != 0 || len(apps[2].Platforms) != 0 {
		t.Fatalf("hosted apps unexpectedly gained installer records")
	}

	// The web-console desktop release depends on this exact catalog row being
	// live and enabled before scenario-to-desktop can promote to LPBS.
	console := apps[2]
	if console.AppKey != "web-console" || console.Metadata["enabled"] != true || console.Metadata["catalog_status"] != "live" {
		t.Fatalf("web-console release app is not live/enabled: %#v", console)
	}

	var value any
	if err := json.Unmarshal(downloadSeedJSON, &value); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	if got, want := hex.EncodeToString(digest[:]), "8424ba47e9712535411599e17db2af9da491b58e15adcbeae0c7c0dfce3cca67"; got != want {
		t.Fatalf("embedded delivery seed digest = %s, want %s", got, want)
	}
}
