package cleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"storage-manager/hostfs"
	"storage-manager/hostpaths"
	"storage-manager/internal/module"
	"storage-manager/internal/orchestrator"
	"storage-manager/internal/providers"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	repocontract "github.com/vrooli/repo-contract-go"
	cleanupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup/cleanup_v1connect"
)

// Module builds the cleanup domain.
//
// The database handle backs the durable half of the store: the active policy
// and the audit trail. Passing nil falls back to fully in-memory state, which
// is what the endpoint-codegen binary and unit tests want — neither has a live
// database, and neither needs the operator's policy to survive anything.
func Module(logger *log.Logger, db *database.RoutedDB, fileRoots *filerouting.RoutedRoots) module.Module {
	registry, err := defaultRegistry(fileRoots)
	if err != nil {
		if logger == nil {
			logger = log.New(io.Discard, "", 0)
		}
		logger.Fatalf("cleanup registry: %v", err)
	}

	var store orchestrator.Store = orchestrator.NewMemoryStore()
	if db != nil {
		store = orchestrator.NewSQLiteStore(db)
	}
	return ModuleWithService(orchestrator.NewService(registry, store, nil))
}

func ModuleWithService(service Service) module.Module {
	connectPath, connectHandler := cleanupconnect.NewCleanupServiceHandler(NewConnectHandler(requireService(service)))
	return module.Module{
		Name: "cleanup",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// defaultRegistry builds the production provider registry.
//
// Every file provider needs two things to do anything at all: a filesystem seam
// to walk, and roots to walk within. Until this wiring existed it had neither —
// BuiltInDeps.FileSystem was left nil and all four root lists were empty
// literals — so each provider reported "filesystem seam unavailable" and
// estimated zero bytes on a host with 70 GB of reclaimable temp files. The
// planning and policy layers were correct throughout; nothing was ever
// connected to the disk.
func defaultRegistry(fileRoots *filerouting.RoutedRoots) (*providers.Registry, error) {
	roots := hostpaths.Resolve()
	files := hostfs.New(hostfs.Options{})

	stateDir, _ := os.UserConfigDir()
	var ledger *providers.FileDockerUsageLedger
	var ledgerErr error
	if fileRoots != nil {
		ledger, ledgerErr = providers.NewRoutedFileDockerUsageLedger(fileRoots)
	} else {
		ledger, ledgerErr = providers.NewFileDockerUsageLedger(filepath.Join(stateDir, "vrooli", "storage-manager", "docker-usage-ledger.json"))
	}
	if ledgerErr != nil {
		return nil, ledgerErr
	}
	ollamaLedger, err := providers.NewFileOllamaUsageLedger(filepath.Join(stateDir, "vrooli", "storage-manager", "ollama-usage-ledger.json"))
	if err != nil {
		return nil, err
	}
	repoRoot, _ := repocontract.ResolveRepoRoot()
	if repoRoot == "" {
		repoRoot, _ = os.Getwd()
	}
	home, _ := os.UserHomeDir()
	binRoot, err := resolveScenarioBinariesRoot(repoRoot, home)
	if err != nil {
		return nil, fmt.Errorf("resolve scenario binary root: %w", err)
	}
	ollama := providers.NewHTTPOllamaModelInventory(resolveOllamaBaseURL())
	builtIns, err := providers.ConservativeBuiltIns(providers.BuiltInDeps{
		FileSystem:        files,
		ProcessLiveness:   hostfs.NewProcessLiveness(),
		ProcessRunner:     hostfs.NewProcessRunner(),
		Clock:             schedule.System(),
		DockerImageLedger: ledger,
		OllamaModelProvider: providers.NewOllamaModelRetentionProvider(
			ollama,
			ollamaLedger,
			filepath.Join(repoRoot, "resources", "ollama", "model-policy.json"),
			schedule.System(),
		),

		TrashRoots:           roots.Trash,
		TmpRoots:             roots.Tmp,
		GoBuildCacheRoots:    roots.GoBuildCache,
		PlaywrightCacheRoots: roots.PlaywrightCache,
		ScenarioBinariesRoot: binRoot,
		Saturated:            autohealSaturationProbe(http.DefaultClient),
	})
	if err != nil {
		return nil, err
	}
	return providers.NewRegistry(builtIns...)
}

// autohealSaturationProbe keeps irreversible cleanup behind the same
// host-pressure gate used by the control plane. Discovery is resolved at call
// time so a restarted autoheal instance or a port change is handled without
// restarting storage-manager.
func autohealSaturationProbe(client *http.Client) func(context.Context) (bool, error) {
	if client == nil {
		client = http.DefaultClient
	}
	return func(ctx context.Context) (bool, error) {
		baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-autoheal")
		if err != nil {
			return false, fmt.Errorf("resolve vrooli-autoheal: %w", err)
		}
		requestCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/v1/checks/system-host-pressure", nil)
		if err != nil {
			return false, fmt.Errorf("build host-pressure request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return false, fmt.Errorf("query host pressure: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return false, fmt.Errorf("query host pressure: HTTP %s", resp.Status)
		}
		var payload struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return false, fmt.Errorf("decode host pressure: %w", err)
		}
		return strings.EqualFold(strings.TrimSpace(payload.Status), "critical"), nil
	}
}

func resolveScenarioBinariesRoot(repoRoot, home string) (string, error) {
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return "", err
	}
	entry, err := contract.RuntimeHomeEntry(home, repocontract.HomeKeyBin)
	if err != nil {
		return "", err
	}
	return entry.AbsPath, nil
}

func resolveOllamaBaseURL() string {
	if raw := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")); raw != "" {
		return strings.TrimRight(raw, "/")
	}
	host := strings.TrimSpace(os.Getenv("OLLAMA_HOST"))
	port := strings.TrimSpace(os.Getenv("OLLAMA_PORT"))
	if host == "" {
		host = "127.0.0.1"
	}
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		if port == "" {
			return strings.TrimRight(host, "/")
		}
		return strings.TrimRight(host, "/") + ":" + port
	}
	if strings.Contains(host, ":") {
		return "http://" + host
	}
	if port == "" {
		port = "11434"
	}
	return "http://" + host + ":" + port
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "cleanup_provider_list",
		Path:        cleanupconnect.CleanupServiceListProvidersProcedure,
		Method:      "POST",
		Summary:     "List cleanup providers",
		Description: "Returns the registered cleanup providers and their safety metadata.",
		Category:    "cleanup",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"providers": "array<Provider>"}},
	},
	{
		ID:          "cleanup_policy_get",
		Path:        cleanupconnect.CleanupServiceGetPolicyProcedure,
		Method:      "POST",
		Summary:     "Get cleanup policy",
		Description: "Returns the active cleanup policy profile and per-provider settings.",
		Category:    "cleanup",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"policy": "Policy"}},
	},
	{
		ID:          "cleanup_policy_set_profile",
		Path:        cleanupconnect.CleanupServiceSetPolicyProfileProcedure,
		Method:      "POST",
		Summary:     "Set cleanup policy profile",
		Description: "Switches the active policy to conservative, balanced, or aggressive defaults.",
		Category:    "cleanup",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"profile": "string (conservative|balanced|aggressive)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"policy": "Policy"}},
	},
	{
		ID:          "cleanup_plan_create",
		Path:        cleanupconnect.CleanupServiceCreatePlanProcedure,
		Method:      "POST",
		Summary:     "Create cleanup plan",
		Description: "Runs provider estimates and previews to create a stable cleanup plan id without mutating host state.",
		Category:    "cleanup",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"plan": "Plan"}},
	},
	{
		ID:          "cleanup_plan_apply",
		Path:        cleanupconnect.CleanupServiceApplyPlanProcedure,
		Method:      "POST",
		Summary:     "Apply cleanup plan",
		Description: "Applies an approved plan using policy/provider version checks and an idempotency key.",
		Category:    "cleanup",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"plan_id":         "string (required)",
			"policy_version":  "string (required)",
			"approval_mode":   "string",
			"approval_token":  "string",
			"idempotency_key": "string (required)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"reclaimed_bytes": "int64", "already_applied": "bool"}},
	},
	{
		ID:          "cleanup_pressure_report",
		Path:        cleanupconnect.CleanupServiceReportPressureProcedure,
		Method:      "POST",
		Summary:     "Report disk pressure",
		Description: "Inbound disk-pressure signal from a safeguard. Warning records the observation; high runs estimate and preview without deleting; critical applies safe-tier providers with no operator present. Duplicate concurrent reports of the same partition and band collapse into one execution.",
		Category:    "cleanup",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"source_scenario": "string",
			"partition":       "string (required)",
			"used_percent":    "double",
			"band":            "enum (PRESSURE_BAND_WARNING|PRESSURE_BAND_HIGH|PRESSURE_BAND_CRITICAL, required)",
			"available_bytes": "int64",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"action":                   "enum (observed|previewed|applied|deduplicated|suppressed)",
			"plan_id":                  "string",
			"estimated_bytes":          "int64",
			"reclaimed_bytes":          "int64",
			"providers_applied":        "array<string>",
			"providers_withheld":       "array<string>",
			"autonomous_apply_enabled": "bool",
		}},
	},
	{
		ID:          "cleanup_audit_list",
		Path:        cleanupconnect.CleanupServiceListAuditProcedure,
		Method:      "POST",
		Summary:     "List cleanup audit events",
		Description: "Returns immutable cleanup policy, plan, apply, and replay audit events with redacted messages.",
		Category:    "cleanup",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"events": "array<AuditEvent>"}},
	},
}
