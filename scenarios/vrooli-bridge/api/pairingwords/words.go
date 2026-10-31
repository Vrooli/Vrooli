// Package pairingwords derives a short, human-checkable confirmation from the
// control-plane and node public keys. It is intentionally deterministic and
// contains no private-key material.
package pairingwords

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

var wordList = []string{
	"amber", "atlas", "bloom", "canyon", "cedar", "comet", "coral", "dawn",
	"ember", "falcon", "garden", "harbor", "ivory", "juniper", "lantern", "meadow",
	"orbit", "pebble", "quartz", "raven", "river", "saffron", "summit", "thunder",
	"velvet", "willow", "winter", "zephyr", "acorn", "beacon", "clover", "drift",
}

// Derive returns three stable words from both base64-encoded Ed25519 public
// keys. The key labels make the input order explicit and prevent accidental
// reuse as an unrelated digest.
func Derive(controlPlanePublicKey, nodePublicKey string) ([]string, error) {
	cp, err := base64.StdEncoding.DecodeString(strings.TrimSpace(controlPlanePublicKey))
	if err != nil || len(cp) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("decode control-plane public key: %w", err)
	}
	node, err := base64.StdEncoding.DecodeString(strings.TrimSpace(nodePublicKey))
	if err != nil || len(node) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("decode node public key: %w", err)
	}
	hash := sha256.New()
	hash.Write([]byte("vrooli-bridge-pairing-v1\x00"))
	hash.Write(cp)
	hash.Write([]byte{0})
	hash.Write(node)
	digest := hash.Sum(nil)
	words := make([]string, 3)
	for i := range words {
		index := int(digest[i*2])<<8 | int(digest[i*2+1])
		words[i] = wordList[index%len(wordList)]
	}
	return words, nil
}

// Fingerprint returns a short, stable label for one Ed25519 public key, in the
// shape an operator already knows from SSH host keys. It is shown beside the
// confirmation words so a person comparing two screens has a second value to
// read that is derived from the key rather than chosen by the sender.
func Fingerprint(publicKey string) string {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKey))
	if err != nil || len(key) != ed25519.PublicKeySize {
		return ""
	}
	digest := sha256.Sum256(key)
	return fmt.Sprintf("ed25519:%x…%x", digest[:2], digest[len(digest)-2:])
}

func String(controlPlanePublicKey, nodePublicKey string) (string, error) {
	words, err := Derive(controlPlanePublicKey, nodePublicKey)
	if err != nil {
		return "", err
	}
	return strings.Join(words, " "), nil
}
