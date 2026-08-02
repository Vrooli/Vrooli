package storage

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/retention"
	corestorage "github.com/vrooli/api-core/storage"
	"storage-manager/internal/census"
	"storage-manager/internal/module"
	"storage-manager/internal/placement"
)

type ModuleDeps struct {
	RepoRoot string
	DB       *database.RoutedDB
}

func Module(d ModuleDeps) module.Module {
	return module.Module{Name: "storage", Mount: func(r *mux.Router) {
		store := census.NewSnapshotStore(d.DB)
		placementService := placement.New(d.DB)
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
			history, err := store.History(context.Background(), root, limit)
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
			out := retentionInventory{Findings: discovery.Findings}
			for _, owner := range discovery.Configs {
				record := retentionOwner{Kind: string(owner.Kind), ID: owner.ID, ManifestPath: owner.ManifestPath}
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
					for _, spec := range specs {
						record.Budgets = append(record.Budgets, retentionBudget{Name: spec.Budget.Name, TargetKind: string(spec.Target.Kind), MaxAge: spec.Budget.MaxAge.String(), MaxBytes: spec.Budget.MaxBytes})
					}
				}
				out.Owners = append(out.Owners, record)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/placement", func(w http.ResponseWriter, req *http.Request) {
			inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: d.RepoRoot, Platform: corestorage.Platform(runtime.GOOS)})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			platform := corestorage.Platform(strings.TrimSpace(req.URL.Query().Get("platform")))
			if platform == "" {
				platform = corestorage.Platform(runtime.GOOS)
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
			leverRegistry, leverErr := corestorage.BuildLeverRegistry(levers, nil)
			out.LeverWarnings = leverRegistry.Warnings
			if leverErr != nil {
				out.LeverError = leverErr.Error()
			}
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
				if !owner.StorageDeclared {
					suggestion := adoptionSuggestion{Kind: kind, Owner: owner.ID, ManifestPath: owner.ManifestPath, Priority: "review", Reason: "owner has no storage.entries declaration"}
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
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

type retentionInventory struct {
	Owners   []retentionOwner               `json:"owners"`
	Findings []corestorage.InventoryFinding `json:"findings,omitempty"`
}
type retentionOwner struct {
	Kind         string            `json:"kind"`
	ID           string            `json:"id"`
	ManifestPath string            `json:"manifest_path"`
	Budgets      []retentionBudget `json:"budgets,omitempty"`
	Error        string            `json:"error,omitempty"`
}
type retentionBudget struct {
	Name       string `json:"name"`
	TargetKind string `json:"target_kind"`
	MaxAge     string `json:"max_age,omitempty"`
	MaxBytes   int64  `json:"max_bytes,omitempty"`
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
type infraHealthReport struct {
	OwnerCount                int            `json:"owner_count"`
	OwnersWithDeclaredCeiling int            `json:"owners_with_declared_ceiling"`
	DeclaredCeilingCoverage   float64        `json:"declared_ceiling_coverage"`
	SnapshotCount             int            `json:"snapshot_count"`
	Confidence                string         `json:"confidence"`
	GrowthSlopeBytesPerHour   *float64       `json:"growth_slope_bytes_per_hour,omitempty"`
	LatestSnapshot            *census.Report `json:"latest_snapshot,omitempty"`
}

func ratio(n, d int) float64 {
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
	{ID: "storage_placement_plan", Path: "/api/v1/placement/plan", Method: http.MethodPost, Summary: "Preview a placement migration", Description: "Creates a deterministic migration plan after checking source and destination safety.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_placement_migrate", Path: "/api/v1/placement/migrate", Method: http.MethodPost, Summary: "Apply an approved placement migration", Description: "Copy-verifies and removes a declared source only after explicit plan approval.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_placement_audit", Path: "/api/v1/placement/audit", Method: http.MethodGet, Summary: "Read placement migration audit", Description: "Returns immutable placement migration outcomes and source-preservation state.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_adoption", Path: "/api/v1/adoption", Method: http.MethodGet, Summary: "Show declaration adoption coverage", Description: "Returns owner-kind coverage and deterministic declaration suggestions.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "storage_infra_health", Path: "/api/v1/infra-health/storage", Method: http.MethodGet, Summary: "Read storage infra-health signal", Description: "Publishes declared-ceiling coverage and the latest persisted census growth signal without rescanning.", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
}
