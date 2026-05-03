// Package notes hosts the /api/v1/notes CRUD handlers.
//
// Layering: this package owns the transport edge (HTTP method/path
// dispatch, request decoding, error envelope writing) and translates
// between the wire types (proto-generated Note) and the domain types
// (store.Note). Domain logic — persistence, ordering, validation —
// lives in store; the handler is intentionally thin (api-steer §7).
//
// Every CRUD scenario adds copies this shape: a Deps struct, a
// NewHandler constructor returning a registered subrouter, and one
// method per route that decodes → calls store → maps errors → writes
// a proto-typed response.
package notes

import (
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes"

	"{{SCENARIO_ID}}/internal/httpx"
	"{{SCENARIO_ID}}/internal/store"
)

// defaultListLimit caps the rows returned by GET /api/v1/notes when
// the caller doesn't specify one. Cursor pagination is the canonical
// follow-up; until then this bound prevents unbounded scans.
const defaultListLimit = 100

// Deps wires the seams the notes handler needs. Logger receives the
// underlying error for every 500 path so operators can correlate the
// log line with the wire-side trace; the wire `message` stays
// human-safe.
type Deps struct {
	Store  store.NoteStore
	Logger *log.Logger
}

// NewHandler returns the /api/v1/notes subrouter. Caller mounts it via
// router.PathPrefix("/api/v1/notes").Handler(notes.NewHandler(deps)).
func NewHandler(d Deps) http.Handler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	h := &handler{deps: d}

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/notes", h.handleList).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/notes", h.handleCreate).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/notes/{id}", h.handleGet).Methods(http.MethodGet)
	return r
}

type handler struct {
	deps Deps
}

func (h *handler) handleList(w http.ResponseWriter, r *http.Request) {
	notes, err := h.deps.Store.List(r.Context(), defaultListLimit)
	if err != nil {
		h.deps.Logger.Printf("notes.List: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "list notes failed")
		return
	}

	resp := &notesv1.ListNotesResponse{
		Notes: make([]*notesv1.Note, 0, len(notes)),
	}
	for _, n := range notes {
		resp.Notes = append(resp.Notes, domainToProto(n))
	}
	writeProto(w, http.StatusOK, resp)
}

func (h *handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[notesv1.CreateNoteRequest](r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
		return
	}
	if req.Title == "" {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "title required")
		return
	}

	created, err := h.deps.Store.Create(r.Context(), store.Note{
		Title: req.Title,
		Body:  req.Body,
	})
	if err != nil {
		h.deps.Logger.Printf("notes.Create: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "create note failed")
		return
	}

	writeProto(w, http.StatusCreated, &notesv1.CreateNoteResponse{Note: domainToProto(created)})
}

func (h *handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	got, err := h.deps.Store.Get(r.Context(), id)
	if err != nil {
		var nf store.ErrNoteNotFound
		if errors.As(err, &nf) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, nf.Error())
			return
		}
		h.deps.Logger.Printf("notes.Get(%q): %v", id, err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "get note failed")
		return
	}
	writeProto(w, http.StatusOK, &notesv1.GetNoteResponse{Note: domainToProto(got)})
}

// writeProto marshals msg with protojson UseProtoNames so the wire
// shape exposes snake_case keys matching the proto declarations
// (created_at, updated_at). UI and CLI clients depend on that shape.
func writeProto(w http.ResponseWriter, status int, msg proto.Message) {
	body, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(msg)
	if err != nil {
		// Marshal of a populated message in this surface cannot fail
		// (no oneofs, no Any). If it does, log and surface a typed
		// error envelope rather than a half-written body.
		log.Printf("notes.writeProto: protojson marshal failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "response marshal failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
