package main

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/trailers"
	"github.com/vrooli/api-core/discovery"
	agent "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	plans "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	"github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans/plans_v1connect"
	swarm "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	swarmconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

// Owner reads establish existence, never authorship or acceptance. Failures do
// not erase trailer values. No owner response is cached across user requests.
type trailerOwners struct {
	resolveURL func(context.Context, string) (string, error)
	httpClient *http.Client
}

func newTrailerOwners() *trailerOwners {
	return &trailerOwners{resolveURL: discovery.ResolveScenarioURLDefault, httpClient: &http.Client{Timeout: time.Second}}
}

func (r *trailerOwners) Resolve(ctx context.Context, key, value string) (string, trailers.ResolutionStatus, string, error) {
	kind := trailers.KindOf(key, value)
	owner := ""
	switch kind {
	case "effort", "epoch", "backlog":
		owner = "swarm-manager"
	case "run":
		owner = "agent-manager"
	case "plan":
		owner = "plan-manager"
	default:
		return "", trailers.Unresolved, "no authoritative owner registered for this work kind", nil
	}
	url, err := r.resolveURL(ctx, owner)
	if err != nil {
		return "", trailers.Unresolved, owner + " is unavailable", nil
	}
	detail := ""
	switch kind {
	case "run":
		out, e := apiconnect.NewAgentManagerServiceClient(r.httpClient, url).GetRun(ctx, ownerReadRequest(ctx, &agent.GetRunRequest{RunId: value}))
		err = e
		if e == nil && out.Msg.GetRun() != nil {
			detail = "run exists in Agent Manager"
		}
	case "plan":
		id, _, _ := strings.Cut(value, "#")
		out, e := plans_v1connect.NewPlansServiceClient(r.httpClient, url).GetPlan(ctx, ownerReadRequest(ctx, &plans.GetPlanRequest{Id: id}))
		err = e
		if e == nil && out.Msg.GetPlan() != nil {
			detail = "plan: " + out.Msg.GetPlan().GetTitle()
		}
	case "backlog":
		kind, name, ok := strings.Cut(value, "/")
		if !ok {
			return "", trailers.Unresolved, "invalid backlog reference", nil
		}
		out, e := swarmconnect.NewBacklogServiceClient(r.httpClient, url).GetItem(ctx, ownerReadRequest(ctx, &swarm.GetBacklogItemRequest{Kind: kind, Name: name}))
		err = e
		if e == nil && out.Msg.GetItem() != nil {
			detail = "backlog: " + out.Msg.GetItem().GetTitle()
		}
	case "effort", "epoch":
		id, epoch, _ := strings.Cut(value, "#")
		id, revision, _ := strings.Cut(id, "@")
		out, e := swarmconnect.NewDevelopmentServiceClient(r.httpClient, url).GetDevelopment(ctx, ownerReadRequest(ctx, &swarm.GetDevelopmentRequest{EffortId: id}))
		err = e
		if e == nil && out.Msg.GetDevelopment() != nil {
			development := out.Msg.GetDevelopment()
			if revision != "" && revision != strconv.FormatInt(development.GetReference().GetRevision(), 10) {
				return "", trailers.Unresolved, "owner current revision differs; historical revision is unverified", nil
			}
			if kind == "epoch" {
				found := false
				for _, artifact := range development.GetArtifacts() {
					if path.Base(artifact.GetRelativePath()) == epoch+".md" {
						found = true
						break
					}
				}
				if !found {
					return "", trailers.Unresolved, "effort exists; epoch artifact is not in the owner manifest", nil
				}
			}
			detail = fmt.Sprintf("%s exists in Swarm Manager", kind)
		}
	}
	if err != nil {
		switch connect.CodeOf(err) {
		case connect.CodePermissionDenied, connect.CodeUnauthenticated:
			return "", trailers.Inaccessible, "", nil
		case connect.CodeNotFound:
			return "", trailers.Unresolved, "owner could not find this reference; deletion is unconfirmed", nil
		default:
			return "", trailers.Unresolved, owner + " evidence unavailable", nil
		}
	}
	if detail == "" {
		return "", trailers.Unresolved, "owner returned no reference evidence", nil
	}
	return value, trailers.Resolved, detail, nil
}

// One bounded read budget per response; at most eight owner reads in flight.
// Duplicate metadata resolves once, while output preserves its original order.
func boundedTrailerReferences(ctx context.Context, entries []trailers.Entry, resolver trailers.Resolver) []trailers.Reference {
	ctx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()
	result := make([]trailers.Reference, len(entries))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < min(8, len(entries)); worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				refs := trailers.ResolveEntries(ctx, entries[index:index+1], resolver)
				if len(refs) > 0 {
					result[index] = refs[0]
				}
			}
		}()
	}
	unique := map[string]int{}
	for index, entry := range entries {
		if entry.Key == "" {
			continue
		}
		key := entry.Key + "\x00" + entry.Value
		if _, exists := unique[key]; !exists {
			unique[key] = index
			jobs <- index
		}
	}
	close(jobs)
	wg.Wait()
	refs := []trailers.Reference{}
	for _, entry := range entries {
		if entry.Key != "" {
			refs = append(refs, result[unique[entry.Key+"\x00"+entry.Value]])
		}
	}
	return refs
}

// Credentials remain request-scoped and are revalidated by each owning API.
// Caller identity claims and mutation authority are never manufactured here.
type ownerReadHeadersKey struct{}

func ownerReadInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			headers := make(http.Header)
			for _, name := range []string{"Authorization", "Cookie", "Cf-Access-Jwt-Assertion"} {
				if value := request.Header().Get(name); value != "" {
					headers.Set(name, value)
				}
			}
			return next(context.WithValue(ctx, ownerReadHeadersKey{}, headers), request)
		}
	})
}
func ownerReadRequest[T any](ctx context.Context, message *T) *connect.Request[T] {
	request := connect.NewRequest(message)
	if headers, ok := ctx.Value(ownerReadHeadersKey{}).(http.Header); ok {
		for key, values := range headers {
			request.Header()[key] = append([]string(nil), values...)
		}
	}
	return request
}
