package capabilityapp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	portabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability"
	portabilityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability/portability_v1connect"
)

// capabilityScenario owns the capability portability grid. The control plane
// used to aggregate the grid itself, which meant two places computed an answer
// about the same manifests and only one of them was the instrument.
const capabilityScenario = "infrastructure-manager"

// capabilityRequestTimeout bounds one delegated read. The grid walks the
// repository's manifest tree, so it is slower than a status ping and still
// bounded: an unbounded wait would turn a wedged scenario into a wedged CLI.
const capabilityRequestTimeout = 30 * time.Second

// capabilityDegradedError is the explicit degraded state. The control plane
// reports that it could not reach the owner and names the command that fixes
// it; it never falls back to computing a partial grid locally, because a
// partial grid printed in the same shape as a complete one is a wrong answer
// that looks like a right one.
type capabilityDegradedError struct {
	Operation string
	Err       error
}

func (e capabilityDegradedError) Error() string {
	return fmt.Sprintf(
		"capability %s is degraded: the %s scenario owns the capability grid and is unreachable (%v). No grid is printed, because a partial grid is indistinguishable from a complete one. Start the owner with: vrooli scenario start %s",
		e.Operation, capabilityScenario, e.Err, capabilityScenario,
	)
}

func (e capabilityDegradedError) Unwrap() error { return e.Err }

// capabilityDegradedReadout is the machine-readable form of the same state, so
// a `--json` consumer receives an explicit degraded envelope rather than an
// empty grid it would read as "this repository declares no capabilities".
type capabilityDegradedReadout struct {
	State       string `json:"state"`
	Operation   string `json:"operation"`
	Owner       string `json:"owner"`
	Reason      string `json:"reason"`
	Remediation string `json:"remediation"`
}

func newCapabilityDegradedReadout(err capabilityDegradedError) capabilityDegradedReadout {
	return capabilityDegradedReadout{
		State:       "degraded",
		Operation:   err.Operation,
		Owner:       capabilityScenario,
		Reason:      fmt.Sprintf("the %s scenario is unreachable: %v", capabilityScenario, err.Err),
		Remediation: "vrooli scenario start " + capabilityScenario,
	}
}

// capabilityClient resolves the owning scenario and calls its typed read
// surface.
func capabilityClient(ctx context.Context) (portabilityconnect.PortabilityServiceClient, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, capabilityScenario)
	if err != nil {
		return nil, err
	}
	return portabilityconnect.NewPortabilityServiceClient(&http.Client{Timeout: capabilityRequestTimeout}, base), nil
}

func fetchCapabilityGrid(ctx context.Context) (*portabilityv1.Grid, error) {
	client, err := capabilityClient(ctx)
	if err != nil {
		return nil, capabilityDegradedError{Operation: "ledger", Err: err}
	}
	resp, err := client.GetGrid(ctx, connect.NewRequest(&portabilityv1.GetGridRequest{}))
	if err != nil {
		return nil, capabilityDegradedError{Operation: "ledger", Err: err}
	}
	grid := resp.Msg.GetGrid()
	if grid == nil {
		return nil, capabilityDegradedError{Operation: "ledger", Err: fmt.Errorf("the owner returned no grid")}
	}
	return grid, nil
}

func fetchCapabilityFleet(ctx context.Context) (*portabilityv1.FleetReadout, error) {
	client, err := capabilityClient(ctx)
	if err != nil {
		return nil, capabilityDegradedError{Operation: "fleet", Err: err}
	}
	resp, err := client.GetFleet(ctx, connect.NewRequest(&portabilityv1.GetFleetRequest{}))
	if err != nil {
		return nil, capabilityDegradedError{Operation: "fleet", Err: err}
	}
	fleet := resp.Msg.GetFleet()
	if fleet == nil {
		return nil, capabilityDegradedError{Operation: "fleet", Err: fmt.Errorf("the owner returned no fleet readout")}
	}
	return fleet, nil
}
