// Package operatorauth contains the one local secret-intake contract shared by
// Bridge's operator commands. It accepts authorization only as a JSON object on
// stdin, seals plaintext to the node key immediately, and returns opaque bytes
// for the Connect request. It never reads environment variables or files.
package operatorauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/packages/proto/sealing"
)

type Target struct {
	MachineID        string
	NodeID           string
	Target           string
	Scope            string
	PlanHash         string
	OperationID      string
	OperatorID       string
	SealingPublicKey []byte
}

// Read accepts either an already sealed envelope or a plaintext passphrase.
// Plaintext is transient stdin input and is sealed before the caller can send
// it to the control plane. Capability bytes are kept opaque for resume flows.
func Read(input io.Reader, target Target) ([]byte, []byte, error) {
	if input == nil {
		return nil, nil, fmt.Errorf("cleanup authorization requires an authorization JSON object on stdin")
	}
	var raw struct {
		Passphrase       string `json:"passphrase"`
		SealedPassphrase string `json:"sealed_passphrase"`
		Capability       string `json:"capability"`
	}
	if err := json.NewDecoder(io.LimitReader(input, 128*1024)).Decode(&raw); err != nil {
		return nil, nil, fmt.Errorf("read cleanup authorization from stdin: %w", err)
	}
	if strings.TrimSpace(raw.SealedPassphrase) == "" && strings.TrimSpace(raw.Passphrase) == "" {
		return nil, nil, fmt.Errorf("cleanup authorization must contain passphrase or sealed_passphrase")
	}
	sealed, err := decodeOpaque(raw.SealedPassphrase, "sealed_passphrase")
	if err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(raw.Passphrase) != "" {
		if len(target.SealingPublicKey) == 0 {
			return nil, nil, fmt.Errorf("cleanup target did not publish a sealing key")
		}
		sealed, err = SealPassphrase(raw.Passphrase, target)
		if err != nil {
			return nil, nil, err
		}
	}
	capability, err := decodeOpaque(raw.Capability, "capability")
	if err != nil {
		return nil, nil, err
	}
	return sealed, capability, nil
}

// SealPassphrase is the interactive counterpart to Read. It exists so the
// canonical onboarding flow can prompt after pairing has published the node
// key, while keeping the same sealing/AAD contract as standalone recovery.
func SealPassphrase(passphrase string, target Target) ([]byte, error) {
	if strings.TrimSpace(passphrase) == "" {
		return nil, fmt.Errorf("cleanup authorization requires a non-empty passphrase")
	}
	if len(target.SealingPublicKey) == 0 {
		return nil, fmt.Errorf("cleanup target did not publish a sealing key")
	}
	plaintext := []byte(passphrase)
	defer zeroBytes(plaintext)
	aad := sealing.Context(target.MachineID, target.NodeID, target.Target, target.Scope, target.PlanHash, target.OperationID, target.OperatorID)
	sealed, err := sealing.Seal(target.SealingPublicKey, plaintext, aad)
	if err != nil {
		return nil, fmt.Errorf("seal cleanup passphrase: %w", err)
	}
	return sealed, nil
}

func zeroBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}

func decodeOpaque(value, field string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	decoded, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(value)
	}
	if err != nil {
		return nil, fmt.Errorf("%s must be base64-encoded opaque bytes: %w", field, err)
	}
	return decoded, nil
}
