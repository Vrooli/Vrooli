// Package protosurface builds a scenario-scoped inventory from the committed
// fleet proto descriptor image.
package protosurface

// Surface is the per-scenario proto inventory that validation and the public
// DescribeScenarioProtos RPC reason over.
type Surface struct {
	Scenario             string
	Files                []File
	Services             []Service
	Messages             []Message
	IntraScenarioImports []Import
	CrossScenarioImports []Import
	RESTExceptionRefs    []RESTExceptionRef
	AdoptionSignals      []AdoptionSignal
	TransportWorld       TransportWorld
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
	FromFile   string
	ToFile     string
	FromDomain string
	ToDomain   string
}

type RESTExceptionRef struct {
	EndpointID string
	Path       string
	Method     string
	Domain     string
	Message    string
	FullName   string
}

type AdoptionSignal struct {
	Name    string
	Present bool
	Detail  string
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
