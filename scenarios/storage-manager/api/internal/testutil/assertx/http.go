// Package assertx holds domain-aware test assertions.
//
// Helpers here cover the canonical handler test; scenarios add focused
// helpers (AssertHealthDependency, AssertTaskStatus, ...) as their
// domain grows. Resist generalising into a god-helper grab bag — small
// focused helpers stay readable; one mega-`AssertResponse` does not.
//
// MustDecodeJSON is the generic JSON-into-struct helper for ad-hoc
// payloads; MustUnmarshalProto is the proto-typed equivalent for the
// canonical wire contract (every endpoint backed by a proto schema in
// packages/proto/schemas/<id>/).
package assertx

import (
	"encoding/json"
	"net/http"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// failer is the minimum surface AssertStatus / MustDecodeJSON need from
// *testing.T. It exists so the helpers' fail paths can be exercised in
// this package's own tests with a recording stub — *testing.T can't be
// faked directly because testing.TB carries an unexported method.
type failer interface {
	Helper()
	Fatalf(format string, args ...any)
}

// AssertStatus fails the test if resp.StatusCode != want. On mismatch,
// it reports the actual code so the failure message is self-contained.
// Body inspection belongs in a separate assertion (e.g., MustDecodeJSON
// + field checks) — keeping AssertStatus focused makes its intent
// obvious at the callsite.
func AssertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	assertStatus(t, resp, want)
}

func assertStatus(t failer, resp *http.Response, want int) {
	t.Helper()
	if resp == nil {
		t.Fatalf("AssertStatus: response is nil (want %d)", want)
		return
	}
	if resp.StatusCode != want {
		t.Fatalf("AssertStatus: got %d, want %d", resp.StatusCode, want)
	}
}

// MustDecodeJSON decodes body into a value of type T or fails the test.
// Generic so callers get a typed result without an extra type
// assertion: `got := MustDecodeJSON[map[string]any](t, body)`.
//
// Prefer MustUnmarshalProto when the wire contract is defined by a
// proto schema — round-tripping through protojson respects field
// presence semantics (oneofs, optional scalars) and unknown-field
// tolerance that hand-written JSON structs lose.
func MustDecodeJSON[T any](t *testing.T, body []byte) T {
	t.Helper()
	return mustDecodeJSON[T](t, body)
}

func mustDecodeJSON[T any](t failer, body []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("MustDecodeJSON: %v\nbody: %s", err, string(body))
	}
	return v
}

// MustUnmarshalProto decodes body into a proto message of type *T or
// fails the test. The two-parameter generic constraint (T plus the
// pointer type PT) lets callers write the natural form:
//
//	got := assertx.MustUnmarshalProto[healthv1.Response](t, body)
//	// got is *healthv1.Response
//
// Without the PT trick, callers would have to write the awkward
// `MustUnmarshalProto[*healthv1.Response, healthv1.Response]` form to
// satisfy proto.Message (which only the pointer type implements).
//
// `DiscardUnknown: true` mirrors the interop-steer guidance: handler
// tests should still succeed when the wire grows fields the proto
// hasn't caught up to, so the test fails on real shape mismatches
// rather than schema-version drift.
func MustUnmarshalProto[T any, PT interface {
	*T
	proto.Message
}](t *testing.T, body []byte) PT {
	t.Helper()
	return mustUnmarshalProto[T, PT](t, body)
}

func mustUnmarshalProto[T any, PT interface {
	*T
	proto.Message
}](t failer, body []byte) PT {
	t.Helper()
	var v T
	msg := PT(&v)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, msg); err != nil {
		t.Fatalf("MustUnmarshalProto: %v\nbody: %s", err, string(body))
	}
	return msg
}
