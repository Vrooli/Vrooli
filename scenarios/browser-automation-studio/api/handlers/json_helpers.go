package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/internal/httpjson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	return httpjson.Decode(w, r, dst)
}

func decodeJSONBodyAllowEmpty(w http.ResponseWriter, r *http.Request, dst any) error {
	return httpjson.DecodeAllowEmpty(w, r, dst)
}

func decodeProtoJSONBody(r *http.Request, msg proto.Message) error {
	if r == nil {
		return fmt.Errorf("request is nil")
	}
	if msg == nil {
		return fmt.Errorf("msg is nil")
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if len(body) == 0 {
		body = []byte("{}")
	}
	return (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, msg)
}

// parseUUIDParam extracts a UUID from the URL parameter with the given name.
// If parsing fails, it writes the provided error to the response and returns false.
// On success, it returns the parsed UUID and true.
func (h *Handler) parseUUIDParam(w http.ResponseWriter, r *http.Request, paramName string, errResponse *APIError) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, paramName)
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, errResponse)
		return uuid.UUID{}, false
	}
	return id, true
}
