package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ed25519KnownHostsLine builds a known_hosts line for host:port, hashed like
// OpenSSH's default when hashed is true, plaintext otherwise.
func ed25519KnownHostsLine(t *testing.T, host string, port int, hashed bool) (line string, keyType string) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := gossh.NewPublicKey(priv.Public())
	if err != nil {
		t.Fatal(err)
	}
	addr := knownhosts.Normalize(net.JoinHostPort(host, strconv.Itoa(port)))
	hostField := addr
	if hashed {
		hostField = knownhosts.HashHostname(addr)
	}
	return knownhosts.Line([]string{hostField}, pub), pub.Type()
}

func writeKnownHosts(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "known_hosts")
	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPinnedHostKeyAlgorithms_HashedEd25519Entry(t *testing.T) {
	// The regression case: a hashed ed25519 entry (what the OpenSSH CLI writes on
	// accept-new) must resolve to ssh-ed25519 so the gossh dial requests it.
	line, kt := ed25519KnownHostsLine(t, "127.0.0.1", 22, true)
	if kt != gossh.KeyAlgoED25519 {
		t.Fatalf("unexpected key type %q", kt)
	}
	path := writeKnownHosts(t, line)

	got := pinnedHostKeyAlgorithms("127.0.0.1", 22, path)
	if len(got) != 1 || got[0] != gossh.KeyAlgoED25519 {
		t.Fatalf("hashed entry: got %v, want [%s]", got, gossh.KeyAlgoED25519)
	}
}

func TestPinnedHostKeyAlgorithms_PlaintextEntryAndPortDefaulting(t *testing.T) {
	line, _ := ed25519KnownHostsLine(t, "10.0.0.9", 22, false)
	path := writeKnownHosts(t, line)

	// port 0 must default to 22 and still match the pinned entry.
	got := pinnedHostKeyAlgorithms("10.0.0.9", 0, path)
	if len(got) != 1 || got[0] != gossh.KeyAlgoED25519 {
		t.Fatalf("plaintext entry: got %v, want [%s]", got, gossh.KeyAlgoED25519)
	}
}

func TestPinnedHostKeyAlgorithms_NoMatchAndMissingFile(t *testing.T) {
	line, _ := ed25519KnownHostsLine(t, "127.0.0.1", 22, true)
	path := writeKnownHosts(t, line)

	// A different host must not match this entry.
	if got := pinnedHostKeyAlgorithms("192.168.1.50", 22, path); got != nil {
		t.Fatalf("unrelated host: got %v, want nil", got)
	}
	// A missing file yields nil (first contact / TOFU).
	if got := pinnedHostKeyAlgorithms("127.0.0.1", 22, filepath.Join(t.TempDir(), "absent")); got != nil {
		t.Fatalf("missing file: got %v, want nil", got)
	}
}

func TestHostKeyAlgorithmsForType_RSAExpands(t *testing.T) {
	got := hostKeyAlgorithmsForType(gossh.KeyAlgoRSA)
	want := []string{gossh.KeyAlgoRSASHA256, gossh.KeyAlgoRSASHA512, gossh.KeyAlgoRSA}
	if len(got) != len(want) {
		t.Fatalf("rsa expansion: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rsa expansion[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
	// A non-RSA type maps to itself.
	if got := hostKeyAlgorithmsForType(gossh.KeyAlgoED25519); len(got) != 1 || got[0] != gossh.KeyAlgoED25519 {
		t.Fatalf("ed25519 mapping: got %v", got)
	}
}
