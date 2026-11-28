package modelstorage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"github.com/vrooli/vrooli/packages/scheduler"

	"image-tools/internal/fetch"
	"image-tools/internal/jobs"
	"image-tools/internal/models"
)

// Errors callers map to actionable responses.
var (
	// ErrAutoAcquireUnavailable: the model is not installed and cannot be
	// acquired automatically (switch off, caller opt-out, source not governed,
	// rights or enablement refusal). The wrapped message names the reason and
	// the explicit recovery command.
	ErrAutoAcquireUnavailable = errors.New("model weights cannot be acquired automatically")
	// ErrAcquireInsufficientDisk: even after offloading eligible idle weights,
	// headroom above the shared disk floor is below what the acquisition needs.
	ErrAcquireInsufficientDisk = errors.New("not enough disk headroom to acquire model weights")
	// ErrAcquireDiskBusy: free space covers the acquisition, but shared disk
	// admission refused it because other jobs hold active reservations. It
	// wraps jobs.ErrCapacityBusy, so the job requeues and waits.
	ErrAcquireDiskBusy = errors.New("shared disk is reserved by other jobs")
)

// Offload triggers recorded on receipts.
const (
	TriggerPeriodic   = "periodic"
	TriggerAcquire    = "pre_acquire"
	TriggerRequested  = "operator"
	TriggerBoot       = "boot"
	ownerJobKeyPrefix = "image-tools:auto-acquire"
)

// Installer is the subset of models.Installer this package drives. The
// Installer remains the only owner of downloads, verification and removal.
type Installer interface {
	Installed(ctx context.Context, id string) bool
	InstallForJob(ctx context.Context, id, jobID string, emit func(progress int, message string)) (models.InstallRecord, error)
	Remove(ctx context.Context, id string) error
}

// Deps wires the facts and effects the service needs.
type Deps struct {
	// Root is the models root; free space is observed on its filesystem.
	Root      string
	Installer Installer
	// Seed resolves the read-only seed catalog. Custom and imported entries
	// are not seed models and are never acquired or offloaded automatically.
	Seed func(id string) (models.Model, bool)
	// Custom lists operator-registered/imported entries (may be nil).
	Custom func(ctx context.Context) ([]models.Model, error)
	// InstallRecords lists durable install receipts.
	InstallRecords func(ctx context.Context) ([]models.InstallRecord, error)
	// LastSuccessfulUse returns the latest successful durable job per model.
	LastSuccessfulUse func(ctx context.Context) (map[string]time.Time, error)
	// ActiveModelIDs returns models referenced by queued/admitting/running jobs.
	ActiveModelIDs func(ctx context.Context) (map[string]bool, error)
	// InUse reports whether another reader or writer holds the model's shared
	// use lease right now (optional; removal re-checks under the exclusive lease).
	InUse func(id string) bool
	// Enabled reports the model's effective enabled state.
	Enabled func(ctx context.Context, m models.Model) (bool, error)
	// RuntimeReady reports the installed model's runtime lifecycle (ready and
	// reason). With it, EnsureInstalled revalidates installed-but-not-ready
	// models whose reason the Installer can resolve.
	RuntimeReady func(ctx context.Context, id string) (bool, string)
	// BackendReady is the installer's post-install backend check. A model
	// whose native backend cannot serve its operations would install as
	// needs_backend, so auto-acquisition refuses it before downloading.
	BackendReady func(ctx context.Context, m models.Model) error
	// Rights is the shared current-context rights owner.
	Rights    models.RightsEvaluator
	DiskAvail fetch.DiskSpaceFunc
	Store     *Store
	// Measure returns allocated bytes for a model path (MeasureAllocated).
	Measure func(path string) (int64, error)
	// AcquireNeedBytes is the full disk peak an acquisition reserves.
	AcquireNeedBytes func(ctx context.Context, m models.Model) (int64, error)
	Now              func() time.Time
	Logger           *log.Logger
	// StagingRoots overrides the directories scanned for acquisition staging
	// (default: Root/staging/{models,adapters}, plus Root/{models,adapters} for
	// leftovers of earlier releases).
	StagingRoots []string
}

// Config holds the service switches and the policy.
type Config struct {
	Policy Policy
	// AutoAcquire lets jobs acquire missing governed weights (default on).
	AutoAcquire bool
	// AutoOffload lets the periodic and pre-acquisition passes remove weights
	// (default on). Status and dry-runs work either way.
	AutoOffload  bool
	Interval     time.Duration
	StartupDelay time.Duration
	// StagingGrace is how long an unowned staging directory must be silent
	// before it is abandoned (default DefaultStagingGrace).
	StagingGrace time.Duration
	// StagingBootDelay postpones the boot sweep so durable jobs requeued by the
	// restart can resume their partial staging first (default 2m).
	StagingBootDelay time.Duration
}

// Service is the model-storage lifecycle owner.
type Service struct {
	deps Deps
	cfg  Config

	passMu    sync.Mutex
	sweepMu   sync.Mutex
	acquireMu sync.Mutex
	acquiring map[string]chan struct{}
}

// New validates deps.
func New(deps Deps, cfg Config) (*Service, error) {
	if deps.Installer == nil || deps.Seed == nil || deps.InstallRecords == nil || deps.DiskAvail == nil || deps.Store == nil {
		return nil, errors.New("modelstorage: Installer, Seed, InstallRecords, DiskAvail and Store are required")
	}
	if strings.TrimSpace(deps.Root) == "" {
		return nil, errors.New("modelstorage: models root is required")
	}
	if deps.Measure == nil {
		deps.Measure = MeasureAllocated
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	if cfg.Policy == (Policy{}) {
		cfg.Policy = DefaultPolicy()
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Minute
	}
	return &Service{deps: deps, cfg: cfg, acquiring: map[string]chan struct{}{}}, nil
}

// Config returns the effective configuration.
func (s *Service) Config() Config { return s.cfg }

// Report is one evaluation of the policy against current facts.
type Report struct {
	ObservedAt  time.Time `json:"observed_at"`
	Trigger     string    `json:"trigger"`
	Policy      Policy    `json:"policy"`
	AutoAcquire bool      `json:"auto_acquire"`
	AutoOffload bool      `json:"auto_offload"`
	Selection
	InstalledBytes int64 `json:"installed_bytes"`
	// Staging totals acquisition staging directories, in progress or left behind.
	Staging StagingSummary `json:"staging"`
}

// Plan evaluates the policy without changing anything. needBytes is extra
// headroom a pending acquisition requires; requested restricts selection to
// named models (an operator's targeted offload).
func (s *Service) Plan(ctx context.Context, trigger string, needBytes int64, requested []string) (Report, error) {
	free, err := s.deps.DiskAvail(s.deps.Root)
	if err != nil {
		return Report{}, fmt.Errorf("modelstorage: observe free disk at %s: %w", s.deps.Root, err)
	}
	candidates, err := s.candidates(ctx)
	if err != nil {
		return Report{}, err
	}
	// The threshold reflects the headroom left once a pending acquisition
	// lands; the reported headroom stays the observed value.
	pressure := s.cfg.Policy.Evaluate(free - max(needBytes, 0))
	pressure.FreeBytes, pressure.HeadroomBytes = free, free-s.cfg.Policy.FloorBytes
	selection := s.cfg.Policy.Select(candidates, pressure, needBytes, requested)
	report := Report{ObservedAt: s.deps.Now().UTC(), Trigger: trigger, Policy: s.cfg.Policy, AutoAcquire: s.cfg.AutoAcquire, AutoOffload: s.cfg.AutoOffload, Selection: selection}
	for _, c := range candidates {
		report.InstalledBytes += c.Bytes
	}
	staging, err := s.Staging(ctx)
	if err != nil {
		s.deps.Logger.Printf("modelstorage: staging scan: %v", err)
	}
	report.Staging = summarizeStaging(staging, s.stagingGrace())
	return report, nil
}

// candidates gathers every installed, weight-backed model with its evidence
// and protections.
func (s *Service) candidates(ctx context.Context) ([]Candidate, error) {
	records, err := s.deps.InstallRecords(ctx)
	if err != nil {
		return nil, fmt.Errorf("modelstorage: list install receipts: %w", err)
	}
	custom := map[string]models.Model{}
	if s.deps.Custom != nil {
		list, err := s.deps.Custom(ctx)
		if err != nil {
			return nil, fmt.Errorf("modelstorage: list custom models: %w", err)
		}
		for _, m := range list {
			custom[m.ID] = m
		}
	}
	lastUse := map[string]time.Time{}
	if s.deps.LastSuccessfulUse != nil {
		if lastUse, err = s.deps.LastSuccessfulUse(ctx); err != nil {
			return nil, fmt.Errorf("modelstorage: read successful model use: %w", err)
		}
	}
	active := map[string]bool{}
	if s.deps.ActiveModelIDs != nil {
		if active, err = s.deps.ActiveModelIDs(ctx); err != nil {
			return nil, fmt.Errorf("modelstorage: read active jobs: %w", err)
		}
	}
	kept, err := s.deps.Store.Kept(ctx)
	if err != nil {
		return nil, err
	}
	now := s.deps.Now()
	out := make([]Candidate, 0, len(records))
	for _, rec := range records {
		if !rec.Installed || strings.TrimSpace(rec.Path) == "" {
			continue
		}
		model, seed := s.deps.Seed(rec.ID)
		if !seed {
			if m, ok := custom[rec.ID]; ok {
				model = m
			} else {
				model = models.Model{ID: rec.ID, Backend: "unknown"}
			}
		}
		if !model.RequiresWeights() {
			continue
		}
		c := Candidate{ModelID: rec.ID, InstalledAt: rec.InstalledAt, Checksum: rec.Checksum, LastSuccessfulUse: lastUse[rec.ID]}
		if model.Source.LocalPath == "" {
			bytes, err := s.deps.Measure(rec.Path)
			if err == nil {
				c.Bytes, c.BytesBasis = bytes, "allocated"
			} else {
				c.Bytes, c.BytesBasis = rec.SizeBytes, "install_receipt"
			}
		} else {
			c.BytesBasis = "external_reference"
		}
		switch {
		case !c.LastSuccessfulUse.IsZero() && !c.LastSuccessfulUse.Before(c.InstalledAt):
			c.LastUse, c.LastUseBasis = c.LastSuccessfulUse, UseBasisSuccessfulJob
		case !c.InstalledAt.IsZero():
			c.LastUse, c.LastUseBasis = c.InstalledAt, UseBasisInstall
		default:
			c.LastUseBasis = UseBasisUnknown
		}
		if !c.LastUse.IsZero() && now.After(c.LastUse) {
			c.Idle = now.Sub(c.LastUse)
		}
		if reason := reacquireBlocker(model, seed); reason != "" {
			c.Protected = append(c.Protected, reason)
		}
		if _, ok := kept[rec.ID]; ok {
			c.Protected = append(c.Protected, ProtectKept)
		}
		if active[rec.ID] {
			c.Protected = append(c.Protected, ProtectActiveJob)
		}
		if s.deps.InUse != nil && s.deps.InUse(rec.ID) {
			c.Protected = append(c.Protected, ProtectInUse)
		}
		if !c.InstalledAt.IsZero() && now.Sub(c.InstalledAt) < s.cfg.Policy.RecentInstall {
			c.Protected = append(c.Protected, ProtectRecentlyInstalled)
		}
		if c.LastUseBasis == UseBasisUnknown {
			c.Protected = append(c.Protected, ProtectUseUnknown)
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModelID < out[j].ModelID })
	return out, nil
}

// reacquireBlocker names why weights could not come back automatically.
func reacquireBlocker(m models.Model, seed bool) string {
	if !seed || m.Source.Manual || strings.TrimSpace(m.Source.LocalPath) != "" || !m.HasFetchStrategy() {
		return ProtectNotReacquirable
	}
	if m.BuildProfile != nil {
		return ProtectLocallyBuilt
	}
	return ""
}

// Apply offloads the report's selected candidates and writes one durable
// receipt per attempt. Protections are re-read immediately before removal;
// the Installer's exclusive use lease is the final in-use guard.
func (s *Service) Apply(ctx context.Context, report Report) ([]Receipt, error) {
	s.passMu.Lock()
	defer s.passMu.Unlock()
	return s.applyLocked(ctx, report)
}

func (s *Service) applyLocked(ctx context.Context, report Report) ([]Receipt, error) {
	selected := make([]Candidate, 0)
	for _, c := range report.Candidates {
		if c.Selected {
			selected = append(selected, c)
		}
	}
	if len(selected) == 0 {
		return nil, nil
	}
	kept, err := s.deps.Store.Kept(ctx)
	if err != nil {
		return nil, err
	}
	active := map[string]bool{}
	if s.deps.ActiveModelIDs != nil {
		if active, err = s.deps.ActiveModelIDs(ctx); err != nil {
			return nil, fmt.Errorf("modelstorage: re-read active jobs before offload: %w", err)
		}
	}
	threshold := int64(-1)
	if !report.Pressure.Unlimited {
		threshold = int64(report.Pressure.IdleThreshold / time.Second)
	}
	var receipts []Receipt
	var errs []error
	for _, c := range selected {
		receipt := Receipt{
			ID: uuid.NewString(), ModelID: c.ModelID, Trigger: report.Trigger, Reason: c.SelectionReason,
			Bytes: c.Bytes, Checksum: c.Checksum, LastUseAt: c.LastUse, LastUseBasis: c.LastUseBasis,
			IdleSeconds: int64(c.Idle / time.Second), IdleThresholdSeconds: threshold, Mode: report.Pressure.Mode,
		}
		receipt.FreeBeforeBytes, _ = s.deps.DiskAvail(s.deps.Root)
		switch {
		case kept[c.ModelID].ModelID != "":
			receipt.Outcome, receipt.Error = OutcomeSkipped, ProtectKept
		case active[c.ModelID]:
			receipt.Outcome, receipt.Error = OutcomeSkipped, ProtectActiveJob
		default:
			if err := s.deps.Installer.Remove(ctx, c.ModelID); err != nil {
				receipt.Outcome, receipt.Error = OutcomeFailed, err.Error()
				if errors.Is(err, models.ErrModelInUse) {
					receipt.Outcome, receipt.Error = OutcomeSkipped, ProtectInUse
				}
			} else {
				receipt.Outcome = OutcomeRemoved
			}
		}
		receipt.FreeAfterBytes, _ = s.deps.DiskAvail(s.deps.Root)
		receipt.CreatedAt = s.deps.Now().UTC()
		if err := s.deps.Store.RecordReceipt(context.WithoutCancel(ctx), receipt); err != nil {
			errs = append(errs, err)
		}
		s.deps.Logger.Printf("modelstorage: offload %s outcome=%s reason=%s trigger=%s bytes=%d idle=%s mode=%s %s",
			c.ModelID, receipt.Outcome, receipt.Reason, receipt.Trigger, receipt.Bytes, c.Idle.Round(time.Minute), receipt.Mode, receipt.Error)
		receipts = append(receipts, receipt)
	}
	return receipts, errors.Join(errs...)
}

// Offload plans and, unless dryRun, applies one pass under the pass lock.
func (s *Service) Offload(ctx context.Context, trigger string, needBytes int64, requested []string, dryRun bool) (Report, []Receipt, error) {
	s.passMu.Lock()
	defer s.passMu.Unlock()
	report, err := s.Plan(ctx, trigger, needBytes, requested)
	if err != nil || dryRun {
		return report, nil, err
	}
	receipts, err := s.applyLocked(ctx, report)
	return report, receipts, err
}

// Receipts returns recent offload receipts.
func (s *Service) Receipts(ctx context.Context, limit int) ([]Receipt, error) {
	return s.deps.Store.Receipts(ctx, limit)
}

// SetKeep pins or unpins a model. Only installable catalog ids are accepted.
func (s *Service) SetKeep(ctx context.Context, id string, keep bool, reason string) error {
	id = strings.TrimSpace(id)
	if _, ok := s.deps.Seed(id); !ok {
		found := false
		if s.deps.Custom != nil {
			list, err := s.deps.Custom(ctx)
			if err != nil {
				return err
			}
			for _, m := range list {
				found = found || m.ID == id
			}
		}
		if !found {
			return fmt.Errorf("%w: %q", models.ErrModelNotFound, id)
		}
	}
	return s.deps.Store.SetKeep(ctx, id, keep, reason, s.deps.Now())
}

// Start runs the periodic offload loop until ctx ends. The first pass runs
// after StartupDelay (or one interval when zero). Abandoned staging is also
// reclaimed once shortly after boot, in the background, so readiness never
// waits on it.
func (s *Service) Start(ctx context.Context) {
	cycle := func(ctx context.Context) error {
		passCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		report, receipts, err := s.Offload(passCtx, TriggerPeriodic, 0, nil, !s.cfg.AutoOffload)
		removed := int64(0)
		for _, r := range receipts {
			if r.Outcome == OutcomeRemoved {
				removed += r.Bytes
			}
		}
		s.deps.Logger.Printf("modelstorage: periodic pass mode=%s headroom=%d idle_threshold=%s selected=%d removed_bytes=%d auto_offload=%t err=%v",
			report.Pressure.Mode, report.Pressure.HeadroomBytes, thresholdText(report.Pressure), report.SelectedBytes, removed, s.cfg.AutoOffload, err)
		return errors.Join(err, s.sweepPass(passCtx, TriggerPeriodic))
	}
	runner := scheduler.New(s.cfg.Interval, cycle)
	runner.Start(ctx)
	if s.cfg.StartupDelay > 0 {
		go func() {
			timer := time.NewTimer(s.cfg.StartupDelay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
			case <-timer.C:
				_ = runner.RunOnce(ctx)
			}
		}()
	}
	go func() {
		delay := s.cfg.StagingBootDelay
		if delay <= 0 {
			delay = DefaultStagingBootDelay
		}
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
		case <-timer.C:
			passCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()
			_ = s.sweepPass(passCtx, TriggerBoot)
		}
	}()
}

// sweepPass reclaims abandoned staging when automatic offload is on and only
// reports it otherwise, logging one summary line.
func (s *Service) sweepPass(ctx context.Context, trigger string) error {
	reclaimed, err := s.SweepStaging(ctx, trigger, !s.cfg.AutoOffload)
	if len(reclaimed) > 0 || err != nil {
		s.deps.Logger.Printf("modelstorage: staging sweep trigger=%s auto_offload=%t %s err=%v", trigger, s.cfg.AutoOffload, stagingText(reclaimed), err)
	}
	return err
}

func thresholdText(p Pressure) string {
	if p.Unlimited {
		return "none"
	}
	return p.IdleThreshold.Round(time.Minute).String()
}

// AutoAcquirable reports nil when m may be acquired automatically now, or an
// ErrAutoAcquireUnavailable-wrapped reason naming the explicit recovery.
func (s *Service) AutoAcquirable(ctx context.Context, m models.Model) error {
	return s.eligible(ctx, m, true)
}

// Revalidatable reports nil when an installed m whose runtime lifecycle is
// not ready may be revalidated automatically. Revalidation reuses the weights
// on disk, so the governed-source gates do not apply; the switch, enabled
// state, rights, smoke contract and backend readiness still do.
func (s *Service) Revalidatable(ctx context.Context, m models.Model, reason string) error {
	if !RevalidatableRuntimeReason(reason) {
		return fmt.Errorf("%w: %q is not runtime-ready (%s), which revalidation cannot resolve — run `image-tools models doctor` and resolve the reported issue", ErrAutoAcquireUnavailable, m.ID, reason)
	}
	return s.eligible(ctx, m, false)
}

func (s *Service) eligible(ctx context.Context, m models.Model, download bool) error {
	refuse := func(reason string) error {
		return fmt.Errorf("%w: %q %s — run `image-tools models install %s`", ErrAutoAcquireUnavailable, m.ID, reason, m.ID)
	}
	if !s.cfg.AutoAcquire {
		if download {
			return refuse("is not installed and automatic acquisition is switched off (IMAGE_TOOLS_MODEL_AUTO_ACQUIRE)")
		}
		return refuse("is not runtime-ready and automatic revalidation is switched off (IMAGE_TOOLS_MODEL_AUTO_ACQUIRE)")
	}
	resolved, seed := s.deps.Seed(m.ID)
	if seed {
		m = resolved
	} else if custom, ok := s.customModel(ctx, m.ID); ok && !download {
		m = custom
	} else if download {
		return refuse("is a custom or imported entry; only governed catalog sources are acquired automatically")
	} else {
		return fmt.Errorf("%w: %q is not in the catalog", ErrAutoAcquireUnavailable, m.ID)
	}
	if !m.RequiresWeights() {
		return nil
	}
	if download && (m.Source.Manual || strings.TrimSpace(m.Source.LocalPath) != "" || !m.HasFetchStrategy()) {
		return fmt.Errorf("%w: %q has a manual source and must be installed per its docs_url (%s)", ErrAutoAcquireUnavailable, m.ID, m.Source.DocsURL)
	}
	if models.NeedsLoadSmokeContract(m) && !models.HasLoadSmokeContract(m) {
		// The install would end smoke_unsupported and the job would fail after
		// the download; refuse before spending bandwidth and disk.
		return fmt.Errorf("%w: %q has no load-smoke contract for its %s artifacts, so its weights could never become runtime-ready — see `image-tools models doctor`", ErrAutoAcquireUnavailable, m.ID, m.Backend)
	}
	if s.deps.Enabled != nil {
		enabled, err := s.deps.Enabled(ctx, m)
		if err != nil {
			return fmt.Errorf("modelstorage: read enabled state for %q: %w", m.ID, err)
		}
		if !enabled {
			return fmt.Errorf("%w: %q is disabled — enable it with `image-tools models enable %s`", ErrAutoAcquireUnavailable, m.ID, m.ID)
		}
	}
	if err := models.RequireLocalUsePermitted(ctx, s.deps.Rights, m); err != nil {
		return fmt.Errorf("%w: %w", ErrAutoAcquireUnavailable, err)
	}
	if s.deps.BackendReady != nil {
		if err := s.deps.BackendReady(ctx, m); err != nil {
			return fmt.Errorf("%w: %q would install without a runnable backend (%v) — see `image-tools backends doctor`", ErrAutoAcquireUnavailable, m.ID, err)
		}
	}
	return nil
}

func (s *Service) customModel(ctx context.Context, id string) (models.Model, bool) {
	if s.deps.Custom == nil {
		return models.Model{}, false
	}
	customs, err := s.deps.Custom(ctx)
	if err != nil {
		return models.Model{}, false
	}
	for _, m := range customs {
		if m.ID == id {
			return m, true
		}
	}
	return models.Model{}, false
}

// EnsureInstalled makes model id's weights present for a job, acquiring them
// through the Installer and shared disk admission when they are missing. When
// headroom would fall below the reserve, eligible idle weights are offloaded
// first. Concurrent callers for the same model wait for one acquisition.
func (s *Service) EnsureInstalled(ctx context.Context, id, jobID string, emit func(progress int, message string)) error {
	if emit == nil {
		emit = func(int, string) {}
	}
	if s.deps.Installer.Installed(ctx, id) && s.revalidationReason(ctx, id) == "" {
		return nil
	}
	release, err := s.lockAcquire(ctx, id)
	if err != nil {
		return err
	}
	defer release()
	installed := s.deps.Installer.Installed(ctx, id)
	reason := ""
	if installed {
		if reason = s.revalidationReason(ctx, id); reason == "" {
			return nil
		}
	}
	m, ok := s.deps.Seed(id)
	if !ok {
		m = models.Model{ID: id}
	}
	if installed {
		if err := s.Revalidatable(ctx, m, reason); err != nil {
			return err
		}
		return s.revalidate(ctx, m, jobID, reason, emit)
	}
	if err := s.AutoAcquirable(ctx, m); err != nil {
		return err
	}
	need := int64(m.SizeMBApprox) << 20 * 2
	if s.deps.AcquireNeedBytes != nil {
		if planned, err := s.deps.AcquireNeedBytes(ctx, m); err == nil && planned > 0 {
			need = planned
		}
	}
	policy := s.cfg.Policy
	free, err := s.deps.DiskAvail(s.deps.Root)
	if err != nil {
		return fmt.Errorf("modelstorage: observe free disk before acquiring %q: %w", id, err)
	}
	offloaded := 0
	if free-policy.FloorBytes < policy.ReserveBytes+need && s.cfg.AutoOffload {
		// Abandoned staging is pure waste: reclaim it before evicting weights.
		if _, err := s.SweepStaging(ctx, TriggerAcquire+":"+id, false); err != nil {
			s.deps.Logger.Printf("modelstorage: pre-acquire staging sweep for %s: %v", id, err)
		}
		if free, err = s.deps.DiskAvail(s.deps.Root); err != nil {
			return fmt.Errorf("modelstorage: observe free disk before acquiring %q: %w", id, err)
		}
	}
	if free-policy.FloorBytes < policy.ReserveBytes+need && s.cfg.AutoOffload {
		emit(0, fmt.Sprintf("making room to acquire %s: offloading idle model weights", id))
		_, receipts, err := s.Offload(ctx, TriggerAcquire+":"+id, need, nil, false)
		if err != nil {
			s.deps.Logger.Printf("modelstorage: pre-acquire offload for %s: %v", id, err)
		}
		for _, r := range receipts {
			if r.Outcome == OutcomeRemoved {
				offloaded++
			}
		}
		if free, err = s.deps.DiskAvail(s.deps.Root); err != nil {
			return fmt.Errorf("modelstorage: observe free disk before acquiring %q: %w", id, err)
		}
	}
	if headroom := free - policy.FloorBytes; headroom < need {
		return fmt.Errorf("%w: %q needs about %s above the %s shared disk floor but only %s is free above it after offloading %d idle model(s); free disk space, unkeep pinned models (`image-tools models unkeep <id>`), or review `image-tools models storage`",
			ErrAcquireInsufficientDisk, id, humanBytes(need), humanBytes(policy.FloorBytes), humanBytes(max(headroom, 0)), offloaded)
	}
	started := s.deps.Now()
	emit(0, fmt.Sprintf("acquiring weights for %s (~%d MB)", id, m.SizeMBApprox))
	// Disk receipts are immutable per owner/job key and admission can be
	// retried for the same durable job, so every attempt gets its own key.
	attempt := ownerJobKeyPrefix + ":" + strings.TrimSpace(jobID) + ":" + uuid.NewString()
	_, err = s.deps.Installer.InstallForJob(ctx, id, attempt, func(progress int, message string) {
		emit(progress, fmt.Sprintf("acquiring weights for %s: %s", id, message))
	})
	if errors.Is(err, sharedcapacity.ErrDiskAdmissionDenied) {
		// Raw headroom was checked above, so the denial is other owners'
		// in-flight reservations: wait for them like any capacity claim.
		return fmt.Errorf("%w: %w: waiting to acquire weights for %s: %v", ErrAcquireDiskBusy, jobs.ErrCapacityBusy, id, err)
	}
	if err != nil {
		return fmt.Errorf("modelstorage: automatic acquisition of %q failed: %w", id, err)
	}
	if !s.deps.Installer.Installed(ctx, id) {
		return fmt.Errorf("modelstorage: automatic acquisition of %q finished but the weights are not installed", id)
	}
	s.deps.Logger.Printf("modelstorage: acquired %s for job %s in %s", id, jobID, s.deps.Now().Sub(started).Round(time.Second))
	return nil
}

// RevalidatableRuntimeReason reports whether an installed model's lifecycle
// reason is one the Installer resolves by re-running its install path on the
// weights already on disk: a pending or stale load smoke, a missing local
// build output, or a backend that may have been provisioned since. A failed
// fresh smoke, an unsupported smoke shape or an unprovisioned environment
// would fail the same way again, so those still need the operator.
func RevalidatableRuntimeReason(reason string) bool {
	switch reason {
	case "smoke_pending", "build_not_ready", "needs_backend":
		return true
	}
	return false
}

// revalidationReason returns the lifecycle reason when an installed model is
// not runtime-ready for a reason revalidation can resolve, else "".
func (s *Service) revalidationReason(ctx context.Context, id string) string {
	if s.deps.RuntimeReady == nil {
		return ""
	}
	ready, reason := s.deps.RuntimeReady(ctx, id)
	if ready || !RevalidatableRuntimeReason(reason) {
		return ""
	}
	return reason
}

// revalidate re-runs the Installer on weights already on disk. Valid
// artifacts are not re-downloaded and plan no disk peak; only the smoke,
// build or backend step runs.
func (s *Service) revalidate(ctx context.Context, m models.Model, jobID, reason string, emit func(int, string)) error {
	started := s.deps.Now()
	emit(0, fmt.Sprintf("revalidating %s (%s)", m.ID, reason))
	attempt := ownerJobKeyPrefix + ":" + strings.TrimSpace(jobID) + ":" + uuid.NewString()
	_, err := s.deps.Installer.InstallForJob(ctx, m.ID, attempt, func(progress int, message string) {
		emit(progress, fmt.Sprintf("revalidating %s: %s", m.ID, message))
	})
	if errors.Is(err, sharedcapacity.ErrDiskAdmissionDenied) {
		return fmt.Errorf("%w: %w: waiting to revalidate %s: %v", ErrAcquireDiskBusy, jobs.ErrCapacityBusy, m.ID, err)
	}
	if err != nil {
		return fmt.Errorf("modelstorage: revalidation of %q failed: %w", m.ID, err)
	}
	s.deps.Logger.Printf("modelstorage: revalidated %s (%s) for job %s in %s", m.ID, reason, jobID, s.deps.Now().Sub(started).Round(time.Second))
	return nil
}

func (s *Service) lockAcquire(ctx context.Context, id string) (func(), error) {
	for {
		s.acquireMu.Lock()
		wait, busy := s.acquiring[id]
		if !busy {
			done := make(chan struct{})
			s.acquiring[id] = done
			s.acquireMu.Unlock()
			return func() {
				s.acquireMu.Lock()
				delete(s.acquiring, id)
				s.acquireMu.Unlock()
				close(done)
			}, nil
		}
		s.acquireMu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-wait:
		}
	}
}

func humanBytes(n int64) string {
	const gib = 1 << 30
	if n >= gib {
		return fmt.Sprintf("%.1f GiB", float64(n)/gib)
	}
	return fmt.Sprintf("%.0f MiB", float64(n)/(1<<20))
}
