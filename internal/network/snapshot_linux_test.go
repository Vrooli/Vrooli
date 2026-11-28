//go:build linux

package network

import (
	"os"
	"path/filepath"
	"testing"
)

const fakeProcNetTCPv4 = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:100E 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 12345 1 0000000000000000 100 0 0 10 0
`

const fakeProcNetTCPv6 = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000000000000:1F90 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 23456 1 0000000000000000 100 0 0 10 0
`

// stubProcNetTCP points the capture at a fake procfs tree. Pass "" to make a
// family's file absent (ENOENT) and "DENIED" to make it unreadable.
func stubProcNetTCP(t *testing.T, v4, v6 string) {
	t.Helper()
	dir := t.TempDir()
	prevV4, prevV6 := procNetTCPv4Path, procNetTCPv6Path
	t.Cleanup(func() {
		procNetTCPv4Path, procNetTCPv6Path = prevV4, prevV6
	})
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		switch content {
		case "":
			return path // never written — reads fail with ENOENT
		case "DENIED":
			if err := os.WriteFile(path, []byte(fakeProcNetTCPv4), 0o000); err != nil {
				t.Fatal(err)
			}
		default:
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return path
	}
	procNetTCPv4Path = write("tcp", v4)
	procNetTCPv6Path = write("tcp6", v6)
}

// TestCaptureSnapshotRequiresIPv4Table pins the Known:false invariant for a
// partial procfs failure: if /proc/net/tcp cannot be read, a readable tcp6
// must NOT produce a Known snapshot — every IPv4-only listener would read
// known-absent and reconcile expires claims on known-absent listeners.
func TestCaptureSnapshotRequiresIPv4Table(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("0o000 files are readable by root")
	}
	stubProcNetTCP(t, "DENIED", fakeProcNetTCPv6)
	snapshot := captureTCPListenerSnapshot(CaptureOptions{AttributeProcesses: true})
	if snapshot.Known {
		t.Fatalf("snapshot must not be Known when the IPv4 table is unreadable; got Known with reason=%q", snapshot.Reason)
	}
}

// TestCaptureSnapshotUnreadableIPv6TableDegradesToUnknown pins the mirrored
// case: a present-but-unreadable tcp6 means IPv6 listeners may exist that the
// snapshot cannot see, so it must not be Known.
func TestCaptureSnapshotUnreadableIPv6TableDegradesToUnknown(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("0o000 files are readable by root")
	}
	stubProcNetTCP(t, fakeProcNetTCPv4, "DENIED")
	snapshot := captureTCPListenerSnapshot(CaptureOptions{AttributeProcesses: true})
	if snapshot.Known {
		t.Fatalf("snapshot must not be Known when a present IPv6 table is unreadable; got Known with reason=%q", snapshot.Reason)
	}
}

// TestCaptureSnapshotMissingIPv6TableIsComplete pins the one tolerated gap:
// tcp6 ENOENT means IPv6 is disabled, so the v4 set alone is complete and the
// snapshot stays Known.
func TestCaptureSnapshotMissingIPv6TableIsComplete(t *testing.T) {
	stubProcNetTCP(t, fakeProcNetTCPv4, "")
	snapshot := captureTCPListenerSnapshot(CaptureOptions{AttributeProcesses: true})
	if !snapshot.Known {
		t.Fatalf("snapshot must be Known when tcp6 is absent (IPv6 disabled); reason=%q", snapshot.Reason)
	}
	if !snapshot.Listening(4110).Listening {
		t.Fatal("expected port 4110 from the v4 table to be listening")
	}
	if snapshot.Listening(8080).Listening {
		t.Fatal("port 8080 must not be listening without the v6 table")
	}
}

// TestCaptureSnapshotMergesBothFamilies pins the union: a port present in
// either table is listening.
func TestCaptureSnapshotMergesBothFamilies(t *testing.T) {
	stubProcNetTCP(t, fakeProcNetTCPv4, fakeProcNetTCPv6)
	snapshot := captureTCPListenerSnapshot(CaptureOptions{AttributeProcesses: true})
	if !snapshot.Known {
		t.Fatalf("snapshot must be Known; reason=%q", snapshot.Reason)
	}
	for _, port := range []int{4110, 8080} {
		if !snapshot.Listening(port).Listening {
			t.Fatalf("expected port %d to be listening", port)
		}
	}
}

// TestCaptureSnapshotMissingIPv4TableDegradesToUnknown pins total v4 absence
// (e.g. a procfs mount oddity): no Known snapshot.
func TestCaptureSnapshotMissingIPv4TableDegradesToUnknown(t *testing.T) {
	stubProcNetTCP(t, "", fakeProcNetTCPv6)
	snapshot := captureTCPListenerSnapshot(CaptureOptions{AttributeProcesses: true})
	if snapshot.Known {
		t.Fatal("snapshot must not be Known when /proc/net/tcp is absent")
	}
}

// TestCapturePortsOnlySkipsAttributionSubprocess pins the fork-free path:
// reconciliation reads only Known/Listening, so a capture that does not ask
// for attribution must answer the port set without shelling out to ss. The
// port set itself still comes from procfs and must be unchanged.
func TestCapturePortsOnlySkipsAttributionSubprocess(t *testing.T) {
	stubProcNetTCP(t, fakeProcNetTCPv4, fakeProcNetTCPv6)

	snapshot := captureTCPListenerSnapshot(CaptureOptions{})
	if !snapshot.Known {
		t.Fatalf("snapshot must be Known; reason=%q", snapshot.Reason)
	}
	if snapshot.Tool != "procfs" {
		t.Fatalf("attribution must be skipped, so tool must stay %q; got %q", "procfs", snapshot.Tool)
	}
	for _, port := range []int{4110, 8080} {
		state := snapshot.Listening(port)
		if !state.Known || !state.Listening {
			t.Fatalf("port %d must still be reported listening without attribution", port)
		}
		if len(state.Listeners) != 0 {
			t.Fatalf("port %d must carry no attribution; got %d listeners", port, len(state.Listeners))
		}
	}
}
