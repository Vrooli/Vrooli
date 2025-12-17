package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/ports"
	"knowledge-observatory/internal/services/ingest"
)

type fakeVectorStore struct {
	deletedID        string
	deletedCollection string
	deleteErr        error
}

func (f *fakeVectorStore) EnsureCollection(ctx context.Context, collection string, vectorSize int) error {
	return nil
}
func (f *fakeVectorStore) UpsertPoint(ctx context.Context, collection string, id string, vector []float64, payload map[string]interface{}) error {
	return nil
}
func (f *fakeVectorStore) DeletePoint(ctx context.Context, collection string, id string) error {
	f.deletedCollection = collection
	f.deletedID = id
	return f.deleteErr
}
func (f *fakeVectorStore) Search(ctx context.Context, collection string, vector []float64, limit int, threshold float64) ([]ports.VectorSearchResult, error) {
	return nil, nil
}
func (f *fakeVectorStore) ListCollections(ctx context.Context) ([]string, error) { return []string{}, nil }
func (f *fakeVectorStore) CountPoints(ctx context.Context, collection string) (int, error) {
	return 0, nil
}

type fakeMetadataStore struct {
	collection string
	ok         bool
	err        error
}

func (f *fakeMetadataStore) UpsertKnowledgeMetadata(ctx context.Context, vectorID, collectionName, contentHash, sourceScenario, sourceType string) error {
	return nil
}
func (f *fakeMetadataStore) InsertIngestHistory(ctx context.Context, row ports.IngestHistoryRow) error {
	return nil
}
func (f *fakeMetadataStore) LookupCollectionForVectorID(ctx context.Context, vectorID string) (string, bool, error) {
	return f.collection, f.ok, f.err
}

func TestHandleDeleteRecord(t *testing.T) {
	vs := &fakeVectorStore{}
	meta := &fakeMetadataStore{collection: "knowledge_chunks_v1", ok: true}
	svc := &ingest.Service{
		VectorStore: vs,
		Embedder:    nil,
		Metadata:    meta,
	}

	srv := &Server{
		ingestService: svc,
	}

	req := httptest.NewRequest("DELETE", "/api/v1/knowledge/records/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"record_id": "abc"})
	rec := httptest.NewRecorder()

	srv.handleDeleteRecord(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if vs.deletedID != "abc" || vs.deletedCollection != "knowledge_chunks_v1" {
		t.Fatalf("delete called with %s/%s", vs.deletedCollection, vs.deletedID)
	}
	var decoded DeleteRecordResponse
	if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !decoded.Deleted {
		t.Fatalf("expected deleted=true")
	}
}

func TestHandleDeleteRecordNotFound(t *testing.T) {
	vs := &fakeVectorStore{}
	meta := &fakeMetadataStore{collection: "", ok: false}
	svc := &ingest.Service{
		VectorStore: vs,
		Metadata:    meta,
	}
	srv := &Server{ingestService: svc}

	req := httptest.NewRequest("DELETE", "/api/v1/knowledge/records/missing", nil)
	req = mux.SetURLVars(req, map[string]string{"record_id": "missing"})
	rec := httptest.NewRecorder()

	srv.handleDeleteRecord(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteRecordVectorStoreError(t *testing.T) {
	vs := &fakeVectorStore{deleteErr: errors.New("boom")}
	meta := &fakeMetadataStore{collection: "knowledge_chunks_v1", ok: true}
	svc := &ingest.Service{
		VectorStore: vs,
		Metadata:    meta,
	}
	srv := &Server{ingestService: svc}

	req := httptest.NewRequest("DELETE", "/api/v1/knowledge/records/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"record_id": "abc"})
	rec := httptest.NewRecorder()

	srv.handleDeleteRecord(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

