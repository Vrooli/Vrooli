// Package reactcomponentlibrary is ui-health's integration adapter for the
// scenarios/react-component-library scenario. It wraps the generated
// InventoryService Connect-RPC client with discovery (via api-core/discovery),
// bounded retry, and re-resolution on transport failure (interop-steer §8–§12).
//
// Wire boundary:
//
//	ui-health internal/aisearch.FilesystemDiscoverySource
//	         |
//	         v
//	api/integrations/reactcomponentlibrary.Client
//	         |
//	         v  (Connect-RPC; URL re-resolved on transport failure)
//	scenarios/react-component-library
package reactcomponentlibrary

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"syscall"

	"connectrpc.com/connect"

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory"
	"github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory/inventory_v1connect"
)

// Client wraps the generated InventoryService Connect client with discovery
// and recovery behavior. The base URL is resolved lazily and re-resolved on
// transport-class failures so scenario restarts don't poison the client.
type Client struct {
	resolver URLResolver
	policy   Policy
	http     *http.Client

	mu         sync.RWMutex
	baseURL    string
	inventory  inventory_v1connect.InventoryServiceClient
	components componentsconnect.ComponentsServiceClient
	resolved   atomic.Bool
}

// New constructs a Client. The Connect client is built lazily on the first
// successful Resolve; transport failures invalidate the cache and trigger
// re-resolution on the next call.
func New(resolver URLResolver, policy Policy) *Client {
	if policy.PerCallTimeout == 0 {
		policy = DefaultPolicy()
	}
	return &Client{
		resolver: resolver,
		policy:   policy,
		http:     &http.Client{Timeout: policy.PerCallTimeout},
	}
}

// refresh resolves the current URL and rebuilds the typed Connect client.
func (c *Client) refresh() error {
	base, err := c.resolver.Resolve()
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = base
	c.inventory = inventory_v1connect.NewInventoryServiceClient(c.http, base)
	c.components = componentsconnect.NewComponentsServiceClient(c.http, base)
	c.resolved.Store(true)
	return nil
}

// CatalogEntry is the minimal catalog projection needed by ui-health's
// vendored-version rule. The integration owns the wire translation so the
// rule does not depend on react-component-library's protobuf shape.
type CatalogEntry struct {
	LibraryID          string
	Latest             string
	Draft              string
	DeprecatedVersions []string
}

// ListCatalog resolves the live react-component-library catalog through the
// discovered ComponentsService. It retries transport failures after dropping
// the cached scenario URL, matching ScanScenario's lifecycle behavior.
func (c *Client) ListCatalog(ctx context.Context) ([]CatalogEntry, error) {
	if err := c.ensure(); err != nil {
		return nil, fmt.Errorf("react-component-library: resolve base URL: %w", err)
	}
	attempts := c.policy.MaxRetries + 1
	var lastErr error
	for i := 0; i < attempts; i++ {
		c.mu.RLock()
		catalog := c.components
		c.mu.RUnlock()
		resp, err := catalog.ListComponents(ctx, connect.NewRequest(&componentsv1.ListComponentsRequest{Limit: 2000}))
		if err == nil {
			entries := make([]CatalogEntry, 0, len(resp.Msg.GetComponents()))
			for _, component := range resp.Msg.GetComponents() {
				if component == nil {
					continue
				}
				entries = append(entries, CatalogEntry{
					LibraryID:          component.GetLibraryId(),
					Latest:             component.GetLatestVersion(),
					Draft:              component.GetDraftVersion(),
					DeprecatedVersions: nil,
				})
			}
			return entries, nil
		}
		lastErr = err
		if !isTransportFailure(err) {
			return nil, err
		}
		c.invalidate()
		if rerr := c.refresh(); rerr != nil {
			return nil, fmt.Errorf("re-resolve react-component-library: %w (last call: %v)", rerr, err)
		}
	}
	return nil, lastErr
}

// ensure wires the client if not already resolved.
func (c *Client) ensure() error {
	if c.resolved.Load() {
		return nil
	}
	return c.refresh()
}

// invalidate marks the client as needing re-resolution. If the resolver is
// itself a CachedResolver, also drop its cached URL so the next refresh
// hits api-core/discovery rather than returning a stale port.
func (c *Client) invalidate() {
	c.resolved.Store(false)
	if cr, ok := c.resolver.(*CachedResolver); ok {
		cr.Invalidate()
	}
}

// BaseURL returns the currently resolved base URL (diagnostics/tests).
func (c *Client) BaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL
}

// ScanScenario calls react-component-library's InventoryService.ScanScenario
// with bounded retry + re-resolution on transport failure.
func (c *Client) ScanScenario(ctx context.Context, scenario string) (*inventoryv1.ScanScenarioResponse, error) {
	if err := c.ensure(); err != nil {
		return nil, fmt.Errorf("react-component-library: resolve base URL: %w", err)
	}
	attempts := c.policy.MaxRetries + 1
	var lastErr error
	for i := 0; i < attempts; i++ {
		c.mu.RLock()
		inv := c.inventory
		c.mu.RUnlock()
		resp, err := inv.ScanScenario(ctx, connect.NewRequest(&inventoryv1.ScanScenarioRequest{Scenario: scenario}))
		if err == nil {
			return resp.Msg, nil
		}
		lastErr = err
		if !isTransportFailure(err) {
			// Schema/validation/4xx errors aren't worth re-resolving for;
			// surface them immediately (interop-steer §12 decision flow).
			return nil, err
		}
		c.invalidate()
		if rerr := c.refresh(); rerr != nil {
			return nil, fmt.Errorf("re-resolve react-component-library: %w (last call: %v)", rerr, err)
		}
	}
	return nil, lastErr
}

// ScanSubjects calls the bounded InventoryService.Scan path. It is kept
// separate from InventoryClient so existing whole-scenario discovery callers
// remain source-compatible while asset-target validation adopts the subject
// contract.
func (c *Client) ScanSubjects(ctx context.Context, subjects []*inventoryv1.Subject) (*inventoryv1.ScanResponse, error) {
	if err := c.ensure(); err != nil {
		return nil, fmt.Errorf("react-component-library: resolve base URL: %w", err)
	}
	attempts := c.policy.MaxRetries + 1
	var lastErr error
	for i := 0; i < attempts; i++ {
		c.mu.RLock()
		inv := c.inventory
		c.mu.RUnlock()
		resp, err := inv.Scan(ctx, connect.NewRequest(&inventoryv1.ScanRequest{Subjects: subjects}))
		if err == nil {
			return resp.Msg, nil
		}
		lastErr = err
		if !isTransportFailure(err) {
			return nil, err
		}
		c.invalidate()
		if rerr := c.refresh(); rerr != nil {
			return nil, fmt.Errorf("re-resolve react-component-library: %w (last call: %v)", rerr, err)
		}
	}
	return nil, lastErr
}

// isTransportFailure reports whether the error is one of the transport-class
// failures that justifies URL re-resolution. Schema/4xx errors are not
// transport failures even when the underlying connection succeeded.
func isTransportFailure(err error) bool {
	if err == nil {
		return false
	}
	// Connect-RPC wraps transport errors as CodeUnavailable.
	var ce *connect.Error
	if errors.As(err, &ce) {
		switch ce.Code() {
		case connect.CodeUnavailable, connect.CodeDeadlineExceeded:
			return true
		}
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}
