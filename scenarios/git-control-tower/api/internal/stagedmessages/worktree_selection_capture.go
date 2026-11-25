package stagedmessages

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"git-control-tower/internal/reviewjobs"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

func queuedSnapshotDigest(snapshot *v1.StagedMessageSnapshot) (string, error) {
	type fileIdentity struct {
		Path, OldMode, NewMode, OldBlobID, NewBlobID, Status string
	}
	canonical := struct {
		RepositoryID, BaseHead string
		Files                  []fileIdentity
	}{RepositoryID: snapshot.GetRepositoryId(), BaseHead: snapshot.GetBaseHead()}
	for _, file := range snapshot.GetFiles() {
		canonical.Files = append(canonical.Files, fileIdentity{Path: file.GetPath(), OldMode: file.GetOldMode(), NewMode: file.GetNewMode(), OldBlobID: file.GetOldBlobId(), NewBlobID: file.GetNewBlobId(), Status: file.GetStatus()})
	}
	data, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (s *Service) QueueWorktreeSelection(ctx context.Context, repo, dir string, req *v1.QueueWorktreeSelectionRequest) (*v1.QueuedWorktreeSelection, error) {
	if req == nil || strings.TrimSpace(req.SelectionId) == "" || len(req.SelectionId) > 200 {
		return nil, fmt.Errorf("bounded selection id is required")
	}
	paths, err := normalizeWorktreePaths(req.Paths)
	if err != nil {
		return nil, err
	}
	var captured Captured
	switch req.Source {
	case v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_INDEX_TO_WORKTREE:
		captured, err = captureWorktreeSnapshot(ctx, repo, dir, paths, s.now)
	case v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_HEAD_TO_INDEX:
		captured, err = s.captureSelectedStaged(ctx, repo, dir, paths, true)
	default:
		return nil, fmt.Errorf("selection source is required")
	}
	if err != nil {
		return nil, err
	}
	if len(captured.Snapshot.Files) == 0 {
		return nil, fmt.Errorf("selected snapshot is empty")
	}
	captured.Snapshot.Digest, err = queuedSnapshotDigest(captured.Snapshot)
	if err != nil {
		return nil, err
	}
	var current Captured
	if req.Source == v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_HEAD_TO_INDEX {
		// The second pass only confirms the exact index identities. The first
		// capture already has the immutable bytes, so do not reread every blob.
		current, err = s.captureSelectedStaged(ctx, repo, dir, paths, false)
	} else {
		current, err = captureWorktreeSnapshot(ctx, repo, dir, paths, s.now)
	}
	if err != nil {
		return nil, err
	}
	currentDigest, err := queuedSnapshotDigest(current.Snapshot)
	if err != nil {
		return nil, err
	}
	if currentDigest != captured.Snapshot.Digest {
		return nil, fmt.Errorf("%w: selected content changed during queue capture", ErrConflict)
	}
	input := DraftInput{FormatVersion: 2, Snapshot: captured.Snapshot, WorktreeCapture: true, CapturedWorktree: captured.Bytes, QueuedSelection: true, QueuedSource: req.Source}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxDraftInputBytes {
		return nil, fmt.Errorf("immutable selection exceeds %d bytes", MaxDraftInputBytes)
	}
	state := reviewjobs.WorktreeQueueQueued
	if req.Source == v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_HEAD_TO_INDEX {
		state = reviewjobs.WorktreeQueueAwaitingUnstage
	}
	entry := reviewjobs.WorktreeQueueEntry{ID: req.SelectionId, RepositoryID: repo, SnapshotDigest: captured.Snapshot.Digest, Paths: paths, RequestKey: "selection:" + req.SelectionId, State: state, SelectionID: req.SelectionId, SelectionSource: int32(req.Source), SelectionPayload: string(raw)}
	queued, _, err := s.store.EnqueueCapturedSelection(ctx, entry)
	if err != nil {
		return nil, err
	}
	return queuedSelection(queued), nil
}

func (s *Service) captureSelectedStaged(ctx context.Context, repo, dir string, paths []string, captureContent bool) (Captured, error) {
	head, err := gitRead(ctx, dir, "rev-parse", "--verify", "HEAD")
	if err != nil {
		head = nil
	}
	baseHead := strings.TrimSpace(string(head))
	snap := &v1.StagedMessageSnapshot{RepositoryId: repo, BaseHead: baseHead}
	data := map[string]FileBytes{}
	total := 0
	args := []string{"diff", "--cached", "--raw", "--no-abbrev", "--no-renames", "--no-ext-diff", "--no-textconv", "-z"}
	if baseHead != "" {
		args = append(args, baseHead)
	}
	args = append(args, "--")
	args = append(args, paths...)
	raw, err := gitRead(ctx, dir, args...)
	if err != nil {
		return Captured{}, err
	}
	selected := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		selected[path] = struct{}{}
	}
	rows := bytes.Split(raw, []byte{0})
	for i := 0; i < len(rows) && len(rows[i]) > 0; i += 2 {
		if i+1 >= len(rows) {
			return Captured{}, fmt.Errorf("incomplete selected staged diff")
		}
		fields := strings.Fields(strings.TrimPrefix(string(rows[i]), ":"))
		path := string(rows[i+1])
		if len(fields) != 5 || !utf8.ValidString(path) {
			return Captured{}, fmt.Errorf("invalid selected staged diff entry")
		}
		if _, ok := selected[path]; !ok {
			return Captured{}, fmt.Errorf("git returned an unexpected selected staged path")
		}
		f := &v1.StagedMessageFile{Path: path, OldPath: path, OldMode: fields[0], NewMode: fields[1], OldBlobId: fields[2], NewBlobId: fields[3], Status: fields[4]}
		if zeroOID(f.OldBlobId) {
			f.OldBlobId = ""
			f.OldMode = ""
		}
		if zeroOID(f.NewBlobId) {
			f.NewBlobId = ""
			f.NewMode = ""
		}
		snap.Files = append(snap.Files, f)
		if f.NewBlobId == "" {
			continue // Staged deletion; it has no stage-0 index entry or new blob.
		}
	}
	if len(snap.Files) != len(paths) {
		return Captured{}, fmt.Errorf("selected paths are not all staged")
	}
	// Validate stage-0 index ownership in one Git process rather than one
	// process per path. This also rejects unresolved merge entries and catches
	// index changes between the diff and index reads.
	indexArgs := append([]string{"ls-files", "--stage", "-z", "--"}, paths...)
	indexRaw, err := gitRead(ctx, dir, indexArgs...)
	if err != nil {
		return Captured{}, err
	}
	indexEntries := make(map[string][]stagedIndexEntry, len(paths))
	for _, row := range bytes.Split(indexRaw, []byte{0}) {
		if len(row) == 0 {
			continue
		}
		meta, path, ok := strings.Cut(string(row), "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 || !utf8.ValidString(path) {
			return Captured{}, fmt.Errorf("invalid selected staged index entry")
		}
		if _, ok := selected[path]; !ok {
			return Captured{}, fmt.Errorf("git returned an unexpected staged index path")
		}
		indexEntries[path] = append(indexEntries[path], stagedIndexEntry{mode: fields[0], oid: fields[1], stage: fields[2]})
	}
	for _, file := range snap.Files {
		entries := indexEntries[file.Path]
		if file.NewBlobId == "" {
			if len(entries) != 0 {
				return Captured{}, fmt.Errorf("selected deletion %q changed in the index during capture", file.Path)
			}
			continue
		}
		if len(entries) != 1 || entries[0].stage != "0" || entries[0].oid != file.NewBlobId || entries[0].mode != file.NewMode {
			return Captured{}, fmt.Errorf("selected path %q has an unmerged or changed index entry", file.Path)
		}
	}
	if captureContent {
		var oids []string
		for _, file := range snap.Files {
			if file.OldBlobId != "" {
				oids = append(oids, file.OldBlobId)
			}
			if file.NewBlobId != "" {
				oids = append(oids, file.NewBlobId)
			}
		}
		blobs, sizes, readErr := readQueuedBlobs(ctx, dir, oids)
		if readErr != nil {
			return Captured{}, readErr
		}
		for _, file := range snap.Files {
			before, after := blobs[file.OldBlobId], blobs[file.NewBlobId]
			if total+sizes[file.OldBlobId]+sizes[file.NewBlobId] > maxQueuedCaptureBytes {
				return Captured{}, fmt.Errorf("selected staged content exceeds the %d MiB immutable queue capture limit", maxQueuedCaptureBytes>>20)
			}
			total += sizes[file.OldBlobId] + sizes[file.NewBlobId]
			file.Binary = bytes.IndexByte(after, 0) >= 0 || !utf8.Valid(after)
			data[file.Path] = FileBytes{Before: before, After: after}
		}
	}
	snap.CapturedAt = snapshotTime(s.now)
	snap.Digest, err = queuedSnapshotDigest(snap)
	if err != nil {
		return Captured{}, err
	}
	return Captured{Snapshot: snap, Bytes: data}, nil
}

type stagedIndexEntry struct{ mode, oid, stage string }

// maxQueuedCaptureBytes bounds exact staged content independently of the much
// smaller explanatory evidence budget used by suggestion snapshots. Keeping
// this below the JSON input ceiling leaves room for base64 and file metadata.
const maxQueuedCaptureBytes = 12 << 20

func readQueuedBlobs(ctx context.Context, dir string, oids []string) (map[string][]byte, map[string]int, error) {
	return readQueuedBlobsWithin(ctx, dir, oids, maxQueuedCaptureBytes)
}

func readQueuedBlobsWithin(ctx context.Context, dir string, oids []string, budget int) (map[string][]byte, map[string]int, error) {
	if len(oids) == 0 {
		return map[string][]byte{}, map[string]int{}, nil
	}
	batchInput, sizesByOID, err := inspectQueuedBlobInventory(ctx, dir, oids, budget)
	if err != nil {
		return nil, nil, err
	}
	output, err := gitReadInput(ctx, dir, batchInput, "cat-file", "--batch")
	if err != nil {
		return nil, nil, err
	}
	blobs, err := decodeQueuedBlobBatch(output, oids, sizesByOID)
	if err != nil {
		return nil, nil, err
	}
	return blobs, sizesByOID, nil
}

// inspectQueuedBlobInventory validates object identities and enforces the
// aggregate immutable-capture budget before Git returns any blob contents.
func inspectQueuedBlobInventory(ctx context.Context, dir string, oids []string, budget int) ([]byte, map[string]int, error) {
	input := make([]byte, 0, len(oids)*42)
	for _, oid := range oids {
		input = append(input, oid...)
		input = append(input, '\n')
	}
	checked, err := gitReadInput(ctx, dir, input, "cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)")
	if err != nil {
		return nil, nil, err
	}
	checks := bytes.Split(bytes.TrimSpace(checked), []byte{'\n'})
	if len(checks) != len(oids) {
		return nil, nil, fmt.Errorf("git returned an incomplete staged blob inventory")
	}
	sizesByOID := make(map[string]int, len(oids))
	var batchInput []byte
	total := 0
	for index, row := range checks {
		size, sizeErr := queuedBlobInventorySize(row, oids[index], budget-total)
		if sizeErr != nil {
			return nil, nil, sizeErr
		}
		total += size
		sizesByOID[oids[index]] = size
		batchInput = append(batchInput, oids[index]...)
		batchInput = append(batchInput, '\n')
	}
	return batchInput, sizesByOID, nil
}

func queuedBlobInventorySize(row []byte, expectedOID string, remainingBytes int) (int, error) {
	fields := strings.Fields(string(row))
	if len(fields) != 3 || fields[0] != expectedOID || fields[1] != "blob" {
		return 0, fmt.Errorf("selected staged content contains an unsupported Git object")
	}
	size, err := strconv.Atoi(fields[2])
	if err != nil || size < 0 || size > remainingBytes {
		return 0, fmt.Errorf("selected staged content exceeds the %d MiB immutable queue capture limit", maxQueuedCaptureBytes>>20)
	}
	return size, nil
}

// decodeQueuedBlobBatch verifies every response header, body length, separator,
// and trailing byte before returning immutable captured contents.
func decodeQueuedBlobBatch(output []byte, oids []string, sizesByOID map[string]int) (map[string][]byte, error) {
	reader := bufio.NewReader(bytes.NewReader(output))
	blobs := make(map[string][]byte, len(oids))
	for _, oid := range oids {
		blob, err := readQueuedBlob(reader, oid, sizesByOID[oid])
		if err != nil {
			return nil, err
		}
		blobs[oid] = blob
	}
	if _, err := reader.ReadByte(); err != io.EOF {
		return nil, fmt.Errorf("staged blob capture contains trailing data")
	}
	return blobs, nil
}

func readQueuedBlob(reader *bufio.Reader, expectedOID string, expectedSize int) ([]byte, error) {
	header, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("incomplete staged blob capture")
	}
	fields := strings.Fields(header)
	if len(fields) != 3 || fields[0] != expectedOID || fields[1] != "blob" || fields[2] != strconv.Itoa(expectedSize) {
		return nil, fmt.Errorf("staged blob identity changed during capture")
	}
	blob := make([]byte, expectedSize)
	if _, err = io.ReadFull(reader, blob); err != nil {
		return nil, fmt.Errorf("incomplete staged blob content")
	}
	separator, err := reader.ReadByte()
	if err != nil || separator != '\n' {
		return nil, fmt.Errorf("invalid staged blob framing")
	}
	return blob, nil
}

func queuedSelection(e reviewjobs.WorktreeQueueEntry) *v1.QueuedWorktreeSelection {
	return &v1.QueuedWorktreeSelection{SelectionId: e.SelectionID, RepositoryId: e.RepositoryID, Source: v1.WorktreeSelectionSource(e.SelectionSource), Paths: append([]string(nil), e.Paths...), SnapshotDigest: e.SnapshotDigest, State: queuedSelectionState(e.State), FailureReason: e.FailureReason}
}

func queuedSelectionState(state reviewjobs.WorktreeQueueState) v1.QueuedWorktreeSelectionState {
	if state == reviewjobs.WorktreeQueueAwaitingUnstage {
		return v1.QueuedWorktreeSelectionState_QUEUED_WORKTREE_SELECTION_STATE_AWAITING_UNSTAGE
	}
	if state == reviewjobs.WorktreeQueueFailed {
		return v1.QueuedWorktreeSelectionState_QUEUED_WORKTREE_SELECTION_STATE_FAILED
	}
	if state == reviewjobs.WorktreeQueueStale {
		return v1.QueuedWorktreeSelectionState_QUEUED_WORKTREE_SELECTION_STATE_STALE
	}
	return v1.QueuedWorktreeSelectionState_QUEUED_WORKTREE_SELECTION_STATE_QUEUED
}
