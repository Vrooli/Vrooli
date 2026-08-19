package api

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	cliv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1/cliv1connect"
)

func TestScenarioStatusMapsNewerRegistryToActionablePrecondition(t *testing.T) {
	cause := &scenarioruntime.SchemaCompatibilityError{DatabaseVersion: 8, BinaryVersion: 7}
	err := controlPlaneScenarioStatusError(cause)
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("error code = %s, want failed_precondition; error = %v", connect.CodeOf(err), err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("error = %v, want wrapped schema compatibility cause", err)
	}
	for _, fragment := range []string{"schema_version 8", "supported 7", "vrooli develop"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("error = %q, want actionable fragment %q", err, fragment)
		}
	}
}

func TestScenarioControlPlaneServiceIsMounted(t *testing.T) {
	app := New(ResolveRepoRoot(), t.TempDir())
	server := httptest.NewServer(app.Router())
	defer server.Close()
	client := cliv1connect.NewScenarioControlPlaneServiceClient(server.Client(), server.URL)

	if _, err := client.ListScenarios(context.Background(), connect.NewRequest(&cliv1.ListScenariosRequest{})); err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	requests := []struct {
		name string
		call func() error
	}{
		{name: "GetScenarioStatus", call: func() error {
			_, err := client.GetScenarioStatus(context.Background(), connect.NewRequest(&cliv1.GetScenarioStatusRequest{}))
			return err
		}},
		{name: "GetScenarioLogs", call: func() error {
			_, err := client.GetScenarioLogs(context.Background(), connect.NewRequest(&cliv1.GetScenarioLogsRequest{}))
			return err
		}},
		{name: "StartScenario", call: func() error {
			_, err := client.StartScenario(context.Background(), connect.NewRequest(&cliv1.StartScenarioRequest{}))
			return err
		}},
		{name: "StopScenario", call: func() error {
			_, err := client.StopScenario(context.Background(), connect.NewRequest(&cliv1.StopScenarioRequest{}))
			return err
		}},
		{name: "RestartScenario", call: func() error {
			_, err := client.RestartScenario(context.Background(), connect.NewRequest(&cliv1.RestartScenarioRequest{}))
			return err
		}},
		{name: "SetupScenario", call: func() error {
			_, err := client.SetupScenario(context.Background(), connect.NewRequest(&cliv1.SetupScenarioRequest{}))
			return err
		}},
	}
	for _, request := range requests {
		t.Run(request.name, func(t *testing.T) {
			err := request.call()
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("error code = %s, want invalid_argument; error = %v", connect.CodeOf(err), err)
			}
		})
	}
}
