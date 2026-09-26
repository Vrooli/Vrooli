package capabilities

import "context"

// Status is the source-level integration contract used by dependency health
// when the analyzer or one of its declared providers is not running.
type Status string

const (
	StatusAvailable   Status = "available"
	StatusUnavailable Status = "unavailable"
)

type PlatformVerdict struct {
	Support string `json:"support,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// RegistryMetadata describes dependency analysis itself. Manifest
// dependencies remain in service.json and are not repeated in Known.
type RegistryMetadata struct {
	Platform PlatformVerdict `json:"platform"`
}

var Metadata = RegistryMetadata{Platform: PlatformVerdict{Support: "supported", Reason: "Dependency analysis is host-neutral; platform-specific deployability is resolved from declared inputs."}}

type Checker interface {
	Check(context.Context) (Status, string)
}

type Def struct {
	ID              string
	Description     string
	DependencyKind  string
	DependencySlug  string
	ActionKind      string
	ActionLabel     string
	OperatorCommand string
	Platform        PlatformVerdict
}

// Known contains optional capabilities only. service.json is the source of
// truth for declared dependencies.
var Known = []Def{}

type StaticChecker struct{ Available bool }

func (c StaticChecker) Check(context.Context) (Status, string) {
	if c.Available {
		return StatusAvailable, "dependency analysis provider is reachable"
	}
	return StatusUnavailable, "dependency analysis provider is unavailable; rerun after recovery"
}
