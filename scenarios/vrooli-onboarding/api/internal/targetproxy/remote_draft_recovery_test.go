package targetproxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/vrooli/internal/operatorstate"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session"
	sessionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session/sessionv1connect"
	sessionhandler "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/session"
	sessiondomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/session"
)

type bridgeSessionReacher struct {
	endpoint string
	offline  bool
}

func (r *bridgeSessionReacher) CallScenario(ctx context.Context, request nodereach.ScenarioRequest) ([]byte, error) {
	if r.offline {
		return nil, &nodereach.Error{Kind: nodereach.ErrNodeUnavailable, Node: request.NodeID, Err: errors.New("bridge channel offline")}
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint+request.Procedure, bytes.NewReader(request.Body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/proto")
	httpRequest.Header.Set("Accept", "application/proto")
	response, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, errors.New(response.Status)
	}
	return body, nil
}

func verifiedPrincipal(source identity.AuthSource) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			return next(identity.WithPrincipal(ctx, identity.Principal{
				Kind: identity.ActorHuman, Subject: "node-operator", Verified: true,
				Source: source, Sources: []identity.AuthSource{source},
			}), request)
		}
	})
}

func TestRemoteTargetDraftSurvivesBridgeLossOnOwningNode(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", "..", "..", ".."))
	store := operatorstate.New(operatorstate.Config{
		RepoRoot: repoRoot, StateRoot: filepath.Join(t.TempDir(), "state"),
		Now: func() time.Time { return time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC) },
	})
	service := sessiondomain.Service{
		GetDraftFn: func(ctx context.Context, target, actor string) (sessiondomain.Draft, error) {
			document, err := store.Load(ctx)
			if err != nil {
				return sessiondomain.Draft{}, err
			}
			return draftDomain(document.Drafts[operatorstate.DraftKey(target, actor)]), nil
		},
		SaveDraftFn: func(ctx context.Context, target, actor, expectedRevision, baseRevision, stepID string, choices map[string]string) (sessiondomain.Draft, error) {
			document, err := store.SaveDraft(ctx, target, actor, expectedRevision, baseRevision, stepID, choices)
			if err != nil {
				return sessiondomain.Draft{}, err
			}
			return draftDomain(document.Drafts[operatorstate.DraftKey(target, actor)]), nil
		},
	}
	remoteHandler := verifiedSessionHandler(service, nil, identity.SourcePersonalLocal)
	remote := httptest.NewServer(remoteHandler)
	defer remote.Close()
	reacher := &bridgeSessionReacher{endpoint: remote.URL}
	front := httptest.NewServer(verifiedSessionHandler(sessiondomain.Service{}, reacher, identity.SourceScenarioAuthenticator))
	defer front.Close()
	client := sessionconnect.NewSessionServiceClient(http.DefaultClient, front.URL)

	saveRequest := &sessionv1.SaveDraftRequest{
		Target: "node-1", BaseRevision: "setup-5", StepId: "core-set",
		Choices: map[string]string{"scenario.alpha": "enabled", "region": "north"},
	}
	save, err := client.SaveDraft(context.Background(), connect.NewRequest(saveRequest))
	if err != nil {
		t.Fatalf("save remote draft: %v", err)
	}
	if save.Msg.GetDraft().GetStepId() != "core-set" || save.Msg.GetDraft().GetChoices()["region"] != "north" {
		t.Fatalf("saved draft = %+v", save.Msg.GetDraft())
	}
	if save.Msg.GetDraft().GetTarget() != "node-1" || saveRequest.GetTarget() != "node-1" {
		t.Fatalf("caller target changed: response=%q request=%q", save.Msg.GetDraft().GetTarget(), saveRequest.GetTarget())
	}
	stored, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load target owner state: %v", err)
	}
	ownerDraft := stored.Drafts[operatorstate.DraftKey("local", "personal_local:node-operator")]
	if ownerDraft.StepID != "core-set" || ownerDraft.Choices["region"] != "north" {
		t.Fatalf("target owner persisted draft = %+v", ownerDraft)
	}

	reacher.offline = true
	_, err = client.GetDraft(context.Background(), connect.NewRequest(&sessionv1.GetDraftRequest{Target: "node-1"}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("bridge loss error = %v, want unavailable", err)
	}
	reacher.offline = false
	readback, err := client.GetDraft(context.Background(), connect.NewRequest(&sessionv1.GetDraftRequest{Target: "node-1"}))
	if err != nil {
		t.Fatalf("read remote draft after bridge recovery: %v", err)
	}
	if got := readback.Msg.GetDraft(); got.GetStepId() != "core-set" || got.GetChoices()["scenario.alpha"] != "enabled" || got.GetChoices()["region"] != "north" {
		t.Fatalf("draft after bridge recovery = %+v", got)
	}
}

func verifiedSessionHandler(service sessiondomain.Service, reacher Reacher, source identity.AuthSource) http.Handler {
	_, handler := sessionconnect.NewSessionServiceHandler(sessionhandler.NewConnectHandler(service), connect.WithInterceptors(
		verifiedPrincipal(source),
		Interceptor(reacher),
	))
	return handler
}

func draftDomain(draft operatorstate.Draft) sessiondomain.Draft {
	return sessiondomain.Draft{Target: draft.Target, Actor: draft.Actor, BaseRevision: draft.BaseRevision, Revision: draft.Revision, StepID: draft.StepID, Choices: draft.Choices, UpdatedAt: draft.UpdatedAt}
}
