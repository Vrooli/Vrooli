package driver

import "testing"

// TestOverlayFactories pins the static (no-I/O) contract of each
// overlay-flavored factory: ID, RequiresBwrap, Capabilities. Adding a
// new flavor = a new row here.
func TestOverlayFactories(t *testing.T) {
	cfg := Config{BaseDir: t.TempDir()}
	cases := []struct {
		name         string
		drv          *OverlayDriver
		wantID       DriverID
		wantBwrap    IsolationMode
		wantHomeOver bool
	}{
		{"userns", NewOverlayfsUserNSDriver(cfg, testDeps()), DriverOverlayfsUserNS, ModeBwrapRequired, true},
		{"root", NewOverlayfsRootDriver(cfg, testDeps()), DriverOverlayfsRoot, ModeBwrapRequired, true},
		{"fuse", NewFuseOverlayfsDriver(cfg, testDeps()), DriverFuseOverlayfs, ModeBwrapPreferred, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.drv.ID(); got != tc.wantID {
				t.Errorf("ID() = %v, want %v", got, tc.wantID)
			}
			if got := tc.drv.RequiresBwrap(); got != tc.wantBwrap {
				t.Errorf("RequiresBwrap() = %v, want %v", got, tc.wantBwrap)
			}
			caps := tc.drv.Capabilities()
			if caps.HomeOverlay != tc.wantHomeOver {
				t.Errorf("Capabilities.HomeOverlay = %v, want %v", caps.HomeOverlay, tc.wantHomeOver)
			}
			if caps.NamespaceIsolation != tc.wantBwrap {
				t.Errorf("Capabilities.NamespaceIsolation = %v, want %v", caps.NamespaceIsolation, tc.wantBwrap)
			}
			if !caps.CoW {
				t.Error("Capabilities.CoW = false, want true for overlay flavors")
			}
			if tc.drv.Version() == "" {
				t.Error("Version() should not be empty")
			}
		})
	}
}
