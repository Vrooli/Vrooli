package wallet

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// QualifyRecovery checks a persisted, scoped encrypted backup by restoring it
// into transient owner memory and matching the immutable public address.
// The passphrase remains same-host custody; independent recovery is unverified.
func (s *Service) QualifyRecovery(ctx context.Context, id, network, operator string) (Wallet, error) {
	if _, err := s.Get(ctx, id, network, operator); err != nil {
		return Wallet{}, err
	}
	err := s.repository.WithCustodyLock(ctx, id, func(repository Repository) error {
		w, err := repository.Get(ctx, id)
		if err != nil {
			return err
		}
		locked := *s
		locked.repository = repository
		return locked.qualifyRecovery(ctx, w)
	})
	if err != nil {
		return Wallet{}, err
	}
	return s.Get(ctx, id, network, operator)
}

func (s *Service) qualifyRecovery(ctx context.Context, w Wallet) error {
	if s.recoveryPath == nil {
		return ErrUnavailable
	}
	path, err := s.recoveryPath(ctx, backupName(w.ID))
	if err != nil {
		return err
	}
	identity, err := credentialauthority.ParseIdentity(w.CustodyReference)
	if err != nil {
		return ErrMismatch
	}
	pass, err := s.custody.ResolveOrMint(identity, recoveryField, witness{repository: s.repository, ctx: ctx, id: w.ID, reference: w.CustodyReference, recovery: true}, func() (string, error) { return credentialauthority.RandomBase64(32) })
	if err != nil {
		return fmt.Errorf("%w: recovery passphrase unavailable", ErrUnavailable)
	}
	blob, readErr := readBackup(path)
	switch {
	case readErr == nil:
	case errors.Is(readErr, os.ErrNotExist) && w.RecoverySHA256 == "":
		if err = ctx.Err(); err != nil {
			return err
		}
		blob, err = s.custody.ExportRecovery([]credentialauthority.RecoveryEntry{{Identity: identity, Field: keyField}}, pass)
		if err != nil {
			return fmt.Errorf("%w: encrypted recovery export failed", ErrUnavailable)
		}
		if err = verifyRecoveryBinding(blob, pass, w); err != nil {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		stat, err := os.Lstat(filepath.Dir(path))
		if err != nil || !stat.IsDir() || stat.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("%w: recovery directory must be private", ErrUnavailable)
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return fmt.Errorf("%w: recovery creation refused", ErrUnavailable)
		}
		_, writeErr := file.Write(blob)
		syncErr := file.Sync()
		closeErr := file.Close()
		if err = errors.Join(writeErr, syncErr, closeErr); err != nil {
			return err
		}
		blob, err = readBackup(path)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: recovery readback failed", ErrUnavailable)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(blob))
	if w.RecoverySHA256 != "" && digest != w.RecoverySHA256 {
		return fmt.Errorf("%w: recovery checksum mismatch", ErrMismatch)
	}
	if err = verifyRecoveryBinding(blob, pass, w); err != nil {
		return err
	}
	return s.repository.SetRecovery(ctx, w.ID, digest)
}

// ExportPortableRecoveryBundle creates an encrypted, key-only wallet bundle
// under a secret supplied by the operator. The secret is deliberately not
// resolved from or written to same-host Credential Authority. Callers must
// transfer the returned ciphertext through an independently qualified backup
// destination and keep the secret off-host.
func (s *Service) ExportPortableRecoveryBundle(ctx context.Context, id, network, operator, operatorSecret string) ([]byte, error) {
	if len(operatorSecret) < 32 || len(operatorSecret) > 4096 {
		return nil, fmt.Errorf("%w: portable recovery requires an operator-held secret of at least 32 bytes", ErrInvalid)
	}
	if _, err := s.binding(ctx, id, network, operator); err != nil {
		return nil, err
	}
	var bundle []byte
	err := s.repository.WithCustodyLock(ctx, id, func(repository Repository) error {
		w, err := repository.Get(ctx, id)
		if err != nil {
			return err
		}
		locked := *s
		locked.repository = repository
		key, err := locked.key(w)
		if err != nil {
			return err
		}
		defer key.Zero()
		if w.Address == "" || w.Address != Address(key.PubKey()) || w.PublicKey != hex.EncodeToString(key.PubKey().SerializeUncompressed()) {
			return ErrMismatch
		}
		identity, err := credentialauthority.ParseIdentity(w.CustodyReference)
		if err != nil {
			return ErrMismatch
		}
		bundle, err = locked.custody.ExportRecovery([]credentialauthority.RecoveryEntry{{Identity: identity, Field: keyField}}, operatorSecret)
		if err != nil {
			return fmt.Errorf("%w: portable recovery export failed", ErrUnavailable)
		}
		return verifyRecoveryBinding(bundle, operatorSecret, w)
	})
	if err != nil {
		return nil, err
	}
	return bundle, nil
}

type PortableRecoveryArtifact struct {
	RelativePath string
	SHA256       string
	SizeBytes    uint64
}

// PersistPortableRecoveryBundle writes the encrypted, key-only bundle under
// Treasury's data root so an existing Data Backup Manager target can snapshot
// it. The operator secret is used only to create the ciphertext; it is never
// stored with the artifact or in Credential Authority.
func (s *Service) PersistPortableRecoveryBundle(ctx context.Context, id, network, operator, operatorSecret string) (PortableRecoveryArtifact, error) {
	blob, err := s.ExportPortableRecoveryBundle(ctx, id, network, operator, operatorSecret)
	if err != nil {
		return PortableRecoveryArtifact{}, err
	}
	relativePath := portableBackupName(id)
	path, err := s.recoveryPath(ctx, relativePath)
	if err != nil {
		return PortableRecoveryArtifact{}, err
	}
	directory := filepath.Dir(path)
	if err = os.MkdirAll(directory, 0o700); err != nil {
		return PortableRecoveryArtifact{}, err
	}
	directoryInfo, err := os.Lstat(directory)
	if err != nil || !directoryInfo.IsDir() || directoryInfo.Mode().Perm()&0o077 != 0 {
		return PortableRecoveryArtifact{}, fmt.Errorf("%w: portable recovery directory must be private", ErrUnavailable)
	}
	if existing, readErr := readBackup(path); readErr == nil {
		if !bytes.Equal(existing, blob) {
			return PortableRecoveryArtifact{}, fmt.Errorf("%w: portable recovery artifact already exists with different ciphertext", ErrMismatch)
		}
		return portableRecoveryArtifact(relativePath, blob), nil
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return PortableRecoveryArtifact{}, fmt.Errorf("%w: existing portable recovery artifact is not a safe regular file", ErrMismatch)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return PortableRecoveryArtifact{}, fmt.Errorf("%w: portable recovery creation refused", ErrUnavailable)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(path)
		}
	}()
	_, writeErr := file.Write(blob)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err = errors.Join(writeErr, syncErr, closeErr); err != nil {
		return PortableRecoveryArtifact{}, err
	}
	readback, err := readBackup(path)
	if err != nil || !bytes.Equal(readback, blob) {
		return PortableRecoveryArtifact{}, fmt.Errorf("%w: portable recovery readback mismatch", ErrMismatch)
	}
	committed = true
	return portableRecoveryArtifact(relativePath, blob), nil
}

func portableRecoveryArtifact(relativePath string, blob []byte) PortableRecoveryArtifact {
	digest := sha256.Sum256(blob)
	return PortableRecoveryArtifact{RelativePath: filepath.ToSlash(relativePath), SHA256: hex.EncodeToString(digest[:]), SizeBytes: uint64(len(blob))}
}

func portableBackupName(id string) string {
	return filepath.Join("portable-recovery", id+".bundle")
}

// RestorePersistedPortableRecoveryBundle restores the encrypted artifact that
// DBM placed in this Treasury instance's private data root. This lets a second
// node recover without copying a bundle through an operator's local filesystem.
func (s *Service) RestorePersistedPortableRecoveryBundle(ctx context.Context, id, network, operator, operatorSecret string) (Wallet, error) {
	path, err := s.recoveryPath(ctx, portableBackupName(id))
	if err != nil {
		return Wallet{}, err
	}
	bundle, err := readBackup(path)
	if err != nil {
		return Wallet{}, fmt.Errorf("%w: portable recovery artifact is unavailable or unsafe", ErrUnavailable)
	}
	return s.RestorePortableRecoveryBundle(ctx, id, network, operator, bundle, operatorSecret)
}

// RestorePortableRecoveryBundle restores only the missing, previously minted
// wallet scalar. It verifies the ciphertext, external secret, wallet identity
// and immutable public address in transient memory before Credential Authority
// is changed. Existing or mismatched keys are never replaced.
func (s *Service) RestorePortableRecoveryBundle(ctx context.Context, id, network, operator string, blob []byte, operatorSecret string) (Wallet, error) {
	if len(operatorSecret) < 32 || len(operatorSecret) > 4096 || len(blob) == 0 || len(blob) > 1<<20 {
		return Wallet{}, ErrInvalid
	}
	w, err := s.binding(ctx, id, network, operator)
	if err != nil {
		return Wallet{}, err
	}
	err = s.repository.WithCustodyLock(ctx, id, func(repository Repository) error {
		w, err := repository.Get(ctx, id)
		if err != nil {
			return err
		}
		locked := *s
		locked.repository = repository
		if _, keyErr := locked.key(w); keyErr == nil {
			return fmt.Errorf("%w: portable recovery refuses to replace an existing key", ErrMismatch)
		} else if !errors.Is(keyErr, credentialauthority.ErrUnconfigured) {
			return keyErr
		}
		minted, err := repository.Minted(ctx, w.ID, false)
		if err != nil || !minted {
			return fmt.Errorf("%w: portable recovery requires a previously witnessed wallet key", ErrMismatch)
		}
		if err = verifyRecoveryBinding(blob, operatorSecret, w); err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = locked.custody.RestoreRecovery(blob, operatorSecret); err != nil {
			return fmt.Errorf("%w: portable recovery restore failed", ErrUnavailable)
		}
		key, err := locked.key(w)
		if err != nil {
			return err
		}
		defer key.Zero()
		if Address(key.PubKey()) != w.Address || hex.EncodeToString(key.PubKey().SerializeUncompressed()) != w.PublicKey {
			return ErrMismatch
		}
		return nil
	})
	if err != nil {
		return Wallet{}, err
	}
	return s.Get(ctx, w.ID, w.Network, operator)
}

func readBackup(path string) ([]byte, error) {
	stat, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !stat.Mode().IsRegular() || stat.Mode().Perm() != 0o600 || stat.Size() > 1<<20 {
		return nil, ErrMismatch
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, 1<<20))
}

// Recover restores only a witnessed missing scalar, never an existing
// mismatched key or a lost passphrase. The repository lock serializes every
// wallet-owned mint/backup/recovery operation across independent connections.
func (s *Service) Recover(ctx context.Context, id, network, operator string) (Wallet, error) {
	if _, err := s.binding(ctx, id, network, operator); err != nil {
		return Wallet{}, err
	}
	err := s.repository.WithCustodyLock(ctx, id, func(repository Repository) error {
		w, err := repository.Get(ctx, id)
		if err != nil {
			return err
		}
		locked := *s
		locked.repository = repository
		return locked.recover(ctx, w)
	})
	if err != nil {
		return Wallet{}, err
	}
	return s.Get(ctx, id, network, operator)
}

func (s *Service) recover(ctx context.Context, w Wallet) error {
	if w.Address == "" || w.PublicKey == "" || w.RecoverySHA256 == "" || s.recoveryPath == nil {
		return ErrUnavailable
	}
	identity, err := credentialauthority.ParseIdentity(w.CustodyReference)
	if err != nil {
		return ErrMismatch
	}
	key, err := s.key(w)
	if err == nil {
		defer key.Zero()
		if Address(key.PubKey()) != w.Address || hex.EncodeToString(key.PubKey().SerializeUncompressed()) != w.PublicKey {
			return ErrMismatch
		}
		return nil
	}
	if !errors.Is(err, credentialauthority.ErrUnconfigured) {
		return err
	}
	minted, err := s.repository.Minted(ctx, w.ID, false)
	if err != nil || !minted {
		return ErrMismatch
	}
	pass, err := s.custody.Resolve(identity, recoveryField)
	if err != nil {
		return ErrUnavailable
	}
	path, err := s.recoveryPath(ctx, backupName(w.ID))
	if err != nil {
		return err
	}
	blob, err := readBackup(path)
	if err != nil {
		return ErrUnavailable
	}
	if fmt.Sprintf("%x", sha256.Sum256(blob)) != w.RecoverySHA256 {
		return fmt.Errorf("%w: recovery checksum mismatch", ErrMismatch)
	}
	if err = verifyRecoveryBinding(blob, pass, w); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = s.custody.RestoreRecovery(blob, pass); err != nil {
		return ErrUnavailable
	}
	return nil
}

func verifyRecoveryBinding(blob []byte, pass string, w Wallet) error {
	identity, err := credentialauthority.ParseIdentity(w.CustodyReference)
	if err != nil {
		return ErrMismatch
	}
	manifest, err := credentialauthority.InspectRecovery(blob, pass)
	if err != nil || len(manifest.Entries) != 1 || manifest.Entries[0].Identity != identity || manifest.Entries[0].Field != keyField {
		return ErrMismatch
	}
	stage := recoveryStage{}
	defer stage.clear()
	authority, err := credentialauthority.NewAuthority(stage)
	if err != nil {
		return ErrUnavailable
	}
	if err = authority.RestoreRecovery(blob, pass); err != nil {
		return ErrMismatch
	}
	key, err := (&Service{custody: authority}).key(w)
	if err != nil {
		return ErrMismatch
	}
	defer key.Zero()
	if Address(key.PubKey()) != w.Address || hex.EncodeToString(key.PubKey().SerializeUncompressed()) != w.PublicKey {
		return ErrMismatch
	}
	return nil
}

// recoveryStage is transient validation inside the existing custody owner,
// not a persistent credential backend. It is never reachable through an API.
type recoveryStage map[string]string

func (s recoveryStage) Put(service, key, value string) error {
	s[service+"\x00"+key] = value
	return nil
}

func (s recoveryStage) Get(service, key string) (string, error) {
	value, ok := s[service+"\x00"+key]
	if !ok {
		return "", credentialauthority.ErrNotFound
	}
	return value, nil
}
func (s recoveryStage) Delete(service, key string) error { delete(s, service+"\x00"+key); return nil }
func (s recoveryStage) clear() {
	for key := range s {
		delete(s, key)
	}
}
