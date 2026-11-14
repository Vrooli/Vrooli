package cliapp

import (
	"fmt"
	"io"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// protoJSONOptions matches Connect-Go's wire format: snake_case field names so
// machine consumers can round-trip through protojson.Unmarshal / fromJson, plus
// indented output for readability when piped through tools like jq.
var protoJSONOptions = protojson.MarshalOptions{
	UseProtoNames: true,
	Multiline:     true,
	Indent:        "  ",
}

// PrintProtoJSON marshals msg as canonical proto JSON to w. Field names match
// the wire format every Connect-RPC client speaks, so the same `jq` queries
// work against `cli foo list --json` and a direct `curl` of the API.
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
// JSON consumers receive the proto wire shape — no `summary` / `retrieval_hints`
// wrapper. Those are human affordances; machine consumers parse the typed
// payload directly.
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
