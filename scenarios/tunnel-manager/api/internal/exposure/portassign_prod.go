package exposure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/schedule"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation/validation_v1connect"
)

// structureHealthScenario is the sibling scenario that owns the port-band SSOT
// and the assign/release primitive TM calls to make a ranged scenario exposable.
const structureHealthScenario = "structure-health"

// StructureHealthPortAssigner is the production PortAssigner. It resolves
// structure-health's API at call time (via the discovery seam) and drives its
// ValidationService AssignFixedPort/ReleaseFixedPort RPCs. It is best-effort by
// design: when structure-health is unreachable EnsureFixed returns an error the
// service swallows, so an already-fixed scenario still exposes.
type StructureHealthPortAssigner struct {
	// Resolve returns structure-health's base URL. Defaults to the discovery
	// package helper; injectable for tests.
	Resolve func(ctx context.Context) (string, error)
	// HTTPClient is the transport for the Connect client (defaults to a 10s
	// http.Client).
	HTTPClient connect.HTTPClient
	// PortName is the listener port to switch (defaults to "ui").
	PortName string
}

var _ PortAssigner = (*StructureHealthPortAssigner)(nil)

func (a *StructureHealthPortAssigner) client(ctx context.Context) (validationconnect.ValidationServiceClient, error) {
	resolve := a.Resolve
	if resolve == nil {
		resolve = func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, structureHealthScenario)
		}
	}
	baseURL, err := resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve %s API: %w", structureHealthScenario, err)
	}
	httpClient := a.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return validationconnect.NewValidationServiceClient(httpClient, baseURL), nil
}

func (a *StructureHealthPortAssigner) EnsureFixed(ctx context.Context, scenario string) (bool, error) {
	client, err := a.client(ctx)
	if err != nil {
		return false, err
	}
	resp, err := client.AssignFixedPort(ctx, connect.NewRequest(&validationv1.PortSwitchRequest{
		Scenario: scenario,
		PortName: a.PortName,
		Apply:    true,
	}))
	if err != nil {
		return false, fmt.Errorf("assign fixed port for %q via structure-health: %w", scenario, err)
	}
	if resp == nil || resp.Msg == nil {
		return false, fmt.Errorf("structure-health returned no assign-port response")
	}
	// assignedByTM only when TM switched a previously-ranged scenario this call.
	return resp.Msg.GetChanged() && resp.Msg.GetAssignedPort() > 0, nil
}

func (a *StructureHealthPortAssigner) Release(ctx context.Context, scenario string) error {
	client, err := a.client(ctx)
	if err != nil {
		return err
	}
	_, err = client.ReleaseFixedPort(ctx, connect.NewRequest(&validationv1.PortSwitchRequest{
		Scenario: scenario,
		PortName: a.PortName,
		Apply:    true,
	}))
	if err != nil {
		return fmt.Errorf("release fixed port for %q via structure-health: %w", scenario, err)
	}
	return nil
}

// sqlitePortOwnership is the production PortOwnership over the
// tm_port_assignments table.
type sqlitePortOwnership struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLitePortOwnership constructs the production PortOwnership store.
func NewSQLitePortOwnership(db SQLExecutor, clk schedule.Clock) PortOwnership {
	if clk == nil {
		clk = schedule.System()
	}
	return &sqlitePortOwnership{db: db, clock: clk}
}

var _ PortOwnership = (*sqlitePortOwnership)(nil)

func (s *sqlitePortOwnership) Record(ctx context.Context, scenario string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tm_port_assignments (scenario, assigned_at) VALUES (?, ?)
		 ON CONFLICT(scenario) DO UPDATE SET assigned_at = excluded.assigned_at`,
		scenario, s.clock.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("record port ownership %q: %w", scenario, err)
	}
	return nil
}

func (s *sqlitePortOwnership) Owned(ctx context.Context, scenario string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM tm_port_assignments WHERE scenario = ?", scenario).Scan(&one)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("query port ownership %q: %w", scenario, err)
	}
	return true, nil
}

func (s *sqlitePortOwnership) Clear(ctx context.Context, scenario string) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM tm_port_assignments WHERE scenario = ?", scenario); err != nil {
		return fmt.Errorf("clear port ownership %q: %w", scenario, err)
	}
	return nil
}
