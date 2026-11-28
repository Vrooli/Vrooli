// Package destinationtopology compares destination failure domains using only
// topology facts supplied by an owning adapter. It performs no host discovery.
package destinationtopology

// EndpointKind describes what is known about the endpoint that stores data.
type EndpointKind string

const (
	EndpointUnknown        EndpointKind = "unknown"
	EndpointLocal          EndpointKind = "local"
	EndpointRemoteHost     EndpointKind = "remote_host"
	EndpointProviderHosted EndpointKind = "provider_hosted"
)

// Evidence contains only positively established identity facts. Empty identity
// values mean unknown; names, paths, backend kinds, and URLs are not identities.
type Evidence struct {
	// HostID is a stable identity for the computer physically hosting a local
	// destination or a remote-host endpoint. Populate only from a trusted host
	// or fleet-node identity source.
	HostID string
	// DeviceID is a stable identity for the physical storage device. Populate
	// from the platform's device/volume identity resolver, not a mount path.
	DeviceID string
	// Endpoint says whether the actual storage endpoint is local, on a known
	// remote host, or explicitly provider-hosted. Backend kind (for example
	// S3) alone must leave this Unknown.
	Endpoint EndpointKind
}

// Relation is the conservative relationship between two destinations.
type Relation string

const (
	RelationUnknown     Relation = "unknown"
	RelationSameDevice  Relation = "same_device"
	RelationSameHost    Relation = "same_host"
	RelationIndependent Relation = "independent"
)

// Compare returns the strongest relationship proven by the supplied facts.
// Independent means distinct computer failure domains; same-host storage is
// never considered independent, even when devices differ.
func Compare(a, b Evidence) Relation {
	if known(a.HostID) && known(b.HostID) {
		if a.HostID == b.HostID {
			if known(a.DeviceID) && known(b.DeviceID) && a.DeviceID == b.DeviceID {
				return RelationSameDevice
			}
			return RelationSameHost
		}
		return RelationIndependent
	}

	// A provider-hosted endpoint is a separate physical failure domain only
	// when the owner has evidence it is hosted by the provider, rather than a
	// local S3-compatible service. Its backend name is not sufficient evidence.
	if isProviderHosted(a) && isLocalHost(b) || isProviderHosted(b) && isLocalHost(a) {
		return RelationIndependent
	}
	return RelationUnknown
}

func isProviderHosted(e Evidence) bool { return e.Endpoint == EndpointProviderHosted }
func isLocalHost(e Evidence) bool      { return e.Endpoint == EndpointLocal && known(e.HostID) }
func known(s string) bool              { return s != "" }
