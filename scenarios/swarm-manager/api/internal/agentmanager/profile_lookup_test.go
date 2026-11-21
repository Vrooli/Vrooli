package agentmanager

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type profileListHandler struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	profiles  []*domainpb.AgentProfile
	listCalls int
}

func (h *profileListHandler) ListProfiles(_ context.Context, r *connect.Request[api.ListProfilesRequest]) (*connect.Response[api.ListProfilesResponse], error) {
	h.listCalls++
	limit, offset := int(r.Msg.GetLimit()), int(r.Msg.GetOffset())
	if limit == 0 {
		limit = 50
	}
	if offset > len(h.profiles) {
		offset = len(h.profiles)
	}
	end := offset + limit
	if end > len(h.profiles) {
		end = len(h.profiles)
	}
	items := append([]*domainpb.AgentProfile(nil), h.profiles[offset:end]...)
	return connect.NewResponse(&api.ListProfilesResponse{Profiles: items, Total: int32(len(h.profiles)), HasMore: end < len(h.profiles)}), nil
}

func profileServer(t *testing.T, handler *profileListHandler) *httptest.Server {
	t.Helper()
	path, connectHandler := apiconnect.NewAgentManagerServiceHandler(handler)
	mux := http.NewServeMux()
	mux.Handle(path, connectHandler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func ownerProfile(id, key, owner string) *domainpb.AgentProfile {
	return &domainpb.AgentProfile{Id: id, Name: key, ProfileKey: key, OwnerScenario: owner}
}

func TestInitializeResolvesExistingOwnerProfileWithoutDeclarationMutation(t *testing.T) {
	h := &profileListHandler{profiles: []*domainpb.AgentProfile{
		ownerProfile("default-id", "swarm-manager/default", "swarm-manager"),
		ownerProfile("analysis-id", "swarm-manager/analysis", "swarm-manager"),
	}}
	server := profileServer(t, h)
	svc := NewAgentService(AgentServiceConfig{ProfileKey: "swarm-manager/default", Enabled: true, Timeout: time.Second})
	svc.client = NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	if err := svc.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if got := svc.GetProfileID(); got != "default-id" {
		t.Fatalf("profile ID=%q", got)
	}
	if _, err := svc.profileRefFor("swarm-manager/analysis"); err != nil {
		t.Fatalf("same-owner profile not retained: %v", err)
	}
	if h.listCalls != 1 {
		t.Fatalf("ListProfiles calls=%d, want one read", h.listCalls)
	}
}

func TestInitializeDoesNotAcceptForeignOwnerOrCreateMissingProfile(t *testing.T) {
	h := &profileListHandler{profiles: []*domainpb.AgentProfile{ownerProfile("foreign-id", "swarm-manager/default", "another-scenario")}}
	server := profileServer(t, h)
	svc := NewAgentService(AgentServiceConfig{ProfileKey: "swarm-manager/default", Enabled: true, Timeout: time.Second})
	svc.client = NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	err := svc.Initialize(context.Background())
	if err == nil || !strings.Contains(err.Error(), "required profile") || !strings.Contains(err.Error(), "declaration owner") {
		t.Fatalf("Initialize error=%v", err)
	}
	if h.listCalls != 1 {
		t.Fatalf("ListProfiles calls=%d", h.listCalls)
	}
}

func TestInitializeRejectsBlankIDForOwnedProfile(t *testing.T) {
	h := &profileListHandler{profiles: []*domainpb.AgentProfile{ownerProfile("", "swarm-manager/default", "swarm-manager")}}
	server := profileServer(t, h)
	svc := NewAgentService(AgentServiceConfig{ProfileKey: "swarm-manager/default", Enabled: true, Timeout: time.Second})
	svc.client = NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	if err := svc.Initialize(context.Background()); err == nil || !strings.Contains(err.Error(), "invalid profile identity") {
		t.Fatalf("Initialize error=%v", err)
	}
}

func TestInitializeWarnsWhenResolvedOwnerProfileHasDeclarationDrift(t *testing.T) {
	p := ownerProfile("default-id", "swarm-manager/default", "swarm-manager")
	p.SourcePath = ".vrooli/agent-manager/default.json"
	p.SourceHash = "declared"
	p.LastAppliedHash = "applied"
	p.LocalOverride = true
	h := &profileListHandler{profiles: []*domainpb.AgentProfile{p}}
	server := profileServer(t, h)
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	defer slog.SetDefault(previous)
	svc := NewAgentService(AgentServiceConfig{ProfileKey: "swarm-manager/default", Enabled: true, Timeout: time.Second})
	svc.client = NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	if err := svc.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if !strings.Contains(output.String(), "agent-manager profile has declaration drift metadata") || !strings.Contains(output.String(), "swarm-manager/default") {
		t.Fatalf("drift diagnostic missing: %s", output.String())
	}
}

func TestInitializeAcceptsRequiredOwnerMatchWithinBoundedPageCoverage(t *testing.T) {
	const pageSize = 128
	const pageCount = 8
	profiles := make([]*domainpb.AgentProfile, pageSize*pageCount)
	profiles[0] = ownerProfile("default-id", "swarm-manager/default", "swarm-manager")
	for i := 1; i < len(profiles); i++ {
		profiles[i] = ownerProfile(fmt.Sprintf("id-%d", i), fmt.Sprintf("swarm-manager/profile-%d", i), "swarm-manager")
	}
	h := &profileListHandler{profiles: profiles}
	server := profileServer(t, h)
	svc := NewAgentService(AgentServiceConfig{ProfileKey: "swarm-manager/default", Enabled: true, Timeout: time.Second})
	svc.client = NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	if err := svc.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if got := svc.GetProfileID(); got != "default-id" {
		t.Fatalf("profile ID=%q", got)
	}
	if _, err := svc.profileRefFor("swarm-manager/profile-1024"); err == nil {
		t.Fatal("profile outside observed page bound was admitted")
	}
}
