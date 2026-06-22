package httputil

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
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
	// Delegate the actual write to the shared WriteRaw helper so the secure
	// header floor is applied in one place and this file performs no direct
	// ResponseWriter writes.
	return WriteRaw(w, status, "application/json", payload)
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

func ValidateProtoRequest(w http.ResponseWriter, log *slog.Logger, r *http.Request, badRequestMessage string, msg proto.Message) bool {
	if err := ValidateProto(msg); err != nil {
		if IsValidationError(err) {
			WriteAPIError(w, log, r, apierrors.Validation("body", badRequestMessage))
		} else {
			WriteAPIError(w, log, r, apierrors.Internal("failed to validate request", err))
		}
		return false
	}
	return true
}
