package provider

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
)

// TestProviderManifestCoversService asserts every RPC on
// ProviderLifecycleService is bound or omitted in cli/manifest.json.
func TestProviderManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, cliapptest.ReadManifest(t), plv1.File_audio_tools_v1_provider_lifecycle_provider_lifecycle_proto, "ProviderLifecycleService")
}

func TestProviderLogsCarriesStreamingPrimitiveEvidence(t *testing.T) {
	app := cliapptest.NewTestApp(t, nil)
	group, err := Register(app, cliapptest.ReadManifest(t))
	if err != nil {
		t.Fatalf("register provider commands: %v", err)
	}
	for _, command := range group.Subcommands {
		if command.Name == "logs" {
			if got := command.PrimitiveEvidence(); got != cliapp.PrimitiveStreaming {
				t.Fatalf("provider logs primitive evidence = %q, want streaming", got)
			}
			return
		}
	}
	t.Fatal("provider logs command missing")
}
