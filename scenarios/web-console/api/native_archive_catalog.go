package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/cli-core/agenttranscript"
	"web-console/internal/dbx"
)

// NativeArchiveRoots are explicit native transcript roots. Keeping the roots
// explicit lets a caller scope indexing to an authorized home or an isolated
// test tree; discovery never silently scans the current user's home.
type NativeArchiveRoots map[string][]string

// Clone copies every root slice so a running indexer owns an immutable snapshot.
func (r NativeArchiveRoots) Clone() NativeArchiveRoots {
	clone := make(NativeArchiveRoots, len(r))
	for harness, paths := range r {
		clone[harness] = append([]string(nil), paths...)
	}
	return clone
}

// Paths returns a copy of all home roots configured for one harness.
func (r NativeArchiveRoots) Paths(harness string) []string {
	return append([]string(nil), r[harness]...)
}

// FirstPath returns the primary home root for one harness, when present.
func (r NativeArchiveRoots) FirstPath(harness string) string {
	paths := r[harness]
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

// NativeArchiveRootsFromHome returns the conventional local archive roots
// below home without reading them. The caller decides whether and when to
// discover those paths.
func NativeArchiveRootsFromHome(home string) NativeArchiveRoots {
	roots := make(NativeArchiveRoots)
	for _, format := range agenttranscript.Formats() {
		roots[format.Harness] = agenttranscript.HomeRootPaths(home, format.Harness)
	}
	if strings.TrimSpace(home) != "" {
		dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
		if dataHome == "" {
			dataHome = filepath.Join(home, ".local", "share")
		}
		databasePath := filepath.Join(dataHome, "opencode", "opencode.db")
		roots[openCodeDatabaseRoot] = []string{databasePath}
		cacheHome := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME"))
		if cacheHome == "" {
			cacheHome = filepath.Join(home, ".cache")
		}
		roots[openCodeSnapshotRoot] = []string{openCodeSnapshotCachePath(cacheHome, databasePath)}
	}
	return roots
}

// DiscoverNativeArchiveFiles catalogs supported append-only/native archive
// files beneath caller-selected roots. Files are ordered deterministically;
// symlinks and non-regular files are not followed. Each archive receives a
// stable local session identity derived from its harness and native session.
func DiscoverNativeArchiveFiles(ctx context.Context, roots NativeArchiveRoots) ([]NativeArchiveFile, error) {
	prepared, err := prepareOpenCodeNativeRoots(ctx, roots)
	if err != nil {
		return nil, err
	}
	roots = prepared
	var files []NativeArchiveFile
	appendWalk := func(root, harness string) error {
		if strings.TrimSpace(root) == "" {
			return nil
		}
		root = filepath.Clean(root)
		rootHash := sha256.Sum256([]byte(root))
		err := agenttranscript.WalkNativeFiles(ctx, root, agenttranscript.NativeWalkOptions{
			Errors: agenttranscript.NativeWalkSkipMissingRoot,
		}, func(path string, _ fs.DirEntry, info os.FileInfo) error {
			nativeID, ok := agenttranscript.NativeArchiveSessionID(harness, path, info)
			if !ok || strings.TrimSpace(nativeID) == "" {
				return nil
			}
			statSignature := nativeArchiveStatSignature(info)
			format, _ := agenttranscript.FormatFor(harness)
			if format.ArchiveSnapshot {
				cursorSignature, err := cursorArchiveStatSignature(path)
				if err != nil {
					return fmt.Errorf("fingerprint Cursor database %s: %w", path, err)
				}
				statSignature = cursorSignature
			}
			nativeID = strings.TrimSpace(nativeID)
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			catalogKey := "native-archive/" + harness + "/" + hex.EncodeToString(rootHash[:8]) + "/" + filepath.ToSlash(rel)
			files = append(files, NativeArchiveFile{
				Harness: harness, Path: path, SourceKey: catalogKey, CatalogKey: catalogKey,
				WebConsoleSessionID: "wc-archive-" + harness + "-" + nativeID,
				NativeSessionID:     nativeID, Size: info.Size(),
				StatSignature: statSignature,
			})
			return nil
		})
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, format := range agenttranscript.Formats() {
		for _, root := range roots.Paths(format.Harness) {
			if err := appendWalk(root, format.Harness); err != nil {
				return nil, fmt.Errorf("discover %s archives: %w", format.Harness, err)
			}
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Harness == files[j].Harness {
			return files[i].Path < files[j].Path
		}
		return files[i].Harness < files[j].Harness
	})
	return files, nil
}

func nativeArchiveStatSignature(info os.FileInfo) string {
	system := reflect.ValueOf(info.Sys())
	if system.IsValid() && system.Kind() == reflect.Pointer && !system.IsNil() {
		system = system.Elem()
	}
	dev, ino, ctime := "", "", ""
	if system.IsValid() && system.Kind() == reflect.Struct {
		dev = archiveStatInteger(system, "Dev")
		ino = archiveStatInteger(system, "Ino")
		for _, name := range []string{"Ctim", "Ctimespec", "Ctime"} {
			field := system.FieldByName(name)
			if field.IsValid() {
				sec, nsec := archiveStatInteger(field, "Sec"), archiveStatInteger(field, "Nsec")
				if sec != "" || nsec != "" {
					ctime = sec + ":" + nsec
					break
				}
			}
		}
	}
	return fmt.Sprintf("%d:%d:%s:%s:%s", info.Size(), info.ModTime().UnixNano(), dev, ino, ctime)
}

func archiveStatInteger(value reflect.Value, fieldName string) string {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}
	if value.Kind() == reflect.Struct {
		value = value.FieldByName(fieldName)
	}
	if !value.IsValid() {
		return ""
	}
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", value.Uint())
	default:
		return ""
	}
}

type nativeArchiveCatalogRecord struct {
	CatalogKey          string
	Path                string
	Harness             string
	WebConsoleSessionID string
	NativeSessionID     string
	StatSignature       string
	ContentDigest       string
	Generation          string
	Size                int64
	UpdatedAt           time.Time
}

type nativeArchiveCatalogStore interface {
	load(context.Context) ([]nativeArchiveCatalogRecord, error)
	replace(context.Context, []nativeArchiveCatalogRecord) error
}

type sqlNativeArchiveCatalogStore struct{ db dbx.Handle }

func newSQLNativeArchiveCatalogStore(db dbx.Handle) nativeArchiveCatalogStore {
	return &sqlNativeArchiveCatalogStore{db: db}
}

func (s *sqlNativeArchiveCatalogStore) ensure(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS web_console_native_archive_catalog (
		catalog_key TEXT PRIMARY KEY,
		path TEXT NOT NULL,
		harness TEXT NOT NULL,
		web_console_session_id TEXT NOT NULL,
		native_session_id TEXT NOT NULL,
		stat_signature TEXT NOT NULL,
		content_digest TEXT NOT NULL,
		generation TEXT NOT NULL,
		size INTEGER NOT NULL,
		updated_at TEXT NOT NULL
	)`)
	return err
}

func (s *sqlNativeArchiveCatalogStore) load(ctx context.Context) ([]nativeArchiveCatalogRecord, error) {
	if err := s.ensure(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT catalog_key,path,harness,web_console_session_id,native_session_id,stat_signature,content_digest,generation,size,updated_at FROM web_console_native_archive_catalog`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []nativeArchiveCatalogRecord
	for rows.Next() {
		var record nativeArchiveCatalogRecord
		var updated string
		if err := rows.Scan(&record.CatalogKey, &record.Path, &record.Harness, &record.WebConsoleSessionID, &record.NativeSessionID, &record.StatSignature, &record.ContentDigest, &record.Generation, &record.Size, &updated); err != nil {
			return nil, err
		}
		record.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *sqlNativeArchiveCatalogStore) replace(ctx context.Context, records []nativeArchiveCatalogRecord) error {
	if err := s.ensure(ctx); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.ExecContext(ctx, `DELETE FROM web_console_native_archive_catalog`); err != nil {
		return err
	}
	for _, record := range records {
		if _, err := tx.ExecContext(ctx, `INSERT INTO web_console_native_archive_catalog(catalog_key,path,harness,web_console_session_id,native_session_id,stat_signature,content_digest,generation,size,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			record.CatalogKey, record.Path, record.Harness, record.WebConsoleSessionID, record.NativeSessionID,
			record.StatSignature, record.ContentDigest, record.Generation, record.Size, record.UpdatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type memoryNativeArchiveCatalogStore struct {
	mu      sync.Mutex
	records map[string]nativeArchiveCatalogRecord
}

func newMemoryNativeArchiveCatalogStore() nativeArchiveCatalogStore {
	return &memoryNativeArchiveCatalogStore{records: make(map[string]nativeArchiveCatalogRecord)}
}

func (s *memoryNativeArchiveCatalogStore) load(context.Context) ([]nativeArchiveCatalogRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]nativeArchiveCatalogRecord, 0, len(s.records))
	for _, record := range s.records {
		out = append(out, record)
	}
	return out, nil
}

func (s *memoryNativeArchiveCatalogStore) replace(_ context.Context, records []nativeArchiveCatalogRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = make(map[string]nativeArchiveCatalogRecord, len(records))
	for _, record := range records {
		s.records[record.CatalogKey] = record
	}
	return nil
}

type nativeArchiveCatalog struct {
	mu    sync.Mutex
	store nativeArchiveCatalogStore
}

func newMemoryNativeArchiveCatalog() *nativeArchiveCatalog {
	return &nativeArchiveCatalog{store: newMemoryNativeArchiveCatalogStore()}
}

func newSQLNativeArchiveCatalog(db dbx.Handle) *nativeArchiveCatalog {
	return &nativeArchiveCatalog{store: newSQLNativeArchiveCatalogStore(db)}
}

// Reconcile retains a source generation for append-only growth, and creates a
// new session generation for rewrites, truncations, or explicit removals.
// Prefix verification makes same-size and larger replacement detectable.
func (c *nativeArchiveCatalog) Reconcile(ctx context.Context, files []NativeArchiveFile, reset func(string) error) ([]NativeArchiveFile, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	oldRows, err := c.store.load(ctx)
	if err != nil {
		return nil, err
	}
	old := make(map[string]nativeArchiveCatalogRecord, len(oldRows))
	bySessionOld := make(map[string][]nativeArchiveCatalogRecord)
	for _, record := range oldRows {
		old[record.CatalogKey] = record
		bySessionOld[record.WebConsoleSessionID] = append(bySessionOld[record.WebConsoleSessionID], record)
	}
	next := make(map[string]nativeArchiveCatalogRecord, len(files))
	bySessionNext := make(map[string][]string)
	resetSessions := make(map[string]struct{})
	for _, file := range files {
		key := file.CatalogKey
		if key == "" {
			key = file.SourceKey
		}
		if key == "" {
			return nil, fmt.Errorf("archive %s has no stable catalog key", file.Path)
		}
		record := nativeArchiveCatalogRecord{
			CatalogKey: key, Path: file.Path, Harness: file.Harness, WebConsoleSessionID: file.WebConsoleSessionID,
			NativeSessionID: file.NativeSessionID, StatSignature: file.StatSignature, Size: file.Size,
		}
		previous, existed := old[key]
		if existed && previous.StatSignature == file.StatSignature && previous.Size == file.Size {
			record.ContentDigest = previous.ContentDigest
			record.Generation = previous.Generation
			record.UpdatedAt = previous.UpdatedAt
		} else {
			digest, err := digestNativeFile(file.Path, file.Size)
			if err != nil {
				return nil, fmt.Errorf("fingerprint native archive %s: %w", file.Path, err)
			}
			record.ContentDigest = digest
			record.UpdatedAt = time.Now().UTC()
			if existed {
				appendOnly := file.Harness != "cursor" && file.Size >= previous.Size
				if appendOnly {
					prefixDigest, err := digestNativeFile(file.Path, previous.Size)
					if err != nil {
						return nil, fmt.Errorf("verify native archive prefix %s: %w", file.Path, err)
					}
					appendOnly = prefixDigest == previous.ContentDigest
				}
				if appendOnly {
					record.Generation = previous.Generation
				} else {
					resetSessions[file.WebConsoleSessionID] = struct{}{}
				}
			}
		}
		next[key] = record
		bySessionNext[file.WebConsoleSessionID] = append(bySessionNext[file.WebConsoleSessionID], key)
	}
	for key, previous := range old {
		if _, exists := next[key]; !exists {
			resetSessions[previous.WebConsoleSessionID] = struct{}{}
		}
	}
	for sessionID := range resetSessions {
		if err := reset(sessionID); err != nil {
			return nil, fmt.Errorf("reset archive projection %s: %w", sessionID, err)
		}
	}
	for sessionID, keys := range bySessionNext {
		generation := ""
		for _, previous := range bySessionOld[sessionID] {
			if previous.Generation != "" {
				generation = previous.Generation
				break
			}
		}
		if _, reset := resetSessions[sessionID]; reset || generation == "" {
			parts := make([]string, 0, len(keys)+1)
			parts = append(parts, generation)
			for _, key := range keys {
				parts = append(parts, key+"="+next[key].ContentDigest)
			}
			sort.Strings(parts)
			generation = agenttranscript.SHA256Bytes([]byte(strings.Join(parts, "\n")))
		}
		for _, key := range keys {
			record := next[key]
			previous, existed := old[key]
			if !existed || previous.Generation != generation {
				record.UpdatedAt = time.Now().UTC()
			}
			record.Generation = generation
			next[key] = record
		}
	}
	ordered := make([]nativeArchiveCatalogRecord, 0, len(next))
	for _, record := range next {
		ordered = append(ordered, record)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].CatalogKey < ordered[j].CatalogKey })
	if !nativeArchiveCatalogRecordsEqual(oldRows, ordered) {
		if err := c.store.replace(ctx, ordered); err != nil {
			return nil, fmt.Errorf("persist native archive catalog: %w", err)
		}
	}
	filesByKey := make(map[string]NativeArchiveFile, len(files))
	for _, file := range files {
		key := file.CatalogKey
		if key == "" {
			key = file.SourceKey
		}
		filesByKey[key] = file
	}
	resolved := make([]NativeArchiveFile, 0, len(ordered))
	for _, record := range ordered {
		file := filesByKey[record.CatalogKey]
		file.CatalogKey = record.CatalogKey
		file.Generation = record.Generation
		file.ContentDigest = record.ContentDigest
		file.SourceKey = record.CatalogKey + "#generation=" + record.Generation
		resolved = append(resolved, file)
	}
	return resolved, nil
}

func nativeArchiveCatalogRecordsEqual(old, next []nativeArchiveCatalogRecord) bool {
	if len(old) != len(next) {
		return false
	}
	previous := make(map[string]nativeArchiveCatalogRecord, len(old))
	for _, record := range old {
		previous[record.CatalogKey] = record
	}
	for _, record := range next {
		prior, ok := previous[record.CatalogKey]
		if !ok || prior.Path != record.Path || prior.Harness != record.Harness || prior.WebConsoleSessionID != record.WebConsoleSessionID || prior.NativeSessionID != record.NativeSessionID || prior.StatSignature != record.StatSignature || prior.ContentDigest != record.ContentDigest || prior.Generation != record.Generation || prior.Size != record.Size {
			return false
		}
	}
	return true
}

func digestNativeFile(path string, limit int64) (string, error) {
	return agenttranscript.SHA256FilePrefixExact(path, limit)
}
