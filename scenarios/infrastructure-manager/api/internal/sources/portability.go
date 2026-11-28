package sources

import (
	"context"
	"time"

	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/portability"
)

// PortabilitySourceID identifies the capability-grid source in availability
// reporting.
const PortabilitySourceID = "infrastructure-manager/portability"

// PortabilityReader reads the capability grid this scenario itself owns.
//
// It is a source rather than a direct call because the ladder must report a
// failed capability read the same way it reports a failed peer read. An
// in-process dependency that cannot fail is a dependency whose failure is
// invisible, and the host-OS dimension of every ladder cell is joined from
// this grid.
type PortabilityReader struct {
	Grid interface {
		Grid(ctx context.Context) (portability.Grid, error)
	}
}

func (r PortabilityReader) ReadGrid(ctx context.Context) (portability.Grid, error) {
	if r.Grid == nil {
		return portability.Grid{}, errPortabilityUnwired
	}
	return r.Grid.Grid(ctx)
}

// ReadPortability runs the capability-grid read under the standard per-source
// deadline.
func ReadPortability(ctx context.Context, reader PortabilityReader, timeout time.Duration) TypedResult[portability.Grid] {
	return ReadTyped(ctx, PortabilitySourceID, reader.ReadGrid, timeout)
}
