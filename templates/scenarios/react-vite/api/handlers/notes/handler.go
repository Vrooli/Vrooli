// Package notes hosts the /api/v1/notes CRUD handlers.
//
// Layering: this package owns the transport edge (HTTP method/path
// dispatch, request decoding, error envelope writing) and translates
// between the wire types (proto-generated Note) and the domain types
// (notes.Note). Validation, defaults, and any cross-handler policy live
// in notes.Service; persistence lives in notes.Repository. The handler
// is intentionally thin: decode → call service → translate errors →
// write response (api-steer §7).
//
// Every CRUD scenario adds copies this shape: a Deps struct, a
// NewHandler constructor returning a registered subrouter, and one
// method per route that decodes → calls service → maps errors → writes
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
	"{{SCENARIO_ID}}/internal/notes"
)

// Deps wires the seams the notes handler needs. Logger receives the
// underlying error for every 500 path so operators can correlate the
// log line with the wire-side trace; the wire `message` stays
// human-safe.
type Deps struct {
	Service notes.Service
	Logger  *log.Logger
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
	// Pass 0 so the service substitutes its default. The handler is
	// transport-only — limit policy lives one floor down.
	results, err := h.deps.Service.List(r.Context(), 0)
	if err != nil {
		h.deps.Logger.Printf("notes.List: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "list notes failed")
		return
	}

	resp := &notesv1.ListNotesResponse{
		Notes: make([]*notesv1.Note, 0, len(results)),
	}
	for _, n := range results {
		resp.Notes = append(resp.Notes, domainToProto(n))
	}
	h.writeProto(w, http.StatusOK, resp)
}

func (h *handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	// Malformed-JSON / unknown-field rejection stays at the transport
	// edge: the bytes never become a domain object, so routing the
	// failure through the service would force the service to define an
	// error type for "you sent garbage." Cleaner to keep the boundary
	// tight.
	req, err := httpx.DecodeJSON[notesv1.CreateNoteRequest](r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
		return
	}

	created, err := h.deps.Service.Create(r.Context(), notes.CreateInput{
		Title: req.Title,
		Body:  req.Body,
	})
	if err != nil {
		var inv notes.ErrInvalidNote
		if errors.As(err, &inv) {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, inv.Error())
			return
		}
		h.deps.Logger.Printf("notes.Create: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "create note failed")
		return
	}

	h.writeProto(w, http.StatusCreated, &notesv1.CreateNoteResponse{Note: domainToProto(created)})
}

func (h *handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	got, err := h.deps.Service.Get(r.Context(), id)
	if err != nil {
		var nf notes.ErrNoteNotFound
		if errors.As(err, &nf) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, nf.Error())
			return
		}
		h.deps.Logger.Printf("notes.Get(%q): %v", id, err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "get note failed")
		return
	}
	h.writeProto(w, http.StatusOK, &notesv1.GetNoteResponse{Note: domainToProto(got)})
}

// writeProto marshals msg with protojson UseProtoNames so the wire
// shape exposes snake_case keys matching the proto declarations
// (created_at, updated_at). UI and CLI clients depend on that shape.
//
// Method (not free function) so the marshal-failure fallback uses
// h.deps.Logger — the same buffer-backed logger handler tests inject —
// instead of leaking through the global log package. The marshal call
// itself is unreachable in practice (no oneofs, no Any, no recursion),
// but keeping the seam consistent matters: scenarios copy this shape
// into new handlers and inherit logger discipline for free.
func (h *handler) writeProto(w http.ResponseWriter, status int, msg proto.Message) {
	body, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(msg)
	if err != nil {
		h.deps.Logger.Printf("notes.writeProto: protojson marshal failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "response marshal failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
