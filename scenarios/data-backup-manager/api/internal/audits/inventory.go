package audits

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// sqliteMagic is the 16-byte header every SQLite database file starts with
// ("SQLite format 3\000"). Detecting by content rather than extension keeps the
// walker generic — it finds a SQLite file regardless of how a scenario named it.
var sqliteMagic = []byte("SQLite format 3\x00")

// sqliteCandidate is a SQLite file discovered during a walk: its absolute path
// (for the read-only DB checker) and its tree-relative path (for the report).
type sqliteCandidate struct {
	Abs string
	Rel string
}

// walkOptions controls which generic signals a walk computes.
type walkOptions struct {
	// includeContentHash streams every regular file's bytes into the tree
	// content hash. Skipped for huge trees where the path-list hash + counts
	// are proof enough.
	includeContentHash bool
	// detectSQLite peeks each regular file's magic header to find SQLite DBs.
	detectSQLite bool
}

// walkResult is the raw output of a walk: the generic summary (without the
// SQLite facts, which the DB checker fills in) plus the discovered SQLite
// candidates.
type walkResult struct {
	summary InventorySummary
	sqlite  []sqliteCandidate
}

// walkTree computes the generic inventory of the tree rooted at dir. It walks in
// deterministic lexical order (filepath.WalkDir sorts entries), so the path-list
// hash is independent of filesystem traversal order. Symlinks are recorded by
// their target and never followed — a restored or captured tree can contain
// links to directories, and following them would escape the artifact entirely.
// Permission/IO errors on individual entries are recorded in UnreadablePaths
// rather than aborting the whole walk, so a single bad file still yields a
// usable (if flagged) proof.
func walkTree(dir string, opts walkOptions, now time.Time) (walkResult, error) {
	pathHash := sha256.New()
	contentHash := sha256.New()
	var (
		summary    InventorySummary
		candidates []sqliteCandidate
	)
	summary.CapturedAt = now

	root := filepath.Clean(dir)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// Record the unreadable entry and keep going (skip a directory we
			// cannot read rather than abort the audit).
			rel := relPath(root, path)
			summary.UnreadablePaths = append(summary.UnreadablePaths, rel)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		rel := relPath(root, path)
		if rel == "." {
			// The tree root itself is not counted as a directory entry.
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			summary.UnreadablePaths = append(summary.UnreadablePaths, rel)
			return nil
		}
		if mt := info.ModTime(); mt.After(summary.MaxModTime) {
			summary.MaxModTime = mt
		}

		switch {
		case d.IsDir():
			summary.Directories++
			fmt.Fprintf(pathHash, "d\t%s\n", rel)
		case d.Type()&fs.ModeSymlink != 0:
			summary.Symlinks++
			target, linkErr := os.Readlink(path)
			if linkErr != nil {
				summary.UnreadablePaths = append(summary.UnreadablePaths, rel)
				target = ""
			}
			fmt.Fprintf(pathHash, "l\t%s\t%s\n", rel, target)
			if opts.includeContentHash {
				fmt.Fprintf(contentHash, "%s\nsymlink:%s\n", rel, target)
			}
		case d.Type().IsRegular():
			summary.Files++
			summary.RegularBytes += info.Size()
			fmt.Fprintf(pathHash, "f\t%s\t%d\n", rel, info.Size())
			if err := hashRegularFile(path, rel, opts, contentHash, &summary, &candidates); err != nil {
				// Unreadable file: flagged, not fatal.
				summary.UnreadablePaths = append(summary.UnreadablePaths, rel)
			}
		default:
			summary.Other++
			fmt.Fprintf(pathHash, "o\t%s\n", rel)
		}
		return nil
	})
	if err != nil {
		return walkResult{}, fmt.Errorf("walk %q: %w", dir, err)
	}

	summary.PathListSHA256 = fmt.Sprintf("%x", pathHash.Sum(nil))
	if opts.includeContentHash {
		summary.TreeContentSHA = fmt.Sprintf("%x", contentHash.Sum(nil))
	}
	sort.Strings(summary.UnreadablePaths)
	return walkResult{summary: summary, sqlite: candidates}, nil
}

// hashRegularFile streams a regular file into the content hash (when enabled)
// and peeks its magic header to detect a SQLite database (when enabled). It
// opens the file at most once for both jobs.
func hashRegularFile(path, rel string, opts walkOptions, contentHash io.Writer, summary *InventorySummary, candidates *[]sqliteCandidate) error {
	if !opts.includeContentHash && !opts.detectSQLite {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if opts.detectSQLite {
		header := make([]byte, len(sqliteMagic))
		n, _ := io.ReadFull(f, header)
		if n == len(sqliteMagic) && string(header) == string(sqliteMagic) {
			*candidates = append(*candidates, sqliteCandidate{Abs: path, Rel: rel})
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}

	if opts.includeContentHash {
		fmt.Fprintf(contentHash, "%s\n", rel)
		if _, err := io.Copy(contentHash, f); err != nil {
			return err
		}
	}
	return nil
}

// relPath returns path relative to root, or path itself if it cannot be made
// relative (never panics).
func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
