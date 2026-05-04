package httpx

import (
	"fmt"
	"io"
	"net/http"
	"reflect"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// DecodeProtoJSON parses a structured JSON request body into a proto
// message. The parser is strict by default: unknown fields are rejected
// so request-shape drift is caught at the API boundary.
//
// Returns the zero value plus a wrapped error on parse failure;
// handlers translate that error into a 400 ErrorEnvelope via
// WriteError(w, 400, CodeInvalidRequest, err.Error()).
//
// Binary/file-upload endpoints are the exception: keep file bytes in
// multipart parts or upload streams, and decode only the structured
// metadata part through protojson.
func DecodeProtoJSON[T proto.Message](r *http.Request) (T, error) {
	var zero T
	if r.Body == nil {
		return zero, fmt.Errorf("decode proto JSON: empty request body")
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return zero, fmt.Errorf("decode proto JSON: read body: %w", err)
	}
	if len(body) == 0 {
		return zero, fmt.Errorf("decode proto JSON: empty request body")
	}

	msg := newProto[T]()
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, msg); err != nil {
		return zero, fmt.Errorf("decode proto JSON: %w", err)
	}
	return msg, nil
}

func newProto[T proto.Message]() T {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil || t.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("DecodeProtoJSON: type %T is not a pointer to proto.Message", zero))
	}
	return reflect.New(t.Elem()).Interface().(T)
}
