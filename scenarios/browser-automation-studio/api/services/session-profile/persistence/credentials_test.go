package persistence

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// [REQ:BAS-RH-J14] Rotation keeps existing identities recoverable after restart.
func TestProfileCredentialMintReuseAndRotation(t *testing.T) {
	store := &testutil.CredentialStore{}
	root := t.TempDir()
	config := FileRepositoryConfig{Authority: store.Authority}
	repo := NewFileRepositoryWithConfig(root, nil, config)
	old := &SessionProfile{ID: "old", Name: "Before rotation", StorageState: []byte(`{"cookies":[{"value":"synthetic"}]}`)}
	require.NoError(t, repo.Create(old))
	require.Equal(t, 1, store.Writes, "first use must mint exactly one credential")
	authority, err := store.Authority()
	require.NoError(t, err)
	value, err := authority.Require(profileCredentialIdentity, profileCredentialField)
	require.NoError(t, err)
	var ring profileKeyring
	require.NoError(t, json.Unmarshal([]byte(value), &ring))
	ring.Keys[2], ring.Active = bytes.Repeat([]byte{2}, 32), 2
	rotated, err := json.Marshal(ring)
	require.NoError(t, err)
	require.NoError(t, authority.Put(profileCredentialIdentity, profileCredentialField, string(rotated)))
	restarted := NewFileRepositoryWithConfig(root, nil, config)
	got, err := restarted.Get(old.ID)
	require.NoError(t, err)
	require.Equal(t, old, got, "rotation must preserve the complete old identity")
	require.NoError(t, restarted.Create(&SessionProfile{ID: "new", Name: "After rotation"}))
	data, err := os.ReadFile(filepath.Join(root, "new.json"))
	require.NoError(t, err)
	var document profileDocument
	require.NoError(t, json.Unmarshal(data, &document))
	require.Equal(t, 2, document.KeyVersion)
	require.Equal(t, 2, store.Writes, "save must reuse the active key")
	delete(ring.Keys, 1)
	missing, err := json.Marshal(ring)
	require.NoError(t, err)
	require.NoError(t, authority.Put(profileCredentialIdentity, profileCredentialField, string(missing)))
	_, err = restarted.Get(old.ID)
	require.Error(t, err, "missing historical key must fail visibly")
	require.NoError(t, authority.Put(profileCredentialIdentity, profileCredentialField, string(rotated)))
	got, err = restarted.Get(old.ID)
	require.NoError(t, err)
	require.Equal(t, old, got, "restored key must recover the original profile")
}

// [REQ:BAS-RH-J14] Missing credentials never cause implicit identity replacement.
func TestProfileCredentialLossCannotRemint(t *testing.T) {
	for _, retained := range []string{"profile-and-witness", "profile-only", "witness-only"} {
		t.Run(retained, func(t *testing.T) {
			store := &testutil.CredentialStore{}
			root := t.TempDir()
			config := FileRepositoryConfig{Authority: store.Authority}
			repo := NewFileRepositoryWithConfig(root, nil, config)
			require.NoError(t, repo.Create(&SessionProfile{ID: "identity"}))
			authority, err := store.Authority()
			require.NoError(t, err)
			original, err := authority.Require(profileCredentialIdentity, profileCredentialField)
			require.NoError(t, err)
			if retained == "profile-only" {
				require.NoError(t, os.Remove(filepath.Join(root, profileKeyWitness)))
			}
			if retained == "witness-only" {
				require.NoError(t, repo.Delete("identity"))
			}
			require.NoError(t, authority.Delete(profileCredentialIdentity, profileCredentialField))
			before, err := os.ReadDir(root)
			require.NoError(t, err)
			restarted := NewFileRepositoryWithConfig(root, nil, config)
			err = restarted.Create(&SessionProfile{ID: "replacement"})
			var refused *credentialauthority.MintRefusedError
			require.ErrorAs(t, err, &refused)
			require.Equal(t, 1, store.Writes, "credential loss must never re-mint")
			after, err := os.ReadDir(root)
			require.NoError(t, err)
			require.Equal(t, before, after, "refused save must not change file entries")
			require.NoError(t, authority.Put(profileCredentialIdentity, profileCredentialField, original))
			if retained != "witness-only" {
				got, err := restarted.Get("identity")
				require.NoError(t, err)
				require.Equal(t, ProfileID("identity"), got.ID)
				_, err = os.Stat(filepath.Join(root, profileKeyWitness))
				require.NoError(t, err, "restored data must retain persistent key history")
			}
		})
	}
}

func TestProfileCredentialFailuresPreserveSnapshot(t *testing.T) {
	for _, fault := range []string{"provider", "invalid-json", "invalid-active", "invalid-old-key", "write-witness"} {
		t.Run(fault, func(t *testing.T) {
			store := &testutil.CredentialStore{}
			files := NewMockFileSystem()
			repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{FileSystem: files, Authority: store.Authority})
			require.NoError(t, repo.Create(&SessionProfile{ID: "identity", Name: "Original"}))
			original, ok := files.GetFile("/data/identity.json")
			require.True(t, ok)
			authority, err := store.Authority()
			require.NoError(t, err)
			keyring, err := authority.Require(profileCredentialIdentity, profileCredentialField)
			require.NoError(t, err)
			var replacement string
			switch fault {
			case "provider":
				store.Err = errors.New("synthetic locked credential store")
			case "invalid-json":
				replacement = "{"
			case "invalid-active":
				replacement = `{"active":2,"keys":{}}`
			case "invalid-old-key":
				replacement = `{"active":1,"keys":{"1":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","2":"AA=="}}`
			case "write-witness":
				require.NoError(t, files.Remove("/data/"+profileKeyWitness))
				files.WriteFileErr = errors.New("synthetic full disk")
			}
			if replacement != "" {
				require.NoError(t, authority.Put(profileCredentialIdentity, profileCredentialField, replacement))
			}
			_, err = repo.Update("identity", func(p *SessionProfile) error { p.Name = "Rejected"; return nil })
			require.Error(t, err)
			after, ok := files.GetFile("/data/identity.json")
			require.True(t, ok)
			require.Equal(t, original, after, "credential failure must preserve the acknowledged bytes")
			store.Err, files.WriteFileErr = nil, nil
			require.NoError(t, authority.Put(profileCredentialIdentity, profileCredentialField, keyring))
			got, err := repo.Get("identity")
			require.NoError(t, err)
			require.Equal(t, "Original", got.Name)
		})
	}
}
