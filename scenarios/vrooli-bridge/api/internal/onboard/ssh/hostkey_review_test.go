package ssh

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForgetHostKeyRemovesOnlyReviewedHost(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, knownHostsFileName)
	key, _ := ed25519KnownHostsLine(t, "127.0.0.1", 22, false)
	other, _ := ed25519KnownHostsLine(t, "127.0.0.2", 22, false)
	require.NoError(t, os.WriteFile(path, []byte(key+"\n"+other+"\n"), 0o600))
	svc := NewService(dir)
	require.NoError(t, svc.ForgetHostKey("127.0.0.1", 22))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(data), "127.0.0.1")
	require.Contains(t, string(data), "127.0.0.2")
}
