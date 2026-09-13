package teams

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"prompt-manager/internal/store"
	"prompt-manager/internal/testutil/fixtures"
)

func TestTeamMetadataConnectCRUDPreservesRecordsAndExplicitClears(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	relations := store.NewFileRelationStore(root)
	teamStore := store.NewFileTeamStore(root, root, relations)
	handler := NewHandlers(teamStore, store.NewFileAgentStore(root), relations, nil, nil)
	path, mount := NewConnectMount(handler)
	mux := http.NewServeMux()
	mux.Handle(path, mount)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := teamsconnect.NewTeamsServiceClient(server.Client(), server.URL)
	fixture := fixtures.IndependentTeam("aquila", "Aquila", fixtures.WithEnabled(false))
	fixture.Purpose = "delivery"
	fixture.Lifetime = "finite"
	fixture.EffortRefs = []string{"effort:aquila"}
	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	input := &teamsv1.TeamInput{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, input); err != nil {
		t.Fatal(err)
	}
	created, err := client.CreateTeam(ctx, connect.NewRequest(&teamsv1.CreateTeamRequest{Team: input}))
	if err != nil {
		t.Fatal(err)
	}
	if created.Msg.Purpose != "delivery" || created.Msg.Lifetime != "finite" || len(created.Msg.EffortRefs) != 1 || created.Msg.Enabled {
		t.Fatalf("create metadata: %+v", created.Msg)
	}
	teamPath := filepath.Join(root, "teams", "aquila", "team.json")
	stored, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(stored, &document); err != nil {
		t.Fatal(err)
	}
	document["objectivesServed"] = json.RawMessage(`[{"id":"T1","role":"contributor","coverage":"partial","note":"retain","acknowledgedRevision":"rev3","unknown":"retained"}]`)
	document["instrument"] = json.RawMessage(`{"retain":true}`)
	if err := store.SaveJSON(teamPath, &document); err != nil {
		t.Fatal(err)
	}
	updated, err := client.UpdateTeam(ctx, connect.NewRequest(&teamsv1.UpdateTeamRequest{
		Id: "aquila", Team: &teamsv1.TeamInput{DisplayName: "Aquila launch", Lifetime: "standing"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name", "lifetime"}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Msg.Purpose != "delivery" || updated.Msg.Lifetime != "standing" || len(updated.Msg.EffortRefs) != 1 || updated.Msg.Enabled {
		t.Fatalf("independent update: %+v", updated.Msg)
	}
	if len(updated.Msg.ObjectivesServed) != 1 || updated.Msg.ObjectivesServed[0].AcknowledgedRevision != "rev3" {
		t.Fatalf("objectives lost: %+v", updated.Msg.ObjectivesServed)
	}
	listed, err := client.ListTeams(ctx, connect.NewRequest(&teamsv1.ListTeamsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Msg.Teams) != 1 || listed.Msg.Teams[0].Purpose != "delivery" || len(listed.Msg.Teams[0].ObjectivesServed) != 1 {
		t.Fatalf("list lost metadata: %+v", listed.Msg)
	}
	// Metadata does not enable execution; an explicit boolean update remains
	// separately supported even when protobuf omits its false default.
	if err := teamStore.Update(ctx, "aquila", &store.Team{Enabled: true, EnabledSet: true}); err != nil {
		t.Fatal(err)
	}
	active, err := client.UpdateTeam(ctx, connect.NewRequest(&teamsv1.UpdateTeamRequest{
		Id: "aquila", Team: &teamsv1.TeamInput{Purpose: "supervision"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"purpose"}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !active.Msg.Enabled || active.Msg.Lifetime != "standing" || active.Msg.Purpose != "supervision" {
		t.Fatalf("metadata update changed scheduling eligibility: %+v", active.Msg)
	}
	cleared, err := client.UpdateTeam(ctx, connect.NewRequest(&teamsv1.UpdateTeamRequest{
		Id: "aquila", Team: &teamsv1.TeamInput{},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"purpose", "lifetime", "effort_refs", "enabled"}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Msg.Purpose != "" || cleared.Msg.Lifetime != "" || len(cleared.Msg.EffortRefs) != 0 || cleared.Msg.Enabled {
		t.Fatalf("explicit clears lost at Connect boundary: %+v", cleared.Msg)
	}
	_, err = client.UpdateTeam(ctx, connect.NewRequest(&teamsv1.UpdateTeamRequest{
		Id: "aquila", Team: &teamsv1.TeamInput{Purpose: "free-to-act"}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"purpose"}},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid purpose: %v", err)
	}
	got, err := client.GetTeam(ctx, connect.NewRequest(&teamsv1.GetTeamRequest{Id: "aquila"}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.Purpose != "" || len(got.Msg.ObjectivesServed) != 1 {
		t.Fatalf("invalid mutation persisted or objectives lost: %+v", got.Msg)
	}
	stored, err = os.ReadFile(teamPath)
	if err != nil {
		t.Fatal(err)
	}
	var after map[string]json.RawMessage
	if err := json.Unmarshal(stored, &after); err != nil {
		t.Fatal(err)
	}
	var objective []map[string]any
	if err := json.Unmarshal(after["objectivesServed"], &objective); err != nil {
		t.Fatal(err)
	}
	if objective[0]["unknown"] != "retained" || len(after["instrument"]) == 0 {
		t.Fatal("Connect update erased adjacent-owner records")
	}
	_, err = client.DeleteTeam(ctx, connect.NewRequest(&teamsv1.DeleteTeamRequest{Id: "aquila"}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetTeam(ctx, connect.NewRequest(&teamsv1.GetTeamRequest{Id: "aquila"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("deleted team still readable: %v", err)
	}
}
