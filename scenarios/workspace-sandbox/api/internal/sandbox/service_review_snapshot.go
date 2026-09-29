package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/types"
)

func normalizeReviewPaths(paths []string) ([]string, error) {
	if len(paths) == 0 || len(paths) > types.MaxReviewFiles {
		return nil, types.NewValidationError("paths", "explicit bounded review paths are required")
	}
	paths = slices.Clone(paths)
	for _, path := range paths {
		if !filepath.IsLocal(path) || filepath.Clean(path) != path || strings.Contains(path, "\\") {
			return nil, types.NewValidationError("paths", "review paths must be canonical scope-relative paths")
		}
		for _, part := range strings.Split(filepath.ToSlash(path), "/") {
			if part == ".git" {
				return nil, types.NewValidationError("paths", "Git metadata is not review source")
			}
		}
	}
	sort.Strings(paths)
	var result []string
	for _, path := range paths {
		if !reviewPathSelected(result, path) {
			result = append(result, path)
		}
	}
	return result, nil
}

func reviewPathSelected(paths []string, path string) bool {
	for _, selection := range paths {
		if selection == "." || path == selection || strings.HasPrefix(path, selection+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// scanReviewTree returns a sorted Git-representable tree without following leaf
// symlinks. The optional sink persists one bounded body at a time.
func scanReviewTree(ctx context.Context, rootPath string, paths []string, sink func([]byte) error) ([]types.ReviewFile, int64, error) {
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, 0, err
	}
	defer root.Close()
	files := make([]types.ReviewFile, 0)
	var total int64
	for _, selection := range paths {
		if _, err := root.Lstat(selection); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, 0, err
		}
		err := fs.WalkDir(root.FS(), filepath.ToSlash(selection), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.Name() == ".git" {
				if entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			if len(files) >= types.MaxReviewFiles {
				return errors.New("review file-count limit exceeded")
			}
			path = filepath.FromSlash(path)
			body, info, err := diff.ReadFileBytesLimit(rootPath, path, types.MaxReviewFileBytes)
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
				return fmt.Errorf("unsupported review file type at %q", path)
			}
			total += int64(len(body))
			if total > types.MaxReviewInputBytes {
				return errors.New("review byte limit exceeded")
			}
			if sink != nil {
				if err := sink(body); err != nil {
					return err
				}
			}
			files = append(files, types.ReviewFile{FileFingerprint: types.FileFingerprint{Path: path, SHA256: diff.HashPatch(string(body)), Mode: fingerprintMode(info.Mode())}, Size: int64(len(body))})
			return nil
		})
		if err != nil {
			return nil, 0, err
		}
	}
	slices.SortFunc(files, func(a, b types.ReviewFile) int { return strings.Compare(a.Path, b.Path) })
	return files, total, nil
}

func (s *Service) CaptureReviewSnapshot(ctx context.Context, req *types.ReviewSnapshotRequest) (*types.ReviewSnapshot, error) {
	if req == nil || req.SandboxID == uuid.Nil || req.RequestID == uuid.Nil {
		return nil, types.NewValidationError("request", "sandbox and review request UUIDs are required")
	}
	paths, err := normalizeReviewPaths(req.Paths)
	if err != nil {
		return nil, err
	}
	if s.archiveRepo == nil || s.blobs == nil {
		return nil, errors.New("review evidence storage is unavailable")
	}
	release, err := s.lockReview(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	existing, err := s.archiveRepo.GetReviewSnapshot(ctx, req.SandboxID, req.RequestID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if !slices.Equal(existing.Paths, paths) {
			return nil, types.NewValidationError("requestId", "review request already captured a different selection")
		}
		return existing, nil
	}
	if err := s.archiveRepo.CheckReviewCapacity(ctx); err != nil {
		return nil, err
	}
	sb, err := s.Get(ctx, req.SandboxID)
	if err != nil {
		return nil, err
	}
	if sb.Status != types.StatusStopped {
		return nil, types.NewValidationError("status", "stop the sandbox before retaining review input")
	}
	if err := s.requireNoPreparedApproval(ctx, sb.ID); err != nil {
		return nil, err
	}
	if err := s.requireUnpublishedArchive(ctx, sb.ID); err != nil {
		return nil, err
	}
	if err := s.ensureDiffDirectories(sb); err != nil {
		return nil, err
	}
	changes, err := s.driver.GetChangedFiles(ctx, sb)
	if err != nil {
		return nil, err
	}
	changes = filterDiffChanges(changes)
	if len(changes) > types.MaxReviewFiles {
		return nil, errors.New("review changed-file limit exceeded")
	}
	for _, change := range changes {
		if !filepath.IsLocal(change.FilePath) || filepath.Clean(change.FilePath) != change.FilePath || !reviewPathSelected(paths, change.FilePath) {
			return nil, types.NewValidationError("paths", "review selection must include every changed file")
		}
	}
	snapshot := &types.ReviewSnapshot{ID: types.ReviewSnapshotID(sb.ID, req.RequestID), RequestID: req.RequestID, SandboxID: sb.ID, CreatedAt: s.clock.Now().UTC(), ProjectRoot: sb.ProjectRoot, ScopePath: sb.ScopePath, Owner: sb.Owner, Paths: paths}
	published := false
	defer func() {
		if !published {
			// A lost SQL response may already have published. Unavailable reads
			// retain bytes; never clean the source sandbox's archive namespace.
			retained, err := s.archiveRepo.GetReviewSnapshot(ctx, sb.ID, req.RequestID)
			if err == nil && retained == nil {
				_ = s.blobs.DeleteSandbox(ctx, snapshot.ID.String())
			}
		}
	}()
	snapshot.Before, snapshot.InputBytes, err = scanReviewTree(ctx, sb.LowerDir, paths, func(body []byte) error {
		_, err := s.blobs.Put(ctx, snapshot.ID.String(), body)
		return err
	})
	if err != nil {
		return nil, err
	}
	captured, err := s.captureApprovalDiff(ctx, sb, changes, types.MaxReviewFileBytes, types.MaxReviewInputBytes)
	if err != nil {
		return nil, err
	}
	defer captured.close()
	snapshot.InputBytes += int64(len(captured.result.UnifiedDiff))
	for _, file := range captured.result.Files {
		snapshot.InputBytes += file.FileSize
	}
	if snapshot.InputBytes > types.MaxReviewInputBytes {
		return nil, errors.New("review byte limit exceeded")
	}
	before := make(map[string]types.ReviewFile, len(snapshot.Before))
	after := make(map[string]types.ReviewFile, len(snapshot.Before))
	for _, file := range snapshot.Before {
		before[file.Path], after[file.Path] = file, file
	}
	for _, change := range captured.result.Files {
		actual, err := fingerprintAt(captured.sandbox.LowerDir, change.FilePath)
		if err != nil {
			return nil, err
		}
		want := before[change.FilePath].FileFingerprint
		want.Path = change.FilePath
		if actual != want {
			return nil, errors.New("review baseline changed during capture")
		}
	}
	archive := newDiffArchive(sb, types.StatusApproved) // index builder only; never published as a terminal archive
	captured.sandbox.ID = snapshot.ID                   // private blob namespace; original sandbox state is unchanged
	if _, err := s.captureBlobs(ctx, &captured.sandbox, archive, captured.result); err != nil {
		return nil, err
	}
	snapshot.PatchSHA256, snapshot.Changes, snapshot.Stats = archive.UnifiedDiffSHA256, archive.Files, archive.Stats
	for _, file := range archive.Files {
		if file.ChangeType == types.ChangeTypeDeleted {
			delete(after, file.Path)
			continue
		}
		after[file.Path] = types.ReviewFile{FileFingerprint: types.FileFingerprint{Path: file.Path, SHA256: file.BlobSHA256, Mode: fingerprintMode(os.FileMode(file.FileMode))}, Size: file.Size}
	}
	if len(after) > types.MaxReviewFiles {
		return nil, errors.New("review candidate file-count limit exceeded")
	}
	snapshot.After = make([]types.ReviewFile, 0, len(after))
	for _, file := range after {
		snapshot.After = append(snapshot.After, file)
	}
	slices.SortFunc(snapshot.After, func(a, b types.ReviewFile) int { return strings.Compare(a.Path, b.Path) })
	current, _, err := scanReviewTree(ctx, sb.LowerDir, paths, nil)
	if err != nil {
		return nil, err
	}
	if !slices.Equal(current, snapshot.Before) {
		return nil, errors.New("review baseline changed during capture")
	}
	latest, err := s.driver.GetChangedFiles(ctx, sb)
	if err != nil {
		return nil, err
	}
	latest = filterDiffChanges(latest)
	if len(latest) != len(changes) {
		return nil, errors.New("review change set changed during capture")
	}
	wantChanges := make(map[string]types.ChangeType, len(changes))
	for _, change := range changes {
		wantChanges[change.FilePath] = change.ChangeType
	}
	for _, change := range latest {
		if wantChanges[change.FilePath] != change.ChangeType {
			return nil, errors.New("review change set changed during capture")
		}
		if change.ChangeType != types.ChangeTypeDeleted {
			current, err := fingerprintAtLimit(sb.UpperDir, change.FilePath, types.MaxReviewFileBytes)
			if err != nil {
				return nil, err
			}
			if current != after[change.FilePath].FileFingerprint {
				return nil, errors.New("review candidate changed during capture")
			}
		}
	}
	snapshot.SHA256 = snapshot.ContentSHA256()
	if err := s.archiveRepo.PutReviewSnapshot(ctx, snapshot); err != nil {
		return nil, err
	}
	published = true
	return snapshot, nil
}

func (s *Service) GetReviewSnapshot(ctx context.Context, sandboxID, requestID uuid.UUID) (*types.ReviewSnapshot, error) {
	if s.archiveRepo == nil {
		return nil, errors.New("review evidence storage is unavailable")
	}
	snapshot, err := s.archiveRepo.GetReviewSnapshot(ctx, sandboxID, requestID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, types.NewNotFoundError(requestID.String())
	}
	return snapshot, nil
}

func (s *Service) FetchReviewFile(ctx context.Context, sandboxID, requestID uuid.UUID, side, path string) ([]byte, error) {
	snapshot, err := s.GetReviewSnapshot(ctx, sandboxID, requestID)
	if err != nil {
		return nil, err
	}
	if s.blobs == nil {
		return nil, errors.New("review evidence storage is unavailable")
	}
	files := snapshot.Before
	switch side {
	case "patch":
		if path != "" {
			return nil, types.NewValidationError("path", "patch reads have no file path")
		}
		body, err := s.blobs.Get(ctx, snapshot.ID.String(), snapshot.PatchSHA256)
		if err != nil {
			return nil, fmt.Errorf("retained review patch unavailable: %v", err)
		}
		return body, nil
	case "before":
	case "after":
		files = snapshot.After
	default:
		return nil, types.NewValidationError("side", "review side must be before, after or patch")
	}
	for _, file := range files {
		if file.Path == path {
			body, err := s.blobs.Get(ctx, snapshot.ID.String(), file.SHA256)
			if err != nil {
				// This is lost retained evidence, not a nonexistent requested path.
				return nil, fmt.Errorf("retained review content unavailable: %v", err)
			}
			if int64(len(body)) != file.Size {
				return nil, errors.New("retained review file size is corrupt")
			}
			return body, nil
		}
	}
	return nil, types.NewNotFoundError(path)
}

func (s *Service) MaterializeReviewSnapshot(ctx context.Context, sandboxID, requestID uuid.UUID, expectedSHA256 string) (*types.ReviewWorkspace, error) {
	if s.blobs == nil {
		return nil, errors.New("review evidence storage is unavailable")
	}
	release, err := s.lockReview(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	snapshot, err := s.GetReviewSnapshot(ctx, sandboxID, requestID)
	if err != nil {
		return nil, err
	}
	if expectedSHA256 == "" || expectedSHA256 != snapshot.SHA256 {
		return nil, types.NewValidationError("expectedSha256", "retained review identity does not match")
	}
	root, err := s.blobs.MaterializeReview(ctx, snapshot)
	if err != nil {
		return nil, err
	}
	return &types.ReviewWorkspace{Root: root, SHA256: snapshot.SHA256}, nil
}

func (s *Service) checkReviewSource(ctx context.Context, sb *types.Sandbox, req *types.ApprovalRequest, applied bool) error {
	snapshot, err := s.GetReviewSnapshot(ctx, sb.ID, req.ReviewRequestID)
	if err != nil {
		return err
	}
	if snapshot.ScopePath != sb.ScopePath || snapshot.ProjectRoot != sb.ProjectRoot || snapshot.Owner != sb.Owner {
		return types.NewValidationError("review", "sandbox origin differs from retained review")
	}
	// Metadata alone is not available reviewer evidence. Refuse before effects
	// if any retained body has been lost or corrupted; read one body at a time.
	if s.blobs == nil {
		return errors.New("review evidence storage is unavailable")
	}
	seen := make(map[string]bool)
	for _, files := range [][]types.ReviewFile{snapshot.Before, snapshot.After, {{FileFingerprint: types.FileFingerprint{SHA256: snapshot.PatchSHA256}, Size: -1}}} {
		for _, file := range files {
			if seen[file.SHA256] {
				continue
			}
			body, err := s.blobs.Get(ctx, snapshot.ID.String(), file.SHA256)
			if err != nil {
				return fmt.Errorf("retained review evidence unavailable: %w", err)
			}
			if file.Size >= 0 && int64(len(body)) != file.Size {
				return errors.New("retained review file size is corrupt")
			}
			seen[file.SHA256] = true
		}
	}
	current, _, err := scanReviewTree(ctx, sb.ScopePath, snapshot.Paths, nil)
	if err != nil {
		return err
	}
	want := snapshot.Before
	if applied {
		want = snapshot.After
	}
	if !slices.Equal(current, want) {
		return types.NewValidationError("review", "canonical review input changed; recovery did not write source")
	}
	return nil
}
