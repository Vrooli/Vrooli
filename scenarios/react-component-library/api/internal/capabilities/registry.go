// Package capabilities exposes react-component-library's declared scenario
// integrations through the shared capability-registry contract.
package capabilities

import (
	"context"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
	capabilitysv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/capabilities"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/capabilities/capabilities_v1connect"
)

type (
	DependencyKind = capabilityregistry.DependencyKind
	Status         = capabilityregistry.Status
	ActionKind     = capabilityregistry.ActionKind
	Def            = capabilityregistry.Def
	State          = capabilityregistry.State
	Checker        = capabilityregistry.Checker
	Registry       = capabilityregistry.Registry
)

type RegistryMetadata struct {
	Platform capabilityregistry.PlatformVerdict `json:"platform"`
}

var Metadata = RegistryMetadata{Platform: capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformSupported, Reason: "The component catalog and direct library APIs are host-neutral; optional integrations are declared in service.json."}}

const (
	DependencyScenario      = capabilityregistry.DependencyScenario
	StatusAvailable         = capabilityregistry.StatusAvailable
	StatusUnavailable       = capabilityregistry.StatusUnavailable
	ActionKindScenarioStart = capabilityregistry.ActionKindScenarioStart
)

var Known = []Def{
	{ID: "agent-manager", Name: "Agent Manager", Description: "Assisted extraction and adoption workflows.", DependencyKind: DependencyScenario, DependencySlug: "agent-manager", ActionKind: ActionKindScenarioStart, ActionLabel: "Start agent-manager", OperatorCommand: "vrooli scenario start agent-manager"},
	{ID: "experience-manager", Name: "Experience Manager", Description: "Shared experience contracts and capture reconciliation.", DependencyKind: DependencyScenario, DependencySlug: "experience-manager", ActionKind: ActionKindScenarioStart, ActionLabel: "Start experience-manager", OperatorCommand: "vrooli scenario start experience-manager"},
	{ID: "ui-health", Name: "UI Health", Description: "Shared UI inventory and provenance contracts.", DependencyKind: DependencyScenario, DependencySlug: "ui-health", ActionKind: ActionKindScenarioStart, ActionLabel: "Start ui-health", OperatorCommand: "vrooli scenario start ui-health"},
	{ID: "typescript-code-graph", Name: "TypeScript Code Graph", Description: "Parsed TypeScript import graph for catalog reconciliation.", DependencyKind: DependencyScenario, DependencySlug: "typescript-code-graph", Criticality: capabilityregistry.CriticalityRequired, ActionKind: ActionKindScenarioStart, ActionLabel: "Start typescript-code-graph", OperatorCommand: "vrooli scenario start typescript-code-graph"},
}

// DeclaredCoverage is the server-computed reconciliation between catalog
// declarations and Experience Manager's live evaluator support. A capability
// is checkable only when Experience Manager reports it as provable; every other
// status is deliberately visible as uncheckable/unmeasured.
type DeclaredCoverage struct {
	Capability         string
	Title              string
	Status             string
	Checkable          bool
	Unmeasured         bool
	DeclaredAssetCount int
	AssetIDs           []string
	Blockers           []string
}

type DeclaredCoverageReport struct {
	Capabilities          []DeclaredCoverage
	DeclaredAssetCount    int
	UncheckableAssetCount int
	UnmeasuredAssetCount  int
	DeclarationCount      int
}

// ReconcileDeclared reads the authoritative Experience Manager capability
// derivation and joins it to catalog declarations. The join is kept here, in
// the RCL capability module, so catalog health cannot grow a second evaluator
// vocabulary or a hand-maintained capability list.
func ReconcileDeclared(ctx context.Context, declarations map[string][]string) DeclaredCoverageReport {
	statusByID, err := liveStatuses(ctx)
	if err != nil {
		statusByID = map[string]liveStatus{}
	}
	return ReconcileDeclaredWithStatuses(declarations, statusByID, err)
}

type liveStatus struct {
	Title    string
	Status   string
	Blockers []string
}

func liveStatuses(ctx context.Context) (map[string]liveStatus, error) {
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	base, err := resolver.ResolveScenarioURLDefault(ctx, "experience-manager")
	if err != nil {
		return nil, err
	}
	client := capabilitiesconnect.NewCapabilityStatusServiceClient(&http.Client{Timeout: 10 * time.Second}, base)
	response, err := client.GetStatus(ctx, connect.NewRequest(&capabilitysv1.GetStatusRequest{}))
	if err != nil {
		return nil, err
	}
	out := make(map[string]liveStatus, len(response.Msg.GetCapabilities()))
	for _, item := range response.Msg.GetCapabilities() {
		blockers := make([]string, 0, 2)
		if item.GetBlockingAxis() != "" {
			blockers = append(blockers, "axis "+item.GetBlockingAxis())
		}
		if item.GetBlockingEvidence() != "" {
			blockers = append(blockers, "evidence "+item.GetBlockingEvidence())
		}
		if item.GetStatus() == "no-checker" {
			blockers = append(blockers, "claim evaluator is not implemented")
		}
		out[item.GetId()] = liveStatus{Title: item.GetTitle(), Status: item.GetStatus(), Blockers: blockers}
	}
	return out, nil
}

// ReconcileDeclaredWithStatuses is deterministic so the catalog join can be
// tested without a running Experience Manager. statusErr is retained in the
// result as a visible unmeasured blocker when the live derivation is absent.
func ReconcileDeclaredWithStatuses(declarations map[string][]string, statuses map[string]liveStatus, statusErr error) DeclaredCoverageReport {
	capAssets := map[string][]string{}
	for assetID, capabilityIDs := range declarations {
		for _, capabilityID := range capabilityIDs {
			capabilityID = strings.TrimSpace(capabilityID)
			if capabilityID == "" {
				continue
			}
			capAssets[capabilityID] = append(capAssets[capabilityID], assetID)
		}
	}

	report := DeclaredCoverageReport{}
	uncheckableAssets := map[string]struct{}{}
	unmeasuredAssets := map[string]struct{}{}
	for capabilityID, assetIDs := range capAssets {
		sort.Strings(assetIDs)
		status, known := statuses[capabilityID]
		if !known {
			status.Status = "unmeasured"
			status.Title = capabilityID
			status.Blockers = []string{"capability derivation unavailable"}
			if statusErr == nil {
				status.Blockers = []string{"capability is not present in the live registry"}
			}
		}
		checkable := status.Status == "provable"
		unmeasured := status.Status == "no-checker" || status.Status == "evidence-missing" || status.Status == "unmeasured" || status.Status == "axis-unavailable" || status.Status == "axis-missing"
		row := DeclaredCoverage{Capability: capabilityID, Title: status.Title, Status: status.Status, Checkable: checkable, Unmeasured: unmeasured, DeclaredAssetCount: len(assetIDs), AssetIDs: assetIDs, Blockers: status.Blockers}
		report.Capabilities = append(report.Capabilities, row)
		report.DeclarationCount += len(assetIDs)
		if !checkable {
			for _, assetID := range assetIDs {
				uncheckableAssets[assetID] = struct{}{}
			}
		}
		if unmeasured {
			for _, assetID := range assetIDs {
				unmeasuredAssets[assetID] = struct{}{}
			}
		}
	}
	sort.Slice(report.Capabilities, func(i, j int) bool { return report.Capabilities[i].Capability < report.Capabilities[j].Capability })
	assetSet := map[string]struct{}{}
	for _, capability := range report.Capabilities {
		for _, assetID := range capability.AssetIDs {
			assetSet[assetID] = struct{}{}
		}
	}
	report.DeclaredAssetCount = len(assetSet)
	report.UncheckableAssetCount = len(uncheckableAssets)
	report.UnmeasuredAssetCount = len(unmeasuredAssets)
	return report
}

func NewRegistry() *Registry {
	return capabilityregistry.New(Known, map[string]Checker{
		"agent-manager":         ScenarioChecker{Slug: "agent-manager"},
		"experience-manager":    ScenarioChecker{Slug: "experience-manager"},
		"ui-health":             ScenarioChecker{Slug: "ui-health"},
		"typescript-code-graph": ScenarioChecker{Slug: "typescript-code-graph"},
	}, 30*time.Second)
}

// ScenarioChecker asks the control plane for lifecycle status rather than
// reaching into another scenario's private API.
type ScenarioChecker struct{ Slug string }

func (c ScenarioChecker) Check(ctx context.Context) (Status, string) {
	slug := strings.TrimSpace(c.Slug)
	if slug == "" {
		return StatusUnavailable, "scenario slug is not configured; use the start action"
	}
	if _, err := exec.LookPath("vrooli"); err != nil {
		return StatusUnavailable, "vrooli control-plane executable is unavailable; use the start action"
	}
	out, err := exec.CommandContext(ctx, "vrooli", "scenario", "status", slug, "--json").Output()
	if err != nil {
		return StatusUnavailable, "scenario status unavailable; use the start action"
	}
	body := strings.ToLower(string(out))
	if strings.Contains(body, `"healthy"`) || strings.Contains(body, `"running"`) {
		return StatusAvailable, "scenario is healthy"
	}
	return StatusUnavailable, "scenario is installed but stopped; use the start action"
}
