// Package protoout prints generated messages as proto JSON. Machine output is
// the server's typed message unchanged (proto field names, lossless through
// protojson), never a hand-built map.
package protoout

import (
	"fmt"
	"io"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var marshal = protojson.MarshalOptions{UseProtoNames: true, Indent: "  "}

// Marshal renders msg as indented proto JSON.
func Marshal(msg proto.Message) ([]byte, error) {
	return marshal.Marshal(msg)
}

// Print writes msg as proto JSON to stdout.
func Print(msg proto.Message) error {
	return Write(os.Stdout, msg)
}

// Write writes msg as proto JSON to w.
func Write(w io.Writer, msg proto.Message) error {
	raw, err := marshal.Marshal(msg)
	if err != nil {
		return fmt.Errorf("encode %s: %w", msg.ProtoReflect().Descriptor().FullName(), err)
	}
	_, err = fmt.Fprintln(w, string(raw))
	return err
}

// Unmarshal parses proto JSON into msg; tests use it to prove output is
// lossless.
func Unmarshal(raw []byte, msg proto.Message) error {
	return protojson.Unmarshal(raw, msg)
}
