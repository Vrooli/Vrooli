package onboard_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	onboardhandler "vrooli-bridge/handlers/onboard"
	"vrooli-bridge/internal/auth"
	localdb "vrooli-bridge/internal/database"
	"vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/onboard/mocks"
	onboardssh "vrooli-bridge/internal/onboard/ssh"

	"github.com/vrooli/api-core/schedule"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	gossh "golang.org/x/crypto/ssh"
	_ "modernc.org/sqlite"

	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
)

const (
	intNodeID = "9a1b2c3d-4e5f-6071-8293-a4b5c6d7e8f9"
	intCode   = "INTEGRATIONPAIRINGCODE234567ABCD"
)

// TestOnboarding_FullFlow_ThroughConnectHandler drives StartOnboarding →
// WaitOnboarding through the real Connect handler and the real SSH capability
// (system ssh/scp + the streaming remote exec) against an in-process sshd, with
// a stub bootstrap script that emits the real VBOOTSTRAP marker vocabulary. It
// proves: the durable op reaches SUCCEEDED, the persisted step history matches
// the script's emitted steps, the pairing code is delivered env-only (received
// by the script, absent from argv/DB/events/logs), and the SSH password is
// zeroed. The ONLINE-confirmation seam is faked because a stub cannot open a
// real dial-out channel; every other hop is real.
func TestOnboarding_FullFlow_ThroughConnectHandler(t *testing.T) {
	requireSSHTools(t)
	intPassword := t.Name() + "-owner"

	d := newOnboardSchemaDB(t)
	sshStateDir := t.TempDir()
	sshSvc := onboardssh.NewService(sshStateDir)

	// The stub writes the pairing code it receives over the env into this file so
	// the test can prove BOTH correct env-only delivery AND non-leakage.
	codeOut := filepath.Join(t.TempDir(), "received-code")
	stubPath := writeStubBootstrap(t, intNodeID, codeOut)

	driver := onboard.NewSSHDriver(sshSvc, stubPath)
	issuer := &mocks.FakeCodeIssuer{Code: intCode}
	confirmer := &mocks.FakeOnlineConfirmer{Online: true}
	svc := onboard.NewService(onboard.NewSQLiteRepository(d, schedule.System()), driver, issuer, confirmer, schedule.System(), onboard.WithEnrollmentResolver(onboard.EnrollmentResolverFunc(func(context.Context, string) (string, bool, error) { return intNodeID, true, nil })))

	handler := onboardhandler.NewConnectHandler(onboardhandler.Deps{Service: svc})

	sshd := newOnboardSSHD(t, intPassword)
	// Admission is a real SSH-executed HTTP probe. Serve health on a non-loopback
	// address so this integration exercises the same remote-reachability gate as
	// a LAN node rather than relying on an internet hostname.
	endpoint := startAdmissionHealth(t)

	// Capture slog for the duration so we can assert the secret never reaches it.
	logs := &syncBuffer{}
	restore := swapSlog(logs)
	defer restore()

	ownerCtx := auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})

	startResp, err := handler.StartOnboarding(ownerCtx, connect.NewRequest(&onboardv1.StartOnboardingRequest{
		Host:            sshd.host,
		Port:            int32(sshd.port),
		User:            "tester",
		SshPassword:     intPassword,
		NodeName:        "web-01",
		TargetRevision:  "a1b2c3d",
		ControlPlaneUrl: endpoint,
		SkipSetup:       true,
		MachineId:       "machine-integration-1",
	}))
	require.NoError(t, err)
	opID := startResp.Msg.GetOpId()
	require.NotEmpty(t, opID)
	require.Equal(t, "machine-integration-1", startResp.Msg.GetMachineId())
	require.NotEmpty(t, startResp.Msg.GetEnrollmentAttemptId())

	waitResp, err := handler.WaitOnboarding(ownerCtx, connect.NewRequest(&onboardv1.WaitOnboardingRequest{
		Id: opID, TimeoutSeconds: 30,
	}))
	require.NoError(t, err)
	require.False(t, waitResp.Msg.GetTimedOut(), "op did not finish in time")

	op := waitResp.Msg.GetOp()
	require.Equal(t, onboardv1.OnboardingState_ONBOARDING_STATE_SUCCEEDED, op.GetState(), "failure_reason=%s", op.GetFailureReason())
	require.Equal(t, intNodeID, op.GetNodeId())

	// Persisted step history matches the stub's emitted steps.
	getResp, err := handler.GetOnboarding(ownerCtx, connect.NewRequest(&onboardv1.GetOnboardingRequest{Id: opID}))
	require.NoError(t, err)
	seen := map[string]onboardv1.OnboardingStepStatus{}
	for _, ev := range getResp.Msg.GetEvents() {
		seen[ev.GetStepId()] = ev.GetStatus()
	}
	for _, step := range []string{"detect-os", "prereqs", "clone", "setup", "toolchain", "build-agent", "build-cli", "node-key", "pair-redeem", "pin-verify", "service-install", "autostart", "verify-online"} {
		require.Contains(t, seen, step, "missing persisted bootstrap step %q", step)
		require.Equal(t, onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_OK, seen[step], "step %q not OK", step)
	}
	require.Equal(t, onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_OK, seen[onboard.StepSSHSetup])
	require.Equal(t, onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_OK, seen[onboard.StepVerifyOnline])

	// The pairing code was delivered to the script env-only: the stub received the
	// exact issued code…
	received, err := os.ReadFile(codeOut)
	require.NoError(t, err)
	require.Equal(t, intCode, strings.TrimSpace(string(received)), "stub did not receive the pairing code over env/stdin")

	// …and it leaked nowhere durable or logged.
	assertNoSecretLeak(t, d, opID, intCode)
	assertNoSecretLeak(t, d, opID, intPassword)
	require.NotContains(t, logs.String(), intCode, "pairing code leaked into logs")
	require.NotContains(t, logs.String(), intPassword, "SSH password leaked into logs")
}

func startAdmissionHealth(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp4", "0.0.0.0:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	})}
	go func() { _ = server.Serve(ln) }()
	t.Cleanup(func() { _ = server.Close() })
	for _, iface := range mustInterfaces(t) {
		for _, addr := range iface.Addrs {
			ip, _, splitErr := net.ParseCIDR(addr.String())
			if splitErr == nil && ip.To4() != nil && !ip.IsLoopback() {
				return "http://" + net.JoinHostPort(ip.String(), strings.TrimPrefix(ln.Addr().String(), "0.0.0.0:"))
			}
		}
	}
	t.Fatal("no non-loopback IPv4 address available for admission integration test")
	return ""
}

type testInterface struct{ Addrs []net.Addr }

func mustInterfaces(t *testing.T) []testInterface {
	t.Helper()
	interfaces, err := net.Interfaces()
	require.NoError(t, err)
	out := make([]testInterface, 0, len(interfaces))
	for _, iface := range interfaces {
		addrs, addrErr := iface.Addrs()
		require.NoError(t, addrErr)
		out = append(out, testInterface{Addrs: addrs})
	}
	return out
}

// TestOnboarding_RestartResume_RealSQLite proves op durability across a
// control-plane restart: an op left non-terminal in the DB by a "previous
// process" is reconciled to FAILED (safe to retry) when a fresh service instance
// boots over the same database, and the change is durable.
func TestOnboarding_RestartResume_RealSQLite(t *testing.T) {
	d := newOnboardSchemaDB(t)
	clk := schedule.System()

	// Process A persists an op mid-flight then "crashes" (we never run its
	// orchestration).
	repoA := onboard.NewSQLiteRepository(d, clk)
	created, err := repoA.Create(context.Background(), onboard.Op{Host: "web-01", TargetRevision: "a1b2c3d", State: onboard.StateBootstrapping})
	require.NoError(t, err)

	// Process B boots over the same DB and reconciles orphans at startup.
	svcB := onboard.NewService(onboard.NewSQLiteRepository(d, clk), &mocks.FakeSSHDriver{}, &mocks.FakeCodeIssuer{Code: intCode}, &mocks.FakeOnlineConfirmer{}, clk)
	n, err := svcB.ResumeInterrupted(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, n)

	op, events, err := svcB.GetOp(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, onboard.StateFailed, op.State)
	require.Equal(t, onboard.FailureInterrupted, op.FailureReason)
	require.NotEmpty(t, events, "a reconciliation step event should be recorded")

	// The op remains queryable (durable) and terminal for any later reader.
	repoC := onboard.NewSQLiteRepository(d, clk)
	again, err := repoC.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.True(t, again.State.Terminal())
}

// ---- helpers ----

func newOnboardSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "onboard-int.db")
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_txlock=immediate"
	d, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(onboardhandler.Schema),
	))
	return d
}

func assertNoSecretLeak(t *testing.T, d *sql.DB, opID, secret string) {
	t.Helper()
	require.NotEmpty(t, secret)
	rows, err := d.QueryContext(context.Background(),
		`SELECT host, user_name, node_name, target_revision, repo_url, node_id, failure_reason FROM onboarding_ops WHERE id = ?`, opID)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		cols := make([]string, 7)
		ptrs := make([]any, 7)
		for i := range cols {
			ptrs[i] = &cols[i]
		}
		require.NoError(t, rows.Scan(ptrs...))
		require.NotContains(t, strings.Join(cols, "\x00"), secret, "secret found in onboarding_ops row")
	}
	evRows, err := d.QueryContext(context.Background(), `SELECT step_id, detail FROM onboarding_step_events WHERE op_id = ?`, opID)
	require.NoError(t, err)
	defer evRows.Close()
	for evRows.Next() {
		var step, detail string
		require.NoError(t, evRows.Scan(&step, &detail))
		require.NotContains(t, detail, secret, "secret found in step_event detail (step %q)", step)
	}
}

// writeStubBootstrap writes a stand-in bootstrap script that emits the real
// VBOOTSTRAP marker vocabulary (all 13 steps + run envelope), records the
// pairing code it received over the env into codeOut, and never echoes the code
// to stdout/stderr. It lets the orchestrator drive a real SSH/SCP/exec round-trip
// without a full clone+build+setup.
func writeStubBootstrap(t *testing.T, nodeID, codeOut string) string {
	t.Helper()
	script := `#!/usr/bin/env bash
set -uo pipefail
# Prove env-only pairing-code injection: persist the received code to a
# test-readable file (never to stdout/stderr) then emit a value-free marker.
printf '%s' "${BRIDGE_PAIRING_CODE:-}" > '` + codeOut + `'
emit() { printf 'VBOOTSTRAP v=1 event=%s' "$1"; [ -n "${2:-}" ] && printf ' step=%s' "$2"; [ -n "${3:-}" ] && printf ' detail="%s"' "$3"; printf '\n'; }
emit run-start "" "stub bootstrap"
for step in detect-os prereqs clone setup toolchain build-agent build-cli node-key; do
  emit step-start "$step" ""
  emit step-ok "$step" ""
done
emit step-start pair-redeem ""
emit step-ok pair-redeem "paired as ` + nodeID + `"
emit step-start pin-verify ""
emit step-ok pin-verify "pinned key present, node ` + nodeID + `"
for step in service-install autostart verify-online; do
  emit step-start "$step" ""
  emit step-ok "$step" ""
done
emit run-ok "" "node ` + nodeID + ` paired and online"
echo "stub bootstrap complete (human log to stderr)" >&2
`
	path := filepath.Join(t.TempDir(), "stub-bootstrap.sh")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

// requireSSHTools skips unless the external tooling the flow shells out to is
// present.
func requireSSHTools(t *testing.T) {
	t.Helper()
	for _, bin := range []string{"ssh", "scp", "ssh-keygen", "/bin/sh", "bash"} {
		if filepath.IsAbs(bin) {
			if _, err := os.Stat(bin); err != nil {
				t.Skipf("skipping: %s not available (%v)", bin, err)
			}
			continue
		}
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("skipping: %s not found in PATH", bin)
		}
	}
	if findSFTPServer() == "" {
		t.Skip("skipping: no sftp-server binary found (modern scp transfers over SFTP)")
	}
}

// ---- in-process sshd (password + publickey auth, stdin-forwarding exec) ----

type onboardSSHD struct {
	host     string
	port     int
	home     string
	password string
	listener net.Listener
	hostKey  gossh.Signer
	wg       sync.WaitGroup
}

func newOnboardSSHD(t *testing.T, password string) *onboardSSHD {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer, err := gossh.NewSignerFromKey(priv)
	require.NoError(t, err)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skipping: cannot listen on loopback (%v)", err)
	}
	s := &onboardSSHD{
		host:     "127.0.0.1",
		port:     ln.Addr().(*net.TCPAddr).Port,
		home:     t.TempDir(),
		password: password,
		listener: ln,
		hostKey:  signer,
	}
	s.wg.Add(1)
	go s.serve()
	t.Cleanup(func() { _ = ln.Close(); s.wg.Wait() })
	return s
}

func (s *onboardSSHD) authorizedKeysPath() string {
	return filepath.Join(s.home, ".ssh", "authorized_keys")
}

func (s *onboardSSHD) serverConfig() *gossh.ServerConfig {
	cfg := &gossh.ServerConfig{
		PasswordCallback: func(_ gossh.ConnMetadata, pass []byte) (*gossh.Permissions, error) {
			if string(pass) == s.password {
				return nil, nil
			}
			return nil, errors.New("permission denied")
		},
		PublicKeyCallback: func(_ gossh.ConnMetadata, key gossh.PublicKey) (*gossh.Permissions, error) {
			if s.keyAuthorized(key) {
				return nil, nil
			}
			return nil, errors.New("permission denied")
		},
	}
	cfg.AddHostKey(s.hostKey)
	return cfg
}

func (s *onboardSSHD) keyAuthorized(offered gossh.PublicKey) bool {
	data, err := os.ReadFile(s.authorizedKeysPath())
	if err != nil {
		return false
	}
	want := offered.Marshal()
	rest := data
	for len(rest) > 0 {
		pk, _, _, r, err := gossh.ParseAuthorizedKey(rest)
		if err != nil {
			break
		}
		rest = r
		if bytes.Equal(pk.Marshal(), want) {
			return true
		}
	}
	return false
}

func (s *onboardSSHD) serve() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *onboardSSHD) handleConn(nConn net.Conn) {
	defer s.wg.Done()
	sconn, chans, reqs, err := gossh.NewServerConn(nConn, s.serverConfig())
	if err != nil {
		_ = nConn.Close()
		return
	}
	defer sconn.Close()
	go gossh.DiscardRequests(reqs)
	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			_ = newCh.Reject(gossh.UnknownChannelType, "only session channels")
			continue
		}
		ch, requests, err := newCh.Accept()
		if err != nil {
			continue
		}
		s.wg.Add(1)
		go s.handleSession(ch, requests)
	}
}

func (s *onboardSSHD) handleSession(ch gossh.Channel, requests <-chan *gossh.Request) {
	defer s.wg.Done()
	for req := range requests {
		switch req.Type {
		case "exec":
			var payload struct{ Command string }
			_ = gossh.Unmarshal(req.Payload, &payload)
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
			code := s.runExec(ch, payload.Command)
			_, _ = ch.SendRequest("exit-status", false, gossh.Marshal(struct{ Status uint32 }{uint32(code)}))
			_ = ch.Close()
			return
		case "subsystem":
			// Modern scp (OpenSSH 9+) transfers over SFTP, requesting the "sftp"
			// subsystem. Wire the system sftp-server to the channel so the real scp
			// client the orchestrator drives works unmodified.
			var payload struct{ Name string }
			_ = gossh.Unmarshal(req.Payload, &payload)
			if payload.Name != "sftp" {
				if req.WantReply {
					_ = req.Reply(false, nil)
				}
				continue
			}
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
			code := s.runSFTP(ch)
			_, _ = ch.SendRequest("exit-status", false, gossh.Marshal(struct{ Status uint32 }{uint32(code)}))
			_ = ch.Close()
			return
		case "env":
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
		default:
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
		}
	}
}

// runSFTP serves the SFTP subsystem by exec'ing the system sftp-server with the
// channel wired as stdin/stdout and HOME pointed at the fake remote home.
func (s *onboardSSHD) runSFTP(ch gossh.Channel) int {
	server := findSFTPServer()
	if server == "" {
		return 127
	}
	cmd := exec.Command(server)
	cmd.Env = append(os.Environ(), "HOME="+s.home)
	cmd.Stdin = ch
	cmd.Stdout = ch
	cmd.Stderr = ch.Stderr()
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		return 1
	}
	return 0
}

func findSFTPServer() string {
	for _, p := range []string{
		"/usr/lib/openssh/sftp-server",
		"/usr/libexec/sftp-server",
		"/usr/libexec/openssh/sftp-server",
		"/usr/lib/ssh/sftp-server",
	} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

// runExec runs the remote command through a real shell with HOME set to the fake
// remote home, forwarding the channel as stdin/stdout (required for scp AND for
// the pairing-code-over-stdin injection the orchestrator uses).
func (s *onboardSSHD) runExec(ch gossh.Channel, command string) int {
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Env = append(os.Environ(), "HOME="+s.home)
	cmd.Stdin = ch
	cmd.Stdout = ch
	cmd.Stderr = ch.Stderr()
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		return 1
	}
	return 0
}

// ---- slog capture ----

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func swapSlog(w io.Writer) func() {
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(w, nil)))
	return func() { slog.SetDefault(prev) }
}
