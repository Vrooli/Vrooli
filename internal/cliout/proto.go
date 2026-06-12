package cliout

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// WriteProtoJSON writes a proto message as the canonical CLI JSON: snake_case
// field names, every field present (EmitUnpopulated), and standard
// encoding/json formatting (2-space indent, single space after the colon).
//
// protojson's own marshaler emits non-standard, intentionally non-deterministic
// whitespace (e.g. two spaces after the colon). We marshal compact and re-indent
// through encoding/json so the wire shape matches the rest of the CLI's output
// byte-for-byte in formatting — every vrooli.cli.v1 contract looks identical to
// the hand-rolled JSON it replaced, only typed.
func WriteProtoJSON(w io.Writer, msg proto.Message) error {
	return writeProtoJSON(w, msg, true)
}

// WriteProtoJSONCamel is WriteProtoJSON with camelCase (proto json_name) field
// names. Use it only for the handful of contracts whose pre-proto wire format
// was camelCase, so the migration stays byte-faithful.
func WriteProtoJSONCamel(w io.Writer, msg proto.Message) error {
	return writeProtoJSON(w, msg, false)
}

func writeProtoJSON(w io.Writer, msg proto.Message, useProtoNames bool) error {
	if w == nil {
		return errors.New("writer is required")
	}
	compact, err := protojson.MarshalOptions{
		UseProtoNames:   useProtoNames,
		EmitUnpopulated: true,
	}.Marshal(msg)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, compact, "", "  "); err != nil {
		return err
	}
	buf.WriteByte('\n')
	_, err = w.Write(buf.Bytes())
	return err
}
