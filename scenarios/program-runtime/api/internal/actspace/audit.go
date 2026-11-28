// Package actspace owns the denominator-to-registry audit seam. The
// denominator remains a spacedoc; the registry remains the mechanical source
// of callable truth. This package only translates between the two.
package actspace

import (
	"context"

	"github.com/vrooli/api-core/spacedoc"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
)

// Registry is the only capability this audit needs from the live registry.
type Registry interface {
	ResolveActCells(context.Context, []*bindingsv1.ActCell) []*bindingsv1.ActCellVerdict
}

// Audit converts every denominator cell into the registry's typed audit RPC
// shape. The owner field is deliberately preserved as the operation token: it
// is the stable bridge between the authored taxonomy and the live namespace.
func Audit(ctx context.Context, registry Registry, definition *spacedoc.SpaceDefinition) []*bindingsv1.ActCellVerdict {
	if definition == nil || registry == nil {
		return nil
	}
	cells := make([]*bindingsv1.ActCell, 0, len(definition.Cells))
	for _, cell := range definition.Cells {
		cells = append(cells, &bindingsv1.ActCell{Id: cell.ID, Operations: []string{cell.Owner}, AuthoredStatus: string(cell.Status)})
	}
	return registry.ResolveActCells(ctx, cells)
}

// Confidence is derived strictly from audit coverage. A complete audit is
// measured but partial because explicit unbound/external cells remain honest
// gaps; an incomplete audit stays sketch.
func Confidence(verdicts []*bindingsv1.ActCellVerdict) spacedoc.DenominatorConfidence {
	if len(verdicts) == 0 {
		return spacedoc.ConfidenceSketch
	}
	for _, verdict := range verdicts {
		if verdict == nil || !verdict.GetAudited() {
			return spacedoc.ConfidenceSketch
		}
	}
	return spacedoc.ConfidencePartial
}
