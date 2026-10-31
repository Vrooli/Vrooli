package appctx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"connectrpc.com/connect"
	heartbeatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestDecodeIgnoresNilResultAndEmptyBody(t *testing.T) {
	if err := decode([]byte(`{"ok":true}`), nil); err != nil {
		t.Fatalf("nil result should not decode: %v", err)
	}
	var result map[string]bool
	if err := decode(nil, &result); err != nil {
		t.Fatalf("empty body should not decode: %v", err)
	}
}

func TestDecodeUnmarshalsJSON(t *testing.T) {
	var result struct {
		OK bool `json:"ok"`
	}
	if err := decode([]byte(`{"ok":true}`), &result); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !result.OK {
		t.Fatal("expected decoded true value")
	}
}

func TestRPCBodyUnwrapsAnEmptyRepeatedFieldAsJSONArray(t *testing.T) {
	body, handled, err := rpcBody(connect.NewResponse(&skillsv1.ListSkillVariantsResponse{}), "variants", true, nil)
	if err != nil {
		t.Fatalf("unwrap empty variants: %v", err)
	}
	if !handled || string(body) != "[]" {
		t.Fatalf("body=%s handled=%t", body, handled)
	}
}

func TestNormalizeInt64JSONFieldRestoresLegacyNumber(t *testing.T) {
	body, err := normalizeInt64JSONField([]byte(`{"durationMs":"42","status":"dry-run"}`), "durationMs")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"durationMs":42,"status":"dry-run"}` {
		t.Fatalf("body=%s", body)
	}
}

func TestRenameJSONFieldPreservesLegacyDryRunShape(t *testing.T) {
	body, err := renameJSONField([]byte(`{"dryRun":true,"plan":{}}`), "dryRun", "dry_run")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"dry_run":true,"plan":{}}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestCallTeamsSetRolesUsesTypedConnectRequestAndResponse(t *testing.T) {
	requests := make(chan *teamsv1.SetRolesRequest, 1)
	serverErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != teamsconnect.TeamsServiceSetRolesProcedure {
			serverErrors <- fmt.Errorf("path=%q, want %q", r.URL.Path, teamsconnect.TeamsServiceSetRolesProcedure)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			serverErrors <- err
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		request := &teamsv1.SetRolesRequest{}
		if err := proto.Unmarshal(body, request); err != nil {
			serverErrors <- err
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests <- request
		response, err := proto.Marshal(&teamsv1.GetRolesResponse{Roles: []*teamsv1.Role{{Id: "lead", Name: "Lead"}}})
		if err != nil {
			serverErrors <- err
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/proto")
		_, _ = w.Write(response)
	}))
	defer server.Close()
	client := teamsconnect.NewTeamsServiceClient(server.Client(), server.URL)
	body, handled, err := callTeams(context.Background(), client, "PUT", []string{"teams", "team-1", "roles"}, nil, map[string]any{
		"roles": []map[string]string{{"id": "lead", "name": "Lead"}},
	})
	if err != nil {
		t.Fatalf("callTeams: %v", err)
	}
	select {
	case serverErr := <-serverErrors:
		t.Fatal(serverErr)
	default:
	}
	if !handled {
		t.Fatal("SetRoles request was not handled by Connect adapter")
	}
	request := <-requests
	if request.GetTeamId() != "team-1" || len(request.GetRoles()) != 1 || request.GetRoles()[0].GetId() != "lead" || request.GetRoles()[0].GetName() != "Lead" {
		t.Fatalf("typed request=%+v", request)
	}
	var decoded []map[string]string
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode response body %s: %v", body, err)
	}
	if len(decoded) != 1 || decoded[0]["id"] != "lead" || decoded[0]["name"] != "Lead" {
		t.Fatalf("response body=%s", body)
	}
}

func TestCallHeartbeatAttemptsUsesTypedQueryAndPreservesResponse(t *testing.T) {
	requests := make(chan *heartbeatv1.QueryRequest, 1)
	serverErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != heartbeatconnect.HeartbeatServiceListHeartbeatAttemptsProcedure {
			serverErrors <- fmt.Errorf("path=%q, want %q", r.URL.Path, heartbeatconnect.HeartbeatServiceListHeartbeatAttemptsProcedure)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			serverErrors <- err
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		request := &heartbeatv1.QueryRequest{}
		if err := proto.Unmarshal(body, request); err != nil {
			serverErrors <- err
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests <- request
		data, err := structpb.NewStruct(map[string]any{
			"attempts": []any{map[string]any{"id": "attempt-1", "status": "failed", "extra": "preserved"}},
			"total":    3, "hasMore": true,
		})
		if err != nil {
			serverErrors <- err
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		response, err := proto.Marshal(&heartbeatv1.JsonResponse{Data: structpb.NewStructValue(data)})
		if err != nil {
			serverErrors <- err
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/proto")
		_, _ = w.Write(response)
	}))
	defer server.Close()
	client := heartbeatconnect.NewHeartbeatServiceClient(server.Client(), server.URL)
	body, handled, err := callHeartbeat(context.Background(), client, "GET", []string{"heartbeat-attempts"}, url.Values{
		"team_id": {"rcl-delivery"}, "agent_id": {"orchestrator"}, "limit": {"7"},
	}, nil)
	if err != nil {
		t.Fatalf("callHeartbeat: %v", err)
	}
	select {
	case serverErr := <-serverErrors:
		t.Fatal(serverErr)
	default:
	}
	if !handled {
		t.Fatal("ListHeartbeatAttempts was not handled by Connect adapter")
	}
	request := <-requests
	want := map[string]string{"team_id": "rcl-delivery", "agent_id": "orchestrator", "limit": "7"}
	if !reflect.DeepEqual(request.GetQuery(), want) {
		t.Fatalf("typed query = %#v, want %#v", request.GetQuery(), want)
	}
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode response body %s: %v", body, err)
	}
	attempts, ok := response["attempts"].([]any)
	if !ok || len(attempts) != 1 || attempts[0].(map[string]any)["extra"] != "preserved" || response["total"] != float64(3) || response["hasMore"] != true {
		t.Fatalf("response body=%s", body)
	}
}
