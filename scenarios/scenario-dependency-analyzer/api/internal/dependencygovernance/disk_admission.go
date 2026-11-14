package dependencygovernance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

func reserveInstallDiskJob(ctx context.Context, broker diskJobBroker, planner diskAdmissionPlanner, resolution installgateway.Resolution) (capacity.DiskJobLease, installgateway.DiskAdmissionPlan, error) {
	if broker == nil {
		return nil, installgateway.DiskAdmissionPlan{}, fmt.Errorf("shared host disk admission broker is unavailable")
	}
	if planner == nil {
		planner = installgateway.PlanDiskAdmission
	}
	plan, err := planner(ctx, resolution)
	if err != nil {
		return nil, plan, err
	}
	if plan.Unsupported != "" {
		return nil, plan, fmt.Errorf("install path is not fully covered by disk admission: %s", plan.Unsupported)
	}
	paths := make([]capacity.DiskJobPath, 0, len(plan.Paths))
	var total int64
	for _, path := range plan.Paths {
		if strings.TrimSpace(path.Role) == "" || strings.TrimSpace(path.Path) == "" || path.PeakBytes <= 0 {
			return nil, plan, fmt.Errorf("disk admission plan contains an incomplete path requirement")
		}
		if path.PeakBytes > installgateway.DiskJobPlanningCeilingBytes || total > installgateway.DiskJobPlanningCeilingBytes-path.PeakBytes {
			return nil, plan, fmt.Errorf("disk admission plan exceeds the %d-byte per-job planning ceiling", installgateway.DiskJobPlanningCeilingBytes)
		}
		total += path.PeakBytes
		canonical, err := canonicalDiskAdmissionPath(path.Path)
		if err != nil {
			return nil, plan, fmt.Errorf("canonicalize disk admission path for %s: %w", path.Role, err)
		}
		paths = append(paths, capacity.DiskJobPath{Path: canonical, Role: path.Role, PeakBytes: path.PeakBytes})
	}
	paths = mergeSameDiskAdmissionPaths(paths)
	if total <= 0 || total > installgateway.DiskJobPlanningCeilingBytes {
		return nil, plan, fmt.Errorf("disk admission estimate %d exceeds the %d-byte per-job planning ceiling", total, installgateway.DiskJobPlanningCeilingBytes)
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, plan, fmt.Errorf("create disk admission job identity: %w", err)
	}
	lease, err := broker.ReserveDiskJob(ctx, capacity.DiskJobRequest{
		OwnerID:    diskJobOwnerID,
		JobID:      "install-" + hex.EncodeToString(nonce[:]),
		Paths:      paths,
		FloorBytes: installgateway.DiskJobFloorBytes,
	})
	if err != nil {
		return nil, plan, fmt.Errorf("host disk admission refused install before writes: %w", err)
	}
	return lease, plan, nil
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
