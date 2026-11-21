package wallet

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-webauthn/x/crypto/secp256k1/ecdsa"
)

// PortableRecoveryRevocation is a public tombstone signed by the wallet key
// before Treasury deletes that key. It lets a current off-host snapshot state
// that this wallet's earlier encrypted bundles are revoked.
type PortableRecoveryRevocation struct {
	Version     int    `json:"version"`
	Epoch       uint64 `json:"epoch"`
	WalletID    string `json:"wallet_id"`
	Network     string `json:"network"`
	Address     string `json:"address"`
	PublicKey   string `json:"public_key"`
	SuccessorID string `json:"successor_wallet_id"`
	RevokedAt   string `json:"revoked_at"`
	Signature   string `json:"signature"`
}

func portableRevocationMessage(r PortableRecoveryRevocation) string {
	return strings.Join([]string{
		"Vrooli Treasury portable wallet recovery revocation v1",
		"No spending authorization.",
		"wallet:" + r.WalletID,
		"network:" + r.Network,
		"address:" + r.Address,
		"public_key:" + r.PublicKey,
		"successor:" + r.SuccessorID,
		fmt.Sprintf("epoch:%d", r.Epoch),
		"revoked_at:" + r.RevokedAt,
	}, "\n")
}

// VerifyPortableRecoveryRevocation checks a tombstone against its own
// immutable public key. It never needs the operator secret or private scalar.
func VerifyPortableRecoveryRevocation(r PortableRecoveryRevocation) bool {
	if r.Version != 1 || r.Epoch < 2 || !validID.MatchString(r.WalletID) || !validID.MatchString(r.SuccessorID) || r.WalletID == r.SuccessorID || !supported(r.Network) || r.Address == "" || r.PublicKey == "" || r.RevokedAt == "" {
		return false
	}
	signature, err := hex.DecodeString(strings.TrimPrefix(r.Signature, "0x"))
	if err != nil || len(signature) != 65 || signature[64] < 27 || signature[64] > 28 {
		return false
	}
	compact := append([]byte{signature[64]}, signature[:64]...)
	public, _, err := ecdsa.RecoverCompact(compact, personalHash(portableRevocationMessage(r)))
	return err == nil && Address(public) == r.Address && hex.EncodeToString(public.SerializeUncompressed()) == r.PublicKey
}

func (s *Service) preparePortableRevocation(ctx context.Context, w Wallet) ([]byte, error) {
	if w.RevokedAt == "" || w.SuccessorID == "" {
		return nil, ErrMismatch
	}
	if s.recoveryPath == nil {
		return nil, ErrUnavailable
	}
	dir, err := s.recoveryPath(ctx, portableBackupName(w.ID))
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if info, statErr := os.Lstat(dir); statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%w: portable revocation directory must be a private directory", ErrUnavailable)
	}
	path := filepath.Join(dir, "revocation.json")
	tombstone, readErr := readBackup(path)
	if readErr == nil {
		var existing PortableRecoveryRevocation
		if json.Unmarshal(tombstone, &existing) != nil || !VerifyPortableRecoveryRevocation(existing) || existing.WalletID != w.ID || existing.Network != w.Network || existing.Address != w.Address || existing.SuccessorID != w.SuccessorID || existing.RevokedAt != w.RevokedAt {
			return nil, ErrMismatch
		}
	} else if errors.Is(readErr, os.ErrNotExist) {
		key, keyErr := s.key(w)
		if keyErr != nil {
			return nil, fmt.Errorf("%w: revoked key is absent before its off-host tombstone was written", ErrUnavailable)
		}
		defer key.Zero()
		if Address(key.PubKey()) != w.Address || hex.EncodeToString(key.PubKey().SerializeUncompressed()) != w.PublicKey {
			return nil, ErrMismatch
		}
		proof := PortableRecoveryRevocation{Version: 1, Epoch: 2, WalletID: w.ID, Network: w.Network, Address: w.Address, PublicKey: w.PublicKey, SuccessorID: w.SuccessorID, RevokedAt: w.RevokedAt}
		compact := ecdsa.SignCompact(key, personalHash(portableRevocationMessage(proof)), false)
		ethereum := append(append([]byte{}, compact[1:]...), compact[0])
		if ethereum[64] > 28 {
			return nil, ErrMismatch
		}
		proof.Signature = "0x" + hex.EncodeToString(ethereum)
		if !VerifyPortableRecoveryRevocation(proof) {
			return nil, ErrMismatch
		}
		tombstone, err = json.Marshal(proof)
		if err != nil {
			return nil, err
		}
		if err = writePrivateArtifact(path, tombstone); err != nil {
			return nil, fmt.Errorf("%w: write off-host wallet revocation tombstone: %v", ErrUnavailable, err)
		}
	} else {
		return nil, fmt.Errorf("%w: portable revocation tombstone is unsafe", ErrMismatch)
	}
	// The signed tombstone and absence of a live bundle are the exact tree that
	// the DBM publisher must verify as its newest snapshot before rotation can
	// revoke the scalar.
	if err := os.Remove(filepath.Join(dir, "wallet.bundle")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: remove revoked portable wallet bundle: %v", ErrUnavailable, err)
	}
	return append([]byte(nil), tombstone...), nil
}
