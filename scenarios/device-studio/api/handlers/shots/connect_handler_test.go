package shots

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	projects "device-studio/internal/projects"
	domain "device-studio/internal/shots"
	"github.com/stretchr/testify/require"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

func TestRevisionConflictIsTypedAndRedacted(t *testing.T) {
	err := toConnectError(domain.ErrRevisionConflict{ShotID: "shot-a", Expected: 4, Current: 5}, nil)
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.NotContains(t, ce.Message(), "shot-a")
	var found bool
	for _, detail := range ce.Details() {
		message, detailErr := detail.Value()
		require.NoError(t, detailErr)
		conflict, ok := message.(*shotsv1.ShotConflict)
		if ok {
			found = true
			require.Equal(t, "shot-a", conflict.ShotId)
			require.EqualValues(t, 4, conflict.ExpectedRevision)
			require.EqualValues(t, 5, conflict.CurrentRevision)
		}
	}
	require.True(t, found)
}

func TestUnexpectedErrorsAreRedacted(t *testing.T) {
	err := toConnectError(errors.New("sqlite file /secret/path failed"), nil)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "/secret/path")
}

func TestMissingProjectFromDraftValidationMapsNotFound(t *testing.T) {
	err := toConnectError(projects.ErrProjectNotFound{ID: "project-secret"}, nil)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "project-secret")
}
