package workflows

// AssetType identifies the physical BAS asset family.
type AssetType string

const (
	AssetTypeCase         AssetType = "case"
	AssetTypeFlow         AssetType = "flow"
	AssetTypeAction       AssetType = "action"
	AssetTypeSeed         AssetType = "seed"
	AssetTypeRegistryOnly AssetType = "registry_only"
)

// AssetRole identifies how Workflow Health exposes an asset downstream.
type AssetRole string

const (
	AssetRoleValidationCase AssetRole = "validation_case"
	AssetRoleAgentFlow      AssetRole = "agent_flow"
	AssetRoleFragment       AssetRole = "fragment"
	AssetRoleSeed           AssetRole = "seed"
	AssetRoleRegistryOnly   AssetRole = "registry_only"
)

// ScenarioWorkflowCatalog is the normalized BAS catalog for one scenario.
type ScenarioWorkflowCatalog struct {
	Scenario          string           `json:"scenario"`
	ScenarioDir       string           `json:"scenario_dir"`
	Registry          RegistrySnapshot `json:"registry"`
	Assets            []WorkflowAsset  `json:"assets"`
	Cases             []WorkflowCase   `json:"cases"`
	Flows             []WorkflowFlow   `json:"flows"`
	Actions           []WorkflowAction `json:"actions"`
	Seeds             []SeedContract   `json:"seeds"`
	DependencyEdges   []DependencyEdge `json:"dependency_edges"`
	RegistryOnlyPaths []string         `json:"registry_only_paths"`
}

// RegistrySnapshot captures existing bas/registry.json facts without making
// the generated registry the source of truth for non-case assets.
type RegistrySnapshot struct {
	Path          string          `json:"path,omitempty"`
	Exists        bool            `json:"exists"`
	Scenario      string          `json:"scenario,omitempty"`
	GeneratedAt   string          `json:"generated_at,omitempty"`
	ExecutionMode string          `json:"execution_mode,omitempty"`
	Entries       []RegistryEntry `json:"entries,omitempty"`
}

type RegistryEntry struct {
	File         string   `json:"file"`
	Description  string   `json:"description,omitempty"`
	Order        string   `json:"order,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
	Fixtures     []string `json:"fixtures,omitempty"`
	Reset        string   `json:"reset,omitempty"`
}

// WorkflowAsset is the shared normalized representation of cases, flows, and actions.
type WorkflowAsset struct {
	ID                    string            `json:"id"`
	Scenario              string            `json:"scenario"`
	Path                  string            `json:"path"`
	Type                  AssetType         `json:"type"`
	Role                  AssetRole         `json:"role"`
	Name                  string            `json:"name,omitempty"`
	Description           string            `json:"description,omitempty"`
	Version               string            `json:"version,omitempty"`
	Order                 string            `json:"order,omitempty"`
	ExecutionMode         string            `json:"execution_mode,omitempty"`
	Reset                 string            `json:"reset,omitempty"`
	Labels                map[string]string `json:"labels,omitempty"`
	Requirements          []RequirementLink `json:"requirements,omitempty"`
	Selectors             []SelectorRef     `json:"selectors,omitempty"`
	Routes                []RouteRef        `json:"routes,omitempty"`
	Safety                SafetyProfile     `json:"safety"`
	Dependencies          []DependencyEdge  `json:"dependencies,omitempty"`
	NodeCount             int               `json:"node_count"`
	EnvelopeUnknownFields []string          `json:"envelope_unknown_fields,omitempty"`
	ParseError            string            `json:"parse_error,omitempty"`
}

type WorkflowCase struct {
	WorkflowAsset
}

type WorkflowFlow struct {
	WorkflowAsset
}

type WorkflowAction struct {
	WorkflowAsset
}

type SeedContract struct {
	ID       string `json:"id"`
	Scenario string `json:"scenario"`
	Path     string `json:"path"`
	Name     string `json:"name"`
}

type RequirementLink struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

type SelectorRef struct {
	NodeID string `json:"node_id,omitempty"`
	Key    string `json:"key,omitempty"`
	Raw    string `json:"raw"`
	Path   string `json:"path,omitempty"`
}

type RouteRef struct {
	NodeID   string `json:"node_id,omitempty"`
	Scenario string `json:"scenario,omitempty"`
	Path     string `json:"path,omitempty"`
	Source   string `json:"source,omitempty"`
}

type SafetyProfile struct {
	ExecutionMode        string `json:"execution_mode,omitempty"`
	Reset                string `json:"reset,omitempty"`
	Mutating             bool   `json:"mutating"`
	RequiresIsolation    bool   `json:"requires_isolation"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
}

type DependencyEdge struct {
	FromAssetID string `json:"from_asset_id"`
	FromPath    string `json:"from_path"`
	ToPath      string `json:"to_path,omitempty"`
	ToAssetID   string `json:"to_asset_id,omitempty"`
	Kind        string `json:"kind"`
	Source      string `json:"source,omitempty"`
}
