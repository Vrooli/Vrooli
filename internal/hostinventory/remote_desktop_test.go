package hostinventory

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

func TestClassifyRemoteDesktopAgreementFixtures(t *testing.T) {
	tests := []struct {
		name         string
		os           string
		systemd      bool
		outputs      map[string][]byte
		wantProvider string
		wantPresent  bool
		wantActive   bool
	}{
		{
			name:         "gnome system active",
			os:           "linux",
			systemd:      true,
			outputs:      map[string][]byte{"systemctl is-enabled gnome-remote-desktop.service": []byte("enabled\n"), "systemctl is-active gnome-remote-desktop.service": []byte("active\n")},
			wantProvider: "gnome-system",
			wantPresent:  true,
			wantActive:   true,
		},
		{
			name:         "gnome headless active",
			os:           "linux",
			systemd:      true,
			outputs:      map[string][]byte{"grdctl status": []byte("Status: enabled\n"), "pgrep -a -f gnome-remote-desktop-daemon": []byte("122 /bin/sh -c pgrep -a -f gnome-remote-desktop-daemon\n123 /usr/libexec/gnome-remote-desktop-daemon\n")},
			wantProvider: "gnome-headless",
			wantPresent:  true,
			wantActive:   true,
		},
		{
			name:         "xrdp active",
			os:           "linux",
			systemd:      true,
			outputs:      map[string][]byte{"systemctl list-unit-files xrdp.service": []byte("xrdp.service enabled\n"), "systemctl is-active xrdp.service": []byte("active\n")},
			wantProvider: "xrdp",
			wantPresent:  true,
			wantActive:   true,
		},
		{
			name:    "linux no provider",
			os:      "linux",
			systemd: true,
			outputs: map[string][]byte{},
		},
		{
			name:         "windows active",
			os:           "windows",
			outputs:      map[string][]byte{"sc.exe query TermService": []byte("SERVICE_NAME: TermService\nSTATE : 4 RUNNING\n")},
			wantProvider: "windows-termservice",
			wantPresent:  true,
			wantActive:   true,
		},
		{
			name:         "windows stopped",
			os:           "windows",
			outputs:      map[string][]byte{"sc.exe query TermService": []byte("SERVICE_NAME: TermService\nSTATE : 1 STOPPED\n")},
			wantProvider: "windows-termservice",
			wantPresent:  true,
			wantActive:   false,
		},
		{
			name:         "mac screen sharing",
			os:           "darwin",
			outputs:      map[string][]byte{"launchctl print system/com.apple.screensharing": []byte("service\n")},
			wantProvider: "macos-screen-sharing",
			wantPresent:  true,
			wantActive:   true,
		},
		{
			name:    "gnome user shared",
			os:      "linux",
			systemd: true,
			outputs: map[string][]byte{
				"pgrep -a -f gnome-remote-desktop-daemon":           []byte("122 /bin/sh -c pgrep -a -f gnome-remote-desktop-daemon\n123 /usr/libexec/gnome-remote-desktop-daemon\n"),
				"systemctl is-enabled gnome-remote-desktop.service": []byte("disabled\n"),
				"systemctl is-active gnome-remote-desktop.service":  []byte("inactive\n"),
				"id -u alice": []byte("1000\n"),
				"env XDG_RUNTIME_DIR=/run/user/1000 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus systemctl --user is-enabled gnome-remote-desktop.service": []byte("enabled\n"),
				"env XDG_RUNTIME_DIR=/run/user/1000 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus systemctl --user is-active gnome-remote-desktop.service":  []byte("active\n"),
				"env XDG_RUNTIME_DIR=/run/user/1000 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus grdctl status":                                            []byte("Status: enabled\n"),
			},
			wantProvider: "gnome-user-shared",
			wantPresent:  true,
			wantActive:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var facts RemoteDesktopCapability
			if tt.name == "gnome user shared" {
				facts = ClassifyRemoteDesktopWithDisplayAndUser(context.Background(), tt.os, tt.systemd, true, "alice", &shelltest.Fake{Outputs: tt.outputs})
			} else {
				facts = ClassifyRemoteDesktop(context.Background(), tt.os, tt.systemd, &shelltest.Fake{Outputs: tt.outputs})
			}
			if facts.SelectedProvider != tt.wantProvider {
				t.Fatalf("SelectedProvider = %q, want %q; facts=%#v", facts.SelectedProvider, tt.wantProvider, facts)
			}
			if tt.wantProvider == "" {
				if facts.Supported || facts.Observed {
					t.Fatalf("empty fixture unexpectedly supported/observed: %#v", facts)
				}
				return
			}
			provider, ok := facts.Provider(tt.wantProvider)
			if !ok || provider.Present != tt.wantPresent || provider.Active != tt.wantActive {
				t.Fatalf("provider = %#v, ok=%v, want present=%v active=%v", provider, ok, tt.wantPresent, tt.wantActive)
			}
		})
	}
}
