package activitynetwork

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func TestRootlessKitArgsPublishOnlyTheLoopbackEdge(t *testing.T) {
	cfg := Config{
		StateDir: filepath.Join(string(filepath.Separator), "tmp", "activity-network"),
		HostPort: 11453, GuestPort: 11453, PublishEdge: true, UID: 1000, GID: 1000,
		ChildCommand: []string{"/opt/vrooli/resource-reranker", "activity-runtime", "--backend", "/opt/reranker/server"},
	}
	args, err := RootlessKitArgs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"--net=slirp4netns", "--disable-host-loopback", "--slirp4netns-sandbox=true", "--slirp4netns-seccomp=true", "--port-driver=builtin", "127.0.0.1:11453:11453/tcp", "/opt/vrooli/resource-reranker", "--backend /opt/reranker/server"} {
		if !strings.Contains(joined, want) {
			t.Errorf("rootlesskit args %q missing %q", joined, want)
		}
	}
	if strings.Contains(joined, "21453:") || strings.Contains(joined, "0.0.0.0:") {
		t.Fatalf("rootlesskit args expose an upstream/wildcard port: %q", joined)
	}
}

func TestMissionResidencyManifestsHaveDistinctEdgeAndNativePorts(t *testing.T) {
	tests := []struct {
		resource string
		edge     int
		upstream int
	}{
		{resource: "whisper", edge: 8090, upstream: 18090},
		{resource: "reranker", edge: 11453, upstream: 21453},
		{resource: "ollama", edge: 11434, upstream: 21434},
		{resource: "kyutai-stt", edge: 8094, upstream: 18094},
	}
	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			var manifest struct {
				Ports []struct {
					Name      string `json:"name"`
					Container int    `json:"container"`
					Host      int    `json:"host"`
				} `json:"ports"`
				ActivityNetwork struct {
					EdgePort     string `json:"edge_port"`
					UpstreamPort string `json:"upstream_port"`
				} `json:"activity_network"`
			}
			path := filepath.Join("..", "..", "..", "resources", tt.resource, "resource.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				t.Fatal(err)
			}
			ports := map[string]struct{ host, container int }{}
			for _, port := range manifest.Ports {
				ports[port.Name] = struct{ host, container int }{port.Host, port.Container}
			}
			if ports["http"].host != tt.edge || ports["upstream"].host != 0 || ports["upstream"].container != tt.upstream {
				t.Fatalf("manifest edge/upstream ports = %+v/%+v, want host %d/private %d", ports["http"], ports["upstream"], tt.edge, tt.upstream)
			}
			if manifest.ActivityNetwork.EdgePort != "http" || manifest.ActivityNetwork.UpstreamPort != "upstream" {
				t.Fatalf("activity_network declaration = %+v", manifest.ActivityNetwork)
			}
			cfg := Config{StateDir: "/tmp/" + tt.resource, UID: 1000, GID: 1000, ChildCommand: []string{"/opt/vrooli/" + tt.resource, "serve"}}
			args, err := RootlessKitArgs(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(strings.Join(args, " "), "--publish") || strings.Contains(strings.Join(args, " "), fmt.Sprintf(":%d", tt.upstream)) {
				t.Fatalf("rootless manifest route unexpectedly publishes a TCP port: %q", args)
			}
		})
	}
}

func TestRootlessKitConfigRejectsUnsafeNetworkInputs(t *testing.T) {
	base := Config{StateDir: "/tmp/activity-network", HostPort: 11434, GuestPort: 11434, UID: 1000, GID: 1000, ChildCommand: []string{"/usr/bin/resource-ollama"}}
	cases := map[string]func(*Config){
		"relative state":           func(c *Config) { c.StateDir = "relative" },
		"wildcard host port value": func(c *Config) { c.HostPort = 0 },
		"out-of-range guest port":  func(c *Config) { c.GuestPort = 65536 },
		"negative uid":             func(c *Config) { c.UID = -1 },
		"path search command":      func(c *Config) { c.ChildCommand[0] = "resource-ollama" },
		"empty command":            func(c *Config) { c.ChildCommand = nil },
		"nul argument":             func(c *Config) { c.ChildCommand = append(c.ChildCommand, "bad\x00arg") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := base
			cfg.ChildCommand = append([]string(nil), base.ChildCommand...)
			mutate(&cfg)
			if _, err := RootlessKitArgs(cfg); err == nil {
				t.Fatal("unsafe activity network configuration was accepted")
			}
		})
	}
}

func TestOwnerNamespaceArgsRequireAnExactOwnerCommand(t *testing.T) {
	args, err := OwnerNamespaceArgs(321, []string{"/usr/bin/resource-ollama", "capacity", "degrade", "--to", "cpu"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--user", "--preserve-credentials", "--net", "--target", "321", "--", "/usr/bin/resource-ollama", "capacity", "degrade", "--to", "cpu"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("owner namespace argv = %q, want %q", args, want)
	}
	for name, mutate := range map[string]func(*int, *[]string){
		"invalid pid":  func(pid *int, _ *[]string) { *pid = 1 },
		"path lookup":  func(_ *int, command *[]string) { (*command)[0] = "resource-ollama" },
		"nul argument": func(_ *int, command *[]string) { (*command)[1] = "bad\x00arg" },
	} {
		t.Run(name, func(t *testing.T) {
			pid, command := 321, []string{"/usr/bin/resource-ollama", "capacity"}
			mutate(&pid, &command)
			if _, err := OwnerNamespaceArgs(pid, command); err == nil {
				t.Fatal("unsafe owner namespace command was accepted")
			}
		})
	}
}

func TestPrepareSingleIDMapHelpersRestrictsMappingsToOwner(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "helpers")
	if err := PrepareSingleIDMapHelpers(dir, 1234, 2345); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"newuidmap": `[ "$3" = 1234 ]`,
		"newgidmap": `[ "$3" = 2345 ]`,
	} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Errorf("%s mode = %04o, want 0700", name, info.Mode().Perm())
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), want) {
			t.Errorf("%s helper does not write only the owner mapping", name)
		}
		if strings.Contains(string(body), "100000") || strings.Contains(string(body), "65536") {
			t.Errorf("%s helper grants subordinate IDs", name)
		}
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("helper directory mode = %04o, want 0700", info.Mode().Perm())
	}
}

// TestRootlessChildEdge is run by TestRootlessKitPublishesDeclaredEdgeOnly in
// a temporary network namespace. It is inert in the normal package run.
func TestRootlessChildEdge(t *testing.T) {
	if os.Getenv("VROOLI_ACTIVITY_NETWORK_TEST_CHILD") != "1" {
		return
	}
	server := func(address, body string) *http.Server {
		return &http.Server{Addr: address, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		})}
	}
	edge := server("0.0.0.0:"+os.Getenv("VROOLI_ACTIVITY_NETWORK_TEST_GUEST_PORT"), "EDGE")
	upstream := server("127.0.0.1:"+os.Getenv("VROOLI_ACTIVITY_NETWORK_TEST_UPSTREAM_PORT"), "UPSTREAM")
	if socket := os.Getenv("VROOLI_ACTIVITY_NETWORK_TEST_PEER_SOCKET"); socket != "" {
		if err := startRootlessPeerUIDProbe(socket, os.Getenv("VROOLI_ACTIVITY_NETWORK_TEST_PEER_RESULT")); err != nil {
			t.Fatal(err)
		}
	}
	go func() { _ = edge.ListenAndServe() }()
	go func() { _ = upstream.ListenAndServe() }()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = edge.Shutdown(shutdownCtx)
	_ = upstream.Shutdown(shutdownCtx)
}

// TestRootlessOwnerProbe is invoked with nsenter by the integration test. It
// proves the owner can reach the private daemon port without publishing it.
func TestRootlessOwnerProbe(t *testing.T) {
	if os.Getenv("VROOLI_ACTIVITY_NETWORK_TEST_OWNER_PROBE") != "1" {
		return
	}
	response, err := http.Get("http://127.0.0.1:" + os.Getenv("VROOLI_ACTIVITY_NETWORK_TEST_UPSTREAM_PORT"))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fmt.Fprint(os.Stdout, string(body))
}

func TestRootlessKitPublishesDeclaredEdgeOnly(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("rootless network namespaces are Linux-only")
	}
	rootlesskit, err := exec.LookPath("rootlesskit")
	if err != nil {
		t.Skip("rootlesskit is not installed")
	}
	if _, err := os.Stat("/etc/subuid"); err != nil {
		t.Skip("rootless subordinate UID mapping is unavailable")
	}
	if _, err := os.Stat("/etc/subgid"); err != nil {
		t.Skip("rootless subordinate GID mapping is unavailable")
	}
	uid, gid := os.Getuid(), os.Getgid()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	hostPort := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	guestPort := reserveLoopbackPort(t)
	upstreamPort := reserveLoopbackPort(t)
	for upstreamPort == hostPort {
		upstreamPort = reserveLoopbackPort(t)
	}
	root, err := os.MkdirTemp("/tmp", "vrooli-activity-network-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	stateDir := filepath.Join(root, "rootlesskit")
	helperDir := filepath.Join(root, "uidmap")
	if err := PrepareSingleIDMapHelpers(helperDir, uid, gid); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		StateDir: stateDir, HostPort: hostPort, GuestPort: guestPort, PublishEdge: true, UID: uid, GID: gid,
		ChildCommand: []string{executable, "-test.run=^TestRootlessChildEdge$"},
	}
	args, err := RootlessKitArgs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(rootlesskit, args...)
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "PATH=") {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env,
		"PATH="+helperDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"VROOLI_ACTIVITY_NETWORK_TEST_CHILD=1",
		"VROOLI_ACTIVITY_NETWORK_TEST_GUEST_PORT="+strconv.Itoa(guestPort),
		"VROOLI_ACTIVITY_NETWORK_TEST_UPSTREAM_PORT="+strconv.Itoa(upstreamPort),
	)
	peerSocket := filepath.Join(root, "peer.sock")
	peerResult := filepath.Join(root, "peer.txt")
	cmd.Env = append(cmd.Env, "VROOLI_ACTIVITY_NETWORK_TEST_PEER_SOCKET="+peerSocket, "VROOLI_ACTIVITY_NETWORK_TEST_PEER_RESULT="+peerResult)
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		t.Fatalf("start rootless activity network: %v", err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			_ = cmd.Wait()
		}
	})

	client := &http.Client{Timeout: 250 * time.Millisecond}
	endpoint := "http://127.0.0.1:" + strconv.Itoa(hostPort) + "/"
	deadline := time.Now().Add(12 * time.Second)
	edgeObserved := false
	for time.Now().Before(deadline) {
		if cmd.ProcessState != nil {
			t.Fatalf("rootless network exited before the edge answered: %s", output.String())
		}
		response, requestErr := client.Get(endpoint)
		if requestErr == nil {
			body, readErr := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(body) != "EDGE" {
				t.Fatalf("host loopback edge response = %q, want EDGE", body)
			}
			edgeObserved = true
			break
		}
		if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
			t.Fatalf("rootless network child exited: %s", output.String())
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !edgeObserved {
		t.Fatalf("host loopback edge did not become ready: %s", output.String())
	}
	if err := probeUnpublishedUpstream(hostPort, upstreamPort); err != nil {
		t.Fatalf("unpublished upstream contract: %v", err)
	}
	childPID, err := os.ReadFile(filepath.Join(stateDir, "child_pid"))
	if err != nil {
		t.Fatalf("read RootlessKit child PID: %v (%s)", err, output.String())
	}
	childPIDValue, err := strconv.Atoi(strings.TrimSpace(string(childPID)))
	if err != nil {
		t.Fatalf("parse RootlessKit child PID %q: %v", childPID, err)
	}
	peerConn, err := net.DialTimeout("unix", peerSocket, time.Second)
	if err != nil {
		t.Fatalf("connect RootlessKit peer identity probe: %v", err)
	}
	_ = peerConn.Close()
	peerDeadline := time.Now().Add(2 * time.Second)
	var peerIdentity []byte
	for time.Now().Before(peerDeadline) {
		peerIdentity, err = os.ReadFile(peerResult)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("read RootlessKit Unix peer identity: %v", err)
	}
	var peerPID, peerUID int
	if _, err := fmt.Sscanf(string(peerIdentity), "%d %d", &peerPID, &peerUID); err != nil {
		t.Fatalf("parse RootlessKit Unix peer identity %q: %v", peerIdentity, err)
	}
	if peerPID != os.Getpid() || peerUID != 0 {
		t.Fatalf("RootlessKit child sees Unix peer pid=%d uid=%d, want host caller pid=%d mapped uid=0", peerPID, peerUID, os.Getpid())
	}
	if parent := cliutil.ProcessParent(childPIDValue); parent != cmd.Process.Pid {
		t.Fatalf("RootlessKit child parent PID = %d, want managed RootlessKit PID %d", parent, cmd.Process.Pid)
	}
	ownerArgs, err := OwnerNamespaceArgs(childPIDValue, []string{executable, "-test.run=^TestRootlessOwnerProbe$"})
	if err != nil {
		t.Fatal(err)
	}
	ownerProbe := exec.Command("nsenter", ownerArgs...)
	ownerProbe.Env = append(cmd.Env, "VROOLI_ACTIVITY_NETWORK_TEST_OWNER_PROBE=1")
	ownerOutput, err := ownerProbe.CombinedOutput()
	if err != nil {
		t.Fatalf("owner access to private upstream via nsenter failed: %v: %s", err, ownerOutput)
	}
	if !bytes.Contains(ownerOutput, []byte("UPSTREAM")) {
		t.Fatalf("owner private upstream response = %q, want UPSTREAM", ownerOutput)
	}
}

func reserveLoopbackPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func probeUnpublishedUpstream(_ int, upstreamPort int) error {
	conn, err := net.DialTimeout("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(upstreamPort)), 250*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return fmt.Errorf("host unexpectedly connected to upstream port %d", upstreamPort)
	}
	return nil
}
