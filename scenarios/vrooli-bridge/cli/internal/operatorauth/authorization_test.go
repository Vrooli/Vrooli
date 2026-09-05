package operatorauth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/vrooli/vrooli/packages/proto/sealing"
)

func TestReadSealsPlaintextAndKeepsCapabilityOpaque(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	input, err := json.Marshal(map[string]string{
		"passphrase": "correct horse",
		"capability": base64.RawStdEncoding.EncodeToString([]byte("opaque-capability")),
	})
	if err != nil {
		t.Fatal(err)
	}
	// A valid X25519 public key is required by Seal; use the deterministic test
	// vector from the package contract rather than accepting a plaintext fallback.
	private, err := sealing.PrivateKeyFromRaw(key)
	if err != nil {
		t.Fatal(err)
	}
	sealed, capability, err := Read(bytes.NewReader(input), Target{
		MachineID: "machine", NodeID: "node", Target: "mini.local", Scope: "all", OperationID: "op", OperatorID: "owner", SealingPublicKey: private.PublicKey().Bytes(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(capability) != "opaque-capability" {
		t.Fatalf("capability = %q", capability)
	}
	if bytes.Contains(sealed, []byte("correct horse")) {
		t.Fatal("sealed envelope contains plaintext passphrase")
	}
	opened, err := sealing.Open(private, sealed, sealing.Context("machine", "node", "mini.local", "all", "", "op", "owner"))
	if err != nil || string(opened) != "correct horse" {
		t.Fatalf("opened sealed passphrase = %q, err=%v", opened, err)
	}
}
