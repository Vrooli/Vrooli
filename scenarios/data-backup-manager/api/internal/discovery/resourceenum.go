package discovery

import (
	"context"
	"fmt"
	"strings"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ResourceRef is one platform resource as reported by the CLI: enough to read
// its durable_data declaration without hardcoding the repo layout.
type ResourceRef struct {
	Name         string
	Driver       string
	Enabled      bool
	ManifestPath string
}

// CLIResourceEnumerator enumerates resources through the typed `vrooli resource
// list` contract, keeping only enabled host-filesystem (external-cli) resources
// with a manifest path — the only ones that can declare durable host state.
//
// Errors propagate. A missing or broken CLI surfaces as a discovery error
// rather than silently yielding zero targets: for a backup manager, quietly
// enumerating nothing is more dangerous than failing loudly, and propagating
// matches the CompositeScanner contract ("a real failure is never silently
// hidden").
type CLIResourceEnumerator struct {
	client *vroolicli.Client
}

// NewResourceEnumerator constructs the production enumerator.
func NewResourceEnumerator() *CLIResourceEnumerator {
	return &CLIResourceEnumerator{client: vroolicli.New()}
}

// Compile-time guarantee.
var _ ResourceEnumerator = (*CLIResourceEnumerator)(nil)

func (e *CLIResourceEnumerator) Enumerate(ctx context.Context) ([]ResourceRef, error) {
	resp, err := e.client.ListResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("enumerate resources: %w", err)
	}
	return filterEnabledExternalCLI(resp.GetResources()), nil
}

// filterEnabledExternalCLI keeps only enabled external-cli resources that carry
// a manifest path — the resources that can declare durable host state.
func filterEnabledExternalCLI(resources []*cliv1.Resource) []ResourceRef {
	refs := make([]ResourceRef, 0, len(resources))
	for _, r := range resources {
		if !r.GetEnabled() || r.GetDriver() != "external-cli" {
			continue
		}
		if strings.TrimSpace(r.GetManifestPath()) == "" {
			continue
		}
		refs = append(refs, ResourceRef{
			Name:         r.GetName(),
			Driver:       r.GetDriver(),
			Enabled:      r.GetEnabled(),
			ManifestPath: r.GetManifestPath(),
		})
	}
	return refs
}
