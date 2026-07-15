package ssh

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// scriptedRunner returns queued results in order, then a default auth failure.
type scriptedRunner struct {
	queue []runResult
	calls int
}

type runResult struct {
	res Result
	err error
}

func (r *scriptedRunner) Run(_ context.Context, _ Config, _ string, _ RunOptions) (Result, error) {
	i := r.calls
	r.calls++
	if i < len(r.queue) {
		return r.queue[i].res, r.queue[i].err
	}
	return Result{ExitCode: 255}, &SSHError{Category: ErrAuth, Message: "auth failed"}
}

// recordingCopier captures the password it was handed and returns a canned resp.
type recordingCopier struct {
	gotPassword string
	called      bool
	resp        CopyKeyResponse
}

func (c *recordingCopier) CopyKey(_ context.Context, req CopyKeyRequest) CopyKeyResponse {
	c.called = true
	c.gotPassword = req.Password
	return c.resp
}

// seedKeypair pre-writes a keypair so FirstTouch skips real ssh-keygen, keeping
// the credential-handling unit tests hermetic (no external binaries).
func seedKeypair(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nx\n"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".pub"), []byte("ssh-ed25519 AAAAC3Nz test\n"), 0o600); err != nil {
		t.Fatalf("write pub: %v", err)
	}
}

func TestFirstTouchZeroesPasswordAfterCopy(t *testing.T) {
	dir := t.TempDir()
	seedKeypair(t, dir, "bridge-onboard")

	runner := &scriptedRunner{queue: []runResult{
		{res: Result{ExitCode: 255}, err: &SSHError{Category: ErrAuth, Message: "Permission denied"}}, // initial test fails
		{res: Result{Stdout: "ok"}}, // final retest succeeds
	}}
	copier := &recordingCopier{resp: CopyKeyResponse{Outcome: Outcome{OK: true, Status: StatusSuccess}, KeyCopied: true}}

	svc := NewService(dir, WithRunner(runner), WithKeyCopier(copier))

	pw := []byte("hunter2")
	res, err := svc.FirstTouch(context.Background(), FirstTouchRequest{
		Host: "example.test", User: "u", Password: pw, KeyName: "bridge-onboard",
	})
	if err != nil {
		t.Fatalf("FirstTouch error: %v", err)
	}
	if !res.OK || !res.ConnectionVerified {
		t.Fatalf("expected success after copy+retest, got %+v", res)
	}
	if !copier.called || copier.gotPassword != "hunter2" {
		t.Fatalf("copier should receive the password, got called=%v pw=%q", copier.called, copier.gotPassword)
	}
	if !allZero(pw) {
		t.Errorf("password slice not zeroed: %v", pw)
	}
}

func TestFirstTouchZeroesPasswordEvenWhenCopyFails(t *testing.T) {
	dir := t.TempDir()
	seedKeypair(t, dir, "bridge-onboard")

	runner := &scriptedRunner{queue: []runResult{
		{res: Result{ExitCode: 255}, err: &SSHError{Category: ErrAuth, Message: "Permission denied"}},
	}}
	copier := &recordingCopier{resp: CopyKeyResponse{Outcome: Outcome{OK: false, Status: StatusAuthFailed, Message: "bad password"}}}

	svc := NewService(dir, WithRunner(runner), WithKeyCopier(copier))

	pw := []byte("wrong-pw")
	res, err := svc.FirstTouch(context.Background(), FirstTouchRequest{
		Host: "example.test", User: "u", Password: pw, KeyName: "bridge-onboard",
	})
	if err != nil {
		t.Fatalf("FirstTouch error: %v", err)
	}
	if res.OK {
		t.Fatalf("expected failure when copy fails, got %+v", res)
	}
	if !allZero(pw) {
		t.Errorf("password slice not zeroed on failure path: %v", pw)
	}
}

func TestFirstTouchAlreadyPasswordlessNeedsNoPassword(t *testing.T) {
	dir := t.TempDir()
	seedKeypair(t, dir, "bridge-onboard")

	runner := &scriptedRunner{queue: []runResult{{res: Result{Stdout: "ok"}}}} // initial test succeeds
	copier := &recordingCopier{}
	svc := NewService(dir, WithRunner(runner), WithKeyCopier(copier))

	res, err := svc.FirstTouch(context.Background(), FirstTouchRequest{Host: "h", User: "u", KeyName: "bridge-onboard"})
	if err != nil {
		t.Fatalf("FirstTouch error: %v", err)
	}
	if !res.OK || !res.AlreadyPasswordless {
		t.Fatalf("expected already-passwordless short-circuit, got %+v", res)
	}
	if copier.called {
		t.Errorf("copier must not run when SSH is already passwordless")
	}
}

func TestFirstTouchRequiresPasswordWhenNotYetTrusted(t *testing.T) {
	dir := t.TempDir()
	seedKeypair(t, dir, "bridge-onboard")

	runner := &scriptedRunner{queue: []runResult{
		{res: Result{ExitCode: 255}, err: &SSHError{Category: ErrAuth, Message: "Permission denied"}},
	}}
	copier := &recordingCopier{}
	svc := NewService(dir, WithRunner(runner), WithKeyCopier(copier))

	res, err := svc.FirstTouch(context.Background(), FirstTouchRequest{Host: "h", User: "u", KeyName: "bridge-onboard"})
	if err != nil {
		t.Fatalf("FirstTouch error: %v", err)
	}
	if res.OK || res.Status != StatusAuthFailed {
		t.Errorf("expected auth_failed when no password supplied and host untrusted, got %+v", res)
	}
	if copier.called {
		t.Errorf("copier must not run without a password")
	}
}

// fakeProvisioner records the request it was handed and returns a canned state.
type fakeProvisioner struct {
	called      bool
	gotPassword string
	state       SudoState
}

func (p *fakeProvisioner) Provision(_ context.Context, req ProvisionSudoRequest) ProvisionSudoResult {
	p.called = true
	p.gotPassword = string(req.Password)
	return ProvisionSudoResult{State: p.state}
}

func TestFirstTouchProvisionsSudoAfterKeyInstall(t *testing.T) {
	dir := t.TempDir()
	seedKeypair(t, dir, "bridge-onboard")

	runner := &scriptedRunner{queue: []runResult{
		{res: Result{ExitCode: 255}, err: &SSHError{Category: ErrAuth, Message: "Permission denied"}},
		{res: Result{Stdout: "ok"}},
	}}
	copier := &recordingCopier{resp: CopyKeyResponse{Outcome: Outcome{OK: true, Status: StatusSuccess}, KeyCopied: true}}
	prov := &fakeProvisioner{state: SudoStateProvisioned}

	svc := NewService(dir, WithRunner(runner), WithKeyCopier(copier), WithSudoProvisioner(prov))

	pw := []byte("hunter2")
	res, err := svc.FirstTouch(context.Background(), FirstTouchRequest{
		Host: "example.test", User: "u", Password: pw, KeyName: "bridge-onboard", ProvisionSudo: true,
	})
	if err != nil {
		t.Fatalf("FirstTouch error: %v", err)
	}
	if !res.OK || !res.SudoProvisioned || res.SudoState != SudoStateProvisioned {
		t.Fatalf("expected sudo provisioned, got %+v", res)
	}
	if !prov.called || prov.gotPassword != "hunter2" {
		t.Fatalf("provisioner should receive the still-live password, got called=%v pw=%q", prov.called, prov.gotPassword)
	}
	// The longer hold window still ends zeroed on the success path.
	if !allZero(pw) {
		t.Errorf("password slice not zeroed after sudo provisioning: %v", pw)
	}
}

func TestFirstTouchDeclinesSudoWhenNotRequested(t *testing.T) {
	dir := t.TempDir()
	seedKeypair(t, dir, "bridge-onboard")

	runner := &scriptedRunner{queue: []runResult{
		{res: Result{ExitCode: 255}, err: &SSHError{Category: ErrAuth, Message: "Permission denied"}},
		{res: Result{Stdout: "ok"}},
	}}
	copier := &recordingCopier{resp: CopyKeyResponse{Outcome: Outcome{OK: true, Status: StatusSuccess}, KeyCopied: true}}
	prov := &fakeProvisioner{state: SudoStateProvisioned}

	svc := NewService(dir, WithRunner(runner), WithKeyCopier(copier), WithSudoProvisioner(prov))

	pw := []byte("hunter2")
	res, err := svc.FirstTouch(context.Background(), FirstTouchRequest{
		Host: "example.test", User: "u", Password: pw, KeyName: "bridge-onboard", ProvisionSudo: false,
	})
	if err != nil {
		t.Fatalf("FirstTouch error: %v", err)
	}
	if res.SudoProvisioned || res.SudoState != SudoStateDeclined {
		t.Fatalf("expected declined sudo, got %+v", res)
	}
	if prov.called {
		t.Errorf("provisioner must not run when ProvisionSudo is false")
	}
	if !allZero(pw) {
		t.Errorf("password slice not zeroed on declined path: %v", pw)
	}
}

func TestFirstTouchAlreadyPasswordlessStillProvisionsSudo(t *testing.T) {
	dir := t.TempDir()
	seedKeypair(t, dir, "bridge-onboard")

	// Initial passwordless test succeeds — the key already works. No password is
	// supplied (a pure re-run); the provisioner reports password-unavailable.
	runner := &scriptedRunner{queue: []runResult{{res: Result{Stdout: "ok"}}}}
	copier := &recordingCopier{}
	prov := &fakeProvisioner{state: SudoStatePasswordUnavailable}

	svc := NewService(dir, WithRunner(runner), WithKeyCopier(copier), WithSudoProvisioner(prov))

	res, err := svc.FirstTouch(context.Background(), FirstTouchRequest{
		Host: "h", User: "u", KeyName: "bridge-onboard", ProvisionSudo: true,
	})
	if err != nil {
		t.Fatalf("FirstTouch error: %v", err)
	}
	if !res.OK || !res.AlreadyPasswordless {
		t.Fatalf("expected already-passwordless short-circuit, got %+v", res)
	}
	if !prov.called {
		t.Errorf("sudo provisioning must still run on the already-passwordless SSH path")
	}
	if res.SudoProvisioned || res.SudoState != SudoStatePasswordUnavailable {
		t.Errorf("expected password-unavailable sudo state, got %+v", res)
	}
	if copier.called {
		t.Errorf("copier must not run when SSH is already passwordless")
	}
}

func TestFirstTouchRejectsEmptyHost(t *testing.T) {
	svc := NewService(t.TempDir())
	if _, err := svc.FirstTouch(context.Background(), FirstTouchRequest{User: "u"}); err == nil {
		t.Fatal("expected an error for empty host")
	}
}

func TestValidateKeyPathRejectsTraversal(t *testing.T) {
	svc := NewService("/var/lib/bridge/onboard-ssh")
	if err := svc.validateKeyPath("/var/lib/bridge/onboard-ssh/../secrets"); err == nil {
		t.Error("expected traversal to be rejected")
	}
	if err := svc.validateKeyPath("/etc/passwd"); err == nil {
		t.Error("expected out-of-state-dir path to be rejected")
	}
	if err := svc.validateKeyPath("/var/lib/bridge/onboard-ssh/bridge-onboard"); err != nil {
		t.Errorf("expected in-dir path to be accepted, got %v", err)
	}
}
