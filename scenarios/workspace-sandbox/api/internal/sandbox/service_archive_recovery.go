package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/types"

	"github.com/google/uuid"
)

func (s *Service) lockReview(ctx context.Context) (func(), error) {
	if s.blobs == nil {
		return func() {}, nil
	}
	return s.blobs.LockReview(ctx)
}

func (s *Service) requireNoPreparedApproval(ctx context.Context, id uuid.UUID) error {
	if s.archiveRepo == nil {
		return nil
	}
	intent, err := s.archiveRepo.GetPreparedApproval(ctx, id)
	if err != nil {
		return err
	}
	if intent != nil {
		return types.NewValidationErrorWithHint("approval", "sandbox has a pending prepared approval", "Retry the original approval to reconcile retained evidence before other mutations")
	}
	return nil
}

func (s *Service) lockUnprepared(ctx context.Context, id uuid.UUID) (func(), error) {
	release, err := s.lockReview(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.requireNoPreparedApproval(ctx, id); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

func fingerprintMode(mode os.FileMode) int {
	if mode&os.ModeSymlink != 0 {
		return 120000
	}
	if mode.Perm()&0111 != 0 {
		return 100755
	}
	return 100644
}

func fingerprintAt(root, path string) (types.FileFingerprint, error) {
	return fingerprintAtLimit(root, path, 0)
}

func fingerprintAtLimit(root, path string, limit int64) (types.FileFingerprint, error) {
	if !filepath.IsLocal(path) || path == "." || filepath.Clean(path) != path {
		return types.FileFingerprint{}, fmt.Errorf("invalid approval path %q", path)
	}
	result := types.FileFingerprint{Path: path}
	content, info, err := diff.ReadFileBytesLimit(root, path, limit)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
		return result, fmt.Errorf("unsupported source type at %q", path)
	}
	result.Mode = fingerprintMode(info.Mode())
	result.SHA256 = fmt.Sprintf("%x", sha256.Sum256(content))
	return result, nil
}

func matchFingerprints(root string, want []types.FileFingerprint) (bool, error) {
	for _, expected := range want {
		actual, err := fingerprintAt(root, expected.Path)
		if err != nil {
			return false, err
		}
		if actual != expected {
			return false, nil
		}
	}
	return true, nil
}

func (s *Service) prepareApproval(ctx context.Context, sb *types.Sandbox, req *types.ApprovalRequest, captured *capturedApprovalDiff, archive *types.DiffArchive) (*types.PreparedApproval, error) {
	archive.SnapshotAt = s.clock.Now().UTC()
	intent := &types.PreparedApproval{Archive: *archive, Request: *req, ScopePath: sb.ScopePath}
	for _, entry := range archive.Files {
		before, err := fingerprintAt(captured.sandbox.LowerDir, entry.Path)
		if err != nil {
			return nil, err
		}
		intent.Before = append(intent.Before, before)
	}
	matches, err := matchFingerprints(sb.ScopePath, intent.Before)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, types.NewValidationError("approval", "canonical source no longer matches the captured baseline; no source was written")
	}
	if err := s.archiveRepo.PutPreparedApproval(ctx, intent); err != nil {
		return nil, fmt.Errorf("persist approval intent before apply: %w", err)
	}
	return intent, nil
}

// recoverPreparedApproval never re-diffs a mutable overlay. It either publishes
// already-present effects or applies the original retained patch against its
// exact before state. Mixed/divergent state is not permission to overwrite.
func (s *Service) recoverPreparedApproval(ctx context.Context, sb *types.Sandbox, req *types.ApprovalRequest) (*selectedApplyResult, error) {
	if s.archiveRepo == nil || s.blobs == nil {
		return nil, nil
	}
	intent, err := s.archiveRepo.GetPreparedApproval(ctx, sb.ID)
	if err != nil || intent == nil {
		return nil, err
	}
	archive := &intent.Archive
	if intent.ScopePath != sb.ScopePath || archive.ProjectRoot != sb.ProjectRoot || archive.Owner != sb.Owner || archive.ArchiveState != types.ArchiveStateComplete || archive.SandboxStatus != types.StatusApproved || intent.Request.CreateCommit || len(archive.Files) == 0 || len(intent.Before) != len(archive.Files) {
		return nil, errors.New("prepared approval does not match its sandbox or capture contract")
	}
	if err := checkReviewedPatch(req.ExpectedPatchSHA256, archive.UnifiedDiffSHA256); err != nil {
		return nil, err
	}
	original, retry := intent.Request, *req
	original.ExpectedPatchSHA256, retry.ExpectedPatchSHA256 = "", ""
	originalJSON, err := json.Marshal(original)
	if err != nil {
		return nil, fmt.Errorf("invalid retained request: %w", err)
	}
	retryJSON, err := json.Marshal(retry)
	if err != nil || string(originalJSON) != string(retryJSON) {
		return nil, types.NewValidationError("approval", "retry differs from the original prepared approval request")
	}
	patch, err := s.blobs.Get(ctx, sb.ID.String(), archive.UnifiedDiffSHA256)
	if err != nil {
		return nil, fmt.Errorf("read prepared patch: %w", err)
	}
	var after []types.FileFingerprint
	var changes []*types.FileChange
	seen := map[string]bool{}
	for i, entry := range archive.Files {
		if seen[entry.Path] || intent.Before[i].Path != entry.Path {
			return nil, errors.New("prepared approval has duplicate or misaligned file identities")
		}
		seen[entry.Path] = true
		body, err := s.blobs.Get(ctx, sb.ID.String(), entry.BlobSHA256)
		if err != nil || int64(len(body)) != entry.Size {
			return nil, fmt.Errorf("prepared file %q is missing or invalid: %v", entry.Path, err)
		}
		expected := types.FileFingerprint{Path: entry.Path}
		switch entry.ChangeType {
		case types.ChangeTypeAdded, types.ChangeTypeModified:
			expected.SHA256, expected.Mode = entry.BlobSHA256, fingerprintMode(os.FileMode(entry.FileMode))
		case types.ChangeTypeDeleted:
		default:
			return nil, fmt.Errorf("unsupported prepared change type %q", entry.ChangeType)
		}
		after = append(after, expected)
		changes = append(changes, &types.FileChange{SandboxID: sb.ID, FilePath: entry.Path, ChangeType: entry.ChangeType, FileSize: entry.Size, FileMode: entry.FileMode})
	}
	matchesAfter, err := matchFingerprints(sb.ScopePath, after)
	if err != nil {
		return nil, err
	}
	if matchesAfter {
		return &selectedApplyResult{Success: true, Recovered: true, Changes: changes, TotalChanges: len(changes), PatchSHA256: archive.UnifiedDiffSHA256, AppliedAt: archive.SnapshotAt, PreparedArchive: archive, PreparedApproval: intent}, nil
	}
	matchesBefore, err := matchFingerprints(sb.ScopePath, intent.Before)
	if err != nil {
		return nil, err
	}
	if !matchesBefore {
		return nil, types.NewValidationError("approval", "canonical source is neither the retained before nor after state; recovery did not write source")
	}
	result, err := s.applyGeneratedChanges(ctx, sb, &intent.Request, string(patch), changes, nil, len(changes), archive)
	if result != nil {
		result.PreparedApproval = intent
	}
	return result, err
}

func (s *Service) recordPreparedApprovalProvenance(ctx context.Context, sb *types.Sandbox, intent *types.PreparedApproval) error {
	revision := "sandbox-approval:" + sb.ID.String() + ":" + intent.Archive.UnifiedDiffSHA256
	req := intent.Request
	runID := req.AgentManagerRunID
	if runID == "" {
		runID = intent.Archive.AgentManagerRunID
	}
	changes := make([]*types.AppliedChange, 0, len(intent.Archive.Files))
	for _, file := range intent.Archive.Files {
		digest := ""
		if file.ChangeType != types.ChangeTypeDeleted {
			digest = "sha256:" + file.BlobSHA256
		}
		changes = append(changes, &types.AppliedChange{
			ID:        uuid.NewSHA1(uuid.NameSpaceURL, []byte(revision+":"+file.Path)),
			SandboxID: sb.ID, SandboxOwner: sb.Owner, SandboxOwnerType: string(sb.OwnerType),
			FilePath: filepath.Join(intent.ScopePath, file.Path), ProjectRoot: sb.ProjectRoot,
			ChangeType: string(file.ChangeType), FileSize: file.Size, ContentDigest: digest,
			EvidenceRevision: revision, AppliedAt: intent.Archive.SnapshotAt,
			AgentManagerRunID: runID, ConversationID: req.ConversationID,
			CostUSD: req.Cost, RunOutcome: req.RunOutcome,
			ProvenanceState: string(types.ProvenanceFileStateApplied),
		})
	}
	return s.repo.RecordAppliedChanges(ctx, changes)
}

// recoverArchivedTurnProvenance reconciles effects that reached the working
// tree before provenance failed and lifecycle deletion archived the sandbox.
// It never applies archived bytes, resurrects the sandbox, or invents origin.
// Every file must still match the archive, and every record commits together.
func (s *Service) recoverArchivedTurnProvenance(ctx context.Context, sb *types.Sandbox, req *types.TurnCheckpointRequest) (*types.TurnCheckpointResult, error) {
	if s.archiveRepo == nil || s.blobs == nil {
		return nil, fmt.Errorf("finalization recovery requires a retained archive")
	}
	archive, err := s.archiveRepo.Get(ctx, sb.ID)
	if err != nil {
		return nil, err
	}
	if archive == nil || archive.ArchiveState != types.ArchiveStateComplete || archive.SandboxStatus != types.StatusDeleted {
		return nil, fmt.Errorf("finalization recovery requires a complete deleted-sandbox archive")
	}
	if req.AgentManagerRunID != archive.AgentManagerRunID || req.AgentManagerRunID != metadataString(sb.Metadata, metadataAgentManagerRunID) {
		return nil, fmt.Errorf("finalization recovery run does not match the retained archive origin")
	}
	rootPath := sb.ScopePath
	if rootPath == "" {
		rootPath = sb.ProjectRoot
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, fmt.Errorf("open recovery scope: %w", err)
	}
	defer root.Close()
	revision := fmt.Sprintf("sandbox-archive:%s:%s", sb.ID, archive.SnapshotAt.UTC().Format(time.RFC3339Nano))
	changes := make([]*types.AppliedChange, 0, len(archive.Files))
	var total int64
	seen := map[string]bool{}
	for _, entry := range archive.Files {
		path := filepath.Clean(entry.Path)
		if !filepath.IsLocal(path) || path == "." || path == ".git" || strings.HasPrefix(path, ".git/") || seen[path] {
			return nil, fmt.Errorf("invalid or duplicate archive path %q", entry.Path)
		}
		seen[path] = true
		change := &types.FileChange{FilePath: path, ChangeType: entry.ChangeType, FileSize: entry.Size}
		if evaluateAcceptance(sb, change).Status != types.AcceptanceStatusAccepted {
			return nil, fmt.Errorf("archive path %q requires manual review", path)
		}
		digest := ""
		if entry.ChangeType == types.ChangeTypeDeleted {
			if _, err := root.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("deleted archive path %q is present or unreadable", path)
			}
		} else {
			if entry.ChangeType != types.ChangeTypeAdded && entry.ChangeType != types.ChangeTypeModified {
				return nil, fmt.Errorf("unsupported archive change for %q", path)
			}
			content, err := s.blobs.Get(ctx, sb.ID.String(), entry.BlobSHA256)
			if err != nil {
				return nil, fmt.Errorf("read retained archive file %q: %w", path, err)
			}
			sum := sha256.Sum256(content)
			if fmt.Sprintf("%x", sum) != entry.BlobSHA256 || int64(len(content)) != entry.Size {
				return nil, fmt.Errorf("retained archive file %q does not match its digest or size", path)
			}
			info, err := root.Lstat(path)
			if err != nil || !info.Mode().IsRegular() {
				return nil, fmt.Errorf("archive path %q is absent or not a regular file", path)
			}
			file, err := root.Open(path)
			if err != nil {
				return nil, fmt.Errorf("read current archive path %q: %w", path, err)
			}
			hash := sha256.New()
			n, readErr := io.Copy(hash, file)
			closeErr := file.Close()
			if readErr != nil || closeErr != nil || n != entry.Size || fmt.Sprintf("%x", hash.Sum(nil)) != entry.BlobSHA256 {
				return nil, fmt.Errorf("archive path %q changed since finalization; recovery did not write source", path)
			}
			digest = "sha256:" + entry.BlobSHA256
			total += entry.Size
		}
		changes = append(changes, &types.AppliedChange{
			ID:        uuid.NewSHA1(uuid.NameSpaceURL, []byte(revision+":"+path)),
			SandboxID: sb.ID, SandboxOwner: sb.Owner, SandboxOwnerType: string(sb.OwnerType),
			FilePath: filepath.Join(rootPath, path), ProjectRoot: sb.ProjectRoot,
			ChangeType: string(entry.ChangeType), FileSize: entry.Size, AppliedAt: archive.SnapshotAt,
			ContentDigest: digest, EvidenceRevision: revision, AgentManagerRunID: req.AgentManagerRunID,
			ConversationID: req.ConversationID, CostUSD: req.Cost, RunOutcome: req.RunOutcome,
			ProvenanceState: string(types.ProvenanceFileStateApplied),
		})
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for _, change := range changes {
		prior, err := tx.GetFileProvenance(ctx, change.FilePath, sb.ProjectRoot, int(^uint32(0)>>1))
		if err != nil {
			return nil, err
		}
		found := false
		for _, row := range prior {
			// A partial checkpoint may have persisted this exact file before
			// another file failed. Preserve that original record, including its
			// timestamp and commit link, instead of attributing the effect twice.
			if row.SandboxID == change.SandboxID && row.AgentManagerRunID == change.AgentManagerRunID && row.ContentDigest == change.ContentDigest && row.ChangeType == change.ChangeType && row.ProvenanceState == change.ProvenanceState {
				found = true
			}
			if row.ID != change.ID {
				continue
			}
			if row.SandboxID != change.SandboxID || row.AgentManagerRunID != change.AgentManagerRunID || row.ContentDigest != change.ContentDigest || row.EvidenceRevision != change.EvidenceRevision || row.ChangeType != change.ChangeType {
				return nil, fmt.Errorf("conflicting recovered provenance for %q", change.FilePath)
			}
			found = true
		}
		if !found {
			if err := tx.RecordAppliedChanges(ctx, []*types.AppliedChange{change}); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.logAuditEvent(ctx, sb, "turn_provenance_recovered", req.Actor, "agent", map[string]interface{}{"agentManagerRunId": req.AgentManagerRunID, "evidenceRevision": revision, "applied": len(changes)})
	return &types.TurnCheckpointResult{SandboxID: sb.ID, Status: sb.Status, Success: true, Applied: len(changes), AppliedAt: archive.SnapshotAt, AppliedSizeBytes: total, CheckpointID: revision, DiffPath: fmt.Sprintf("/api/v1/sandboxes/%s/diff", sb.ID)}, nil
}
