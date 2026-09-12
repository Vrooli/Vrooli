package releases

import (
	"fmt"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

// CellResult is the release-owned projection of one matrix cell.
type CellResult struct {
	ID          string
	Required    bool
	Disposition deliveryramp.Disposition
	Target      deliveryramp.Target
	References  []deliveryramp.EvidenceReference
}

// EvaluateGate requires every required cell to pass with producer-owned,
// reference-only evidence. Missing and unavailable cells stay terminal.
func EvaluateGate(cells []CellResult, producer, runID, detail string) (deliveryramp.Disposition, error) {
	if len(cells) == 0 {
		return deliveryramp.DispositionFailed, fmt.Errorf("Android release gate has no validation cells")
	}
	for _, cell := range cells {
		if !cell.Required {
			continue
		}
		if cell.Disposition != deliveryramp.DispositionPass {
			return deliveryramp.DispositionFailed, fmt.Errorf("required Android cell %q is %s", cell.ID, cell.Disposition)
		}
		if _, err := deliveryramp.NewTargetVerdict(deliveryramp.TargetVerdictInput{Producer: producer, Target: cell.Target, Disposition: cell.Disposition, RunID: runID, Detail: detail, References: cell.References}); err != nil {
			return deliveryramp.DispositionFailed, fmt.Errorf("required Android cell %q has invalid evidence: %w", cell.ID, err)
		}
	}
	return deliveryramp.DispositionPass, nil
}
