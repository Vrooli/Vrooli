package ssh

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ----------------------------------------------------------------------------
// Unit tests — the provisioner's state machine + credential handling, driven
// through a fake stream seam (no host, no ssh binary).
// ----------------------------------------------------------------------------

type recordedExec struct {
	command string
	stdin   []byte
}

// fakeStreamer records each remote exec (command + stdin) and returns programmed
// results, split by the sudo pre-check vs the write invocation.
type fakeStreamer struct {
	execs   []recordedExec
	onProbe func() (Result, error) // `sudo -n true`
	onWrite func() (Result, error) // `sudo -S ...`
}

func (f *fakeStreamer) stream(_ context.Context, _ Config, command string, opts StreamOptions) (Result, error) {
	f.execs = append(f.execs, recordedExec{command: command, stdin: append([]byte(nil), opts.Stdin...)})
	if strings.Contains(command, "sudo -n true") {
		if f.onProbe != nil {
			return f.onProbe()
		}
		return Result{ExitCode: 1}, nil
	}
	if f.onWrite != nil {
		return f.onWrite()
	}
	return Result{}, nil
}

func newFakeProvisioner(probe, write func() (Result, error)) (*execSudoProvisioner, *fakeStreamer) {
	f := &fakeStreamer{onProbe: probe, onWrite: write}
	return &execSudoProvisioner{stream: f.stream}, f
}

func ok0() (Result, error)   { return Result{ExitCode: 0}, nil }
func fail1() (Result, error) { return Result{ExitCode: 1}, nil }
func fail3() (Result, error) { return Result{ExitCode: 3}, nil }

func TestProvisionAlreadyPasswordlessSkipsWrite(t *testing.T) {
	prov, f := newFakeProvisioner(ok0, nil)
	out := prov.Provision(context.Background(), ProvisionSudoRequest{Host: "h", User: "u", Password: []byte("pw")})
	if out.State != SudoStateAlreadyPasswordless {
		t.Fatalf("state = %q, want already-passwordless", out.State)
	}
	if len(f.execs) != 1 {
		t.Fatalf("expected only the probe to run, got %d execs", len(f.execs))
	}
}

func TestProvisionPasswordUnavailableWhenNoPassword(t *testing.T) {
	prov, f := newFakeProvisioner(fail1, nil)
	out := prov.Provision(context.Background(), ProvisionSudoRequest{Host: "h", User: "u"})
	if out.State != SudoStatePasswordUnavailable {
		t.Fatalf("state = %q, want password-unavailable", out.State)
	}
	if len(f.execs) != 1 {
		t.Fatalf("no write must run without a password, got %d execs", len(f.execs))
	}
}

func TestProvisionWritesWithPasswordOnStdinOnly(t *testing.T) {
	const password = "hunter2"
	prov, f := newFakeProvisioner(fail1, ok0)
	out := prov.Provision(context.Background(), ProvisionSudoRequest{Host: "h", User: "svc-user", Password: []byte(password)})
	if out.State != SudoStateProvisioned {
		t.Fatalf("state = %q, want provisioned", out.State)
	}
	if len(f.execs) != 2 {
		t.Fatalf("expected probe + write, got %d execs", len(f.execs))
	}
	write := f.execs[1]
	// The password rides stdin (with a trailing newline for `sudo -S`)...
	if string(write.stdin) != password+"\n" {
		t.Errorf("write stdin = %q, want the password + newline", write.stdin)
	}
	// ...and NEVER the command string (the argv/log boundary).
	if strings.Contains(write.command, password) {
		t.Errorf("password leaked into the command string: %q", write.command)
	}
	if !strings.Contains(write.command, "sudo -S -p ''") {
		t.Errorf("write command should elevate via `sudo -S -p ''`, got %q", write.command)
	}
	// The grant is the documented full-sudo drop-in for the service user.
	if !strings.Contains(write.command, "svc-user ALL=(ALL) NOPASSWD: ALL") {
		t.Errorf("write command missing the drop-in content, got %q", write.command)
	}
}

func TestProvisionFailedWhenWriteExitsNonZero(t *testing.T) {
	prov, _ := newFakeProvisioner(fail1, fail3)
	out := prov.Provision(context.Background(), ProvisionSudoRequest{Host: "h", User: "u", Password: []byte("pw")})
	if out.State != SudoStateFailed {
		t.Fatalf("state = %q, want failed", out.State)
	}
}

func TestProvisionRejectsUnsafeUsername(t *testing.T) {
	prov, f := newFakeProvisioner(fail1, ok0)
	out := prov.Provision(context.Background(), ProvisionSudoRequest{Host: "h", User: "bad user; rm -rf /", Password: []byte("pw")})
	if out.State != SudoStateFailed {
		t.Fatalf("state = %q, want failed for an unsafe username", out.State)
	}
	if len(f.execs) != 0 {
		t.Errorf("no remote exec should run for an unsafe username, got %d", len(f.execs))
	}
}

// ----------------------------------------------------------------------------
// Integration tests — the REAL provisioner over system ssh against the in-process
// sshd, with a fake sudo/visudo on PATH so the write / visudo-verify / rollback /
// idempotence path executes for real against a temp sudoers.d dir.
// ----------------------------------------------------------------------------

// installFakeSudoTools writes fake `sudo` and `visudo` executables into a temp
// dir returned for prepending to the remote PATH. The fakes model just enough
// real behaviour to exercise the provisioner: password-on-stdin for `sudo -S`,
// a marker-driven `sudo -n` pre-check, and a marker-driven `visudo -c` failure.
func installFakeSudoTools(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	sudo := `#!/bin/sh
case "$1" in
  -n)
    d="${VROOLI_BRIDGE_SUDOERS_DIR:-/etc/sudoers.d}"
    [ -f "$HOME/.sudo_n_disabled" ] && exit 1
    [ -f "$d/vrooli-bridge" ] && exit 0
    [ -f "$HOME/.sudo_nopasswd" ] && exit 0
    exit 1
    ;;
  -S)
    shift
    [ "$1" = "-p" ] && shift 2
    IFS= read -r __pw
    [ "$__pw" != "$FAKE_SUDO_PASSWORD" ] && { echo "sudo: incorrect password" >&2; exit 1; }
    exec "$@"
    ;;
  *)
    exec "$@"
    ;;
esac
`
	visudo := `#!/bin/sh
[ -f "$HOME/.visudo_fail" ] && exit 1
file=""
for a in "$@"; do file="$a"; done
grep -q 'NOPASSWD' "$file" 2>/dev/null && exit 0
exit 1
`
	if err := os.WriteFile(filepath.Join(dir, "sudo"), []byte(sudo), 0o755); err != nil {
		t.Fatalf("write fake sudo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "visudo"), []byte(visudo), 0o755); err != nil {
		t.Fatalf("write fake visudo: %v", err)
	}
	return dir
}

// keyedSudoFixture stands up an sshd with the bridge key already installed and a
// fake sudo/visudo on PATH, returning the provisioner, a base request, and the
// sudoers dir the drop-in lands in.
type keyedSudoFixture struct {
	server     *testSSHD
	prov       *execSudoProvisioner
	baseReq    ProvisionSudoRequest
	sudoersDir string
	password   string
}

func newKeyedSudoFixture(t *testing.T) keyedSudoFixture {
	t.Helper()
	password := t.Name() + "-owner"
	server := newTestSSHD(t, password)
	binDir := installFakeSudoTools(t)
	sudoersDir := t.TempDir()
	server.setEnv(
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_SUDO_PASSWORD="+password,
		sudoersDirEnv+"="+sudoersDir,
	)

	stateDir := t.TempDir()
	svc := NewService(stateDir)

	// Install the bridge key (no sudo yet) so Provision can dial with key auth.
	pw := []byte(password)
	res, err := svc.FirstTouch(context.Background(), FirstTouchRequest{
		Host: server.host, Port: server.port, User: "vrooli-onboard", Password: pw, KeyName: "bridge-onboard",
	})
	if err != nil || !res.OK {
		t.Fatalf("first touch to install key failed: %+v err=%v", res, err)
	}

	return keyedSudoFixture{
		server:     server,
		prov:       &execSudoProvisioner{stream: svc.RunStreaming},
		sudoersDir: sudoersDir,
		password:   password,
		baseReq: ProvisionSudoRequest{
			Host: server.host, Port: server.port, User: "vrooli-onboard",
			KeyPath: filepath.Join(stateDir, "bridge-onboard"), KnownHostsFile: svc.knownHostsPath(),
		},
	}
}

func (f keyedSudoFixture) dropInPath() string { return filepath.Join(f.sudoersDir, "vrooli-bridge") }

// withPassword returns a copy of the base request carrying a fresh password copy.
func (f keyedSudoFixture) withPassword() ProvisionSudoRequest {
	r := f.baseReq
	r.Password = []byte(f.password)
	return r
}

func TestSudoProvisionWritesVerifiesAndIsIdempotent(t *testing.T) {
	fx := newKeyedSudoFixture(t)

	// 1. Fresh write + visudo verify.
	out := fx.prov.Provision(context.Background(), fx.withPassword())
	if out.State != SudoStateProvisioned {
		t.Fatalf("first provision state = %q, want provisioned", out.State)
	}
	assertMode(t, fx.dropInPath(), 0o440)
	content, err := os.ReadFile(fx.dropInPath())
	if err != nil {
		t.Fatalf("read drop-in: %v", err)
	}
	if got := strings.TrimSpace(string(content)); got != "vrooli-onboard ALL=(ALL) NOPASSWD: ALL" {
		t.Errorf("drop-in content = %q, want the NOPASSWD grant for the service user", got)
	}

	// 2. Idempotent re-run: the drop-in now grants NOPASSWD, so `sudo -n true`
	//    passes and no password is needed.
	out2 := fx.prov.Provision(context.Background(), fx.baseReq) // no password
	if out2.State != SudoStateAlreadyPasswordless {
		t.Fatalf("re-run state = %q, want already-passwordless", out2.State)
	}

	// 3. Byte-compare no-op: force the write path (disable the -n short-circuit) AND
	//    arm a visudo failure. The script must byte-compare the identical content
	//    and exit 0 BEFORE ever calling visudo — proving the no-op. Were the
	//    byte-compare absent it would rewrite, visudo would fail, and the state
	//    would be `failed`.
	writeMarker(t, fx.server.home, ".sudo_n_disabled")
	writeMarker(t, fx.server.home, ".visudo_fail")
	out3 := fx.prov.Provision(context.Background(), fx.withPassword())
	if out3.State != SudoStateProvisioned {
		t.Fatalf("byte-compare re-run state = %q, want provisioned (no-op)", out3.State)
	}
	if _, err := os.Stat(fx.dropInPath()); err != nil {
		t.Errorf("drop-in should be untouched by the no-op re-run: %v", err)
	}
}

func TestSudoProvisionRollsBackOnVerifyFailure(t *testing.T) {
	fx := newKeyedSudoFixture(t)
	writeMarker(t, fx.server.home, ".visudo_fail")

	out := fx.prov.Provision(context.Background(), fx.withPassword())
	if out.State != SudoStateFailed {
		t.Fatalf("state = %q, want failed when visudo rejects the drop-in", out.State)
	}
	if _, err := os.Stat(fx.dropInPath()); !os.IsNotExist(err) {
		t.Errorf("no drop-in must be left behind on verify failure (stat err = %v)", err)
	}
}

func TestSudoProvisionAlreadyPasswordlessNeedsNoPassword(t *testing.T) {
	fx := newKeyedSudoFixture(t)
	// Simulate a pre-existing NOPASSWD rule (not our drop-in).
	writeMarker(t, fx.server.home, ".sudo_nopasswd")

	out := fx.prov.Provision(context.Background(), fx.baseReq) // no password
	if out.State != SudoStateAlreadyPasswordless {
		t.Fatalf("state = %q, want already-passwordless", out.State)
	}
	if _, err := os.Stat(fx.dropInPath()); !os.IsNotExist(err) {
		t.Errorf("no drop-in should be written when sudo is already passwordless")
	}
}

func TestSudoProvisionPasswordUnavailable(t *testing.T) {
	fx := newKeyedSudoFixture(t)
	// No drop-in, no pre-existing rule, and no password → distinct, non-failing.
	out := fx.prov.Provision(context.Background(), fx.baseReq)
	if out.State != SudoStatePasswordUnavailable {
		t.Fatalf("state = %q, want password-unavailable", out.State)
	}
}

// TestFirstTouchProvisionsSudoEndToEnd drives the whole first touch with
// ProvisionSudo set against the in-process sshd + fake sudo tools: the key is
// installed, the drop-in written and verified, the result reflects it, and the
// owner password is zeroed.
func TestFirstTouchProvisionsSudoEndToEnd(t *testing.T) {
	password := t.Name() + "-owner"
	server := newTestSSHD(t, password)
	binDir := installFakeSudoTools(t)
	sudoersDir := t.TempDir()
	server.setEnv(
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_SUDO_PASSWORD="+password,
		sudoersDirEnv+"="+sudoersDir,
	)

	stateDir := t.TempDir()
	svc := NewService(stateDir)

	pw := []byte(password)
	res, err := svc.FirstTouch(context.Background(), FirstTouchRequest{
		Host: server.host, Port: server.port, User: "vrooli-onboard", Password: pw,
		KeyName: "bridge-onboard", ProvisionSudo: true,
	})
	if err != nil {
		t.Fatalf("FirstTouch error: %v", err)
	}
	if !res.OK || !res.SudoProvisioned || res.SudoState != SudoStateProvisioned {
		t.Fatalf("expected sudo provisioned, got OK=%v SudoProvisioned=%v state=%q", res.OK, res.SudoProvisioned, res.SudoState)
	}
	assertMode(t, filepath.Join(sudoersDir, "vrooli-bridge"), 0o440)
	if !allZero(pw) {
		t.Errorf("password slice not zeroed after sudo provisioning: %v", pw)
	}
	// The password never leaked to disk anywhere under the bridge state dir.
	assertNoSecretOnDisk(t, stateDir, password)
	// ...nor into the drop-in we wrote.
	assertNoSecretOnDisk(t, sudoersDir, password)
}

func writeMarker(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
		t.Fatalf("write marker %s: %v", name, err)
	}
}
