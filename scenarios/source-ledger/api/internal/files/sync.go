package files

// Read-time sync: bring the index in line with the files before any read.

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"path"
	"path/filepath"
	"time"
)

// Sync brings the index for corpus (or one topic) in line with the files.
// budget <= 0 means no limit. It returns the corpus paths left unsynced.
func (s *Service) Sync(ctx context.Context, corpus, topic string, budget time.Duration) (SyncReport, map[string]bool, error) {
	corpus, root, err := s.root(corpus)
	if err != nil {
		return SyncReport{}, nil, err
	}
	if err := validTopic(topic); err != nil {
		return SyncReport{}, nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	started := time.Now()
	report := SyncReport{At: started.UTC()}
	walked := Walk(root, topic)
	entries := walked.Entries
	report.Problems, report.Stats = walked.Problems, walked.Stats
	report.Seen = len(entries)
	prefix := ""
	if topic != "" {
		prefix = path.Join("topics", topic) + "/"
	}
	have, err := s.store.indexedUnder(ctx, corpus, prefix)
	if err != nil {
		return report, nil, err
	}
	changed := statChanged(entries, have)
	// Whatever is left in `have` no longer exists (or is now excluded).
	if err := s.store.removeAll(ctx, corpus, have); err != nil {
		return report, nil, err
	}
	report.Removed = len(have)
	pending, err := s.applyWithin(ctx, corpus, root, changed, &report, started, budget)
	if err != nil {
		return report, nil, err
	}
	report.Pending = len(pending)
	report.Partial = len(pending) > 0
	report.Duration = time.Since(started)
	return report, pending, nil
}

// statChanged returns the walked entries whose size, mtime or ctime differ
// from the index, removing every walked path from have.
func statChanged(entries []Entry, have map[string]indexed) []Entry {
	var changed []Entry
	for _, e := range entries {
		old, ok := have[e.Path]
		delete(have, e.Path)
		if !ok || old.Size != e.Size || old.MtimeNS != e.MtimeNS || old.ChangedNS != e.ChangedNS {
			changed = append(changed, e)
		}
	}
	return changed
}

// applyWithin reindexes changed files in batches until budget (<= 0: none)
// runs out, counting them in report, and returns the paths left pending.
func (s *Service) applyWithin(ctx context.Context, corpus, root string, changed []Entry, report *SyncReport, started time.Time, budget time.Duration) (map[string]bool, error) {
	pending := map[string]bool{}
	for i := 0; i < len(changed); i += syncBatch {
		if budget > 0 && time.Since(started) > budget {
			for _, e := range changed[i:] {
				pending[e.Path] = true
			}
			break
		}
		n, err := s.apply(ctx, corpus, root, changed[i:min(i+syncBatch, len(changed))])
		report.Changed += n
		if err != nil {
			return nil, err
		}
	}
	return pending, nil
}

// apply reindexes a batch in one transaction. Content-identical files only
// get their stat refreshed.
func (s *Service) apply(ctx context.Context, corpus, root string, batch []Entry) (int, error) {
	paths := make([]string, len(batch))
	for i, e := range batch {
		paths[i] = e.Path
	}
	tx, err := s.store.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	have := map[string]string{}
	for _, p := range paths {
		var h string
		if err := tx.QueryRowContext(ctx, `SELECT hash FROM files WHERE corpus = ? AND path = ?`, corpus, p).Scan(&h); err == nil {
			have[p] = h
		}
	}
	changed := 0
	for _, e := range batch {
		did, err := s.reindex(ctx, tx, corpus, root, e, have[e.Path])
		if err != nil {
			return changed, err
		}
		if did {
			changed++
		}
	}
	return changed, tx.Commit()
}

// reindex rereads one file inside tx: a vanished or unreadable file is
// dropped, identical content only refreshes its stat, anything else is
// rechunked. It reports whether the content changed.
func (s *Service) reindex(ctx context.Context, tx *sql.Tx, corpus, root string, e Entry, oldHash string) (bool, error) {
	content, firstLine, err := readCapped(filepath.Join(root, filepath.FromSlash(e.Path)), e.Kind, e.Size, e.MaxBytes)
	if err != nil {
		// Vanished or unreadable between walk and read: drop it.
		return false, s.store.remove(ctx, tx, corpus, e.Path)
	}
	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:])
	if oldHash == hash {
		return false, s.store.touch(ctx, tx, corpus, e)
	}
	return true, s.store.replace(ctx, tx, corpus, e, hash, chunkFile(e.Kind, content, firstLine))
}
