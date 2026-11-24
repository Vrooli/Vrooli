package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	goruntime "runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/clock"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/safeguards"
)

var ErrUnsupportedPlatform = hostreqkit.ErrUnsupportedPlatform

// These compatibility names remain next to the runtime operations while
// callers migrate to hostreqkit's canonical requirement result types.
type (
	SupportClass    = hostreqkit.SupportClass
	ExecutionState  = hostreqkit.ExecutionState
	ItemStatus      = hostreqkit.ItemStatus
	ToolStatus      = hostreqkit.ToolStatus
	SafeguardStatus = hostreqkit.SafeguardStatus
	Report          = hostreqkit.Report
)

const (
	SupportSupported              = hostreqkit.SupportSupported
	SupportUnsupported            = hostreqkit.SupportUnsupported
	SupportNotApplicable          = hostreqkit.SupportNotApplicable
	SupportManualOnly             = hostreqkit.SupportManualOnly
	ExecutionPending              = hostreqkit.ExecutionPending
	ExecutionAlreadyPresent       = hostreqkit.ExecutionAlreadyPresent
	ExecutionWouldInstall         = hostreqkit.ExecutionWouldInstall
	ExecutionWouldApply           = hostreqkit.ExecutionWouldApply
	ExecutionInstalled            = hostreqkit.ExecutionInstalled
	ExecutionApplied              = hostreqkit.ExecutionApplied
	ExecutionRebootRequired       = hostreqkit.ExecutionRebootRequired
	ExecutionManualActionRequired = hostreqkit.ExecutionManualActionRequired
	ExecutionUnsupported          = hostreqkit.ExecutionUnsupported
	ExecutionNotApplicable        = hostreqkit.ExecutionNotApplicable
	ExecutionFailed               = hostreqkit.ExecutionFailed
)

type Host = hostreqkit.Host

type EnsureOptions = hostreqkit.EnsureOptions

type PortabilityBacklogReport struct {
	NotImplemented int      `json:"not_implemented"`
	Total          int      `json:"total_safeguards"`
	Entries        []string `json:"entries,omitempty"`
}

// SafeguardPortabilityBacklog counts platform-status entries that are
// explicitly not implemented. not_applicable is closed work and is excluded.
func SafeguardPortabilityBacklog() (PortabilityBacklogReport, error) {
	manifests, err := safeguardManifests()
	if err != nil {
		return PortabilityBacklogReport{}, err
	}
	report := PortabilityBacklogReport{Total: len(manifests), Entries: []string{}}
	for _, manifest := range manifests {
		for platform, status := range manifest.PlatformStatus {
			if status.Status != "not_implemented" {
				continue
			}
			report.NotImplemented++
			report.Entries = append(report.Entries, manifest.Name+"/"+platform)
		}
	}
	slices.Sort(report.Entries)
	return report, nil
}

// SafeguardInvariantCoverage walks the embedded safeguard declaration sites so
// a count cannot hide a manifest that the evaluator forgot to visit.
func SafeguardInvariantCoverage() (hostreqkit.Coverage, error) {
	manifests, err := safeguardManifests()
	if err != nil {
		return hostreqkit.Coverage{}, err
	}
	sites := make([]hostreqkit.Site, 0, len(manifests))
	for _, manifest := range manifests {
		invariants := make([]hostreqkit.Invariant, 0, len(manifest.Invariants))
		for _, declaration := range manifest.Invariants {
			invariants = append(invariants, hostreqkit.Invariant{ID: declaration.ID, Kind: declaration.Kind, Statement: declaration.Statement, Severity: declaration.Severity, Applicability: declaration.Applicability})
		}
		sites = append(sites, hostreqkit.Site{Name: manifest.Name, Invariants: invariants, Walked: true})
	}
	registry := hostreqkit.NewRegistry(hostreqkit.AptProvider{}, hostreqkit.DarwinProvider{})
	coverage, _ := hostreqkit.EvaluateSites(context.Background(), registry, sites, hostreqkit.Facts{OS: goruntime.GOOS})
	return coverage, nil
}

// SafeguardHostStateAudit compares declared safeguard artifacts with the
// current host observation. The legacy NVIDIA policy path is accepted as an
// adoption-equivalent observation until the operator can install the owned
// replacement with privilege.
func SafeguardHostStateAudit() (hostinventory.AuditReport, error) {
	manifests, err := safeguardManifests()
	if err != nil {
		return hostinventory.AuditReport{}, err
	}
	var declared []hostinventory.DeclaredArtifact
	var observed []hostinventory.ObservedArtifact
	for _, manifest := range manifests {
		storageValue, ok := manifest.Storage["entries"].(map[string]any)
		if !ok {
			continue
		}
		for _, raw := range storageValue {
			entry, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			path, _ := entry["path"].(string)
			if path == "" {
				continue
			}
			declared = append(declared, hostinventory.DeclaredArtifact{Owner: manifest.Name, Path: path})
			if hash, hashErr := hostinventory.HashFile(path); hashErr == nil {
				observed = append(observed, hostinventory.ObservedArtifact{Path: path, ContentHash: hash})
				continue
			}
			if manifest.Name == "nvidia_driver" {
				if hash, hashErr := hostinventory.HashFile("/etc/apt/apt.conf.d/52unattended-upgrades-nvidia-blacklist"); hashErr == nil {
					observed = append(observed, hostinventory.ObservedArtifact{Path: path, ContentHash: hash})
				}
			}
		}
	}
	return hostinventory.AuditArtifacts(declared, observed), nil
}

func Current() Host {
	return currentHost()
}

func InspectRequirements(environment string, resolution hostreq.Resolution) (Report, error) {
	env := hostreqspec.NormalizeEnvironment(environment)
	return inspectResolution(Current(), env, resolution)
}

func EnsureRequirements(opts EnsureOptions, resolution hostreq.Resolution) (Report, error) {
	opts.Environment = hostreqspec.NormalizeEnvironment(opts.Environment)
	return ensureResolution(opts, resolution)
}

// EnsureTool inspects and (unless opts.DryRun) installs a single host tool by
// name through its registered runtime handler, returning the final status. It is
// the engine behind `vrooli host install <tool>`: it honors the capability gate
// (a not-applicable tool is returned as such, never installed) and the
// url/release fetch path (no sudo, into ~/.vrooli/bin). A tool with no
// registered handler comes back unsupported.
func EnsureTool(name string, opts EnsureOptions) (ItemStatus, error) {
	opts.Environment = hostreqspec.NormalizeEnvironment(opts.Environment)
	host := Current()
	requirement := hostreqspec.ResolvedRequirement{
		Name:     strings.TrimSpace(name),
		Kind:     hostreqspec.KindTool,
		Required: true,
	}
	status := inspectRequirement(host, requirement)
	if status.ExecutionState == hostreqkit.ExecutionAlreadyPresent {
		if reconciler, ok := lookupHandler(requirement.Kind, requirement.Name); ok == nil && reconciler != nil {
			if present, supported := reconciler.(interface {
				ReconcilePresent(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error)
			}); supported {
				updated, reconcileErr := present.ReconcilePresent(host, status, opts)
				if reconcileErr != nil {
					return status, reconcileErr
				}
				return annotateBlockingReason(updated), nil
			}
		}
		return status, nil
	}
	updated, err := applyRequirement(host, status, opts)
	if err != nil {
		return status, err
	}
	return annotateBlockingReason(updated), nil
}

// EnsureSafeguard applies one named host safeguard without running the wider
// project setup lifecycle. It is the narrow, auditable path for high-risk host
// state (kernel drivers, firewall rules, and similar) when an operator needs
// to repair one capability rather than re-run every setup requirement.
func EnsureSafeguard(name string, opts EnsureOptions) (ItemStatus, error) {
	name = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "-", "_")
	opts.Environment = hostreqspec.NormalizeEnvironment(opts.Environment)
	host := Current()
	root, err := os.Getwd()
	if err != nil {
		return ItemStatus{}, fmt.Errorf("resolve project root for safeguard %q: %w", name, err)
	}
	requirement, err := hostreq.ResolveSafeguard(root, name, host.OS)
	if err != nil {
		return ItemStatus{}, err
	}
	status := inspectRequirement(host, requirement)
	if requirementSatisfied(status) {
		return status, nil
	}
	updated, err := applyRequirement(host, status, opts)
	if err != nil {
		return status, err
	}
	return annotateBlockingReason(updated), nil
}

// InspectSafeguard performs the unprivileged read half of focused safeguard
// handling. It never calls Apply and is the control-plane boundary for
// observed host state consumers.
func InspectSafeguard(name string) (ItemStatus, error) {
	root, err := os.Getwd()
	if err != nil {
		return ItemStatus{}, fmt.Errorf("resolve project root for safeguard %q: %w", name, err)
	}
	return InspectSafeguardAt(root, name)
}

// InspectSafeguardAt performs the unprivileged read half of focused safeguard
// handling against an explicit repository root. It never calls Apply and is
// safe for typed consumers running from a scenario module directory.
func InspectSafeguardAt(root, name string) (ItemStatus, error) {
	name = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "-", "_")
	if name == "" {
		return ItemStatus{}, fmt.Errorf("safeguard name is required")
	}
	host := Current()
	requirement, err := hostreq.ResolveSafeguard(root, name, host.OS)
	if err != nil {
		return ItemStatus{}, err
	}
	return inspectRequirement(host, requirement), nil
}

// safeguardManifests decodes the embedded safeguard catalog. It performs no
// host probing: the manifests are compiled in, so a caller that needs one
// safeguard pays only for the JSON, not for thirty inspection handlers.
func safeguardManifests() ([]hostreqkit.SafeguardManifest, error) {
	dirs, err := fs.ReadDir(safeguards.Manifests, ".")
	if err != nil {
		return nil, fmt.Errorf("read safeguard catalog: %w", err)
	}
	manifests := make([]hostreqkit.SafeguardManifest, 0, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		data, readErr := fs.ReadFile(safeguards.Manifests, dir.Name()+"/safeguard.json")
		if readErr != nil {
			return nil, fmt.Errorf("read safeguard %s: %w", dir.Name(), readErr)
		}
		var manifest hostreqkit.SafeguardManifest
		if unmarshalErr := json.Unmarshal(data, &manifest); unmarshalErr != nil {
			return nil, fmt.Errorf("parse safeguard %s: %w", dir.Name(), unmarshalErr)
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}

// observeSafeguard samples one already-decoded manifest. Both the roster and
// the focused read go through here so the two can never report a different
// shape for the same safeguard.
func observeSafeguard(root string, manifest hostreqkit.SafeguardManifest, observedAt time.Time) hostreqkit.ObservedSafeguard {
	item := hostreqkit.ObservedSafeguard{
		Name: manifest.Name, Capability: manifest.Capability, CapabilityRole: manifest.CapabilityRole,
		Platforms: append([]string(nil), manifest.Platforms...), ObservedAt: observedAt,
	}
	status, inspectErr := InspectSafeguardAt(root, manifest.Name)
	if inspectErr != nil {
		item.SupportClass = hostreqkit.SupportUnsupported
		item.ExecutionState = hostreqkit.ExecutionFailed
		item.Notes = []string{inspectErr.Error()}
		return item
	}
	item.SupportClass = status.SupportClass
	item.ExecutionState = status.ExecutionState
	item.Notes = append([]string(nil), status.Notes...)
	return item
}

// ObserveSafeguardAt returns the read-only observation for a single named
// safeguard. It is the focused twin of ListObservedSafeguardsAt, for consumers
// reporting on one safeguard: sampling the whole roster to answer a question
// about one member charges that member for every other handler's probe cost.
//
// The requirement resolves exactly as InspectSafeguardAt resolves it -- name
// from the embedded catalog, operator config from root -- so the focused read
// and the focused apply always agree about what they are looking at, including
// which configuration the verdict is about. Catalog metadata (capability,
// platforms) is layered on from the same manifest.
//
// A name the catalog does not carry is an error, never an empty observation, so
// a typo cannot be read as "this safeguard is fine". Like the roster form, it
// never calls Apply.
func ObserveSafeguardAt(root, name string, now func() time.Time) (hostreqkit.ObservedSafeguard, error) {
	if strings.TrimSpace(root) == "" {
		return hostreqkit.ObservedSafeguard{}, fmt.Errorf("repository root is required")
	}
	if now == nil {
		now = clock.Real{}.Now
	}
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "-", "_")
	if normalized == "" {
		return hostreqkit.ObservedSafeguard{}, fmt.Errorf("safeguard name is required")
	}
	status, err := InspectSafeguardAt(root, normalized)
	if err != nil {
		return hostreqkit.ObservedSafeguard{}, err
	}
	item := hostreqkit.ObservedSafeguardFromStatus(status, now().UTC())
	item.Name = normalized
	if manifest, found := catalogSafeguard(normalized); found {
		item.Capability = manifest.Capability
		item.CapabilityRole = manifest.CapabilityRole
		item.Platforms = append([]string(nil), manifest.Platforms...)
	}
	return item, nil
}

// catalogSafeguard finds one embedded manifest by name. It performs no host
// probing; the manifests are compiled in.
func catalogSafeguard(name string) (hostreqkit.SafeguardManifest, bool) {
	manifests, err := safeguardManifests()
	if err != nil {
		return hostreqkit.SafeguardManifest{}, false
	}
	for _, manifest := range manifests {
		if manifest.Name == name {
			return manifest, true
		}
	}
	return hostreqkit.SafeguardManifest{}, false
}

// ListObservedSafeguardsAt returns the complete, read-only host safeguard
// observation surface owned by the control plane. Errors for an individual
// safeguard remain in that safeguard's notes and failed state; one broken
// probe cannot make the rest of the roster disappear.
func ListObservedSafeguardsAt(root string, now func() time.Time) ([]hostreqkit.ObservedSafeguard, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("repository root is required")
	}
	if now == nil {
		now = clock.Real{}.Now
	}
	manifests, err := safeguardManifests()
	if err != nil {
		return nil, err
	}
	result := make([]hostreqkit.ObservedSafeguard, 0, len(manifests))
	for _, manifest := range manifests {
		result = append(result, observeSafeguard(root, manifest, now().UTC()))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func ensureResolution(opts EnsureOptions, resolution hostreq.Resolution) (Report, error) {
	report, err := inspectResolution(Current(), opts.Environment, resolution)
	if err != nil {
		return Report{}, err
	}
	if !opts.AutoInstall {
		return report, missingRequiredError(report, opts)
	}

	for index, status := range report.Tools {
		if opts.OnOperation != nil {
			opts.OnOperation("Applying tool " + status.Name)
		}
		if requirementSatisfied(status) {
			continue
		}
		if updated, skip := skipOptional(status, opts.IncludeOptional); skip {
			report.Tools[index] = updated
			continue
		}
		updated, applyErr := applyRequirement(report.Host, status, opts)
		if applyErr != nil {
			return Report{}, applyErr
		}
		report.Tools[index] = annotateBlockingReason(updated)
	}
	for index, status := range report.Safeguards {
		if opts.OnOperation != nil {
			opts.OnOperation("Applying safeguard " + status.Name)
		}
		if requirementSatisfied(status) {
			continue
		}
		if updated, skip := skipOptional(status, opts.IncludeOptional); skip {
			report.Safeguards[index] = updated
			continue
		}
		updated, applyErr := applyRequirement(report.Host, status, opts)
		if applyErr != nil {
			return Report{}, applyErr
		}
		report.Safeguards[index] = annotateBlockingReason(updated)
	}

	report = summarizeReport(report)
	return report, missingRequiredError(report, opts)
}

// markOptionalSkipped is the path for optional items that the operator did not
// opt into via --include-optional. We deliberately do not call Apply: the item
// stays in whatever Pending-flavored state Inspect returned, but we tag it so
// the renderer routes it into the "Optional — opt in with --include-optional"
// group instead of the generic Pending bucket.
func markOptionalSkipped(status ItemStatus) ItemStatus {
	status.BlockingReason = hostreqkit.BlockingOptionalSkipped
	return status
}

func skipOptional(status ItemStatus, includeOptional bool) (ItemStatus, bool) {
	if status.Required || !isPendingState(status.ExecutionState) {
		return status, false
	}
	switch status.OperatorChoice {
	case hostreqspec.OperatorChoiceDeclined:
		status.BlockingReason = hostreqkit.BlockingOperatorDeclined
		return status, true
	case "":
		if !includeOptional {
			return markOptionalSkipped(status), true
		}
	case hostreqspec.OperatorChoiceNotRecorded:
		if !includeOptional {
			status.BlockingReason = hostreqkit.BlockingOperatorChoiceMissing
			return status, true
		}
	case hostreqspec.OperatorChoiceOptedIn:
		if !includeOptional && !status.ConfigNonDefault {
			status.BlockingReason = hostreqkit.BlockingOptionalSkipped
			status.Notes = append(status.Notes, "optional safeguard is opted in but uses manifest defaults; supply non-default configuration or rerun with --include-optional")
			return status, true
		}
	}
	return status, false
}
