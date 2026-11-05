package testutil

import (
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// DecodeProtoJSON decodes the response body as proto JSON into the provided message.
func DecodeProtoJSON[T proto.Message](tb testing.TB, rec *httptest.ResponseRecorder, msg T) T {
	tb.Helper()
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(rec.Body.Bytes(), msg); err != nil {
		tb.Fatalf("decode proto JSON: %v", err)
	}
	return msg
}
