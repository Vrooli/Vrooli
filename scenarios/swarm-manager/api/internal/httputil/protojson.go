// Package httputil provides helpers for proto JSON encoding/decoding.
//
// DOC: docs/internal/ERROR-SEMANTICS.md
// DOC: docs/internal/SEAMS.md
package httputil

import (
	"errors"
	"io"
	"net/http"
	"swarm-manager/internal/apierr"
	"sync"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	protoValidator     protovalidate.Validator
	protoValidatorOnce sync.Once
	protoValidatorErr  error
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

// DecodeProtoJSONStrict reads JSON from the request body into a proto message
// and rejects unknown fields.
func DecodeProtoJSONStrict(r *http.Request, msg proto.Message) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return protojson.UnmarshalOptions{DiscardUnknown: false}.Unmarshal(body, msg)
}

// ValidateProto enforces protovalidate constraints on a proto message.
func ValidateProto(msg proto.Message) error {
	protoValidatorOnce.Do(func() {
		protoValidator, protoValidatorErr = protovalidate.New()
	})
	if protoValidatorErr != nil {
		return protoValidatorErr
	}
	return protoValidator.Validate(msg)
}

// IsValidationError reports whether err is a protovalidate validation error.
func IsValidationError(err error) bool {
	var validationErr *protovalidate.ValidationError
	return errors.As(err, &validationErr)
}

// ValidateProtoRequest validates a proto request payload and writes an error response when invalid.
func ValidateProtoRequest(w http.ResponseWriter, logPrefix, badRequestMessage string, msg proto.Message) bool {
	if err := ValidateProto(msg); err != nil {
		if IsValidationError(err) {
			apierr.MapError(w, logPrefix, apierr.BadRequest("%s", badRequestMessage))
		} else {
			apierr.MapError(w, logPrefix, apierr.Internal("failed to validate request"))
		}
		return false
	}
	return true
}
