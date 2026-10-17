package providers

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"storage-manager/internal/cleanup"
)

// A non-live scenario instance -- "<scenario>@<variant>" -- owns private
// directories named "<scenario>_<variant>" under every storage class root
// (~/.vrooli/data/vrooli/web-console_presentation and its siblings). The
// lifecycle creates them on first start and nothing removes them when the
// instance is stopped for good, so a shadow build or a one-off smoke variant
// leaves its whole namespace behind permanently. No provider owned that
// storage before this one.
//
// The class roots involved (data, state, config) are contract-PROTECTED, which
// is correct: no age-or-size rule may walk them. This provider is the narrow
// exception the protection rationale already anticipates for `bin` -- it proves
// a specific directory belongs to a specific non-live instance that the control
// plane is not running, and it surfaces only that directory. Everything else in
// the root stays invisible to it.
const (
	instanceVariantSeparator = "_"
	// instanceQuarantineRetention is how long a quarantined instance namespace
	// is kept before its bytes are actually reclaimed. Instance storage is
	// private application data, not a cache: the reversible window is the
	// difference between a mistake an operator can undo and one they cannot.
	instanceQuarantineRetention = 7 * 24 * time.Hour
)

// InstanceLiveness reports which scenario instances the control plane is
// currently running.
//
// This is deliberately a registry question, not a filesystem question. A
// stopped instance and a running one look identical on disk -- mtime says only
// when bytes last changed, and an idle-but-running instance can be older than a
// dead one. Guessing from the filesystem here would eventually delete the
// storage of a live process.
type InstanceLiveness interface {
	// RunningInstances returns the set of instance slugs the control plane has
	// running, in the canonical "<scenario>@<variant>" form ("<scenario>" for
	// the live instance).
	RunningInstances(ctx context.Context) (map[string]struct{}, error)
}

// InstanceNamespaceRoot is one storage class root that holds one directory per
// scenario instance.
type InstanceNamespaceRoot struct {
	// Class is the storage class name ("data", "state", ...), used only to
	// describe a finding.
	Class string
	// Path is the "<class-root>/<app>" directory whose immediate children are
	// instance namespaces.
	Path string
}

type OrphanedInstanceStorageProviderConfig struct {
	// NamespaceRoots are the class roots to scan.
	NamespaceRoots []InstanceNamespaceRoot
	// KnownScenarios is the scenario inventory a directory prefix must resolve
	// against. Without it this provider refuses to run: a scenario name
	// contains hyphens and a variant may too, so "<prefix>_<suffix>" is not a
	// parse, it is a lookup.
	KnownScenarios []string
	// Quarantine is the directory a reclaimed namespace is moved into. Apply
	// refuses without it.
	Quarantine string
	// Retention overrides instanceQuarantineRetention.
	Retention time.Duration
}

type OrphanedInstanceStorageProvider struct {
	meta      cleanup.ProviderMetadata
	files     cleanup.FileSystem
	instances InstanceLiveness
	clock     cleanup.Clock

	roots      []InstanceNamespaceRoot
	known      []string
	quarantine string
	retention  time.Duration
}

func NewOrphanedInstanceStorageProvider(files cleanup.FileSystem, instances InstanceLiveness, clock cleanup.Clock, cfg OrphanedInstanceStorageProviderConfig) *OrphanedInstanceStorageProvider {
	roots := make([]InstanceNamespaceRoot, 0, len(cfg.NamespaceRoots))
	proven := make([]string, 0, len(cfg.NamespaceRoots))
	for _, root := range cfg.NamespaceRoots {
		path := filepath.Clean(strings.TrimSpace(root.Path))
		if path == "" || path == "." || !filepath.IsAbs(path) {
			continue
		}
		roots = append(roots, InstanceNamespaceRoot{Class: strings.TrimSpace(root.Class), Path: path})
		proven = append(proven, path)
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Path < roots[j].Path })
	sort.Strings(proven)

	known := make([]string, 0, len(cfg.KnownScenarios))
	for _, name := range cfg.KnownScenarios {
		if name = strings.TrimSpace(name); name != "" {
			known = append(known, name)
		}
	}
	// Longest first: if both "web" and "web-console" were scenarios,
	// "web-console_presentation" belongs to the longer one.
	sort.Slice(known, func(i, j int) bool {
		if len(known[i]) != len(known[j]) {
			return len(known[i]) > len(known[j])
		}
		return known[i] < known[j]
	})

	retention := cfg.Retention
	if retention <= 0 {
		retention = instanceQuarantineRetention
	}

	return &OrphanedInstanceStorageProvider{
		meta: cleanup.ProviderMetadata{
			ID:                  "orphaned-instance-storage",
			Name:                "Orphaned non-live instance storage",
			Version:             "v1",
			OwnerScenario:       "storage-manager",
			SafetyTier:          cleanup.SafetyTierSafeWithOwner,
			DefaultMode:         cleanup.ProviderModeDisabled,
			DefaultApproval:     cleanup.ApprovalModeOwner,
			SupportedPlatforms:  []string{"linux", "darwin"},
			IrreversibleEffects: []string{"moves a stopped non-live instance's private storage to quarantine and removes it after the retention window"},
			TestSubstitute:      "fake-filesystem-and-instance-registry",
			ProvenOrphanRoots:   proven,
		},
		files:      files,
		instances:  instances,
		clock:      clock,
		roots:      roots,
		known:      known,
		quarantine: filepath.Clean(strings.TrimSpace(cfg.Quarantine)),
		retention:  retention,
	}
}

func (p *OrphanedInstanceStorageProvider) Metadata() cleanup.ProviderMetadata { return p.meta }

func (p *OrphanedInstanceStorageProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	preview, err := p.Preview(ctx, cleanup.PreviewRequest{Scope: req.Scope, Policy: req.Policy})
	if err != nil {
		return cleanup.Estimate{}, err
	}
	return cleanup.Estimate{
		ProviderID:       p.meta.ID,
		ProviderVersion:  p.meta.Version,
		EstimatedBytes:   sumPreviewBytes(preview.Items),
		ItemCount:        len(preview.Items),
		RequiresApproval: req.Policy.ApprovalMode != cleanup.ApprovalModeNone,
		BlockedReason:    preview.BlockedReason,
		ObservedAt:       p.now(req.Scope),
		MinAge:           req.Policy.MinAge,
		MaxBytes:         req.Policy.MaxBytes,
	}, nil
}

func (p *OrphanedInstanceStorageProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	out := cleanup.Preview{
		ProviderID:      p.meta.ID,
		ProviderVersion: p.meta.Version,
		MinAge:          req.Policy.MinAge,
		MaxBytes:        req.Policy.MaxBytes,
	}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	if reason := p.unavailable(); reason != "" {
		out.BlockedReason = reason
		return out, nil
	}

	// Fail closed. An unreadable registry is not evidence that nothing is
	// running; it is evidence that this provider cannot tell, and the honest
	// answer to "is that instance alive?" when the registry is unavailable is
	// to propose nothing at all.
	running, err := p.instances.RunningInstances(ctx)
	if err != nil {
		out.BlockedReason = "instance registry could not be read: " + cleanup.Redact(err.Error())
		return out, nil
	}

	now := p.now(req.Scope)
	unresolved := 0
	heldByAge := 0
	// One instance owns a directory in several class roots, so a per-directory
	// warning repeats the same fact up to five times and buries the rest.
	warnedRunning := make(map[string]struct{})
	for _, root := range p.roots {
		entries, readErr := p.files.ReadDir(ctx, root.Path)
		if readErr != nil {
			if !isMissing(readErr) {
				out.Warnings = append(out.Warnings, fmt.Sprintf("%s: class root could not be listed; skipped", root.Class))
			}
			continue
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return cleanup.Preview{}, err
			}
			if !entry.IsDir {
				continue
			}
			name := filepath.Base(entry.Path)
			key, class := p.classify(name)
			switch class {
			case classLiveInstance:
				// A live instance's own directory. Never a target, and not a
				// finding worth reporting either.
				continue
			case classUnattributable:
				// The state root also holds host safeguard directories such as
				// "agent_session_containment" whose prefix is not a scenario.
				// Counted so an operator can see this provider looked and
				// declined, rather than silently ignoring them.
				unresolved++
				continue
			}
			if _, alive := running[key.slug()]; alive {
				if _, warned := warnedRunning[key.slug()]; !warned {
					warnedRunning[key.slug()] = struct{}{}
					out.Warnings = append(out.Warnings, fmt.Sprintf("%s: instance is running; skipped", key.slug()))
				}
				continue
			}
			bytes, newest, measured := p.measure(ctx, entry.Path)
			if !measured {
				out.Warnings = append(out.Warnings, fmt.Sprintf("%s (%s): size could not be measured; skipped", key.slug(), root.Class))
				continue
			}
			if req.Policy.MinAge > 0 && !newest.IsZero() && now.Sub(newest) < req.Policy.MinAge {
				// Counted, not silent. A stopped instance held only by the age
				// window is indistinguishable from one this provider failed to
				// detect, and that ambiguity costs more to diagnose than the
				// warning costs to print.
				heldByAge++
				continue
			}
			if req.Policy.MaxBytes > 0 && sumPreviewBytes(out.Items)+bytes > req.Policy.MaxBytes {
				out.Warnings = append(out.Warnings, "max reclaim limit reached")
				continue
			}
			out.Items = append(out.Items, cleanup.PreviewItem{
				ID:          stableItemID(p.meta.ID, entry.Path),
				Path:        entry.Path,
				Description: fmt.Sprintf("%s storage of stopped instance %s", root.Class, key.slug()),
				Bytes:       bytes,
				Action:      "instance-storage-quarantine",
				SafetyTier:  p.meta.SafetyTier,
			})
		}
	}
	if heldByAge > 0 {
		out.Warnings = append(out.Warnings, fmt.Sprintf("%d stopped instance %s younger than the %s minimum age; held", heldByAge, plural(heldByAge, "directory is", "directories are"), req.Policy.MinAge))
	}
	if unresolved > 0 {
		out.Warnings = append(out.Warnings, fmt.Sprintf("%d %s not be attributed to a known scenario; skipped", unresolved, plural(unresolved, "directory could", "directories could")))
	}
	return out, nil
}

func (p *OrphanedInstanceStorageProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != "" && req.ProviderVersion != p.meta.Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s version mismatch: got %q want %q", p.meta.ID, req.ProviderVersion, p.meta.Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s apply requires idempotency key", p.meta.ID)
	}
	result := cleanup.ApplyResult{ProviderID: p.meta.ID}
	if req.ApprovalMode != cleanup.ApprovalModeOwner && req.ApprovalMode != cleanup.ApprovalModeOperator {
		result.SkippedItems = previewItemIDs(req.Preview.Items)
		result.Warnings = append(result.Warnings, "owner or operator approval required")
		return result, nil
	}
	if reason := p.unavailable(); reason != "" {
		result.SkippedItems = previewItemIDs(req.Preview.Items)
		result.Warnings = append(result.Warnings, reason)
		return result, nil
	}
	if p.quarantine == "" || p.quarantine == "." {
		// Instance storage is private application data. Deleting it outright
		// on the strength of a preview taken some time earlier is the one
		// failure this provider must not have, so the reversible path is the
		// only path.
		result.SkippedItems = previewItemIDs(req.Preview.Items)
		result.Warnings = append(result.Warnings, "no quarantine root is configured; refusing to move instance storage")
		return result, nil
	}

	expired, err := p.expireQuarantine(ctx)
	if err != nil {
		return result, err
	}
	result.ReclaimedBytes += expired.ReclaimedBytes
	result.AppliedItems = append(result.AppliedItems, expired.AppliedItems...)
	result.Warnings = append(result.Warnings, expired.Warnings...)

	running, err := p.instances.RunningInstances(ctx)
	if err != nil {
		result.SkippedItems = append(result.SkippedItems, previewItemIDs(req.Preview.Items)...)
		result.Warnings = append(result.Warnings, "instance registry could not be re-read; nothing quarantined")
		result.Applied = result.ReclaimedBytes > 0
		return result, nil
	}

	stamp := p.now(cleanup.ObservationScope{}).UTC().Unix()
	for _, item := range req.Preview.Items {
		if err := ctx.Err(); err != nil {
			return cleanup.ApplyResult{}, err
		}
		path := filepath.Clean(item.Path)
		root, ok := p.rootFor(path)
		if !ok {
			result.SkippedItems = append(result.SkippedItems, item.ID)
			result.Warnings = append(result.Warnings, item.ID+": path is not an instance namespace directory")
			continue
		}
		key, ok := p.resolveInstanceDir(filepath.Base(path))
		if !ok {
			// Re-derived from the path at apply time rather than trusted from
			// the preview, so a stale or forged item cannot name a live
			// instance's directory.
			result.SkippedItems = append(result.SkippedItems, item.ID)
			result.Warnings = append(result.Warnings, item.ID+": path does not belong to a known non-live instance")
			continue
		}
		if _, alive := running[key.slug()]; alive {
			result.SkippedItems = append(result.SkippedItems, item.ID)
			result.Warnings = append(result.Warnings, item.ID+": instance started again; skipped")
			continue
		}
		if _, statErr := p.files.Stat(ctx, path); statErr != nil {
			if isMissing(statErr) {
				result.AlreadyDone = true
				continue
			}
			result.SkippedItems = append(result.SkippedItems, item.ID)
			result.Warnings = append(result.Warnings, item.ID+": "+cleanup.Redact(statErr.Error()))
			continue
		}
		if err := p.files.MkdirAll(ctx, p.quarantine); err != nil {
			return result, fmt.Errorf("create instance storage quarantine: %w", err)
		}
		destination := filepath.Join(p.quarantine, fmt.Sprintf("%d-%s-%s", stamp, root.Class, filepath.Base(path)))
		if err := p.files.Rename(ctx, path, destination); err != nil {
			result.SkippedItems = append(result.SkippedItems, item.ID)
			result.Warnings = append(result.Warnings, item.ID+": "+cleanup.Redact(err.Error()))
			continue
		}
		result.AppliedItems = append(result.AppliedItems, destination)
	}
	if len(result.AppliedItems) > 0 {
		result.Applied = true
		result.Warnings = append(result.Warnings, fmt.Sprintf("instance storage quarantined for %s; no bytes reclaimed until expiry", p.retention))
	}
	result.Applied = result.Applied || result.ReclaimedBytes > 0
	return result, nil
}

func (p *OrphanedInstanceStorageProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "stopped instance namespaces were moved to quarantine through the FileSystem seam"}, nil
}

func (p *OrphanedInstanceStorageProvider) unavailable() string {
	switch {
	case p.files == nil:
		return "filesystem seam unavailable"
	case p.instances == nil:
		return "instance registry seam unavailable"
	case len(p.roots) == 0:
		return "instance storage roots unavailable"
	case len(p.known) == 0:
		return "scenario inventory unavailable"
	default:
		return ""
	}
}

// instanceRef is a resolved "<scenario>_<variant>" directory name.
type instanceRef struct {
	scenario string
	variant  string
}

func (r instanceRef) slug() string { return r.scenario + "@" + r.variant }

// dirClass is what a namespace directory name turned out to be.
type dirClass int

const (
	// classUnattributable is a name this provider cannot tie to a scenario.
	classUnattributable dirClass = iota
	// classLiveInstance is a live instance's own directory: the name IS a
	// scenario slug, with no variant suffix.
	classLiveInstance
	// classNonLiveInstance is "<scenario>_<variant>" for a real scenario.
	classNonLiveInstance
)

// classify attributes a namespace directory name.
//
// Splitting on "_" is not a parse here. Scenario slugs contain hyphens, variants
// may contain hyphens and digits, and the state root additionally holds host
// safeguard directories ("agent_session_containment", "docker_host_firewall")
// whose prefix is not a scenario at all. The only sound rule is: the prefix must
// BE a known scenario. A name that is itself a known scenario is a LIVE
// instance's directory and is recognised before anything else, so a live
// namespace can never fall through into the instance branch.
func (p *OrphanedInstanceStorageProvider) classify(name string) (instanceRef, dirClass) {
	for _, scenario := range p.known {
		if name == scenario {
			return instanceRef{}, classLiveInstance
		}
	}
	for _, scenario := range p.known {
		if !strings.HasPrefix(name, scenario+instanceVariantSeparator) {
			continue
		}
		variant := name[len(scenario)+len(instanceVariantSeparator):]
		if variant == "" || variant == liveVariant {
			// "<scenario>_live" is not a shape the naming SSOT produces; the
			// live instance has no suffix. Refuse rather than invent a meaning.
			return instanceRef{}, classUnattributable
		}
		return instanceRef{scenario: scenario, variant: variant}, classNonLiveInstance
	}
	return instanceRef{}, classUnattributable
}

// resolveInstanceDir is the apply-time guard: it accepts only a non-live
// instance directory.
func (p *OrphanedInstanceStorageProvider) resolveInstanceDir(name string) (instanceRef, bool) {
	key, class := p.classify(name)
	return key, class == classNonLiveInstance
}

// liveVariant mirrors scenarioruntime.DefaultVariant. It is duplicated rather
// than imported because the control plane's internal packages are not reachable
// from a scenario module; the constant is part of a stable naming contract.
const liveVariant = "live"

func (p *OrphanedInstanceStorageProvider) rootFor(path string) (InstanceNamespaceRoot, bool) {
	parent := filepath.Dir(path)
	for _, root := range p.roots {
		if parent == root.Path {
			return root, true
		}
	}
	return InstanceNamespaceRoot{}, false
}

// measure sums a namespace directory and returns the newest modification time
// in it. A traversal error makes the result unusable: a partially measured tree
// can look both small and old when it is neither.
func (p *OrphanedInstanceStorageProvider) measure(ctx context.Context, path string) (int64, time.Time, bool) {
	var total int64
	var newest time.Time
	err := p.files.Walk(ctx, path, func(info cleanup.FileInfo) error {
		if !info.IsDir {
			total += info.Size
		}
		if info.ModTime.After(newest) {
			newest = info.ModTime
		}
		return nil
	})
	if err != nil {
		return 0, time.Time{}, false
	}
	return total, newest, true
}

func (p *OrphanedInstanceStorageProvider) expireQuarantine(ctx context.Context) (cleanup.ApplyResult, error) {
	result := cleanup.ApplyResult{ProviderID: p.meta.ID}
	entries, err := p.files.ReadDir(ctx, p.quarantine)
	if err != nil {
		// A missing quarantine is the normal first-run state.
		return result, nil
	}
	cutoff := p.now(cleanup.ObservationScope{}).Add(-p.retention).Unix()
	for _, entry := range entries {
		name := filepath.Base(entry.Path)
		stampText, remainder, found := strings.Cut(name, "-")
		if !found {
			continue
		}
		stamp, parseErr := strconv.ParseInt(stampText, 10, 64)
		if parseErr != nil || stamp > cutoff {
			continue
		}
		_, dirName, found := strings.Cut(remainder, "-")
		if !found {
			continue
		}
		if _, ok := p.resolveInstanceDir(dirName); !ok {
			// Only entries this provider could have created are expired. A
			// quarantine directory is shared ground; nothing else in it is ours
			// to delete.
			continue
		}
		bytes, _, measured := p.measure(ctx, entry.Path)
		if !measured {
			result.Warnings = append(result.Warnings, name+": quarantined size could not be measured; kept")
			continue
		}
		if err := p.files.RemoveAll(ctx, entry.Path); err != nil {
			result.Warnings = append(result.Warnings, name+": "+cleanup.Redact(err.Error()))
			continue
		}
		result.ReclaimedBytes += bytes
		result.AppliedItems = append(result.AppliedItems, entry.Path)
	}
	return result, nil
}

func plural(count int, one, many string) string {
	if count == 1 {
		return one
	}
	return many
}

func (p *OrphanedInstanceStorageProvider) now(scope cleanup.ObservationScope) time.Time {
	if !scope.Now.IsZero() {
		return scope.Now
	}
	if p.clock != nil {
		return p.clock.Now()
	}
	return time.Now().UTC()
}
