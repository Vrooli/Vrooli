package validation

type Resolution string

const (
	ResolutionMissing    Resolution = "missing"
	ResolutionUnreadable Resolution = "unreadable"
	ResolutionResolved   Resolution = "resolved"
)

type APIKind string

const (
	APIKindAbsent   APIKind = "absent"
	APIKindDeclared APIKind = "declared"
	APIKindGo       APIKind = "go"
)

type ServiceManifest struct {
	PortsAPI          bool
	HealthAPIPath     string
	HealthAPICheck    bool
	HealthAPICheckURL string
	ParseError        string
}

type Target struct {
	Scenario                string
	RootPath                string
	Resolution              Resolution
	APIKind                 APIKind
	APIDir                  string
	HasAPIDir               bool
	ServiceManifestPath     string
	ServiceManifestReadable bool
	Service                 ServiceManifest
	Lifecycle               LifecycleResult
	Health                  HealthProbeResult
	HTTP                    HTTPSemanticsResult
	Runtime                 RuntimeHygieneResult
	Diagnostics             []string
}
