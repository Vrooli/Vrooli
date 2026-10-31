package cliapp

// The durable_run primitive.
//
// A durable_run command drives a server-owned run whose lifecycle is: start ->
// follow/wait/reattach -> render. The failure mode this primitive eliminates is a
// command taking a DIFFERENT execution path under --json than under human output
// (e.g. --json blocking synchronously while human streams a durable run). The fix
// is structural: the run's START is mode-blind, and the output mode selects ONLY
// the renderer. Human, --json, and --jsonl therefore share one run-ownership path
// by construction.
//
// cli-core owns the skeleton (RunDurable: start once, then dispatch to the mode's
// renderer); the scenario supplies the start closure and the three renderers,
// which stay scenario-specific (proto follow streams, human printers, JSONL event
// vocabularies). This is the L4 shape the durable_run exception certifies.

// DurableRunMode selects ONLY the renderer for a durable_run command. It never
// changes whether or how the run starts.
type DurableRunMode int

const (
	// DurableRunHuman renders live human progress (banner, streamed phases, final
	// summary). The default mode.
	DurableRunHuman DurableRunMode = iota
	// DurableRunJSON blocks to completion and renders one final JSON object.
	DurableRunJSON
	// DurableRunJSONL streams the canonical newline-delimited event vocabulary.
	DurableRunJSONL
)

// DurableRunModeFrom maps the --json / --jsonl output flags to a DurableRunMode.
// --jsonl wins when both are set (an event stream is the more specific request).
func DurableRunModeFrom(json, jsonl bool) DurableRunMode {
	switch {
	case jsonl:
		return DurableRunJSONL
	case json:
		return DurableRunJSON
	default:
		return DurableRunHuman
	}
}

// DurableRunSpec wires a durable_run command's mode-blind start to its per-mode
// renderers. H is the started-run handle the renderers consume (run id, ETA,
// client, follow context, printer, …).
//
// Start takes NO output mode: the run's start and follow ownership cannot branch
// on how output is rendered. Only the renderers differ per mode — that is the
// entire behavioral difference between human, --json, and --jsonl.
type DurableRunSpec[H any] struct {
	// Start begins the server-owned run and returns the handle to follow/render.
	Start func() (H, error)
	// RenderStartError renders a start failure for the mode (e.g. a JSON error
	// object vs human busy guidance). Optional: nil returns the raw start error.
	// It receives the mode because rendering an error is a rendering concern; the
	// start itself already ran mode-blind.
	RenderStartError func(mode DurableRunMode, err error) error
	// Human/JSON/JSONL render the started run for their mode.
	Human func(handle H) error
	JSON  func(handle H) error
	JSONL func(handle H) error
}

// RunDurable is the durable_run primitive: it starts the run ONCE (mode-blind)
// and dispatches the started handle to the mode's renderer. Because Start is the
// single start path for every mode, a durable_run command cannot select a
// separate synchronous lifecycle for --json — the guarantee behind verified
// durable_run maturity.
func RunDurable[H any](mode DurableRunMode, spec DurableRunSpec[H]) error {
	handle, err := spec.Start()
	if err != nil {
		if spec.RenderStartError != nil {
			return spec.RenderStartError(mode, err)
		}
		return err
	}
	switch mode {
	case DurableRunJSON:
		return spec.JSON(handle)
	case DurableRunJSONL:
		return spec.JSONL(handle)
	default:
		return spec.Human(handle)
	}
}

// LegacyPrimitiveHandler pairs an argv-parsing func([]string) error handler with
// the cli-core primitive class it was built from. It is the evidence-carrying
// analog of PrimitiveHandler for top-level commands that parse their own argv
// (rather than going through ArgSchema/RunContext) — for example a durable_run
// command like `test-genie execute`. The observed primitive is stamped in an
// UNEXPORTED field by a cli-core constructor (DurableRunLegacy) and travels onto
// the command's evidence via WithLegacyPrimitive, so a scenario cannot forge it;
// CLI Health reconciles it against the command's declared architecture exactly
// like a proto primitive.
type LegacyPrimitiveHandler struct {
	// primitive is the observed cli-core primitive class. Unexported so it cannot
	// be forged outside cli-core; read it via Primitive().
	primitive PrimitiveClass
	// Run is the argv-parsing handler closure.
	Run func(args []string) error
}

// Primitive returns the observed cli-core primitive class (empty for a zero
// handler). Read-only: only a cli-core constructor can stamp it.
func (h LegacyPrimitiveHandler) Primitive() PrimitiveClass { return h.primitive }

// DurableRunLegacy tags an argv-parsing durable_run command handler with
// durable_run evidence. Use it when the command's flow goes through RunDurable but
// the command still parses argv itself (rather than via ArgSchema/RunContext).
func DurableRunLegacy(run func(args []string) error) LegacyPrimitiveHandler {
	return LegacyPrimitiveHandler{primitive: PrimitiveDurableRun, Run: run}
}

// StreamingLegacy tags an argv-parsing server-stream/follow command with
// streaming evidence. Use it for commands that own a long-lived stream and
// therefore cannot be represented as a single proto_list/proto_mutation call.
func StreamingLegacy(run func(args []string) error) LegacyPrimitiveHandler {
	return LegacyPrimitiveHandler{primitive: PrimitiveStreaming, Run: run}
}

// WithLegacyPrimitive wires a LegacyPrimitiveHandler onto the command: it sets Run
// (the argv handler) and records the observed primitive class as evidence.
// Analogous to WithPrimitive for the RunCtx path.
func (c Command) WithLegacyPrimitive(h LegacyPrimitiveHandler) Command {
	c.Run = h.Run
	c.primitiveEvidence = h.primitive
	return c
}
