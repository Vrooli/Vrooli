package transcriptindex

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
)

// A file key is the device and inode the filesystem gave a transcript, and a
// filesystem reuses an inode once its file is deleted. Codex deletes each
// rollout when it compresses it, so the compressed rollout of one session can
// get the inode an older session's rollout had. Such a file is not a move or
// a rewrite of the checkpoint holding its key: the checkpoint's file is gone,
// and the new file records another session. The checkpoint moves to a
// retired key, keeping every byte it read, and the new file is indexed as
// itself (joining its own lineage).

// keyReused reports a candidate holding the device and inode of a checkpoint
// whose file is gone and which recorded another session.
func keyReused(row *FileRow, candidate Candidate) bool {
	if row.Owner != OwnerIndexer || row.Path == candidate.Path || row.SessionID == "" {
		return false
	}
	if _, err := os.Lstat(row.Path); !errors.Is(err, fs.ErrNotExist) {
		return false // the old path still resolves (a hard link) or cannot be checked
	}
	identity := ReadIdentity(candidate.Path, candidate.Root.Harness)
	return identity.SessionID != "" && identity.SessionID != row.SessionID
}

// reusedKeyLock is the refusal a replacement gave a reused key before reuse
// was recognised: the staged file named another session.
const reusedKeyLock = "replacement session identity changed"

// retireKey moves row off its file key. A checkpoint locked only because its
// key's new file named another session is restored to what it had read.
func (ix *Indexer) retireKey(ctx context.Context, t *tick, row *FileRow) error {
	owner, ok := ix.store.(interface {
		RetireTranscriptKey(ctx context.Context, expected, retired FileRow) error
	})
	if !ok {
		return errors.New("transcript key retirement owner unavailable")
	}
	retired := *row
	retired.Key = row.Key + "~" + strconv.FormatInt(row.FirstSeenAt.UnixNano(), 10)
	retired.Active = false
	if retired.Status == StatusUnreadable && retired.Error == sourceReconcilePrefix+reusedKeyLock {
		retired.Status, retired.Error = StatusIndexed, ""
	}
	if err := owner.RetireTranscriptKey(ctx, *row, retired); err != nil {
		return fmt.Errorf("retire reused file key: %w", err)
	}
	delete(t.byKey, row.Key)
	delete(ix.tails, row.Key)
	t.byKey[retired.Key] = &retired
	if t.byPath[row.Path] == row {
		t.byPath[row.Path] = &retired
	}
	return nil
}
