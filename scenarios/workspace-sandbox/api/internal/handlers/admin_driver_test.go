package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/driverpref"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/testutil/mocks/procmocks"
	"workspace-sandbox/internal/testutil/mocks/sandboxiface"
)

func TestSelectDriverOverlayfsUserNSOutsideNamespacePersistsPreferenceForRestart(t *testing.T) {
	baseDir := t.TempDir()
	starter := procmocks.NewFakeStarter()
	starter.SetLookPath("unshare", "/usr/bin/unshare")
	starter.AddCommand("unshare --user true", procmocks.CommandBehavior{})

	current := mocks.NewFakeDriver()
	current.IDValue = driver.DriverCopy

	live := newLive(t, &sandboxiface.FakeService{}, withDriver(current), func(h *handlers.Handlers) {
		h.Config = config.Config{
			Driver: config.DriverConfig{BaseDir: baseDir},
		}
		h.Starter = starter
		h.InUserNamespace = false
	})

	resp, body := live.DoJSON(t, "POST", "/api/v1/driver/select", `{"driverId":"overlayfs-userns"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	var got handlers.SelectDriverResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !got.Success {
		t.Fatalf("success = false; body=%s", body)
	}
	if !got.RequiresRestart {
		t.Fatalf("requiresRestart = false, want true; body=%s", body)
	}
	if got.SelectedDriver != string(driver.DriverOverlayfsUserNS) {
		t.Fatalf("selectedDriver = %q, want %q", got.SelectedDriver, driver.DriverOverlayfsUserNS)
	}

	pref, err := driverpref.Load(baseDir)
	if err != nil {
		t.Fatalf("load saved preference: %v", err)
	}
	if pref != driver.DriverOverlayfsUserNS {
		t.Fatalf("preference = %q, want %q", pref, driver.DriverOverlayfsUserNS)
	}
	if current.ID() != driver.DriverCopy {
		t.Fatalf("current driver changed to %q, want %q", current.ID(), driver.DriverCopy)
	}
}
