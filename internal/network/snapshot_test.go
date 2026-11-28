package network

import (
	"reflect"
	"sort"
	"testing"
)

func TestParseProcNetTCPListenPorts(t *testing.T) {
	fixture := []byte(`  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000:100E 00000000:0000 0A 00000000:00000000 00:00000000 00000000   999        0 58470 1 0000000000000000 100 0 0 10 0
   1: 0100007F:0277 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 901845572 1 0000000000000000 100 0 0 10 0
   2: 0100007F:1538 0100007F:8AE6 01 00000000:00000000 00:00000000 00000000  1000        0 12345 1 0000000000000000 20 4 30 10 -1
   3: malformed line
`)
	ports := parseProcNetTCPListenPorts(fixture)
	sort.Ints(ports)
	if !reflect.DeepEqual(ports, []int{631, 4110}) {
		t.Fatalf("ports = %v, want [631 4110] (0x0277=631, 0x100E=4110; ESTABLISHED row excluded)", ports)
	}
}

func TestParseProcNetTCP6ListenPorts(t *testing.T) {
	fixture := []byte(`  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000000000000:1F90 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 99999 1 0000000000000000 100 0 0 10 0
`)
	ports := parseProcNetTCPListenPorts(fixture)
	if !reflect.DeepEqual(ports, []int{8080}) {
		t.Fatalf("ports = %v, want [8080]", ports)
	}
}

func TestParseSSListenerAttribution(t *testing.T) {
	fixture := []byte(`LISTEN 0      511                                0.0.0.0:9090  0.0.0.0:* users:(("node",pid=680003,fd=19))
LISTEN 0      4096                             127.0.0.1:5432  0.0.0.0:* users:(("postgres",pid=1234,fd=5),("postgres",pid=1234,fd=6))
LISTEN 0      4096                                  [::]:631   [::]:*
`)
	// Injected resolver keeps the parser pure: fixture PIDs never touch the
	// real /proc, so the test cannot flake when those PIDs exist on the host.
	labels := map[int]string{680003: "/usr/bin/node server.js"}
	attributed := parseSSListenerAttribution(fixture, func(pid int) string { return labels[pid] })
	if len(attributed[9090]) != 1 || attributed[9090][0].PID != 680003 {
		t.Fatalf("port 9090 attribution = %#v", attributed[9090])
	}
	if attributed[9090][0].Label != "/usr/bin/node server.js" {
		t.Fatalf("label = %q, want resolver-provided cmdline", attributed[9090][0].Label)
	}
	// Duplicate pid across fds collapses to one listener.
	if len(attributed[5432]) != 1 || attributed[5432][0].PID != 1234 {
		t.Fatalf("port 5432 attribution = %#v", attributed[5432])
	}
	// The resolver has no entry for PID 1234; label falls back to the ss comm name.
	if attributed[5432][0].Label != "postgres" {
		t.Fatalf("label = %q, want ss comm fallback", attributed[5432][0].Label)
	}
	if _, ok := attributed[631]; ok {
		t.Fatal("line without users token must contribute no attribution")
	}
}

func TestParseNetstatListenPorts(t *testing.T) {
	fixture := []byte(`Active Internet connections (including servers)
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
tcp4       0      0  127.0.0.1.5432         *.*                    LISTEN
tcp46      0      0  *.8080                 *.*                    LISTEN
tcp4       0      0  192.168.1.5.52345      142.250.65.78.443      ESTABLISHED
udp4       0      0  *.5353                 *.*
`)
	ports := parseNetstatListenPorts(fixture)
	sort.Ints(ports)
	if !reflect.DeepEqual(ports, []int{5432, 8080}) {
		t.Fatalf("ports = %v, want [5432 8080]", ports)
	}
}

func TestParseLsofFieldAttribution(t *testing.T) {
	fixture := []byte(`p1234
cpostgres
n127.0.0.1:5432
p5678
cnode
n*:8080
n[::1]:8081
`)
	attributed := parseLsofFieldAttribution(fixture)
	if len(attributed[5432]) != 1 || attributed[5432][0].PID != 1234 || attributed[5432][0].Label != "postgres" {
		t.Fatalf("port 5432 = %#v", attributed[5432])
	}
	if len(attributed[8080]) != 1 || attributed[8080][0].PID != 5678 || attributed[8080][0].Label != "node" {
		t.Fatalf("port 8080 = %#v", attributed[8080])
	}
	if len(attributed[8081]) != 1 || attributed[8081][0].PID != 5678 {
		t.Fatalf("v6 port 8081 = %#v", attributed[8081])
	}
}

// TestSnapshotUnknownNeverReadsAsNotListening pins the key evidence
// invariant: a failed capture must answer Known:false for every port — never
// Known:true/Listening:false, which downstream reconcile would treat as a
// stale claim and expire.
func TestSnapshotUnknownNeverReadsAsNotListening(t *testing.T) {
	snapshot := TCPListenerSnapshot{Reason: "capture failed"}
	state := snapshot.Listening(8080)
	if state.Known || state.Listening {
		t.Fatalf("state = %#v, want zero-value (unknown)", state)
	}

	known := TCPListenerSnapshot{Known: true, Ports: map[int][]SnapshotListener{9090: nil}}
	if got := known.Listening(9090); !got.Known || !got.Listening {
		t.Fatalf("expected known+listening, got %#v", got)
	}
	if got := known.Listening(9091); !got.Known || got.Listening {
		t.Fatalf("expected known+not-listening, got %#v", got)
	}
	if got := known.Listening(0); got.Known {
		t.Fatalf("port 0 must be unknown, got %#v", got)
	}
}

func TestPortInspectionFromSnapshot(t *testing.T) {
	failed := PortInspectionFromSnapshot(TCPListenerSnapshot{Reason: "nope", Tool: "test"}, 80)
	if failed.Inspection.Available || failed.Inspection.Reason != "nope" {
		t.Fatalf("failed-capture inspection = %#v", failed.Inspection)
	}
	snapshot := TCPListenerSnapshot{Known: true, Tool: "test", Ports: map[int][]SnapshotListener{
		80: {{PID: 42, Label: "nginx"}},
		81: nil,
	}}
	withPID := PortInspectionFromSnapshot(snapshot, 80)
	if len(withPID.Listeners) != 1 || withPID.Listeners[0].PID != 42 || withPID.Listeners[0].Command != "nginx" {
		t.Fatalf("attributed inspection = %#v", withPID.Listeners)
	}
	unattributed := PortInspectionFromSnapshot(snapshot, 81)
	if len(unattributed.Listeners) != 1 || unattributed.Listeners[0].PID != 0 {
		t.Fatalf("unattributed listening port must still report a listener marker, got %#v", unattributed.Listeners)
	}
	idle := PortInspectionFromSnapshot(snapshot, 82)
	if len(idle.Listeners) != 0 || !idle.Inspection.Available {
		t.Fatalf("idle port inspection = %#v", idle)
	}
}
