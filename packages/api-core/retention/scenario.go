package retention

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/scenario"
	"github.com/vrooli/api-core/storage"
)

// This file is the no-Go-code path. A component declares budgets in its
// manifest, calls NewForScenario once at startup, and gets enforcement. If
// wiring needed per-component code, adoption would stall at exactly the
// components that already have hand-rolled retention — the ones with the least
// incentive to change.

// Finding reports a budget whose byte ceiling is what limited what was kept.
//
// It is deliberately not an error. The prune succeeded; the signal is about the
// producer, which is generating data faster than its declared age horizon
// allows. Reporting it is what stops retention from silently hiding the defect
// it compensates for — the 30-day autoheal policy was correctly configured and
// running the whole time its database grew to 41% of the disk.
type Finding struct {
	// Scenario is the component instance the budget belongs to, variant
	// included, so a shadow's finding is not mistaken for live's.
	Scenario string
	// Budget is the manifest key that declared the ceiling.
	Budget string
	// Target is the resolved absolute path the budget bounds.
	Target string
	// Rationale is the author's declared reason for the ceiling, carried
	// through so whoever reads the finding knows what drives the volume.
	Rationale string
	// UsedBytes is the measured size after the cycle.
	UsedBytes int64
	// MaxBytes is the declared ceiling.
	MaxBytes int64
	// Deleted is how many items the cycle removed to hold the ceiling.
	Deleted int64
}

// String renders the finding as one operator-readable line.
func (f Finding) String() string {
	msg := fmt.Sprintf("retention budget %q in %s is bound by its byte ceiling: %s of %s used after removing %d items",
		f.Budget, f.Scenario, FormatBytes(f.UsedBytes), FormatBytes(f.MaxBytes), f.Deleted)
	if f.Rationale != "" {
		msg += " (" + f.Rationale + ")"
	}
	return msg
}

// ScenarioConfig configures an engine built from a component's own manifest.
type ScenarioConfig struct {
	// Manifest is the raw manifest JSON. When nil it is read from
	// ManifestPath, or discovered by walking up from StartDir.
	Manifest []byte
	// ManifestPath is an explicit manifest location.
	ManifestPath string
	// StartDir is where manifest discovery begins. Defaults to the working
	// directory.
	StartDir string
	// Scenario is the bare scenario slug used as the namespace fallback.
	// Defaults to scenario.Name().
	Scenario string
	// RootOverride forces all storage class roots under one directory.
	RootOverride string
	// Registry resolves budgets declaring pruner "custom".
	Registry *Registry
	// OpenDatabase opens a SQLite database. Required only when a sqlite_table
	// budget is declared.
	OpenDatabase func(path string) (Execer, error)
	// Interval between cycles. Defaults to DefaultInterval.
	Interval time.Duration
	// RunOnStart runs one cycle immediately rather than waiting an interval.
	RunOnStart bool
	// AllowFullVacuum permits the one-time full VACUUM that converts a
	// database to incremental auto-vacuum. Off by default; it belongs to an
	// explicit operator command, never to startup.
	AllowFullVacuum bool
	// MaxDuration bounds one Prune call's wall clock. Defaults to
	// DefaultCycleDuration for a scheduled engine.
	MaxDuration time.Duration
	// BatchSize is rows per delete statement.
	BatchSize int
	// ReclaimPercent is how far below its ceiling a byte-bound prune reduces a
	// target. Defaults to DefaultReclaimPercent.
	ReclaimPercent int
	// BatchPause is how long the pruner waits between delete batches. Defaults
	// to DefaultBatchPause for a scheduled engine, which shares its database
	// with live traffic; an operator command passes its own.
	BatchPause time.Duration
	// Now supplies the current time. Defaults to time.Now.
	Now func() time.Time
	// Clock drives the scheduler. Defaults to SystemClock.
	Clock Clock
	// FreeSpace probes available bytes. Defaults to FreeSpace.
	FreeSpace FreeSpaceFunc
	// Logger receives every cycle result. Defaults to slog.Default.
	Logger *slog.Logger
	// OnFinding receives a Finding for every budget bound by its byte ceiling.
	OnFinding func(Finding)
}

// EnforcementReceipt is the durable lifecycle evidence for one retention
// owner. LastCycleTime records that the scheduler ran; LastEnforcementTime is
// populated only when the cycle completed without an engine error. Keeping the
// receipt in the owner-scoped state class lets independent tools report
// enforcement without reaching into the owner's process or database.
type EnforcementReceipt struct {
	Owner               string     `json:"owner"`
	LastCycleTime       time.Time  `json:"last_cycle_time"`
	LastEnforcementTime *time.Time `json:"last_enforcement_time,omitempty"`
	LastError           string     `json:"last_error,omitempty"`
}

const enforcementReceiptRelativePath = "retention/enforcement-receipt.json"

// EnforcementReceiptPath resolves the owner-scoped receipt location without
// consulting the caller's current scenario namespace. This is intentionally
// usable by fleet observers such as storage-manager.
func EnforcementReceiptPath(ownerID string) (string, error) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return "", fmt.Errorf("retention receipt: owner id is required")
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		return "", fmt.Errorf("build storage resolver: %w", err)
	}
	return resolver.Path(storage.Options{ScenarioID: ownerID}, storage.ClassState, enforcementReceiptRelativePath)
}

// ReadEnforcementReceipt reads the last durable receipt for an owner. A
// missing receipt is reported as an ordinary filesystem error so callers can
// distinguish "never enforced" from a malformed receipt.
func ReadEnforcementReceipt(ownerID string) (EnforcementReceipt, error) {
	path, err := EnforcementReceiptPath(ownerID)
	if err != nil {
		return EnforcementReceipt{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return EnforcementReceipt{}, err
	}
	var receipt EnforcementReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return EnforcementReceipt{}, fmt.Errorf("decode retention receipt %s: %w", path, err)
	}
	return receipt, nil
}

// OwnerKind identifies the manifest owner whose storage budgets are enforced.
// The wire contract is shared by scenarios, resources, tools, and safeguards;
// keeping the owner identity here prevents each owner type from inventing a
// separate retention bootstrap path.
type OwnerKind string

const (
	OwnerScenario  OwnerKind = "scenario"
	OwnerResource  OwnerKind = "resource"
	OwnerTool      OwnerKind = "tool"
	OwnerSafeguard OwnerKind = "safeguard"
)

// OwnerConfig is the owner-neutral form of ScenarioConfig. Owner-specific
// manifests still resolve through the same bounded retention engine and
// resolver, while the identity is retained for diagnostics and future owner
// roots.
type OwnerConfig struct {
	Kind OwnerKind
	ID   string
	ScenarioConfig
}

// OwnerDiscovery is the deterministic manifest census used by retention
// bootstrap code. Findings remain data so callers can surface malformed owners
// without preventing healthy owners from starting.
type OwnerDiscovery struct {
	Configs  []OwnerConfig
	Findings []storage.InventoryFinding
}

// DiscoverOwners loads native manifests for every supported owner kind.
func DiscoverOwners(repoRoot string) (OwnerDiscovery, error) {
	inventory, err := storage.LoadOwnerInventory(storage.InventoryOptions{RepoRoot: repoRoot})
	if err != nil {
		return OwnerDiscovery{}, err
	}
	discovery := OwnerDiscovery{Findings: inventory.Findings}
	for _, owner := range inventory.Owners {
		if owner.ID == "" {
			continue
		}
		discovery.Configs = append(discovery.Configs, OwnerConfig{
			Kind: OwnerKind(owner.Kind), ID: owner.ID,
			ScenarioConfig: ScenarioConfig{ManifestPath: owner.ManifestPath, Scenario: owner.ID},
		})
	}
	return discovery, nil
}

// Manager owns a component's retention engine and its scheduler.
type Manager struct {
	engine      *Engine
	scheduler   *Scheduler
	scenario    string
	paths       map[string]string
	log         *slog.Logger
	ownerKind   OwnerKind
	ownerID     string
	receiptPath string
	now         func() time.Time

	// specs and openDatabase are retained for the unbudgeted-table audit, which
	// asks what the manifest did NOT declare and so needs both the declaration
	// and a way to look at the file.
	specs        []Spec
	openDatabase func(path string) (Execer, error)

	startOnce sync.Once
	stopOnce  sync.Once
	cancel    context.CancelFunc
}

// NewForScenario builds a retention manager from a component's own manifest.
//
// A manifest with no retention block yields a manager with no budgets and no
// error: declaring one is not mandatory, and a component that declares nothing
// must keep working unchanged. Start and Stop on such a manager are no-ops.
func NewForScenario(cfg ScenarioConfig) (*Manager, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	manifest, err := loadManifest(cfg)
	if err != nil {
		return nil, err
	}
	specs, err := ParseManifest(manifest)
	if err != nil {
		return nil, err
	}

	// The namespace root is the variant-aware identity the lifecycle injects.
	// Resolving through it — rather than hardcoding the slug — is what makes a
	// shadow variant prune its own data and never live's.
	fallback := strings.TrimSpace(cfg.Scenario)
	if fallback == "" {
		fallback = scenario.Name()
	}
	scenarioID, err := storage.ScenarioNamespace(fallback)
	if err != nil {
		return nil, fmt.Errorf("resolve storage namespace: %w", err)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		return nil, fmt.Errorf("build storage resolver: %w", err)
	}
	opts := storage.Options{ScenarioID: scenarioID, RootOverride: cfg.RootOverride}

	paths := make(map[string]string, len(specs))
	resolvePath := func(target Target) (string, error) {
		return target.Resolve(resolver, opts)
	}
	for _, spec := range specs {
		path, err := resolvePath(spec.Target)
		if err != nil {
			return nil, fmt.Errorf("budget %q: %w", spec.Budget.Name, err)
		}
		paths[spec.Budget.Name] = path
	}
	receiptPath, err := resolver.Path(opts, storage.ClassState, enforcementReceiptRelativePath)
	if err != nil {
		return nil, fmt.Errorf("resolve enforcement receipt: %w", err)
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}

	m := &Manager{
		scenario:     scenarioID,
		paths:        paths,
		log:          logger,
		specs:        specs,
		openDatabase: cfg.OpenDatabase,
		receiptPath:  receiptPath,
		now:          now,
	}

	// A scheduled cycle gets a wall-clock allowance. On a table with hundreds of
	// millions of rows to remove, an unbounded cycle would hold the write lock
	// for hours and starve the ingest path it is protecting; it stops cleanly,
	// reports Incomplete, and resumes on the next tick.
	cycleDuration := cfg.MaxDuration
	if cycleDuration == 0 {
		cycleDuration = DefaultCycleDuration
	}

	// A scheduled cycle runs against a database that is serving traffic at the
	// same time, so it yields between batches by default. An operator command
	// builds its pruner directly and gets no pause, because there is nothing to
	// yield to and the operator is waiting on the result.
	batchPause := cfg.BatchPause
	if batchPause == 0 {
		batchPause = DefaultBatchPause
	}

	var builtin BuiltinFactory
	if len(specs) > 0 {
		builtin, err = NewBuiltinFactory(BuiltinConfig{
			ResolvePath:     resolvePath,
			OpenDatabase:    cfg.OpenDatabase,
			BatchSize:       cfg.BatchSize,
			ReclaimPercent:  cfg.ReclaimPercent,
			BatchPause:      batchPause,
			AllowFullVacuum: cfg.AllowFullVacuum,
			MaxDuration:     cycleDuration,
			Now:             cfg.Now,
			FreeSpace:       cfg.FreeSpace,
			Logger:          logger,
		})
		if err != nil {
			return nil, err
		}
	}

	engine, err := NewEngine(EngineConfig{
		Specs:    specs,
		Builtin:  builtin,
		Registry: cfg.Registry,
		Now:      cfg.Now,
		Observe:  m.observe(cfg.OnFinding),
	})
	if err != nil {
		return nil, err
	}
	m.engine = engine

	scheduler, err := NewScheduler(SchedulerConfig{
		Engine:     engine,
		Interval:   cfg.Interval,
		Clock:      cfg.Clock,
		RunOnStart: cfg.RunOnStart,
		OnCycle: func(results []Result, err error) {
			m.recordEnforcementReceipt(err)
			if err != nil {
				logger.Error("retention cycle failed", "scenario", scenarioID, "error", err)
			}
			_ = results
		},
	})
	if err != nil {
		return nil, err
	}
	m.scheduler = scheduler
	return m, nil
}

func (m *Manager) recordEnforcementReceipt(cycleErr error) {
	if m == nil || strings.TrimSpace(m.receiptPath) == "" {
		return
	}
	now := time.Now
	if m.now != nil {
		now = m.now
	}
	receipt := EnforcementReceipt{Owner: m.scenario, LastCycleTime: now()}
	if previous, err := os.ReadFile(m.receiptPath); err == nil {
		var prior EnforcementReceipt
		if err := json.Unmarshal(previous, &prior); err == nil {
			receipt.LastEnforcementTime = prior.LastEnforcementTime
		}
	}
	if cycleErr == nil {
		last := receipt.LastCycleTime
		receipt.LastEnforcementTime = &last
	} else {
		receipt.LastError = cycleErr.Error()
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		m.log.Warn("retention receipt encode failed", "scenario", m.scenario, "error", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(m.receiptPath), 0o755); err != nil {
		m.log.Warn("retention receipt directory failed", "scenario", m.scenario, "error", err)
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(m.receiptPath), ".enforcement-receipt-*.tmp")
	if err != nil {
		m.log.Warn("retention receipt temp file failed", "scenario", m.scenario, "error", err)
		return
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		m.log.Warn("retention receipt write failed", "scenario", m.scenario, "error", err)
		return
	}
	if err := tmp.Close(); err != nil {
		m.log.Warn("retention receipt close failed", "scenario", m.scenario, "error", err)
		return
	}
	if err := os.Rename(tmpPath, m.receiptPath); err != nil {
		m.log.Warn("retention receipt publish failed", "scenario", m.scenario, "error", err)
	}
}

// NewForOwner builds retention for any declared storage owner. The current
// storage resolver uses the scenario namespace for all lifecycle-managed
// owners; callers provide RootOverride when a resource/tool has a dedicated
// root. Invalid owner kinds fail closed instead of silently using a scenario
// default.
func NewForOwner(cfg OwnerConfig) (*Manager, error) {
	switch cfg.Kind {
	case OwnerScenario, OwnerResource, OwnerTool, OwnerSafeguard:
	default:
		return nil, fmt.Errorf("unknown retention owner kind %q", cfg.Kind)
	}
	if strings.TrimSpace(cfg.Scenario) == "" {
		cfg.Scenario = strings.TrimSpace(cfg.ID)
	}
	if cfg.ManifestPath == "" {
		start := strings.TrimSpace(cfg.StartDir)
		if start == "" {
			if wd, err := os.Getwd(); err == nil {
				start = wd
			}
		}
		if found, ok := findOwnerManifest(start, cfg.Kind); ok {
			cfg.ManifestPath = found
		}
	}
	manager, err := NewForScenario(cfg.ScenarioConfig)
	if err != nil {
		return nil, err
	}
	manager.ownerKind = cfg.Kind
	manager.ownerID = firstNonEmpty(cfg.ID, cfg.Scenario)
	return manager, nil
}

// observe returns the per-result hook: it logs every cycle, including a skipped
// compaction, and raises a finding when the byte ceiling is what bound the
// result.
func (m *Manager) observe(onFinding func(Finding)) func(Result, Spec) {
	return func(r Result, spec Spec) {
		attrs := []any{
			"scenario", m.scenario,
			"budget", r.Budget,
			"bound_by", r.BoundBy.String(),
			"deleted", r.Deleted,
			"freed_bytes", r.FreedBytes,
			"used_bytes", r.After.Bytes,
			"items", r.After.Items,
			"incomplete", r.Incomplete,
		}
		if r.CompactSkipped {
			// A skipped compaction that is not surfaced is a silent one, and the
			// space it left behind will not be explained by anything else.
			attrs = append(attrs, "compact_skipped", true, "compact_skip_reason", r.CompactSkipReason)
		}
		m.log.Info("retention cycle", attrs...)

		if r.BoundBy != BoundBytes || onFinding == nil {
			return
		}
		onFinding(Finding{
			Scenario:  m.scenario,
			Budget:    r.Budget,
			Target:    m.paths[r.Budget],
			Rationale: spec.Rationale,
			UsedBytes: r.After.Bytes,
			MaxBytes:  spec.Budget.MaxBytes,
			Deleted:   r.Deleted,
		})
	}
}

// Engine returns the underlying engine, for a one-shot cycle outside the
// schedule.
func (m *Manager) Engine() *Engine { return m.engine }

// ScenarioID returns the variant-aware namespace the budgets resolved under.
func (m *Manager) ScenarioID() string { return m.scenario }

// Budgets returns the declared budgets, so a component can log what it is
// enforcing at startup.
func (m *Manager) Budgets() []Budget { return m.engine.Budgets() }

// ResolvedPath returns the absolute path a budget resolved to.
func (m *Manager) ResolvedPath(budget string) (string, bool) {
	path, ok := m.paths[budget]
	return path, ok
}

// Start begins the schedule in the background. It is a no-op when no budgets are
// declared, so a component can call it unconditionally.
func (m *Manager) Start(ctx context.Context) {
	if len(m.engine.Budgets()) == 0 {
		return
	}
	m.startOnce.Do(func() {
		runCtx, cancel := context.WithCancel(ctx)
		m.cancel = cancel

		// The audit runs once at startup, off the critical path. It reports what
		// the manifest does not cover, which is the one thing a set of correct
		// budgets cannot tell you about itself.
		go func() {
			unbudgeted, err := m.AuditUnbudgetedTables(runCtx)
			if err != nil {
				m.log.Warn("retention audit for unbudgeted tables failed",
					"scenario", m.scenario, "error", err)
				return
			}
			for _, table := range unbudgeted {
				m.log.Warn("unbudgeted table in a budgeted database",
					"scenario", m.scenario,
					"database", table.Database,
					"table", table.Table,
					"bytes", table.Bytes,
					"detail", table.String())
			}
		}()

		go m.scheduler.Run(runCtx)
	})
}

// Stop halts the schedule and waits for an in-flight cycle to return, so
// shutdown does not race a prune mid-batch.
func (m *Manager) Stop() {
	m.stopOnce.Do(func() {
		if m.cancel == nil {
			return
		}
		m.cancel()
		<-m.scheduler.Done()
	})
}

// loadManifest resolves the manifest bytes from whichever source cfg supplies.
func loadManifest(cfg ScenarioConfig) ([]byte, error) {
	if cfg.Manifest != nil {
		return cfg.Manifest, nil
	}
	path := strings.TrimSpace(cfg.ManifestPath)
	if path == "" {
		start := strings.TrimSpace(cfg.StartDir)
		if start == "" {
			wd, err := os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("locate manifest: %w", err)
			}
			start = wd
		}
		found, ok := findManifest(start)
		if !ok {
			// No manifest is not an error: it means no budgets, which is the
			// state every component starts in.
			return []byte(`{}`), nil
		}
		path = found
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}
	return data, nil
}

// findManifest walks up from dir looking for the nearest .vrooli/service.json,
// mirroring how projectmeta locates the same file.
func findManifest(dir string) (string, bool) {
	for {
		candidate := filepath.Join(dir, ".vrooli", "service.json")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func findOwnerManifest(dir string, kind OwnerKind) (string, bool) {
	name := map[OwnerKind]string{
		OwnerScenario: "service.json", OwnerResource: "resource.json",
		OwnerTool: "tool.json", OwnerSafeguard: "safeguard.json",
	}[kind]
	if name == "" {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, name)
		if kind == OwnerScenario {
			candidate = filepath.Join(dir, ".vrooli", name)
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
