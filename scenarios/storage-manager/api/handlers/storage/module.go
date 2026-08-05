package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/retention"
	corestorage "github.com/vrooli/api-core/storage"
	"storage-manager/internal/census"
	"storage-manager/internal/module"
	"storage-manager/internal/placement"
	managerRetention "storage-manager/internal/retention"
)

type ModuleDeps struct {
	RepoRoot string
	DB       *database.RoutedDB
}

func Module(d ModuleDeps) module.Module {
	return module.Module{Name: "storage", Mount: func(r *mux.Router) {
		store := census.NewSnapshotStore(d.DB)
		placementService := placement.New(d.DB)
		snapshotRoot := func(root string) string {
			if canonical, err := census.DeviceRoot(root); err == nil && strings.TrimSpace(canonical) != "" {
				return canonical
			}
			return root
		}
		r.HandleFunc("/api/v1/storage/inventory", func(w http.ResponseWriter, req *http.Request) {
			if strings.TrimSpace(d.RepoRoot) == "" {
				http.Error(w, "repository root is unavailable", http.StatusServiceUnavailable)
				return
			}
			inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: d.RepoRoot, Platform: corestorage.Platform(runtime.GOOS)})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(inventory)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/census", func(w http.ResponseWriter, req *http.Request) {
			root := d.RepoRoot
			if requested := strings.TrimSpace(req.URL.Query().Get("root")); requested != "" {
				root = requested
			}
			force := strings.EqualFold(req.URL.Query().Get("force"), "true") || req.URL.Query().Get("force") == "1"
			if !force {
				latest, latestErr := store.Latest(req.Context(), snapshotRoot(root))
				if latestErr != nil {
					http.Error(w, latestErr.Error(), http.StatusInternalServerError)
					return
				}
				if latest != nil {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(latest)
					return
				}
			}
			inventory, inventoryErr := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: d.RepoRoot, Platform: corestorage.Platform(runtime.GOOS)})
			if inventoryErr != nil {
				http.Error(w, inventoryErr.Error(), http.StatusInternalServerError)
				return
			}
			report, err := census.ScanInventory(root, inventory)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			report, err = store.Save(req.Context(), report)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(report)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/census/history", func(w http.ResponseWriter, req *http.Request) {
			root := d.RepoRoot
			if requested := strings.TrimSpace(req.URL.Query().Get("root")); requested != "" {
				root = requested
			}
			limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
			history, err := store.History(context.Background(), snapshotRoot(root), limit)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(history)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/retention/owners", func(w http.ResponseWriter, req *http.Request) {
			discovery, err := retention.DiscoverOwners(d.RepoRoot)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			inventory, inventoryErr := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: d.RepoRoot, Platform: corestorage.Platform(runtime.GOOS)})
			if inventoryErr != nil {
				http.Error(w, inventoryErr.Error(), http.StatusInternalServerError)
				return
			}
			enforcement := managerRetention.Enforcer{RepoRoot: d.RepoRoot, Platform: corestorage.Platform(runtime.GOOS)}
			storageEnforcement, _ := enforcement.Enforce(req.Context(), inventory)
			out := retentionInventory{Findings: discovery.Findings}
			for _, owner := range discovery.Configs {
				record := retentionOwner{Kind: string(owner.Kind), ID: owner.ID, ManifestPath: owner.ManifestPath, EnforcementState: "unenforced"}
				if receipt, receiptErr := retention.ReadEnforcementReceipt(owner.ID); receiptErr == nil {
					record.LastCycleTime = &receipt.LastCycleTime
					record.LastEnforcementTime = receipt.LastEnforcementTime
					record.LastEnforcementError = receipt.LastError
				}
				data, readErr := os.ReadFile(owner.ManifestPath)
				if readErr != nil {
					record.Error = readErr.Error()
					out.Owners = append(out.Owners, record)
					continue
				}
				specs, parseErr := retention.ParseManifest(data)
				if parseErr != nil {
					record.Error = parseErr.Error()
				} else {
					if len(specs) == 0 {
						record.EnforcementState = "unbounded"
					}
					for _, spec := range specs {
						record.Budgets = append(record.Budgets, retentionBudget{Name: spec.Budget.Name, TargetKind: string(spec.Target.Kind), MaxAge: spec.Budget.MaxAge.String(), MaxBytes: spec.Budget.MaxBytes})
					}
				}
				storageBudget := false
				if normalized, found := findOwner(inventory.Owners, owner.ID); found {
					for _, entry := range normalized.StorageEntries {
						if entry.Budget == nil || (entry.Budget.MaxBytes == "" && entry.Budget.MaxAge == "") {
							continue
						}
						storageBudget = true
						if entry.Budget.MaxBytes == "" {
							continue
						}
						max, parseErr := retention.ParseBytes(entry.Budget.MaxBytes)
						if parseErr != nil {
							continue
						}
						seen := false
						for _, budget := range record.Budgets {
							seen = seen || budget.Name == entry.Name
						}
						if !seen {
							record.Budgets = append(record.Budgets, retentionBudget{Name: entry.Name, TargetKind: "storage_entry", MaxBytes: max, Rationale: entry.Budget.Rationale})
						}
						observed, _ := entryBytes(d.RepoRoot, normalized, entry)
						if observed > max {
							record.EnforcementState = "over_budget"
							record.Findings = append(record.Findings, retentionFinding{Code: "RETENTION_BOUND_BYTES", Budget: entry.Name, ObservedBytes: observed, MaxBytes: max, Message: "observed bytes exceed the declared ceiling"})
						}
					}
				}
				if storageBudget {
					result, enforced := storageEnforcement[owner.ID]
					switch {
					case enforced && result.Error == "" && len(record.Findings) == 0:
						record.EnforcementState = "governed"
					case !enforced || result.Error != "":
						// A declared budget that could not be measured or enforced is
						// not unbounded. Keep the operator-visible distinction explicit.
						record.EnforcementState = "unenforced"
					}
				} else if len(record.Budgets) > 0 && ownerIsRunning(req.Context(), owner.Kind, owner.ID) && len(record.Findings) == 0 {
					record.EnforcementState = "governed"
				}
				if record.LastEnforcementError != "" && record.EnforcementState == "governed" {
					record.EnforcementState = "enforcement_failed"
				}
				out.Owners = append(out.Owners, record)
			}
			for _, owner := range out.Owners {
				switch owner.EnforcementState {
				case "governed":
					out.Summary.Governed++
				case "unenforced":
					out.Summary.Unenforced++
				case "unbounded":
					out.Summary.Unbounded++
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/placement", func(w http.ResponseWriter, req *http.Request) {
			platform := corestorage.NormalizePlatform(req.URL.Query().Get("platform"))
			if platform == "" {
				platform = corestorage.HostPlatform()
			}
			if platform == "" {
				http.Error(w, "platform must be linux, macos, or windows", http.StatusBadRequest)
				return
			}
			inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: d.RepoRoot, Platform: platform})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			out := placementView{Platform: platform, Owners: []placementOwner{}}
			levers := []corestorage.Lever{}
			for _, owner := range inventory.Owners {
				for _, entry := range owner.StorageEntries {
					path, resolveErr := corestorage.ResolveOwnerStoragePath(d.RepoRoot, owner, entry, platform, corestorage.PlatformSeams{})
					record := placementOwner{Kind: string(owner.Kind), Owner: owner.ID, Entry: entry.Name, Rung: string(entry.Rung), Applicable: resolveErr == nil}
					if resolveErr != nil {
						record.Error = resolveErr.Error()
					} else {
						record.Path = path
						if entry.Relocation != nil {
							levers = append(levers, corestorage.Lever{Key: entry.Relocation.Key, Owner: owner.ID, Entry: entry.Name, Target: path, Scope: corestorage.LeverScope(entry.Relocation.Scope)})
						}
					}
					out.Owners = append(out.Owners, record)
				}
			}
			leverRegistry, leverErr := corestorage.BuildLeverRegistryWithExports(levers, nil, resourceEnvironmentExports(d.RepoRoot))
			out.LeverWarnings = leverRegistry.Warnings
			if leverErr != nil {
				out.LeverError = leverErr.Error()
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/placement/verify", func(w http.ResponseWriter, req *http.Request) {
			platform := corestorage.NormalizePlatform(req.URL.Query().Get("platform"))
			if platform == "" {
				platform = corestorage.HostPlatform()
			}
			if platform == "" {
				http.Error(w, "platform must be linux, macos, or windows", http.StatusBadRequest)
				return
			}
			inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: d.RepoRoot, Platform: platform})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			out := placementService.Verify(req.Context(), d.RepoRoot, inventory.Owners, platform)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/placement/plan", func(w http.ResponseWriter, req *http.Request) {
			var input struct{ Entry, Source, Destination string }
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			plan, err := placementService.Preview(req.Context(), input.Entry, input.Source, input.Destination)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(plan)
		}).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/placement/migrate", func(w http.ResponseWriter, req *http.Request) {
			var input struct {
				PlanID   string `json:"plan_id"`
				Approved bool   `json:"approved"`
			}
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			audit, err := placementService.Migrate(req.Context(), input.PlanID, input.Approved)
			if err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(audit)
		}).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/placement/audit", func(w http.ResponseWriter, req *http.Request) {
			audit, err := placementService.Audit(req.Context(), 20)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(audit)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/adoption", func(w http.ResponseWriter, req *http.Request) {
			inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: d.RepoRoot, Platform: corestorage.Platform(runtime.GOOS)})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			measure := strings.EqualFold(req.URL.Query().Get("measure"), "true")
			limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
			if limit < 1 || limit > 1000 {
				limit = 25
			}
			out := adoptionReport{TotalOwners: len(inventory.Owners), Findings: len(inventory.Findings), ByKind: map[string]adoptionKind{}, Suggestions: []adoptionSuggestion{}}
			for _, owner := range inventory.Owners {
				kind := string(owner.Kind)
				summary := out.ByKind[kind]
				summary.Total++
				out.Summary.Total++
				if owner.StorageDeclared {
					summary.StorageDeclared++
					out.Summary.StorageDeclared++
				}
				if len(owner.StorageEntries) > 0 {
					summary.WithStorage++
					out.Summary.WithStorage++
				}
				budgeted := 0
				for _, entry := range owner.StorageEntries {
					if entry.Budget != nil {
						budgeted++
					}
				}
				if budgeted > 0 {
					summary.WithBudget++
					out.Summary.WithBudget++
				}
				out.ByKind[kind] = summary
				if !owner.StorageDeclared || budgeted == 0 {
					reason := "owner has no storage.entries declaration"
					if owner.StorageDeclared {
						reason = "owner declares storage but has no budgeted entry"
					}
					suggestion := adoptionSuggestion{Kind: kind, Owner: owner.ID, ManifestPath: owner.ManifestPath, Priority: "review", Reason: reason}
					if measure {
						suggestion.ObservedBytes, suggestion.MeasurementComplete = ownerObservedBytes(owner)
					}
					out.Suggestions = append(out.Suggestions, suggestion)
				}
			}
			if measure {
				sort.Slice(out.Suggestions, func(i, j int) bool {
					if out.Suggestions[i].ObservedBytes != out.Suggestions[j].ObservedBytes {
						return out.Suggestions[i].ObservedBytes > out.Suggestions[j].ObservedBytes
					}
					if out.Suggestions[i].Kind != out.Suggestions[j].Kind {
						return out.Suggestions[i].Kind < out.Suggestions[j].Kind
					}
					return out.Suggestions[i].Owner < out.Suggestions[j].Owner
				})
			}
			if len(out.Suggestions) > limit {
				out.Suggestions = out.Suggestions[:limit]
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/declare/inspect", func(w http.ResponseWriter, req *http.Request) {
			inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: d.RepoRoot, Platform: corestorage.Platform(runtime.GOOS)})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			owner, ok := findOwner(inventory.Owners, req.URL.Query().Get("owner"))
			if !ok {
				http.Error(w, "owner is required and must match an inventory id", http.StatusBadRequest)
				return
			}
			out := inspectOwner(owner, d.RepoRoot)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/declare/suggest", func(w http.ResponseWriter, req *http.Request) {
			inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: d.RepoRoot, Platform: corestorage.Platform(runtime.GOOS)})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			owner, ok := findOwner(inventory.Owners, req.URL.Query().Get("owner"))
			if !ok {
				http.Error(w, "owner is required and must match an inventory id", http.StatusBadRequest)
				return
			}
			out := suggestOwner(owner, d.RepoRoot)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/declare/check", func(w http.ResponseWriter, req *http.Request) {
			inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: d.RepoRoot, Platform: corestorage.Platform(runtime.GOOS)})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			kindFilter := strings.TrimSpace(req.URL.Query().Get("kind"))
			out := declarationCheck{Owners: []declarationCheckRow{}}
			for _, owner := range inventory.Owners {
				if kindFilter != "" && string(owner.Kind) != kindFilter {
					continue
				}
				budgeted := 0
				for _, entry := range owner.StorageEntries {
					if entry.Budget != nil {
						budgeted++
					}
				}
				entries := len(owner.StorageEntries)
				out.Owners = append(out.Owners, declarationCheckRow{
					Kind:            string(owner.Kind),
					Owner:           owner.ID,
					Declared:        entries > 0,
					HasStorageBlock: owner.StorageDeclared,
					Entries:         entries,
					Budgeted:        budgeted,
				})
			}
			out.Total = len(out.Owners)
			byKind := map[string]*declarationKindRow{}
			kindOrder := []string{}
			for _, row := range out.Owners {
				if row.Declared {
					out.Declared++
				} else if row.HasStorageBlock {
					out.EmptyBlocks++
				}
				if row.Budgeted > 0 {
					out.Bounded++
				}
				out.Entries += row.Entries
				out.BudgetedEntries += row.Budgeted

				agg, ok := byKind[row.Kind]
				if !ok {
					agg = &declarationKindRow{Kind: row.Kind}
					byKind[row.Kind] = agg
					kindOrder = append(kindOrder, row.Kind)
				}
				agg.Total++
				if row.Declared {
					agg.Declared++
				}
				if row.Budgeted > 0 {
					agg.Bounded++
				}
				agg.Entries += row.Entries
				agg.BudgetedEntries += row.Budgeted
			}
			sort.Strings(kindOrder)
			out.ByKind = make([]declarationKindRow, 0, len(kindOrder))
			for _, kind := range kindOrder {
				out.ByKind = append(out.ByKind, *byKind[kind])
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/infra-health/storage", func(w http.ResponseWriter, req *http.Request) {
			root := d.RepoRoot
			if requested := strings.TrimSpace(req.URL.Query().Get("root")); requested != "" {
				root = requested
			}
			history, err := store.History(req.Context(), root, 1)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: d.RepoRoot, Platform: corestorage.Platform(runtime.GOOS)})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			withCeiling := 0
			for _, owner := range inventory.Owners {
				for _, entry := range owner.StorageEntries {
					if entry.Budget != nil {
						withCeiling++
						break
					}
				}
			}
			out := infraHealthReport{OwnerCount: len(inventory.Owners), OwnersWithDeclaredCeiling: withCeiling, DeclaredCeilingCoverage: ratio(withCeiling, len(inventory.Owners)), SnapshotCount: len(history), Confidence: "unknown"}
			if len(history) > 0 {
				out.Confidence = history[0].Confidence
				out.LatestSnapshot = &history[0]
				out.GrowthSlopeBytesPerHour = history[0].GrowthSlopeBytesPerHour
				budgetedOwners := map[string]bool{}
				enforced, _ := (managerRetention.Enforcer{RepoRoot: d.RepoRoot, Platform: corestorage.Platform(runtime.GOOS)}).Enforce(req.Context(), inventory)
				for ownerID, result := range enforced {
					if result.Error == "" {
						budgetedOwners[ownerID] = true
					}
				}
				declaredOwners := map[string]bool{}
				for _, owner := range inventory.Owners {
					for _, entry := range owner.StorageEntries {
						if entry.Budget != nil {
							declaredOwners[owner.ID] = true
							break
						}
					}
				}
				for _, entry := range history[0].Entries {
					if budgetedOwners[entry.Owner] {
						out.MeasuredBytesUnderEnforcedCeiling += entry.Bytes
					}
					if declaredOwners[entry.Owner] {
						out.MeasuredBytesUnderDeclaredCeiling += entry.Bytes
					}
				}
				out.EnforcedCeilingCoverage = ratio64(out.MeasuredBytesUnderEnforcedCeiling, history[0].MeasuredBytes)
				if history[0].Closed {
					out.DeclaredCeilingMeasuredCoverage = ratio64(out.MeasuredBytesUnderDeclaredCeiling, history[0].MeasuredBytes)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

func ownerIsRunning(ctx context.Context, kind retention.OwnerKind, owner string) bool {
	owner = strings.TrimSpace(owner)
	if owner == "" || !safeOwnerID(owner) {
		return false
	}
	commandName := "scenario"
	commandArgs := []string{"status", owner, "--json"}
	if kind == retention.OwnerResource {
		commandName = "resource"
		commandArgs = []string{"status", owner, "--json"}
	}
	// owner is restricted to a manifest identifier; exec.CommandContext does
	// not invoke a shell, so this is not shell interpolation.
	command := exec.CommandContext(ctx, "vrooli", append([]string{commandName}, commandArgs...)...) // #nosec G204,G702 -- validated owner id and fixed executable/arguments
	output, err := command.Output()
	if err != nil {
		return false
	}
	var status struct {
		Status   string `json:"status"`
		Scenario struct {
			Status       string `json:"status"`
			HealthStatus string `json:"health_status"`
		} `json:"scenario"`
	}
	if json.Unmarshal(output, &status) != nil {
		return false
	}
	current := status.Status
	if current == "" {
		current = status.Scenario.Status
		if current == "" {
			current = status.Scenario.HealthStatus
		}
	}
	switch strings.ToLower(strings.TrimSpace(current)) {
	case "running", "healthy", "ready", "started":
		return true
	default:
		return false
	}
}

func safeOwnerID(owner string) bool {
	for _, r := range owner {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func resourceEnvironmentExports(repoRoot string) map[string]string {
	result := map[string]string{}
	base := filepath.Join(repoRoot, "resources")
	var paths []string
	_ = filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() != "resource.json" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	sort.Strings(paths)
	for _, path := range paths {
		var manifest struct {
			EnvironmentExports struct {
				Static         map[string]string `json:"static"`
				FromPorts      map[string]string `json:"from_ports"`
				FromRuntimeEnv []string          `json:"from_runtime_env"`
				Derived        map[string]struct {
					Template string `json:"template"`
				} `json:"derived"`
			} `json:"environment_exports"`
		}
		data, err := os.ReadFile(path)
		if err != nil || json.Unmarshal(data, &manifest) != nil {
			continue
		}
		for key, value := range manifest.EnvironmentExports.Static {
			result[key] = value
		}
		for key, value := range manifest.EnvironmentExports.FromPorts {
			result[key] = value
		}
		for _, key := range manifest.EnvironmentExports.FromRuntimeEnv {
			result[key] = "$runtime"
		}
		for key, value := range manifest.EnvironmentExports.Derived {
			result[key] = value.Template
		}
	}
	return result
}

type retentionInventory struct {
	Owners   []retentionOwner               `json:"owners"`
	Findings []corestorage.InventoryFinding `json:"findings,omitempty"`
	Summary  retentionSummary               `json:"summary"`
}

type retentionSummary struct {
	Governed   int `json:"governed"`
	Unenforced int `json:"unenforced"`
	Unbounded  int `json:"unbounded"`
}

// declarationCheck reports adoption, not manifest readability. "Declared"
// counts owners with at least one storage entry; an owner carrying an empty
// storage block is counted in EmptyBlocks, never as declared. Conflating the
// two reported 208/208 coverage on a fleet where 132 owners had an entry and
// 6 had a ceiling.
type declarationCheck struct {
	Total int `json:"total"`
	// Declared counts owners with >= 1 storage entry.
	Declared int `json:"declared"`
	// Bounded counts owners with >= 1 entry carrying a budget.
	Bounded int `json:"bounded"`
	// EmptyBlocks counts owners whose manifest has a storage block but no
	// entries — the population `declare suggest` is aimed at.
	EmptyBlocks int `json:"empty_blocks"`
	// Entries and BudgetedEntries are entry-level totals across all owners.
	Entries         int                   `json:"entries"`
	BudgetedEntries int                   `json:"budgeted_entries"`
	ByKind          []declarationKindRow  `json:"by_kind"`
	Owners          []declarationCheckRow `json:"owners"`
}
type declarationKindRow struct {
	Kind            string `json:"kind"`
	Total           int    `json:"total"`
	Declared        int    `json:"declared"`
	Bounded         int    `json:"bounded"`
	Entries         int    `json:"entries"`
	BudgetedEntries int    `json:"budgeted_entries"`
}
type declarationCheckRow struct {
	Kind  string `json:"kind"`
	Owner string `json:"owner"`
	// Declared is entry-backed: true only when Entries > 0.
	Declared bool `json:"declared"`
	// HasStorageBlock preserves the manifest-level fact Declared used to carry,
	// so an empty block stays distinguishable from a missing one.
	HasStorageBlock bool `json:"has_storage_block"`
	Entries         int  `json:"entries"`
	Budgeted        int  `json:"budgeted"`
}
type declarationInspect struct {
	Kind          string                 `json:"kind"`
	Owner         string                 `json:"owner"`
	ManifestPath  string                 `json:"manifest_path"`
	Declared      []declarationEntryView `json:"declared"`
	ObservedBytes int64                  `json:"observed_bytes"`
	Complete      bool                   `json:"complete"`
}
type declarationEntryView struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	Budget   string `json:"budget,omitempty"`
	Budgeted bool   `json:"budgeted"`
}
type declarationSuggestion struct {
	Kind          string         `json:"kind"`
	Owner         string         `json:"owner"`
	ManifestPath  string         `json:"manifest_path"`
	MeasuredAt    string         `json:"measured_at"`
	ObservedBytes int64          `json:"observed_bytes"`
	Complete      bool           `json:"complete"`
	Block         map[string]any `json:"block"`
}
type retentionOwner struct {
	Kind                 string             `json:"kind"`
	ID                   string             `json:"id"`
	ManifestPath         string             `json:"manifest_path"`
	Budgets              []retentionBudget  `json:"budgets,omitempty"`
	EnforcementState     string             `json:"enforcement_state"`
	LastCycleTime        *time.Time         `json:"last_cycle_time,omitempty"`
	LastEnforcementTime  *time.Time         `json:"last_enforcement_time"`
	LastEnforcementError string             `json:"last_enforcement_error,omitempty"`
	Findings             []retentionFinding `json:"findings,omitempty"`
	Error                string             `json:"error,omitempty"`
}
type retentionBudget struct {
	Name       string `json:"name"`
	TargetKind string `json:"target_kind"`
	MaxAge     string `json:"max_age,omitempty"`
	MaxBytes   int64  `json:"max_bytes,omitempty"`
	Rationale  string `json:"rationale,omitempty"`
}
type retentionFinding struct {
	Code          string `json:"code"`
	Budget        string `json:"budget"`
	ObservedBytes int64  `json:"observed_bytes"`
	MaxBytes      int64  `json:"max_bytes"`
	Message       string `json:"message"`
}
type placementView struct {
	Platform      corestorage.Platform       `json:"platform"`
	Owners        []placementOwner           `json:"owners"`
	LeverWarnings []corestorage.LeverWarning `json:"lever_warnings,omitempty"`
	LeverError    string                     `json:"lever_error,omitempty"`
}
type placementOwner struct {
	Kind       string `json:"kind"`
	Owner      string `json:"owner"`
	Entry      string `json:"entry"`
	Rung       string `json:"rung"`
	Path       string `json:"path,omitempty"`
	Applicable bool   `json:"applicable"`
	Error      string `json:"error,omitempty"`
}
type adoptionKind struct {
	Total           int `json:"total"`
	StorageDeclared int `json:"storage_declared"`
	WithStorage     int `json:"with_storage"`
	WithBudget      int `json:"with_budget"`
}
type adoptionSuggestion struct {
	Kind                string `json:"kind"`
	Owner               string `json:"owner"`
	ManifestPath        string `json:"manifest_path"`
	Priority            string `json:"priority"`
	Reason              string `json:"reason"`
	ObservedBytes       int64  `json:"observed_bytes,omitempty"`
	MeasurementComplete bool   `json:"measurement_complete"`
}
type adoptionReport struct {
	TotalOwners int                     `json:"total_owners"`
	Findings    int                     `json:"findings"`
	Summary     adoptionKind            `json:"summary"`
	ByKind      map[string]adoptionKind `json:"by_kind"`
	Suggestions []adoptionSuggestion    `json:"suggestions"`
}

func findOwner(owners []corestorage.OwnerManifest, id string) (corestorage.OwnerManifest, bool) {
	id = strings.TrimSpace(id)
	for _, owner := range owners {
		if owner.ID == id || filepath.ToSlash(owner.ManifestPath) == filepath.ToSlash(id) {
			return owner, true
		}
	}
	return corestorage.OwnerManifest{}, false
}

func inspectOwner(owner corestorage.OwnerManifest, repoRoot string) declarationInspect {
	out := declarationInspect{Kind: string(owner.Kind), Owner: owner.ID, ManifestPath: owner.ManifestPath, Declared: []declarationEntryView{}}
	for _, entry := range owner.StorageEntries {
		bytes, ok := entryBytes(repoRoot, owner, entry)
		row := declarationEntryView{Name: entry.Name, Path: entry.Path.Value, Bytes: bytes, Budgeted: entry.Budget != nil}
		if entry.Budget != nil {
			row.Budget = entry.Budget.MaxBytes
		}
		out.Declared = append(out.Declared, row)
		out.ObservedBytes += bytes
		out.Complete = out.Complete || ok
	}
	if len(owner.StorageEntries) == 0 {
		out.ObservedBytes, out.Complete = ownerObservedBytes(owner)
	}
	return out
}

func suggestOwner(owner corestorage.OwnerManifest, repoRoot string) declarationSuggestion {
	measured := inspectOwner(owner, repoRoot)
	entries := map[string]any{}
	if len(owner.StorageEntries) > 0 {
		for _, entry := range owner.StorageEntries {
			bytes, _ := entryBytes(repoRoot, owner, entry)
			entries[entry.Name] = suggestedEntry(entry.Path.Value, entry.Kind, entry.Class, entry.Format, bytes, entry.Regenerable)
			if entry.Format == "sqlite" {
				entries[entry.Name+"_wal"] = suggestedEntry(entry.Path.Value+"-wal", "file", corestorage.ClassState, "", 0, true)
				entries[entry.Name+"_shm"] = suggestedEntry(entry.Path.Value+"-shm", "file", corestorage.ClassState, "", 0, true)
			}
		}
	} else if owner.Kind == corestorage.OwnerScenario {
		base := filepath.Dir(filepath.Dir(owner.ManifestPath))
		for _, name := range []string{"data", "cache", "state", "logs", "storage"} {
			path := filepath.Join(base, name)
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				class := string(corestorage.ClassState)
				if name == "data" || name == "storage" {
					class = string(corestorage.ClassData)
				}
				entries[name] = suggestedEntry(name, "dir", corestorage.Class(class), "", directorySize(path), name == "cache" || name == "logs")
			}
		}
	}
	if len(entries) == 0 {
		entries["data"] = suggestedEntry("data", "dir", corestorage.ClassData, "", 0, false)
	}
	return declarationSuggestion{Kind: string(owner.Kind), Owner: owner.ID, ManifestPath: owner.ManifestPath, MeasuredAt: time.Now().UTC().Format(time.RFC3339), ObservedBytes: measured.ObservedBytes, Complete: measured.Complete, Block: map[string]any{"storage": map[string]any{"entries": entries}}}
}

func suggestedEntry(path, kind string, class corestorage.Class, format string, observed int64, regenerable bool) map[string]any {
	max := observed * 2
	if max < 1<<20 {
		max = 1 << 20
	}
	max = ((max + (1 << 20) - 1) / (1 << 20)) * (1 << 20)
	entry := map[string]any{"rung": "owned", "path": path, "kind": kind, "class": string(class), "regenerable": regenerable, "budget": map[string]any{"max_bytes": formatBytesDeclaration(max), "rationale": fmt.Sprintf("Measured at %d bytes on %s; ceiling is two times observed size.", observed, time.Now().UTC().Format("2006-01-02"))}, "rationale": "Suggested from the owner's measured storage surface."}
	if format != "" {
		entry["format"] = format
	}
	return entry
}

func formatBytesDeclaration(bytes int64) string {
	for _, unit := range []struct {
		value int64
		name  string
	}{{1 << 30, "GiB"}, {1 << 20, "MiB"}, {1 << 10, "KiB"}} {
		if bytes >= unit.value && bytes%unit.value == 0 {
			return fmt.Sprintf("%d%s", bytes/unit.value, unit.name)
		}
	}
	return fmt.Sprintf("%dB", bytes)
}

func entryBytes(repoRoot string, owner corestorage.OwnerManifest, entry corestorage.StorageEntry) (int64, bool) {
	path, err := corestorage.ResolveOwnerStoragePath(repoRoot, owner, entry, corestorage.Platform(runtime.GOOS), corestorage.PlatformSeams{})
	if err != nil {
		return 0, false
	}
	return directorySize(path), true
}

func directorySize(path string) int64 {
	var total int64
	_ = filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if info, statErr := entry.Info(); statErr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

type infraHealthReport struct {
	OwnerCount                        int            `json:"owner_count"`
	OwnersWithDeclaredCeiling         int            `json:"owners_with_declared_ceiling"`
	DeclaredCeilingCoverage           float64        `json:"declared_ceiling_coverage"`
	SnapshotCount                     int            `json:"snapshot_count"`
	Confidence                        string         `json:"confidence"`
	GrowthSlopeBytesPerHour           *float64       `json:"growth_slope_bytes_per_hour,omitempty"`
	MeasuredBytesUnderEnforcedCeiling int64          `json:"measured_bytes_under_enforced_ceiling"`
	EnforcedCeilingCoverage           float64        `json:"enforced_ceiling_coverage"`
	MeasuredBytesUnderDeclaredCeiling int64          `json:"measured_bytes_under_declared_ceiling"`
	DeclaredCeilingMeasuredCoverage   float64        `json:"declared_ceiling_measured_coverage"`
	LatestSnapshot                    *census.Report `json:"latest_snapshot,omitempty"`
}

func ratio(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func ratio64(n, d int64) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func ownerObservedBytes(owner corestorage.OwnerManifest) (int64, bool) {
	if owner.Kind != corestorage.OwnerScenario {
		return 0, false
	}
	base := filepath.Dir(filepath.Dir(owner.ManifestPath))
	candidates := []string{"data", "logs", "storage", "state", "uploads", "models", "cache", "runtime", filepath.Join("api", "data"), filepath.Join("api", "storage")}
	var total int64
	complete := true
	files := 0
	foundRoot := false
	for _, name := range candidates {
		root := filepath.Join(base, name)
		if _, err := os.Stat(root); err != nil {
			continue
		}
		foundRoot = true
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				complete = false
				return nil
			}
			if entry.IsDir() {
				for _, skip := range []string{"node_modules", ".git", "coverage"} {
					if entry.Name() == skip && path != root {
						return filepath.SkipDir
					}
				}
				return nil
			}
			files++
			if files > 50000 || total > 64<<20 {
				complete = false
				return filepath.SkipAll
			}
			info, err := entry.Info()
			if err != nil {
				complete = false
				return nil
			}
			total += info.Size()
			return nil
		})
		if err != nil {
			complete = false
		}
	}
	return total, complete && foundRoot
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "storage_inventory", Path: "/api/v1/storage/inventory", Method: http.MethodGet, Summary: "List storage owners and declarations", Description: "Deterministic owner-neutral inventory across scenarios, resources, tools, and safeguards.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_census", Path: "/api/v1/census", Method: http.MethodGet, Summary: "Measure declared and unattributed storage", Description: "Read-only closed accounting over the selected root.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_census_history", Path: "/api/v1/census/history", Method: http.MethodGet, Summary: "Read persisted census history", Description: "Returns immutable census snapshots and growth observations for the selected root.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_retention_owners", Path: "/api/v1/retention/owners", Method: http.MethodGet, Summary: "List owner retention budgets", Description: "Loads retention declarations across scenarios, resources, tools, and safeguards with typed parse errors.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_placement_show", Path: "/api/v1/placement", Method: http.MethodGet, Summary: "Show resolved storage placement", Description: "Resolves portable owner declarations for a requested platform without changing host state.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_placement_verify", Path: "/api/v1/placement/verify", Method: http.MethodGet, Summary: "Verify storage placement for a platform", Description: "Resolves every declaration and distinguishes declared absence from an unresolvable path without moving bytes.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_placement_plan", Path: "/api/v1/placement/plan", Method: http.MethodPost, Summary: "Preview a placement migration", Description: "Creates a deterministic migration plan after checking source and destination safety.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_placement_migrate", Path: "/api/v1/placement/migrate", Method: http.MethodPost, Summary: "Apply an approved placement migration", Description: "Copy-verifies and removes a declared source only after explicit plan approval.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_placement_audit", Path: "/api/v1/placement/audit", Method: http.MethodGet, Summary: "Read placement migration audit", Description: "Returns immutable placement migration outcomes and source-preservation state.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_adoption", Path: "/api/v1/adoption", Method: http.MethodGet, Summary: "Show declaration adoption coverage", Description: "Returns owner-kind coverage and deterministic declaration suggestions.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_declare_inspect", Path: "/api/v1/declare/inspect", Method: http.MethodGet, Summary: "Inspect one declaration", Description: "Shows declared entries beside measured bytes for one owner.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_declare_suggest", Path: "/api/v1/declare/suggest", Method: http.MethodGet, Summary: "Suggest a storage declaration", Description: "Emits a measured, schema-valid storage block for one owner.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_declare_check", Path: "/api/v1/declare/check", Method: http.MethodGet, Summary: "Check declaration coverage", Description: "Reports declared and budgeted coverage by owner kind.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_infra_health", Path: "/api/v1/infra-health/storage", Method: http.MethodGet, Summary: "Read storage infra-health signal", Description: "Publishes declared-ceiling coverage and the latest persisted census growth signal without rescanning.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
}
