package dependencygovernance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/capacity"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/installgateway"
)

const diskJobOwnerID = "scenario-dependency-analyzer"

type diskJobBroker interface {
	ReserveDiskJob(context.Context, capacity.DiskJobRequest) (capacity.DiskJobLease, error)
}

type diskAdmissionPlanner func(context.Context, installgateway.Resolution) (installgateway.DiskAdmissionPlan, error)

// WithDiskJobBroker attaches the shared host capacity broker to governed
// installs. Apply requests fail closed when the broker is absent or cannot
// reserve every write path.
func WithDiskJobBroker(broker interface {
	ReserveDiskJob(context.Context, capacity.DiskJobRequest) (capacity.DiskJobLease, error)
}) ConnectOption {
	return func(h *connectHandler) { h.diskBroker = broker }
}

// installDiskPlan is an install's admitted disk requirement: the planner's
// result with every path canonicalized and merged. Dry-run and apply build it
// the same way, so a preview never approves what apply would refuse.
type installDiskPlan struct {
	plan  installgateway.DiskAdmissionPlan
	paths []capacity.DiskJobPath
	total int64
	// class is capacity.DiskJobClassMetadata when apply downloads nothing and
	// writes only manifests, lockfiles and module-cache bookkeeping, and empty
	// for every job that can grow the package store or node_modules.
	class string
}

// Release frees what planning staged (for example a pnpm staging directory).
func (p installDiskPlan) Release() { p.plan.Release() }

// planInstallDisk runs the install's disk-admission planner and validates the
// result without reserving anything. A shared capacity broker must exist,
// because apply cannot proceed without one.
func planInstallDisk(ctx context.Context, broker diskJobBroker, planner diskAdmissionPlanner, resolution installgateway.Resolution) (installDiskPlan, error) {
	if broker == nil {
		return installDiskPlan{}, fmt.Errorf("shared host disk admission broker is unavailable")
	}
	if planner == nil {
		planner = installgateway.PlanDiskAdmission
	}
	plan, err := planner(ctx, resolution)
	if err != nil {
		plan.Release()
		return installDiskPlan{}, err
	}
	admitted := installDiskPlan{plan: plan}
	if plan.Unsupported != "" {
		return admitted, errors.New(plan.Unsupported)
	}
	var total int64
	paths := make([]capacity.DiskJobPath, 0, len(plan.Paths))
	for _, path := range plan.Paths {
		if strings.TrimSpace(path.Role) == "" || strings.TrimSpace(path.Path) == "" || path.PeakBytes <= 0 {
			return admitted, fmt.Errorf("disk admission plan contains an incomplete path requirement")
		}
		if path.PeakBytes > installgateway.DiskJobPlanningCeilingBytes || total > installgateway.DiskJobPlanningCeilingBytes-path.PeakBytes {
			return admitted, fmt.Errorf("disk admission plan exceeds the %d-byte per-job planning ceiling", installgateway.DiskJobPlanningCeilingBytes)
		}
		total += path.PeakBytes
		canonical, err := canonicalDiskAdmissionPath(path.Path)
		if err != nil {
			return admitted, fmt.Errorf("canonicalize disk admission path for %s: %w", path.Role, err)
		}
		paths = append(paths, capacity.DiskJobPath{Path: canonical, Role: path.Role, PeakBytes: path.PeakBytes})
	}
	if total <= 0 || total > installgateway.DiskJobPlanningCeilingBytes {
		return admitted, fmt.Errorf("disk admission estimate %d exceeds the %d-byte per-job planning ceiling", total, installgateway.DiskJobPlanningCeilingBytes)
	}
	admitted.paths = mergeSameDiskAdmissionPaths(paths)
	admitted.total = total
	// An apply that downloads nothing and writes no package store (inputs
	// published without linking, a Go install of cached or first-party modules)
	// is the metadata class: a small, bounded write that is not held to the
	// install-sized floor. Everything that can fetch or link is not.
	if plan.WritesInputsOnly && total <= capacity.DiskJobMetadataMaxPeakBytes {
		admitted.class = capacity.DiskJobClassMetadata
	}
	return admitted, nil
}

// reserve asks the shared broker for the planned paths.
func (p installDiskPlan) reserve(ctx context.Context, broker diskJobBroker) (capacity.DiskJobLease, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("create disk admission job identity: %w", err)
	}
	lease, err := broker.ReserveDiskJob(ctx, capacity.DiskJobRequest{
		OwnerID:    diskJobOwnerID,
		JobID:      "install-" + hex.EncodeToString(nonce[:]),
		Paths:      p.paths,
		FloorBytes: p.floorBytes(),
		Class:      p.class,
	})
	if err != nil {
		return nil, fmt.Errorf("host disk admission refused install before writes: %w", err)
	}
	return lease, nil
}

// floorBytes is the floor the job asks to be held to: the install-sized floor,
// or none of its own for a metadata job, which the capacity owner holds to its
// metadata floor.
func (p installDiskPlan) floorBytes() int64 {
	if p.class == capacity.DiskJobClassMetadata {
		return 0
	}
	return installgateway.DiskJobFloorBytes
}

// summary is the preview line reviewers see in a dry run.
func (p installDiskPlan) summary() string {
	kind := ""
	if p.class == capacity.DiskJobClassMetadata {
		kind = " as a metadata job (manifests and lockfiles only; no download, no package-store write)"
	}
	return fmt.Sprintf("disk admission (preview): peak %s across %d path(s)%s; the same admission gates --apply", humanBytes(p.total), len(p.paths), kind)
}

func humanBytes(n int64) string {
	const unit = 1 << 10
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for value := n / unit; value >= unit; value /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func canonicalDiskAdmissionPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	candidate := filepath.Clean(absolute)
	var suffix []string
	for {
		info, statErr := os.Lstat(candidate)
		if statErr == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("admission target is not a directory")
			}
			resolved, resolveErr := filepath.EvalSymlinks(candidate)
			if resolveErr != nil {
				return "", resolveErr
			}
			return filepath.Clean(filepath.Join(append([]string{resolved}, suffix...)...)), nil
		}
		if !os.IsNotExist(statErr) {
			return "", statErr
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return "", fmt.Errorf("no existing parent directory")
		}
		suffix = append([]string{filepath.Base(candidate)}, suffix...)
		candidate = parent
	}
}

func mergeSameDiskAdmissionPaths(paths []capacity.DiskJobPath) []capacity.DiskJobPath {
	byPath := make(map[string]capacity.DiskJobPath, len(paths))
	for _, path := range paths {
		if old, ok := byPath[path.Path]; ok {
			old.PeakBytes += path.PeakBytes
			old.Role = strings.Trim(old.Role+","+path.Role, ",")
			byPath[path.Path] = old
		} else {
			byPath[path.Path] = path
		}
	}
	out := make([]capacity.DiskJobPath, 0, len(byPath))
	for _, path := range byPath {
		out = append(out, path)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

type diskJobRun struct {
	Output  string
	RunErr  error
	Receipt capacity.DiskJobReceipt
	LifeErr error
}

const diskObservationInterval = time.Second

func observeAndExtendDiskJob(ctx context.Context, lease capacity.DiskJobLease) error {
	observation, err := lease.Observe(ctx)
	if err != nil {
		return fmt.Errorf("observe disk job headroom: %w", err)
	}
	receipt := lease.Reservation()
	baseFree := make(map[string]int64, len(receipt.Filesystems))
	for _, filesystem := range receipt.Filesystems {
		baseFree[filesystem.FilesystemID] = filesystem.AvailableBytes
	}
	var totalPeak int64
	for _, path := range receipt.Paths {
		if path.PeakBytes > math.MaxInt64-totalPeak {
			return fmt.Errorf("disk job peak estimate overflowed")
		}
		totalPeak += path.PeakBytes
	}
	extensions := make([]capacity.DiskJobPath, 0)
	for _, filesystem := range observation.Filesystems {
		baseline, ok := baseFree[filesystem.FilesystemID]
		if !ok {
			return fmt.Errorf("disk job observation returned an unreserved filesystem %s", filesystem.FilesystemID)
		}
		growth := baseline - filesystem.CurrentFreeBytes
		if growth < 0 {
			growth = 0
		}
		peak := filesystem.CurrentJobPeakBytes
		if peak <= 0 || growth < peak*4/5 {
			continue
		}
		// Extend before the measured growth reaches the current reservation.
		// Double the observed growth to leave room for another sampling interval
		// and cancel if either that extension or the owner ceiling cannot fit.
		if growth > math.MaxInt64/2 {
			return fmt.Errorf("observed disk growth cannot be safely represented for extension")
		}
		wantPeak := growth * 2
		if wantPeak <= peak {
			wantPeak = peak + 1<<20
		}
		extra := wantPeak - peak
		if extra <= 0 || totalPeak > installgateway.DiskJobPlanningCeilingBytes-extra {
			return fmt.Errorf("observed disk growth is approaching its lease and cannot be safely extended within the %d-byte job ceiling", installgateway.DiskJobPlanningCeilingBytes)
		}
		if filesystem.SafeAdditionalPeakBytes < extra {
			return fmt.Errorf("observed disk growth needs %d additional bytes but only %d can be reserved while preserving the floor", extra, filesystem.SafeAdditionalPeakBytes)
		}
		var chosen *capacity.DiskJobPathReceipt
		for i := range receipt.Paths {
			if receipt.Paths[i].FilesystemID == filesystem.FilesystemID && receipt.Paths[i].PeakBytes > 0 {
				chosen = &receipt.Paths[i]
				break
			}
		}
		if chosen == nil {
			return fmt.Errorf("disk job cannot extend filesystem %s because its write path is missing", filesystem.FilesystemID)
		}
		extensions = append(extensions, capacity.DiskJobPath{Path: chosen.Path, Role: chosen.Role + ",observed-growth-extension", PeakBytes: chosen.PeakBytes + extra})
		totalPeak += extra
	}
	if len(extensions) == 0 {
		return nil
	}
	if _, err := lease.Extend(ctx, extensions); err != nil {
		return fmt.Errorf("extend disk job before observed growth exceeds reservation: %w", err)
	}
	return nil
}

// runWithDiskJob keeps the lease alive while the installer runs. Complete is
// called only after the installer has returned, which means its synchronous
// staging/publication cleanup has finished. A renewal failure cancels the
// install context and leaves the durable reservation for owner reconciliation.
func runWithDiskJob(ctx context.Context, lease capacity.DiskJobLease, resolution installgateway.Resolution, run func(context.Context) (string, error)) diskJobRun {
	if lease == nil {
		return diskJobRun{LifeErr: fmt.Errorf("disk job lease is unavailable")}
	}
	opCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	runDone := make(chan diskJobRun, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				runDone <- diskJobRun{RunErr: fmt.Errorf("installer panicked: %v", recovered), LifeErr: fmt.Errorf("installer cleanup is unverified; disk reservation retained for reconciliation")}
			}
		}()
		output, err := run(opCtx)
		runDone <- diskJobRun{Output: output, RunErr: err}
	}()
	ticker := time.NewTicker(capacity.RecommendedLeaseHeartbeatInterval())
	defer ticker.Stop()
	observeTicker := time.NewTicker(diskObservationInterval)
	defer observeTicker.Stop()
	var result diskJobRun
	for {
		select {
		case result = <-runDone:
			goto finished
		case <-ticker.C:
			if err := lease.Heartbeat(ctx); err != nil {
				cancel()
				result = <-runDone
				result.Receipt = lease.Reservation()
				result.LifeErr = fmt.Errorf("disk job heartbeat failed; install was cancelled and reservation retained for reconciliation: %w", err)
				return result
			}
		case <-observeTicker.C:
			if err := observeAndExtendDiskJob(ctx, lease); err != nil {
				cancel()
				result = <-runDone
				result.Receipt = lease.Reservation()
				result.LifeErr = fmt.Errorf("disk headroom was lost; install was cancelled and reservation retained for reconciliation: %w", err)
				return result
			}
		case <-ctx.Done():
			cancel()
			result = <-runDone
			result.RunErr = context.Cause(ctx)
			if result.RunErr == nil {
				result.RunErr = ctx.Err()
			}
			goto finished
		}
	}

finished:
	if result.LifeErr != nil {
		return result
	}
	if err := installgateway.VerifyDiskAdmissionCleanup(resolution, result.RunErr == nil); err != nil {
		result.Receipt = lease.Reservation()
		result.LifeErr = fmt.Errorf("disk job cleanup could not be verified; reservation remains retained for reconciliation: %w", err)
		return result
	}
	completeCtx, completeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer completeCancel()
	previousReceipt := lease.Reservation()
	completed, completeErr := lease.Complete(completeCtx)
	if completeErr != nil {
		result.Receipt = previousReceipt
		result.LifeErr = fmt.Errorf("disk job completion could not be verified; reservation remains retained: %w", completeErr)
	} else {
		result.Receipt = completed
	}
	return result
}

func diskReceiptNote(receipt capacity.DiskJobReceipt) string {
	var paths []string
	for _, path := range receipt.Paths {
		paths = append(paths, fmt.Sprintf("%s=%s:%d", path.Role, path.FilesystemID, path.PeakBytes))
	}
	var filesystems []string
	for _, fs := range receipt.Filesystems {
		actual := "unknown"
		if fs.ActualDeltaBytes != nil {
			actual = fmt.Sprintf("%d", *fs.ActualDeltaBytes)
		}
		filesystems = append(filesystems, fmt.Sprintf("%s peak=%d actual=%s available=%d other_reserved=%d floor=%d", fs.FilesystemID, fs.PeakBytes, actual, fs.AvailableBytes, fs.OtherReservedBytes, fs.FloorBytes))
	}
	return fmt.Sprintf("disk admission receipt: id=%s status=%s cleanup_verified=%t paths=[%s] filesystems=[%s]", receipt.ID, receipt.Status, receipt.CleanupVerified, strings.Join(paths, ","), strings.Join(filesystems, ";"))
}
