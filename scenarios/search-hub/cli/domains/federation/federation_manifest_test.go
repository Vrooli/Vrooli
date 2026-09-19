package federation

import (
	"os"
	"path/filepath"
	"testing"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	"github.com/vrooli/cli-core/cliapp"
)

// TestFederationManifestCoversRoutingServiceStatus asserts that the federation
// domain's "status" verb is bound in cli/manifest.json against
// RoutingService.Status — the CLI-side parity guard for the federation group.
//
// This mirrors the pattern used by query_manifest_test.go
// (TestQueryManifestCoversRoutingService) and providers_manifest_test.go
// (TestProvidersManifestCoversRegistryService): every RPC on the service used
// by a domain must have a corresponding CLI binding or an explicit omitted[]
// entry in manifest.json, so adding an RPC to the proto without a CLI
// command fails here rather than silently.
//
// The federation domain owns RoutingService.Status; RoutingService.Query is
// covered by the query domain's test. Together the two tests give full
// RoutingService coverage at the CLI level.
func TestFederationManifestCoversRoutingServiceStatus(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, routingv1.File_search_hub_v1_routing_routing_proto, "RoutingService")
}
