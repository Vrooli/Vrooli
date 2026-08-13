// Package registration publishes Code Facts' committed search descriptors to
// Search Hub without making Search Hub a startup dependency. The shared
// searchregister-go bridge is the preferred fleet path; this small local
// adapter exists because the governed dependency gateway cannot resolve that
// first-party module from the monorepo's public module proxy.
package registration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

type descriptorFile struct {
	Providers []json.RawMessage `json:"providers"`
}

// Register performs bounded, best-effort self-registration. A provider with a
// malformed descriptor is reported individually; one bad leaf never prevents
// the other leaf from registering or Code Facts from serving its API.
func Register(ctx context.Context, path string, logger *log.Logger) {
	if logger == nil {
		logger = log.Default()
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
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "search-hub")
	if err != nil {
		logger.Printf("code-facts search registration deferred: resolve search-hub: %v", err)
		return
	}
	client := registryconnect.NewRegistryServiceClient(http.DefaultClient, baseURL)
	for _, rawProvider := range file.Providers {
		var descriptor registryv1.ProviderDescriptor
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawProvider, &descriptor); err != nil {
			logger.Printf("code-facts search registration skipped provider: %v", err)
			continue
		}
		if descriptor.GetStatusEndpoint() != nil && descriptor.GetIndexTimestampField() == "" {
			descriptor.IndexTimestampField = "last_indexed_at"
		}
		var lastErr error
		for attempt := 1; attempt <= 3; attempt++ {
			attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			_, lastErr = client.RegisterProvider(attemptCtx, connect.NewRequest(&registryv1.RegisterProviderRequest{Descriptor_: &descriptor}))
			cancel()
			if lastErr == nil {
				logger.Printf("code-facts search provider registered: %s", descriptor.GetProviderId())
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
