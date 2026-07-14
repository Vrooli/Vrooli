package ssh

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFirstTouchEstablishesPasswordlessSSH drives the full desired flow against
// a real in-process sshd: from {host, user, password} the service ends with
// working passwordless key-based SSH, the password destroyed, and the keypair +
// known_hosts persisted with the right perms.
func TestFirstTouchEstablishesPasswordlessSSH(t *testing.T) {
	const password = "s3cr3t-owner-pw"
	server := newTestSSHD(t, password)

	stateDir := t.TempDir()
	svc := NewService(stateDir)

	pw := []byte(password)
	res, err := svc.FirstTouch(context.Background(), FirstTouchRequest{
		Host:     server.host,
		Port:     server.port,
		User:     "vrooli-onboard",
		Password: pw,
		KeyName:  "bridge-onboard",
	})
	if err != nil {
		t.Fatalf("FirstTouch(%s) error: %v", server.fmtAddr(), err)
	}

	// Acceptance: final retest succeeded with no password.
	if !res.OK || !res.ConnectionVerified {
		t.Fatalf("expected verified passwordless SSH, got %+v", res)
	}
	if !res.KeyGenerated {
		t.Errorf("expected a freshly generated key on first touch")
	}
	if res.AlreadyPasswordless {
		t.Errorf("host was not passwordless before first touch; AlreadyPasswordless must be false")
	}
	if !res.CopyKeyAttempted {
		t.Errorf("expected the key-copy step to run")
	}
	if res.Status != StatusSuccess {
		t.Errorf("status = %q, want %q", res.Status, StatusSuccess)
	}

	// Password material zeroed after use.
	if !allZero(pw) {
		t.Errorf("password slice was not zeroed after FirstTouch: %v", pw)
	}

	// Keypair on disk 0600 under bridge state, dir 0700.
	keyPath := filepath.Join(stateDir, "bridge-onboard")
	assertMode(t, keyPath, 0o600)
	assertMode(t, stateDir, 0o700)
	assertMode(t, svc.knownHostsPath(), 0o600)

	// The public key really landed in the remote authorized_keys.
	pubBytes, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		t.Fatalf("read generated public key: %v", err)
	}
	pubLine := strings.Fields(strings.TrimSpace(string(pubBytes)))
	authorized, err := os.ReadFile(server.authorizedKeysPath())
	if err != nil {
		t.Fatalf("read remote authorized_keys: %v", err)
	}
	if len(pubLine) < 2 || !strings.Contains(string(authorized), pubLine[1]) {
		t.Errorf("installed key not found in remote authorized_keys")
	}

	// No password material persisted anywhere under the bridge state dir.
	assertNoSecretOnDisk(t, stateDir, password)

	// Idempotent re-run: the host is now passwordless, so no password is needed
	// and nothing is generated or copied.
	res2, err := svc.FirstTouch(context.Background(), FirstTouchRequest{
		Host:    server.host,
		Port:    server.port,
		User:    "vrooli-onboard",
		KeyName: "bridge-onboard",
	})
	if err != nil {
		t.Fatalf("second FirstTouch error: %v", err)
	}
	if !res2.OK || !res2.AlreadyPasswordless {
		t.Errorf("re-run should short-circuit as already passwordless, got %+v", res2)
	}
	if res2.KeyGenerated || res2.CopyKeyAttempted {
		t.Errorf("re-run must not regenerate or re-copy the key, got %+v", res2)
	}
}

// TestFirstTouchInstallsKeyAgainIdempotently proves that re-copying against a
// host that already has the key reports already-authorized rather than
// duplicating the entry — the copy step itself is replay-safe.
func TestCopyKeyIsIdempotentOnRepeat(t *testing.T) {
	const password = "pw-idem"
	server := newTestSSHD(t, password)
	stateDir := t.TempDir()
	svc := NewService(stateDir)

	if _, err := svc.GenerateKey(GenerateKeyRequest{Type: KeyTypeEd25519, Filename: "bridge-onboard"}); err != nil {
		t.Fatalf("generate key: %v", err)
	}
	keyPath := filepath.Join(stateDir, "bridge-onboard")

	first := ExecKeyCopier{}.CopyKey(context.Background(), CopyKeyRequest{
		Host:           server.host,
		Port:           server.port,
		User:           "u",
		KeyPath:        keyPath,
		KnownHostsFile: svc.knownHostsPath(),
		Password:       password,
	})
	if !first.OK || !first.KeyCopied {
		t.Fatalf("first copy should install the key, got %+v", first)
	}

	second := ExecKeyCopier{}.CopyKey(context.Background(), CopyKeyRequest{
		Host:           server.host,
		Port:           server.port,
		User:           "u",
		KeyPath:        keyPath,
		KnownHostsFile: svc.knownHostsPath(),
		Password:       password,
	})
	if !second.OK || !second.AlreadyExists || second.KeyCopied {
		t.Fatalf("second copy should be a no-op already-exists, got %+v", second)
	}

	authorized, err := os.ReadFile(server.authorizedKeysPath())
	if err != nil {
		t.Fatalf("read authorized_keys: %v", err)
	}
	pubBytes, _ := os.ReadFile(keyPath + ".pub")
	pubData := strings.Fields(strings.TrimSpace(string(pubBytes)))[1]
	if got := strings.Count(string(authorized), pubData); got != 1 {
		t.Errorf("key appears %d times in authorized_keys, want exactly 1 (no duplication)", got)
	}
}

// TestRemoteRunnerExecutesAfterFirstTouch confirms the lifted system-ssh remote
// runner works over the same key + known_hosts the first touch established —
// the seam later phases (real service install) build on.
func TestRemoteRunnerExecutesAfterFirstTouch(t *testing.T) {
	const password = "pw-run"
	server := newTestSSHD(t, password)
	stateDir := t.TempDir()
	svc := NewService(stateDir)

	pw := []byte(password)
	res, err := svc.FirstTouch(context.Background(), FirstTouchRequest{
		Host: server.host, Port: server.port, User: "u", Password: pw, KeyName: "bridge-onboard",
	})
	if err != nil || !res.OK {
		t.Fatalf("first touch failed: %+v err=%v", res, err)
	}

	cfg := NewConfig(server.host, server.port, "u", filepath.Join(stateDir, "bridge-onboard"), svc.knownHostsPath())
	out, err := ExecRunner{}.Run(context.Background(), cfg, "echo bridge-remote-ok", TestConnectionOptions())
	if err != nil {
		t.Fatalf("remote exec error: %v", err)
	}
	if !strings.Contains(out.Stdout, "bridge-remote-ok") {
		t.Errorf("remote stdout = %q, want it to contain the echoed marker", out.Stdout)
	}
}

func allZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s perms = %o, want %o", path, got, want)
	}
}

// assertNoSecretOnDisk walks dir and fails if the secret appears in any file —
// the durable proof that no password material was persisted.
func assertNoSecretOnDisk(t *testing.T, dir, secret string) {
	t.Helper()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // unreadable file is not a secret leak
		}
		if strings.Contains(string(data), secret) {
			t.Errorf("password material leaked into %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk state dir: %v", err)
	}
}
