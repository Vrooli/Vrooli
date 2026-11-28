package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/trailers"
	swarm "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	swarmconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

type trailerDevelopmentOwner struct {
	swarmconnect.UnimplementedDevelopmentServiceHandler
	calls atomic.Int32
}

func (o *trailerDevelopmentOwner) GetDevelopment(_ context.Context, req *connect.Request[swarm.GetDevelopmentRequest]) (*connect.Response[swarm.GetDevelopmentResponse], error) {
	o.calls.Add(1)
	if req.Header().Get("Authorization") != "Bearer fixture-reader" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("owner requires authentication"))
	}
	return connect.NewResponse(&swarm.GetDevelopmentResponse{Development: &swarm.DevelopmentView{Reference: &swarm.DevelopmentReference{EffortId: req.Msg.GetEffortId(), Revision: 3}, Artifacts: []*swarm.DevelopmentArtifact{{RelativePath: "epochs/E3.md"}}}}), nil
}

func TestTrailerOwnerEvidenceIsTypedRevisionAwareAndPrivateSafe(t *testing.T) {
	owner := &trailerDevelopmentOwner{}
	path, handler := swarmconnect.NewDevelopmentServiceHandler(owner)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	resolver := &trailerOwners{resolveURL: func(context.Context, string) (string, error) { return server.URL, nil }, httpClient: server.Client()}
	ctx := context.WithValue(context.Background(), ownerReadHeadersKey{}, http.Header{"Authorization": []string{"Bearer fixture-reader"}})
	for _, test := range []struct {
		key, value string
		want       trailers.ResolutionStatus
	}{
		{"Vrooli-Effort", "effort:example@3", trailers.Resolved},
		{"Vrooli-Effort", "effort:example@2", trailers.Unresolved},
		{"Vrooli-Epoch", "effort:example#E3", trailers.Resolved},
		{"Vrooli-Epoch", "effort:example#E4", trailers.Unresolved},
	} {
		_, status, _, err := resolver.Resolve(ctx, test.key, test.value)
		if err != nil || status != test.want {
			t.Fatalf("%s: status=%s err=%v", test.value, status, err)
		}
	}
	refs := trailers.ResolveEntries(context.Background(), []trailers.Entry{{Key: "Vrooli-Effort", Value: "effort:example", Supported: true}}, resolver)
	if len(refs) != 1 || refs[0].Status != trailers.Inaccessible || refs[0].OwnerID != "" {
		t.Fatalf("private reference leaked: %+v", refs)
	}
}

type slowTrailerOwner struct{ calls atomic.Int32 }

func (o *slowTrailerOwner) Resolve(ctx context.Context, _, _ string) (string, trailers.ResolutionStatus, string, error) {
	o.calls.Add(1)
	<-ctx.Done()
	return "", trailers.Unresolved, "bounded owner timeout", nil
}
func TestTrailerReadBudgetAndDuplicateResolution(t *testing.T) {
	owner := &slowTrailerOwner{}
	entries := []trailers.Entry{}
	for i := 0; i < 50; i++ {
		entries = append(entries, trailers.Entry{Key: "Vrooli-Effort", Value: "effort:example", Supported: true})
	}
	start := time.Now()
	refs := boundedTrailerReferences(context.Background(), entries, owner)
	if len(refs) != 50 || owner.calls.Load() != 1 || time.Since(start) > 2*time.Second {
		t.Fatalf("unbounded/duplicate owner reads: calls=%d refs=%d", owner.calls.Load(), len(refs))
	}
}
