package sandbox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type lifecycleStatusFixture struct {
	commonconnect.UnimplementedLifecycleMaintenanceServiceHandler
	standing *commonv1.LifecycleStanding
	err      error
}

func (f lifecycleStatusFixture) Status(context.Context, *connect.Request[commonv1.LifecycleStatusRequest]) (*connect.Response[commonv1.LifecycleStatusResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing: f.standing}), nil
}

func TestWorkspaceSandboxProviderReadinessUsesOwnerStatusConservatively(t *testing.T) {
	now := time.Date(2026, 11, 13, 15, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		standing *commonv1.LifecycleStanding
		want     string
		known    bool
	}{
		{
			name: "open",
			standing: &commonv1.LifecycleStanding{
				Provider: "workspace-sandbox", Scenario: "workspace-sandbox",
				Phase:      commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN,
				ObservedAt: timestamppb.New(now),
			},
			want: "open", known: true,
		},
		{
			name: "closed maintenance",
			standing: &commonv1.LifecycleStanding{
				Provider: "workspace-sandbox", Scenario: "workspace-sandbox", AdmissionClosed: true,
				Phase:      commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_DRAINED,
				ObservedAt: timestamppb.New(now),
			},
			want: "closed", known: true,
		},
		{
			name: "foreign owner",
			standing: &commonv1.LifecycleStanding{
				Provider: "agent-manager", Scenario: "agent-manager",
				Phase:      commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN,
				ObservedAt: timestamppb.New(now),
			},
			want: "unknown",
		},
		{
			name: "missing observation time",
			standing: &commonv1.LifecycleStanding{
				Provider: "workspace-sandbox", Scenario: "workspace-sandbox",
				Phase: commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN,
			},
			want: "unknown",
		},
		{
			name: "inconsistent open phase",
			standing: &commonv1.LifecycleStanding{
				Provider: "workspace-sandbox", Scenario: "workspace-sandbox", AdmissionClosed: true,
				Phase:      commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN,
				ObservedAt: timestamppb.New(now),
			},
			want: "closed", known: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, handler := commonconnect.NewLifecycleMaintenanceServiceHandler(lifecycleStatusFixture{standing: tt.standing})
			mux := http.NewServeMux()
			mux.Handle(path, handler)
			server := httptest.NewServer(mux)
			defer server.Close()

			got := NewWorkspaceSandboxProvider(server.URL).ReadProviderReadiness(t.Context())
			if string(got.State) != tt.want || got.Known != tt.known {
				t.Fatalf("readiness = %+v, want state=%q known=%t", got, tt.want, tt.known)
			}
			if tt.known && (got.ObservedAt == nil || !got.ObservedAt.Equal(now)) {
				t.Fatalf("observed_at = %v, want %v", got.ObservedAt, now)
			}
			if !tt.known && got.Message != "Workspace Sandbox readiness could not be confirmed." {
				t.Fatalf("unknown message is not safe/stable: %q", got.Message)
			}
		})
	}
}

func TestWorkspaceSandboxProviderReadinessTransportFailureIsUnknown(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	got := NewWorkspaceSandboxProvider(server.URL).ReadProviderReadiness(t.Context())
	if got.Known || got.State != "unknown" || got.ObservedAt != nil {
		t.Fatalf("transport failure readiness = %+v", got)
	}
}
