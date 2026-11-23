//go:build linux

package callerid

import (
	"bufio"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const helperEnv = "CALLERID_HELPER_ADDR"

// TestCallerHelperProcess is not a test: it is the child process the loopback
// identification test connects from, because the lookup ignores the server's
// own process.
func TestCallerHelperProcess(t *testing.T) {
	addr := os.Getenv(helperEnv)
	if addr == "" {
		t.Skip("helper process only")
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		os.Exit(2)
	}
	defer conn.Close()
	_, _ = bufio.NewReader(conn).ReadByte() // hold the connection until the parent closes it
	os.Exit(0)
}

// connectFromHelper accepts one loopback connection from a child process and
// returns the server end.
func connectFromHelper(t *testing.T) net.Conn {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	helper := exec.Command(os.Args[0], "-test.run=^TestCallerHelperProcess$")
	helper.Env = append(os.Environ(), helperEnv+"="+listener.Addr().String())
	if err := helper.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = helper.Process.Kill(); _ = helper.Wait() })
	_ = listener.(*net.TCPListener).SetDeadline(time.Now().Add(10 * time.Second))
	conn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestIdentifyFindsTheConnectingProcessAndRemembersItsSocket(t *testing.T) {
	conn := connectFromHelper(t)
	resolver := NewResolver()

	got := resolver.Identify(conn.LocalAddr(), conn.RemoteAddr())
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, "/"+filepath.Base(exe)) || strings.HasPrefix(got, Unknown) {
		t.Fatalf("caller = %q, want the helper process named by its executable %s", got, filepath.Base(exe))
	}
	if len(resolver.sockets) != 1 {
		t.Fatalf("resolver remembered %d sockets, want 1", len(resolver.sockets))
	}

	// The remembered answer is keyed by the live socket, so it is served
	// without a process scan even after the scan could no longer succeed.
	for inode := range resolver.sockets {
		resolver.sockets[inode] = "scenario:remembered/helper"
	}
	if again := resolver.Identify(conn.LocalAddr(), conn.RemoteAddr()); again != "scenario:remembered/helper" {
		t.Fatalf("second identify = %q, want the remembered caller", again)
	}
}

func TestIdentifyRefusesNonLoopbackPeers(t *testing.T) {
	local := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 11434}
	remote := &net.TCPAddr{IP: net.IPv4(192, 0, 2, 7), Port: 40000}
	if got := NewResolver().Identify(local, remote); got != Unknown {
		t.Fatalf("non-loopback peer = %q, want unknown", got)
	}
	var nilResolver *Resolver
	if got := nilResolver.Identify(local, remote); got != Unknown {
		t.Fatalf("nil resolver = %q, want unknown", got)
	}
}

func TestProcSocketAddrsMatchKernelEncoding(t *testing.T) {
	v4 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 11434}
	if !procSocketAddrs(v4, false)["0100007F:2CAA"] {
		t.Fatalf("v4 encoding = %v", procSocketAddrs(v4, false))
	}
	if !procSocketAddrs(v4, true)["0000000000000000FFFF00000100007F:2CAA"] {
		t.Fatalf("v4-mapped encoding = %v", procSocketAddrs(v4, true))
	}
	v6 := &net.TCPAddr{IP: net.IPv6loopback, Port: 80}
	if !procSocketAddrs(v6, true)["00000000000000000000000001000000:0050"] {
		t.Fatalf("v6 encoding = %v", procSocketAddrs(v6, true))
	}
}

// The exact socket-diagnostics lookup must find the same socket the
// /proc/net/tcp scan does; it is the fast path, not a different answer.
func TestSocketDiagnosticsFindsTheSameSocketAsTheTCPTable(t *testing.T) {
	conn := connectFromHelper(t)
	server, client := conn.LocalAddr().(*net.TCPAddr), conn.RemoteAddr().(*net.TCPAddr)
	diagInode, diagUID, ok := diagSocketInode(client, server)
	if !ok {
		t.Fatal("socket diagnostics did not find the loopback client socket")
	}
	tableInode, tableUID, ok := clientSocketInode("/proc/net", client, server)
	if !ok {
		t.Fatal("the TCP table did not list the loopback client socket")
	}
	if diagInode != tableInode || diagUID != tableUID {
		t.Fatalf("diagnostics = %s/%d, table = %s/%d", diagInode, diagUID, tableInode, tableUID)
	}
	if _, _, ok := diagSocketInode(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1}, server); ok {
		t.Fatal("a connection that does not exist must not resolve")
	}
}

func TestNewestFirstFollowsPIDAllocationAcrossAWrap(t *testing.T) {
	pids := []int{4188311, 12, 1906510, 4000000, 1906400, 1906600}
	newestFirst(pids, 1906510)
	want := []int{1906510, 1906400, 12, 4188311, 4000000, 1906600}
	for i := range want {
		if pids[i] != want[i] {
			t.Fatalf("order = %v, want %v", pids, want)
		}
	}
	plain := []int{3, 9, 5}
	newestFirst(plain, 0)
	if plain[0] != 9 || plain[2] != 3 {
		t.Fatalf("unknown last PID must fall back to descending, got %v", plain)
	}
}

func TestLastAllocatedPIDReadsLoadavg(t *testing.T) {
	if lastAllocatedPID("/proc") <= 0 {
		t.Fatal("the live /proc/loadavg names the last allocated PID")
	}
	if lastAllocatedPID(t.TempDir()) != 0 {
		t.Fatal("a missing loadavg reads as unknown")
	}
}
