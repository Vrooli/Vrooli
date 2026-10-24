package onboard_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	onboardhandler "vrooli-bridge/handlers/onboard"
	"vrooli-bridge/internal/auth"
	internalssh "vrooli-bridge/internal/onboard/ssh"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
)

func TestGetOnboardingPublicKeyGeneratesAndNeverPublishesPrivateMaterial(t *testing.T) {
	sshSvc := internalssh.NewService(t.TempDir())
	h := onboardhandler.NewConnectHandler(onboardhandler.Deps{SSH: sshSvc})
	ctx := auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})

	resp, err := h.GetOnboardingPublicKey(ctx, connect.NewRequest(&onboardv1.GetOnboardingPublicKeyRequest{}))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, strings.HasPrefix(resp.Msg.PublicKey, "ssh-ed25519 "))
	require.Equal(t, "ssh-ed25519", resp.Msg.KeyType)
	require.NotEmpty(t, resp.Msg.Fingerprint)
	require.NotContains(t, resp.Msg.PublicKey, "PRIVATE KEY")
	_, err = os.Stat(filepath.Join(sshSvc.StateDir(), "bridge-onboard"))
	require.NoError(t, err)
}
