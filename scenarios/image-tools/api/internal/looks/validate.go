// Chain validation for Looks.
//
// A Look is a linear step chain. Before this file existed the store checked only
// that each step named a catalog op: an operator could save a Look that put a
// text-producing step before an image-consuming one, and nothing noticed until
// the run failed (or silently did the wrong thing).
//
// Validation is deliberately data-driven: each step resolves to a capability
// manifest, whose declared ports answer "what does this consume, what does it
// produce". The only non-data inputs are (a) the engine lane, derived from the
// catalogs, and (b) the workflow source, which supplies the first image.
package looks

import (
	"fmt"
	"strings"

	"image-tools/internal/ai"
	"image-tools/internal/capability"
	internalops "image-tools/internal/ops"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

// ImageToolsScenario is the only scenario whose steps this service can execute
// today. Other values are representable in the contract (see COMPOSITION.md) but
// rejected for execution rather than silently skipped.
const ImageToolsScenario = "image-tools"

// invalid wraps ErrInvalid so the Connect handler maps every validation failure
// to InvalidArgument.
func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// LaneForOperation derives the engine lane a Look step must use. The lane is a
// catalog membership question, not the capability mechanism: `naturalize` is a
// compute-mechanism op that still runs on the model-backed job lane, so deriving
// from mechanism would misclassify it.
func LaneForOperation(op string) (looksv1.StepKind, bool) {
	if _, ok := ai.Get(op); ok {
		return looksv1.StepKind_STEP_KIND_AI, true
	}
	if internalops.Has(op) {
		return looksv1.StepKind_STEP_KIND_DETERMINISTIC, true
	}
	return looksv1.StepKind_STEP_KIND_UNSPECIFIED, false
}

// Validate enforces the structural and type rules a usable Look must satisfy:
// a name, at least one step, every step resolving to an executable operation,
// an explicit kind (when present) matching that operation's lane, and every
// required input satisfiable by the previous step, the workflow source, or an
// external mask/prompt.
func Validate(look *looksv1.Look) error {
	if look == nil {
		return invalid("look is required")
	}
	if strings.TrimSpace(look.GetName()) == "" {
		return invalid("a name is required")
	}
	steps := look.GetSteps()
	if len(steps) == 0 {
		return invalid("a Look needs at least one step")
	}
	prevOutput := ""
	for i, step := range steps {
		if err := validateStep(i, step, prevOutput); err != nil {
			return err
		}
		manifest, _ := capability.Get(step.GetOperation())
		prevOutput = manifest.IO.Output.Kind
	}
	return nil
}

func validateStep(index int, step *looksv1.LookStep, prevOutput string) error {
	op := step.GetOperation()
	label := fmt.Sprintf("step %d (%s)", index+1, op)

	if op == "" {
		return invalid("step %d has no operation", index+1)
	}
	if scenario := step.GetScenario(); scenario != "" && scenario != ImageToolsScenario {
		return invalid("%s: cross-scenario step %q is not executable here yet; author it in the workflow surface once the shared operation catalog lands", label, scenario)
	}
	manifest, ok := capability.Get(op)
	if !ok {
		return invalid("%s: unknown operation", label)
	}
	lane, executable := LaneForOperation(op)
	if !executable {
		return invalid("%s: this operation is not available as a workflow step (it has no durable AI or deterministic runner)", label)
	}
	switch step.GetKind() {
	case looksv1.StepKind_STEP_KIND_UNSPECIFIED:
		// Derived; Compile fills the resolved lane.
	case lane:
		// Explicit and correct.
	default:
		return invalid("%s: declared kind %s does not match the operation's lane %s", label, step.GetKind(), lane)
	}

	if declared := step.GetOutputKind(); declared != "" && declared != manifest.IO.Output.Kind {
		return invalid("%s: declared output kind %q does not match the operation's %q", label, declared, manifest.IO.Output.Kind)
	}

	ports := make(map[string]capability.PortSpec, len(manifest.IO.Inputs))
	for _, port := range manifest.IO.Inputs {
		ports[port.Name] = port
	}
	fromSource := make(map[string]bool, len(step.GetInputs()))
	for _, in := range step.GetInputs() {
		port, ok := ports[in.GetName()]
		if !ok {
			return invalid("%s: declares input %q, which the operation does not have", label, in.GetName())
		}
		if in.GetKind() != "" && in.GetKind() != port.Kind {
			return invalid("%s: input %q declared kind %q does not match %q", label, in.GetName(), in.GetKind(), port.Kind)
		}
		switch in.GetFrom() {
		case "", "previous":
		case "source":
			fromSource[in.GetName()] = true
		default:
			return invalid("%s: input %q has unknown source %q (expected source or previous)", label, in.GetName(), in.GetFrom())
		}
	}

	for _, port := range manifest.IO.Inputs {
		if !port.Required {
			continue
		}
		switch port.Kind {
		case "mask", "text":
			// Supplied externally / filled from the Look prompt.
			continue
		case "image":
			if index == 0 || prevOutput == "image" || fromSource[port.Name] {
				continue
			}
			return invalid("%s: needs an image input, but the previous step produces %q", label, prevOutput)
		default:
			if prevOutput == port.Kind || fromSource[port.Name] {
				continue
			}
			return invalid("%s: needs a %q input, which no previous step produces", label, port.Kind)
		}
	}
	return nil
}
