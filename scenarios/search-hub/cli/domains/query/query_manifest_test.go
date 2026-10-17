package query

import (
	"os"
	"path/filepath"
	"testing"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	"github.com/vrooli/cli-core/cliapp"
)

// TestQueryManifestCoversRoutingService asserts every RPC on RoutingService is
// either bound to a CLI command or explicitly listed in the manifest's
// omitted[]. The query domain binds RoutingService.Query; the federation
// domain binds RoutingService.Status (see federation/federation_manifest_test.go).
// Adding an RPC to routing.proto without a binding or omission fails here —
// the CLI-side parity guard mirroring the API's TestProtoConnectParity.
func TestQueryManifestCoversRoutingService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, routingv1.File_search_hub_v1_routing_routing_proto, "RoutingService")
}
