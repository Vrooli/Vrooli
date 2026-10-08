package pairing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestWritePinnedKey_WritesModeAndPath persists the key at the exact contract
// path (<dir>/control_plane.pub) with 0600 perms — what the node-agent reads at
// startup.
func TestWritePinnedKey_WritesModeAndPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyB64 := base64.StdEncoding.EncodeToString(pub)

	path, err := writePinnedKey(dir, keyB64)
	if err != nil {
		t.Fatalf("writePinnedKey: %v", err)
	}
	if want := filepath.Join(dir, "control_plane.pub"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != keyB64 {
		t.Fatalf("file contents = %q, want %q", got, keyB64)
	}

	if runtime.GOOS != "windows" { // POSIX perms only
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("perm = %o, want 600", perm)
		}
	}
}

// TestWritePinnedKey_RejectsEmptyKey guards against pinning an empty file when
// the server returns no key.
func TestWritePinnedKey_RejectsEmptyKey(t *testing.T) {
	if _, err := writePinnedKey(t.TempDir(), "   "); err == nil {
		t.Fatal("writePinnedKey accepted an empty key")
	}
}

// TestPairingCodeFrom covers the flag/env precedence the bootstrap installer
// relies on: the code may come from --code or $BRIDGE_PAIRING_CODE (kept off
// argv so ps cannot leak it), the flag wins when both are set, surrounding
// whitespace is trimmed, and neither source present is a hard error before any
// RPC burns the single-use code.
func TestPairingCodeFrom(t *testing.T) {
	cases := []struct {
		name    string
		flag    string
		env     string
		want    string
		wantErr bool
	}{
		{name: "flag only", flag: "CODE-A", env: "", want: "CODE-A"},
		{name: "env only", flag: "", env: "CODE-B", want: "CODE-B"},
		{name: "flag wins over env", flag: "CODE-A", env: "CODE-B", want: "CODE-A"},
		{name: "trims whitespace", flag: "", env: "  CODE-C \n", want: "CODE-C"},
		{name: "neither is an error", flag: "  ", env: "", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pairingCodeFrom(tc.flag, tc.env)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got code %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("pairingCodeFrom: %v", err)
			}
			if got != tc.want {
				t.Fatalf("code = %q, want %q", got, tc.want)
			}
		})
	}
}
