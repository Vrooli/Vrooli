package persistence

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	profileCredentialIdentity credentialauthority.Identity = "vrooli/browser-automation-studio" // #nosec G101 -- Public authority namespace; generated key material is resolved below.
	profileCredentialField                                 = "session-profile-keyring"
	profileKeyWitness                                      = ".keyring-witness"
)

// profileKeyring retains historical versions so rotating the active key does not
// make previously acknowledged profiles unreadable. The authority stores it.
type profileKeyring struct {
	Active int            `json:"active"`
	Keys   map[int][]byte `json:"keys"`
}

func (r *FileRepository) encryptionKeys() (*profileKeyring, error) {
	authority, err := r.authority()
	if err != nil {
		return nil, fmt.Errorf("profile credential authority: %w", err)
	}
	authority.Recheck()
	value, err := authority.ResolveOrMint(profileCredentialIdentity, profileCredentialField, r, mintProfileKeys)
	if err != nil {
		return nil, err
	}
	var ring profileKeyring
	if err := json.Unmarshal([]byte(value), &ring); err != nil {
		return nil, errors.New("profile encryption keyring is invalid; restore the credential")
	}
	if ring.Active < 1 || len(ring.Keys[ring.Active]) != 32 {
		return nil, errors.New("profile encryption keyring has no valid active key")
	}
	for version, key := range ring.Keys {
		if version < 1 || len(key) != 32 {
			return nil, errors.New("profile encryption keyring contains an invalid key version")
		}
	}
	if err := r.RecordMint(profileCredentialIdentity, profileCredentialField); err != nil {
		return nil, fmt.Errorf("record profile key history: %w", err)
	}
	return &ring, nil
}

func mintProfileKeys() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	value, err := json.Marshal(profileKeyring{Active: 1, Keys: map[int][]byte{1: key}})
	return string(value), err
}

// Minted is deliberately conservative: data without its witness still proves
// prior use. Never replace a lost credential, even after all profiles are deleted.
func (r *FileRepository) Minted(credentialauthority.Identity, string) (bool, error) {
	if _, err := r.fs.Stat(filepath.Join(r.root, profileKeyWitness)); err == nil {
		return true, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	entries, err := r.fs.ReadDir(r.root)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".protected") {
			return true, nil
		}
	}
	return false, nil
}

func (r *FileRepository) RecordMint(credentialauthority.Identity, string) error {
	if _, err := r.fs.Stat(filepath.Join(r.root, profileKeyWitness)); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return r.fs.WriteFileAtomic(filepath.Join(r.root, profileKeyWitness), []byte("profile-keyring/v1\n"), 0o600)
}
