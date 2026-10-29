package wallet

import (
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
