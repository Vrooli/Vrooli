package cliapp

import (
	"encoding/json"
	"fmt"
	"io"

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
	return PrimitiveHandler{primitive: PrimitiveProtoList, Run: func(ctx RunContext) error {
		resp, err := call(ctx)
		if err != nil {
			return err
		}
		return RenderProtoList(ctx, resp, report(ctx, resp))
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
