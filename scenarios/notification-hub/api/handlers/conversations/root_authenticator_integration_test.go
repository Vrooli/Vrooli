package conversations

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	health "notification-hub/handlers/health"
	conversation "notification-hub/internal/conversations"
	localdb "notification-hub/internal/database"
	delivery "notification-hub/internal/delivery"
	"notification-hub/internal/hub"
	"notification-hub/internal/identity"
	integration "notification-hub/internal/integrationconfig"
	notification "notification-hub/internal/notifications"
	preference "notification-hub/internal/preferences"
	recipient "notification-hub/internal/recipients"
	routing "notification-hub/internal/routing"
	"notification-hub/internal/testutil/hubtest"
)

type authenticatorFixture struct {
	URL, Account string
	Recovery     []string
}
type fixtureResolver string

func (r fixtureResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return string(r), nil
}

// This test runs a real authenticator in a child test process because Go's
// internal-package boundaries prevent either scenario importing the other.
// Both services use temporary SQLite databases and only test identities.
func TestRootEnrollmentAndDecisionWithAuthenticatorMFA(t *testing.T) {
	if testing.Short() {
		t.Skip("cross-scenario integration")
	}
	fixturePath := filepath.Join(t.TempDir(), "auth.json")
	cmd := exec.Command("go", "test", "./handlers/mfa", "-run", "^TestHubFixture$", "-count=1", "-timeout=2m")
	cmd.Dir = "../../../../scenario-authenticator/api"
	cmd.Env = append(os.Environ(), "HUB_AUTH_FIXTURE_FILE="+fixturePath)
	var logs bytes.Buffer
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(fixturePath)
		if err := cmd.Wait(); err != nil {
			t.Errorf("authenticator fixture: %v: %s", err, logs.String())
		}
	})
	var f authenticatorFixture
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(fixturePath)
		if err == nil && json.Unmarshal(b, &f) == nil && f.URL != "" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if f.URL == "" || len(f.Recovery) < 3 {
		t.Fatal("authenticator fixture did not start")
	}
	ctx := context.Background()
	probe, probeErr := (identity.Forwarder{Resolver: fixtureResolver(f.URL)}).VerifyDecision(ctx, "owner@example.test", "Passw0rd", "", "", "", strings.Repeat("a", 64))
	if probeErr != nil || !probe.MFARequired {
		t.Fatalf("authenticator probe: %+v %v", probe, probeErr)
	}
	db := databasetest.NewSQLite(t)
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(health.Schema), database.SchemaProviderFunc(integration.Schema), database.SchemaProviderFunc(recipient.Schema), database.SchemaProviderFunc(notification.Schema), database.SchemaProviderFunc(preference.Schema), database.SchemaProviderFunc(routing.Schema), database.SchemaProviderFunc(delivery.Schema), database.SchemaProviderFunc(conversation.Schema)); err != nil {
		t.Fatal(err)
	}
	svc := hubtest.New(t, database.NewFromPrimary(db), nil, nil)
	svc.SetDefaultRecipient("owner")
	svc.SetAppEligibility(func(context.Context, string) (string, error) { return "https://hub.example", nil })
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[{"source":"cloudflare_access","subject":"cf-test","canonical_subject":"owner","scopes":["root:rebaseline"]}]`)
	h := &handler{service: svc, stepUp: identity.Forwarder{Resolver: fixtureResolver(f.URL)}}
	owner := identity.OwnerAuthenticator{Providers: []authn.Provider{enrollmentProvider{"cf-test"}}, OwnerSubject: func(context.Context) string { return "owner" }}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/identity/root-enrollment", owner.Middleware(http.HandlerFunc(h.rootEnrollment)))
	mux.HandleFunc("/api/v1/identity/root-decision-read", h.rootDecisionRead)
	path, handler := connectv1.NewConversationsServiceHandler(h)
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	post := func(path string, payload map[string]string) (int, map[string]any) {
		t.Helper()
		body, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, server.URL+path, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://hub.example")
		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		var result map[string]any
		_ = json.Unmarshal(b, &result)
		if bytes.Contains(b, []byte("Passw0rd")) {
			t.Fatal("response disclosed password")
		}
		return resp.StatusCode, result
	}
	credentials := func() map[string]string {
		return map[string]string{"email": "owner@example.test", "password": "Passw0rd"}
	}
	status, challenge := post("/api/v1/identity/root-enrollment", credentials())
	if status != 200 || challenge["mfaRequired"] != true {
		t.Fatalf("enrollment challenge: %d %+v", status, challenge)
	}
	c := credentials()
	c["mfaChallenge"], _ = challenge["mfaChallenge"].(string)
	c["recoveryCode"] = f.Recovery[0]
	if status, result := post("/api/v1/identity/root-enrollment", c); status != 200 || result["mfaRequired"] != false {
		t.Fatalf("enrollment: %d %+v", status, result)
	}
	if status, _ := post("/api/v1/identity/root-enrollment", c); status == 200 {
		t.Fatal("replayed enrollment challenge accepted")
	}
	ask, _, err := svc.AskWithSpec(ctx, hub.AskSpec{Recipient: "owner", Title: "Rebaseline", Body: "Review", Question: "Approve?", Options: []hub.AskOption{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}}, Deadline: time.Now().Add(time.Hour), DeadlineSpecified: true, SensitivityLabel: "private", Source: "root-rebaseline", SourceEventType: "root.rebaseline.decision_requested.v1", IdempotencyKey: "isolated-integration", ProviderDecisionContext: &hub.ProviderDecisionContext{SchemaVersion: 1, Scope: hub.ProviderDecisionScopeRootRebaseline, CanonicalTermsSHA256: strings.Repeat("a", 64)}})
	if err != nil {
		t.Fatal(err)
	}
	c = credentials()
	c["askId"] = ask.ID
	status, challenge = post("/api/v1/identity/root-decision-read", c)
	if status != 200 || challenge["mfaRequired"] != true {
		t.Fatalf("read challenge: %d %+v", status, challenge)
	}
	c["mfaChallenge"], _ = challenge["mfaChallenge"].(string)
	c["recoveryCode"] = f.Recovery[1]
	if status, result := post("/api/v1/identity/root-decision-read", c); status != 200 || result["mfaRequired"] != false {
		t.Fatalf("decision read: %d %+v", status, result)
	} else if readAsk, ok := result["ask"].(map[string]any); !ok || readAsk["id"] != ask.ID {
		t.Fatalf("decision read returned wrong ask: %+v", result)
	}
	client := connectv1.NewConversationsServiceClient(server.Client(), server.URL)
	request := &v1.AnswerRootWithPasswordRequest{AskId: ask.ID, Answer: "approve", Email: "owner@example.test", Password: "Passw0rd", RevisionCount: int32(ask.RevisionCount)}
	first, err := client.AnswerRootWithPassword(ctx, connect.NewRequest(request))
	if err != nil || !first.Msg.GetMfaRequired() {
		t.Fatalf("answer challenge: %v", err)
	}
	request.MfaChallenge = first.Msg.GetMfaChallenge()
	request.RecoveryCode = f.Recovery[2]
	answered, err := client.AnswerRootWithPassword(ctx, connect.NewRequest(request))
	if err != nil || answered.Msg.GetAnsweredAt() == "" {
		t.Fatalf("answer: %v", err)
	}
	if _, err := client.AnswerRootWithPassword(ctx, connect.NewRequest(request)); err == nil {
		t.Fatal("replayed answer accepted")
	}
	stored, err := svc.GetAsk(ctx, ask.ID)
	if err != nil || stored.State != hub.AskAnswered {
		t.Fatalf("persisted answer: %+v %v", stored, err)
	}
}
