package inference

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"ai-gateway/internal/providers"
	providermocks "ai-gateway/internal/providers/mocks"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
)

func newTestAttemptLedger(t *testing.T) *SQLAttemptLedger {
	t.Helper()
	db := dbtest.NewSQLite(t)
	provider := apidb.SchemaProviderFunc(AttemptSchema)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, provider))
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, provider))
	return NewSQLAttemptLedger(db)
}

func hostedTestRepository(t *testing.T, serverURL string, key func() string, ledger AttemptLedger, candidates ...Candidate) *ResourceRepository {
	t.Helper()
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"classify.fast": {Candidates: candidates},
	}}
	adapters := []providers.Adapter{{
		Provider: providers.ProviderMetered, Locality: "remote",
		Door: &providers.HostedDoorClient{BaseURL: serverURL, ResolveDeviceKey: func(context.Context) (string, error) { return key(), nil }},
	}}
	repository, err := NewResourceRepository(catalog, adapters, ledger)
	require.NoError(t, err)
	return repository
}

func TestResourceRepositoryRecoversExactAttemptRejectsChangedDigestAndFailsClosedOnRotation(t *testing.T) {
	const root = "4e4e5be93eae40108d011c8a2ff391d4"
	key := "operator-test-device-key-generation-one"
	var mu sync.Mutex
	posts, receipts, models := 0, 0, 0
	var postKeys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/receipt":
			mu.Lock()
			receipts++
			mu.Unlock()
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"status":"unknown"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/models":
			mu.Lock()
			models++
			mu.Unlock()
			_, _ = fmt.Fprint(w, `{"data":[{"id":"test-model"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/ai/chat/completions":
			mu.Lock()
			posts++
			postKeys = append(postKeys, r.Header.Get("Idempotency-Key"))
			mu.Unlock()
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ledger := newTestAttemptLedger(t)
	getKey := func() string { return key }
	runner := &providermocks.FakeRunner{DefaultOK: true, Default: providers.Result{Stdout: `{"response":"\"fallback\""}`}}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"classify.fast": {Candidates: []Candidate{
			{Provider: providers.ProviderMetered, ResourceRole: "metered", Reason: "hosted"},
			{Provider: providers.ProviderOllama, ResourceRole: "chat.default", Reason: "local fallback"},
		}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{
		{Provider: providers.ProviderMetered, Locality: "remote", Door: &providers.HostedDoorClient{BaseURL: server.URL, ResolveDeviceKey: func(context.Context) (string, error) { return getKey(), nil }}},
		{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner},
	}, ledger)
	require.NoError(t, err)
	request := ProviderRequest{RootOperationID: root, Source: "source", SchemaJSON: `{"type":"string"}`, Role: "classify.fast", Profile: sharedv1.Profile_PROFILE_REMOTE_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC}
	first, err := repository.Run(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, first.Recovery)
	require.Equal(t, inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_UNCERTAIN, first.Recovery.GetState())
	require.Equal(t, 1, posts)
	require.Len(t, postKeys, 1)
	require.Empty(t, runner.Commands, "uncertainty must stop candidate fallback")

	// The same logical root resolves the saved child receipt and never posts again.
	second, err := repository.Run(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, second.Recovery)
	require.Equal(t, first.Recovery.GetAttemptReference(), second.Recovery.GetAttemptReference())
	require.Equal(t, 1, posts)
	require.Equal(t, 1, models, "recovery must not fetch models or re-enter provider selection")

	changed := request
	changed.Source = "different content"
	_, err = repository.Run(context.Background(), changed)
	require.ErrorIs(t, err, ErrRootConflict)
	require.Equal(t, 1, posts)

	key = "operator-test-device-key-generation-two"
	_, err = repository.Run(context.Background(), request)
	require.ErrorIs(t, err, ErrRootConflict)
	require.Equal(t, 1, posts, "rotation must fail closed before any redispatch")
	mu.Lock()
	require.Equal(t, 3, receipts)
	mu.Unlock()
}

func TestResourceRepositoryPreDispatchFailureKeepsFallbackEligible(t *testing.T) {
	const root = "fd5b841818b04a129989b131dfc9dafe"
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/ai/receipt":
			w.WriteHeader(http.StatusNotFound)
		case "/api/v1/ai/models":
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		case "/api/v1/ai/chat/completions":
			posts++
			_, _ = fmt.Fprint(w, `{"choices":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	runner := &providermocks.FakeRunner{
		Results: map[string]providers.Result{
			"resource-ollama policy resolve --role chat.default --json": {Stdout: `{"role":"chat.default","model":"local-test"}`},
		},
		DefaultOK: true,
		Default:   providers.Result{Stdout: `{"response":"\"local-result\""}`},
	}
	configureQualifiedLocalRole(runner, "chat.default", "local-test", "local")
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"classify.fast": {Candidates: []Candidate{
			{Provider: providers.ProviderMetered, ResourceRole: "metered", Reason: "hosted"},
			{Provider: providers.ProviderOllama, ResourceRole: "chat.default", Reason: "local"},
		}},
	}}
	repository, err := NewResourceRepositoryWithRightsContextSource(catalog, []providers.Adapter{
		{Provider: providers.ProviderMetered, Locality: "remote", Door: &providers.HostedDoorClient{BaseURL: server.URL, ResolveDeviceKey: func(context.Context) (string, error) { return "test-device-key", nil }}},
		{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner},
	}, fixtureRightsContext, newTestAttemptLedger(t))
	require.NoError(t, err)
	result, err := repository.Run(context.Background(), ProviderRequest{
		RootOperationID: root, Source: "source", SchemaJSON: `{"type":"string"}`, Role: "classify.fast", PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC,
	})
	require.NoError(t, err)
	require.Equal(t, providers.ProviderOllama, result.Provider)
	require.Equal(t, `"local-result"`, result.ValueJSON)
	require.Zero(t, posts, "a failed models lookup is proven pre-dispatch")
}

func TestServiceSchemaRepairCreatesDistinctHostedAttempt(t *testing.T) {
	const root = "b135874d957e4e55a6ac4ceb11672b15"
	const deviceKey = "test-device-for-schema-repair"
	var posts int
	var postKeys []string
	posted := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/receipt":
			if posted[r.Header.Get("Idempotency-Key")] {
				_, _ = fmt.Fprint(w, `{"status":"settled","charged_credits":9,"private_provider_receipt":{"output":"PRIVATE"}}`)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/models":
			_, _ = fmt.Fprint(w, `{"data":[{"id":"test-model"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/ai/chat/completions":
			posts++
			postKeys = append(postKeys, r.Header.Get("Idempotency-Key"))
			posted[r.Header.Get("Idempotency-Key")] = true
			content := `{"kind":"wrong"}`
			if posts == 2 {
				content = `{"kind":"ok"}`
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\ndata: [DONE]\n\n", content)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	repository := hostedTestRepository(t, server.URL, func() string { return deviceKey }, newTestAttemptLedger(t), Candidate{Provider: providers.ProviderMetered, ResourceRole: "metered", Reason: "hosted"})
	service := NewService(repository)
	request := ProviderRequest{
		RootOperationID: root, Source: "source",
		SchemaJSON: `{"type":"object","required":["kind"],"properties":{"kind":{"enum":["ok"]}}}`,
		Role:       "classify.fast", PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC,
	}
	response := service.Run(context.Background(), request)
	require.True(t, response.GetValidated(), "one locally rejected response gets one new repair attempt")
	require.Equal(t, 2, posts)
	require.Len(t, postKeys, 2)
	require.NotEmpty(t, postKeys[0])
	require.NotEqual(t, postKeys[0], postKeys[1])

	// A retry after delivery to Gateway returns only the accounting receipt.
	replayed := service.Run(context.Background(), request)
	require.Equal(t, inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_OUTPUT_UNAVAILABLE, replayed.GetRecovery().GetState())
	require.Equal(t, "settled", replayed.GetRecovery().GetAccountingStatus())
	require.EqualValues(t, 9, replayed.GetRecovery().GetChargedCredits())
	require.Empty(t, replayed.GetValueJson())
	require.Empty(t, replayed.GetProvider())
	require.Empty(t, replayed.GetModel())
	encoded, err := protojson.Marshal(replayed)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "PRIVATE")
	require.NotContains(t, string(encoded), "private_provider_receipt")
	require.Equal(t, 2, posts)
}

type failAtSetAttemptLedger struct {
	AttemptLedger
	setCalls int
	failAt   int
}

func (l *failAtSetAttemptLedger) Set(ctx context.Context, root string, ordinal int, state AttemptState, status string, credits *int64) error {
	l.setCalls++
	if l.setCalls == l.failAt {
		return fmt.Errorf("injected attempt-ledger state update failure")
	}
	return l.AttemptLedger.Set(ctx, root, ordinal, state, status, credits)
}

func TestResourceRepositoryReceiptRecoveryRemainsExactKeyWhenLedgerStateWriteFails(t *testing.T) {
	const root = "a1790c22216f4b6dbf2470a9343e0185"
	receiptReads, modelReads, providerPosts := 0, 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/receipt":
			receiptReads++
			if receiptReads <= 2 {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = fmt.Fprint(w, `{"status":"settled","charged_credits":7}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/models":
			modelReads++
			_, _ = fmt.Fprint(w, `{"data":[{"id":"test-model"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/ai/chat/completions":
			providerPosts++
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ledger := &failAtSetAttemptLedger{AttemptLedger: newTestAttemptLedger(t), failAt: 2}
	repository := hostedTestRepository(t, server.URL, func() string { return "test-device-key" }, ledger, Candidate{Provider: providers.ProviderMetered, ResourceRole: "metered", Reason: "hosted"})
	request := ProviderRequest{RootOperationID: root, Source: "source", SchemaJSON: `{"type":"string"}`, Role: "classify.fast", PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC}

	first, err := repository.Run(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_UNCERTAIN, first.Recovery.GetState())
	require.Equal(t, 1, ledger.setCalls)

	// The receipt is authoritative even when caching its terminal accounting
	// state fails; the durable attempt key still makes the next read recovery-only.
	second, err := repository.Run(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_SETTLED, second.Recovery.GetState())
	require.EqualValues(t, 7, second.Recovery.GetChargedCredits())
	require.Equal(t, 2, ledger.setCalls)

	third, err := repository.Run(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_SETTLED, third.Recovery.GetState())
	require.Equal(t, 3, ledger.setCalls, "later receipt read should persist the terminal state")
	require.Equal(t, 1, providerPosts, "ledger status write failure must never cause hosted redispatch")
	require.Equal(t, 1, modelReads, "receipt recovery must not resume provider selection")
	require.Equal(t, 4, receiptReads)
}
