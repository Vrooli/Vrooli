package httputil

import (
	"errors"
	"io"
	"net/http"
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

func ProtoJSON(w http.ResponseWriter, msg proto.Message) error {
	return ProtoJSONWithStatus(w, http.StatusOK, msg)
}

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

func DecodeProtoJSON(r *http.Request, msg proto.Message) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(body, msg)
}

func ValidateProto(msg proto.Message) error {
	protoValidatorOnce.Do(func() {
		protoValidator, protoValidatorErr = protovalidate.New()
	})
	if protoValidatorErr != nil {
		return protoValidatorErr
	}
	return protoValidator.Validate(msg)
}

func IsValidationError(err error) bool {
	var validationErr *protovalidate.ValidationError
	return errors.As(err, &validationErr)
}

func ValidateProtoRequest(w http.ResponseWriter, logPrefix, badRequestMessage string, msg proto.Message) bool {
	if err := ValidateProto(msg); err != nil {
		if IsValidationError(err) {
			BadRequest(w, logPrefix, badRequestMessage)
		} else {
			InternalError(w, logPrefix, "failed to validate request")
		}
		return false
	}
	return true
}
