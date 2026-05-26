package destinations

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"

	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
	destinationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations/destinations_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"data-backup-manager/cli/internal/testutil"
)

// stubDestinationsService is an in-test DestinationsServiceHandler that records
// the request it receives and returns a canned destination.
type stubDestinationsService struct {
	gotCreate *destinationsv1.CreateDestinationRequest
}

func (s *stubDestinationsService) CreateDestination(_ context.Context, req *connect.Request[destinationsv1.CreateDestinationRequest]) (*connect.Response[destinationsv1.CreateDestinationResponse], error) {
	s.gotCreate = req.Msg
	return connect.NewResponse(&destinationsv1.CreateDestinationResponse{
		Destination: &destinationsv1.Destination{
			Id:          "dst-1",
			Name:        req.Msg.Name,
			BackendKind: req.Msg.BackendKind,
			Location:    req.Msg.Location,
		},
	}), nil
}

func (s *stubDestinationsService) GetDestination(_ context.Context, req *connect.Request[destinationsv1.GetDestinationRequest]) (*connect.Response[destinationsv1.GetDestinationResponse], error) {
	return connect.NewResponse(&destinationsv1.GetDestinationResponse{
		Destination: &destinationsv1.Destination{Id: req.Msg.Id},
	}), nil
}

func (s *stubDestinationsService) ListDestinations(_ context.Context, _ *connect.Request[destinationsv1.ListDestinationsRequest]) (*connect.Response[destinationsv1.ListDestinationsResponse], error) {
	return connect.NewResponse(&destinationsv1.ListDestinationsResponse{}), nil
}

func (s *stubDestinationsService) UpdateDestination(_ context.Context, req *connect.Request[destinationsv1.UpdateDestinationRequest]) (*connect.Response[destinationsv1.UpdateDestinationResponse], error) {
	return connect.NewResponse(&destinationsv1.UpdateDestinationResponse{
		Destination: &destinationsv1.Destination{Id: req.Msg.Id},
	}), nil
}

func (s *stubDestinationsService) DeleteDestination(_ context.Context, _ *connect.Request[destinationsv1.DeleteDestinationRequest]) (*connect.Response[destinationsv1.DeleteDestinationResponse], error) {
	return connect.NewResponse(&destinationsv1.DeleteDestinationResponse{Removed: true}), nil
}

func (s *stubDestinationsService) GetDestinationUsage(_ context.Context, req *connect.Request[destinationsv1.GetDestinationUsageRequest]) (*connect.Response[destinationsv1.GetDestinationUsageResponse], error) {
	return connect.NewResponse(&destinationsv1.GetDestinationUsageResponse{}), nil
}

// TestCreateDestinationCommand is the per-domain check: the create command parses
// its flags (including --backend → proto enum) and calls the generated
// DestinationsService client against a real Connect server.
func TestCreateDestinationCommand(t *testing.T) {
	stub := &stubDestinationsService{}
	mux := http.NewServeMux()
	path, h := destinationsconnect.NewDestinationsServiceHandler(stub)
	mux.Handle(path, h)
	app := testutil.NewTestApp(t, mux)

	hs := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "name"}, {Name: "backend"}, {Name: "location"}, {Name: "cap-bytes"}, {Name: "cap-policy"},
	}}
	ctx, stdout := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"name":     "my-backup",
			"backend":  "filesystem",
			"location": "/backups/my-backup",
		},
	})

	if err := hs.create(ctx); err != nil {
		t.Fatalf("create: %v", err)
	}
	out := stdout.String()

	if stub.gotCreate == nil {
		t.Fatal("server did not receive CreateDestination")
	}
	if stub.gotCreate.Name != "my-backup" || stub.gotCreate.Location != "/backups/my-backup" {
		t.Fatalf("request fields wrong: %+v", stub.gotCreate)
	}
	if stub.gotCreate.BackendKind != destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM {
		t.Fatalf("backend not mapped to enum: %v", stub.gotCreate.BackendKind)
	}
	if !strings.Contains(out, "dst-1") {
		t.Fatalf("output missing destination id: %q", out)
	}
}

// TestParseBackendKind pins the --backend flag → proto enum mapping.
func TestParseBackendKind(t *testing.T) {
	cases := map[string]destinationsv1.BackendKind{
		"filesystem": destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM,
		"s3":         destinationsv1.BackendKind_BACKEND_KIND_S3,
	}
	for in, want := range cases {
		got, err := parseBackendKind(in)
		if err != nil || got != want {
			t.Errorf("parseBackendKind(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := parseBackendKind("nope"); err == nil {
		t.Error("parseBackendKind(\"nope\") should error")
	}
}

// TestRegisterDestinationsLoadsFromManifest proves the manifest wiring produces
// the expected subcommands.
func TestRegisterDestinationsLoadsFromManifest(t *testing.T) {
	manifest := readManifest(t)
	app := &cliapp.ScenarioApp{}
	group, err := Register(app, manifest)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	got := map[string]bool{}
	for _, c := range group.Subcommands {
		got[c.Name] = true
	}
	for _, want := range []string{"create", "get", "list", "update", "delete", "usage"} {
		if !got[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}
