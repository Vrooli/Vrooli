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

//nolint:gocyclo // ordered repair phases encode distinct ownership, rollback, and verification outcomes.
func (s RepairService) Repair(ctx context.Context, req RepairRequest) (RepairResult, error) {
	started := time.Now()
	result := RepairResult{Scope: req.Scope, Status: RepairComplete}
	if req.FollowSymlinks {
		return result, errors.New("recursive ownership repair cannot follow symlinks")
	}
	if req.MaxEntries == 0 {
		req.MaxEntries = 1_000_000
	}
	if req.Deadline.IsZero() {
		req.Deadline = time.Now().Add(tuning.RepairDeadline)
	}
	if s.ResolveRoot == nil {
		s.ResolveRoot = resolveManagedRoot
	}
	canonical, err := s.ResolveRoot(req.Scope.RootClass)
	if err != nil {
		return result, err
	}
	root := canonical
	if strings.TrimSpace(req.Scope.RootPath) != "" {
		root = filepath.Clean(req.Scope.RootPath)
		if !withinRepairRoot(root, canonical) {
			return result, fmt.Errorf("repair scope path is outside the canonical managed root")
		}
	}
	info, err := os.Lstat(root)
	if err != nil {
		if os.IsNotExist(err) {
			result.Duration = time.Since(started)
			return result, nil
		}
		return result, fmt.Errorf("stat managed root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return result, errors.New("managed root cannot be a symlink")
	}
	resumeAfter := strings.TrimSpace(req.ResumeAfter)
	if resumeAfter != "" {
		resumeAfter = filepath.Clean(resumeAfter)
		if !withinRepairRoot(resumeAfter, root) {
			return result, fmt.Errorf("repair resume path is outside the canonical managed root")
		}
	}
	if err := validateRepairIdentity(req.ExpectedUID, req.ExpectedGID); err != nil {
		return result, err
	}

	resumeStarted := resumeAfter == ""
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.Failed++
			result.Failures = append(result.Failures, RepairFailure{Path: path, Kind: "access", Action: "reported", Reason: walkErr.Error()})
			return nil
		}
		if !resumeStarted {
			switch {
			case path == resumeAfter:
				// The continuation point was already processed. Descend when it
				// is a directory so its children can become the next batch.
				if entry.IsDir() {
					return nil
				}
				resumeStarted = true
				return nil
			case withinRepairRoot(resumeAfter, path):
				// The cursor is below this directory. Keep descending without
				// counting the already-processed ancestor again.
				return nil
			case path < resumeAfter:
				// WalkDir is lexical. This whole subtree precedes the cursor.
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			default:
				resumeStarted = true
			}
		}
		if err := ctx.Err(); err != nil {
			result.Status = RepairPartial
			return err
		}
		if time.Now().After(req.Deadline) {
			result.Status = RepairPartial
			return context.DeadlineExceeded
		}
		if result.Scanned >= req.MaxEntries {
			result.Status = RepairPartial
			return errRepairEntryLimit
		}
		result.Scanned++
		result.LastPath = path
		if entry.Type()&os.ModeSymlink != 0 {
			result.Skipped++
			return nil
		}
		identity, identityErr := entryIdentity(path)
		if identityErr != nil {
			result.Failed++
			result.Failures = append(result.Failures, RepairFailure{Path: path, Kind: "stat", Action: "reported", Reason: identityErr.Error()})
			return nil
		}
		if identity.UID == req.ExpectedUID && identity.GID == req.ExpectedGID {
			result.Skipped++
			return nil
		}
		if !req.Apply {
			result.Skipped++
			result.Failures = append(result.Failures, RepairFailure{Path: path, Kind: "ownership_mismatch", Action: "planned", Reason: fmt.Sprintf("uid/gid %d/%d does not match %d/%d", identity.UID, identity.GID, req.ExpectedUID, req.ExpectedGID)})
			return nil
		}
		if err := repairEntryOwnership(path, req.ExpectedUID, req.ExpectedGID); err != nil {
			result.Failed++
			result.Failures = append(result.Failures, RepairFailure{Path: path, Kind: "ownership_mismatch", Action: "failed", Reason: err.Error()})
			return nil
		}
		result.Repaired++
		result.Verification.Checked++
		verified, verifyErr := entryIdentity(path)
		if verifyErr != nil || verified.UID != req.ExpectedUID || verified.GID != req.ExpectedGID {
			result.Failed++
			result.Verification.Failed++
			reason := "post-repair ownership verification failed"
			if verifyErr != nil {
				reason = verifyErr.Error()
			}
			result.Failures = append(result.Failures, RepairFailure{Path: path, Kind: "verification", Action: "failed", Reason: reason})
		}
		return nil
	})
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
