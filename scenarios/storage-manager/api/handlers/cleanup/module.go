package cleanup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/packages/artifactledger"

	"storage-manager/hostfs"
	"storage-manager/hostpaths"
	"storage-manager/internal/census"
	cleanupcore "storage-manager/internal/cleanup"
	"storage-manager/internal/growth"
	"storage-manager/internal/module"
	"storage-manager/internal/orchestrator"
	"storage-manager/internal/providers"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
	coreRetention "github.com/vrooli/api-core/retention"
	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	corestorage "github.com/vrooli/api-core/storage"
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
	return ModuleWithContext(context.Background(), logger, db, fileRoots)
}

// ModuleWithContext gives server-owned cleanup workers the API process
// lifetime. Request contexts still control how long callers wait, but do not
// detach recovery work from orderly service shutdown.
func ModuleWithContext(serviceContext context.Context, logger *log.Logger, db *database.RoutedDB, fileRoots *filerouting.RoutedRoots) module.Module {
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
	service := orchestrator.NewServiceWithContext(serviceContext, registry, store, nil)
	if baseURL := strings.TrimSpace(os.Getenv("VROOLI_MEMORY_API_BASE")); baseURL != "" {
		service.SetJournalAppender(orchestrator.NewJournalAppender(baseURL))
	}
	if resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{}); err == nil {
		if stateRoot, resolveErr := resolver.Resolve(corestorage.Options{}); resolveErr == nil {
			service.SetRecoveryLockPath(filepath.Join(stateRoot.StateDir, "recovery.lock"))
		}
	}
	if err := service.ReconcileInterruptedRecoveryRuns(context.Background()); err != nil && logger != nil {
		logger.Printf("recovery ledger reconciliation failed: %v", err)
	}
	// Resolve vrooli-events lazily only after startup reconciliation. Recovery
	// reconciliation can publish records; resolving a peer during that path
	// would consume the API readiness window. The client resolves once when the
	// first live event is emitted and degrades cleanly if events are unavailable.
	service.SetEventPublisher(eventbus.NewDiscoveredClient(context.Background()))
	wireWarningPressure(service, db)
	return ModuleWithService(service)
}

func wireWarningPressure(service *orchestrator.Service, db *database.RoutedDB) {
	repoRoot, _ := repocontract.ResolveRepoRoot()
	if strings.TrimSpace(repoRoot) == "" {
		repoRoot, _ = os.Getwd()
	}
	growthStore := growth.NewStore(db)
	service.SetWarningDependencies(orchestrator.WarningDependencies{
		FastestUnbounded: func(ctx context.Context) (orchestrator.WarningGrowthTarget, bool, error) {
			inventory, err := coreRetentionInventory(repoRoot)
			if err != nil {
				return orchestrator.WarningGrowthTarget{}, false, err
			}
			ceilings := make(map[string]int64)
			for _, owner := range inventory.Owners {
				for _, entry := range owner.StorageEntries {
					if entry.Budget == nil || entry.Budget.MaxBytes == "" {
						continue
					}
					max, parseErr := coreRetention.ParseBytes(entry.Budget.MaxBytes)
					if parseErr == nil {
						ceilings[string(owner.Kind)+"/"+owner.ID+"/"+entry.Name] = max
					}
				}
			}
			root := "/"
			if canonical, rootErr := census.DeviceRoot(root); rootErr == nil && canonical != "" {
				root = canonical
			}
			report, err := growthStore.Build(ctx, root, 24*time.Hour, ceilings)
			if err != nil {
				return orchestrator.WarningGrowthTarget{}, false, err
			}
			for _, row := range report.Owners {
				if row.CeilingStatus == "unbounded" && row.SlopeBytesPerHour > 0 {
					return orchestrator.WarningGrowthTarget{OwnerKind: row.OwnerKind, OwnerID: row.OwnerID, EntryName: row.EntryName, CurrentBytes: row.CurrentBytes, SlopeBytesPerHour: row.SlopeBytesPerHour}, true, nil
				}
			}
			return orchestrator.WarningGrowthTarget{}, false, nil
		},
		FileBug: fileScenarioQABug,
	})
}

func coreRetentionInventory(repoRoot string) (corestorage.OwnerInventory, error) {
	return corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: repoRoot, Platform: corestorage.Platform(runtime.GOOS)})
}

type warningBugCapture struct {
	Title          string            `json:"title"`
	SignalType     string            `json:"signal_type"`
	Severity       string            `json:"severity"`
	Repro          []string          `json:"repro"`
	Expected       string            `json:"expected"`
	Actual         string            `json:"actual"`
	Description    string            `json:"description"`
	Context        map[string]string `json:"context"`
	HonestyFlags   []string          `json:"honesty_flags"`
	IdempotencyKey string            `json:"idempotency_key"`
}

func fileScenarioQABug(ctx context.Context, report orchestrator.WarningBugReport) (string, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "prompt-manager")
	if err != nil {
		return "", fmt.Errorf("resolve prompt-manager: %w", err)
	}
	payload, err := json.Marshal(warningBugCapture{Title: report.Title, SignalType: report.SignalType, Severity: report.Severity, Repro: report.Repro, Expected: report.Expected, Actual: report.Actual, Description: report.Description, Context: report.Context, HonestyFlags: report.HonestyFlags, IdempotencyKey: report.IdempotencyKey})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/v1/teams/scenario-qa/bugs/capture", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("capture scenario-qa bug: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("capture scenario-qa bug: HTTP %s", resp.Status)
	}
	var result struct {
		DraftID   string `json:"draft_id"`
		Knowledge struct {
			ID string `json:"id"`
		} `json:"knowledge"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode scenario-qa bug: %w", err)
	}
	if result.Knowledge.ID != "" {
		return result.Knowledge.ID, nil
	}
	if result.DraftID != "" {
		return result.DraftID, nil
	}
	return "", fmt.Errorf("scenario-qa bug capture returned no reference")
}

func ModuleWithService(service Service) module.Module {
	connectPath, connectHandler := cleanupconnect.NewCleanupServiceHandler(NewConnectHandler(requireService(service)))
	return module.Module{
		Name: "cleanup",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			if approvals, ok := service.(standingApprovalService); ok {
				r.HandleFunc("/api/v1/cleanup/approvals", func(w http.ResponseWriter, req *http.Request) {
					policy, err := service.CurrentPolicy(req.Context())
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(policy.StandingApprovals)
				}).Methods(http.MethodGet)
				r.HandleFunc("/api/v1/cleanup/approvals/{provider}", func(w http.ResponseWriter, req *http.Request) {
					provider := mux.Vars(req)["provider"]
					if req.Method == http.MethodDelete {
						if _, err := approvals.RevokeStandingApproval(req.Context(), provider); err != nil {
							http.Error(w, err.Error(), http.StatusBadRequest)
							return
						}
						w.WriteHeader(http.StatusNoContent)
						return
					}
					var input struct {
						ApprovedAt         time.Time         `json:"approved_at"`
						ApprovedBy         string            `json:"approved_by"`
						HostID             string            `json:"host_id"`
						SubjectConstraints map[string]string `json:"subject_constraints"`
					}
					if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
						http.Error(w, "invalid approval body", http.StatusBadRequest)
						return
					}
					policy, err := approvals.SetStandingApproval(req.Context(), provider, orchestrator.StandingApproval{ApprovedAt: input.ApprovedAt, ApprovedBy: input.ApprovedBy, HostID: input.HostID, SubjectConstraints: input.SubjectConstraints})
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(policy.StandingApprovals[provider])
				}).Methods(http.MethodPost, http.MethodDelete)
			}
		},
		Endpoints: Endpoints,
	}
}

type standingApprovalService interface {
	SetStandingApproval(context.Context, string, orchestrator.StandingApproval) (orchestrator.Policy, error)
	RevokeStandingApproval(context.Context, string) (orchestrator.Policy, error)
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
	// The removal ledger is resolved from home, not from a running scenario:
	// a reclamation that cannot be recorded must not happen, and a ledger that
	// needed a service would be unavailable exactly when the control plane is
	// repairing itself.
	removalLedger, err := artifactledger.New(home)
	if err != nil {
		return nil, fmt.Errorf("resolve removal ledger: %w", err)
	}
	ollama := providers.NewHTTPOllamaModelInventory(resolveOllamaBaseURL())
	governedSpecs, err := governedRootSpecs(repoRoot)
	if err != nil {
		return nil, err
	}
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

		TrashRoots:               roots.Trash,
		TmpRoots:                 roots.Tmp,
		ScratchRoots:             hostpaths.ScratchRoots(repoRoot),
		GoBuildCacheRoots:        roots.GoBuildCache,
		PlaywrightCacheRoots:     roots.PlaywrightCache,
		ScenarioBinariesRoot:     binRoot,
		RemovalLedger:            removalLedger,
		RuntimeHomeProviders:     runtimeHomeProviderConfigs(repoRoot, home, newRuntimeHomeBrokerRepairer()),
		GovernedRootSpecs:        governedSpecs,
		OrphanedDatabaseProvider: providers.NewOrphanedDatabaseProviderWithVerifier(files, hostfs.NewProcessLiveness(), schedule.System(), filepath.Join(home, ".local", "share", "vrooli", "vrooli-autoheal", "autoheal.sqlite"), autohealLiveDatabasePath(home), 30*24*time.Hour, providers.VerifySQLiteQuickCheck, filepath.Join(home, ".vrooli", "state", "storage-manager", "quarantine")),
		Saturated:                autohealSaturationProbe(http.DefaultClient),
		OwnerScenarioClient: &cleanupcore.HTTPScenarioProviderClient{
			ResolveURL: discovery.ResolveScenarioURLDefault,
			HTTPClient: http.DefaultClient,
		},
		OwnerProviderConfigs: ownerScenarioProviderConfigs(repoRoot),
		Broker:               providers.NewPrivilegeBrokerClient(),
	})
	if err != nil {
		return nil, err
	}
	return providers.NewRegistry(builtIns...)
}

func autohealLiveDatabasePath(home string) string {
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{AppID: "vrooli", Profile: corestorage.ProfileAuto})
	if err == nil {
		// This control-plane process has its own lifecycle-injected storage
		// namespace (storage-manager). Do not let that namespace redirect the
		// path of the separate autoheal scenario; the explicit scenario ID is the
		// authority for this cross-scenario diagnostic.
		if path, pathErr := resolver.Path(corestorage.Options{ScenarioID: "vrooli-autoheal"}, corestorage.ClassData, "autoheal.sqlite"); pathErr == nil {
			return path
		}
	}
	return filepath.Join(home, ".vrooli", "data", "vrooli", "vrooli-autoheal", "autoheal.sqlite")
}

func governedRootSpecs(repoRoot string) ([]providers.RootSpec, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Storage struct {
			Roots []providers.RootSpec `json:"roots"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	out := make([]providers.RootSpec, 0, len(doc.Storage.Roots))
	for _, spec := range doc.Storage.Roots {
		if !spec.Applicable() || (spec.Tier != cleanupcore.SafetyTierSafe && spec.Tier != cleanupcore.SafetyTierRegenerable) {
			continue
		}
		if err := providers.ValidateRootSpec(spec); err != nil {
			return nil, err
		}
		out = append(out, spec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// governedRootProviderConfigs adapts declarative cache roots to the generic
// file provider. Adding a root is therefore a contract change, not a new Go
// registry branch. Owner-leased roots remain withheld until the owner-budget
// authority is available.
func governedRootProviderConfigs(repoRoot, home string) []providers.FileProviderConfig {
	data, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		return nil
	}
	var doc struct {
		Storage struct {
			Roots []struct {
				ID         string `json:"id"`
				Root       string `json:"root"`
				Tier       string `json:"tier"`
				MaxAge     string `json:"max_age"`
				MaxBytes   string `json:"max_bytes"`
				LeaseCheck string `json:"lease_check"`
			} `json:"roots"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	configs := make([]providers.FileProviderConfig, 0, len(doc.Storage.Roots))
	for _, spec := range doc.Storage.Roots {
		if spec.ID == "" || spec.Root == "" || (spec.Tier != string(cleanupcore.SafetyTierSafe) && spec.Tier != string(cleanupcore.SafetyTierRegenerable)) || spec.LeaseCheck != "none" {
			continue
		}
		maxAge, maxBytes, parseErr := parseGovernedRootLimits(spec.MaxAge, spec.MaxBytes)
		if parseErr != nil {
			continue
		}
		tier := cleanupcore.SafetyTier(spec.Tier)
		configs = append(configs, providers.FileProviderConfig{
			ID: "spec-" + spec.ID, Name: "Governed " + spec.ID,
			Roots:       []string{expandGovernedRoot(spec.Root, repoRoot, home)},
			Description: "Declarative governed root", Tier: tier,
			RetentionMaxAge: maxAge, RetentionMaxBytes: maxBytes,
		})
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].ID < configs[j].ID })
	return configs
}

func parseGovernedRootLimits(maxAge, maxBytes string) (time.Duration, int64, error) {
	var age time.Duration
	var bytes int64
	var err error
	if maxAge != "" {
		age, err = coreRetention.ParseAge(maxAge)
		if err != nil {
			return 0, 0, err
		}
	}
	if maxBytes != "" {
		bytes, err = coreRetention.ParseBytes(maxBytes)
		if err != nil {
			return 0, 0, err
		}
	}
	return age, bytes, nil
}

func expandGovernedRoot(raw, repoRoot, home string) string {
	value := strings.TrimSpace(raw)
	value = strings.ReplaceAll(value, "$USER_HOME", home)
	value = strings.ReplaceAll(value, "$VROOLI_HOME", filepath.Join(home, ".vrooli"))
	value = strings.ReplaceAll(value, "$REPO_ROOT", repoRoot)
	if strings.HasPrefix(value, "~/") {
		value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
	}
	return filepath.Clean(value)
}

func ownerScenarioProviderConfigs(repoRoot string) []providers.OwnerProviderConfig {
	paths, _ := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", ".vrooli", "service.json"))
	configs := make([]providers.OwnerProviderConfig, 0)
	for _, manifestPath := range paths {
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var manifest struct {
			Storage struct {
				Entries map[string]struct {
					Regenerable bool `json:"regenerable"`
					Budget      *struct {
						MaxAge   string `json:"max_age"`
						MaxBytes string `json:"max_bytes"`
					} `json:"budget"`
				} `json:"entries"`
				CleanupProviders []struct {
					ID              string   `json:"id"`
					Name            string   `json:"name"`
					SafetyTier      string   `json:"safety_tier"`
					DefaultMode     string   `json:"default_mode"`
					DefaultApproval string   `json:"default_approval"`
					StorageEntries  []string `json:"storage_entries"`
				} `json:"cleanup_providers"`
			} `json:"storage"`
		}
		if json.Unmarshal(data, &manifest) != nil {
			continue
		}
		owner := filepath.Base(filepath.Dir(filepath.Dir(manifestPath)))
		for _, declaration := range manifest.Storage.CleanupProviders {
			linkedEntries := make(map[string]struct{}, len(declaration.StorageEntries))
			for _, name := range declaration.StorageEntries {
				linkedEntries[name] = struct{}{}
			}
			ownerBudget := false
			// A provider can self-approve only a declared regenerable budget
			// that it explicitly owns. Missing or unknown links do not confer
			// authority, even when another entry in the manifest is budgeted.
			for name := range linkedEntries {
				entry, exists := manifest.Storage.Entries[name]
				if exists && entry.Regenerable && entry.Budget != nil && (entry.Budget.MaxAge != "" || entry.Budget.MaxBytes != "") {
					ownerBudget = true
					break
				}
			}
			configs = append(configs, providers.OwnerProviderConfig{
				ID: declaration.ID, Name: declaration.Name, OwnerScenario: owner,
				SafetyTier: cleanupcore.SafetyTier(declaration.SafetyTier), DefaultMode: cleanupcore.ProviderMode(declaration.DefaultMode), DefaultApproval: cleanupcore.ApprovalMode(declaration.DefaultApproval),
				StorageEntries: append([]string(nil), declaration.StorageEntries...),
				OwnerBudget:    ownerBudget,
			})
		}
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].ID < configs[j].ID })
	return configs
}

func runtimeHomeProviderConfigs(repoRoot, home string, repairers ...cleanupcore.OwnershipRepairer) []providers.FileProviderConfig {
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return nil
	}
	entries, err := contract.RuntimeHomeEntries(home)
	if err != nil {
		return nil
	}
	configs := make([]providers.FileProviderConfig, 0)
	var repairer cleanupcore.OwnershipRepairer
	if len(repairers) > 0 {
		repairer = repairers[0]
	}
	for _, entry := range entries {
		// A contract-protected entry is never a generic cleanup provider. This
		// reads the declaration rather than naming keys: `bin` was hard-coded
		// here and again, in a second spelling, in the retention enforcer, so
		// the two guards could disagree and any third consumer inherited
		// neither. Protection now travels with the entry.
		//
		// Protected entries are not unowned: scenario-binaries still removes a
		// proven orphan CLI triple under its own ledger and lease rules. What
		// is refused is bulk, age-or-size-driven walking of a shared root.
		if entry.Protected || !entry.Regenerable || entry.Cleanup != "storage_manager" || entry.Retention == nil {
			continue
		}
		limits, parseErr := runtimeHomeRetentionLimits(entry.Retention)
		if parseErr != nil {
			// Contract validation already rejects this in normal operation. Keep
			// provider construction fail-closed if a caller supplies a bad root.
			continue
		}
		configs = append(configs, providers.FileProviderConfig{
			ID: "runtime-home-" + entry.Key, Name: "Runtime home " + entry.Key,
			Roots: []string{entry.AbsPath}, Description: "Remove contract-eligible runtime-home entries",
			TopLevelEntries: true, RetentionMaxAge: limits.MaxAge, RetentionMaxBytes: limits.MaxBytes,
			ProtectActive: entry.Retention.ProtectActive,
			RepairClass:   entry.Key, OwnershipRepairer: repairer,
		})
	}
	return configs
}

type runtimeHomeBrokerRepairer struct {
	socket string
	uid    uint32
	gid    uint32
}

func newRuntimeHomeBrokerRepairer() cleanupcore.OwnershipRepairer {
	socket := privilegeBrokerSocketPath()
	current, err := user.Current()
	if err != nil {
		return &runtimeHomeBrokerRepairer{socket: socket}
	}
	uid, uidErr := strconv.ParseUint(current.Uid, 10, 32)
	gid, gidErr := strconv.ParseUint(current.Gid, 10, 32)
	if uidErr != nil || gidErr != nil {
		return &runtimeHomeBrokerRepairer{socket: socket}
	}
	return &runtimeHomeBrokerRepairer{socket: socket, uid: uint32(uid), gid: uint32(gid)}
}

func privilegeBrokerSocketPath() string {
	return platformgo.PrivilegeBrokerSocketPath()
}

func (r *runtimeHomeBrokerRepairer) Repair(ctx context.Context, class string) (cleanupcore.OwnershipRepairResult, error) {
	if r == nil || r.uid == 0 || r.gid == 0 {
		return cleanupcore.OwnershipRepairResult{}, fmt.Errorf("privilege broker is unavailable")
	}
	request := struct {
		Version     string `json:"version"`
		RequestID   string `json:"request_id"`
		Action      string `json:"action"`
		RuntimeHome struct {
			Class       string `json:"class"`
			ExpectedUID uint32 `json:"expected_uid"`
			ExpectedGID uint32 `json:"expected_gid"`
		} `json:"runtime_home"`
	}{Version: "v1", RequestID: "storage-manager-runtime-home-" + class, Action: "runtime-home.ownership.repair"}
	request.RuntimeHome.Class = class
	request.RuntimeHome.ExpectedUID = r.uid
	request.RuntimeHome.ExpectedGID = r.gid
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", r.socket)
	if err != nil {
		return cleanupcore.OwnershipRepairResult{}, err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return cleanupcore.OwnershipRepairResult{}, err
	}
	var result struct {
		Status   string `json:"status"`
		Code     string `json:"code"`
		Evidence struct {
			Repaired uint64 `json:"repaired"`
			Failed   uint64 `json:"failed"`
		} `json:"evidence"`
	}
	if err := json.NewDecoder(io.LimitReader(conn, 64<<10)).Decode(&result); err != nil {
		return cleanupcore.OwnershipRepairResult{}, err
	}
	if result.Status == "failed" {
		return cleanupcore.OwnershipRepairResult{Failed: result.Evidence.Failed, Code: result.Code}, nil
	}
	return cleanupcore.OwnershipRepairResult{Repaired: result.Evidence.Repaired, Failed: result.Evidence.Failed, Code: result.Code}, nil
}

func runtimeHomeRetentionLimits(policy *repocontract.RetentionPolicy) (providers.RuntimeHomeRetentionConfig, error) {
	if policy == nil {
		return providers.RuntimeHomeRetentionConfig{}, nil
	}
	var out providers.RuntimeHomeRetentionConfig
	var err error
	if policy.MaxAge != "" {
		out.MaxAge, err = coreRetention.ParseAge(policy.MaxAge)
		if err != nil {
			return out, err
		}
	}
	if policy.MaxBytes != "" {
		out.MaxBytes, err = coreRetention.ParseBytes(policy.MaxBytes)
	}
	return out, err
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
		Description: "Inbound disk-pressure signal from a safeguard. Warning, high, and critical apply only provably safe-tier providers with no operator present; owner, conditional, and forbidden providers remain withheld. Duplicate concurrent reports of the same partition and band collapse into one execution.",
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
	{
		ID:          "cleanup_recovery_start",
		Path:        cleanupconnect.CleanupServiceStartRecoveryProcedure,
		Method:      "POST",
		Summary:     "Start recovery run",
		Description: "Starts a server-owned recovery run and returns its id without waiting for filesystem work.",
		Category:    "cleanup",
	},
	{
		ID:          "cleanup_recovery_wait",
		Path:        cleanupconnect.CleanupServiceWaitRecoveryProcedure,
		Method:      "POST",
		Summary:     "Wait for recovery run",
		Description: "Waits for one server-owned recovery run to reach a terminal result.",
		Category:    "cleanup",
	},
	{
		ID:          "cleanup_recovery_history",
		Path:        cleanupconnect.CleanupServiceListRecoveryProcedure,
		Method:      "POST",
		Summary:     "List recovery history",
		Description: "Lists recent server-owned recovery runs.",
		Category:    "cleanup",
	},
	{
		ID:          "cleanup_standing_approvals",
		Path:        "/api/v1/cleanup/approvals",
		Method:      http.MethodGet,
		Summary:     "List standing approvals",
		Description: "Returns host-local approvals for conditional recovery providers.",
		Category:    "cleanup",
	},
}
