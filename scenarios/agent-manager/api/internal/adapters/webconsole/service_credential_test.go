package webconsole

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"
)

type ownerKey struct{}

// verifyingSessions models Web Console's contract: the stable-create owner is
// the verified caller, and an unverified caller gets none. The fixture
// "verifies" exactly one bearer value; real verification lives in api-core.
type verifyingSessions struct {
	sessionsv1connect.UnimplementedSessionsServiceHandler
	mu          sync.Mutex
	createCalls int
	sawAgentHdr bool
}

func ownerFrom(ctx context.Context) string { owner, _ := ctx.Value(ownerKey{}).(string); return owner }

func (v *verifyingSessions) List(ctx context.Context, _ *connect.Request[sessionsv1.ListRequest]) (*connect.Response[sessionsv1.ListResponse], error) {
	return connect.NewResponse(&sessionsv1.ListResponse{SupportsStableCreateIdentity: true, StableCreateOwner: ownerFrom(ctx)}), nil
}

func (v *verifyingSessions) Create(ctx context.Context, req *connect.Request[sessionsv1.CreateRequest]) (*connect.Response[sessionsv1.CreateResponse], error) {
	v.mu.Lock()
	v.createCalls++
	v.mu.Unlock()
	raw, _ := proto.MarshalOptions{Deterministic: true}.Marshal(req.Msg)
	digest := sha256.Sum256(raw)
	fp := hex.EncodeToString(digest[:])
	return connect.NewResponse(&sessionsv1.CreateResponse{Session: &sessionsv1.Session{Id: req.Msg.RequestedSessionId, Owner: ownerFrom(ctx), CreateFingerprint: fp}}), nil
}

func newVerifyingServer(t *testing.T, sessions *verifyingSessions) string {
	t.Helper()
	path, handler := sessionsv1connect.NewSessionsServiceHandler(sessions)
	mux := http.NewServeMux()
	mux.Handle(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Identity-Token") != "" {
			sessions.mu.Lock()
			sessions.sawAgentHdr = true
			sessions.mu.Unlock()
		}
		ctx := r.Context()
		if r.Header.Get("Authorization") == "Bearer service-token" {
			ctx = context.WithValue(ctx, ownerKey{}, "agent-manager")
		}
		handler.ServeHTTP(w, r.WithContext(ctx))
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestServicePrincipalClientObtainsStableCreateOwner(t *testing.T) {
	sessions := &verifyingSessions{}
	base := newVerifyingServer(t, sessions)
	exchanges := 0
	source := NewServiceTokenSource(func() (string, error) { return "vsp1.agent-manager.fixture", nil },
		func(_ context.Context, credential string) (string, time.Time, error) {
			exchanges++
			if credential != "vsp1.agent-manager.fixture" {
				return "", time.Time{}, errors.New("wrong credential")
			}
			return "service-token", time.Now().Add(time.Hour), nil
		}, nil)
	client := NewClient(base, NewServicePrincipalHTTPClient(source))

	// Agent Manager speaks as its service principal; no agent identity header is sent.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id := "6f2c8e8e-5d0b-4b8a-9f62-6d0e8f1d2a11"
	got, err := client.CreateSession(ctx, CreateSessionParams{RequestedSessionID: id, LaunchCommand: "true"})
	if err != nil {
		t.Fatalf("authenticated stable create refused: %v", err)
	}
	if got != id || sessions.createCalls != 1 {
		t.Fatalf("id=%q createCalls=%d", got, sessions.createCalls)
	}
	if _, err := client.CreateSession(ctx, CreateSessionParams{RequestedSessionID: "0b9f7d0e-1c1a-4f0e-8a51-2a7f1a4d5e22", LaunchCommand: "true"}); err != nil {
		t.Fatal(err)
	}
	if exchanges != 1 {
		t.Fatalf("token not cached: %d exchanges", exchanges)
	}
	if sessions.sawAgentHdr {
		t.Fatal("agent identity header forwarded to web-console")
	}
}

func TestUnverifiedClientStillRefusesBeforeCreate(t *testing.T) {
	for name, source := range map[string]*ServiceTokenSource{
		"credential missing": NewServiceTokenSource(func() (string, error) { return "", errors.New("unconfigured") },
			func(context.Context, string) (string, time.Time, error) {
				return "service-token", time.Now().Add(time.Hour), nil
			}, nil),
		"exchange refused": NewServiceTokenSource(func() (string, error) { return "vsp1.agent-manager.revoked", nil },
			func(context.Context, string) (string, time.Time, error) {
				return "", time.Time{}, errors.New("unauthenticated")
			}, nil),
		"token not verified by web-console": NewServiceTokenSource(func() (string, error) { return "vsp1.agent-manager.x", nil },
			func(context.Context, string) (string, time.Time, error) {
				return "forged-token", time.Now().Add(time.Hour), nil
			}, nil),
	} {
		t.Run(name, func(t *testing.T) {
			sessions := &verifyingSessions{}
			client := NewClient(newVerifyingServer(t, sessions), NewServicePrincipalHTTPClient(source))
			_, err := client.CreateSession(context.Background(), CreateSessionParams{RequestedSessionID: "6f2c8e8e-5d0b-4b8a-9f62-6d0e8f1d2a11", LaunchCommand: "true"})
			if err == nil || !strings.Contains(err.Error(), "stable-create identity unsupported; no Create submitted") {
				t.Fatalf("err=%v", err)
			}
			if sessions.createCalls != 0 {
				t.Fatal("Create submitted without a verified stable-create owner")
			}
			// Non-stable reads still work anonymously.
			if _, err := client.ListSessions(context.Background()); err != nil {
				t.Fatalf("anonymous list failed: %v", err)
			}
		})
	}
}

func TestServiceTokenSourceRefreshesBeforeExpiry(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	calls := 0
	source := NewServiceTokenSource(func() (string, error) { return "c", nil },
		func(context.Context, string) (string, time.Time, error) {
			calls++
			return "t" + strings.Repeat("x", calls), now.Add(10 * time.Minute), nil
		}, clock)
	first, err := source.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(7 * time.Minute)
	if same, _ := source.Token(context.Background()); same != first || calls != 1 {
		t.Fatal("token refreshed too early")
	}
	now = now.Add(2 * time.Minute) // within the refresh margin
	if renewed, _ := source.Token(context.Background()); renewed == first || calls != 2 {
		t.Fatal("token not refreshed inside the margin")
	}
}
