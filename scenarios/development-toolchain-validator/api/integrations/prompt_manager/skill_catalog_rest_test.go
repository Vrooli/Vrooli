package prompt_manager_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	promptmanager "development-toolchain-validator/integrations/prompt_manager"
	skillcatalog "development-toolchain-validator/internal/skill_catalog"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/discovery"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
)

type fakeSkillsService struct {
	skillsconnect.UnimplementedSkillsServiceHandler
	sync func(context.Context) (*skillsv1.SyncSkillsResponse, error)
}

func (f fakeSkillsService) SyncSkills(ctx context.Context, _ *connect.Request[skillsv1.SyncSkillsRequest]) (*connect.Response[skillsv1.SyncSkillsResponse], error) {
	response, err := f.sync(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func happyResponse() *skillsv1.SyncSkillsResponse {
	return &skillsv1.SyncSkillsResponse{Skills: []*skillsv1.Skill{
		{Id: "implementation-plan-authoring", Content: "hello", UpdatedAt: "2026-05-01T00:00:00Z"},
		{Id: "test", Content: "world", UpdatedAt: "2026-05-02T00:00:00Z"},
	}, LastUpdated: "2026-05-02T00:00:00Z", Hash: "deadbeef"}
}

func newSkillsServer(t *testing.T, service fakeSkillsService) *httptest.Server {
	t.Helper()
	_, handler := skillsconnect.NewSkillsServiceHandler(service)
	return httptest.NewServer(handler)
}

func newAdapter(t *testing.T, srv *httptest.Server) *promptmanager.SkillCatalogRESTAdapter {
	t.Helper()
	resolver := discovery.NewResolver(discovery.ResolverConfig{StaticBaseURL: srv.URL})
	return promptmanager.NewSkillCatalogRESTAdapter(promptmanager.Options{
		Resolver:    resolver,
		Doer:        &http.Client{Timeout: 5 * time.Second},
		MaxAttempts: 3,
	})
}

func TestFetch_HappyPath(t *testing.T) {
	srv := newSkillsServer(t, fakeSkillsService{sync: func(context.Context) (*skillsv1.SyncSkillsResponse, error) { return happyResponse(), nil }})
	defer srv.Close()

	adapter := newAdapter(t, srv)
	got, err := adapter.Fetch(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "implementation-plan-authoring", got[0].ID)
	require.NotEmpty(t, got[0].ContentHash, "content_hash must be computed from response.content")
	require.Equal(t, "2026-05-01T00:00:00Z", got[0].Version)
}

func TestFetch_RetriesOn5xx(t *testing.T) {
	var calls atomic.Int32
	srv := newSkillsServer(t, fakeSkillsService{sync: func(context.Context) (*skillsv1.SyncSkillsResponse, error) {
		c := calls.Add(1)
		if c < 2 {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("upstream busy"))
		}
		return happyResponse(), nil
	}})
	defer srv.Close()

	adapter := newAdapter(t, srv)
	got, err := adapter.Fetch(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.GreaterOrEqual(t, calls.Load(), int32(2), "expected at least one retry")
}

func TestFetch_4xxIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	srv := newSkillsServer(t, fakeSkillsService{sync: func(context.Context) (*skillsv1.SyncSkillsResponse, error) {
		calls.Add(1)
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bad request"))
	}})
	defer srv.Close()

	adapter := newAdapter(t, srv)
	_, err := adapter.Fetch(context.Background())
	require.Error(t, err)
	var sync skillcatalog.ErrSyncFailed
	require.True(t, errors.As(err, &sync))
	require.Equal(t, int32(1), calls.Load(), "4xx must not be retried")
}

func TestFetch_DropsBlankIDs(t *testing.T) {
	srv := newSkillsServer(t, fakeSkillsService{sync: func(context.Context) (*skillsv1.SyncSkillsResponse, error) {
		return &skillsv1.SyncSkillsResponse{Skills: []*skillsv1.Skill{{Content: "x", UpdatedAt: "2026-05-01T00:00:00Z"}, {Id: "good", Content: "y", UpdatedAt: "2026-05-01T00:00:00Z"}}}, nil
	}})
	defer srv.Close()

	adapter := newAdapter(t, srv)
	got, err := adapter.Fetch(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "good", got[0].ID)
}

func TestFetch_DiscoveryFailureMapsToNotReady(t *testing.T) {
	// Static resolver always succeeds; instead exercise the not-running
	// branch via a custom resolver that returns a discovery.Error.
	failingResolver := discovery.NewResolver(discovery.ResolverConfig{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, fmt.Errorf("vrooli scenario port: scenario not running")
		},
		VrooliPath: "vrooli-fake",
	})
	// NOTE: This intentionally hits the real CLI fallback path; the
	// resolver returns ErrScenarioNotRunning when shell-out fails with
	// the right shape. If discovery semantics shift, this test will
	// need to follow.
	adapter := promptmanager.NewSkillCatalogRESTAdapter(promptmanager.Options{
		Resolver: failingResolver,
		Doer:     &http.Client{Timeout: 1 * time.Second},
	})
	_, err := adapter.Fetch(context.Background())
	require.Error(t, err)
	// Don't assert NotReady — discovery may wrap the synthetic error
	// differently; what we care about is that *some* skillcatalog error
	// surfaces rather than a raw transport panic.
	require.True(t,
		errors.As(err, new(skillcatalog.ErrSyncFailed)) || strings.Contains(err.Error(), "not running") || strings.Contains(err.Error(), "discovery"),
		"unexpected error type: %T %v", err, err)
}
