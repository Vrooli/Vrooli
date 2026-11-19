package destinations

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/sysmounts"

	"github.com/vrooli/vrooli/packages/driveprofile"
)

// defaultListLimit caps List when the caller passes 0.
const defaultListLimit = 100

// nearCapThreshold is the fraction of cap_bytes at which usage transitions from
// WITHIN to NEAR (i.e. >= 90 % of cap).
const nearCapThreshold = 0.90

// Service is the application surface the destinations handlers depend on. It
// owns validation, the separate-root rule, encryption-on proof, and the
// cap-decision method the runs domain calls.
type Service interface {
	// CreateDestination validates and creates a destination, provisioning the
	// underlying kopia repository. Returns ErrInvalidDestination on validation
	// failure (including separate-root violation).
	CreateDestination(ctx context.Context, in CreateInput) (Destination, error)
	// ImportExistingDestination reconnects an exact existing repository from a
	// non-secret descriptor into an empty catalog. It must never call RepoCreate.
	ImportExistingDestination(ctx context.Context, in ImportExistingInput) (Destination, error)

	// GetDestination returns a destination by id. ErrDestinationNotFound
	// propagates verbatim.
	GetDestination(ctx context.Context, id string) (Destination, error)
	// ResolveForUse re-observes a bound filesystem destination and converges
	// mountpoint-dependent paths before operational consumers use it.
	ResolveForUse(ctx context.Context, id string) (Destination, error)
	// WithResolvedDestination holds this API process's per-destination
	// serialization lock through one immediate repository operation after
	// re-resolving its path. It requires write-ready filesystem evidence.
	WithResolvedDestination(ctx context.Context, id string, use func(Destination) error) error
	// WithResolvedDestinationForRead re-resolves a destination for a read-only
	// repository operation. It proves device identity, mount, bundle identity and
	// repository readability, but does not require a clean/writable filesystem.
	// Reading snapshots or usage from a dirty volume is safe; writing is not.
	WithResolvedDestinationForRead(ctx context.Context, id string, use func(Destination) error) error
	// WithResolvedDestinationForWrite is the write gate. It runs the bounded
	// write/read/fsync probe, so an unobservable filesystem state is admitted
	// with a warning on physical evidence rather than being blocked for lack of
	// a cleanliness signal. Redundancy is a separate question: how many devices
	// hold a copy is reported by coverage and never gates a write.
	WithResolvedDestinationForWrite(ctx context.Context, id string, use func(Destination) error) error

	// ListDestinations returns up to pageSize destinations ordered by name.
	// pageSize <= 0 uses the default.
	ListDestinations(ctx context.Context, pageSize int) ([]Destination, error)
	// ListTargetSnapshots returns validated immutable snapshots for a DBM
	// target identity, newest first. It never guesses a source from a path.
	ListTargetSnapshots(ctx context.Context, destinationID, owner, name string) ([]engine.Snapshot, error)

	// UpdateDestination updates mutable destination fields. A recovery identity
	// is accepted only after the readiness owner re-observes the stable UUID or
	// serial and proves it matches the requested mount root.
	UpdateDestination(ctx context.Context, in UpdateInput) (Destination, error)
	BindRecoveryIdentity(ctx context.Context, id string, identity destinationreadiness.DeviceIdentity) (Destination, error)

	// DeleteDestination removes a destination's catalog row. When
	// deleteRepository is true, it first removes local resource-kopia metadata
	// and credential-authority refs for the underlying repository.
	DeleteDestination(ctx context.Context, id string, deleteRepository bool) (bool, error)

	// GetDestinationUsage returns current usage bytes, cap, state, and policy
	// for the given destination.
	GetDestinationUsage(ctx context.Context, id string) (UsageReport, error)

	// WouldBlock reports whether a write of pendingBytes to the given destination
	// would be blocked by the cap policy. With CAP_POLICY_ALERT_BLOCK and
	// cap_bytes > 0, returns blocked=true when current usage + pendingBytes
	// exceeds cap_bytes. CAP_POLICY_ALERT_ONLY never blocks; cap_bytes == 0
	// (no cap) never blocks. WouldBlock never deletes anything.
	WouldBlock(ctx context.Context, destinationID string, pendingBytes int64) (blocked bool, reason string, err error)

	// ReconcileCredentialReferences upgrades legacy destination references and
	// generated filesystem recovery metadata to the canonical authority
	// identity. It is safe to run on every boot; a read-only destination remains
	// pending for a later retry.
	ReconcileCredentialReferences(ctx context.Context) error
}

// UsageReport is the result type returned by GetDestinationUsage.
type UsageReport struct {
	UsageBytes int64
	CapBytes   int64
	UsageState UsageState
	CapPolicy  CapPolicy
}

// ReadinessGate is the narrow readiness surface CreateDestination consults
// before provisioning a filesystem repository. It is an interface so the
// destinations domain does not depend on the readiness implementation, matching
// how preflight declares its own view of the same service.
type ReadinessGate interface {
	Analyze(ctx context.Context, in destinationreadiness.AnalyzeInput) (destinationreadiness.Report, error)
	// WriteProbe runs the bounded write/read/fsync probe. It is only called when
	// the caller has opted into a degraded write.
	WriteProbe(ctx context.Context, location string) (destinationreadiness.WriteProbeResult, error)
}

type recoveryIdentityInspector interface {
	InspectDevice(ctx context.Context, expected destinationreadiness.DeviceIdentity) (destinationreadiness.Inspection, error)
}

type service struct {
	repo          Repository
	eng           engine.KopiaEngine
	bundle        BundleWriter
	protectedRoot string
	readiness     ReadinessGate
	volumeLease   VolumeOperationLease
}

var destinationLockTable = struct {
	sync.Mutex
	locks map[string]*destinationLock
}{locks: map[string]*destinationLock{}}

type destinationLock struct {
	permit chan struct{}
	refs   int
}

func lockDestination(ctx context.Context, id string) (func(), error) {
	destinationLockTable.Lock()
	lock := destinationLockTable.locks[id]
	if lock == nil {
		lock = &destinationLock{permit: make(chan struct{}, 1)}
		lock.permit <- struct{}{}
		destinationLockTable.locks[id] = lock
	}
	lock.refs++
	destinationLockTable.Unlock()
	select {
	case <-ctx.Done():
		destinationLockTable.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(destinationLockTable.locks, id)
		}
		destinationLockTable.Unlock()
		return nil, ctx.Err()
	case <-lock.permit:
	}
	if err := ctx.Err(); err != nil {
		lock.permit <- struct{}{}
		destinationLockTable.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(destinationLockTable.locks, id)
		}
		destinationLockTable.Unlock()
		return nil, err
	}
	return func() {
		lock.permit <- struct{}{}
		destinationLockTable.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(destinationLockTable.locks, id)
		}
		destinationLockTable.Unlock()
	}, nil
}

// Option customises the Service beyond its required collaborators.
type Option func(*service)

// WithReadinessGate makes CreateDestination refuse filesystem destinations that
// fail readiness. Without it the service keeps its previous behaviour, so an
// S3-only deployment and existing tests need no readiness implementation.
func WithReadinessGate(gate ReadinessGate) Option {
	return func(s *service) { s.readiness = gate }
}

// WithVolumeOperationLease installs the control-plane lease used to serialize
// DBM repository operations with supported host volume changes. Bound
// filesystem destinations fail closed when no lease provider is configured.
func WithVolumeOperationLease(lease VolumeOperationLease) Option {
	return func(s *service) { s.volumeLease = lease }
}

// NewService constructs the production Service. bundle materializes the
// self-describing filesystem destination bundle (README/RECOVERY/manifest +
// repositories/<slug>.kopia); pass nil for an S3-only deployment or a fake in
// tests.
func NewService(repo Repository, eng engine.KopiaEngine, bundle BundleWriter, protectedRoot string, opts ...Option) Service {
	s := &service{
		repo:          repo,
		eng:           eng,
		bundle:        bundle,
		protectedRoot: filepath.Clean(protectedRoot),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

// ResolveForUse resolves a bound filesystem bundle from its stable device
// identity and volume-relative locator. It fails closed on uncertain volume
// state, missing/colliding paths, or mismatched bundle ownership. Resource-
// kopia and generated metadata are repaired idempotently without creating a
// repository or changing credential identity.
func (s *service) ResolveForUse(ctx context.Context, id string) (Destination, error) {
	release, err := lockDestination(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	defer release()
	var resolved Destination
	err = s.withVolumeOperationLease(ctx, id, func() error {
		var resolveErr error
		resolved, resolveErr = s.resolveForUse(ctx, id)
		return resolveErr
	})
	return resolved, err
}

func (s *service) WithResolvedDestination(ctx context.Context, id string, use func(Destination) error) error {
	return s.WithResolvedDestinationForWrite(ctx, id, use)
}

func (s *service) WithResolvedDestinationForWrite(ctx context.Context, id string, use func(Destination) error) error {
	if use == nil {
		return ErrInvalidDestination{Field: "operation", Reason: "required"}
	}
	release, err := lockDestination(ctx, id)
	if err != nil {
		return err
	}
	defer release()
	return s.withVolumeOperationLease(ctx, id, func() error {
		d, resolveErr := s.resolveForUse(ctx, id)
		if resolveErr != nil {
			return resolveErr
		}
		return use(d)
	})
}

func (s *service) WithResolvedDestinationForRead(ctx context.Context, id string, use func(Destination) error) error {
	if use == nil {
		return ErrInvalidDestination{Field: "operation", Reason: "required"}
	}
	// Reads do not take the exclusive write lock or the host volume operation
	// lease. Those exist to serialize writers and to keep an unmount/repair from
	// racing a write; holding them for a snapshot or usage read lets one slow
	// writer or recovery probe block every read on the drive. A concurrent
	// rebind can make a read fail; it is retried after the next resolution.
	d, err := s.resolveForRead(ctx, id)
	if err != nil {
		return err
	}
	return use(d)
}

func (s *service) withVolumeOperationLease(ctx context.Context, id string, use func() error) error {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if d.BackendKind != BackendFilesystem || d.DeviceIdentity == nil || !d.DeviceIdentity.StrongIdentity() || strings.TrimSpace(d.RelativePath) == "" {
		return use() // Explicit legacy filesystem paths and S3 keep their established behavior.
	}
	if s.volumeLease == nil {
		return ErrInvalidDestination{Field: "identity", Reason: "host volume operation lease is unavailable"}
	}
	inspector, ok := s.readiness.(recoveryIdentityInspector)
	if !ok {
		return ErrInvalidDestination{Field: "identity", Reason: "stable device inspection is unavailable"}
	}
	observed, err := inspector.InspectDevice(ctx, *d.DeviceIdentity)
	if err != nil {
		return fmt.Errorf("observe destination before acquiring host volume lease: %w", err)
	}
	if !d.DeviceIdentity.MatchesDevice(observed.Identity) || !observed.Identity.StrongIdentity() || !observed.Mounted {
		return ErrInvalidDestination{Field: "identity", Reason: "destination volume identity or mount is not proven before acquiring the host volume lease"}
	}
	release, err := s.volumeLease.Acquire(ctx, observed.Identity)
	if err != nil {
		return fmt.Errorf("acquire host volume operation lease: %w", err)
	}
	defer release()
	return use()
}

// resolveForRead proves everything a read needs — stable identity, mount,
// bundle ownership, contained repository path, and a working repository
// connection — without demanding a clean or writable filesystem. A dirty NTFS
// volume cannot safely accept writes, but its snapshots are still readable.
func (s *service) resolveForRead(ctx context.Context, id string) (Destination, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	if d.BackendKind != BackendFilesystem || d.DeviceIdentity == nil || !d.DeviceIdentity.StrongIdentity() || strings.TrimSpace(d.RelativePath) == "" {
		return d, nil // Explicit legacy filesystem paths and S3 keep their established behavior.
	}
	if s.readiness == nil || s.bundle == nil {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "filesystem destination recovery is unavailable"}
	}
	inspector, ok := s.readiness.(recoveryIdentityInspector)
	if !ok {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "stable device inspection is unavailable"}
	}
	observed, err := inspector.InspectDevice(ctx, *d.DeviceIdentity)
	if err != nil {
		return Destination{}, fmt.Errorf("re-observe destination volume: %w", err)
	}
	if !d.DeviceIdentity.MatchesDevice(observed.Identity) || !observed.Identity.StrongIdentity() || !observed.Mounted {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "destination volume identity or mount is not proven for read access"}
	}
	mountRoot := filepath.Clean(observed.Identity.Mountpoint)
	if !filepath.IsAbs(mountRoot) || mountRoot == string(filepath.Separator) {
		return Destination{}, ErrInvalidDestination{Field: "mountpoint", Reason: "verified volume has no safe non-root mountpoint"}
	}
	bundleRoot, err := destinationreadiness.ResolveRelativePath(mountRoot, d.RelativePath)
	if err != nil {
		return Destination{}, ErrInvalidDestination{Field: "relative_path", Reason: err.Error()}
	}
	_, resolvedBundle, err := resolveContainedPath(mountRoot, bundleRoot)
	if err != nil {
		return Destination{}, ErrInvalidDestination{Field: "relative_path", Reason: "destination bundle resolves outside its mounted volume: " + err.Error()}
	}
	repoPath := RepositoryPathFor(bundleRoot, d.Name)
	_, resolvedRepo, err := resolveContainedPath(bundleRoot, repoPath)
	if err != nil {
		return Destination{}, ErrInvalidDestination{Field: "repository_path", Reason: "repository resolves outside its destination bundle: " + err.Error()}
	}
	meta := BundleMetadata{
		DestinationID: d.ID, Name: d.Name, Backend: string(d.BackendKind),
		BundleRoot: resolvedBundle, RelativePath: d.RelativePath, RepositoryPath: resolvedRepo,
		EncryptionAlgorithm: d.EncryptionAlgorithm, SecretRef: d.SecretRef, CreatedAt: d.CreatedAt,
	}
	if err := s.bundle.ValidateRelocation(ctx, meta); err != nil {
		return Destination{}, fmt.Errorf("validate read-only destination bundle: %w", err)
	}
	if _, err := repositoryUniqueID(resolvedRepo); err != nil {
		return Destination{}, fmt.Errorf("existing repository marker is unavailable: %w", err)
	}
	if err := s.eng.RepoRebind(ctx, d.Name, resolvedRepo); err != nil {
		return Destination{}, fmt.Errorf("rebind resource-kopia repository path for read: %w", err)
	}
	status, err := s.eng.RepoStatus(ctx, d.Name)
	if err != nil {
		return Destination{}, fmt.Errorf("verify read-only repository connection: %w", err)
	}
	if !status.Connected || strings.TrimSpace(status.EncryptionAlgorithm) == "" {
		return Destination{}, ErrInvalidDestination{Field: "repository", Reason: "read-only repository connection or encryption was not verified"}
	}
	d.Location, d.RepositoryLocation = resolvedBundle, resolvedRepo
	d.DeviceIdentity = &observed.Identity
	d.DeviceIdentityObservedAt = time.Now().UTC()
	return d, nil
}

func (s *service) resolveForUse(ctx context.Context, id string) (Destination, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	if d.ProvisioningState != "" && d.ProvisioningState != ProvisioningReady {
		return Destination{}, ErrInvalidDestination{Field: "provisioning", Reason: "destination creation is incomplete; retry the original create request to reconcile it before use"}
	}
	if d.BackendKind != BackendFilesystem || d.DeviceIdentity == nil || !d.DeviceIdentity.StrongIdentity() || strings.TrimSpace(d.RelativePath) == "" {
		return d, nil // Explicit legacy filesystem paths and S3 keep their established behavior.
	}
	if s.readiness == nil || s.bundle == nil {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "filesystem destination recovery is unavailable"}
	}
	inspector, ok := s.readiness.(recoveryIdentityInspector)
	if !ok {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "stable device inspection is unavailable"}
	}
	observed, err := inspector.InspectDevice(ctx, *d.DeviceIdentity)
	if err != nil {
		return Destination{}, fmt.Errorf("re-observe destination volume: %w", err)
	}
	// One write verdict for every writer. The write probe is the physical proof:
	// it is run whenever the mount is plausibly writable, so an unobservable
	// state is decided on evidence instead of being blocked by default.
	probe := destinationreadiness.WriteProbeResult{}
	if observed.LocationExists && observed.LocationIsDirectory && observed.Mounted && !observed.ReadOnly && !observed.DeviceWriteProtected {
		observedProbe, probeErr := s.readiness.WriteProbe(ctx, d.Location)
		if probeErr != nil {
			observedProbe = destinationreadiness.WriteProbeResult{Ran: true, Error: probeErr.Error()}
		}
		probe = observedProbe
	}
	verdict, verdictReason := destinationreadiness.EvaluateWriteVerdict(destinationreadiness.WriteVerdictInput{
		Mounted:              observed.Mounted,
		ReadOnly:             observed.ReadOnly,
		DeviceWriteProtected: observed.DeviceWriteProtected,
		FilesystemState:      observed.FilesystemState,
		Probe:                probe,
	})
	filesystemReady := verdict == destinationreadiness.WriteReady || verdict == destinationreadiness.WriteDegraded
	if !d.DeviceIdentity.MatchesDevice(observed.Identity) || !observed.Identity.StrongIdentity() || !filesystemReady {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "destination volume identity, mount, writable state, or filesystem health is not proven ready: " + verdictReason}
	}
	mountRoot := filepath.Clean(observed.Identity.Mountpoint)
	if !filepath.IsAbs(mountRoot) || mountRoot == string(filepath.Separator) {
		return Destination{}, ErrInvalidDestination{Field: "mountpoint", Reason: "verified volume has no safe non-root mountpoint"}
	}
	bundleRoot, err := destinationreadiness.ResolveRelativePath(mountRoot, d.RelativePath)
	if err != nil {
		return Destination{}, ErrInvalidDestination{Field: "relative_path", Reason: err.Error()}
	}
	_, resolvedBundle, err := resolveContainedPath(mountRoot, bundleRoot)
	if err != nil {
		return Destination{}, ErrInvalidDestination{Field: "relative_path", Reason: "destination bundle resolves outside its mounted volume: " + err.Error()}
	}
	readiness, err := s.readiness.Analyze(ctx, destinationreadiness.AnalyzeInput{
		DestinationID: d.ID, ExpectedDevice: *d.DeviceIdentity, Location: bundleRoot,
		AllowWriteProbe: true,
	})
	if err != nil {
		return Destination{}, fmt.Errorf("analyze relocated destination readiness: %w", err)
	}
	if !readiness.Mounted || !sameStrongDeviceIdentity(*d.DeviceIdentity, readiness.Identity) {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "destination path is not served by the verified mounted volume"}
	}
	for _, check := range readiness.Checks {
		switch check.Code {
		case "destination_present", "destination_missing", "destination_inaccessible", "directory_inaccessible", "mounted_read_write", "filesystem_state", "destination_dirty", "filesystem_suitability", "write_probe":
			// filesystem_suitability and a probe-backed degraded filesystem_state
			// are advisory warnings, not blocks. Corruption evidence (dirty or
			// needs-check) is still a Fail and still blocks.
			if check.Code == "filesystem_suitability" && check.Severity == destinationreadiness.SeverityWarning {
				continue
			}
			if check.Code == "filesystem_state" && check.Severity == destinationreadiness.SeverityWarning {
				continue
			}
			if check.Severity != destinationreadiness.SeverityPass {
				return Destination{}, ErrInvalidDestination{Field: "readiness", Reason: check.Message}
			}
		}
	}
	repoPath := RepositoryPathFor(bundleRoot, d.Name)
	_, resolvedRepo, err := resolveContainedPath(bundleRoot, repoPath)
	if err != nil {
		return Destination{}, ErrInvalidDestination{Field: "repository_path", Reason: "repository resolves outside its destination bundle: " + err.Error()}
	}
	if _, err := repositoryUniqueID(resolvedRepo); err != nil {
		return Destination{}, fmt.Errorf("existing repository marker is unavailable: %w", err)
	}
	dests, err := s.repo.List(ctx, defaultListLimit)
	if err != nil {
		return Destination{}, fmt.Errorf("check destination locator collisions: %w", err)
	}
	for _, other := range dests {
		if other.ID == d.ID || other.BackendKind != BackendFilesystem {
			continue
		}
		// Two destinations may not share one bundle root (one manifest cannot
		// describe both), and their repository paths may not overlap. A bundle
		// nested *inside* another destination's bundle root is legitimate — for
		// example a per-scope repository under a drive-wide bundle — so bundle
		// roots are compared for equality, not containment.
		otherBundle := other.Location
		if other.DeviceIdentity != nil && other.DeviceIdentity.MatchesDevice(observed.Identity) && strings.TrimSpace(other.RelativePath) != "" {
			if otherRoot, resolveErr := destinationreadiness.ResolveRelativePath(mountRoot, other.RelativePath); resolveErr == nil {
				otherBundle = otherRoot
			}
		}
		if otherBundle != "" && sameResolvedPath(resolvedBundle, otherBundle) {
			return Destination{}, ErrInvalidDestination{Field: "relative_path", Reason: "another destination already claims this bundle path"}
		}
		candidates := []string{other.RepositoryLocation}
		if strings.TrimSpace(other.RelativePath) != "" && otherBundle != "" {
			candidates = append(candidates, RepositoryPathFor(otherBundle, other.Name))
		}
		for _, candidate := range candidates {
			if strings.TrimSpace(candidate) == "" {
				continue
			}
			if resolvedOther, evalErr := filepath.EvalSymlinks(candidate); evalErr == nil && diskPathsOverlap(resolvedRepo, resolvedOther) {
				return Destination{}, ErrInvalidDestination{Field: "repository_path", Reason: "another destination already claims the resolved repository path"}
			}
		}
	}
	meta := BundleMetadata{DestinationID: d.ID, Name: d.Name, Backend: string(d.BackendKind), BundleRoot: bundleRoot, RelativePath: d.RelativePath, RepositoryPath: repoPath, EncryptionAlgorithm: d.EncryptionAlgorithm, SecretRef: d.SecretRef, CreatedAt: d.CreatedAt}
	if err := s.bundle.ValidateRelocation(ctx, meta); err != nil {
		return Destination{}, fmt.Errorf("validate relocated destination bundle: %w", err)
	}
	if err := s.eng.RepoRebind(ctx, d.Name, repoPath); err != nil {
		return Destination{}, fmt.Errorf("rebind resource-kopia repository path: %w", err)
	}
	status, err := s.eng.RepoStatus(ctx, d.Name)
	if err != nil {
		return Destination{}, fmt.Errorf("verify rebound repository connection: %w", err)
	}
	if !status.Connected || status.EncryptionAlgorithm == "" {
		return Destination{}, ErrInvalidDestination{Field: "repository", Reason: "rebound repository connection or encryption was not verified"}
	}
	if _, err := s.eng.SnapshotList(ctx, d.Name, ""); err != nil {
		return Destination{}, fmt.Errorf("enumerate snapshots in rebound repository: %w", err)
	}
	if err := s.bundle.RebindMetadata(ctx, meta); err != nil {
		return Destination{}, fmt.Errorf("repair destination bundle path metadata: %w", err)
	}
	d.Location, d.RepositoryLocation = bundleRoot, repoPath
	d.DeviceIdentity = &observed.Identity
	d.DeviceIdentityObservedAt = time.Now().UTC()
	return s.repo.Update(ctx, d)
}

func diskPathsOverlap(a, b string) bool {
	return pathContains(a, b) || pathContains(b, a)
}

// sameResolvedPath reports whether two paths name the same directory. It
// resolves symlinks when possible and compares case-insensitively so a
// case-insensitive volume (NTFS, APFS default) cannot hide a duplicate bundle
// behind different capitalization.
func sameResolvedPath(a, b string) bool {
	resolve := func(p string) string {
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Clean(resolved)
		}
		return filepath.Clean(p)
	}
	return strings.EqualFold(resolve(a), resolve(b))
}

func pathContains(root, child string) bool {
	root, child = filepath.Clean(root), filepath.Clean(child)
	rel, err := filepath.Rel(root, child)
	return err == nil && (rel == "." || (rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))))
}

func resolveContainedPath(rootPath, childPath string) (string, string, error) {
	resolvedRoot, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve containment root: %w", err)
	}
	resolvedChild, err := filepath.EvalSymlinks(childPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve contained path: %w", err)
	}
	if resolvedChild == resolvedRoot || !pathContains(resolvedRoot, resolvedChild) {
		return "", "", fmt.Errorf("resolved path escapes its owner directory")
	}
	return resolvedRoot, resolvedChild, nil
}

func sameStrongDeviceIdentity(expected, observed destinationreadiness.DeviceIdentity) bool {
	if !expected.StrongIdentity() || !observed.StrongIdentity() || !expected.MatchesDevice(observed) {
		return false
	}
	if expected.UUID != "" && !strings.EqualFold(strings.TrimSpace(expected.UUID), strings.TrimSpace(observed.UUID)) {
		return false
	}
	if expected.Serial != "" && strings.TrimSpace(expected.Serial) != strings.TrimSpace(observed.Serial) {
		return false
	}
	return true
}

// gateOnReadiness refuses a filesystem destination whose readiness report
// fails. It draws one line the operator cannot cross: a driver that faults in
// kernel context is refused even with acknowledgement, because that risk is to
// the host rather than to the backup, and the remedy — remounting through a
// userspace driver — is cheap. Every other failure is an operator judgement
// call and can be acknowledged.
//
// A service with no readiness gate configured keeps the previous behaviour.
func (s *service) gateOnReadiness(ctx context.Context, location string, acknowledged, allowUnrepairable bool) (destinationreadiness.Report, error) {
	if s.readiness == nil {
		return destinationreadiness.Report{}, nil
	}
	report, err := s.readiness.Analyze(ctx, destinationreadiness.AnalyzeInput{Location: location})
	if err != nil {
		// Readiness that cannot run is not readiness that passed. Refusing here
		// keeps an unprovable destination out of the catalog rather than
		// admitting it on the strength of a broken probe.
		return destinationreadiness.Report{}, ErrInvalidDestination{Field: "location", Reason: "destination readiness could not be evaluated: " + err.Error()}
	}

	if reason, blocked := unrepairableFilesystem(report.Identity.Filesystem, allowUnrepairable); blocked {
		return destinationreadiness.Report{}, ErrInvalidDestination{Field: "filesystem", Reason: reason}
	}

	for _, check := range report.Checks {
		if check.Code != "filesystem_suitability" || check.Severity != destinationreadiness.SeverityFail {
			continue
		}
		return destinationreadiness.Report{}, ErrInvalidDestination{Field: "location", Reason: check.Message + "; " + check.NextAction}
	}
	if report.OverallSeverity == destinationreadiness.SeverityUnknown || report.OverallSeverity == "" {
		// Capacity is intentionally unknown until a plan selects its targets.
		// A profiled filesystem whose OS/driver does not expose a cleanliness
		// signal is also acceptable at creation time: the readiness check still
		// reports it truthfully, and the write gate still requires a passing
		// probe (plus a second device) before a snapshot is admitted. Refusing
		// creation would make a perfectly usable exFAT or NTFS drive impossible
		// to add at all.
		unknownReadiness := false
		unknownChecks := 0
		for _, check := range report.Checks {
			if check.Severity != destinationreadiness.SeverityUnknown {
				continue
			}
			unknownChecks++
			switch check.Code {
			case "capacity":
				// Unknown until a plan selects targets.
			case "filesystem_state":
				if _, profiled := driveprofile.Lookup(report.Identity.Filesystem); profiled {
					continue
				}
				unknownReadiness = true
			default:
				unknownReadiness = true
			}
		}
		if unknownReadiness || unknownChecks == 0 {
			return destinationreadiness.Report{}, ErrInvalidDestination{Field: "location", Reason: "destination readiness is unknown; establish the current mounted-volume identity and filesystem health before creating a repository"}
		}
	}

	if report.OverallSeverity != destinationreadiness.SeverityFail || acknowledged {
		return report, nil
	}
	return destinationreadiness.Report{}, ErrInvalidDestination{Field: "location", Reason: readinessRefusalReason(report)}
}

func (s *service) recheckCreateReadiness(ctx context.Context, location string, expected destinationreadiness.DeviceIdentity, acknowledged, allowUnrepairable bool) error {
	report, err := s.gateOnReadiness(ctx, location, acknowledged, allowUnrepairable)
	if err != nil {
		return err
	}
	if expected.StrongIdentity() && !sameStrongDeviceIdentity(expected, report.Identity) {
		return ErrInvalidDestination{Field: "identity", Reason: "filesystem destination identity changed before repository creation"}
	}
	return nil
}

// readinessRefusalReason names the first failing check so the operator is told
// which condition to fix, not merely that something failed.
func readinessRefusalReason(report destinationreadiness.Report) string {
	for _, check := range report.Checks {
		if check.Severity != destinationreadiness.SeverityFail {
			continue
		}
		reason := "destination is not ready: " + check.Message
		if strings.TrimSpace(check.NextAction) != "" {
			reason += "; " + check.NextAction
		}
		return reason
	}
	return "destination is not ready; inspect destination readiness before creating it"
}

func (s *service) CreateDestination(ctx context.Context, in CreateInput) (Destination, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Destination{}, ErrInvalidDestination{Field: "name", Reason: "required"}
	}
	// A destination name doubles as its kopia repository name, so it must be a
	// stable slug — never an arbitrary label with spaces or slashes.
	if !ValidSlug(name) {
		return Destination{}, ErrInvalidDestination{
			Field:  "name",
			Reason: "must be slug-safe: lowercase letters, digits and hyphens only (e.g. elements-local)",
		}
	}
	if !in.Backend.Valid() {
		return Destination{}, ErrInvalidDestination{Field: "backend_kind", Reason: "must be filesystem or s3"}
	}
	location := strings.TrimSpace(in.Location)
	if location == "" {
		return Destination{}, ErrInvalidDestination{Field: "location", Reason: "required"}
	}

	capPolicy := in.CapPolicy
	if capPolicy == "" {
		capPolicy = CapPolicyAlertBlock
	}

	// A persisted creating row is the durable intent for a RepoCreate call.
	// Reconcile it by connecting to the exact repository path; never replay
	// RepoCreate after an outcome may have become ambiguous.
	existing, lookupErr := s.repo.GetByName(ctx, name)
	if lookupErr == nil {
		if existing.ProvisioningState == ProvisioningCreating {
			return Destination{}, ErrInvalidDestination{Field: "provisioning", Reason: "repository create outcome is unknown; preserve this attempt for explicit owner reconciliation"}
		}
		if existing.ProvisioningState != ProvisioningCreated {
			return Destination{}, ErrInvalidDestination{Field: "name", Reason: "destination already exists"}
		}
		return s.reconcilePendingCreate(ctx, existing, in, location, capPolicy)
	}
	var notFound ErrDestinationNotFound
	if !errors.As(lookupErr, &notFound) {
		return Destination{}, lookupErr
	}

	// repositoryLocation is the concrete backend path resource-kopia targets.
	// For filesystem it nests under the operator-facing bundle root; for S3 it
	// is the bucket/prefix itself.
	var repositoryLocation string
	var readinessReport destinationreadiness.Report

	// Build the kopia RepoSpec and create the repository.
	spec := engine.RepoSpec{
		Name:    name,
		Backend: string(in.Backend),
	}
	switch in.Backend {
	case BackendFilesystem:
		// Separate-root rule: a filesystem destination must not point under the
		// protectedRoot (the storage root the manager protects).
		clean := filepath.Clean(location)
		if clean == s.protectedRoot || strings.HasPrefix(clean, s.protectedRoot+string(filepath.Separator)) {
			return Destination{}, ErrInvalidDestination{
				Field:  "location",
				Reason: "filesystem destination must not point inside the protected root",
			}
		}
		// Readiness runs before anything is written to the drive: a refusal must
		// not leave a half-provisioned bundle behind on a destination we just
		// declared unfit.
		var err error
		readinessReport, err = s.gateOnReadiness(ctx, location, in.AcknowledgeReadinessFailure, in.AcknowledgeUnrepairableFilesystem)
		if err != nil {
			return Destination{}, err
		}
		if err := s.recheckCreateReadiness(ctx, location, readinessReport.Identity, in.AcknowledgeReadinessFailure, in.AcknowledgeUnrepairableFilesystem); err != nil {
			return Destination{}, err
		}
		if readinessReport.Identity.StrongIdentity() {
			if s.volumeLease == nil {
				return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "host volume operation lease is unavailable"}
			}
			release, leaseErr := s.volumeLease.Acquire(ctx, readinessReport.Identity)
			if leaseErr != nil {
				return Destination{}, fmt.Errorf("acquire host volume operation lease before provisioning: %w", leaseErr)
			}
			defer release()
		}
		repositoryLocation = RepositoryPathFor(location, name)
		// Materialize the bundle root + repository directory before creating the
		// repository so kopia writes into repositories/<slug>.kopia, not the bare
		// operator-facing root.
		if s.bundle != nil {
			if err := s.bundle.PrepareRepository(ctx, location, repositoryLocation); err != nil {
				return Destination{}, err
			}
		}
		spec.Path = repositoryLocation
	case BackendS3:
		s3, parseErr := parseS3Location(location)
		if parseErr != nil {
			return Destination{}, ErrInvalidDestination{Field: "location", Reason: parseErr.Error()}
		}
		repositoryLocation = location
		spec.Bucket = s3.Bucket
		spec.Endpoint = s3.Endpoint
		spec.Prefix = s3.Prefix
		spec.Region = s3.Region
		spec.DisableTLS = s3.DisableTLS
	case BackendSFTP:
		sftp, parseErr := parseSFTPLocation(location)
		if parseErr != nil {
			return Destination{}, ErrInvalidDestination{Field: "location", Reason: parseErr.Error()}
		}
		repositoryLocation = location
		spec.SFTPHost = sftp.Host
		spec.SFTPPort = sftp.Port
		spec.SFTPUsername = sftp.Username
		spec.SFTPRemotePath = sftp.RemotePath
		spec.SFTPKnownHosts = sftp.KnownHosts
	}
	if in.Backend == BackendFilesystem {
		if err := s.recheckCreateReadiness(ctx, location, readinessReport.Identity, in.AcknowledgeReadinessFailure, in.AcknowledgeUnrepairableFilesystem); err != nil {
			return Destination{}, err
		}
	}

	d := Destination{
		Name:               name,
		BackendKind:        in.Backend,
		Location:           location,
		RepositoryLocation: repositoryLocation,
		CapBytes:           in.CapBytes,
		CapPolicy:          capPolicy,
		ProvisioningState:  ProvisioningCreating,
		// resource-kopia owns the secret value; DBM stores only its deterministic
		// authority reference so the detached bundle can identify the key owner.
		SecretRef: s.eng.PassphraseRef(name),
	}
	if in.Backend == BackendFilesystem && readinessReport.Identity.DevicePath != "" {
		identity := readinessReport.Identity
		d.DeviceIdentity = &identity
		d.DeviceIdentityObservedAt = readinessReport.ObservedAt
		// Persist the volume-relative locator whenever readiness observed a
		// mountpoint. The absolute location remains a useful display value, but
		// recovery must be able to resolve the destination after a replug changes
		// the mountpoint or drive letter.
		if relative, relativeErr := destinationreadiness.RelativePathUnderMount(location, identity.Mountpoint); relativeErr == nil {
			d.RelativePath = relative
		}
	}

	saved, err := s.repo.Create(ctx, d)
	if err != nil {
		return Destination{}, err
	}
	if err := s.eng.RepoCreate(ctx, spec); err != nil {
		return Destination{}, fmt.Errorf("destination %q has an unknown repository create outcome; preserve the pending intent for explicit owner reconciliation without retrying create or connect: %w", name, err)
	}
	saved.ProvisioningState = ProvisioningCreated
	saved, err = s.repo.Update(ctx, saved)
	if err != nil {
		return Destination{}, fmt.Errorf("repository %q was created but its confirmed outcome could not be persisted; preserve the pending intent for owner reconciliation: %w", name, err)
	}

	return s.finishPendingCreate(ctx, saved, in.AcknowledgeReadinessFailure, in.AcknowledgeUnrepairableFilesystem)
}

// ImportExistingDestination reconnects an exact existing repository into a
// clean catalog. Filesystem bundles additionally require their strong device
// identity and repository marker anchor. S3/SFTP descriptors must name the
// exact snapshots required by the rehearsal, which are checked after connect.
// Every backend uses RepoConnect/RepoRebind and never RepoCreate.
func (s *service) ImportExistingDestination(ctx context.Context, in ImportExistingInput) (Destination, error) {
	in.ID = strings.TrimSpace(in.ID)
	in.Name = strings.TrimSpace(in.Name)
	in.Location = strings.TrimSpace(in.Location)
	if in.ID == "" || !ValidSlug(in.Name) || in.Location == "" || !in.Backend.Valid() {
		return Destination{}, ErrInvalidDestination{Field: "descriptor", Reason: "destination id, slug-safe name, supported backend, and location are required"}
	}
	expectedSnapshots := make(map[string]struct{}, len(in.ExpectedSnapshotIDs))
	for _, id := range in.ExpectedSnapshotIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			return Destination{}, ErrInvalidDestination{Field: "expected_snapshot_ids", Reason: "snapshot IDs cannot be empty"}
		}
		if _, duplicate := expectedSnapshots[id]; duplicate {
			return Destination{}, ErrInvalidDestination{Field: "expected_snapshot_ids", Reason: "snapshot IDs must be unique"}
		}
		expectedSnapshots[id] = struct{}{}
	}
	if _, err := s.repo.GetByName(ctx, in.Name); err == nil {
		return Destination{}, ErrInvalidDestination{Field: "name", Reason: "a destination with this name already exists; import never replaces or overwrites a catalog entry"}
	} else {
		var notFound ErrDestinationNotFound
		if !errors.As(err, &notFound) {
			return Destination{}, err
		}
	}
	if _, err := s.repo.GetByID(ctx, in.ID); err == nil {
		return Destination{}, ErrInvalidDestination{Field: "id", Reason: "the destination identity is already used by another catalog entry"}
	} else {
		var notFound ErrDestinationNotFound
		if !errors.As(err, &notFound) {
			return Destination{}, err
		}
	}
	secretRef := s.eng.PassphraseRef(in.Name)
	if secretRef == "" {
		return Destination{}, ErrInvalidDestination{Field: "secret_ref", Reason: "repository credential identity could not be derived"}
	}
	repositoryLocation := in.Location
	var meta BundleMetadata
	var volumeRelativePath string
	var identity *destinationreadiness.DeviceIdentity
	var observedAt time.Time
	var releaseLease func()
	defer func() {
		if releaseLease != nil {
			releaseLease()
		}
	}()
	spec := engine.RepoSpec{Name: in.Name, Backend: string(in.Backend)}
	switch in.Backend {
	case BackendFilesystem:
		if s.bundle == nil || s.readiness == nil {
			return Destination{}, ErrInvalidDestination{Field: "bundle", Reason: "filesystem bundle and mounted-volume inspectors are required"}
		}
		location, err := filepath.Abs(filepath.Clean(in.Location))
		if err != nil {
			return Destination{}, fmt.Errorf("resolve filesystem bundle root: %w", err)
		}
		in.Location = location
		existing, err := s.bundle.ReadExisting(ctx, location)
		if err != nil {
			return Destination{}, ErrInvalidDestination{Field: "bundle", Reason: err.Error()}
		}
		meta = existing.Metadata
		if meta.DestinationID != in.ID || meta.Name != in.Name || meta.Backend != string(in.Backend) || meta.SecretRef != secretRef {
			return Destination{}, ErrInvalidDestination{Field: "descriptor", Reason: "filesystem bundle identity differs from the supplied non-secret descriptor"}
		}
		if !in.CreatedAt.IsZero() && !in.CreatedAt.Equal(meta.CreatedAt) {
			return Destination{}, ErrInvalidDestination{Field: "created_at", Reason: "descriptor creation timestamp differs from the filesystem bundle"}
		}
		if _, resolvedRepository, resolveErr := resolveContainedPath(location, meta.RepositoryPath); resolveErr != nil || filepath.Clean(resolvedRepository) != filepath.Clean(meta.RepositoryPath) {
			return Destination{}, ErrInvalidDestination{Field: "repository_path", Reason: "existing repository must be a real path contained directly under the verified bundle"}
		}
		readinessReport, err := s.gateOnReadiness(ctx, location, false, true)
		if err != nil {
			return Destination{}, err
		}
		if !readinessReport.Identity.StrongIdentity() {
			return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "a stable mounted-volume UUID or serial is required to import a filesystem bundle"}
		}
		mountRoot := filepath.Clean(readinessReport.Identity.Mountpoint)
		volumeRelativePath, err = filepath.Rel(mountRoot, location)
		if err != nil || volumeRelativePath == "." || volumeRelativePath == ".." || strings.HasPrefix(volumeRelativePath, ".."+string(filepath.Separator)) || filepath.ToSlash(volumeRelativePath) != filepath.ToSlash(meta.RelativePath) {
			return Destination{}, ErrInvalidDestination{Field: "relative_path", Reason: "bundle is not at its anchored location under the currently observed volume"}
		}
		if err := s.recheckCreateReadiness(ctx, location, readinessReport.Identity, false, true); err != nil {
			return Destination{}, err
		}
		if s.volumeLease == nil {
			return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "host volume operation lease is unavailable"}
		}
		releaseLease, err = s.volumeLease.Acquire(ctx, readinessReport.Identity)
		if err != nil {
			return Destination{}, fmt.Errorf("acquire host volume operation lease before repository import: %w", err)
		}
		if err := s.recheckCreateReadiness(ctx, location, readinessReport.Identity, false, true); err != nil {
			return Destination{}, err
		}
		if err := s.bundle.ValidateRelocation(ctx, meta); err != nil {
			return Destination{}, ErrInvalidDestination{Field: "bundle", Reason: "bundle/repository identity validation failed: " + err.Error()}
		}
		meta.BundleRoot = location
		meta.RepositoryPath = RepositoryPathFor(location, in.Name)
		if err := s.eng.RepoRebind(ctx, in.Name, meta.RepositoryPath); err != nil {
			return Destination{}, fmt.Errorf("connect existing filesystem repository %q: %w", in.Name, err)
		}
		identity = &readinessReport.Identity
		observedAt = time.Now().UTC()
		repositoryLocation = meta.RepositoryPath
	case BackendS3:
		s3, err := parseS3Location(in.Location)
		if err != nil {
			return Destination{}, ErrInvalidDestination{Field: "location", Reason: err.Error()}
		}
		spec.Bucket, spec.Endpoint, spec.Prefix, spec.Region, spec.DisableTLS = s3.Bucket, s3.Endpoint, s3.Prefix, s3.Region, s3.DisableTLS
	case BackendSFTP:
		sftp, err := parseSFTPLocation(in.Location)
		if err != nil {
			return Destination{}, ErrInvalidDestination{Field: "location", Reason: err.Error()}
		}
		spec.SFTPHost, spec.SFTPPort, spec.SFTPUsername, spec.SFTPRemotePath, spec.SFTPKnownHosts = sftp.Host, sftp.Port, sftp.Username, sftp.RemotePath, sftp.KnownHosts
	}
	if in.Backend != BackendFilesystem {
		if in.CreatedAt.IsZero() {
			return Destination{}, ErrInvalidDestination{Field: "created_at", Reason: "source catalog creation time is required for remote destination import"}
		}
		if err := s.eng.RepoConnect(ctx, spec); err != nil {
			return Destination{}, fmt.Errorf("connect existing repository %q: %w", in.Name, err)
		}
	}
	status, err := s.eng.RepoStatus(ctx, in.Name)
	if err != nil {
		return Destination{}, fmt.Errorf("verify connected repository %q: %w", in.Name, err)
	}
	if !status.Connected || strings.TrimSpace(status.EncryptionAlgorithm) == "" {
		return Destination{}, ErrInvalidDestination{Field: "repository", Reason: "existing repository did not prove encrypted connected status"}
	}
	snapshots, err := s.eng.SnapshotList(ctx, in.Name, "")
	if err != nil {
		return Destination{}, fmt.Errorf("enumerate existing repository history %q: %w", in.Name, err)
	}
	if len(snapshots) == 0 {
		return Destination{}, ErrInvalidDestination{Field: "repository", Reason: "connected repository contains no readable snapshots"}
	}
	present := make(map[string]struct{}, len(snapshots))
	for _, snapshot := range snapshots {
		present[snapshot.ID] = struct{}{}
	}
	for snapshotID := range expectedSnapshots {
		if _, ok := present[snapshotID]; !ok {
			return Destination{}, ErrInvalidDestination{Field: "repository", Reason: "connected repository does not contain every expected snapshot"}
		}
	}
	if in.Backend == BackendFilesystem {
		if err := s.bundle.RebindMetadata(ctx, meta); err != nil {
			return Destination{}, fmt.Errorf("update path projection after repository import: %w", err)
		}
	}
	capPolicy := in.CapPolicy
	if capPolicy == "" {
		capPolicy = CapPolicyAlertBlock
	}
	createdAt := in.CreatedAt
	if in.Backend == BackendFilesystem {
		createdAt = meta.CreatedAt
	}
	destination := Destination{
		ID: in.ID, Name: in.Name, BackendKind: in.Backend,
		Location: in.Location, RepositoryLocation: repositoryLocation,
		CapBytes: in.CapBytes, CapPolicy: capPolicy,
		EncryptionAlgorithm: status.EncryptionAlgorithm, ProvisioningState: ProvisioningReady,
		SecretRef: secretRef, CreatedAt: createdAt,
		RelativePath:   filepath.ToSlash(volumeRelativePath),
		DeviceIdentity: identity, DeviceIdentityObservedAt: observedAt,
	}
	return s.repo.Create(ctx, destination)
}

func (s *service) finishPendingCreate(ctx context.Context, pending Destination, acknowledged, allowUnrepairable bool) (Destination, error) {
	// RepoStatus proves encryption is on (EncryptionAlgorithm must be non-empty).
	status, err := s.eng.RepoStatus(ctx, pending.Name)
	if err != nil {
		return Destination{}, fmt.Errorf("destination %q has a pending create intent; repository status failed, so preserve it for owner reconciliation and do not infer repository identity: %w", pending.Name, err)
	}
	if !status.Connected || status.EncryptionAlgorithm == "" {
		return Destination{}, ErrInvalidDestination{
			Field:  "encryption_algorithm",
			Reason: "kopia repository connection or encryption could not be verified",
		}
	}
	pending.EncryptionAlgorithm = status.EncryptionAlgorithm
	if pending.BackendKind == BackendFilesystem {
		if pending.DeviceIdentity != nil {
			if err := s.recheckCreateReadiness(ctx, pending.Location, *pending.DeviceIdentity, acknowledged, allowUnrepairable); err != nil {
				return Destination{}, fmt.Errorf("destination %q has a pending create intent; readiness must clear before reconciliation: %w", pending.Name, err)
			}
		}
	}
	saved, err := s.repo.Update(ctx, pending)
	if err != nil {
		return Destination{}, fmt.Errorf("destination %q has a pending create intent; catalog update failed, so preserve it for owner reconciliation: %w", pending.Name, err)
	}
	if saved.BackendKind == BackendFilesystem && s.bundle != nil {
		if err := s.bundle.WriteMetadata(ctx, BundleMetadata{
			DestinationID:       saved.ID,
			Name:                saved.Name,
			Backend:             string(saved.BackendKind),
			BundleRoot:          saved.Location,
			RelativePath:        saved.RelativePath,
			RepositoryPath:      saved.RepositoryLocation,
			EncryptionAlgorithm: saved.EncryptionAlgorithm,
			SecretRef:           saved.SecretRef,
			CreatedAt:           saved.CreatedAt,
			Host:                hostname(),
			User:                username(),
		}); err != nil {
			return Destination{}, fmt.Errorf("destination %q has a pending create intent; bundle metadata finalization failed, so preserve it for owner reconciliation: %w", saved.Name, err)
		}
	}
	saved.ProvisioningState = ProvisioningReady
	ready, err := s.repo.Update(ctx, saved)
	if err != nil {
		return Destination{}, fmt.Errorf("destination %q has complete repository metadata but remains pending; retry the same request to finalize catalog state: %w", saved.Name, err)
	}
	return ready, nil
}

func (s *service) reconcilePendingCreate(ctx context.Context, pending Destination, in CreateInput, requestedLocation string, capPolicy CapPolicy) (Destination, error) {
	if pending.ProvisioningState != ProvisioningCreated {
		return Destination{}, ErrInvalidDestination{Field: "provisioning", Reason: "repository create success is not durably confirmed; automatic connect or retry is unsafe"}
	}
	if pending.BackendKind != in.Backend || pending.CapBytes != in.CapBytes || pending.CapPolicy != capPolicy {
		return Destination{}, ErrInvalidDestination{Field: "name", Reason: "pending destination create exists with different settings; preserve it for explicit reconciliation"}
	}
	if pending.BackendKind == BackendS3 {
		if filepath.Clean(pending.Location) != filepath.Clean(requestedLocation) {
			return Destination{}, ErrInvalidDestination{Field: "location", Reason: "pending destination create path differs; preserve it for explicit reconciliation"}
		}
		return s.finishPendingCreate(ctx, pending, in.AcknowledgeReadinessFailure, in.AcknowledgeUnrepairableFilesystem)
	}
	if pending.DeviceIdentity == nil || !pending.DeviceIdentity.StrongIdentity() || pending.RelativePath == "" || s.readiness == nil || s.bundle == nil {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "pending filesystem destination lacks a complete strong-identity intent; preserve it for explicit reconciliation"}
	}
	inspector, ok := s.readiness.(recoveryIdentityInspector)
	if !ok {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "stable device inspection is unavailable for pending destination reconciliation"}
	}
	observed, err := inspector.InspectDevice(ctx, *pending.DeviceIdentity)
	if err != nil || !sameStrongDeviceIdentity(*pending.DeviceIdentity, observed.Identity) || !observed.Mounted || observed.ReadOnly || observed.FilesystemState != sysmounts.FilesystemStateClean {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "pending destination volume is not proven at a clean, mounted, writable identity"}
	}
	requestedRelative, err := destinationreadiness.RelativePathUnderMount(requestedLocation, observed.Identity.Mountpoint)
	if err != nil || filepath.Clean(requestedRelative) != filepath.Clean(pending.RelativePath) {
		return Destination{}, ErrInvalidDestination{Field: "location", Reason: "retry does not match the pending volume-relative destination intent"}
	}
	if s.volumeLease == nil {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "host volume operation lease is unavailable for pending destination reconciliation"}
	}
	release, err := s.volumeLease.Acquire(ctx, observed.Identity)
	if err != nil {
		return Destination{}, fmt.Errorf("acquire host volume operation lease for pending destination: %w", err)
	}
	defer release()

	// Re-observe after the lease, resolve the same relative locator at the new
	// mount, and connect only. This path never invokes RepoCreate or cleanup.
	observed, err = inspector.InspectDevice(ctx, observed.Identity)
	if err != nil || !sameStrongDeviceIdentity(*pending.DeviceIdentity, observed.Identity) || !observed.Mounted || observed.ReadOnly || observed.FilesystemState != sysmounts.FilesystemStateClean {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "pending destination volume changed while acquiring its lease"}
	}
	if !filepath.IsAbs(filepath.Clean(observed.Identity.Mountpoint)) || filepath.Clean(observed.Identity.Mountpoint) == string(filepath.Separator) {
		return Destination{}, ErrInvalidDestination{Field: "mountpoint", Reason: "verified volume has no safe non-root mountpoint after lease acquisition"}
	}
	bundleRoot, err := destinationreadiness.ResolveRelativePath(observed.Identity.Mountpoint, pending.RelativePath)
	if err != nil {
		return Destination{}, err
	}
	repositoryPath := RepositoryPathFor(bundleRoot, pending.Name)
	resolvedMount, resolvedBundle, err := resolveContainedPath(observed.Identity.Mountpoint, bundleRoot)
	if err != nil {
		return Destination{}, ErrInvalidDestination{Field: "relative_path", Reason: "pending bundle resolves outside the verified volume: " + err.Error()}
	}
	_, resolvedRepository, err := resolveContainedPath(bundleRoot, repositoryPath)
	if err != nil {
		return Destination{}, ErrInvalidDestination{Field: "repository_path", Reason: "pending repository resolves outside its bundle: " + err.Error()}
	}
	if _, err := repositoryUniqueID(resolvedRepository); err != nil {
		return Destination{}, fmt.Errorf("inspect pending repository marker: %w", err)
	}
	destinations, err := s.repo.List(ctx, defaultListLimit)
	if err != nil {
		return Destination{}, fmt.Errorf("check pending destination locator collisions: %w", err)
	}
	for _, other := range destinations {
		if other.ID == pending.ID || other.BackendKind != BackendFilesystem {
			continue
		}
		for _, candidate := range []string{other.Location, other.RepositoryLocation} {
			if resolvedOther, evalErr := filepath.EvalSymlinks(candidate); evalErr == nil && diskPathsOverlap(resolvedBundle, resolvedOther) {
				return Destination{}, ErrInvalidDestination{Field: "relative_path", Reason: "another destination claims the pending bundle or repository path"}
			}
		}
		if other.DeviceIdentity != nil && other.DeviceIdentity.MatchesDevice(observed.Identity) {
			otherRoot, resolveErr := destinationreadiness.ResolveRelativePath(resolvedMount, other.RelativePath)
			if resolveErr == nil {
				if otherResolved, evalErr := filepath.EvalSymlinks(otherRoot); evalErr == nil && diskPathsOverlap(resolvedBundle, otherResolved) {
					return Destination{}, ErrInvalidDestination{Field: "relative_path", Reason: "another destination claims the pending volume-relative path"}
				}
			}
		}
	}
	if err := s.recheckCreateReadiness(ctx, bundleRoot, observed.Identity, in.AcknowledgeReadinessFailure, in.AcknowledgeUnrepairableFilesystem); err != nil {
		return Destination{}, fmt.Errorf("pending destination readiness: %w", err)
	}
	// A durable Created state alone does not prove that the repository currently
	// at this path belongs to this destination attempt. Require the manifest and
	// Kopia unique ID anchor before RepoRebind can mutate resource-kopia routing.
	meta := BundleMetadata{DestinationID: pending.ID, Name: pending.Name, Backend: string(pending.BackendKind), BundleRoot: bundleRoot, RelativePath: pending.RelativePath, RepositoryPath: repositoryPath, EncryptionAlgorithm: pending.EncryptionAlgorithm, SecretRef: pending.SecretRef, CreatedAt: pending.CreatedAt, Host: hostname(), User: username()}
	if err := s.bundle.ValidatePendingCreate(ctx, meta); err != nil {
		return Destination{}, fmt.Errorf("pending destination identity is not anchored by its manifest and repository; preserve for explicit reconciliation: %w", err)
	}
	if err := s.eng.RepoRebind(ctx, pending.Name, repositoryPath); err != nil {
		return Destination{}, fmt.Errorf("connect pending repository without recreating it: %w", err)
	}
	if err := s.bundle.RebindMetadata(ctx, meta); err != nil {
		return Destination{}, fmt.Errorf("rebind verified pending destination metadata: %w", err)
	}
	pending.Location = bundleRoot
	pending.RepositoryLocation = repositoryPath
	pending.DeviceIdentity = &observed.Identity
	pending.DeviceIdentityObservedAt = time.Now().UTC()
	if _, err := s.repo.Update(ctx, pending); err != nil {
		return Destination{}, fmt.Errorf("persist resolved pending destination path: %w", err)
	}
	return s.finishPendingCreate(ctx, pending, in.AcknowledgeReadinessFailure, in.AcknowledgeUnrepairableFilesystem)
}

// hostname / username best-effort host identity for the manifest. Empty on
// error — they are optional, non-secret context only.
func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}

func username() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return os.Getenv("USERNAME")
}

func (s *service) GetDestination(ctx context.Context, id string) (Destination, error) {
	if strings.TrimSpace(id) == "" {
		return Destination{}, ErrInvalidDestination{Field: "id", Reason: "required"}
	}
	return s.repo.GetByID(ctx, id)
}

// snapshotMatchesTarget accepts both snapshot identity generations:
//   - current: dbm.owner/dbm.name tags (Kopia stores them as tag:dbm.* keys,
//     normalized by the engine);
//   - legacy: the target is the suffix of Kopia's source path, e.g.
//     "user@host:/…/dbm:/agent-manager/data".
//
// A present tag that names another target rejects the snapshot, so a
// misattributed snapshot is never returned for the wrong target. The source
// suffix is only consulted when the corresponding tag is absent.
func snapshotMatchesTarget(snapshot engine.Snapshot, owner, name string) bool {
	if strings.TrimSpace(snapshot.ID) == "" {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, snapshot.StartTime); err != nil {
		return false
	}
	tagOwner, tagName := "", ""
	for _, tag := range snapshot.Tags {
		switch {
		case strings.HasPrefix(tag, "dbm.owner:"):
			tagOwner = strings.TrimPrefix(tag, "dbm.owner:")
		case strings.HasPrefix(tag, "dbm.name:"):
			tagName = strings.TrimPrefix(tag, "dbm.name:")
		}
	}
	if tagOwner != "" && tagOwner != owner {
		return false
	}
	if tagName != "" && tagName != name {
		return false
	}
	if tagOwner == owner && tagName == name {
		return true
	}
	if snapshot.Source == "dbm://"+owner+"/"+name {
		return true
	}
	// A partially tagged snapshot still needs the legacy source suffix to
	// identify the target; tags alone are not evidence for a missing half.
	return strings.HasSuffix(snapshot.Source, ":/"+owner+"/"+name)
}

func (s *service) ListDestinations(ctx context.Context, pageSize int) ([]Destination, error) {
	if pageSize <= 0 {
		pageSize = defaultListLimit
	}
	return s.repo.List(ctx, pageSize)
}

func (s *service) ListTargetSnapshots(ctx context.Context, destinationID, owner, name string) ([]engine.Snapshot, error) {
	destinationID, owner, name = strings.TrimSpace(destinationID), strings.TrimSpace(owner), strings.TrimSpace(name)
	if destinationID == "" || owner == "" || name == "" || strings.ContainsAny(owner+name, "\x00/\\") {
		return nil, ErrInvalidDestination{Field: "snapshot identity", Reason: "destination, owner and name are required stable DBM identifiers"}
	}
	var snapshots []engine.Snapshot
	err := s.WithResolvedDestinationForRead(ctx, destinationID, func(d Destination) error {
		var listErr error
		snapshots, listErr = s.eng.SnapshotListByTags(ctx, d.Name, []string{
			"dbm.owner:" + owner, "dbm.name:" + name,
		})
		if listErr == nil && len(snapshots) == 0 {
			// Legacy snapshots predate the DBM tags and exist only in the source
			// path suffix. Pay for the repository-wide inventory only when the
			// tag-filtered path proves there is nothing newer to show.
			snapshots, listErr = s.eng.SnapshotList(ctx, d.Name, "")
		}
		return listErr
	})
	if err != nil {
		return nil, fmt.Errorf("list target snapshots: %w", err)
	}
	filtered := make([]engine.Snapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		// The engine may return a repository-wide inventory. Unrelated snapshots
		// are expected, not an integrity failure: selecting the exact target is
		// this function's job. New snapshots carry dbm.owner/dbm.name tags; older
		// snapshots carry the target as the source path suffix ":/owner/name".
		if !snapshotMatchesTarget(snapshot, owner, name) {
			continue
		}
		filtered = append(filtered, snapshot)
	}
	sort.Slice(filtered, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339Nano, filtered[i].StartTime)
		tj, _ := time.Parse(time.RFC3339Nano, filtered[j].StartTime)
		if ti.Equal(tj) {
			return filtered[i].ID < filtered[j].ID
		}
		return ti.After(tj)
	})
	return filtered, nil
}

func (s *service) UpdateDestination(ctx context.Context, in UpdateInput) (Destination, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Destination{}, ErrInvalidDestination{Field: "id", Reason: "required"}
	}
	release, err := lockDestination(ctx, in.ID)
	if err != nil {
		return Destination{}, err
	}
	defer release()
	existing, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		return Destination{}, err
	}
	if existing.ProvisioningState != "" && existing.ProvisioningState != ProvisioningReady {
		return Destination{}, ErrInvalidDestination{Field: "provisioning", Reason: "destination creation is incomplete; reconcile it before changing settings"}
	}
	existing.CapBytes = in.CapBytes
	if in.CapPolicy != "" {
		existing.CapPolicy = in.CapPolicy
	}
	return s.repo.Update(ctx, existing)
}

func (s *service) BindRecoveryIdentity(ctx context.Context, id string, expected destinationreadiness.DeviceIdentity) (Destination, error) {
	if strings.TrimSpace(id) == "" {
		return Destination{}, ErrInvalidDestination{Field: "id", Reason: "required"}
	}
	release, err := lockDestination(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	defer release()
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	if existing.BackendKind != BackendFilesystem {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "only filesystem destinations can bind a host volume"}
	}
	mountRoot := filepath.Clean(strings.TrimSpace(expected.Mountpoint))
	if !expected.StrongIdentity() {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "a stable UUID or serial is required"}
	}
	if !filepath.IsAbs(mountRoot) || mountRoot == string(filepath.Separator) {
		return Destination{}, ErrInvalidDestination{Field: "identity.mountpoint", Reason: "an absolute non-root mount location is required"}
	}
	relative, relErr := destinationreadiness.RelativePathUnderMount(existing.Location, mountRoot)
	if relErr != nil {
		return Destination{}, ErrInvalidDestination{Field: "identity.mountpoint", Reason: relErr.Error()}
	}
	inspector, ok := s.readiness.(recoveryIdentityInspector)
	if !ok {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "stable device inspection is unavailable"}
	}
	observed, err := inspector.InspectDevice(ctx, expected)
	if err != nil {
		return Destination{}, fmt.Errorf("re-observe recovery volume identity: %w", err)
	}
	if !expected.MatchesDevice(observed.Identity) || !observed.Identity.StrongIdentity() {
		return Destination{}, ErrInvalidDestination{Field: "identity", Reason: "observed volume does not match the requested stable identity"}
	}
	if observed.Mounted && filepath.Clean(observed.Identity.Mountpoint) != mountRoot {
		return Destination{}, ErrInvalidDestination{Field: "identity.mountpoint", Reason: "the volume is currently mounted at a different location"}
	}
	observed.Identity.Mountpoint = mountRoot
	existing.DeviceIdentity = &observed.Identity
	existing.DeviceIdentityObservedAt = time.Now().UTC()
	existing.RelativePath = relative
	return s.repo.Update(ctx, existing)
}

func (s *service) ReconcileCredentialReferences(ctx context.Context) error {
	dests, err := s.repo.List(ctx, defaultListLimit)
	if err != nil {
		return fmt.Errorf("list destinations for credential-reference reconciliation: %w", err)
	}
	var failures []string
	for _, d := range dests {
		canonical := s.eng.PassphraseRef(d.Name)
		if canonical == "" || d.SecretRef == canonical {
			continue
		}
		// The catalog is the authoritative runtime record consumed by DBM's
		// orchestration. Update it even when a read-only filesystem prevents the
		// generated bundle metadata from being refreshed; leaving the catalog on
		// a retired provider-shaped reference makes a healthy repository appear
		// unavailable and keeps Vault paths alive in shipped/read APIs.
		d.SecretRef = canonical
		if _, err := s.repo.Update(ctx, d); err != nil {
			failures = append(failures, fmt.Sprintf("%s catalog: %v", d.Name, err))
			continue
		}
		if d.BackendKind == BackendFilesystem && s.bundle != nil {
			if err := s.WithResolvedDestination(ctx, d.ID, func(current Destination) error {
				return s.bundle.RefreshMetadata(ctx, BundleMetadata{
					DestinationID:       current.ID,
					Name:                current.Name,
					Backend:             string(current.BackendKind),
					BundleRoot:          current.Location,
					RelativePath:        current.RelativePath,
					RepositoryPath:      current.RepositoryLocation,
					EncryptionAlgorithm: current.EncryptionAlgorithm,
					SecretRef:           current.SecretRef,
					CreatedAt:           current.CreatedAt,
				})
			}); err != nil {
				failures = append(failures, fmt.Sprintf("%s metadata: %v", d.Name, err))
			}
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("destination credential-reference reconciliation incomplete: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (s *service) DeleteDestination(ctx context.Context, id string, deleteRepository bool) (bool, error) {
	if strings.TrimSpace(id) == "" {
		return false, ErrInvalidDestination{Field: "id", Reason: "required"}
	}
	if deleteRepository {
		var removed bool
		err := s.WithResolvedDestination(ctx, id, func(d Destination) error {
			if err := s.eng.RepoDelete(ctx, d.Name); err != nil {
				return err
			}
			var deleteErr error
			removed, deleteErr = s.repo.Delete(ctx, id)
			return deleteErr
		})
		return removed, err
	}
	release, err := lockDestination(ctx, id)
	if err != nil {
		return false, err
	}
	defer release()
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	if existing.ProvisioningState != "" && existing.ProvisioningState != ProvisioningReady {
		return false, ErrInvalidDestination{Field: "provisioning", Reason: "destination creation is incomplete; automatic deletion is unsafe, reconcile it first"}
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) GetDestinationUsage(ctx context.Context, id string) (UsageReport, error) {
	var d Destination
	var stats engine.RepoStats
	err := s.WithResolvedDestinationForRead(ctx, id, func(current Destination) error {
		d = current
		var statsErr error
		stats, statsErr = s.eng.RepoStats(ctx, current.Name)
		return statsErr
	})
	if err != nil {
		return UsageReport{}, err
	}
	state := computeUsageState(stats.SizeBytes, d.CapBytes)
	return UsageReport{
		UsageBytes: stats.SizeBytes,
		CapBytes:   d.CapBytes,
		UsageState: state,
		CapPolicy:  d.CapPolicy,
	}, nil
}

func (s *service) WouldBlock(ctx context.Context, destinationID string, pendingBytes int64) (bool, string, error) {
	var blocked bool
	var reason string
	err := s.WithResolvedDestinationForRead(ctx, destinationID, func(d Destination) error {
		if d.CapBytes == 0 || d.CapPolicy == CapPolicyAlertOnly {
			return nil
		}
		stats, err := s.eng.RepoStats(ctx, d.Name)
		if err != nil {
			return err
		}
		if stats.SizeBytes+pendingBytes > d.CapBytes {
			blocked = true
			reason = "storage cap exceeded; write blocked by alert+block policy"
		}
		return nil
	})
	if err != nil {
		return false, "", err
	}
	return blocked, reason, nil
}

// computeUsageState classifies usage against cap. cap == 0 means no cap → always WITHIN.
func computeUsageState(usageBytes, capBytes int64) UsageState {
	if capBytes == 0 {
		return UsageStateWithin
	}
	if usageBytes >= capBytes {
		return UsageStateOver
	}
	if float64(usageBytes) >= float64(capBytes)*nearCapThreshold {
		return UsageStateNear
	}
	return UsageStateWithin
}

// unrepairableFilesystem refuses a filesystem this host has no metadata repair
// tool for, unless the operator explicitly accepts the risk.
//
// The point is not that such a filesystem works badly day to day — NTFS works
// fine through ntfs-3g. The point is the guarantee after an unsafe shutdown:
// this host can reset NTFS's dirty flag, but it cannot repair an MFT record, so
// one interrupted write can leave files permanently unreadable with no local
// remedy. ext4 replays its journal and has a real repair tool, so it can meet
// that guarantee.
func unrepairableFilesystem(filesystem string, acknowledged bool) (string, bool) {
	if acknowledged {
		return "", false
	}
	if driveprofile.NormalizeOS(runtime.GOOS) != "linux" {
		return "", false
	}
	profile, ok := driveprofile.Lookup(filesystem)
	if !ok || profile.RepairIsMetadata {
		return "", false
	}
	limitation := strings.TrimSpace(profile.RepairLimitation)
	if limitation == "" {
		limitation = "this host has no check or repair tool for it"
	}
	return fmt.Sprintf(
		"%s cannot be repaired on this host: %s. A destination on it is safe until the first unsafe shutdown and unrecoverable after it. Use a filesystem this host can repair (ext4), or pass --allow-unrepairable if you accept that risk",
		profile.Name, limitation,
	), true
}
