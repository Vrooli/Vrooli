package aisearch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"ui-health/integrations/reactcomponentlibrary"

	contractsprovenancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/contracts/provenance"
	contractswidgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/contracts/widget"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory"
)

// DiscoverySource enumerates scenarios and produces SurfaceRecord values
// per scenario by dispatching to the appropriate per-framework
// component-library scenario over Connect-RPC. The interface is kept narrow
// so tests can replace it with an in-memory fake.
type DiscoverySource interface {
	ListScenarios(ctx context.Context) ([]string, error)
	Discover(ctx context.Context, scenario string) ([]SurfaceRecord, error)
}

// InventoryClient is the seam between discovery and a remote
// InventoryService implementation. The default impl wraps a generated
// Connect-Go client; tests substitute fakes.
type InventoryClient interface {
	ScanScenario(ctx context.Context, scenario string) (*inventoryv1.ScanScenarioResponse, error)
}

// FrameworkDispatchRule pairs a service.json template id with the
// component-library scenario id that implements InventoryService for that
// framework. v1 has exactly one rule (react-vite → react-component-library);
// future rules append without code changes. The dispatch URL is *not*
// stored here — it is resolved per-call via the integrations adapter so
// scenario restarts don't poison the dispatcher (interop-steer §9).
type FrameworkDispatchRule struct {
	TemplateID string // e.g. "react-vite"
	Library    string // e.g. "react-component-library"
}

// DefaultDispatchRules returns the v1 rule set.
func DefaultDispatchRules() []FrameworkDispatchRule {
	return []FrameworkDispatchRule{
		{TemplateID: "react-vite", Library: "react-component-library"},
	}
}

// FilesystemDiscoverySource walks the repo's scenarios/ tree, resolves each
// scenario's framework from service.json, and dispatches the scan to the
// matching component-library scenario over Connect-RPC. Outbound calls go
// through per-library integration adapters that own discovery, retry, and
// transport-failure re-resolution.
type FilesystemDiscoverySource struct {
	RepoRoot string
	Rules    []FrameworkDispatchRule
	Clients  map[string]InventoryClient // keyed by library id; built lazily

	mu sync.Mutex
}

// NewFilesystemDiscoverySource constructs a discovery source rooted at the
// given repo path with the default dispatch rules and the production
// react-component-library integration adapter (api-core/discovery-backed,
// with re-resolution on transport failure).
func NewFilesystemDiscoverySource(repoRoot string) *FilesystemDiscoverySource {
	return &FilesystemDiscoverySource{
		RepoRoot: repoRoot,
		Rules:    DefaultDispatchRules(),
		Clients: map[string]InventoryClient{
			"react-component-library": reactcomponentlibrary.New(
				reactcomponentlibrary.DefaultResolver(),
				reactcomponentlibrary.DefaultPolicy(),
			),
		},
	}
}

// ListScenarios returns every directory under scenarios/ in the repo.
// Scenarios whose service.json template id has no dispatch rule are still
// listed; Discover returns an empty result with a logged note for them.
func (d *FilesystemDiscoverySource) ListScenarios(_ context.Context) ([]string, error) {
	if strings.TrimSpace(d.RepoRoot) == "" {
		return nil, fmt.Errorf("repo root is required")
	}
	entries, err := os.ReadDir(filepath.Join(d.RepoRoot, "scenarios"))
	if err != nil {
		return nil, fmt.Errorf("read scenarios/: %w", err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

// Discover returns SurfaceRecord values for one scenario by reading its
// service.json, looking up the dispatch rule, and calling the library's
// InventoryService.ScanScenario over Connect-RPC.
func (d *FilesystemDiscoverySource) Discover(ctx context.Context, scenario string) ([]SurfaceRecord, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, fmt.Errorf("scenario is required")
	}

	templateID, err := readTemplateID(d.RepoRoot, scenario)
	if err != nil {
		return nil, err
	}
	rule, ok := d.lookupRule(templateID)
	if !ok {
		return nil, nil
	}
	client, ok := d.clientFor(rule)
	if !ok {
		return nil, fmt.Errorf("no InventoryClient registered for library %q", rule.Library)
	}
	resp, err := client.ScanScenario(ctx, scenario)
	if err != nil {
		return nil, fmt.Errorf("scan %s via %s: %w", scenario, rule.Library, err)
	}

	provenanceByPath := make(map[string]*contractsprovenancev1.ComponentProvenance, len(resp.GetProvenance()))
	for _, p := range resp.GetProvenance() {
		provenanceByPath[p.GetFilePath()] = p
	}
	widgetByPath := make(map[string]*contractswidgetv1.WidgetDeclaration, len(resp.GetWidgets()))
	for _, w := range resp.GetWidgets() {
		widgetByPath[w.GetFilePath()] = w
	}

	out := make([]SurfaceRecord, 0, len(resp.GetSurfaces()))
	for _, s := range resp.GetSurfaces() {
		rec := SurfaceRecord{
			Scenario:    s.GetScenario(),
			Slot:        s.GetSlot(),
			Kind:        s.GetKind().String(),
			DisplayName: s.GetDisplayName(),
			Description: s.GetDescription(),
			FilePath:    s.GetFilePath(),
		}
		if rec.Scenario == "" {
			rec.Scenario = scenario
		}
		if p, ok := provenanceByPath[s.GetFilePath()]; ok {
			rec.Provenance = provenanceFromProto(p)
		}
		if w, ok := widgetByPath[s.GetFilePath()]; ok {
			rec.Widget = widgetFromProto(w)
		}
		out = append(out, rec)
	}
	return out, nil
}

func (d *FilesystemDiscoverySource) lookupRule(templateID string) (FrameworkDispatchRule, bool) {
	for _, r := range d.Rules {
		if r.TemplateID == templateID {
			return r, true
		}
	}
	return FrameworkDispatchRule{}, false
}

func (d *FilesystemDiscoverySource) clientFor(rule FrameworkDispatchRule) (InventoryClient, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	c, ok := d.Clients[rule.Library]
	return c, ok && c != nil
}

// readTemplateID extracts generation.template.id from a scenario's
// service.json. Returns ("", nil) when the file is absent or malformed —
// those scenarios are silently skipped so missing-service-json is not a
// dispatch error.
func readTemplateID(repoRoot, scenario string) (string, error) {
	path := filepath.Join(repoRoot, "scenarios", scenario, ".vrooli", "service.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read service.json: %w", err)
	}
	var doc struct {
		Generation struct {
			Template struct {
				ID string `json:"id"`
			} `json:"template"`
		} `json:"generation"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", nil
	}
	return doc.Generation.Template.ID, nil
}

func provenanceFromProto(p *contractsprovenancev1.ComponentProvenance) *ProvenancePayload {
	if p == nil {
		return nil
	}
	return &ProvenancePayload{
		Provenance:     p.GetProvenance().String(),
		Library:        p.GetLibrary(),
		LibraryVersion: p.GetLibraryVersion(),
		ComponentName:  p.GetComponentName(),
		AdoptionID:     p.GetAdoptionId(),
		AppliedAt:      p.GetAppliedAt(),
		SourceSha256:   p.GetSourceSha256(),
		DriftHash:      p.GetDriftHash(),
		FilePath:       p.GetFilePath(),
	}
}

func widgetFromProto(w *contractswidgetv1.WidgetDeclaration) *WidgetPayload {
	if w == nil {
		return nil
	}
	return &WidgetPayload{
		WidgetID:        w.GetWidgetId(),
		ComponentName:   w.GetComponentName(),
		PropsSchemaJSON: w.GetPropsSchemaJson(),
		Slot:            w.GetSlot().String(),
		Scope:           w.GetScope().String(),
		Description:     w.GetDescription(),
		FilePath:        w.GetFilePath(),
	}
}
