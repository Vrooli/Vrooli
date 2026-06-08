// Package discovery is the discovery domain's transport-agnostic service: it
// enumerates the local resources and scenarios available to compose into tasks,
// plus the task operation types and category groupings, behind a short TTL
// cache. The Connect handler in api/handlers/discovery translates these Go
// types to/from the generated proto messages and maps the sentinel errors
// below via ToConnectError. This is the reference domain service (see
// docs/internal/MIGRATION-GUIDE.md).
package discovery

import (
	"errors"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// Sentinel errors surfaced by the service. ToConnectError maps them to Connect
// status codes at the transport edge.
var (
	// ErrResourceNotFound is returned when GetResource cannot find a resource.
	ErrResourceNotFound = errors.New("resource not found")
	// ErrScenarioNotFound is returned when GetScenario cannot find a scenario.
	ErrScenarioNotFound = errors.New("scenario not found")
	// ErrEmptyName is returned when a name-keyed lookup is missing its name.
	ErrEmptyName = errors.New("name is required")
)

// cacheTTL bounds how long discovery results are reused before re-running the
// (CLI-backed, expensive) discovery sweep.
const cacheTTL = 30 * time.Second

// cache stores the last resource/scenario sweep with per-kind timestamps.
type cache struct {
	resources     []tasks.ResourceInfo
	scenarios     []tasks.ScenarioInfo
	resourcesTime time.Time
	scenariosTime time.Time
	ttl           time.Duration
	mu            sync.RWMutex
}

func (c *cache) getResources() ([]tasks.ResourceInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Since(c.resourcesTime) > c.ttl {
		return nil, false
	}
	return c.resources, true
}

func (c *cache) setResources(resources []tasks.ResourceInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resources = resources
	c.resourcesTime = time.Now()
}

func (c *cache) getScenarios() ([]tasks.ScenarioInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Since(c.scenariosTime) > c.ttl {
		return nil, false
	}
	return c.scenarios, true
}

func (c *cache) setScenarios(scenarios []tasks.ScenarioInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scenarios = scenarios
	c.scenariosTime = time.Now()
}

// Service answers discovery queries. Construct with NewService.
type Service struct {
	assembler *prompts.Assembler
	cache     *cache
}

// NewService builds a discovery Service. assembler supplies the operation
// catalogue; it may be nil, in which case Operations returns an empty slice.
func NewService(assembler *prompts.Assembler) *Service {
	return &Service{
		assembler: assembler,
		cache:     &cache{ttl: cacheTTL},
	}
}

// Resources returns the discovered resources and whether the result came from
// cache. Discovery failures degrade to an empty slice (never an error) so the
// UI never blocks on a flaky CLI sweep — mirroring the prior REST behaviour.
func (s *Service) Resources(refresh bool) ([]tasks.ResourceInfo, bool) {
	if !refresh {
		if cached, hit := s.cache.getResources(); hit {
			return cached, true
		}
	}
	resources, err := DiscoverResources()
	if err != nil {
		resources = []tasks.ResourceInfo{}
	}
	s.cache.setResources(resources)
	return resources, false
}

// Scenarios returns the discovered scenarios and whether the result came from
// cache. Discovery failures degrade to an empty slice.
func (s *Service) Scenarios(refresh bool) ([]tasks.ScenarioInfo, bool) {
	if !refresh {
		if cached, hit := s.cache.getScenarios(); hit {
			return cached, true
		}
	}
	scenarios, err := DiscoverScenarios()
	if err != nil {
		scenarios = []tasks.ScenarioInfo{}
	}
	s.cache.setScenarios(scenarios)
	return scenarios, false
}

// Resource returns one discovered resource by name. Returns ErrEmptyName when
// name is blank and ErrResourceNotFound when no resource matches.
func (s *Service) Resource(name string, refresh bool) (tasks.ResourceInfo, bool, error) {
	if name == "" {
		return tasks.ResourceInfo{}, false, ErrEmptyName
	}
	resources, cacheHit := s.Resources(refresh)
	for _, r := range resources {
		if r.Name == name {
			return r, cacheHit, nil
		}
	}
	return tasks.ResourceInfo{}, cacheHit, ErrResourceNotFound
}

// Scenario returns one discovered scenario by name. Returns ErrEmptyName when
// name is blank and ErrScenarioNotFound when no scenario matches.
func (s *Service) Scenario(name string, refresh bool) (tasks.ScenarioInfo, bool, error) {
	if name == "" {
		return tasks.ScenarioInfo{}, false, ErrEmptyName
	}
	scenarios, cacheHit := s.Scenarios(refresh)
	for _, sc := range scenarios {
		if sc.Name == name {
			return sc, cacheHit, nil
		}
	}
	return tasks.ScenarioInfo{}, cacheHit, ErrScenarioNotFound
}

// Operations returns the configured task operation types (key + description).
func (s *Service) Operations() []tasks.OperationConfig {
	if s.assembler == nil {
		return nil
	}
	config := s.assembler.GetPromptsConfig()
	out := make([]tasks.OperationConfig, 0, len(config.Operations))
	for _, op := range config.Operations {
		out = append(out, op)
	}
	return out
}

// ResourceCategories returns the resource category id → label groupings the
// create-task form offers.
func (s *Service) ResourceCategories() map[string]string {
	return map[string]string{
		"ai-ml":         "AI/ML",
		"communication": "Communication",
		"data":          "Data",
		"security":      "Security",
		"automation":    "Automation",
		"monitoring":    "Monitoring",
		"storage":       "Storage",
		"networking":    "Networking",
		"development":   "Development",
		"productivity":  "Productivity",
		"business":      "Business",
	}
}

// ScenarioCategories returns the scenario category id → label groupings.
func (s *Service) ScenarioCategories() map[string]string {
	return map[string]string{
		"productivity":   "Productivity",
		"ai-tools":       "AI Tools",
		"business":       "Business",
		"personal":       "Personal",
		"automation":     "Automation",
		"entertainment":  "Entertainment",
		"education":      "Education",
		"health-fitness": "Health & Fitness",
		"finance":        "Finance",
		"communication":  "Communication",
	}
}

// ToConnectError maps the service's sentinel errors to Connect status codes.
func ToConnectError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrEmptyName):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ErrResourceNotFound), errors.Is(err, ErrScenarioNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
