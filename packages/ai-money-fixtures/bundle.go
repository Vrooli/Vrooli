package aimoneyfixtures

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type ComponentFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type ComponentManifest struct {
	ID      string          `json:"id"`
	Version string          `json:"version"`
	Files   []ComponentFile `json:"files"`
}
type SignedComponentBundle struct {
	Manifest  ComponentManifest `json:"manifest"`
	Payload   map[string][]byte `json:"-"`
	Signature []byte            `json:"signature"`
	PublicKey ed25519.PublicKey `json:"public_key"`
}

// NewSignedComponentFixture creates an in-memory bundle signed by a deterministic
// test key. The key must never be trusted outside tests.
func NewSignedComponentFixture() (SignedComponentBundle, error) {
	payload := map[string][]byte{"hello-desktop/README.md": []byte("Fixture standalone app: hello desktop\n"), "hello-desktop/.vrooli/service.json": []byte(`{"service":{"name":"hello-desktop","version":"1.0.0"}}`)}
	manifest := ComponentManifest{ID: "hello-desktop", Version: "1.0.0"}
	for path, body := range payload {
		sum := sha256.Sum256(body)
		manifest.Files = append(manifest.Files, ComponentFile{Path: path, SHA256: hex.EncodeToString(sum[:])})
	}
	// Sort before signing so the fixture encoding stays stable.
	if manifest.Files[0].Path > manifest.Files[1].Path {
		manifest.Files[0], manifest.Files[1] = manifest.Files[1], manifest.Files[0]
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return SignedComponentBundle{}, err
	}
	seed := sha256.Sum256([]byte("agent-connect-test-signing-key-v1"))
	private := ed25519.NewKeyFromSeed(seed[:])
	signature := ed25519.Sign(private, encoded)
	return SignedComponentBundle{Manifest: manifest, Payload: payload, Signature: signature, PublicKey: private.Public().(ed25519.PublicKey)}, nil
}

func (b SignedComponentBundle) Verify() error {
	for _, file := range b.Manifest.Files {
		body, ok := b.Payload[file.Path]
		if !ok {
			return fmt.Errorf("missing bundle file %q", file.Path)
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != file.SHA256 {
			return fmt.Errorf("digest mismatch for %q", file.Path)
		}
	}
	encoded, err := json.Marshal(b.Manifest)
	if err != nil {
		return err
	}
	if !ed25519.Verify(b.PublicKey, encoded, b.Signature) {
		return fmt.Errorf("fixture signature invalid")
	}
	return nil
}
