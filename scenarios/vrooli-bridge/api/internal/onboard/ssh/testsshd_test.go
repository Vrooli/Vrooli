package ssh

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	gossh "golang.org/x/crypto/ssh"
)

// testSSHD is an in-process OpenSSH-speaking server used to exercise the real
// first-touch flow end to end: the production code drives the system `ssh`
// binary (key test) and x/crypto password sessions (key copy) against it, and
// the server executes the very shell commands keys_copy issues via a real
// /bin/sh with $HOME pointed at a temp "remote home". Preferring this over
// mocks means the authorized_keys append, TOFU host-key handling, and
// publickey re-test are all validated for real.
type testSSHD struct {
	host     string
	port     int
	home     string // the fake remote $HOME (authorized_keys lands here)
	password string

	listener net.Listener
	hostKey  gossh.Signer
	wg       sync.WaitGroup

	envMu    sync.Mutex
	extraEnv []string // test-controlled env (PATH to fakes, sudoers dir, markers)
}

// setEnv sets the extra environment every remote exec runs with. Test values take
// precedence over the inherited process env (used to place fake sudo/visudo on
// PATH and redirect the sudoers.d dir for the sudo-provisioning tests).
func (s *testSSHD) setEnv(env ...string) {
	s.envMu.Lock()
	s.extraEnv = append([]string(nil), env...)
	s.envMu.Unlock()
}

// execEnv merges HOME + the test's extra env over the inherited process env, with
// the test values winning on duplicate keys (robust regardless of libc getenv
// resolution order).
func (s *testSSHD) execEnv() []string {
	s.envMu.Lock()
	extra := append([]string{"HOME=" + s.home}, s.extraEnv...)
	s.envMu.Unlock()

	override := make(map[string]struct{}, len(extra))
	for _, kv := range extra {
		if i := strings.IndexByte(kv, '='); i > 0 {
			override[kv[:i]] = struct{}{}
		}
	}
	out := append([]string(nil), extra...)
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 {
			if _, dup := override[kv[:i]]; dup {
				continue
			}
		}
		out = append(out, kv)
	}
	return out
}

// requireSSHTools skips the test unless the external tooling the flow shells out
// to is present, matching the plan's skip-with-reason guard.
func requireSSHTools(t *testing.T) {
	t.Helper()
	for _, bin := range []string{"ssh", "scp", "ssh-keygen", "/bin/sh"} {
		name := bin
		if filepath.IsAbs(bin) {
			if _, err := os.Stat(bin); err != nil {
				t.Skipf("skipping: %s not available (%v)", bin, err)
			}
			continue
		}
		if _, err := exec.LookPath(name); err != nil {
			t.Skipf("skipping: %s binary not found in PATH", name)
		}
	}
}

func newTestSSHD(t *testing.T, password string) *testSSHD {
	t.Helper()
	requireSSHTools(t)

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	signer, err := gossh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("host signer: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skipping: cannot listen on loopback (%v)", err)
	}

	tcpAddr := ln.Addr().(*net.TCPAddr)
	s := &testSSHD{
		host:     "127.0.0.1",
		port:     tcpAddr.Port,
		home:     t.TempDir(),
		password: password,
		listener: ln,
		hostKey:  signer,
	}

	s.wg.Add(1)
	go s.serve()
	t.Cleanup(func() {
		_ = ln.Close()
		s.wg.Wait()
	})
	return s
}

func (s *testSSHD) authorizedKeysPath() string {
	return filepath.Join(s.home, ".ssh", "authorized_keys")
}

func (s *testSSHD) serverConfig() *gossh.ServerConfig {
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

func (s *testSSHD) keyAuthorized(offered gossh.PublicKey) bool {
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

func (s *testSSHD) serve() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return // listener closed
		}
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *testSSHD) handleConn(nConn net.Conn) {
	defer s.wg.Done()
	sconn, chans, reqs, err := gossh.NewServerConn(nConn, s.serverConfig())
	if err != nil {
		_ = nConn.Close()
		return // handshake/auth failed (expected on the pre-install key test)
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

func (s *testSSHD) handleSession(ch gossh.Channel, requests <-chan *gossh.Request) {
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

// runExec runs the remote command through a real shell with HOME set to the
// fake remote home, so the keys_copy shell snippet (mkdir -p ~/.ssh, printf >>
// authorized_keys, chmod) actually executes against a real filesystem. The SSH
// channel is wired to the command's stdin so a client that streams stdin (the
// sudo password, the pairing code) reaches the remote process as it would for
// real.
func (s *testSSHD) runExec(ch gossh.Channel, command string) int {
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Env = s.execEnv()
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

// fmtAddr renders the loopback target for logging in tests.
func (s *testSSHD) fmtAddr() string { return fmt.Sprintf("%s:%d", s.host, s.port) }
