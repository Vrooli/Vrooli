package deploymentsvc

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"

	deploymentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments"
	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/errors"
)

func sampleDeployment() *domain.Deployment {
	created := time.Date(2026, 9, 9, 10, 0, 0, 0, time.UTC)
	deployed := created.Add(time.Hour)
	sha := "sha256:bundle"
	step := "deploying"
	return &domain.Deployment{
		ID:          "0f0e1d2c-3b4a-4596-8778-99aabbccddee",
		Name:        "demo-app @ demo.example",
		ScenarioID:  "demo-app",
		Environment: "production",
		Target: identity.TargetRef{
			MachineID:            "machine-7",
			NodeID:               "node-7",
			EnrollmentGeneration: 2,
			Transport:            identity.TransportBridge,
			Locator:              identity.TargetLocator{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli"},
		},
		Fence:           4,
		Status:          domain.StatusDeployed,
		Manifest:        json.RawMessage(`{"version":"1","scenario":{"id":"demo-app"},"edge":{"domain":"demo.example"},"target":{"type":"vps","vps":{"host":"203.0.113.10"}}}`),
		BundleSHA256:    &sha,
		ProgressStep:    &step,
		ProgressPercent: 100,
		CreatedAt:       created,
		UpdatedAt:       deployed,
		LastDeployedAt:  &deployed,
	}
}

// TestDeploymentRoundTripsThroughProtoJSON [REQ:STC-P0-015] proves the
// generated Go contract carries the full identity with proto field names on
// the wire and that the UI fixture in ui/src/lib is the same bytes.
func TestDeploymentRoundTripsThroughProtoJSON(t *testing.T) {
	msg, err := DeploymentProto(sampleDeployment())
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	wire, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(wire, &generic); err != nil {
		t.Fatalf("wire is not JSON: %v", err)
	}
	ref, _ := generic["ref"].(map[string]any)
	if ref["scenario_id"] != "demo-app" || ref["environment"] != "production" {
		t.Fatalf("wire must use proto names: %s", wire)
	}
	if _, camel := ref["scenarioId"]; camel {
		t.Fatalf("wire must not use lowerCamel names: %s", wire)
	}
	target, _ := ref["target"].(map[string]any)
	if target["machine_id"] != "machine-7" || target["enrollment_generation"] != "2" {
		t.Fatalf("target identity lost on the wire: %v", target)
	}
	if generic["fence"] != "4" {
		t.Fatalf("fence lost on the wire: %v", generic["fence"])
	}

	var parsed deploymentsv1.Deployment
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(wire, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !proto.Equal(msg, &parsed) {
		t.Fatalf("round trip changed the message:\n%s", wire)
	}
	back := DeploymentRefFromProto(parsed.GetRef())
	if back != sampleDeployment().Ref() {
		t.Fatalf("domain ref changed: %+v", back)
	}

	// The UI test parses this exact fixture through the generated schema.
	// STC_WRITE_FIXTURE=1 regenerates it from this sample (golden update).
	fixture := filepath.Join("..", "..", "ui", "src", "lib", "fixtures", "deployment.v1.json")
	if os.Getenv("STC_WRITE_FIXTURE") == "1" {
		pretty, err := protojson.MarshalOptions{UseProtoNames: true, Multiline: true, Indent: "  "}.Marshal(msg)
		if err != nil {
			t.Fatalf("marshal fixture: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(fixture), 0o755); err != nil {
			t.Fatalf("mkdir fixture dir: %v", err)
		}
		if err := os.WriteFile(fixture, append(pretty, '\n'), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}
	stored, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read UI fixture %s: %v", fixture, err)
	}
	var storedMsg deploymentsv1.Deployment
	if err := protojson.Unmarshal(stored, &storedMsg); err != nil {
		t.Fatalf("UI fixture is not a valid Deployment: %v", err)
	}
	if !proto.Equal(msg, &storedMsg) {
		t.Fatalf("UI fixture drifted from the Go projection; regenerate it from this test's sample")
	}
}

// TestConnectErrorCarriesTypedDetail [REQ:STC-P0-017] proves P03-A06 for the
// Connect transport: the same stable code reaches the client as an
// errors.v1.Error detail with next action and details intact.
func TestConnectErrorCarriesTypedDetail(t *testing.T) {
	typed := apierrors.New(apierrors.CodeDeploymentSelectorAmbiguous, "two match").
		WithDetail("candidates", []map[string]any{{"id": "a"}, {"id": "b"}}).
		WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "selector", Reference: "id", Label: "Select by id"})
	cerr := ConnectError(typed)
	if cerr.Code() != connect.CodeAborted {
		t.Fatalf("connect code = %v", cerr.Code())
	}
	var found *errorsv1.Error
	for _, detail := range cerr.Details() {
		value, err := detail.Value()
		if err != nil {
			continue
		}
		if e, ok := value.(*errorsv1.Error); ok {
			found = e
		}
	}
	if found == nil {
		t.Fatalf("no errors.v1.Error detail on %v", cerr)
	}
	if found.GetCode() != apierrors.CodeDeploymentSelectorAmbiguous || found.GetNextAction().GetReference() != "id" {
		t.Fatalf("detail = %v", found)
	}
	candidates := found.GetDetails().GetFields()["candidates"].GetListValue().GetValues()
	if len(candidates) != 2 {
		t.Fatalf("candidates = %v", found.GetDetails())
	}

	plain := ConnectError(errors.New("boom"))
	if plain.Code() != connect.CodeInternal {
		t.Fatalf("untyped error must be internal, got %v", plain.Code())
	}
}

type fakeRepo struct {
	deployments []*domain.Deployment
}

func (f *fakeRepo) ResolveDeployments(_ context.Context, sel identity.Selector) ([]identity.DeploymentRef, error) {
	var out []identity.DeploymentRef
	for _, d := range f.deployments {
		if sel.ID != "" && d.ID != sel.ID {
			continue
		}
		if sel.ScenarioID != "" && d.ScenarioID != sel.ScenarioID {
			continue
		}
		if sel.Environment != "" && d.Environment != sel.Environment {
			continue
		}
		out = append(out, d.Ref())
	}
	return out, nil
}

func (f *fakeRepo) GetDeployment(_ context.Context, id string) (*domain.Deployment, error) {
	for _, d := range f.deployments {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, nil
}

func (f *fakeRepo) ListDeployments(_ context.Context, filter domain.ListFilter) ([]*domain.Deployment, error) {
	var out []*domain.Deployment
	for i, d := range f.deployments {
		if i < filter.Offset {
			continue
		}
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
		out = append(out, d)
	}
	return out, nil
}

// TestServiceResolveGetList exercises the three RPCs over a fake repository.
func TestServiceResolveGetList(t *testing.T) {
	a := sampleDeployment()
	b := sampleDeployment()
	b.ID = "1f0e1d2c-3b4a-4596-8778-99aabbccddee"
	b.Environment = "staging"
	svc := New(&fakeRepo{deployments: []*domain.Deployment{a, b}})
	ctx := context.Background()

	resolved, err := svc.ResolveDeployment(ctx, connect.NewRequest(&deploymentsv1.ResolveDeploymentRequest{Selector: &deploymentsv1.DeploymentSelector{ScenarioId: "demo-app", Environment: "staging"}}))
	if err != nil || resolved.Msg.GetRef().GetId() != b.ID || resolved.Msg.GetSchemaVersion() != SchemaVersion {
		t.Fatalf("resolve = %v, %v", resolved, err)
	}
	_, err = svc.ResolveDeployment(ctx, connect.NewRequest(&deploymentsv1.ResolveDeploymentRequest{Selector: &deploymentsv1.DeploymentSelector{ScenarioId: "demo-app"}}))
	var cerr *connect.Error
	if !errors.As(err, &cerr) || cerr.Code() != connect.CodeInvalidArgument {
		t.Fatalf("scenario-only selector must be invalid: %v", err)
	}

	got, err := svc.GetDeployment(ctx, connect.NewRequest(&deploymentsv1.GetDeploymentRequest{Id: a.ID}))
	if err != nil || got.Msg.GetDeployment().GetFence() != 4 || got.Msg.GetDeployment().GetDomain() != "demo.example" {
		t.Fatalf("get = %v, %v", got, err)
	}
	_, err = svc.GetDeployment(ctx, connect.NewRequest(&deploymentsv1.GetDeploymentRequest{Id: "missing"}))
	if !errors.As(err, &cerr) || cerr.Code() != connect.CodeNotFound {
		t.Fatalf("missing must be not found: %v", err)
	}

	page, err := svc.ListDeployments(ctx, connect.NewRequest(&deploymentsv1.ListDeploymentsRequest{PageSize: 1}))
	if err != nil || len(page.Msg.GetDeployments()) != 1 || page.Msg.GetNextPageToken() != "1" {
		t.Fatalf("first page = %v, %v", page, err)
	}
	page, err = svc.ListDeployments(ctx, connect.NewRequest(&deploymentsv1.ListDeploymentsRequest{PageSize: 1, PageToken: page.Msg.GetNextPageToken()}))
	if err != nil || len(page.Msg.GetDeployments()) != 1 || page.Msg.GetNextPageToken() != "" || page.Msg.GetDeployments()[0].GetRef().GetEnvironment() != "staging" {
		t.Fatalf("second page = %v, %v", page, err)
	}
	_, err = svc.ListDeployments(ctx, connect.NewRequest(&deploymentsv1.ListDeploymentsRequest{PageToken: "not-a-number"}))
	if !errors.As(err, &cerr) || cerr.Code() != connect.CodeInvalidArgument {
		t.Fatalf("bad token must be invalid argument: %v", err)
	}
}
