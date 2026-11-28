package replenishment

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	compositionpb "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/composition"
	compositionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/composition/composition_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"music-library/internal/catalog"
)

func TestOutboxUsesReservationIdentityAndCancelsAfterDraftRollback(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(catalog.Schema)))
	store := catalog.NewStore(db)
	track := catalog.Track{SHA256: "a6ab5d3585ec7fd9bf29e3aa973509a3513de8f39d8feeaa774d94986f4f167c", Title: "c-orbit", SourcePath: "/private/existing/c-orbit.wav", TakeID: "existing-aquila-c", ReceiptPath: "/private/existing/c-receipt.json", Tags: []string{"instrumental", "launch-video"}, Purposes: []string{"launch-video"}, BankPriority: 2, Approval: "liked", Feedback: "Existing operator judgment"}
	_, err := store.Put(context.Background(), track)
	require.NoError(t, err)
	reservation, _, err := store.Reserve(context.Background(), catalog.ReserveRequest{Purpose: "launch-video", Project: "mq5-contract", Revision: "cut-1", Consumer: "mq5-run"})
	require.NoError(t, err)
	_, _, err = store.CommitReservation(context.Background(), reservation.ID, "mq5-run", "draft")
	require.NoError(t, err)

	var gotRequest compositionpb.CandidateRefillRequest
	var gotCancel compositionpb.CancelCandidateRefillRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case compositionconnect.CandidateInventoryServiceRequestRefillProcedure:
			require.NoError(t, protojson.Unmarshal(readBody(t, r), &gotRequest))
		case compositionconnect.CandidateInventoryServiceCancelRefillProcedure:
			require.NoError(t, json.Unmarshal(readBody(t, r), &gotCancel))
		default:
			t.Errorf("unexpected Music Tools endpoint %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	worker, err := New(db, nil)
	require.NoError(t, err)
	worker.resolve = func(context.Context) (string, error) { return server.URL, nil }
	didWork, err := worker.deliverOne(context.Background())
	require.NoError(t, err)
	require.True(t, didWork)
	require.Equal(t, reservation.ID, gotRequest.RequestId)
	require.Equal(t, reservation.ID, gotRequest.ReservationId)
	require.Equal(t, track.SHA256, gotRequest.Sha256)
	require.Equal(t, "launch-video", gotRequest.Purpose)
	assertOutboxState(t, db, reservation.ID, "delivered")

	_, err = store.RollbackReservation(context.Background(), reservation.ID, "mq5-run")
	require.NoError(t, err)
	didWork, err = worker.deliverOne(context.Background())
	require.NoError(t, err)
	require.True(t, didWork)
	require.Equal(t, reservation.ID, gotCancel.RequestId)
	assertOutboxState(t, db, reservation.ID, "cancelled")
}

func readBody(t *testing.T, r *http.Request) []byte {
	t.Helper()
	defer r.Body.Close()
	var data json.RawMessage
	require.NoError(t, json.NewDecoder(r.Body).Decode(&data))
	return data
}

func assertOutboxState(t *testing.T, db *sql.DB, eventID, want string) {
	t.Helper()
	var got string
	require.NoError(t, db.QueryRow(`SELECT state FROM music_catalog_replenish_outbox WHERE event_id=?`, eventID).Scan(&got))
	require.Equal(t, want, got)
}
