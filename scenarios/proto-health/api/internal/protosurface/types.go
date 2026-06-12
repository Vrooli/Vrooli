// Package protosurface builds a scenario-scoped inventory from the committed
// fleet proto descriptor image.
package protosurface

// Surface is the per-scenario proto inventory that validation and the public
// DescribeScenarioProtos RPC reason over.
type Surface struct {
	Scenario              string
	Files                 []File
	Services              []Service
	Messages              []Message
	IntraScenarioImports  []Import
	CrossScenarioImports  []Import
	RESTExceptions        []RESTExceptionEndpoint
	RESTExceptionPayloads []RESTExceptionPayloadRef
	RESTExceptionRefs     []RESTExceptionRef
	TransportWorld        TransportWorld
}

type File struct {
	Path        string
	Package     string
	Version     string
	Domain      string
	Stability   string
	Annotations []Annotation
}

type Annotation struct {
	Name  string
	Value string
}

type Service struct {
	FilePath string
	Package  string
	Name     string
	FullName string
	Domain   string
	RPCs     []RPC
}

type RPC struct {
	Name      string
	Input     string
	Output    string
	Transport TransportKind
}

type Message struct {
	FilePath string
	Package  string
	Name     string
	FullName string
	Domain   string
	Fields   []Field
}

type Field struct {
	Name        string
	Type        string
	MessageType string
	EnumType    string
	Repeated    bool
	Optional    bool
	Number      int32
}

type Import struct {
	FromFile     string
	ToFile       string
	FromScenario string
	ToScenario   string
	FromPackage  string
	ToPackage    string
	FromVersion  string
	ToVersion    string
	FromDomain   string
	ToDomain     string
	Kind         ImportKind
}

type ImportKind string

const (
	ImportKindUnspecified   ImportKind = "unspecified"
	ImportKindScenarioLocal ImportKind = "scenario_local"
	ImportKindCrossScenario ImportKind = "cross_scenario"
	ImportKindExternal      ImportKind = "external"
)

type RESTExceptionRef struct {
	EndpointID string
	Path       string
	Method     string
	Domain     string
	Message    string
	FullName   string
}

type RESTExceptionEndpoint struct {
	EndpointID             string
	Path                   string
	Method                 string
	Domain                 string
	Reason                 string
	HasPayloadDeclarations bool
}

type RESTPayloadRole string

const (
	RESTPayloadRoleRequest  RESTPayloadRole = "request"
	RESTPayloadRoleResponse RESTPayloadRole = "response"
	RESTPayloadRoleError    RESTPayloadRole = "error"
)

type RESTPayloadProofStatus string

const (
	RESTPayloadProofNotEvaluated RESTPayloadProofStatus = "not_evaluated"
)

type RESTExceptionPayloadRef struct {
	EndpointID    string
	Path          string
	Method        string
	Domain        string
	Reason        string
	Role          RESTPayloadRole
	ProtoFullName string
	Transport     string
	Conformance   string
	ProofStatus   RESTPayloadProofStatus
}

type TransportWorld string

const (
	TransportWorldUnspecified TransportWorld = "unspecified"
	TransportWorldConnect     TransportWorld = "connect"
	TransportWorldHandRolled  TransportWorld = "hand_rolled"
	TransportWorldMixed       TransportWorld = "mixed"
	TransportWorldNone        TransportWorld = "none"
)

type TransportKind string

const (
	TransportKindUnspecified TransportKind = "unspecified"
	TransportKindConnect     TransportKind = "connect"
	TransportKindREST        TransportKind = "rest"
	TransportKindHandRolled  TransportKind = "hand_rolled"
	TransportKindNotServed   TransportKind = "not_served"
)
