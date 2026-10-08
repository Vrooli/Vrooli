package machines_test

import (
	"context"
	"testing"

	"vrooli-bridge/internal/machines"

	"github.com/stretchr/testify/require"
)

// [REQ:BRG-ME-004] New server host keys fail closed: an enrollment cannot
// replace a verified fingerprint without an explicit review action.
func TestTrustChangeRequiresReview(t *testing.T) {
	d, clk := newDB(t)
	repo := machines.NewSQLiteRepository(d, clk)
	ctx := context.Background()
	_, err := repo.UpsertTrust(ctx, machines.TrustRecord{MachineID: "m1", ClientKeyRef: "ssh-key://machine-m1", HostKeyFingerprint: "SHA256:old", HostKeyState: machines.HostKeyVerified})
	require.NoError(t, err)
	updated, err := repo.UpsertTrust(ctx, machines.TrustRecord{MachineID: "m1", ClientKeyRef: "ssh-key://machine-m1", HostKeyFingerprint: "SHA256:new", HostKeyState: machines.HostKeyVerified})
	require.NoError(t, err)
	require.Equal(t, "SHA256:old", updated.HostKeyFingerprint)
	require.Equal(t, machines.HostKeyReviewRequired, updated.HostKeyState)
	reviewed, err := repo.ReviewHostKey(ctx, "m1", "SHA256:new")
	require.NoError(t, err)
	require.Equal(t, "SHA256:new", reviewed.HostKeyFingerprint)
	require.Equal(t, machines.HostKeyVerified, reviewed.HostKeyState)
}
