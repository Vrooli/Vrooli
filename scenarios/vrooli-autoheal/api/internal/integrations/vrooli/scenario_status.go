package vrooli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type ScenarioStatus struct {
	Success  bool
	Scenario ScenarioStatusDetail
}

type ScenarioStatusDetail struct {
	Name         string
	Status       string
	HealthStatus json.RawMessage
	// Ports maps a role name (e.g. "API_PORT", "UI_PORT") to its allocated
	// port. Carried through because the scenario-level health_status reflects
	// the API probe only: a scenario whose UI has stopped answering still
	// reports "healthy". Component-level reachability needs the port numbers.
	Ports map[string]int32
	// HealthError is the health probe's error detail; empty on success.
	HealthError string
}

// Port returns the allocated port for a role and whether it was present.
func (s ScenarioStatusDetail) Port(role string) (int32, bool) {
	if len(s.Ports) == 0 {
		return 0, false
	}
	port, ok := s.Ports[role]
	if !ok || port <= 0 {
		return 0, false
	}
	return port, true
}

func (s ScenarioStatusDetail) NormalizedHealthStatus() (string, error) {
	if len(s.HealthStatus) == 0 || string(s.HealthStatus) == "null" {
		return "", nil
	}

	var health string
	if err := json.Unmarshal(s.HealthStatus, &health); err != nil {
		return "", ContractError{
			Field:   "scenario.health_status",
			Message: fmt.Sprintf("expected string or null: %v", err),
		}
	}
	return strings.ToLower(strings.TrimSpace(health)), nil
}

// ScenarioStatus runs `vrooli scenario status <name> --json` and returns the
// typed status mapped onto the autoheal domain shape.
func (c *Client) ScenarioStatus(ctx context.Context, scenarioName string) (ScenarioStatus, []byte, error) {
	resp, err := c.cli.ScenarioStatus(ctx, scenarioName)
	if err != nil {
		return ScenarioStatus{}, nil, err
	}
	status, err := scenarioStatusFromProto(resp)
	if err != nil {
		return ScenarioStatus{}, nil, err
	}
	return status, diagnosticJSON(resp), nil
}

// ParseScenarioStatus decodes proto-JSON `vrooli scenario status --json` bytes
// into the domain status, for callers that already hold the raw output.
func ParseScenarioStatus(output []byte) (ScenarioStatus, error) {
	var resp cliv1.ScenarioStatusSingle
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(output, &resp); err != nil {
		return ScenarioStatus{}, err
	}
	return scenarioStatusFromProto(&resp)
}

func scenarioStatusFromProto(resp *cliv1.ScenarioStatusSingle) (ScenarioStatus, error) {
	sc := resp.GetScenario()
	detail := ScenarioStatusDetail{
		Name:         sc.GetName(),
		Status:       sc.GetStatus(),
		HealthStatus: valueToRawJSON(sc.GetHealthStatus()),
		Ports:        sc.GetPorts(),
		HealthError:  sc.GetHealthError(),
	}
	if err := requiredString(detail.Status, "scenario.status"); err != nil {
		return ScenarioStatus{}, err
	}
	if _, err := detail.NormalizedHealthStatus(); err != nil {
		return ScenarioStatus{}, err
	}
	return ScenarioStatus{Success: resp.GetSuccess(), Scenario: detail}, nil
}
