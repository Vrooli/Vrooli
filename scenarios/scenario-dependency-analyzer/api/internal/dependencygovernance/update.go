package dependencygovernance

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/installgateway"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

// UpdateLockedDependency moves packages already in a surface's lockfile to the
// newest version their declared ranges allow. It is how a security release that
// an existing range already admits reaches a scenario whose lockfile predates
// it, without a manifest override or a new approval: no manifest range changes
// and no dependency is added, so there is no governance verdict to give. The
// preview lists every locked version that leaves and enters, and a lockfile that
// would not change is reported as current.
func (h *connectHandler) UpdateLockedDependency(ctx context.Context, req *connect.Request[governancev1.UpdateLockedDependencyRequest]) (*connect.Response[governancev1.UpdateLockedDependencyResponse], error) {
	msg := req.Msg
	ecosystem, names := lockedPackageSelection(msg.GetEcosystem(), msg.GetPackageNames())
	if ecosystem == "" {
		ecosystem = "npm" // a locked-package update is npm-only; a bare name or an advisory means npm
	}
	var selectionNote string
	var fixes map[string][]advisoryFix
	advisories := advisoryList(msg.GetAdvisory())
	scoped := strings.TrimSpace(msg.GetScenario()) != "" && strings.TrimSpace(msg.GetSurface()) != ""
	if len(advisories) > 0 {
		advisory := strings.Join(advisories, ", ")
		if len(names) > 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("select packages either by name or by --advisory, not both"))
		}
		if !scoped {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("--advisory needs --scenario and --surface"))
		}
		evidence, err := h.surfaceEvidence(ctx, msg.GetScenario(), msg.GetSurface())
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		fixes = advisoryFixes(evidence)
		vulnerable := advisoryPackages(evidence, advisories)
		if len(vulnerable) == 0 {
			return connect.NewResponse(&governancev1.UpdateLockedDependencyResponse{
				DryRun: !msg.GetApply(), Verdict: "not_affected",
				Message:   fmt.Sprintf("Security Health reports no exposure to %s in scenarios/%s/%s; nothing to update.", advisory, msg.GetScenario(), msg.GetSurface()),
				NextSteps: []string{fmt.Sprintf("Check the evidence: security-health deps vulnerabilities --scenario %s --json", msg.GetScenario())},
			}), nil
		}
		names = vulnerable
		selectionNote = fmt.Sprintf("advisory %s selects %s", advisory, strings.Join(vulnerable, ", "))
	} else if scoped {
		// Naming packages gets the same exposure judgment; evidence being
		// unavailable only means there is nothing to say about it.
		if evidence, err := h.surfaceEvidence(ctx, msg.GetScenario(), msg.GetSurface()); err == nil {
			fixes = advisoryFixes(evidence)
		}
	}
	resolution, err := installgateway.ResolveLockUpdate(h.repoRoot(), msg.GetScenario(), msg.GetSurface(), ecosystem, names)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resolution.LockOnly = msg.GetLockOnly()
	out := h.runLockChange(ctx, lockChange{
		noun: "update", verb: "update", okVerdict: "updatable", blockedVerdict: "update_blocked", failedVerdict: "update_failed",
		currentIfUnchanged: true, highlight: resolution.UpdatedPackages, fixes: fixes, doneText: "Updated %s in %s via %s",
	}, msg.GetScenario(), msg.GetSurface(), msg.GetApply(), resolution)
	notes := out.Notes
	if selectionNote != "" {
		notes = append([]string{selectionNote}, notes...)
	}
	return connect.NewResponse(&governancev1.UpdateLockedDependencyResponse{
		Updated: out.Done, DryRun: out.DryRun, SourceUpdated: out.SourceUpdated, Blocked: out.Blocked,
		Verdict: out.Verdict, Command: out.Command, PackageManager: out.PackageManager, ManifestPath: out.ManifestPath,
		Message: out.Message, NextSteps: out.NextSteps, Notes: notes,
	}), nil
}

// advisoryEvidenceSource lists vulnerability evidence for a Security Health query.
type advisoryEvidenceSource func(ctx context.Context, req *governancev1.ListSecurityGovernanceGapsRequest) ([]*governancev1.SecurityVulnerabilityEvidence, error)

// WithAdvisoryEvidence overrides where an advisory selector reads vulnerability
// evidence (tests inject a fixed set instead of querying Security Health).
func WithAdvisoryEvidence(source advisoryEvidenceSource) ConnectOption {
	return func(h *connectHandler) { h.advisoryEvidence = source }
}

// advisoryList splits an advisory selector into its ids (comma or space
// separated), dropping empties and duplicates.
func advisoryList(selector string) []string {
	var ids []string
	seen := map[string]struct{}{}
	for _, id := range strings.FieldsFunc(selector, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' }) {
		if _, dup := seen[strings.ToUpper(id)]; !dup {
			seen[strings.ToUpper(id)] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}

// allAdvisories is the --advisory value that selects every exposure Security
// Health reports on the surface instead of naming advisories.
const allAdvisories = "all"

// surfaceEvidence reads the vulnerability evidence for one scenario's surface in
// a single Security Health query: npm findings whose lockfile is the surface's.
func (h *connectHandler) surfaceEvidence(ctx context.Context, scenario, surface string) ([]*governancev1.SecurityVulnerabilityEvidence, error) {
	source := h.advisoryEvidence
	if source == nil {
		source = func(ctx context.Context, req *governancev1.ListSecurityGovernanceGapsRequest) ([]*governancev1.SecurityVulnerabilityEvidence, error) {
			evidence, _, err := h.registry().securityVulnerabilityEvidenceList(ctx, req)
			return evidence, err
		}
	}
	evidence, err := source(ctx, &governancev1.ListSecurityGovernanceGapsRequest{Ecosystem: "npm", Scenario: scenario, Limit: 1000})
	if err != nil {
		return nil, err
	}
	var onThisSurface []*governancev1.SecurityVulnerabilityEvidence
	for _, item := range evidence {
		if strings.EqualFold(item.GetEcosystem(), "npm") && containsFold(item.GetScenarios(), scenario) && onSurface(item.GetSourceFiles(), scenario, surface) {
			onThisSurface = append(onThisSurface, item)
		}
	}
	return onThisSurface, nil
}

// advisoryPackages names the packages the evidence reports as exposed to any of
// the advisories (by id or alias), or to any advisory at all for "all", sorted
// and unique.
func advisoryPackages(evidence []*governancev1.SecurityVulnerabilityEvidence, advisories []string) []string {
	seen := map[string]struct{}{}
	for _, item := range evidence {
		for _, advisory := range advisories {
			if strings.EqualFold(advisory, allAdvisories) || advisoryMatches(item, advisory) {
				seen[item.GetPackageName()] = struct{}{}
				break
			}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func advisoryMatches(item *governancev1.SecurityVulnerabilityEvidence, advisory string) bool {
	return strings.EqualFold(item.GetVulnerabilityId(), advisory) || containsFold(item.GetAliases(), advisory)
}

// onSurface reports whether any evidence file is a lockfile in the surface's
// directory. A path may be scenario-relative ("ui/pnpm-lock.yaml") or
// repository-relative ("scenarios/<scenario>/ui/pnpm-lock.yaml"); retained
// evidence copies elsewhere in the repository never count. Evidence that names
// no file cannot rule the surface out.
func onSurface(files []string, scenario, surface string) bool {
	if len(files) == 0 {
		return true
	}
	surface = strings.Trim(surface, "/")
	for _, file := range files {
		file = strings.TrimPrefix(strings.TrimPrefix(file, "./"), "scenarios/"+scenario+"/")
		if strings.HasPrefix(file, surface+"/") {
			return true
		}
	}
	return false
}

// lockedPackageSelection accepts package names spelled <ecosystem>/<name>, the
// form the CLI takes, and returns the ecosystem they name (or the request's)
// with the bare names. A scoped name such as @types/node has no ecosystem prefix
// (npm/@types/node does).
func lockedPackageSelection(ecosystem string, names []string) (string, []string) {
	bare := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		// An unscoped npm package name never contains a slash, so a slash here
		// can only separate the ecosystem from the name.
		if eco, rest, ok := strings.Cut(name, "/"); ok && !strings.HasPrefix(name, "@") {
			ecosystem, name = eco, rest
		}
		bare = append(bare, name)
	}
	return ecosystem, bare
}
