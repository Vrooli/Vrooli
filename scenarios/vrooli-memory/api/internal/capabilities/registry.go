// Package capabilities is the source-of-truth integration contract for the
// scenario dependencies declared by vrooli-memory. The dependency analyzer
// reads this package when the API is unavailable; the descriptions stay
// deliberately transport-neutral so they can be reused by an API surface.
package capabilities

import "context"

type PlatformVerdict struct {
	Support string `json:"support,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// RegistryMetadata describes memory operations themselves. Manifest
// dependencies remain in service.json and are not repeated in Known.
type RegistryMetadata struct {
	Platform PlatformVerdict `json:"platform"`
}

var Metadata = RegistryMetadata{Platform: PlatformVerdict{Support: "supported", Reason: "Durable local memory operations are host-neutral; optional correlation providers are declared in service.json."}}

type Definition struct {
	ID              string
	Description     string
	DependencyKind  string
	DependencySlug  string
	ActionKind      string
	ActionLabel     string
	OperatorCommand string
	Platform        PlatformVerdict
}

type Checker interface {
	Check(context.Context) (string, string)
}

// Known contains optional capabilities only. service.json is the source of
// truth for declared dependencies.
var Known = []Definition{}

type ScenarioChecker struct{ Slug string }

func (c ScenarioChecker) Check(context.Context) (string, string) {
	if c.Slug == "" {
		return "unavailable", "scenario slug is not configured"
	}
	return "unknown", "scenario status is resolved by the control plane"
}
