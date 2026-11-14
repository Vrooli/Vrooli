package designcapture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func captureRequest() Request {
	html := "<html>exact preview</html>"
	digest := sha256.Sum256([]byte(html))
	hash := strings.Repeat("a", 64)
	return Request{Target: Target{Scenario: "demo", DesignID: "home", Revision: hash, RenderHash: hash, HTMLSHA256: hex.EncodeToString(digest[:]), InputsSHA256: hash, Kind: "preview", Kit: "vrooli-default", Theme: "light", Direction: "ltr"}, HTML: html, Width: 390, Height: 844}
}
func TestDurableCaptureIntentAndDispatchRecovery(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	request := captureRequest()
	op, err := repo.Create(ctx, "attempt-one", request)
	require.NoError(t, err)
	require.Equal(t, Prepared, op.State)
	retry, err := NewSQLiteRepository(db).Create(ctx, "attempt-one", request)
	require.NoError(t, err)
	require.Equal(t, op, retry)
	changed := request
	changed.Width = 1440
	_, err = repo.Create(ctx, "attempt-one", changed)
	require.ErrorIs(t, err, ErrConflict)
	dispatch, err := repo.Transition(ctx, op.ID, op.Version, Dispatching, "", nil, "")
	require.NoError(t, err)
	_, err = repo.Transition(ctx, op.ID, op.Version, Dispatching, "", nil, "")
	require.ErrorIs(t, err, ErrConflict)
	unknown, err := repo.Transition(ctx, op.ID, dispatch.Version, DispatchUnknown, "", nil, "producer response lost")
	require.NoError(t, err)
	retry, err = repo.Create(ctx, "attempt-one", request)
	require.NoError(t, err)
	require.Equal(t, DispatchUnknown, retry.State)
	_, err = repo.Transition(ctx, op.ID, unknown.Version, Dispatching, "", nil, "")
	require.ErrorIs(t, err, ErrConflict)
	running, err := repo.Transition(ctx, op.ID, unknown.Version, Running, "bas-run", nil, "")
	require.NoError(t, err)
	_, err = repo.Transition(ctx, op.ID, running.Version, Completed, "other-run", []Artifact{{Kind: "screenshot", Reference: "artifact:one"}, {Kind: "target", Reference: "artifact:target", Evidence: &TargetEvidence{RenderHash: request.Target.RenderHash, Width: 390, Height: 844}}}, "")
	require.Error(t, err)
	_, err = repo.Transition(ctx, op.ID, running.Version, Completed, "bas-run", nil, "")
	require.Error(t, err)
	cancelling, err := repo.Transition(ctx, op.ID, running.Version, CancelRequested, "bas-run", nil, "")
	require.NoError(t, err)
	// Cancellation request does not imply cancellation: producer completion may win.
	done, err := repo.Transition(ctx, op.ID, cancelling.Version, Completed, "bas-run", []Artifact{{Kind: "screenshot", Reference: "artifact:one"}, {Kind: "target", Reference: "artifact:target", Evidence: &TargetEvidence{RenderHash: request.Target.RenderHash, Width: 390, Height: 844}}}, "")
	require.NoError(t, err)
	require.Equal(t, Completed, done.State)
	require.Equal(t, request, done.Request)
	_, err = repo.Transition(ctx, op.ID, done.Version, Running, "bas-run", nil, "")
	require.ErrorIs(t, err, ErrConflict)
	_, err = db.ExecContext(ctx, `UPDATE design_capture_operations SET request_hash=? WHERE id=?`, strings.Repeat("b", 64), op.ID)
	require.NoError(t, err)
	_, err = repo.Get(ctx, op.ID)
	require.ErrorContains(t, err, "identity mismatch")
}
func TestCaptureIntentRejectsStaleHTMLAndInvalidDimensions(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	request := captureRequest()
	request.HTML += " changed"
	_, err := repo.Create(ctx, "stale", request)
	require.ErrorContains(t, err, "differs")
	request = captureRequest()
	request.Width = 0
	_, err = repo.Create(ctx, "viewport", request)
	require.ErrorContains(t, err, "dimensions")
	request = captureRequest()
	request.Target.Kind = "production"
	_, err = repo.Create(ctx, "kind", request)
	require.ErrorContains(t, err, "preview")
}
