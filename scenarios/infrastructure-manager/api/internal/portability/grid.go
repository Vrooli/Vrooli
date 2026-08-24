package portability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/deployability"
	"github.com/vrooli/vrooli/packages/hostreq"
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
	SituationControlsUnported CapabilitySituation = "controls_unported"
	SituationScopedOut        CapabilitySituation = "scoped_out"
)

// Situations returns the closed situation vocabulary.
func Situations() []CapabilitySituation {
	return []CapabilitySituation{
		SituationBuiltEverywhere,
		SituationNoWorkRequired,
		SituationNoEquivalentEver,
		SituationPeerNobodyWired,
		SituationControlsUnported,
		SituationScopedOut,
	}
}

// PlatformEntry is one cell of the grid: one capability on one host OS. It
// carries the resolution status and the honesty qualification separately,
// because a cross-compiled implementation and one proven on real hardware both
// resolve as implemented and are not the same claim.
type PlatformEntry struct {
	HostOS                      deployability.HostOS                     `json:"host_os"`
	Architecture                string                                   `json:"architecture"`
	Status                      deployability.CapabilityResolutionStatus `json:"status"`
	Qualification               deployability.Qualification              `json:"qualification"`
	ObservedQualification       deployability.Qualification              `json:"observed_qualification"`
	ObservedQualificationReason string                                   `json:"observed_qualification_reason"`
	Implementer                 string                                   `json:"implementer,omitempty"`
	Mechanism                   string                                   `json:"mechanism,omitempty"`
	Reason                      string                                   `json:"reason"`
	QualificationReason         string                                   `json:"qualification_reason"`
	HasImplementation           bool                                     `json:"has_implementation"`
	Controls                    []string                                 `json:"controls,omitempty"`
	Absent                      []string                                 `json:"absent,omitempty"`
	AbsentControls              []string                                 `json:"absent_controls,omitempty"`
	AbsentProviders             []string                                 `json:"absent_providers,omitempty"`
	Declarers                   []deployability.CapabilityDeclarer       `json:"declarers,omitempty"`
	ObservedDeclarers           []ObservedDeclarer                       `json:"observed_declarers,omitempty"`
}

// ResourceArchitectureClaim exposes resource profile architecture declarations
// in the same readout as capability portability.
type ResourceArchitectureClaim struct {
	Resource      string               `json:"resource"`
	HostOS        deployability.HostOS `json:"host_os"`
	Architectures []string             `json:"architectures,omitempty"`
	Support       string               `json:"support"`
	Mismatch      bool                 `json:"mismatch"`
	Reason        string               `json:"reason,omitempty"`
}

// ObservedDeclarer is the host-side evidence for one declared provider or
// control. It is intentionally separate from CapabilityDeclarer: declaration
// resolution is deterministic across hosts, while observation is only valid
// for the host sampled at read time.
type ObservedDeclarer struct {
	Name          string                      `json:"name"`
	State         string                      `json:"state"`
	Qualification deployability.Qualification `json:"qualification"`
	Reason        string                      `json:"reason"`
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

func (e Entry) PlatformFor(hostOS deployability.HostOS, architecture string) (PlatformEntry, bool) {
	for _, platform := range e.Platforms {
		if platform.HostOS == hostOS && platform.Architecture == architecture {
			return platform, true
		}
	}
	return PlatformEntry{}, false
}

// Grid is the whole capability readout. ManifestRoot is part of the readout,
// not metadata about it: a grid is only meaningful against a named tree.
type Grid struct {
	Capabilities       []Entry                     `json:"capabilities"`
	ManifestRoot       string                      `json:"manifest_root"`
	ManifestsRead      int                         `json:"manifests_read"`
	ComputedAt         time.Time                   `json:"computed_at"`
	ObservedSafeguards []hostreq.ObservedSafeguard `json:"observed_safeguards,omitempty"`
	Resources          []ResourceArchitectureClaim `json:"resources,omitempty"`
	NativeEvidence     []NativeEvidence            `json:"native_evidence,omitempty"`
}

// NativeEvidence is runner-produced proof about a target OS/architecture.
type NativeEvidence struct {
	HostOS       deployability.HostOS `json:"host_os"`
	Architecture string               `json:"architecture"`
	Commit       string               `json:"commit,omitempty"`
	GeneratedAt  time.Time            `json:"generated_at"`
	Passed       bool                 `json:"passed"`
	Source       string               `json:"source"`
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
	architectures := []string{"amd64", "arm64"}
	evidence, err := r.NativeEvidence()
	if err != nil {
		return Grid{}, err
	}

	grid := Grid{
		Capabilities:  make([]Entry, 0, len(vocabulary.Capabilities)),
		ManifestRoot:  r.root,
		ManifestsRead: len(manifests),
		ComputedAt:    now.UTC(),
	}
	for _, capability := range vocabulary.Capabilities {
		entry := Entry{Capability: capability, Platforms: make([]PlatformEntry, 0, len(operatingSystems)*len(architectures))}
		statuses := make(map[deployability.HostOS]deployability.CapabilityResolutionStatus, len(operatingSystems))
		for _, hostOS := range operatingSystems {
			resolution := deployability.ResolveCapability(implementations, capability, hostOS)
			statuses[hostOS] = resolution.Status
			for _, architecture := range architectures {
				if resolution.Qualification == deployability.QualificationQualified && hostOS != deployability.HostOSLinux && !hasNativeEvidence(evidence, hostOS, architecture) {
					return Grid{}, fmt.Errorf("qualified portability claim for %s/%s has no host-sampled native evidence", hostOS, architecture)
				}
				entry.Platforms = append(entry.Platforms, PlatformEntry{
					HostOS:                      hostOS,
					Architecture:                architecture,
					Status:                      resolution.Status,
					Qualification:               resolution.Qualification,
					ObservedQualification:       deployability.QualificationUndeclared,
					ObservedQualificationReason: "host observation is not attached to a manifest-only grid read",
					Implementer:                 resolution.Implementer,
					Mechanism:                   resolution.Mechanism,
					Reason:                      resolution.Reason,
					QualificationReason:         resolution.Qualification.Reason(),
					HasImplementation:           resolution.Status.HasImplementation(),
					Controls:                    resolution.Controls,
					Absent:                      resolution.Absent,
					AbsentControls:              resolution.AbsentControls,
					AbsentProviders:             resolution.AbsentProviders,
					Declarers:                   resolution.Declarers,
				})
			}
		}
		entry.Situation, entry.SituationReason, err = classifySituation(capability, statuses, vocabulary.PlatformPolicies)
		if err != nil {
			return Grid{}, err
		}
		grid.Capabilities = append(grid.Capabilities, entry)
	}
	grid.Resources, err = r.ResourceArchitectureClaims()
	if err != nil {
		return Grid{}, err
	}
	grid.NativeEvidence = evidence
	return grid, nil
}

func hasNativeEvidence(evidence []NativeEvidence, hostOS deployability.HostOS, architecture string) bool {
	for _, item := range evidence {
		if item.HostOS == hostOS && item.Architecture == architecture && item.Passed {
			return true
		}
	}
	return false
}

func (r *Reader) NativeEvidence() ([]NativeEvidence, error) {
	paths, err := filepath.Glob(filepath.Join(r.root, ".vrooli", "evidence", "native-platform", "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	result := make([]NativeEvidence, 0, len(paths))
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read native evidence %s: %w", path, readErr)
		}
		var raw struct {
			RunnerOS    string            `json:"runnerOS"`
			RunnerArch  string            `json:"runnerArch"`
			GoArch      string            `json:"goArch"`
			Commit      string            `json:"commit"`
			GeneratedAt time.Time         `json:"generatedAt"`
			Outcomes    map[string]string `json:"outcomes"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode native evidence %s: %w", path, err)
		}
		hostOS, ok := normalizeHostOS(raw.RunnerOS)
		if !ok {
			continue
		}
		arch := raw.GoArch
		if arch == "" {
			arch = raw.RunnerArch
		}
		passed := len(raw.Outcomes) > 0
		for _, outcome := range raw.Outcomes {
			if outcome != "success" && outcome != "passed" {
				passed = false
			}
		}
		result = append(result, NativeEvidence{HostOS: hostOS, Architecture: arch, Commit: raw.Commit, GeneratedAt: raw.GeneratedAt, Passed: passed, Source: "ci/unit-health-native-evidence"})
	}
	return result, nil
}

func (r *Reader) ResourceArchitectureClaims() ([]ResourceArchitectureClaim, error) {
	resources, err := r.Resources()
	if err != nil {
		return nil, err
	}
	claims := make([]ResourceArchitectureClaim, 0)
	for name, resource := range resources {
		for osName, support := range resource.Platforms {
			hostOS, ok := normalizeHostOS(osName)
			if !ok {
				continue
			}
			profile := resource.Deployment.Profiles["desktop"][osName]
			architectures := append([]string(nil), profile.Architectures...)
			sort.Strings(architectures)
			mismatch := support != "unsupported" && len(architectures) == 0
			reason := "resource profile architecture declaration agrees with the platform claim"
			if mismatch {
				reason = "resource platform is declared available but its deployment profile declares no architectures"
			}
			claims = append(claims, ResourceArchitectureClaim{Resource: name, HostOS: hostOS, Architectures: architectures, Support: support, Mismatch: mismatch, Reason: reason})
		}
	}
	sort.Slice(claims, func(i, j int) bool {
		if claims[i].Resource != claims[j].Resource {
			return claims[i].Resource < claims[j].Resource
		}
		return claims[i].HostOS < claims[j].HostOS
	})
	for _, claim := range claims {
		if claim.Mismatch {
			return claims, fmt.Errorf("resource %s/%s has contradictory platform and architecture claims: %s", claim.Resource, claim.HostOS, claim.Reason)
		}
	}
	return claims, nil
}

// AttachObservedQualifications joins the control-plane's read-only safeguard
// observations onto an already resolved grid. It never changes Status or the
// declaration Qualification. A non-local platform is explicitly marked
// host_not_sampled so absence of a host observation cannot look like failure.
func AttachObservedQualifications(grid Grid, observed []hostreq.ObservedSafeguard, hostOS deployability.HostOS) Grid {
	byName := make(map[string]hostreq.ObservedSafeguard, len(observed))
	for _, item := range observed {
		byName[item.Name] = item
	}
	for entryIndex := range grid.Capabilities {
		for platformIndex := range grid.Capabilities[entryIndex].Platforms {
			platform := &grid.Capabilities[entryIndex].Platforms[platformIndex]
			platform.ObservedQualification = deployability.QualificationUndeclared
			platform.ObservedQualificationReason = "host_not_sampled: this platform was not observed on the current host"
			if platform.HostOS != hostOS {
				continue
			}
			platform.ObservedQualificationReason = "no safeguard observation applies to this capability"
			observedRows := make([]ObservedDeclarer, 0, len(platform.Declarers))
			for _, declarer := range platform.Declarers {
				item, ok := byName[declarer.Name]
				if !ok {
					observedRows = append(observedRows, ObservedDeclarer{Name: declarer.Name, State: "host_not_sampled", Qualification: deployability.QualificationUndeclared, Reason: "no control-plane observation exists for this declarer"})
					continue
				}
				qualification, reason := observedQualification(item.ExecutionState)
				observedRows = append(observedRows, ObservedDeclarer{Name: declarer.Name, State: item.ExecutionState, Qualification: qualification, Reason: reason})
				if qualification.Rank() < platform.ObservedQualification.Rank() || platform.ObservedQualification == deployability.QualificationUndeclared {
					platform.ObservedQualification = qualification
					platform.ObservedQualificationReason = reason
				}
			}
			platform.ObservedDeclarers = observedRows
		}
	}
	return grid
}

func observedQualification(state string) (deployability.Qualification, string) {
	switch strings.TrimSpace(state) {
	case "already_present", "applied", "installed":
		return deployability.QualificationQualified, "control-plane observed the safeguard present or applied"
	case "not_applicable":
		return deployability.QualificationIneligible, "control-plane observed the safeguard as not applicable"
	case "unsupported":
		return deployability.QualificationIneligible, "control-plane observed the safeguard as unsupported"
	case "failed", "manual_action_required", "reboot_required":
		return deployability.QualificationDegraded, "control-plane observed an unresolved safeguard action"
	default:
		return deployability.QualificationUnqualified, "control-plane observed the safeguard but it is not yet applied"
	}
}

func currentHostOS() deployability.HostOS {
	switch runtime.GOOS {
	case "darwin":
		return deployability.HostOSMacOS
	case "windows":
		return deployability.HostOSWindows
	default:
		return deployability.HostOSLinux
	}
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
		for declaredOS, declaration := range item.PlatformStatus {
			if hostOS, ok := normalizeHostOS(declaredOS); ok {
				platforms[hostOS] = deployability.PlatformDeclaration{Status: declaration.Status, Mechanism: declaration.Mechanism, Evidence: declaration.Evidence}
			}
		}
		for declaredOS, declaration := range item.PlatformDeclarations {
			if hostOS, ok := normalizeHostOS(declaredOS); ok {
				platforms[hostOS] = deployability.PlatformDeclaration{Status: declaration.Status, Mechanism: declaration.Mechanism, Evidence: declaration.Evidence}
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
		platforms := make(map[string]deployability.PlatformDeclaration, len(item.PlatformStatus)+len(item.PlatformDeclarations))
		for osName, declaration := range item.PlatformStatus {
			platforms[osName] = deployability.PlatformDeclaration{
				Status: declaration.Status, Mechanism: declaration.Mechanism, Evidence: declaration.Evidence,
			}
		}
		for osName, declaration := range item.PlatformDeclarations {
			platforms[osName] = deployability.PlatformDeclaration{
				Status: declaration.Status, Mechanism: declaration.Mechanism, Evidence: declaration.Evidence,
			}
		}
		declarations = append(declarations, deployability.ManifestDeclaration{
			Path: item.Path, Name: item.Name, Capability: item.Capability,
			Role: item.Role, Platforms: platforms,
		})
	}
	return declarations
}

var situationByStatus = map[deployability.CapabilityResolutionStatus]CapabilitySituation{
	deployability.CapabilityImplemented:        SituationBuiltEverywhere,
	deployability.CapabilityDegraded:           SituationBuiltEverywhere,
	deployability.CapabilityIneligible:         SituationScopedOut,
	deployability.CapabilityUnwired:            SituationPeerNobodyWired,
	deployability.CapabilityPeerless:           SituationControlsUnported,
	deployability.CapabilityStatusInvalid:      SituationControlsUnported,
	deployability.CapabilityControlsIncomplete: SituationControlsUnported,
}

func classifySituation(capability string, statuses map[deployability.HostOS]deployability.CapabilityResolutionStatus, policies map[string]map[string]string) (CapabilitySituation, string, error) {
	if len(statuses) == 0 {
		return "", "", fmt.Errorf("capability %q has no resolution statuses", capability)
	}
	for hostOS, status := range statuses {
		if _, ok := situationByStatus[status]; !ok {
			return "", "", fmt.Errorf("capability %q has unmapped resolution status %q for %s", capability, status, hostOS)
		}
	}
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
				return "", "", fmt.Errorf("capability %q policy contains an unsupported value for %s", capability, hostOS)
			}
		}
		if noEquivalent {
			return SituationNoEquivalentEver, "the capability policy declares no equivalent for at least one OS or architecture", nil
		}
		allImplemented := true
		for _, status := range statuses {
			if !status.HasImplementation() {
				allImplemented = false
				break
			}
		}
		if noWork && allImplemented {
			return SituationNoWorkRequired, "the capability policy declares the host OS mechanism native and requiring no setup", nil
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
		return SituationBuiltEverywhere, "a declared implementation resolves on linux, macos, and windows", nil
	}
	for _, hostOS := range operatingSystems {
		if status, ok := statuses[hostOS]; ok && situationByStatus[status] == SituationScopedOut {
			return SituationScopedOut, "the capability is explicitly scoped out on " + string(hostOS), nil
		}
	}
	for _, hostOS := range operatingSystems {
		if status, ok := statuses[hostOS]; ok && situationByStatus[status] == SituationControlsUnported {
			return SituationControlsUnported, "required controls or implementation are not wired on " + string(hostOS), nil
		}
	}
	for _, hostOS := range operatingSystems {
		if status, ok := statuses[hostOS]; ok && situationByStatus[status] == SituationPeerNobodyWired {
			return SituationPeerNobodyWired, "a mechanism is named for an unsupported host OS, but no peer implementation is wired", nil
		}
	}
	return "", "", fmt.Errorf("capability %q has no situation despite %d resolved statuses", capability, len(statuses))
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
