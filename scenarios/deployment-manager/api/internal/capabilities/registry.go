// Package capabilities is the source-of-truth registry for deployment-manager's
// cross-scenario dependencies. The dependency analyzer reads this registry when
// it validates the service manifest, while the control plane remains the only
// process allowed to start or inspect another scenario.
package capabilities

type PlatformVerdict struct {
	Support string `json:"support,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// RegistryMetadata describes deployment planning itself. Manifest
// dependencies remain in service.json and are not repeated in Known.
type RegistryMetadata struct {
	Platform PlatformVerdict `json:"platform"`
}

var Metadata = RegistryMetadata{Platform: PlatformVerdict{Support: "supported", Reason: "Deployment planning is host-neutral; the resolver evaluates the selected delivery tier and host OS."}}

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

// Known contains optional capabilities only. service.json is the source of
// truth for declared dependencies.
var Known = []Definition{}

// SourceDeliveryCapability is the capability-derived selection seam for the
// one-way source profile. It is deliberately independent of runtime tier and
// host platform; unsupported capabilities remain explicit rather than being
// silently mapped to a binary release.
const SourceDeliveryCapability = "delivery.source_repository"

func ResolveDeliveryFormats(capabilities []string) []string {
	for _, capability := range capabilities {
		if capability == SourceDeliveryCapability {
			return []string{"source_repository"}
		}
	}
	return nil
}
