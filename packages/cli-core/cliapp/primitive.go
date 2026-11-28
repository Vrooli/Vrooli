package cliapp

import (
	"encoding/json"
	"fmt"
	"io"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Renderer-separated command primitives.
//
// Each builder returns a func(RunContext) error — the exact handler shape the
// manifest bindings map (LoadFromManifest) and Command.RunCtx both accept — that
// runs the operation, then routes rendering through cli-core's output-mode
// selection. Because the handler closure never sees the --json flag, scenario
// code physically cannot branch its operation on the output format: --json is an
// output contract, not an operation selector. This is the runtime guarantee
// behind the L4 rung of the command-architecture maturity ladder (see
// scenarios/cli-health/docs/reference/cli-architecture-maturity.md).
//
// The declared PrimitiveClass that classifies a command lives in the manifest's
// architecture block (or Command.Architecture for non-manifest RunCtx commands);
// these builders supply the matching, drift-proof implementation. A scenario
// reaches L4 by declaring the class AND building the handler with the matching
// primitive.
//
// Each builder returns a PrimitiveHandler, which pairs the handler closure with
// machine-readable evidence of the primitive class that built it. The evidence
// is held in an UNEXPORTED field, so only a cli-core builder in this package can
// stamp it — a scenario package cannot forge observed evidence with a struct
// literal or field assignment (plan decision D3). CLI Health compares the
// manifest-declared primitive against this observed evidence (see
// ClassifyPrimitiveEvidence). Threading the handler onto a command with its
// evidence is what LoadFromManifestPrimitives and Command.WithPrimitive do.

// PrimitiveHandler pairs a renderer-separated handler closure with the cli-core
// PrimitiveClass that built it. The zero value carries no evidence. Run is the
// exact func(RunContext) error shape Command.RunCtx and the manifest bindings
// map accept, so a PrimitiveHandler can always degrade to a plain handler via
// its Run field when evidence is not needed.
//
// The primitive-class evidence is unexported and only a builder in this package
// can set it: a scenario package can hold or pass a PrimitiveHandler but cannot
// synthesize one that claims a primitive it did not build.
type PrimitiveHandler struct {
	// primitive is the cli-core primitive class this handler was built from — the
	// observed implementation evidence CLI Health reconciles against the manifest
	// declaration. Unexported so it cannot be forged from outside cli-core.
	primitive PrimitiveClass
	// Run is the renderer-separated handler closure.
	Run func(RunContext) error
}

// Primitive returns the cli-core primitive class this handler was built from
// (empty for a zero handler). Read-only: the evidence can only be stamped by a
// cli-core primitive builder.
func (h PrimitiveHandler) Primitive() PrimitiveClass { return h.primitive }

// ProtoList builds a renderer-separated read handler: it runs call, then renders
// the proto response as proto JSON (under --json) or the human ListReport. Pair
// it with a manifest command declaring architecture.primitive "proto_list".
//
// Both call and report take an OperationContext, not a RunContext: neither can
// observe the output mode, so the operation is identical under human and --json
// output by construction. The primitive alone owns rendering.
func ProtoList[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) ListReport,
) PrimitiveHandler {
	return protoListWithJSONOptions(call, report, protoJSONOptions)
}

// ProtoListWithConnectJSONFlag adds an opt-in JSON spelling for consumers
// that share payloads with a Connect JSON endpoint. The named boolean flag
// switches only the serializer's field-name policy to protobuf's standard
// lowerCamelCase mapping; ordinary --json keeps ProtoList's established
// UseProtoNames snake_case contract. The operation and human renderer remain
// identical in either mode.
//
// Use this only when a command's API JSON contract has been checked against
// the actual Connect codec. The opt-in keeps existing CLI scripts stable while
// allowing a specific machine consumer to use the API's field names.
func ProtoListWithConnectJSONFlag[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) ListReport,
	flagName string,
) PrimitiveHandler {
	connectJSONOptions := protojson.MarshalOptions{Multiline: true, Indent: "  "}
	return PrimitiveHandler{primitive: PrimitiveProtoList, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		options := protoJSONOptions
		if ctx.BoolFlag(flagName) {
			options = connectJSONOptions
		}
		return RenderProtoListWithJSONOptions(ctx, resp, report(ctx, resp), options)
	}}
}

// ProtoListEmitUnpopulatedJSON is ProtoList with explicit zero-valued proto
// fields in its machine-readable output. It is intended for diagnostic
// responses where the presence of a zero is meaningful evidence (for example,
// a clean semantic census), while preserving the normal human report path.
func ProtoListEmitUnpopulatedJSON[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) ListReport,
) PrimitiveHandler {
	options := protojson.MarshalOptions{
		UseProtoNames:   true,
		Multiline:       true,
		Indent:          "  ",
		EmitUnpopulated: true,
	}
	return protoListWithJSONOptions(call, report, options)
}

func protoListWithJSONOptions[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) ListReport,
	options protojson.MarshalOptions,
) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveProtoList, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		return RenderProtoListWithJSONOptions(ctx, resp, report(ctx, resp), options)
	}}
}

// ProtoListOutcome is ProtoList plus an outcome mapper for commands whose exit
// code is derived from the response payload (e.g. `validate scenario` renders the
// findings AND exits non-zero when the assessment failed). After rendering the
// response — in either output mode — outcome(resp) is returned as the handler's
// error, so the exit code is identical for human and --json and the operation
// still never observes the output mode. outcome may return nil for success.
//
// The primitive class stays proto_list: the render shape is unchanged; the
// outcome is exit-code policy, not a different lifecycle. So a command using it
// declares architecture.primitive "proto_list" and verifies exactly like ProtoList.
func ProtoListOutcome[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) ListReport,
	outcome func(resp Resp) error,
) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveProtoList, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		return renderThenOutcome(func() error {
			return RenderProtoList(ctx, resp, report(ctx, resp))
		}, func() error {
			if outcome == nil {
				return nil
			}
			return outcome(resp)
		})
	}}
}

// ProtoMutation builds a renderer-separated write handler: proto JSON under
// --json, else the human MutationReport. Declare architecture.primitive
// "proto_mutation". call/report receive an OperationContext and cannot branch on
// output mode.
func ProtoMutation[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) MutationReport,
) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveProtoMutation, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		return RenderProtoMutation(ctx, resp, report(ctx, resp))
	}}
}

// ProtoMutationWithJSONRenderer is ProtoMutation with a domain-specific JSON
// renderer for responses whose protobuf payload contains data already written
// to local output files. The operation remains output-mode agnostic and the
// primitive evidence remains proto_mutation; only post-operation rendering is
// specialized. A nil json renderer preserves the canonical proto JSON output.
func ProtoMutationWithJSONRenderer[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) MutationReport,
	jsonRenderer func(ctx RunContext, resp Resp) error,
) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveProtoMutation, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		if ctx.JSON() && jsonRenderer != nil {
			return jsonRenderer(ctx, resp)
		}
		return RenderProtoMutation(ctx, resp, report(ctx, resp))
	}}
}

// ProtoMutationOutcome is ProtoMutation plus an outcome mapper for write
// commands whose exit code is derived from the response payload after the
// response has been rendered. The primitive class stays proto_mutation: this is
// exit-code policy, not a different render shape.
func ProtoMutationOutcome[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) MutationReport,
	outcome func(ctx OperationContext, resp Resp) error,
) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveProtoMutation, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		return renderThenOutcome(func() error {
			return RenderProtoMutation(ctx, resp, report(ctx, resp))
		}, func() error {
			if outcome == nil {
				return nil
			}
			return outcome(ctx, resp)
		})
	}}
}

// ProtoOperational builds a renderer-separated diagnostic handler: proto JSON
// under --json, else the human OperationalReport. Declare architecture.primitive
// "operational". call/report receive an OperationContext and cannot branch on
// output mode.
func ProtoOperational[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) OperationalReport,
) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveOperational, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		return RenderProtoOperational(ctx, resp, report(ctx, resp))
	}}
}

// ProtoOperationalWithExit is the diagnostic counterpart for status contracts
// whose authoritative response also gates the process result. It renders the
// complete typed response first, then returns the domain-owned exit decision;
// therefore --json remains parseable even when the status is not ready.
func ProtoOperationalWithExit[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) OperationalReport,
	exit func(ctx OperationContext, resp Resp) error,
) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveOperational, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		if err := RenderProtoOperational(ctx, resp, report(ctx, resp)); err != nil {
			return err
		}
		if exit == nil {
			return nil
		}
		return exit(ctx, resp)
	}}
}

// Action builds a renderer-separated single-call handler for non-proto commands
// that still have one operation result and a mutation-shaped human report. Use
// it for REST-backed or local actions that cannot use a Connect-RPC primitive
// but do not have a special lifecycle.
func Action[Resp any](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) MutationReport,
) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveAction, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		return RenderAction(ctx, resp, report(ctx, resp))
	}}
}

// ActionCustomRenderers is Action with separate human and JSON renderers for
// legacy or domain-specific action output. The operation callback still runs
// exactly once and cannot inspect output mode.
func ActionCustomRenderers[Resp any](
	call func(ctx OperationContext) (Resp, error),
	human func(ctx RunContext, resp Resp) error,
	json func(ctx RunContext, resp Resp) error,
) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveAction, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		render := human
		if ctx.JSON() {
			render = json
		}
		if render == nil {
			return fmt.Errorf("custom action renderer is required")
		}
		return render(ctx, resp)
	}}
}

// ProtoCustom builds a renderer-separated single-call handler for proto RPCs
// whose human presentation is domain-specific. JSON remains the canonical
// proto JSON representation, while human receives the typed response and owns
// only its presentation. The operation callback receives OperationContext, so
// it cannot branch on --json. Declare architecture.primitive "action".
//
// This is intended for compact capability/status views that do not fit the
// ListReport, MutationReport, or OperationalReport shapes. Commands with a
// standard report should use the corresponding Proto* primitive instead.
func ProtoCustom[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	human func(ctx RunContext, resp Resp) error,
) PrimitiveHandler {
	return ProtoCustomRenderers(call, human, nil)
}

// ProtoCustomRenderers is ProtoCustom with an optional JSON renderer. When
// json is nil, canonical proto JSON is emitted. Supplying json supports legacy
// commands whose machine-mode output is a domain-specific stream or text shape.
// Both renderers run only after the same output-agnostic operation callback.
func ProtoCustomRenderers[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	human func(ctx RunContext, resp Resp) error,
	json func(ctx RunContext, resp Resp) error,
) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveAction, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		if ctx.JSON() {
			if json != nil {
				return json(ctx, resp)
			}
			return PrintProtoJSON(ctx.Stdout(), resp)
		}
		if human == nil {
			return fmt.Errorf("custom proto human renderer is required")
		}
		return human(ctx, resp)
	}}
}

// Streaming builds a command handler for an intentionally long-lived stream
// whose event lifecycle cannot be represented by a single request/response
// primitive. It stamps streaming exception evidence without claiming renderer
// separation. Declare architecture.exception.class "streaming".
func Streaming(run func(RunContext) error) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveStreaming, Run: run}
}

// ContentAction is an action whose result is the content itself (artifact
// bytes, a log) written straight to RunContext.Stdout so a caller can pipe it.
// It carries the same action evidence as Action; there is no report to render.
func ContentAction(run func(ctx RunContext) error) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveAction, Run: run}
}

// ActionWithExit is Action with a post-render exit decision. It is useful for
// diagnostic scans that must emit their complete report while still returning
// a non-zero status when the report contains a gating finding.
func ActionWithExit[Resp any](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) MutationReport,
	exit func(ctx OperationContext, resp Resp) error,
) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveAction, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		if err := RenderAction(ctx, resp, report(ctx, resp)); err != nil {
			return err
		}
		if exit == nil {
			return nil
		}
		return exit(ctx, resp)
	}}
}

// ExternalDelegation stamps evidence for a command that orchestrates a local
// preparation step and delegates the resulting side effects to another tool or
// scenario. The callback receives RunContext so existing list/report rendering
// remains owned by cli-core while the delegated operation stays on one path.
// Pair it with a manifest external_delegation exception.
func ExternalDelegation(run func(ctx RunContext) error) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveExternalDelegation, Run: run}
}

// Upload builds a renderer-separated multipart upload handler. The operation
// callback owns request construction and upload/decode; cli-core owns the final
// proto render, so human and --json share one upload path. Pair it with a command
// declaring the upload exception.
func Upload[Resp proto.Message](
	call func(ctx OperationContext) (Resp, error),
	report func(ctx OperationContext, resp Resp) MutationReport,
) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitiveUpload, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		return RenderProtoMutation(ctx, resp, report(ctx, resp))
	}}
}

// RenderAction routes a generic action payload through cli-core's render
// contract. JSON receives the operation payload; human receives the mapped
// MutationReport.
func RenderAction(ctx RunContext, payload any, human MutationReport) error {
	if ctx.JSON() {
		return PrintJSON(ctx.Stdout(), payload)
	}
	return ctx.RenderMutation(human)
}

// PrintJSON emits stable indented JSON for non-proto payloads.
func PrintJSON(w io.Writer, payload any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return fmt.Errorf("marshal action json: %w", err)
	}
	return nil
}

func renderThenOutcome(render func() error, outcome func() error) error {
	if err := render(); err != nil {
		return err
	}
	return outcome()
}
