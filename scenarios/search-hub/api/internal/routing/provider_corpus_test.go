package routing_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	internalregistry "search-hub/internal/registry"
)

// providerCorpusDir holds the descriptor fixtures the routing-recall gate routes
// across. They are TEST DATA, not a registration source — see the directory's
// README. The fixtures used to be the //go:embed'd provider seeds; the embed
// path was deleted in favour of scenario self-registration (Phase 2), and the
// descriptors were relocated here so the Ollama routing-recall gate keeps a
// representative landscape.
const providerCorpusDir = "testdata/provider_corpus"

// loadProviderCorpus parses every descriptor fixture, keyed by provider_id.
func loadProviderCorpus(t *testing.T) map[string]*registryv1.ProviderDescriptor {
	t.Helper()
	entries, err := os.ReadDir(providerCorpusDir)
	require.NoError(t, err)
	out := make(map[string]*registryv1.ProviderDescriptor)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		blob, readErr := os.ReadFile(filepath.Join(providerCorpusDir, e.Name()))
		require.NoError(t, readErr)
		d := &registryv1.ProviderDescriptor{}
		require.NoErrorf(t, protojson.Unmarshal(blob, d), "parse corpus fixture %s", e.Name())
		out[d.GetProviderId()] = d
	}
	require.NotEmpty(t, out, "at least one provider corpus fixture must ship")
	return out
}

func corpusIDs(t *testing.T) []string {
	t.Helper()
	corpus := loadProviderCorpus(t)
	ids := make([]string, 0, len(corpus))
	for id := range corpus {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// TestProviderCorpusValid guards every fixture: it must parse and — after
// Normalize — pass the same Validate gate the registry applies on
// RegisterProvider. This means the routing corpus can never drift from "what the
// registry considers a valid, registerable descriptor". (Inherited from the
// deleted providers/seeds_test.go.)
func TestProviderCorpusValid(t *testing.T) {
	for id, d := range loadProviderCorpus(t) {
		require.Equal(t, id, d.GetProviderId(), "fixture map key must equal the descriptor's provider_id")
		internalregistry.Normalize(d)
		require.NoErrorf(t, internalregistry.Validate(d), "corpus fixture %q must be a valid registerable descriptor", id)
	}
}

// TestProviderCorpusIDs pins the full fixture set so a dropped or renamed
// descriptor surfaces immediately. (Inherited from the deleted seeds_test.go.)
func TestProviderCorpusIDs(t *testing.T) {
	require.Equal(t, []string{
		"agent-manager.runs",
		"architecture-cartographer.domain-map",
		"cli-health.commands",
		"code-reference.code",
		"command-center.metrics",
		"contract-registry.contracts",
		"git-control-tower.git-provenance",
		"knowledge-observatory.docs",
		"product-manager-agent.requirements",
		"prompt-manager.action",
		"prompt-manager.skill",
		"scenario-dependency-analyzer.resources",
		"scenario-dependency-analyzer.scenarios",
		"swarm-manager.backlog",
		"swarm-manager.initiative",
		"swarm-manager.records",
		"ui-health.surfaces",
		"ui-health.widgets",
		"vrooli-onboarding.config",
		"web-search.learnings",
	}, corpusIDs(t))
}

// TestProviderCorpusLiveVsGap pins which fixtures are live (ACTIVE, callable
// endpoint) versus tracked gaps (CAPABILITY_GAP, no endpoint), guarding the
// invariant that a gap stub is a TODO row, never a live provider the router
// would fan out to. (Inherited from the deleted seeds_test.go.)
func TestProviderCorpusLiveVsGap(t *testing.T) {
	wantGaps := map[string]bool{
		"agent-manager.runs":                     true,
		"architecture-cartographer.domain-map":   true,
		"code-reference.code":                    true,
		"command-center.metrics":                 true,
		"contract-registry.contracts":            true,
		"git-control-tower.git-provenance":       true,
		"product-manager-agent.requirements":     true,
		"scenario-dependency-analyzer.resources": true,
		"scenario-dependency-analyzer.scenarios": true,
		"vrooli-onboarding.config":               true,
	}
	for id, d := range loadProviderCorpus(t) {
		internalregistry.Normalize(d)
		if wantGaps[id] {
			require.Equalf(t, registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP, d.GetState(), "%s must be a gap stub", id)
			require.Nilf(t, d.GetEndpoint(), "%s gap stub must carry no endpoint", id)
			require.NotEmptyf(t, d.GetIntendedHome(), "%s gap stub must declare intended_home", id)
		} else {
			require.Equalf(t, registryv1.ProviderState_PROVIDER_STATE_ACTIVE, d.GetState(), "%s must be a live provider", id)
			require.NotNilf(t, d.GetEndpoint(), "%s live provider must carry an endpoint", id)
		}
	}
}
