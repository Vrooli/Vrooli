// Package httputil provides helpers for proto JSON encoding/decoding.
package httputil

import (
	"io"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ProtoJSON writes a proto message as JSON using proto field names (snake_case).
func ProtoJSON(w http.ResponseWriter, msg proto.Message) error {
	return ProtoJSONWithStatus(w, http.StatusOK, msg)
}

// ProtoJSONWithStatus writes a proto message as JSON with a specific status code.
func ProtoJSONWithStatus(w http.ResponseWriter, status int, msg proto.Message) error {
	payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, writeErr := w.Write(payload)
	return writeErr
}

// DecodeProtoJSON reads JSON from the request body into a proto message.
func DecodeProtoJSON(r *http.Request, msg proto.Message) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(body, msg)
}
