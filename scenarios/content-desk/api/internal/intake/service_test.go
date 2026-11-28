package intake_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"content-desk/internal/intake"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func setup(t *testing.T) (*sql.DB, intake.Service) {
	t.Helper()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(intake.Schema)))
	return db, intake.NewService(db)
}

func source() intake.Source {
	return intake.Source{ArtifactID: "fixture-doc-1", RevisionID: "rev-2", Producer: "synthetic-fixture", Intent: "article-import"}
}
func good(id string) intake.Item {
	return intake.Item{ExternalID: id, Kind: "article", Title: "Fixture article", Body: "Synthetic content for intake tests."}
}

func TestPreviewAndAdmissionPreserveEverySourceFieldAndPerItemReceipt(t *testing.T) {
	db, s := setup(t)
	defer db.Close()
	ctx := context.Background()
	a := good("a")
	a.MediaPlacements = []json.RawMessage{json.RawMessage(`{"position":2,"url":"https://example.invalid/image.png","alt":"fixture"}`)}
	a.Variants = []intake.Variant{{ID: "v1", Name: "Short version", Channel: "linkedin", Body: "Short synthetic version.", RawFields: map[string]json.RawMessage{"tone": json.RawMessage(`"plain"`)}}}
	a.RawFields = map[string]json.RawMessage{"editor_note": json.RawMessage(`{"keep":true}`)}
	b := good("b")
	b.RejectedFields = []string{"unsafe_claim"}
	b.RawFields = map[string]json.RawMessage{"unsafe_claim": json.RawMessage(`"synthetic rejected claim"`)}
	c := good("c")
	c.Body = ""
	c.RawFields = map[string]json.RawMessage{"unknown": json.RawMessage(`{"nested":[1,2]}`)}
	req := intake.Request{Source: source(), IdempotencyKey: "request-1", Items: []intake.Item{a, b, c}}
	preview, err := s.Preview(ctx, req)
	require.NoError(t, err)
	require.Equal(t, []string{"admissible", "rejected", "unresolved"}, statuses(preview.Assessments))
	items, err := s.Admit(ctx, req)
	require.NoError(t, err)
	require.Len(t, items, 3)
	require.Equal(t, "admissible", items[0].Assessment.Status)
	require.Len(t, items[0].Receipts, 1)
	require.Equal(t, "Short version", items[0].Item.Variants[0].Name)
	require.JSONEq(t, `{"keep":true}`, string(items[0].Item.RawFields["editor_note"]))
	require.JSONEq(t, `{"position":2,"url":"https://example.invalid/image.png","alt":"fixture"}`, string(items[0].Item.MediaPlacements[0]))
	require.Equal(t, []string{"unsafe_claim"}, items[1].Assessment.RejectedFields)
	require.Contains(t, string(items[1].Receipt.RawPayload), "synthetic rejected claim")
	require.Equal(t, []string{"body"}, items[2].Assessment.UnresolvedFields)
	require.Contains(t, string(items[2].Receipt.RawPayload), `"unknown"`)
	// A transport replay returns the durable batch; conflicting reuse is rejected.
	replay, err := s.Admit(ctx, req)
	require.NoError(t, err)
	require.Equal(t, items[0].BatchID, replay[0].BatchID)
	duplicateTransport := req
	duplicateTransport.IdempotencyKey = "another-transport-key"
	duplicateTransport.Items = []intake.Item{c, b, a}
	replay, err = s.Admit(ctx, duplicateTransport)
	require.NoError(t, err)
	require.Equal(t, items[0].BatchID, replay[0].BatchID, "source revision identity prevents duplicate import under a fresh request key")
	var producerItem intake.Item
	require.NoError(t, json.Unmarshal([]byte(`{"external_id":"raw","kind":"article","title":"Raw fixture","body":"synthetic","new_producer_field":{"keep":"all"}}`), &producerItem))
	rawSource := source()
	rawSource.ArtifactID = "raw-fixture-doc"
	rawBatch, err := s.Admit(ctx, intake.Request{Source: rawSource, IdempotencyKey: "raw-fixture", Items: []intake.Item{producerItem}})
	require.NoError(t, err)
	require.Contains(t, string(rawBatch[0].Receipt.RawPayload), `"new_producer_field":{"keep":"all"}`)
	changed := req
	changed.Items = []intake.Item{good("different")}
	_, err = s.Admit(ctx, changed)
	require.ErrorContains(t, err, "reused with a different")
}

func TestRestartAndReconcileOnlyUnresolvedItemsWithoutTitleMerge(t *testing.T) {
	db, s := setup(t)
	defer db.Close()
	ctx := context.Background()
	incomplete := good("need-body")
	incomplete.Body = ""
	rejected := good("rejected")
	rejected.RejectedFields = []string{"claim"}
	first, err := s.Admit(ctx, intake.Request{Source: source(), IdempotencyKey: "restart-1", Items: []intake.Item{good("ready-a"), good("ready-b"), incomplete, rejected}})
	require.NoError(t, err)
	batch := first[0].BatchID
	// Reopen through a new service value to model process restart.
	s = intake.NewService(db)
	updated := good("need-body")
	updated.Body = "Now complete, synthetic update."
	drift := good("ready-a")
	drift.Body = "must not overwrite the admitted source"
	unknown := good("not-in-batch")
	got, err := s.Reconcile(ctx, batch, intake.Request{Source: source(), Items: []intake.Item{updated, drift, unknown}})
	require.NoError(t, err)
	require.Len(t, got, 4)
	byID := map[string]intake.StoredItem{}
	for _, x := range got {
		byID[x.Item.ExternalID] = x
	}
	require.Equal(t, "admissible", byID["need-body"].Assessment.Status)
	require.Len(t, byID["need-body"].Receipts, 2)
	require.Contains(t, string(byID["need-body"].Receipts[0].RawPayload), `"body":""`)
	require.Equal(t, "Synthetic content for intake tests.", byID["ready-a"].Item.Body)
	require.Len(t, byID["ready-a"].Receipts, 1)
	require.Equal(t, "rejected", byID["rejected"].Assessment.Status)
	require.Len(t, byID["rejected"].Receipts, 1)
	// Same title/body do not merge: the external source identity is the key.
	require.NotEqual(t, byID["ready-a"].Item.ExternalID, byID["ready-b"].Item.ExternalID)
	wrong := source()
	wrong.RevisionID = "rev-3"
	_, err = s.Reconcile(ctx, batch, intake.Request{Source: wrong, Items: []intake.Item{updated}})
	require.ErrorContains(t, err, "must match the batch")
}

func statuses(xs []intake.Assessment) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		out = append(out, x.Status)
	}
	return out
}
