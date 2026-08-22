package portability

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/deployability"
)

// CapabilitySituation classifies a whole capability row. The resolution status
// answers "does this run on that OS?"; the situation answers the question an
// operator actually asks — "is the absence a gap, or a decision?".
type CapabilitySituation string

const (
	SituationBuiltEverywhere  CapabilitySituation = "built_everywhere"
	SituationNoWorkRequired   CapabilitySituation = "no_work_required"
	SituationNoEquivalentEver CapabilitySituation = "no_equivalent_ever"
	SituationPeerNobodyWired  CapabilitySituation = "real_peer_nobody_wired"
)

// Situations returns the closed situation vocabulary.
func Situations() []CapabilitySituation {
	return []CapabilitySituation{
		SituationBuiltEverywhere,
		SituationNoWorkRequired,
		SituationNoEquivalentEver,
		SituationPeerNobodyWired,
	}
}

// PlatformEntry is one cell of the grid: one capability on one host OS. It
// carries the resolution status and the honesty qualification separately,
// because a cross-compiled implementation and one proven on real hardware both
// resolve as implemented and are not the same claim.
type PlatformEntry struct {
	HostOS              deployability.HostOS                     `json:"host_os"`
	Status              deployability.CapabilityResolutionStatus `json:"status"`
	Qualification       deployability.Qualification              `json:"qualification"`
	Implementer         string                                   `json:"implementer,omitempty"`
	Mechanism           string                                   `json:"mechanism,omitempty"`
	Reason              string                                   `json:"reason"`
	QualificationReason string                                   `json:"qualification_reason"`
	HasImplementation   bool                                     `json:"has_implementation"`
	Controls            []string                                 `json:"controls,omitempty"`
	Absent              []string                                 `json:"absent,omitempty"`
	Declarers           []deployability.CapabilityDeclarer       `json:"declarers,omitempty"`
}

// Entry is one capability row across every host OS.
type Entry struct {
	Capability      string              `json:"capability"`
	Situation       CapabilitySituation `json:"situation"`
	SituationReason string              `json:"situation_reason"`
	Platforms       []PlatformEntry     `json:"platforms"`
}

// Platform returns the row's entry for one host OS.
func (e Entry) Platform(hostOS deployability.HostOS) (PlatformEntry, bool) {
	for _, platform := range e.Platforms {
		if platform.HostOS == hostOS {
			return platform, true
		}
	}
	return PlatformEntry{}, false
}

// Grid is the whole capability readout. ManifestRoot is part of the readout,
// not metadata about it: a grid is only meaningful against a named tree.
type Grid struct {
	Capabilities  []Entry   `json:"capabilities"`
	ManifestRoot  string    `json:"manifest_root"`
	ManifestsRead int       `json:"manifests_read"`
	ComputedAt    time.Time `json:"computed_at"`
}

// Capability returns one row by name.
func (g Grid) Capability(name string) (Entry, bool) {
	name = strings.TrimSpace(name)
	for _, entry := range g.Capabilities {
		if entry.Capability == name {
			return entry, true
		}
	}
	return Entry{}, false
}

// Grid resolves every capability in the vocabulary against every host OS.
//
// Capabilities named in the vocabulary but implemented nowhere still appear:
// they resolve as peerless on all three platforms. Dropping them would make
// "nobody has built this" indistinguishable from "nobody has named this", and
// the vocabulary exists precisely to name the second case.
func (r *Reader) Grid(now time.Time) (Grid, error) {
	vocabulary, err := r.Vocabulary()
	if err != nil {
		return Grid{}, err
	}
	manifests, err := r.CapabilityManifests()
	if err != nil {
		return Grid{}, err
	}
	if err := deployability.ValidateManifestDeclarations(manifestDeclarations(manifests), vocabulary.Capabilities); err != nil {
		return Grid{}, err
	}
	implementations := capabilityImplementations(manifests)

	grid := Grid{
		Capabilities:  make([]Entry, 0, len(vocabulary.Capabilities)),
		ManifestRoot:  r.root,
		ManifestsRead: len(manifests),
		ComputedAt:    now.UTC(),
	}
	for _, capability := range vocabulary.Capabilities {
		entry := Entry{Capability: capability, Platforms: make([]PlatformEntry, 0, len(operatingSystems))}
		statuses := make(map[deployability.HostOS]deployability.CapabilityResolutionStatus, len(operatingSystems))
		for _, hostOS := range operatingSystems {
			resolution := deployability.ResolveCapability(implementations, capability, hostOS)
			statuses[hostOS] = resolution.Status
			entry.Platforms = append(entry.Platforms, PlatformEntry{
				HostOS:              hostOS,
				Status:              resolution.Status,
				Qualification:       resolution.Qualification,
				Implementer:         resolution.Implementer,
				Mechanism:           resolution.Mechanism,
				Reason:              resolution.Reason,
				QualificationReason: resolution.Qualification.Reason(),
				HasImplementation:   resolution.Status.HasImplementation(),
				Controls:            resolution.Controls,
				Absent:              resolution.Absent,
				Declarers:           resolution.Declarers,
			})
		}
		entry.Situation, entry.SituationReason = classifySituation(capability, statuses, vocabulary.PlatformPolicies)
		grid.Capabilities = append(grid.Capabilities, entry)
	}
	return grid, nil
}

// capabilityImplementations projects the authored manifests onto the pure
// resolver's declaration shape. Every host OS gets an explicit declaration:
// an OS a manifest does not claim is declared unsupported with whatever
// acquisition mechanism the manifest does carry, so the resolver can tell
// "unsupported, and nothing to wire" from "unsupported, but a mechanism
// exists".
func capabilityImplementations(manifests []Manifest) []deployability.CapabilityImplementation {
	implementations := make([]deployability.CapabilityImplementation, 0, len(manifests))
	for _, item := range manifests {
		platforms := make(map[deployability.HostOS]deployability.PlatformDeclaration, len(operatingSystems))
		for _, hostOS := range operatingSystems {
			platforms[hostOS] = deployability.PlatformDeclaration{
				Status:    string(deployability.StatusUnsupported),
				Mechanism: mechanism(item, hostOS),
			}
		}
		for _, declared := range item.Platforms {
			hostOS := deployability.HostOS(strings.TrimSpace(declared))
			if _, ok := platforms[hostOS]; ok {
				platforms[hostOS] = deployability.PlatformDeclaration{Status: string(deployability.StatusSupported)}
			}
		}
		for declaredOS, declaration := range item.PlatformDeclarations {
			if hostOS, ok := normalizeHostOS(declaredOS); ok {
				platforms[hostOS] = deployability.PlatformDeclaration{Status: declaration.Status, Mechanism: declaration.Mechanism}
			}
		}
		implementations = append(implementations, deployability.CapabilityImplementation{
			Name: item.Name, Capability: item.Capability, Role: item.Role, Platforms: platforms,
		})
	}
	return implementations
}

// manifestDeclarations projects the loaded manifests onto the resolver's
// loader-neutral validation shape.
func manifestDeclarations(manifests []Manifest) []deployability.ManifestDeclaration {
	declarations := make([]deployability.ManifestDeclaration, 0, len(manifests))
	for _, item := range manifests {
		platforms := make(map[string]deployability.PlatformDeclaration, len(item.PlatformDeclarations))
		for osName, declaration := range item.PlatformDeclarations {
			platforms[osName] = deployability.PlatformDeclaration{
				Status: declaration.Status, Mechanism: declaration.Mechanism,
			}
		}
		declarations = append(declarations, deployability.ManifestDeclaration{
			Path: item.Path, Name: item.Name, Capability: item.Capability,
			Role: item.Role, Platforms: platforms,
		})
	}
	return declarations
}

func classifySituation(capability string, statuses map[deployability.HostOS]deployability.CapabilityResolutionStatus, policies map[string]map[string]string) (CapabilitySituation, string) {
	if policy := policies[capability]; len(policy) > 0 {
		noEquivalent := false
		noWork := false
		for hostOS, value := range policy {
			switch value {
			case string(SituationNoWorkRequired):
				noWork = true
			case string(SituationNoEquivalentEver):
				noEquivalent = true
			default:
				return SituationNoEquivalentEver, "the capability policy contains an unsupported value for " + hostOS
			}
		}
		if noEquivalent {
			return SituationNoEquivalentEver, "the capability policy declares no equivalent for at least one OS or architecture"
		}
		if noWork {
			return SituationNoWorkRequired, "the capability policy declares the host OS mechanism native and requiring no setup"
		}
	}
	allImplemented := true
	for _, status := range statuses {
		if !status.HasImplementation() {
			allImplemented = false
			break
		}
	}
	if allImplemented {
		return SituationBuiltEverywhere, "a declared implementation resolves on linux, macos, and windows"
	}
	for _, status := range statuses {
		if status == deployability.CapabilityUnwired {
			return SituationPeerNobodyWired, "a mechanism is named for an unsupported host OS, but no peer implementation is wired"
		}
	}
	return SituationNoEquivalentEver, "no implementation or mechanism resolves on at least one host OS"
}

// mechanism names how this manifest would acquire its implementation. It is
// the difference between "unsupported and unwirable" and "unsupported but
// there is something to wire", which is what separates a dead end from work.
func mechanism(item Manifest, hostOS deployability.HostOS) string {
	if hostOS == deployability.HostOSMacOS && item.Packages["brew"] != nil {
		return "brew"
	}
	if rawValuePresent(item.Source) {
		return "source"
	}
	if rawValuePresent(item.Handler) {
		return "handler"
	}
	if item.Manual {
		return "manual"
	}
	return ""
}

func rawValuePresent(value json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(value))
	return trimmed != "" && trimmed != "null" && trimmed != `""` && trimmed != "{}"
}
