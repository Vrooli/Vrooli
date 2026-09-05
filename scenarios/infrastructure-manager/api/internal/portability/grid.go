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

	"github.com/vrooli/binaryfetch"
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
	Evidence                    *deployability.Evidence                  `json:"evidence,omitempty"`
	ObservedQualification       deployability.Qualification              `json:"observed_qualification"`
	ObservedQualificationReason string                                   `json:"observed_qualification_reason"`
	Implementer                 string                                   `json:"implementer,omitempty"`
	Mechanism                   string                                   `json:"mechanism,omitempty"`
	Since                       string                                   `json:"since,omitempty"`
	ReviewBy                    string                                   `json:"review_by,omitempty"`
	AgeDays                     int                                      `json:"age_days,omitempty"`
	Reason                      string                                   `json:"reason"`
	QualificationReason         string                                   `json:"qualification_reason"`
	Policy                      string                                   `json:"policy,omitempty"`
	HasImplementation           bool                                     `json:"has_implementation"`
	Controls                    []string                                 `json:"controls,omitempty"`
	Absent                      []string                                 `json:"absent,omitempty"`
	AbsentControls              []string                                 `json:"absent_controls,omitempty"`
	AbsentProviders             []string                                 `json:"absent_providers,omitempty"`
	Declarers                   []deployability.CapabilityDeclarer       `json:"declarers,omitempty"`
	ObservedDeclarers           []ObservedDeclarer                       `json:"observed_declarers,omitempty"`
}

// ResourceArchitectureClaim exposes one resource's complete architecture
// surface in the same readout as capability portability. The outer entry is
// deliberately one-per-resource; a flat OS row makes a 29-resource inventory
// look like an arbitrary 87-row measurement and drops the resource's driver.
type ResourceArchitectureClaim struct {
	Name            string                  `json:"name"`
	Driver          string                  `json:"driver"`
	AcquisitionKind string                  `json:"acquisitionKind"`
	Platforms       []ResourcePlatformClaim `json:"platforms"`
}

// ResourcePlatformClaim is one resource/host-OS row. Architectures contains
// both declared architecture cells, including explicit unsupported cells, so
// absence of a profile cannot be mistaken for an omitted measurement.
type ResourcePlatformClaim struct {
	HostOS        deployability.HostOS         `json:"host_os"`
	Support       string                       `json:"support"`
	Architectures []ResourceArchitectureStatus `json:"architectures"`
	Mismatch      bool                         `json:"mismatch"`
	Reason        string                       `json:"reason,omitempty"`
}

type ResourceArchitectureStatus struct {
	Architecture          string                      `json:"architecture"`
	Support               string                      `json:"support"`
	Qualification         deployability.Qualification `json:"qualification"`
	Evidence              *QualificationObservation   `json:"evidence,omitempty"`
	AgeDays               int                         `json:"age_days,omitempty"`
	Aged                  bool                        `json:"aged,omitempty"`
	AcquisitionResolvable bool                        `json:"acquisition_resolvable"`
	AcquisitionReason     string                      `json:"acquisition_reason,omitempty"`
	Reason                string                      `json:"reason,omitempty"`
}

// QualificationObservation is proof from a real node that a resource started
// and passed its health contract. Declarations and cross-compiles never create
// one of these records.
type QualificationObservation struct {
	Resource     string    `json:"resource"`
	HostOS       string    `json:"host_os"`
	Architecture string    `json:"architecture"`
	Node         string    `json:"node"`
	RunID        string    `json:"run_id"`
	ObservedAt   time.Time `json:"observed_at"`
	HealthPassed bool      `json:"health_passed"`
}

// SkipBudget is the measured platform-gated test-skip surface. Available is
// explicit because a missing or unreadable source must never look like zero.
type SkipBudget struct {
	Available           bool                         `json:"available"`
	Measured            int                          `json:"measured"`
	Budgets             map[deployability.HostOS]int `json:"budgets,omitempty"`
	RatchetDirection    string                       `json:"ratchetDirection,omitempty"`
	LastRunWithinBudget bool                         `json:"lastRunWithinBudget"`
	Reason              string                       `json:"reason,omitempty"`
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
	Capability         string              `json:"capability"`
	Situation          CapabilitySituation `json:"situation"`
	SituationReason    string              `json:"situation_reason"`
	Platforms          []PlatformEntry     `json:"platforms"`
	PlatformSituations []PlatformSituation `json:"platform_situations"`
}

// PlatformSituation explains which host OS contributes to a capability-level
// situation. The capability-level classification is intentionally lossy; this
// projection keeps the evidence needed to act on an open cell.
type PlatformSituation struct {
	HostOS    deployability.HostOS `json:"host_os"`
	Situation CapabilitySituation  `json:"situation"`
	Reason    string               `json:"reason"`
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
	SkipBudget         SkipBudget                  `json:"skipBudget"`
	NativeEvidence     []NativeEvidence            `json:"native_evidence,omitempty"`
}

// NativeEvidence is runner-produced proof about a target OS/architecture.
type NativeEvidence struct {
	// Kind distinguishes broad CI runner evidence from a capability-scoped
	// hardware validation record. Legacy runner manifests normalize to "ci".
	Kind         string               `json:"kind,omitempty"`
	HostOS       deployability.HostOS `json:"host_os"`
	Architecture string               `json:"architecture"`
	Commit       string               `json:"commit,omitempty"`
	GeneratedAt  time.Time            `json:"generated_at"`
	Passed       bool                 `json:"passed"`
	Source       string               `json:"source"`
	RunID        string               `json:"run_id,omitempty"`
	Host         string               `json:"host,omitempty"`
	Surface      string               `json:"surface,omitempty"`
	ArtifactURI  string               `json:"artifact_uri,omitempty"`
	Capabilities []string             `json:"capabilities,omitempty"`
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
	for _, item := range manifests {
		if pairBlock, ok := item.PlatformDeclarations["pairs"]; ok {
			for _, pair := range pairBlock.Pairs {
				declaration := deployability.PlatformPairDeclaration{
					Console: deployability.HostOS(pair.Console), Node: deployability.HostOS(pair.Node),
					PlatformDeclaration: deployability.PlatformDeclaration{Status: pair.Status, Mechanism: pair.Mechanism, Since: pair.Since, ReviewBy: pair.ReviewBy, Evidence: evidenceValue(pair.Evidence), EvidenceRaw: evidenceRawValue(pair.Evidence)},
				}
				if err := deployability.ValidatePlatformPairEvidence(r.root, item.Path, item.Name, declaration); err != nil {
					return Grid{}, err
				}
			}
		}
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
				cellResolution := resolution
				policy := platformPolicy(vocabulary.PlatformPolicies, capability, hostOS, architecture)
				cellResolution, controlPolicy := applyControlPolicies(cellResolution, vocabulary.ControlPolicies, vocabulary.ControlPolicyReasons, capability, hostOS, architecture)
				if failed, ok := latestFailedNativeEvidence(evidence, hostOS, architecture, capability); ok && cellResolution.Qualification == deployability.QualificationQualified {
					// A scheduled native failure is stronger current evidence than
					// an old supported declaration. Keep the implementation, but
					// decay the honesty rung so stale qualification cannot persist.
					cellResolution.Qualification = deployability.QualificationBuildVerified
					cellResolution.Evidence = nil
					cellResolution.Reason = fmt.Sprintf("native validation %s failed for %s/%s on %s", failed.RunID, hostOS, architecture, failed.Host)
				}
				if policy == string(SituationNoEquivalentEver) {
					declaration := cellResolution.Implementer
					if declaration == "" {
						declaration = "declaration"
					}
					mechanism := cellResolution.Mechanism
					if mechanism == "" {
						mechanism = "unnamed mechanism"
					}
					cellResolution.Status = deployability.CapabilityIneligible
					cellResolution.Qualification = deployability.QualificationIneligible
					cellResolution.Evidence = nil
					cellResolution.Reason = fmt.Sprintf("platform policy %q excludes %s/%s; %s %s is not wired on this platform", policy, hostOS, architecture, declaration, mechanism)
				}
				if cellResolution.Qualification == deployability.QualificationQualified && !cellResolution.Evidence.Complete() {
					return Grid{}, fmt.Errorf("qualified portability claim for %s/%s has no structured evidence naming a run", hostOS, architecture)
				}
				if cellResolution.Qualification == deployability.QualificationQualified && hostOS != deployability.HostOSLinux && !hasNativeEvidence(evidence, hostOS, architecture, capability) {
					return Grid{}, fmt.Errorf("qualified portability claim for %s/%s has no host-sampled native evidence", hostOS, architecture)
				}
				entry.Platforms = append(entry.Platforms, PlatformEntry{
					HostOS:                      hostOS,
					Architecture:                architecture,
					Status:                      cellResolution.Status,
					Qualification:               cellResolution.Qualification,
					Policy:                      strings.Join(nonEmptyPolicies(policy, controlPolicy), ","),
					Evidence:                    cellResolution.Evidence,
					ObservedQualification:       deployability.QualificationUndeclared,
					ObservedQualificationReason: "host observation is not attached to a manifest-only grid read",
					Implementer:                 cellResolution.Implementer,
					Mechanism:                   cellResolution.Mechanism,
					Since:                       cellResolution.Since,
					ReviewBy:                    cellResolution.ReviewBy,
					AgeDays:                     declarationAgeDays(cellResolution, now),
					Reason:                      cellResolution.Reason,
					QualificationReason:         cellResolution.Qualification.Reason(),
					HasImplementation:           cellResolution.Status.HasImplementation(),
					Controls:                    cellResolution.Controls,
					Absent:                      cellResolution.Absent,
					AbsentControls:              cellResolution.AbsentControls,
					AbsentProviders:             cellResolution.AbsentProviders,
					Declarers:                   cellResolution.Declarers,
				})
			}
		}
		entry.Situation, entry.SituationReason, err = classifySituation(capability, statuses, vocabulary.PlatformPolicies)
		if err != nil {
			return Grid{}, err
		}
		entry.PlatformSituations = make([]PlatformSituation, 0, len(operatingSystems))
		for _, hostOS := range operatingSystems {
			status := statuses[hostOS]
			platformSituation, reason := classifyPlatformSituation(capability, hostOS, status, vocabulary.PlatformPolicies)
			entry.PlatformSituations = append(entry.PlatformSituations, PlatformSituation{HostOS: hostOS, Situation: platformSituation, Reason: reason})
		}
		grid.Capabilities = append(grid.Capabilities, entry)
	}
	grid.Resources, err = r.resourceArchitectureClaimsAt(now)
	if err != nil {
		return Grid{}, err
	}
	grid.SkipBudget, err = r.ReadSkipBudget()
	if err != nil {
		return Grid{}, err
	}
	grid.NativeEvidence = evidence
	return grid, nil
}

// ReadSkipBudget reads the repository-owned skip measurement. An absent file
// is an unavailable measurement, not a healthy zero.
func (r *Reader) ReadSkipBudget() (SkipBudget, error) {
	path := filepath.Join(r.root, ".vrooli", "skip-budgets.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return SkipBudget{Reason: "skip budget source is unavailable: .vrooli/skip-budgets.json is absent"}, nil
	}
	if err != nil {
		return SkipBudget{}, fmt.Errorf("read skip budget %s: %w", path, err)
	}
	var raw struct {
		Measured int            `json:"measured_platform_gated_skips"`
		Budgets  map[string]int `json:"budgets"`
		Policy   struct {
			RatchetDirection string `json:"ratchet_direction"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return SkipBudget{}, fmt.Errorf("decode skip budget %s: %w", path, err)
	}
	result := SkipBudget{
		Available:        true,
		Measured:         raw.Measured,
		Budgets:          make(map[deployability.HostOS]int, len(raw.Budgets)),
		RatchetDirection: strings.TrimSpace(raw.Policy.RatchetDirection),
	}
	for hostOS, budget := range raw.Budgets {
		result.Budgets[deployability.HostOS(hostOS)] = budget
	}
	if len(result.Budgets) == 0 {
		result.Reason = "skip budget source has no per-OS budgets"
		return result, nil
	}
	result.LastRunWithinBudget = true
	for hostOS, budget := range result.Budgets {
		if result.Measured > budget {
			result.LastRunWithinBudget = false
			result.Reason = fmt.Sprintf("measured platform-gated skips (%d) exceed the %s budget (%d)", result.Measured, hostOS, budget)
			break
		}
	}
	if result.RatchetDirection == "" {
		result.Reason = "skip budget source does not declare a ratchet direction"
	}
	return result, nil
}

func platformPolicy(policies map[string]map[string]string, capability string, hostOS deployability.HostOS, architecture string) string {
	byOS := policies[capability]
	if byOS == nil {
		return ""
	}
	if value := strings.TrimSpace(byOS[string(hostOS)+"/"+architecture]); value != "" {
		return value
	}
	return strings.TrimSpace(byOS[string(hostOS)])
}

// applyControlPolicies turns an explicitly recorded control decision into a
// cell result. A no_work_required policy means the host OS owns that boundary
// natively; no_equivalent_ever makes the capability ineligible. An absent
// policy leaves the resolver's controls_incomplete result untouched.
func applyControlPolicies(resolution deployability.CapabilityResolution, policies map[string]map[string]map[string]string, reasons map[string]map[string]map[string]string, capability string, hostOS deployability.HostOS, architecture string) (deployability.CapabilityResolution, string) {
	if resolution.Status != deployability.CapabilityControlsIncomplete {
		return resolution, ""
	}
	byControl := policies[capability]
	if len(byControl) == 0 {
		return resolution, ""
	}
	remaining := make([]string, 0, len(resolution.AbsentControls))
	covered := make([]string, 0, len(resolution.AbsentControls))
	for _, control := range resolution.AbsentControls {
		policy := controlPolicy(byControl, control, hostOS, architecture)
		if policy == string(SituationNoWorkRequired) || policy == string(SituationNoEquivalentEver) {
			covered = append(covered, control+"="+policy)
			continue
		}
		remaining = append(remaining, control)
	}
	if len(remaining) > 0 || len(covered) == 0 {
		return resolution, strings.Join(covered, ";")
	}
	for _, control := range resolution.AbsentControls {
		if controlPolicy(byControl, control, hostOS, architecture) == string(SituationNoEquivalentEver) {
			resolution.Status = deployability.CapabilityIneligible
			resolution.Qualification = deployability.QualificationIneligible
			resolution.Implementer = ""
			resolution.Mechanism = ""
			resolution.Evidence = nil
			resolution.Reason = fmt.Sprintf("control policy excludes %s/%s: %s", hostOS, architecture, controlPolicyReason(reasons, capability, control, hostOS, architecture))
			resolution.AbsentControls = nil
			return resolution, strings.Join(covered, ";")
		}
	}
	resolution.Status = deployability.CapabilityImplemented
	if resolution.Qualification == deployability.QualificationDegraded {
		resolution.Status = deployability.CapabilityDegraded
	}
	resolution.AbsentControls = nil
	resolution.Reason = fmt.Sprintf("provider %q resolves; control policies cover %s", resolution.Implementer, strings.Join(covered, ", "))
	return resolution, strings.Join(covered, ";")
}

func controlPolicy(policies map[string]map[string]string, control string, hostOS deployability.HostOS, architecture string) string {
	byTarget := policies[control]
	if value := strings.TrimSpace(byTarget[string(hostOS)+"/"+architecture]); value != "" {
		return value
	}
	return strings.TrimSpace(byTarget[string(hostOS)])
}

func controlPolicyReason(reasons map[string]map[string]map[string]string, capability, control string, hostOS deployability.HostOS, architecture string) string {
	byControl := reasons[capability][control]
	if value := strings.TrimSpace(byControl[string(hostOS)+"/"+architecture]); value != "" {
		return value
	}
	return strings.TrimSpace(byControl[string(hostOS)])
}

func nonEmptyPolicies(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}

func hasNativeEvidence(evidence []NativeEvidence, hostOS deployability.HostOS, architecture, capability string) bool {
	for _, item := range evidence {
		if item.HostOS == hostOS && item.Architecture == architecture && item.Passed && evidenceAppliesTo(item, capability) {
			return true
		}
	}
	return false
}

func evidenceAppliesTo(item NativeEvidence, capability string) bool {
	if len(item.Capabilities) == 0 {
		return true
	}
	for _, candidate := range item.Capabilities {
		if candidate == capability {
			return true
		}
	}
	return false
}

func latestFailedNativeEvidence(evidence []NativeEvidence, hostOS deployability.HostOS, architecture, capability string) (NativeEvidence, bool) {
	var latest NativeEvidence
	found := false
	for _, item := range evidence {
		if item.HostOS != hostOS || item.Architecture != architecture || !evidenceAppliesTo(item, capability) {
			continue
		}
		if !found || item.GeneratedAt.After(latest.GeneratedAt) {
			latest = item
			found = true
		}
	}
	return latest, found && !latest.Passed
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
			Kind         string            `json:"kind"`
			RunnerOS     string            `json:"runnerOS"`
			RunnerArch   string            `json:"runnerArch"`
			GoOS         string            `json:"goOS"`
			GoArch       string            `json:"goArch"`
			HostOS       string            `json:"host_os"`
			Architecture string            `json:"architecture"`
			Commit       string            `json:"commit"`
			GeneratedAt  time.Time         `json:"generatedAt"`
			GeneratedAt2 time.Time         `json:"generated_at"`
			Passed       *bool             `json:"passed"`
			Source       string            `json:"source"`
			RunID        string            `json:"run_id"`
			Host         string            `json:"host"`
			Surface      string            `json:"surface"`
			ArtifactURI  string            `json:"artifact_uri"`
			Capabilities []string          `json:"capabilities"`
			Outcomes     map[string]string `json:"outcomes"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode native evidence %s: %w", path, err)
		}
		if raw.Kind == "hardware-persistence" || raw.HostOS != "" {
			hostOS, ok := normalizeHostOS(raw.HostOS)
			if !ok {
				continue
			}
			arch := raw.Architecture
			if arch == "" {
				arch = raw.GoArch
			}
			generatedAt := raw.GeneratedAt2
			if generatedAt.IsZero() {
				generatedAt = raw.GeneratedAt
			}
			passed := raw.Passed != nil && *raw.Passed
			kind := raw.Kind
			if kind == "" {
				kind = "hardware-persistence"
			}
			result = append(result, NativeEvidence{Kind: kind, HostOS: hostOS, Architecture: arch, Commit: raw.Commit, GeneratedAt: generatedAt, Passed: passed, Source: raw.Source, RunID: raw.RunID, Host: raw.Host, Surface: raw.Surface, ArtifactURI: raw.ArtifactURI, Capabilities: raw.Capabilities})
			continue
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
		result = append(result, NativeEvidence{Kind: "ci", HostOS: hostOS, Architecture: arch, Commit: raw.Commit, GeneratedAt: raw.GeneratedAt, Passed: passed, Source: "ci/unit-health-native-evidence"})
	}
	return result, nil
}

func (r *Reader) ResourceArchitectureClaims() ([]ResourceArchitectureClaim, error) {
	return r.resourceArchitectureClaimsAt(time.Now().UTC())
}

func (r *Reader) resourceArchitectureClaimsAt(now time.Time) ([]ResourceArchitectureClaim, error) {
	resources, err := r.Resources()
	if err != nil {
		return nil, err
	}
	observations := r.qualificationObservations()
	claims := make([]ResourceArchitectureClaim, 0, len(resources))
	for name, resource := range resources {
		claim := ResourceArchitectureClaim{
			Name:            name,
			Driver:          resource.Driver,
			AcquisitionKind: resourceAcquisitionKind(resource),
			Platforms:       make([]ResourcePlatformClaim, 0, len(operatingSystems)),
		}
		for _, hostOS := range operatingSystems {
			osName := string(hostOS)
			support := strings.TrimSpace(resource.Platforms[osName])
			if support == "" {
				support = "unsupported"
			}
			profile := resource.Deployment.Profiles["desktop"][osName]
			declared := make(map[string]struct{}, len(profile.Architectures))
			for _, architecture := range profile.Architectures {
				declared[strings.TrimSpace(architecture)] = struct{}{}
			}
			mismatch := support != "unsupported" && len(declared) == 0
			platform := ResourcePlatformClaim{
				HostOS:        hostOS,
				Support:       support,
				Architectures: make([]ResourceArchitectureStatus, 0, len(resourceArchitectures)),
				Mismatch:      mismatch,
				Reason:        "resource profile architecture declarations agree with the platform claim",
			}
			if mismatch {
				platform.Reason = "resource platform is declared available but its deployment profile declares no architectures"
			}
			for _, architecture := range resourceArchitectures {
				architectureSupport := "unsupported"
				architectureReason := "resource deployment profile does not declare this architecture"
				if _, ok := declared[architecture]; ok {
					architectureSupport = support
					if strings.TrimSpace(profile.Support) != "" {
						architectureSupport = strings.TrimSpace(profile.Support)
					}
					architectureReason = "resource deployment profile declares this architecture"
				}
				cell := ResourceArchitectureStatus{Architecture: architecture, Support: architectureSupport, Qualification: qualificationForResourceSupport(architectureSupport), Reason: architectureReason}
				if _, declaredArchitecture := declared[architecture]; !declaredArchitecture {
					cell.AcquisitionResolvable = true
					cell.AcquisitionReason = "architecture is not declared by the resource profile"
				} else {
					cell.AcquisitionResolvable, cell.AcquisitionReason = resourceAcquisitionResolves(resource, osName, architecture)
				}
				for _, observation := range observations {
					if observation.Resource == name && string(observation.HostOS) == osName && observation.Architecture == architecture {
						cell.Evidence = &observation
						cell.AgeDays = max(0, int(now.Sub(observation.ObservedAt).Hours()/24))
						cell.Aged = cell.AgeDays > 30
						if observation.HealthPassed {
							cell.Qualification = deployability.QualificationQualified
						} else {
							cell.Qualification = deployability.QualificationBuildVerified
						}
						break
					}
				}
				platform.Architectures = append(platform.Architectures, cell)
			}
			claim.Platforms = append(claim.Platforms, platform)
		}
		claims = append(claims, claim)
	}
	sort.Slice(claims, func(i, j int) bool {
		return claims[i].Name < claims[j].Name
	})
	for _, claim := range claims {
		for _, platform := range claim.Platforms {
			if platform.Mismatch {
				return claims, fmt.Errorf("resource %s/%s has contradictory platform and architecture claims: %s", claim.Name, platform.HostOS, platform.Reason)
			}
		}
	}
	return claims, nil
}

func qualificationForResourceSupport(support string) deployability.Qualification {
	if strings.EqualFold(strings.TrimSpace(support), "unsupported") {
		return deployability.QualificationIneligible
	}
	return deployability.QualificationBuildVerified
}

func resourceAcquisitionResolves(resource ResourceInput, osName, architecture string) (bool, string) {
	support := strings.TrimSpace(resource.Platforms[osName])
	if support == "" || strings.EqualFold(support, "unsupported") {
		return true, "platform is unsupported; no acquisition target is required"
	}
	var acquisition *ResourceAcquisitionInput
	if resource.Acquisition != nil {
		acquisition = resource.Acquisition
	} else if resource.ManagedService != nil {
		acquisition = &resource.ManagedService.Acquisition
	}
	if acquisition == nil || len(acquisition.Targets) == 0 {
		return false, "declared platform has no acquisition targets"
	}
	_, err := (binaryfetch.Acquisition{Kind: acquisition.Kind, Targets: acquisition.Targets}).Resolve(binaryfetch.Facts{"os": osName, "arch": architecture})
	if err != nil {
		return false, fmt.Sprintf("no acquisition target resolves for %s/%s: %v", osName, architecture, err)
	}
	return true, "acquisition target resolves for declared OS and architecture"
}

const (
	resourceArchitectureAMD64 = "amd64"
	resourceArchitectureARM64 = "arm64"
)

var resourceArchitectures = []string{resourceArchitectureAMD64, resourceArchitectureARM64}

func resourceAcquisitionKind(resource ResourceInput) string {
	if resource.ManagedService != nil {
		if kind := strings.TrimSpace(resource.ManagedService.Acquisition.Kind); kind != "" {
			return kind
		}
	}
	if resource.Acquisition != nil {
		if kind := strings.TrimSpace(resource.Acquisition.Kind); kind != "" {
			return kind
		}
	}
	return "none"
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
				platforms[hostOS] = deployability.PlatformDeclaration{Status: declaration.Status, Mechanism: declaration.Mechanism, Since: declaration.Since, ReviewBy: declaration.ReviewBy, Evidence: evidenceValue(declaration.Evidence)}
			}
		}
		for declaredOS, declaration := range item.PlatformDeclarations {
			if declaredOS == "pairs" {
				continue
			}
			if hostOS, ok := normalizeHostOS(declaredOS); ok {
				platforms[hostOS] = deployability.PlatformDeclaration{Status: declaration.Status, Mechanism: declaration.Mechanism, Since: declaration.Since, ReviewBy: declaration.ReviewBy, Evidence: evidenceValue(declaration.Evidence)}
			}
		}
		var pairs []deployability.PlatformPairDeclaration
		if pairBlock, ok := item.PlatformDeclarations["pairs"]; ok {
			for _, pair := range pairBlock.Pairs {
				pairs = append(pairs, deployability.PlatformPairDeclaration{
					Console: deployability.HostOS(pair.Console), Node: deployability.HostOS(pair.Node),
					PlatformDeclaration: deployability.PlatformDeclaration{Status: pair.Status, Mechanism: pair.Mechanism, Since: pair.Since, ReviewBy: pair.ReviewBy, Evidence: evidenceValue(pair.Evidence), EvidenceRaw: evidenceRawValue(pair.Evidence)},
				})
			}
		}
		implementations = append(implementations, deployability.CapabilityImplementation{
			Name: item.Name, Capability: item.Capability, Role: item.Role, Platforms: platforms, Pairs: pairs,
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
				Status: declaration.Status, Mechanism: declaration.Mechanism, Since: declaration.Since, ReviewBy: declaration.ReviewBy,
				Evidence: evidenceValue(declaration.Evidence), EvidenceRaw: string(declaration.Evidence),
			}
		}
		for osName, declaration := range item.PlatformDeclarations {
			if osName == "pairs" {
				continue
			}
			platforms[osName] = deployability.PlatformDeclaration{
				Status: declaration.Status, Mechanism: declaration.Mechanism, Since: declaration.Since, ReviewBy: declaration.ReviewBy,
				Evidence: evidenceValue(declaration.Evidence), EvidenceRaw: string(declaration.Evidence),
			}
		}
		var pairs []deployability.PlatformPairDeclaration
		if pairBlock, ok := item.PlatformDeclarations["pairs"]; ok {
			for _, pair := range pairBlock.Pairs {
				pairs = append(pairs, deployability.PlatformPairDeclaration{
					Console: deployability.HostOS(pair.Console), Node: deployability.HostOS(pair.Node),
					PlatformDeclaration: deployability.PlatformDeclaration{Status: pair.Status, Mechanism: pair.Mechanism, Since: pair.Since, ReviewBy: pair.ReviewBy, Evidence: evidenceValue(pair.Evidence), EvidenceRaw: string(pair.Evidence)},
				})
			}
		}
		declarations = append(declarations, deployability.ManifestDeclaration{
			Path: item.Path, Name: item.Name, Capability: item.Capability,
			Role: item.Role, Platforms: platforms, Pairs: pairs,
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
		for hostOS, value := range policy {
			if strings.Contains(hostOS, "/") {
				continue
			}
			if value != string(SituationNoWorkRequired) && value != string(SituationNoEquivalentEver) {
				return "", "", fmt.Errorf("capability %q policy contains an unsupported value for %s", capability, hostOS)
			}
		}
	}

	allImplemented := true
	allUnimplementedClosed := true
	noWork := false
	for hostOS, status := range statuses {
		platformSituation, _ := classifyPlatformSituation(capability, hostOS, status, policies)
		if platformSituation == SituationNoWorkRequired {
			noWork = true
		}
		if !status.HasImplementation() {
			allImplemented = false
			if platformSituation != SituationNoEquivalentEver {
				allUnimplementedClosed = false
			}
		}
	}
	if allUnimplementedClosed && !allImplemented {
		return SituationNoEquivalentEver, "every unimplemented host OS has an explicit no-equivalent policy", nil
	}
	if noWork && allImplemented {
		return SituationNoWorkRequired, "the capability policy declares the host OS mechanism native and requiring no setup", nil
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

func classifyPlatformSituation(capability string, hostOS deployability.HostOS, status deployability.CapabilityResolutionStatus, policies map[string]map[string]string) (CapabilitySituation, string) {
	policy := policies[capability]
	value := strings.TrimSpace(policy[string(hostOS)])
	switch value {
	case string(SituationNoEquivalentEver):
		if !status.HasImplementation() {
			return SituationNoEquivalentEver, "an explicit policy closes this host OS"
		}
	case string(SituationNoWorkRequired):
		if status.HasImplementation() {
			return SituationNoWorkRequired, "an explicit policy records a native host boundary"
		}
	}
	if status.HasImplementation() {
		return SituationBuiltEverywhere, "a provider or complete control set resolves on this host OS"
	}
	if situation, ok := situationByStatus[status]; ok {
		switch situation {
		case SituationPeerNobodyWired:
			return SituationPeerNobodyWired, "a named host mechanism has no wired implementation"
		case SituationScopedOut:
			return SituationScopedOut, "the capability is explicitly scoped out on this host OS"
		default:
			return situation, "required controls or implementation are not wired on this host OS"
		}
	}
	return SituationControlsUnported, fmt.Sprintf("capability %q has an unmapped resolution status %q", capability, status)
}

func declarationAgeDays(resolution deployability.CapabilityResolution, now time.Time) int {
	date := strings.TrimSpace(resolution.Since)
	if date == "" {
		return 0
	}
	started, err := time.Parse("2006-01-02", date)
	if err != nil || now.Before(started) {
		return 0
	}
	return int(now.UTC().Sub(started).Hours() / 24)
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
