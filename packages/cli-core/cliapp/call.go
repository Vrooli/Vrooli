package cliapp

import (
	"fmt"
	"net/url"
	"reflect"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Call performs a typed JSON-over-HTTP request against the scenario API.
//
// Req and Resp are pointer-to-proto-message types (e.g. *notesv1.CreateNoteRequest).
// Marshal/unmarshal use protojson, matching the wire format every scenario
// API uses. On non-2xx responses, the returned error is the typed
// *cliutil.APIError with the raw body — callers wrap it via WrapAPIError to
// surface the typed envelope's code+message.
//
// Pass req as nil for GET-style calls.
//
// This is the helper that replaced the per-handler protojson.Marshal +
// app.Request + protojson.Unmarshal ribbon. It collapses ~10 lines of
// boilerplate per method into a single call.
func Call[Req proto.Message, Resp proto.Message](
	app *ScenarioApp,
	method, path string,
	req Req,
) (Resp, error) {
	var zero Resp

	var body []byte
	if !isNilProto(req) {
		encoded, err := protojson.Marshal(req)
		if err != nil {
			return zero, fmt.Errorf("marshal %T: %w", req, err)
		}
		body = encoded
	}

	respBody, err := app.Request(method, path, nil, body)
	if err != nil {
		return zero, err
	}

	resp := newProto[Resp]()
	if err := protojson.Unmarshal(respBody, resp); err != nil {
		return zero, fmt.Errorf("decode %T: %w", resp, err)
	}
	return resp, nil
}

// CallQuery is the same as Call but for GET requests with query parameters.
func CallQuery[Resp proto.Message](
	app *ScenarioApp,
	path string,
	query url.Values,
) (Resp, error) {
	var zero Resp
	respBody, err := app.Get(path, query)
	if err != nil {
		return zero, err
	}
	resp := newProto[Resp]()
	if err := protojson.Unmarshal(respBody, resp); err != nil {
		return zero, fmt.Errorf("decode %T: %w", resp, err)
	}
	return resp, nil
}

// newProto allocates a fresh instance of the pointer-to-proto-message type T.
// Resp's underlying type is always a pointer (because it satisfies
// proto.Message via its pointer receiver), so this allocates the pointee.
func newProto[T proto.Message]() T {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil || t.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("Call: type %T is not a pointer to proto.Message", zero))
	}
	return reflect.New(t.Elem()).Interface().(T)
}

// isNilProto reports whether a proto.Message-typed value is the typed nil.
// A typed nil ((*notesv1.CreateNoteRequest)(nil)) compares != nil under the
// interface comparison, so we have to check via reflection.
func isNilProto(m proto.Message) bool {
	if m == nil {
		return true
	}
	v := reflect.ValueOf(m)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return true
	}
	return false
}
