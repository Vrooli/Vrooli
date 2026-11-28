package cliapp

import (
	"fmt"
	"io"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// protoJSONOptions is the established CLI --json contract: protobuf field
// names (snake_case) and indented output. Connect JSON uses the standard
// protobuf lowerCamelCase names instead; ProtoListWithConnectJSONFlag exposes
// that spelling as an explicit, additive consumer option.
var protoJSONOptions = protojson.MarshalOptions{
	UseProtoNames: true,
	Multiline:     true,
	Indent:        "  ",
}

// PrintProtoJSON marshals msg using the established CLI proto-name contract.
// This is schema-shaped protobuf JSON, not the canonical lowerCamelCase JSON
// spelling used by Connect's HTTP codec.
func PrintProtoJSON(w io.Writer, msg proto.Message) error {
	return printProtoJSONWithOptions(w, msg, protoJSONOptions)
}

func printProtoJSONWithOptions(w io.Writer, msg proto.Message, options protojson.MarshalOptions) error {
	body, err := options.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal proto json: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	if len(body) == 0 || body[len(body)-1] != '\n' {
		_, err = w.Write([]byte{'\n'})
	}
	return err
}

// RenderProtoList renders payload as proto JSON (when ctx.JSON() is set) or
// the human ListReport otherwise. Use this when the report's Results lines are
// derived from proto fields the JSON consumer also needs; it eliminates the
// per-domain `<thing>JSON` ribbon that would otherwise re-derive the proto
// shape with hand-typed structs.
//
// JSON consumers receive the protobuf message shape — no `summary` /
// `retrieval_hints` wrapper. Those are human affordances; machine consumers
// parse the typed payload directly. ProtoList uses the legacy snake_case
// spelling; opt in to lowerCamelCase when matching a Connect JSON endpoint.
func RenderProtoList(ctx RunContext, payload proto.Message, human ListReport) error {
	return RenderProtoListWithJSONOptions(ctx, payload, human, protoJSONOptions)
}

// RenderProtoListWithJSONOptions is the diagnostic variant of RenderProtoList
// that lets a command opt into a precise protojson policy without allowing
// operation code to branch on --json.
func RenderProtoListWithJSONOptions(ctx RunContext, payload proto.Message, human ListReport, options protojson.MarshalOptions) error {
	if ctx.JSON() {
		return printProtoJSONWithOptions(ctx.Stdout(), payload, options)
	}
	return ctx.RenderList(human)
}

// RenderProtoMutation is the mutation-shaped counterpart to RenderProtoList.
// Same trade-off: JSON output is the proto wire shape; the report's Result /
// NextCommand lines are human-only.
func RenderProtoMutation(ctx RunContext, payload proto.Message, human MutationReport) error {
	if ctx.JSON() {
		return PrintProtoJSON(ctx.Stdout(), payload)
	}
	return ctx.RenderMutation(human)
}

// RenderProtoOperational is the operational-shaped counterpart to
// RenderProtoList. Use it for diagnostic/status commands: JSON output is the
// proto wire shape; the report's Status / Triage / Next Steps lines are
// human-only.
func RenderProtoOperational(ctx RunContext, payload proto.Message, human OperationalReport) error {
	if ctx.JSON() {
		return PrintProtoJSON(ctx.Stdout(), payload)
	}
	return ctx.RenderOperational(human)
}
