package httpx

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func DecodeJSON[T any](tb testing.TB, rec *httptest.ResponseRecorder) T {
	tb.Helper()
	var result T
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		tb.Fatalf("decode response JSON: %v", err)
	}
	return result
}

func DecodeProtoJSON[T proto.Message](tb testing.TB, rec *httptest.ResponseRecorder, msg T) T {
	tb.Helper()
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(rec.Body.Bytes(), msg); err != nil {
		tb.Fatalf("decode proto JSON: %v", err)
	}
	return msg
}
