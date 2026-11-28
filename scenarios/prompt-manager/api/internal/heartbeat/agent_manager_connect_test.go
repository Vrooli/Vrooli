package heartbeat

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestResolveProfileUsesGeneratedConnectProcedure(t *testing.T) {
	const profileID = "11111111-2222-4333-8444-555555555555"
	responseBody, err := protojson.Marshal(&api.ListProfilesResponse{Profiles: []*domain.AgentProfile{{
		Id: profileID, Name: "Heartbeat", ProfileKey: "prompt-manager-heartbeat",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != apiconnect.AgentManagerServiceListProfilesProcedure {
			t.Errorf("request = %s %s, want generated ListProfiles procedure", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content type = %q, want Connect ProtoJSON", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(responseBody)
	}))
	defer server.Close()

	profile, err := newTestClient(t, server).ResolveProfile(context.Background(), "prompt-manager-heartbeat")
	if err != nil {
		t.Fatal(err)
	}
	if profile.GetId() != profileID || profile.GetProfileKey() != "prompt-manager-heartbeat" {
		t.Fatalf("resolved profile = %+v", profile)
	}
}

func TestConnectCreateMutationsMapTypedApprovalToDefinitiveOutcome(t *testing.T) {
	const approvalDetail = "operator approval requested; the operation proceeds once approved"
	approvalError := func() error {
		connectErr := connect.NewError(connect.CodeFailedPrecondition, errors.New("approval pending"))
		detail, err := connect.NewErrorDetail(&api.AuthorizationApprovalPending{
			Subject: "prompt-manager",
			Detail:  approvalDetail,
		})
		if err != nil {
			t.Fatalf("add typed approval detail: %v", err)
		}
		connectErr.AddDetail(detail)
		return connectErr
	}

	t.Run("CreateTask", func(t *testing.T) {
		server := httptest.NewServer(connect.NewUnaryHandler(
			apiconnect.AgentManagerServiceCreateTaskProcedure,
			func(context.Context, *connect.Request[api.CreateTaskRequest]) (*connect.Response[api.CreateTaskResponse], error) {
				return nil, approvalError()
			},
		))
		defer server.Close()

		_, err := newTestClient(t, server).CreateTask(context.Background(), &Task{Title: "heartbeat", ScopePath: "/workspace"})
		var pending *ApprovalPendingError
		if !errors.As(err, &pending) || pending.Stage != "creating_task" || pending.Error() == "" {
			t.Fatalf("CreateTask error = %v, want typed approval pending", err)
		}
		if pending.Cause.Error() != approvalDetail || IsDispatchUncertain(err) {
			t.Fatalf("CreateTask approval outcome = %v, want definitive typed detail", err)
		}
	})

	t.Run("CreateRun", func(t *testing.T) {
		server := httptest.NewServer(connect.NewUnaryHandler(
			apiconnect.AgentManagerServiceCreateRunProcedure,
			func(context.Context, *connect.Request[api.CreateRunRequest]) (*connect.Response[api.CreateRunResponse], error) {
				return nil, approvalError()
			},
		))
		defer server.Close()

		_, err := newTestClient(t, server).CreateRun(
			withCreateRunCaller(context.Background(), "Bearer owner-proof", ""),
			&CreateRunRequest{TaskID: "11111111-2222-4333-8444-555555555555", IdempotencyKey: "run-key"},
		)
		var pending *ApprovalPendingError
		if !errors.As(err, &pending) || pending.Stage != "creating_run" || pending.Cause.Error() != approvalDetail || IsDispatchUncertain(err) {
			t.Fatalf("CreateRun error = %v, want definitive typed approval pending", err)
		}
	})
}
