package control

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"device-control/internal/desktophelper"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type desktopCredentialStore struct {
	value   string
	failure error
	writes  int
}

func (s *desktopCredentialStore) Get(string, string) (string, error) {
	if s.failure != nil {
		return "", s.failure
	}
	if s.value == "" {
		return "", credentialauthority.ErrNotFound
	}
	return s.value, nil
}
func (s *desktopCredentialStore) Put(_, _, value string) error {
	s.value = value
	s.writes++
	return nil
}
func (s *desktopCredentialStore) Delete(string, string) error { s.value = ""; return nil }

func TestDesktopProvisionRetainsPinAndRefusesLostCredential(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux private helper config")
	}
	svc, _ := testService(t)
	store := &desktopCredentialStore{}
	authority, err := credentialauthority.NewAuthority(store)
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0700))
	path := filepath.Join(dir, "helper.json")
	config := desktophelper.Config{Version: 1, Surface: targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "host", HostNodeID: "host"}, OwnerScenario: "device-control", SurfaceID: "desktop"}, SessionID: "os-session", XAuthorityFile: filepath.Join(dir, "authority"), StateDirectory: filepath.Join(dir, "helper"), GrantStatusFile: filepath.Join(dir, "grants")}
	owner, err := svc.provisionDesktopAdmission(authority, path, "fake", config)
	require.NoError(t, err)
	require.NotNil(t, owner)
	require.Equal(t, 1, store.writes)
	provisioned, err := desktophelper.LoadConfig(path)
	require.NoError(t, err)
	public, err := base64.StdEncoding.DecodeString(provisioned.PublicKey)
	require.NoError(t, err)
	require.Len(t, public, 32)
	bytes, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(bytes), store.value, "helper config must not contain private seed")
	_, err = svc.provisionDesktopAdmission(authority, path, "fake", config)
	require.NoError(t, err)
	require.Equal(t, 1, store.writes, "reprovisioning must retain the key")
	store.value = ""
	_, err = svc.provisionDesktopAdmission(authority, path, "fake", config)
	var refused *credentialauthority.MintRefusedError
	require.ErrorAs(t, err, &refused)
	require.Equal(t, 1, store.writes, "credential loss must not silently rotate a pinned key")
	store.failure = errors.New("credential provider unavailable")
	_, err = svc.provisionDesktopAdmission(authority, path, "fake", config)
	require.Error(t, err)
	require.Equal(t, 1, store.writes)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, bytes, after, "failed provisioning must retain the existing public pin")
	config.SessionID = "different-session"
	_, err = svc.provisionDesktopAdmission(authority, path, "fake", config)
	require.ErrorIs(t, err, desktophelper.ErrBootstrap)
}
