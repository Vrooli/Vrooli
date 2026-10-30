package wiring

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/findings"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/pricing"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/toolfriction"
	"agent-manager/internal/usageattribution"

	corestorage "github.com/vrooli/api-core/storage"
)

// frictionBugFiler files a friction problem in scenario-qa's bug inbox through
// the report-bug writer path (the same one the model-policy drift check uses).
type frictionBugFiler struct {
	client promptmanager.ScenarioQAReporter
}

func (f frictionBugFiler) FileFrictionBug(ctx context.Context, bug usageattribution.FrictionBug) (string, error) {
	return f.client.PublishScenarioQABug(ctx, promptmanager.ScenarioQABug{
		Title: bug.Title, SignalType: "unknown", Severity: "minor",
		Repro: bug.Repro, Expected: "an agent finishes the work without this friction", Actual: bug.Actual,
		Description: bug.Description, Context: bug.Context,
		HonestyFlags: []string{"speculative-cause", "repro-not-attempted"},
		// One bug per problem: the bug inbox deduplicates on this key.
		IdempotencyKey: "agent-manager-friction/" + bug.Fingerprint,
	})
}

// toolCallMemoTTL is how long one load of a tool window serves the usage
// surfaces: they refetch every minute and read the friction list, the
// recommendations and the opportunities together.
const toolCallMemoTTL = time.Minute

// usageStores holds the one usage and tool-friction store per database.
var usageStores sync.Map

// usageAttributionStore is the usage and tool-friction store of db, told
// which transcript parser version is current so its coverage can name the
// runs an older parser read, and sharing tool-window loads (WithToolCallMemo).
// There is one per database: supervision, the will-exhaust alert, the
// friction board and the measures read through the same store and memo.
func usageAttributionStore(db *database.DB, orch *orchestration.Orchestrator) *database.UsageAttributionRepository {
	if store, ok := usageStores.Load(db); ok {
		return store.(*database.UsageAttributionRepository)
	}
	store := database.NewUsageAttributionRepository(db).WithToolCallMemo(toolCallMemoTTL)
	if orch != nil {
		store.WithTranscriptParserVersions(orch.TranscriptParserVersion)
	}
	shared, _ := usageStores.LoadOrStore(db, store)
	return shared.(*database.UsageAttributionRepository)
}

// newFrictionService builds the usage service that lists, reports and
// dismisses friction problems over store, or reports false when its inputs
// are absent.
func newFrictionService(deps RouteDependencies, store *database.UsageAttributionRepository) (usageattribution.Service, bool) {
	readings, ok := deps.PricingRepository.(pricing.QuotaReadingSource)
	if !ok || deps.DB == nil {
		return usageattribution.Service{}, false
	}
	return usageattribution.Service{
		Store:    store,
		Readings: readings,
		Pricer:   usageattribution.NewPriceBook(deps.PricingService),
		Statuses: findings.NewFrictionStatusStore(deps.DB),
		Bugs:     frictionBugFiler{client: promptmanager.NewHTTPClient()},
		Tools:    store,
	}, true
}

// portRegistryTTL bounds how stale the port-to-scenario map may be: ports are
// assigned when a scenario starts.
const portRegistryTTL = time.Minute

// apiPortKey is the registry key of a scenario's API port.
const apiPortKey = "API_PORT"

// processPortOwners names the scenario whose API serves a local port, from the
// control plane's process registry
// ($VROOLI_HOME/processes/scenarios/<scenario>/*.json). A UI port (port_key
// UI_PORT) names no one: a request to a scenario's UI is a smoke check, not a
// reach into its internals. An unreadable registry names no owner.
func processPortOwners() toolfriction.PortOwners {
	var mu sync.Mutex
	var owners map[int]string
	var loaded time.Time
	return func(port int) string {
		mu.Lock()
		defer mu.Unlock()
		if owners == nil || time.Since(loaded) > portRegistryTTL {
			owners, loaded = readPortRegistry(), time.Now()
		}
		return owners[port]
	}
}

func readPortRegistry() map[int]string {
	owners := map[int]string{}
	roots, err := corestorage.HostGovernedRootInputs("")
	if err != nil {
		return owners
	}
	paths, _ := filepath.Glob(filepath.Join(roots.RuntimeHome, "processes", "scenarios", "*", "*.json"))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var record struct {
			Scenario string `json:"scenario"`
			Port     int    `json:"port"`
			PortKey  string `json:"port_key"`
		}
		if json.Unmarshal(raw, &record) == nil && record.Port > 0 && record.Scenario != "" && record.PortKey == apiPortKey {
			owners[record.Port] = record.Scenario
		}
	}
	return owners
}
