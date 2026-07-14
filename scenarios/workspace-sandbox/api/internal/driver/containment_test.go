package driver

import (
	"reflect"
	"testing"
)

// TestDeriveWorkspaceLayout pins the single derivation that every
// sandbox-returning handler uses. The matrix covers the driver required
// level crossed with the host containment probe, asserting the layout
// invariant: pathIllusion is true exactly when the effective backend
// provides path-illusion enforcement, and in that case workspacePath is
// NamespaceWorkspacePath; otherwise workspacePath is the host merged dir.
func TestDeriveWorkspaceLayout(t *testing.T) {
	const merged = "/var/lib/workspace-sandbox/abc/merged"

	bwrapInfo := &ContainmentInfo{
		Backend:   "bwrap",
		Available: true,
		Enforcements: []string{
			EnforcementFilesystemWriteContainment,
			EnforcementNetworkDeny,
			EnforcementPIDNamespace,
			EnforcementPathIllusion,
		},
	}
	bwrapMissing := &ContainmentInfo{Backend: "bwrap", Available: false, Enforcements: []string{}}
	noneInfo := &ContainmentInfo{Backend: "none", Available: false, Enforcements: []string{}}
	// A hypothetical future backend that contains writes + network but does
	// NOT rewrite paths (e.g. macOS Seatbelt): identity layout must hold.
	seatbeltNoIllusion := &ContainmentInfo{
		Backend:      "seatbelt",
		Available:    true,
		Enforcements: []string{EnforcementFilesystemWriteContainment, EnforcementNetworkDeny},
	}

	cases := []struct {
		name         string
		level        ContainmentLevel
		info         *ContainmentInfo
		wantPath     string
		wantIllusion bool
		wantBackend  string
		wantEnforceN int
	}{
		{"copy-driver-none", ContainmentNone, bwrapInfo, merged, false, "none", 0},
		{"required-bwrap", ContainmentRequired, bwrapInfo, NamespaceWorkspacePath, true, "bwrap", 4},
		{"preferred-bwrap", ContainmentPreferred, bwrapInfo, NamespaceWorkspacePath, true, "bwrap", 4},
		{"required-bwrap-missing", ContainmentRequired, bwrapMissing, merged, false, "none", 0},
		{"preferred-bwrap-missing", ContainmentPreferred, bwrapMissing, merged, false, "none", 0},
		{"none-backend-off-linux", ContainmentNone, noneInfo, merged, false, "none", 0},
		{"seatbelt-no-path-illusion", ContainmentRequired, seatbeltNoIllusion, merged, false, "seatbelt", 2},
		{"nil-info", ContainmentRequired, nil, merged, false, "none", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, illusion, cont := DeriveWorkspaceLayout(tc.level, tc.info, merged)
			if path != tc.wantPath {
				t.Errorf("workspacePath: got %q, want %q", path, tc.wantPath)
			}
			if illusion != tc.wantIllusion {
				t.Errorf("pathIllusion: got %v, want %v", illusion, tc.wantIllusion)
			}
			if cont == nil {
				t.Fatal("containment must never be nil")
			}
			if cont.Backend != tc.wantBackend {
				t.Errorf("backend: got %q, want %q", cont.Backend, tc.wantBackend)
			}
			if cont.Level != tc.level.String() {
				t.Errorf("level: got %q, want %q", cont.Level, tc.level.String())
			}
			if len(cont.Enforcements) != tc.wantEnforceN {
				t.Errorf("enforcement count: got %d %v, want %d", len(cont.Enforcements), cont.Enforcements, tc.wantEnforceN)
			}
			// Enforcements must always be non-nil so the wire is [] not null.
			if cont.Enforcements == nil {
				t.Error("enforcements must be non-nil (empty slice, not null)")
			}
		})
	}
}

// TestSeatbeltContainmentInfo pins the pure macOS Seatbelt capability
// derivation the darwin probe wraps. It runs on the Linux dev host because
// the report is a pure function of (sandbox-exec available, path): available
// yields backend=seatbelt with exactly filesystem-write-containment +
// network-deny and explicitly no path-illusion/pid-namespace; unavailable
// degrades to the direct-path backend with an empty enforcement list.
func TestSeatbeltContainmentInfo(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		info := seatbeltContainmentInfo(true, "/usr/bin/sandbox-exec")
		if info.Backend != SeatbeltBackendID {
			t.Errorf("backend: got %q, want %q", info.Backend, SeatbeltBackendID)
		}
		if !info.Available {
			t.Error("expected available=true when sandbox-exec is present")
		}
		if info.Path != "/usr/bin/sandbox-exec" {
			t.Errorf("path: got %q, want /usr/bin/sandbox-exec", info.Path)
		}
		wantEnf := []string{EnforcementFilesystemWriteContainment, EnforcementNetworkDeny}
		if !reflect.DeepEqual(info.Enforcements, wantEnf) {
			t.Errorf("enforcements: got %v, want %v", info.Enforcements, wantEnf)
		}
		if hasEnforcement(info.Enforcements, EnforcementPathIllusion) {
			t.Error("seatbelt must NOT report path-illusion")
		}
		if hasEnforcement(info.Enforcements, EnforcementPIDNamespace) {
			t.Error("seatbelt must NOT report pid-namespace")
		}
	})

	t.Run("unavailable", func(t *testing.T) {
		info := seatbeltContainmentInfo(false, "")
		if info.Backend != backendNone {
			t.Errorf("backend: got %q, want %q", info.Backend, backendNone)
		}
		if info.Available {
			t.Error("expected available=false when sandbox-exec is missing")
		}
		if len(info.Enforcements) != 0 {
			t.Errorf("enforcements: got %v, want empty", info.Enforcements)
		}
		if info.Enforcements == nil {
			t.Error("enforcements must be non-nil (empty slice, not null)")
		}
	})
}

// TestDeriveWorkspaceLayout_SeatbeltPayload feeds a simulated darwin Seatbelt
// capability payload through the phase-3 derivation to prove a macOS sandbox
// reports honest partial containment: backend=seatbelt, pathIllusion=false,
// workspacePath=mergedDir (identity layout, no path illusion).
func TestDeriveWorkspaceLayout_SeatbeltPayload(t *testing.T) {
	const merged = "/var/lib/workspace-sandbox/abc/merged"
	info := seatbeltContainmentInfo(true, "/usr/bin/sandbox-exec")

	for _, level := range []ContainmentLevel{ContainmentPreferred, ContainmentRequired} {
		t.Run(level.String(), func(t *testing.T) {
			path, illusion, cont := DeriveWorkspaceLayout(level, info, merged)
			if illusion {
				t.Error("seatbelt provides no path illusion; want pathIllusion=false")
			}
			if path != merged {
				t.Errorf("workspacePath: got %q, want identity %q", path, merged)
			}
			if cont.Backend != SeatbeltBackendID {
				t.Errorf("backend: got %q, want %q", cont.Backend, SeatbeltBackendID)
			}
			if len(cont.Enforcements) != 2 {
				t.Errorf("enforcements: got %v, want the 2 seatbelt guarantees", cont.Enforcements)
			}
		})
	}
}

// TestContainmentLevelString locks the platform-neutral level vocabulary
// used on the wire.
func TestContainmentLevelString(t *testing.T) {
	cases := map[ContainmentLevel]string{
		ContainmentNone:      "none",
		ContainmentPreferred: "preferred",
		ContainmentRequired:  "required",
	}
	for level, want := range cases {
		if got := level.String(); got != want {
			t.Errorf("ContainmentLevel(%d).String() = %q, want %q", level, got, want)
		}
	}
}
