package teams

import (
	"context"
	"encoding/json"
	"errors"
	repocontract "github.com/vrooli/repo-contract-go"
	heartbeatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	heartbeatmount "prompt-manager/handlers/heartbeat"
	heartbeatdomain "prompt-manager/internal/heartbeat"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"prompt-manager/internal/requestpolicy"
	"prompt-manager/internal/store"
	domainteams "prompt-manager/internal/teams"
	"prompt-manager/internal/testutil/fixtures"
)

func TestCreateTeamConnectUsesExactOwnerGrant(t *testing.T) {
	for _, tc := range []struct {
		name      string
		operation string
		target    string
		wantAllow bool
	}{
		{name: "exact team.manage target", operation: "team.manage", target: "prompt-manager:team/commissioned", wantAllow: true},
		{name: "unrelated operation", operation: "agent.manage", target: "agent-manager:*"},
		{name: "wrong team target", operation: "team.manage", target: "prompt-manager:team/another"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			relations := store.NewFileRelationStore(root)
			teamStore := store.NewFileTeamStore(root, root, relations)
			handler := NewHandlers(teamStore, store.NewFileAgentStore(root), relations, nil, nil)
			path, mount := NewConnectMount(handler)
			mux := http.NewServeMux()
			mux.Handle(path, mount)
			principal := identity.Principal{
				Kind: identity.ActorAgent, Subject: "commissioning-run", Owner: "human-owner",
				Source: identity.SourceAgentProvenance, Verified: true,
				Scopes: []string{"prompt-manager:read", "prompt-manager:write"},
				Grant: &identity.OperationGrant{
					ID: "reviewed-team-management", Policy: "team-management", Approver: "human-owner",
					Operations: []string{tc.operation}, Targets: []string{tc.target}, ExpiresAt: time.Now().Add(time.Hour),
				},
			}
			boundary := requestpolicy.PreparationBoundary(mux)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				boundary.ServeHTTP(w, r.WithContext(identity.WithPrincipal(r.Context(), principal)))
			}))
			defer server.Close()
			client := teamsconnect.NewTeamsServiceClient(server.Client(), server.URL)

			team := fixtures.IndependentTeam("commissioned", "Commissioned Team", fixtures.WithEnabled(false))
			team.Purpose = "delivery"
			team.Lifetime = "finite"
			team.EffortRefs = []string{"effort:commissioning-test"}
			raw, err := json.Marshal(team)
			if err != nil {
				t.Fatal(err)
			}
			input := &teamsv1.TeamInput{}
			if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, input); err != nil {
				t.Fatal(err)
			}
			created, err := client.CreateTeam(context.Background(), connect.NewRequest(&teamsv1.CreateTeamRequest{Team: input}))
			if tc.wantAllow {
				if err != nil {
					t.Fatalf("exact owner grant was refused: %v", err)
				}
				if created.Msg.Id != "commissioned" {
					t.Fatalf("created team ID = %q, want commissioned", created.Msg.Id)
				}
				stored, err := teamStore.Get(context.Background(), "commissioned")
				if err != nil {
					t.Fatal(err)
				}
				if stored.CreationProof == nil || stored.CreationProof.Subject != "commissioning-run" {
					t.Fatalf("verified creator was not recorded: proof=%+v", stored.CreationProof)
				}
				return
			}
			if err == nil {
				t.Fatal("owner refused grant unexpectedly created a team")
			}
			if got := connect.CodeOf(err); got != connect.CodePermissionDenied {
				t.Fatalf("request did not reach the team owner check: Connect code=%s err=%v", got, err)
			}
			if _, statErr := os.Stat(filepath.Join(root, "teams", "commissioned", "team.json")); !os.IsNotExist(statErr) {
				t.Fatalf("refused create left team state behind: stat err=%v", statErr)
			}
		})
	}
}

func TestGetTeamConnectUsesExactTeamManagementGrantWithoutCrossRuleTargetMixing(t *testing.T) {
	root := t.TempDir()
	relations := store.NewFileRelationStore(root)
	teamStore := store.NewFileTeamStore(root, root, relations)
	if err := teamStore.Create(context.Background(), fixtures.IndependentTeam("commissioned", "Commissioned Team")); err != nil {
		t.Fatalf("seed team: %v", err)
	}
	handler := NewHandlers(teamStore, store.NewFileAgentStore(root), relations, nil, nil)
	path, mount := NewConnectMount(handler)
	mux := http.NewServeMux()
	mux.Handle(path, mount)
	boundary := requestpolicy.PreparationBoundary(mux)
	principal := identity.Principal{
		Kind: identity.ActorAgent, Subject: "commissioning-run", Owner: "human-owner",
		Source: identity.SourceAgentProvenance, Verified: true,
		Scopes: []string{"prompt-manager:read", "prompt-manager:write"},
		Grant: &identity.OperationGrant{
			ID: "combined-team-and-agent-management", Approver: "human-owner", ExpiresAt: time.Now().Add(time.Hour),
			Rules: []identity.OperationRule{
				{Policy: "team-management", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/commissioned"}},
				{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"prompt-manager:team/another"}},
			},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		boundary.ServeHTTP(w, r.WithContext(identity.WithPrincipal(r.Context(), principal)))
	}))
	defer server.Close()
	client := teamsconnect.NewTeamsServiceClient(server.Client(), server.URL)

	got, err := client.GetTeam(context.Background(), connect.NewRequest(&teamsv1.GetTeamRequest{Id: "commissioned"}))
	if err != nil {
		t.Fatalf("exact team.manage target could not read its team: %v", err)
	}
	if got.Msg.Id != "commissioned" {
		t.Fatalf("read team ID = %q, want commissioned", got.Msg.Id)
	}
	_, err = client.GetTeam(context.Background(), connect.NewRequest(&teamsv1.GetTeamRequest{Id: "another"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-rule target returned %v, want permission denied", err)
	}
}

func TestGetRolesConnectUsesExactTeamManagementGrant(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	relations := store.NewFileRelationStore(root)
	teamStore := store.NewFileTeamStore(root, root, relations)
	if err := teamStore.Create(ctx, fixtures.IndependentTeam("commissioned", "Commissioned Team")); err != nil {
		t.Fatalf("seed team: %v", err)
	}
	if err := teamStore.SetRoles(ctx, "commissioned", &store.TeamRoles{TeamID: "commissioned", Roles: []store.Role{{ID: "lead", Name: "Lead"}}}); err != nil {
		t.Fatalf("seed roles: %v", err)
	}
	legacy := domainteams.NewHandlers(teamStore, store.NewFileAgentStore(root), relations, nil, nil)
	path, mount := NewConnectMount(legacy)
	mux := http.NewServeMux()
	mux.Handle(path, mount)
	boundary := requestpolicy.PreparationBoundary(mux)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal := identity.Principal{
			Kind: identity.ActorAgent, Subject: "commissioning-run", Owner: "human-owner",
			Source: identity.SourceAgentProvenance, Verified: true,
			Scopes: []string{"prompt-manager:read", "prompt-manager:write"},
			Grant: &identity.OperationGrant{
				ID: "reviewed-team-management", Approver: "human-owner", ExpiresAt: time.Now().Add(time.Hour),
				Rules: []identity.OperationRule{{Policy: "team-management", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/commissioned"}}},
			},
		}
		boundary.ServeHTTP(w, r.WithContext(identity.WithPrincipal(r.Context(), principal)))
	}))
	defer server.Close()
	client := teamsconnect.NewTeamsServiceClient(server.Client(), server.URL)

	got, err := client.GetRoles(ctx, connect.NewRequest(&teamsv1.GetRolesRequest{TeamId: "commissioned"}))
	if err != nil {
		t.Fatalf("exact team.manage target could not read roles: %v", err)
	}
	if len(got.Msg.Roles) != 1 || got.Msg.Roles[0].GetId() != "lead" {
		t.Fatalf("roles response = %+v, want seeded role", got.Msg.Roles)
	}
	_, err = client.GetRoles(ctx, connect.NewRequest(&teamsv1.GetRolesRequest{TeamId: "another"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-team roles read returned %v, want permission denied", err)
	}
}

func TestSetRolesConnectRequiresExactTeamManagementTarget(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	relations := store.NewFileRelationStore(root)
	teamStore := store.NewFileTeamStore(root, root, relations)
	if err := teamStore.Create(ctx, fixtures.IndependentTeam("commissioned", "Commissioned Team")); err != nil {
		t.Fatalf("seed team: %v", err)
	}
	legacy := domainteams.NewHandlers(teamStore, store.NewFileAgentStore(root), relations, nil, nil)
	path, mount := NewConnectMount(legacy)
	mux := http.NewServeMux()
	mux.Handle(path, mount)
	boundary := requestpolicy.PreparationBoundary(mux)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal := identity.Principal{
			Kind: identity.ActorAgent, Subject: "commissioning-run", Owner: "human-owner",
			Source: identity.SourceAgentProvenance, Verified: true,
			Scopes: []string{"prompt-manager:read", "prompt-manager:write"},
			Grant: &identity.OperationGrant{
				ID: "reviewed-team-management", Approver: "human-owner", ExpiresAt: time.Now().Add(time.Hour),
				Rules: []identity.OperationRule{{Policy: "team-management", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/other"}}},
			},
		}
		boundary.ServeHTTP(w, r.WithContext(identity.WithPrincipal(r.Context(), principal)))
	}))
	defer server.Close()
	client := teamsconnect.NewTeamsServiceClient(server.Client(), server.URL)
	_, err := client.SetRoles(ctx, connect.NewRequest(&teamsv1.SetRolesRequest{
		TeamId: "commissioned", Roles: []*teamsv1.Role{{Id: "lead", Name: "Lead"}},
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("wrong target SetRoles error=%v, want permission_denied", err)
	}
	roles, err := teamStore.GetRoles(ctx, "commissioned")
	if err != nil {
		t.Fatal(err)
	}
	if roles != nil && len(roles.Roles) != 0 {
		t.Fatalf("refused SetRoles changed persisted roles: %+v", roles)
	}
}

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

// The compiled native CLI crosses its actual Connect bridge and metadata store
// in an isolated runtime. It does not qualify production middleware authority,
// adopted live builds or a reviewer task/run admission.
func TestNativeCLIConfiguresDisabledOnDemandReviewerThroughConnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	root := t.TempDir()
	relations := store.NewFileRelationStore(root)
	teams := store.NewFileTeamStore(root, root, relations)
	agents := store.NewFileAgentStore(root)
	if err := teams.Create(ctx, fixtures.IndependentTeam("delivery", "Delivery", fixtures.WithEnabled(false))); err != nil {
		t.Fatal(err)
	}
	if err := agents.Create(ctx, &store.Agent{ID: "reviewer", DisplayName: "Reviewer"}); err != nil {
		t.Fatal(err)
	}
	if err := relations.SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "delivery", AgentID: "reviewer", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	scheduler := heartbeatdomain.NewScheduler(nil, nil, teams, nil)
	domainHandler := heartbeatdomain.NewHandlers(heartbeatdomain.HandlersDeps{TeamStore: teams, AgentStore: agents, RelationStore: relations, Scheduler: scheduler})
	path, mount := heartbeatmount.NewConnectMount(domainHandler)
	mux := http.NewServeMux()
	mux.Handle(path, mount)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	repo, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "prompt-manager")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	build.Dir = filepath.Join(repo, "scenarios", "prompt-manager", "cli")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v %s", err, output)
	}
	command := exec.CommandContext(ctx, binary, "--api-base", server.URL, "team", "heartbeat-disable", "delivery", "reviewer", "--on-demand", "--profile=prompt-manager/heartbeat-inspection", "--json")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("native CLI: %v %s", err, output)
	}
	var response heartbeatdomain.HeartbeatConfigResponse
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatalf("CLI readback: %v %s", err, output)
	}
	if response.Enabled || response.Schedule != "" || response.Scheduled || response.SchedulerRegistered || response.ProfileKey != "prompt-manager/heartbeat-inspection" {
		t.Fatalf("reviewer timer appeared: %+v", response)
	}
	cfg, err := teams.GetHeartbeatConfig(ctx, "delivery", "reviewer")
	if err != nil || cfg == nil || cfg.Enabled || cfg.Schedule != "" {
		t.Fatalf("owner runtime: %+v %v", cfg, err)
	}
	if err := teams.ValidateHeartbeatRoster(ctx, "delivery"); err != nil {
		t.Fatal(err)
	}
	client := heartbeatconnect.NewHeartbeatServiceClient(server.Client(), server.URL)
	readback, err := client.GetHeartbeat(ctx, connect.NewRequest(&heartbeatv1.MemberRequest{TeamId: "delivery", AgentId: "reviewer"}))
	if err != nil {
		t.Fatal(err)
	}
	got := readback.Msg.GetData().GetStructValue().AsMap()
	if got["enabled"] != false || got["schedule"] != "" || got["profileKey"] != "prompt-manager/heartbeat-inspection" {
		t.Fatalf("Connect owner readback changed: %v", got)
	}
	stateJSON, ok := got["schedulerState"].(map[string]any)
	if !ok || stateJSON["running"] != false || stateJSON["registered"] != float64(0) {
		t.Fatalf("scheduler evidence absent: %v", got)
	}
	if _, present := stateJSON["last_callback"]; present {
		t.Fatalf("unobserved callback acquired a timestamp: %v", stateJSON)
	}

	if scheduler.State().Registered != 0 || scheduler.State().Running || scheduler.State().LastCallback != nil {
		t.Fatalf("on-demand role started scheduler: %+v", scheduler.State())
	}
}

func TestSendMessageConnectBindsOwnerRunAndMapsRefusal(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	relations := store.NewFileRelationStore(root)
	teamStore := store.NewFileTeamStore(root, root, relations)
	if err := teamStore.Create(ctx, fixtures.IndependentTeam("team-1", "Test Team")); err != nil {
		t.Fatal(err)
	}
	for _, member := range []string{"sender-1", "recipient-1"} {
		if err := relations.SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "team-1", AgentID: member, Status: store.MemberStatusActive}); err != nil {
			t.Fatal(err)
		}
	}
	legacy := domainteams.NewHandlers(teamStore, store.NewFileAgentStore(root), relations, nil, nil)
	legacy.SetActiveRunIDLookup(func(teamID, agentID string) (string, bool) {
		return "current-run", teamID == "team-1" && agentID == "sender-1"
	})
	_, mount := NewConnectMount(legacy)
	runID := "stale-run"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller := provenance.NewContext(r.Context(), provenance.Provenance{
			Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: runID,
		})
		mount.ServeHTTP(w, r.WithContext(caller))
	}))
	defer server.Close()
	client := teamsconnect.NewTeamsServiceClient(server.Client(), server.URL)
	send := func() (*connect.Response[teamsv1.Message], error) {
		return client.SendMessage(ctx, connect.NewRequest(&teamsv1.SendMessageRequest{
			TeamId: "team-1", AgentId: "recipient-1", FromAgentId: "sender-1", Content: "status",
		}))
	}
	_, err := send()
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodePermissionDenied {
		t.Fatalf("stale SendMessage error=%v, want permission_denied", err)
	}
	inbox, err := teamStore.GetInbox(ctx, "team-1", "recipient-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(inbox.Messages) != 0 {
		t.Fatalf("refused sender appended %d message(s)", len(inbox.Messages))
	}

	runID = "current-run"
	response, err := send()
	if err != nil {
		t.Fatalf("exact registered owner run SendMessage: %v", err)
	}
	if response.Msg.GetFromAgentId() != "sender-1" || response.Msg.GetToAgentId() != "recipient-1" {
		t.Fatalf("message attribution = %+v", response.Msg)
	}
	inbox, err = teamStore.GetInbox(ctx, "team-1", "recipient-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(inbox.Messages) != 1 {
		t.Fatalf("exact owner run appended %d messages, want 1", len(inbox.Messages))
	}
}
