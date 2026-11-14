package remote

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	authv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	"google.golang.org/protobuf/proto"
)

func TestPortableRecoveryRequestPreparationKeepsSecretOffArgvAndLoadsEncryptedBytes(t *testing.T) {
	secret := "a test-only operator secret that is long enough for the recovery KDF"
	ciphertext := []byte{0x8a, 0x01, 0xff, 0x00, 0x7c}
	bundlePath := filepath.Join(t.TempDir(), "wallet.bundle")
	require.NoError(t, os.WriteFile(bundlePath, ciphertext, 0o600))
	stdin := bytes.NewBufferString(secret + "\n")
	request := &authv1.RestorePortableResearchWalletRecoveryRequest{}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "secret-stdin", Bool: true, LocalOnly: true},
			{Name: "bundle-file", LocalOnly: true},
		}},
		Flags: map[string]string{"bundle-file": bundlePath}, BoolFlags: map[string]bool{"secret-stdin": true},
	})
	baseCalled := false
	prepare := portableRecoveryRequestPreparer(func(_ cliapp.OperationContext, _ string, _ proto.Message) (map[string]string, error) {
		baseCalled = true
		return map[string]string{"X-Test": "present"}, nil
	}, stdin)
	headers, err := prepare(ctx, "TreasuryAdmin.RestorePortableResearchWalletRecovery", request)
	require.NoError(t, err)
	require.True(t, baseCalled)
	require.Equal(t, "present", headers["X-Test"])
	require.Equal(t, secret, request.OperatorSecret)
	require.Equal(t, ciphertext, request.Bundle)
}

func TestPortableRecoveryRequestPreparationRequiresExplicitStdinAndRegularBundle(t *testing.T) {
	request := &authv1.ExportPortableResearchWalletRecoveryRequest{}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "secret-stdin", Bool: true, LocalOnly: true}}}})
	_, err := portableRecoveryRequestPreparer(nil, bytes.NewBufferString("must-not-be-read"))(ctx, "TreasuryAdmin.ExportPortableResearchWalletRecovery", request)
	require.ErrorContains(t, err, "--secret-stdin")
	require.Empty(t, request.OperatorSecret)

	secret := "another test-only operator secret long enough for the KDF"
	path := filepath.Join(t.TempDir(), "bundle-link")
	backing := filepath.Join(t.TempDir(), "bundle")
	require.NoError(t, os.WriteFile(backing, []byte("ciphertext"), 0o600))
	require.NoError(t, os.Symlink(backing, path))
	restore := &authv1.RestorePortableResearchWalletRecoveryRequest{}
	ctx = cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "secret-stdin", Bool: true, LocalOnly: true}, {Name: "bundle-file", LocalOnly: true}}},
		Flags:  map[string]string{"bundle-file": path}, BoolFlags: map[string]bool{"secret-stdin": true},
	})
	_, err = portableRecoveryRequestPreparer(nil, bytes.NewBufferString(secret))(ctx, "TreasuryAdmin.RestorePortableResearchWalletRecovery", restore)
	require.ErrorContains(t, err, "regular file")
	require.Empty(t, restore.Bundle)
}

func TestPortableRecoveryRequestPreparationUsesDBMRestoredServerBundleWhenPathOmitted(t *testing.T) {
	request := &authv1.RestorePortableResearchWalletRecoveryRequest{}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:    cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "secret-stdin", Bool: true, LocalOnly: true}, {Name: "bundle-file", LocalOnly: true}}},
		BoolFlags: map[string]bool{"secret-stdin": true},
	})
	_, err := portableRecoveryRequestPreparer(nil, bytes.NewBufferString("operator test secret long enough for the recovery kdf"))(ctx, "TreasuryAdmin.RestorePortableResearchWalletRecovery", request)
	require.NoError(t, err)
	require.NotEmpty(t, request.OperatorSecret)
	require.Empty(t, request.Bundle, "empty bundle tells the API to use the ciphertext already restored under its Treasury data root")
}
