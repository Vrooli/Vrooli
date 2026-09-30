package scenarios

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"flow-verifier/internal/flows"
)

// Summary is the row the /api/v1/scenarios list returns. The UI's
// scenarios index renders one card per Summary; flowCount drives the
// "12 flows" badge and the empty-state branch.
type Summary struct {
	ID           string `json:"id"`
	DisplayName  string `json:"displayName"`
	Description  string `json:"description,omitempty"`
	Path         string `json:"path"`
	FlowCount    int    `json:"flowCount"`
	DiscoveryErr string `json:"discoveryError,omitempty"`
}

// Detail is the structured projection of one scenario plus the flows
// discovered inside it. Returned by /api/v1/scenarios/{id}.
type Detail struct {
	Summary
	Flows []flows.Summary `json:"flows"`
}

// FlowLister abstracts the flows-discovery dependency so this package
// stays unit-testable: a tiny fake replaces flows.List in tests, and
// production wires the real package via the adapter in module.go.
type FlowLister interface {
	List(root string) ([]flows.Summary, error)
}

// Service enumerates scenarios under a Vrooli root. The root is fixed
// at construction time so handlers don't have to thread it through
// every request and tests don't have to mutate the environment.
type Service struct {
	root  string
	flows FlowLister
}

// NewService constructs the service. The root must be absolute and
// must contain a scenarios/ directory.
func NewService(vrooliRoot string, flowLister FlowLister) *Service {
	return &Service{root: vrooliRoot, flows: flowLister}
}

// Root returns the resolved Vrooli root the service is scanning. The
// UI's empty-state surfaces this so misconfigured deploys are
// self-debugging.
func (s *Service) Root() string { return s.root }

// List walks <root>/scenarios/*/.vrooli/service.json and returns one
// Summary per scenario. Per-scenario flow discovery is best-effort: if
// one scenario's flow tree can't be read, its FlowCount is 0 and
// DiscoveryErr captures the reason — the list endpoint still returns
// 200 with the other scenarios. A scenario directory without a
// service.json is silently skipped (it isn't a generated scenario).
func (s *Service) List() ([]Summary, error) {
	scenariosDir := filepath.Join(s.root, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, fmt.Errorf("scenarios: read %s: %w", scenariosDir, err)
	}
	out := make([]Summary, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(scenariosDir, entry.Name())
		summary, ok, err := s.summarize(path, entry.Name())
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		out = append(out, summary)
	}
	sort.Slice(out, func(i int, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Detail returns the scenario plus its flows. ErrScenarioNotFound when
// the id has no matching directory or that directory has no
// service.json (the same failure mode the UI's 404 page covers).
func (s *Service) Detail(id string) (Detail, error) {
	path := filepath.Join(s.root, "scenarios", id)
	summary, ok, err := s.summarize(path, id)
	if err != nil {
		return Detail{}, err
	}
	if !ok {
		return Detail{}, fmt.Errorf("%w: %s", ErrScenarioNotFound, id)
	}
	// Detail surfaces flow-discovery failures as 500s (vs List's
	// best-effort per-row error string): a detail page that silently
	// hides the failure mode wouldn't help the user fix it.
	if summary.DiscoveryErr != "" {
		return Detail{}, fmt.Errorf("scenarios: list flows for %s: %s", id, summary.DiscoveryErr)
	}
	rows, err := s.flows.List(path)
	if err != nil {
		return Detail{}, fmt.Errorf("scenarios: list flows for %s: %w", id, err)
	}
	if rows == nil {
		rows = []flows.Summary{}
	}
	return Detail{Summary: summary, Flows: rows}, nil
}

// summarize reads a single scenario directory and returns its Summary.
// The (Summary, ok, err) shape lets List skip non-scenario dirs (ok=
// false) without conflating that with a real error.
func (s *Service) summarize(path, id string) (Summary, bool, error) {
	servicePath := filepath.Join(path, ".vrooli", "service.json")
	raw, err := os.ReadFile(servicePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Summary{}, false, nil
		}
		return Summary{}, false, fmt.Errorf("scenarios: read %s: %w", servicePath, err)
	}
	var meta serviceFile
	if err := json.Unmarshal(raw, &meta); err != nil {
		return Summary{}, false, fmt.Errorf("scenarios: parse %s: %w", servicePath, err)
	}
	summary := Summary{
		ID:          id,
		DisplayName: meta.Service.DisplayName,
		Description: meta.Service.Description,
		Path:        path,
	}
	if summary.DisplayName == "" {
		summary.DisplayName = id
	}
	rows, err := s.flows.List(path)
	if err != nil {
		// Best-effort: surface the error on the row but keep going so
		// one broken scenario doesn't take down the inventory page.
		summary.DiscoveryErr = err.Error()
		return summary, true, nil
	}
	summary.FlowCount = len(rows)
	return summary, true, nil
}

// serviceFile mirrors the shape of .vrooli/service.json that this
// package depends on. Only the fields we actually read are declared;
// every other field is ignored.
type serviceFile struct {
	Service struct {
		DisplayName string `json:"displayName"`
		Description string `json:"description"`
	} `json:"service"`
}
