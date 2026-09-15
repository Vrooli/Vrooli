// Package registration publishes Code Facts' committed search descriptors to
// Search Hub without making Search Hub a startup dependency. The shared
// searchregister-go bridge is the preferred fleet path; this small local
// adapter exists because the governed dependency gateway cannot resolve that
// first-party module from the monorepo's public module proxy.
package registration

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"code-facts/internal/logging"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/discovery"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval/eval_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
)

type descriptorFile struct {
	Providers []json.RawMessage `json:"providers"`
}

// TokenStore retains Search Hub's per-leaf control tokens only in memory. It
// accepts any token minted for a Code Facts leaf because both leaves share the
// same physical generation and control plane.
type TokenStore struct {
	mu     sync.RWMutex
	byLeaf map[string]string
}

func NewTokenStore() *TokenStore { return &TokenStore{byLeaf: make(map[string]string)} }

func (store *TokenStore) Set(providerID, token string) {
	if store == nil || strings.TrimSpace(providerID) == "" || strings.TrimSpace(token) == "" {
		return
	}
	store.mu.Lock()
	store.byLeaf[providerID] = strings.TrimSpace(token)
	store.mu.Unlock()
}

func (store *TokenStore) Get(providerID string) string {
	if store == nil {
		return ""
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.byLeaf[providerID]
}

func (store *TokenStore) Matches(token string) bool {
	if store == nil || strings.TrimSpace(token) == "" {
		return false
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	for _, expected := range store.byLeaf {
		if len(expected) == len(token) && subtle.ConstantTimeCompare([]byte(expected), []byte(token)) == 1 {
			return true
		}
	}
	return false
}

// Register performs bounded, best-effort self-registration. A provider with a
// malformed descriptor is reported individually; one bad leaf never prevents
// the other leaf from registering or Code Facts from serving its API.
func Register(ctx context.Context, path string, logger logging.Logger, tokens *TokenStore) {
	if logger == nil {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		logger.Printf("code-facts search registration skipped: %v", err)
		return
	}
	var file descriptorFile
	if err := json.Unmarshal(raw, &file); err != nil {
		logger.Printf("code-facts search registration skipped: invalid search.json: %v", err)
		return
	}
	parsed, err := aisearch.ParseSearchFile(raw)
	if err != nil {
		logger.Printf("code-facts search registration skipped: invalid provider contract: %v", err)
		return
	}
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "search-hub")
	if err != nil {
		logger.Printf("code-facts search registration deferred: resolve search-hub: %v", err)
		return
	}
	client := registryconnect.NewRegistryServiceClient(http.DefaultClient, baseURL)
	evalClient := evalconnect.NewEvalServiceClient(http.DefaultClient, baseURL)
	for index, rawProvider := range file.Providers {
		var descriptor registryv1.ProviderDescriptor
		var transport map[string]json.RawMessage
		if err := json.Unmarshal(rawProvider, &transport); err != nil {
			logger.Printf("code-facts search registration skipped provider: %v", err)
			continue
		}
		delete(transport, "freshness_budget") // file uses Go duration syntax; registry wire uses protobuf duration JSON.
		transportRaw, _ := json.Marshal(transport)
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(transportRaw, &descriptor); err != nil {
			logger.Printf("code-facts search registration skipped provider: %v", err)
			continue
		}
		if descriptor.GetStatusEndpoint() != nil && descriptor.GetIndexTimestampField() == "" {
			descriptor.IndexTimestampField = "last_indexed_at"
		}
		provider := parsed.Providers[index]
		switch strings.ToLower(strings.TrimSpace(provider.Lifecycle)) {
		case "", "production":
			descriptor.Lifecycle = registryv1.Lifecycle_LIFECYCLE_PRODUCTION
		case "fixture":
			descriptor.Lifecycle = registryv1.Lifecycle_LIFECYCLE_FIXTURE
		case "experimental":
			descriptor.Lifecycle = registryv1.Lifecycle_LIFECYCLE_EXPERIMENTAL
		}
		if raw := strings.TrimSpace(provider.FreshnessBudget); raw != "" {
			budget, parseErr := time.ParseDuration(raw)
			if parseErr != nil {
				logger.Printf("code-facts search registration skipped provider %s: invalid freshness budget: %v", provider.ProviderID, parseErr)
				continue
			}
			descriptor.FreshnessBudget = durationpb.New(budget)
		}
		if minimum := provider.Tests.Minimum; minimum != nil {
			descriptor.TestsMinimum = &registryv1.EvalMinimum{
				ReviewedPositive: int32(minimum.ReviewedPositive), Negative: int32(minimum.Negative),
				RequiredTags: append([]string(nil), minimum.RequiredTags...),
			}
		}
		descriptor.JunkLeakOptOutReason = provider.Scoring.JunkLeakOptOutReason
		var lastErr error
		for attempt := 1; attempt <= 3; attempt++ {
			attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			response, registerErr := client.RegisterProvider(attemptCtx, connect.NewRequest(&registryv1.RegisterProviderRequest{
				Descriptor_: &descriptor, ControlToken: tokens.Get(descriptor.GetProviderId()),
			}))
			lastErr = registerErr
			if registerErr == nil {
				tokens.Set(descriptor.GetProviderId(), response.Msg.GetControlToken())
			}
			cancel()
			if lastErr == nil {
				logger.Printf("code-facts search provider registered: %s", descriptor.GetProviderId())
				if len(provider.Tests.Cases) > 0 {
					suiteCtx, suiteCancel := context.WithTimeout(ctx, 5*time.Second)
					_, suiteErr := evalClient.RegisterSuite(suiteCtx, connect.NewRequest(&evalv1.RegisterSuiteRequest{Suite: evalSuite(provider)}))
					suiteCancel()
					if suiteErr != nil {
						logger.Printf("code-facts search corpus registration degraded for %s: %v", descriptor.GetProviderId(), suiteErr)
					}
				}
				break
			}
			if attempt < 3 {
				time.Sleep(time.Duration(attempt) * 250 * time.Millisecond)
			}
		}
		if lastErr != nil {
			logger.Printf("code-facts search provider registration degraded for %s: %v", descriptor.GetProviderId(), fmt.Errorf("after 3 attempts: %w", lastErr))
		}
	}
}

func evalSuite(provider aisearch.ProviderConfig) *evalv1.EvalSuite {
	suite := &evalv1.EvalSuite{
		SuiteId: provider.Tests.ResolvedSuiteID(provider.ProviderID), ProviderId: provider.ProviderID,
		Name: provider.Tests.Name, Description: provider.Tests.Description, State: "active",
	}
	for _, testCase := range provider.Tests.Cases {
		suite.Cases = append(suite.Cases, &evalv1.EvalCase{
			CaseId: testCase.ID, Query: testCase.Query, Scope: testCase.Scope, Status: testCase.Status,
			Tags: append([]string(nil), testCase.Tags...), ExpectIds: append([]string(nil), testCase.ExpectIDs...),
			ExpectWithinTopK: int32(testCase.ExpectWithinTopK), ExpectMinScore: testCase.ExpectMinScore,
			ExpectMaxScore: testCase.ExpectMaxScore, ExpectNoStrongHit: testCase.ExpectNoStrongHit, Note: testCase.Note,
		})
	}
	return suite
}
