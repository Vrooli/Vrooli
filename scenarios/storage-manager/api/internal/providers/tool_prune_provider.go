package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"storage-manager/internal/cleanup"
)

// ToolPruneProvider reclaims a tool-owned cache by invoking the tool's own
// prune command instead of deleting files underneath it.
//
// Some caches cannot be safely reclaimed by a generic age-and-size file walk.
// The Go build cache is the motivating case: Go promises that a Get/Put output
// stays available until the cache is Closed, so a file walk can delete an
// output a running build still depends on. The FileProvider refuses these for
// exactly that reason (see file_provider.go reclaimBlockedReason). The owning
// tool, by contrast, knows its own liveness and concurrency rules, so its own
// prune command is the owner-coordinated reclamation the file providers leave
// to it. `go clean -cache` is precisely that command for the Go build cache.
//
// The command is data, taken verbatim from the repository storage contract
// (.vrooli/repo-contract.json storage.roots[].tool_prune_command) and executed
// only through the ProcessRunner seam. This keeps the seam-only contract
// (internal/cleanup/no_real_cleanup_test.go) intact: no command literal and no
// exec.Command lives in this package, and the set of runnable commands is
// exactly the set an operator declared in the contract.
type ToolPruneProvider struct {
	cfg    ToolPruneProviderConfig
	files  cleanup.FileSystem
	runner cleanup.ProcessRunner
	clock  cleanup.Clock
}

// ToolPruneProviderConfig is the resolved, host-specific view of a declarative
// root that reclaims through a tool command. Root is the absolute path the
// command maintains; it is used only to measure reclaimed bytes, never walked
// for deletion.
type ToolPruneProviderConfig struct {
	ID        string
	Name      string
	Root      string
	Command   cleanup.ProcessCommand
	Tier      cleanup.SafetyTier
	Platforms []string
	Proof     cleanup.RegenerableProof
	// MaxBytes is the root's declared byte ceiling. Pressure recovery runs the
	// prune only when the measured cache exceeds it: the command clears the
	// whole cache, so firing it on every pressure tick would throw away a
	// within-budget cache and rebuild it on the next build. Zero disables the
	// gate. Operator plans are not gated; an explicit plan is its own reason.
	MaxBytes int64
}

// NewToolPruneProvider validates the resolved config against the provider
// metadata contract so a malformed root (for example a regenerable tier without
// complete proof) fails registry construction rather than at delete time.
func NewToolPruneProvider(files cleanup.FileSystem, runner cleanup.ProcessRunner, clock cleanup.Clock, cfg ToolPruneProviderConfig) (*ToolPruneProvider, error) {
	p := &ToolPruneProvider{cfg: cfg, files: files, runner: runner, clock: clock}
	if err := cleanup.ValidateProvider(p); err != nil {
		return nil, err
	}
	return p, nil
}

// GovernedRoots reports the cache the owning tool maintains.
func (p *ToolPruneProvider) GovernedRoots() []string {
	if strings.TrimSpace(p.cfg.Root) == "" {
		return nil
	}
	return []string{p.cfg.Root}
}

func (p *ToolPruneProvider) Metadata() cleanup.ProviderMetadata {
	meta := cleanup.ProviderMetadata{
		ID:                  p.cfg.ID,
		Name:                p.cfg.Name,
		Version:             "v1",
		OwnerScenario:       "storage-manager",
		SafetyTier:          p.cfg.Tier,
		DefaultMode:         cleanup.ProviderModeDisabled,
		DefaultApproval:     cleanup.ApprovalModeOperator,
		SupportedPlatforms:  append([]string(nil), p.cfg.Platforms...),
		IrreversibleEffects: []string{"the tool's own prune command removes cached entries it judges reclaimable"},
		TestSubstitute:      "fake-process-runner",
	}
	// A regenerable cache may run unattended, but only on the same complete
	// proof every regenerable provider must carry. A conditional cache (the Go
	// build cache) keeps operator approval and never runs autonomously, so its
	// 100+ GB is not re-reclaimed every pressure tick on a build-heavy host.
	if p.cfg.Tier == cleanup.SafetyTierRegenerable {
		meta.DefaultApproval = cleanup.ApprovalModeNone
		meta.NoLease = p.cfg.Proof.NoLease
		meta.RegenerableProof = p.cfg.Proof
	}
	return meta
}

func (p *ToolPruneProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	meta := p.Metadata()
	out := cleanup.Estimate{
		ProviderID:       meta.ID,
		ProviderVersion:  meta.Version,
		RequiresApproval: req.Policy.ApprovalMode != cleanup.ApprovalModeNone,
		ObservedAt:       p.now(req.Scope),
	}
	if reason := p.blockedReason(req.Policy); reason != "" {
		out.BlockedReason = reason
		return out, nil
	}
	bytes, err := p.measure(ctx)
	if err != nil {
		return cleanup.Estimate{}, err
	}
	if p.withinCeiling(req.Scope, bytes) {
		return out, nil
	}
	out.EstimatedBytes = bytes
	if bytes > 0 {
		out.ItemCount = 1
	}
	return out, nil
}

// withinCeiling reports whether a recovery observation must leave the cache
// alone because it is at or under its declared ceiling.
func (p *ToolPruneProvider) withinCeiling(scope cleanup.ObservationScope, bytes int64) bool {
	return scope.Recovery && p.cfg.MaxBytes > 0 && bytes <= p.cfg.MaxBytes
}

func (p *ToolPruneProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	meta := p.Metadata()
	out := cleanup.Preview{ProviderID: meta.ID, ProviderVersion: meta.Version}
	if reason := p.blockedReason(req.Policy); reason != "" {
		out.BlockedReason = reason
		return out, nil
	}
	bytes := req.Estimate.EstimatedBytes
	if req.Estimate.ProviderID != meta.ID || bytes <= 0 {
		measured, err := p.measure(ctx)
		if err != nil {
			return cleanup.Preview{}, err
		}
		bytes = measured
	}
	if bytes <= 0 || p.withinCeiling(req.Scope, bytes) {
		// Nothing to reclaim: report no items rather than a zero-byte item the
		// controller would drop anyway.
		return out, nil
	}
	out.Items = append(out.Items, cleanup.PreviewItem{
		ID:          meta.ID + ":tool-prune",
		Path:        p.cfg.Root,
		Description: fmt.Sprintf("%s via `%s`", meta.Name, p.commandString()),
		Bytes:       bytes,
		Action:      "tool-prune",
		SafetyTier:  meta.SafetyTier,
	})
	// The prune command is one indivisible effect over the whole cache, so its
	// single preview item is expected to exceed the controller batch cap. The
	// FileProvider splits a tree into top-level entries; a tool command cannot
	// be split, so it claims the indivisible-entry authority instead.
	out.AllowSingleOvershoot = true
	return out, nil
}

func (p *ToolPruneProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	meta := p.Metadata()
	if req.ProviderVersion != meta.Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s version mismatch: got %q want %q", meta.ID, req.ProviderVersion, meta.Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s apply requires idempotency key", meta.ID)
	}
	if req.ApprovalMode == cleanup.ApprovalModeDisabled {
		return cleanup.ApplyResult{ProviderID: meta.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"provider disabled by policy"}}, nil
	}
	if p.runner == nil {
		return cleanup.ApplyResult{ProviderID: meta.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"process runner unavailable"}}, nil
	}
	// Measure before the command so reclaimed bytes come from an observed delta,
	// never from the estimate: an estimate echoed back as the result would make
	// a no-op prune look like a successful reclamation.
	before := p.sizeBefore(ctx, req.Preview)
	result, runErr := p.runner.Run(ctx, p.cfg.Command)
	if runErr != nil {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s prune command: %w", meta.ID, runErr)
	}
	if result.ExitCode != 0 {
		return cleanup.ApplyResult{
			ProviderID:   meta.ID,
			SkippedItems: previewItemIDs(req.Preview.Items),
			Warnings:     []string{fmt.Sprintf("prune command exited %d: %s", result.ExitCode, cleanup.Redact(strings.TrimSpace(result.Stderr)))},
		}, nil
	}
	after, _ := p.measure(ctx)
	reclaimed := before - after
	if reclaimed < 0 {
		reclaimed = 0
	}
	return cleanup.ApplyResult{
		ProviderID:     meta.ID,
		Applied:        true,
		AppliedItems:   previewItemIDs(req.Preview.Items),
		ReclaimedBytes: reclaimed,
	}, nil
}

func (p *ToolPruneProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "tool-prune provider executes the contract-declared command through the process runner seam"}, nil
}

// blockedReason reports why a prune cannot run, mirroring the file and command
// providers so a disabled or unresolvable root reports a reason rather than
// failing the run.
func (p *ToolPruneProvider) blockedReason(policy cleanup.ProviderPolicy) string {
	switch {
	case !policy.Enabled:
		return "provider disabled by policy"
	case p.runner == nil:
		return "process runner unavailable"
	case strings.TrimSpace(p.cfg.Command.Name) == "":
		return "no tool prune command configured"
	case strings.TrimSpace(p.cfg.Root) == "":
		return "cache root unavailable on this host"
	default:
		return ""
	}
}

// measure sums the regular-file bytes under the cache root. A missing root is a
// normal, correct zero (a host that never ran the tool has no cache).
func (p *ToolPruneProvider) measure(ctx context.Context) (int64, error) {
	if p.files == nil || strings.TrimSpace(p.cfg.Root) == "" {
		return 0, nil
	}
	var total int64
	err := p.files.Walk(ctx, p.cfg.Root, func(info cleanup.FileInfo) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if !info.IsDir {
			total += info.Size
		}
		return nil
	})
	if err != nil {
		if isFileMissing(err) {
			return 0, nil
		}
		return 0, err
	}
	return total, nil
}

// sizeBefore prefers the already-measured preview total and only re-walks when
// the preview carried no usable figure, so a routine apply measures the root
// once rather than twice.
func (p *ToolPruneProvider) sizeBefore(ctx context.Context, preview cleanup.Preview) int64 {
	if n := sumPreviewBytes(preview.Items); n > 0 {
		return n
	}
	n, _ := p.measure(ctx)
	return n
}

func (p *ToolPruneProvider) commandString() string {
	return strings.TrimSpace(p.cfg.Command.Name + " " + strings.Join(p.cfg.Command.Args, " "))
}

func (p *ToolPruneProvider) now(scope cleanup.ObservationScope) time.Time {
	if !scope.Now.IsZero() {
		return scope.Now
	}
	if p.clock != nil {
		return p.clock.Now()
	}
	return time.Time{}
}
