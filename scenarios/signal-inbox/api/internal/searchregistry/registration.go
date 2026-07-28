// Package searchregistry self-registers Signal Inbox's local search provider
// with Search Hub. The descriptor remains authored in .vrooli/search.json; this
// package only maps that file's transport shape onto the registry RPC at boot.
package searchregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/retry"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"
)

const scenarioID = "signal-inbox"

// RegistryClient is the narrow transport seam used by Register. Keeping the
// search-hub dependency here makes boot registration independently testable and
// ensures the inbox continues serving when the optional hub is unavailable.
type RegistryClient interface {
	RegisterProvider(context.Context, *connect.Request[registryv1.RegisterProviderRequest]) (*connect.Response[registryv1.RegisterProviderResponse], error)
}

// Register reads searchFilePath and upserts every declared provider. It is
// intentionally best-effort: callers run it in a goroutine after their listener
// is ready, and a temporarily unavailable Search Hub never fails Signal Inbox.
func Register(ctx context.Context, searchFilePath string, logger *log.Logger) {
	if logger == nil {
		logger = log.Default()
	}
	file, err := aisearch.LoadSearchFile(searchFilePath)
	if err != nil {
		logger.Printf("[%s] search self-registration skipped: %v", scenarioID, err)
		return
	}
	for _, provider := range file.Providers {
		descriptor, err := Descriptor(provider)
		if err != nil {
			logger.Printf("[%s] search self-registration skipped provider %q: %v", scenarioID, provider.ProviderID, err)
			continue
		}
		if err := registerOne(ctx, descriptor, defaultResolveBaseURL, defaultClientFactory); err != nil {
			logger.Printf("[%s] search self-registration of %q degraded (search-hub optional, continuing): %v", scenarioID, descriptor.GetProviderId(), err)
			continue
		}
		logger.Printf("[%s] search provider %q registered with search-hub", scenarioID, descriptor.GetProviderId())
	}
}

type resolver func(context.Context) (string, error)
type clientFactory func(string) RegistryClient

func registerOne(ctx context.Context, descriptor *registryv1.ProviderDescriptor, resolve resolver, newClient clientFactory) error {
	return retry.Do(ctx, retry.Config{MaxAttempts: 5, BaseDelay: time.Second, MaxDelay: 10 * time.Second}, func(int) error {
		baseURL, err := resolve(ctx)
		if err != nil {
			return fmt.Errorf("resolve search-hub: %w", err)
		}
		if _, err := newClient(baseURL).RegisterProvider(ctx, connect.NewRequest(&registryv1.RegisterProviderRequest{Descriptor_: descriptor})); err != nil {
			return fmt.Errorf("register %q: %w", descriptor.GetProviderId(), err)
		}
		return nil
	})
}

func defaultResolveBaseURL(ctx context.Context) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, "search-hub")
}

func defaultClientFactory(baseURL string) RegistryClient {
	return registryconnect.NewRegistryServiceClient(&http.Client{Timeout: 15 * time.Second}, baseURL)
}

// Descriptor maps a search.json provider to the registry wire type. Search Hub
// owns the registry contract; the source descriptor remains the scenario's SSOT.
func Descriptor(provider aisearch.ProviderConfig) (*registryv1.ProviderDescriptor, error) {
	fields := map[string]json.RawMessage{}
	for key, value := range map[string]string{
		"provider_id":    provider.ProviderID,
		"provider_group": provider.ProviderGroup,
		"bucket":         provider.Bucket,
		"type":           provider.Type,
		"description":    provider.Description,
		"scope":          provider.Scope,
	} {
		if value == "" {
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal %s: %w", key, err)
		}
		fields[key] = encoded
	}
	for key, value := range map[string]json.RawMessage{
		"endpoint":         provider.Endpoint,
		"status_endpoint":  provider.StatusEndpoint,
		"result_mapping":   provider.ResultMapping,
		"reindex_endpoint": provider.ReindexEndpoint,
		"config_endpoint":  provider.ConfigEndpoint,
	} {
		if len(value) > 0 {
			fields[key] = value
		}
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("assemble provider %q: %w", provider.ProviderID, err)
	}
	descriptor := &registryv1.ProviderDescriptor{}
	if err := protojson.Unmarshal(raw, descriptor); err != nil {
		return nil, fmt.Errorf("map provider %q: %w", provider.ProviderID, err)
	}
	tuning := provider.ResolvedTuning()
	descriptor.Tuning = &registryv1.Tuning{
		Engine:          tuning.Engine,
		EmbedModel:      tuning.EmbedModel,
		EmbedTaskPrefix: tuning.EmbedTaskPrefix,
		RerankEnabled:   tuning.RerankEnabled,
		RerankBlend:     tuning.RerankBlend,
		RerankShortlist: int32(tuning.RerankShortlist),
		Floor:           &registryv1.FloorConfig{MaxGap: tuning.Floor.MaxGap, HardFloor: tuning.Floor.HardFloor},
	}
	return descriptor, nil
}
