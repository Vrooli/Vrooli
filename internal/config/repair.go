package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	repocontract "github.com/vrooli/repo-contract-go"
)

const RuntimeHomeOwnershipMigration = "runtime-home-ownership"

type RepairScope struct {
	RootClass string `json:"root_class"`
	RootPath  string `json:"root_path"`
	Legacy    bool   `json:"legacy"`
}

type RepairRequest struct {
	Scope          RepairScope
	ExpectedUID    uint32
	ExpectedGID    uint32
	Apply          bool
	MaxEntries     uint64
	Deadline       time.Time
	FollowSymlinks bool
	// ResumeAfter continues a bounded lexical WalkDir after the last path
	// returned by a previous partial repair. It is intentionally a path, not
	// an inode/cookie: callers persist it as migration progress and a repeated
	// scan remains safe if the tree changes between attempts.
	ResumeAfter string
}

type RepairStatus string

const (
	RepairComplete RepairStatus = "complete"
	RepairPartial  RepairStatus = "partial"
	RepairFailed   RepairStatus = "failed"
)

type RepairFailure struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Action string `json:"action"`
	Reason string `json:"reason"`
}

type VerificationResult struct {
	Checked uint64 `json:"checked"`
	Failed  uint64 `json:"failed"`
}

type RepairResult struct {
	Scope        RepairScope        `json:"scope"`
	Status       RepairStatus       `json:"status"`
	Scanned      uint64             `json:"scanned"`
	Repaired     uint64             `json:"repaired"`
	Skipped      uint64             `json:"skipped"`
	Failed       uint64             `json:"failed"`
	Duration     time.Duration      `json:"duration"`
	Verification VerificationResult `json:"verification"`
	Failures     []RepairFailure    `json:"failures,omitempty"`
	// LastPath is the durable continuation point for a bounded partial walk.
	// It is omitted for an empty or complete walk.
	LastPath string `json:"last_path,omitempty"`
}

type RepairService struct {
	ResolveRoot func(string) (string, error)
}

func NewRepairService() RepairService {
	return RepairService{ResolveRoot: resolveManagedRoot}
}

func (s RepairService) Repair(ctx context.Context, req RepairRequest) (RepairResult, error) {
	started := time.Now()
	result := RepairResult{Scope: req.Scope, Status: RepairComplete}
	prepared, root, absent, err := s.prepareRepairRequest(req)
	if err != nil {
		return result, err
	}
	if absent {
		result.Duration = time.Since(started)
		return result, nil
	}
	walker := repairWalker{ctx: ctx, req: prepared, result: &result, resumeAfter: prepared.ResumeAfter, resumeStarted: prepared.ResumeAfter == ""}
	walkErr := filepath.WalkDir(root, walker.visit)
	if walkErr != nil && !errors.Is(walkErr, errRepairEntryLimit) && !errors.Is(walkErr, context.DeadlineExceeded) && !errors.Is(walkErr, context.Canceled) {
		return result, walkErr
	}
	if walkErr != nil {
		result.Status = RepairPartial
	}
	if result.Failed > 0 && result.Status == RepairComplete {
		result.Status = RepairPartial
	}
	if result.Status == RepairComplete {
		result.LastPath = ""
	}
	result.Duration = time.Since(started)
	return result, nil
}

func (s RepairService) prepareRepairRequest(req RepairRequest) (RepairRequest, string, bool, error) {
	if req.FollowSymlinks {
		return req, "", false, errors.New("recursive ownership repair cannot follow symlinks")
	}
	if req.MaxEntries == 0 {
		req.MaxEntries = 1_000_000
	}
	if req.Deadline.IsZero() {
		req.Deadline = time.Now().Add(tuning.RepairDeadline())
	}
	if s.ResolveRoot == nil {
		s.ResolveRoot = resolveManagedRoot
	}
	canonical, err := s.ResolveRoot(req.Scope.RootClass)
	if err != nil {
		return req, "", false, err
	}
	root := canonical
	if strings.TrimSpace(req.Scope.RootPath) != "" {
		root = filepath.Clean(req.Scope.RootPath)
		if !withinRepairRoot(root, canonical) {
			return req, "", false, fmt.Errorf("repair scope path is outside the canonical managed root")
		}
	}
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return req, root, true, nil
	}
	if err != nil {
		return req, "", false, fmt.Errorf("stat managed root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return req, "", false, errors.New("managed root cannot be a symlink")
	}
	if req.ResumeAfter = strings.TrimSpace(req.ResumeAfter); req.ResumeAfter != "" {
		req.ResumeAfter = filepath.Clean(req.ResumeAfter)
		if !withinRepairRoot(req.ResumeAfter, root) {
			return req, "", false, fmt.Errorf("repair resume path is outside the canonical managed root")
		}
	}
	if err := validateRepairIdentity(req.ExpectedUID, req.ExpectedGID); err != nil {
		return req, "", false, err
	}
	return req, root, false, nil
}

type repairWalker struct {
	ctx           context.Context
	req           RepairRequest
	result        *RepairResult
	resumeAfter   string
	resumeStarted bool
}

func (w *repairWalker) visit(path string, entry os.DirEntry, walkErr error) error {
	if walkErr != nil {
		w.recordFailure(path, "access", "reported", walkErr.Error())
		return nil
	}
	if skip, err := w.skipProcessedPath(path, entry); skip {
		return err
	}
	if err := w.boundaryError(); err != nil {
		w.result.Status = RepairPartial
		return err
	}
	w.result.Scanned++
	w.result.LastPath = path
	if entry.Type()&os.ModeSymlink != 0 {
		w.result.Skipped++
		return nil
	}
	return w.repairOwnership(path)
}

func (w *repairWalker) skipProcessedPath(path string, entry os.DirEntry) (bool, error) {
	if w.resumeStarted {
		return false, nil
	}
	switch {
	case path == w.resumeAfter:
		if !entry.IsDir() {
			w.resumeStarted = true
		}
		return true, nil
	case withinRepairRoot(w.resumeAfter, path):
		return true, nil
	case path < w.resumeAfter:
		if entry.IsDir() {
			return true, filepath.SkipDir
		}
		return true, nil
	default:
		w.resumeStarted = true
		return false, nil
	}
}

func (w *repairWalker) boundaryError() error {
	if err := w.ctx.Err(); err != nil {
		return err
	}
	if time.Now().After(w.req.Deadline) {
		return context.DeadlineExceeded
	}
	if w.result.Scanned >= w.req.MaxEntries {
		return errRepairEntryLimit
	}
	return nil
}

func (w *repairWalker) repairOwnership(path string) error {
	identity, err := entryIdentity(path)
	if err != nil {
		w.recordFailure(path, "stat", "reported", err.Error())
		return nil
	}
	if identity.UID == w.req.ExpectedUID && identity.GID == w.req.ExpectedGID {
		w.result.Skipped++
		return nil
	}
	if !w.req.Apply {
		w.result.Skipped++
		w.result.Failures = append(w.result.Failures, RepairFailure{Path: path, Kind: "ownership_mismatch", Action: "planned", Reason: fmt.Sprintf("uid/gid %d/%d does not match %d/%d", identity.UID, identity.GID, w.req.ExpectedUID, w.req.ExpectedGID)})
		return nil
	}
	if err := repairEntryOwnership(path, w.req.ExpectedUID, w.req.ExpectedGID); err != nil {
		w.recordFailure(path, "ownership_mismatch", "failed", err.Error())
		return nil
	}
	w.result.Repaired++
	w.verifyOwnership(path)
	return nil
}

func (w *repairWalker) verifyOwnership(path string) {
	w.result.Verification.Checked++
	verified, err := entryIdentity(path)
	if err == nil && verified.UID == w.req.ExpectedUID && verified.GID == w.req.ExpectedGID {
		return
	}
	w.result.Verification.Failed++
	reason := "post-repair ownership verification failed"
	if err != nil {
		reason = err.Error()
	}
	w.recordFailure(path, "verification", "failed", reason)
}

func (w *repairWalker) recordFailure(path, kind, action, reason string) {
	w.result.Failed++
	w.result.Failures = append(w.result.Failures, RepairFailure{Path: path, Kind: kind, Action: action, Reason: reason})
}

func withinRepairRoot(path, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}

var errRepairEntryLimit = errors.New("repair entry limit reached")

type fileIdentity struct{ UID, GID uint32 }

func resolveManagedRoot(class string) (string, error) {
	home, err := VrooliHome()
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(class)
	if key == "artifacts" {
		key = repocontract.HomeKeyArtifacts
	}
	if key == "backups" {
		key = repocontract.HomeKeyBackups
	}
	switch key {
	case repocontract.HomeKeyBin, repocontract.HomeKeyCache, repocontract.HomeKeyLogs,
		repocontract.HomeKeyMetrics, repocontract.HomeKeyProcesses, repocontract.HomeKeyBuild,
		repocontract.HomeKeyTestRuns, repocontract.HomeKeyBackups, repocontract.HomeKeyArtifacts:
		return repocontract.RuntimeHomeEntryPath(homeDir(home), key)
	default:
		return "", fmt.Errorf("managed runtime-home class %q is not approved", class)
	}
}

func homeDir(runtimeHome string) string {
	return filepath.Dir(runtimeHome)
}
