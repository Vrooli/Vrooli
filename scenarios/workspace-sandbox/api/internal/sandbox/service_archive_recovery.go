package sandbox

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

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
