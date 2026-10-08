package cliapptest

import (
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// MustMarshalProto serialises msg as the wire shape the API would emit
// (snake_case via UseProtoNames=true), or fails the test. Every CLI handler
// test that needs to feed a fake response should reach for this rather than
// hand-writing JSON literals — hand-rolled JSON drifts silently when the
// proto schema grows or renames fields, while a typed proto.Marshal call
// breaks at compile time, surfacing the schema change at the test that
// depends on it.
//
// Canonical usage in a fake API server:
//
//	body := cliapptest.MustMarshalProto(t, &notesv1.ListNotesResponse{
//	    Notes: []*notesv1.Note{{Id: "a", Title: "first"}},
//	})
//	w.Write(body)
//
// UseProtoNames=true mirrors the production handler's
// `(protojson.MarshalOptions{UseProtoNames: true}).Marshal(...)` so the wire
// shape the test feeds matches what production sends — including snake_case
// keys like `created_at` that the CLI's protojson.Unmarshal accepts on the
// read side.
func MustMarshalProto(tb testing.TB, msg proto.Message) []byte {
	tb.Helper()
	body, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(msg)
	if err != nil {
		tb.Fatalf("MustMarshalProto: %v", err)
		return nil
	}
	return body
}
