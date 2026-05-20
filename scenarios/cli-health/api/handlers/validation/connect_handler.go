// Package validation hosts the Connect-RPC handler for cli-health's
// ValidationService. Phase 1 wires the surface end-to-end: every RPC
// returns connect.CodeUnimplemented so callers (CLI, UI, tests) prove the
// proto/Connect path is healthy. Phase 2 fills in the real validators.
package validation

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/validation"
)

// Deps wires the seams the Connect validation handler needs. Logger is the
// only seam in Phase 1; Phase 2 will add the manifest/proto loaders here.
type Deps struct {
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler returns a handler that satisfies the generated
// ValidationServiceHandler interface. Phase 1 returns Unimplemented for
// every RPC; the wiring proves proto+Connect plumbing.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ValidateScenario(_ context.Context, _ *connect.Request[validationv1.ValidateScenarioRequest]) (*connect.Response[validationv1.ValidateScenarioResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation.ValidateScenario: not yet implemented"))
}
