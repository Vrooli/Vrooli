package sqlite

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/resources/sqlite/internal/config"
	"golang.org/x/crypto/pbkdf2"
	_ "modernc.org/sqlite"
)

var namePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// Service provides resource operations backed by SQLite.
type Service struct {
	Config config.Config
}

type DBInfo struct {
	Name     string
	Path     string
	Size     int64
	Modified time.Time
}

type StatusInfo struct {
	Version       string
	DatabasePath  string
	DatabaseCount int
	Databases     []DBInfo
}

type QueryResult struct {
	Columns []string
	Rows    [][]string
}

type GetResult struct {
	Info   map[string]string
	Result *QueryResult
}

type BatchResult struct {
	Duration time.Duration
}

func NewService(cfg config.Config) *Service {
	return &Service{Config: cfg}
}

func (s *Service) ValidateName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return errors.New("name is required")
	case strings.Contains(name, "/"), strings.Contains(name, `\`), strings.Contains(name, ".."):
		return errors.New("name cannot contain path separators or '..'")
	case !namePattern.MatchString(name):
		return errors.New("name may only include letters, numbers, dots, underscores, and dashes")
	default:
		return nil
	}
}

func (s *Service) normalizeName(name string) (string, error) {
	if err := s.ValidateName(name); err != nil {
		return "", err
	}
	if strings.HasSuffix(name, ".db") || strings.HasSuffix(name, ".sqlite") || strings.HasSuffix(name, ".sqlite3") {
		return name, nil
	}
	return name + ".db", nil
}

func (s *Service) databasePath(name string) (string, error) {
	normalized, err := s.normalizeName(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.Config.DatabasePath, normalized), nil
}

func (s *Service) openDatabase(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// Apply connection pragmas
	if err := s.applyPragmas(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (s *Service) applyPragmas(ctx context.Context, db *sql.DB) error {
	pragmas := []string{
		fmt.Sprintf("PRAGMA journal_mode = %s;", s.Config.JournalMode),
		fmt.Sprintf("PRAGMA busy_timeout = %d;", int(s.Config.BusyTimeout/time.Millisecond)),
		fmt.Sprintf("PRAGMA cache_size = %d;", s.Config.CacheSize),
		fmt.Sprintf("PRAGMA page_size = %d;", s.Config.PageSize),
		fmt.Sprintf("PRAGMA synchronous = %s;", s.Config.Synchronous),
		fmt.Sprintf("PRAGMA temp_store = %s;", s.Config.TempStore),
		fmt.Sprintf("PRAGMA mmap_size = %d;", s.Config.MmapSize),
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) CreateDatabase(ctx context.Context, name string) (string, error) {
	if err := s.Config.EnsureDirectories(); err != nil {
		return "", err
	}
	path, err := s.databasePath(name)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("database already exists: %s", filepath.Base(path))
	}
	db, err := s.openDatabase(ctx, path)
	if err != nil {
		return "", err
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, "PRAGMA user_version;"); err != nil {
		return "", err
	}
	if err := os.Chmod(path, s.Config.FilePermissions); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) Execute(ctx context.Context, name, query string) (*QueryResult, error) {
	path, err := s.databasePath(name)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("database not found: %s", filepath.Base(path))
	}

	db, err := s.openDatabase(ctx, path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	kind := strings.ToUpper(firstWord(query))
	switch kind {
	case "SELECT", "PRAGMA", "WITH", "EXPLAIN":
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		var outRows [][]string
		for rows.Next() {
			values := make([]sql.NullString, len(cols))
			ptrs := make([]any, len(cols))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return nil, err
			}
			row := make([]string, len(cols))
			for i, v := range values {
				if v.Valid {
					row[i] = v.String
				} else {
					row[i] = "NULL"
				}
			}
			outRows = append(outRows, row)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return &QueryResult{Columns: cols, Rows: outRows}, nil
	default:
		_, err := db.ExecContext(ctx, query)
		return nil, err
	}
}

func (s *Service) ListDatabases() ([]DBInfo, error) {
	if err := s.Config.EnsureDirectories(); err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(s.Config.DatabasePath)
	if err != nil {
		return nil, err
	}
	var dbs []DBInfo
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !hasDBExtension(name) {
			continue
		}
		full := filepath.Join(s.Config.DatabasePath, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}
		dbs = append(dbs, DBInfo{
			Name:     name,
			Path:     full,
			Size:     info.Size(),
			Modified: info.ModTime(),
		})
	}
	sort.Slice(dbs, func(i, j int) bool { return dbs[i].Name < dbs[j].Name })
	return dbs, nil
}

func (s *Service) RemoveDatabase(name string, force bool) error {
	path, err := s.databasePath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("database not found: %s", filepath.Base(path))
	}
	if !force {
		return errors.New("refusing to remove without --force")
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	_ = os.Remove(path + "-wal")
	_ = os.Remove(path + "-shm")
	return nil
}

func (s *Service) BackupDatabase(ctx context.Context, name string) (string, error) {
	if err := s.Config.EnsureDirectories(); err != nil {
		return "", err
	}
	path, err := s.databasePath(name)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("database not found: %s", filepath.Base(path))
	}
	db, err := s.openDatabase(ctx, path)
	if err != nil {
		return "", err
	}
	defer db.Close()

	ts := time.Now().Format("20060102_150405")
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	backupName := fmt.Sprintf("%s_%s.db", base, ts)
	backupPath := filepath.Join(s.Config.BackupPath, backupName)

	if _, err := db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s';", backupPath)); err != nil {
		return "", err
	}
	if err := os.Chmod(backupPath, s.Config.FilePermissions); err != nil {
		return "", err
	}
	if s.Config.BackupRetentionDays > 0 {
		_ = s.pruneBackups(base, time.Now())
	}
	return backupPath, nil
}

func (s *Service) RestoreDatabase(ctx context.Context, name, backupFile string, force bool) (string, error) {
	if err := s.Config.EnsureDirectories(); err != nil {
		return "", err
	}
	destPath, err := s.databasePath(name)
	if err != nil {
		return "", err
	}
	sourcePath := backupFile
	if !filepath.IsAbs(sourcePath) {
		candidate := filepath.Join(s.Config.BackupPath, backupFile)
		if _, err := os.Stat(candidate); err == nil {
			sourcePath = candidate
		}
	}
	if _, err := os.Stat(sourcePath); err != nil {
		return "", fmt.Errorf("backup not found: %s", backupFile)
	}
	if _, err := os.Stat(destPath); err == nil && !force {
		return "", errors.New("destination exists; use --force to overwrite")
	}
	if err := copyFile(sourcePath, destPath, s.Config.FilePermissions); err != nil {
		return "", err
	}
	// Touch with a quick pragma to ensure readability post-copy.
	db, err := s.openDatabase(ctx, destPath)
	if err == nil {
		_, _ = db.ExecContext(ctx, "PRAGMA integrity_check;")
		_ = db.Close()
	}
	return destPath, nil
}

func (s *Service) GetDatabaseInfo(ctx context.Context, name, query string) (*GetResult, error) {
	path, err := s.databasePath(name)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("database not found: %s", filepath.Base(path))
	}
	db, err := s.openDatabase(ctx, path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if strings.TrimSpace(query) != "" {
		res, err := s.Execute(ctx, name, query)
		if err != nil {
			return nil, err
		}
		return &GetResult{Result: res}, nil
	}

	// Info mode
	stat, _ := os.Stat(path)
	pageCount := querySingleInt(ctx, db, "PRAGMA page_count;")
	pageSize := querySingleInt(ctx, db, "PRAGMA page_size;")
	journalMode := querySingleText(ctx, db, "PRAGMA journal_mode;")

	info := map[string]string{
		"name":         filepath.Base(path),
		"path":         path,
		"size_bytes":   fmt.Sprintf("%d", stat.Size()),
		"pages":        fmt.Sprintf("%d", pageCount),
		"page_size":    fmt.Sprintf("%d", pageSize),
		"journal_mode": journalMode,
		"modified_at":  stat.ModTime().Format(time.RFC3339),
	}
	return &GetResult{Info: info}, nil
}

// Batch executes SQL inside a transaction from a reader.
func (s *Service) Batch(ctx context.Context, name string, reader io.Reader) (*BatchResult, error) {
	path, err := s.databasePath(name)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("database not found: %s", filepath.Base(path))
	}
	db, err := s.openDatabase(ctx, path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	sqlBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &BatchResult{Duration: time.Since(start)}, nil
}

// ImportCSV streams CSV rows into an existing table.
func (s *Service) ImportCSV(ctx context.Context, dbName, table, csvPath string, hasHeader bool, columnOverride []string) (int, error) {
	if err := s.ValidateName(table); err != nil {
		return 0, err
	}
	f, err := os.Open(csvPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	dbPath, err := s.databasePath(dbName)
	if err != nil {
		return 0, err
	}
	db, err := s.openDatabase(ctx, dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	reader := csv.NewReader(f)
	inserted := 0
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	tableColumns, err := s.tableColumns(ctx, db, table)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	insertColumns := tableColumns
	if len(columnOverride) > 0 {
		insertColumns = normalizeColumns(columnOverride)
		if err := s.validateColumnsExist(insertColumns, tableColumns); err != nil {
			_ = tx.Rollback()
			return 0, err
		}
	}

	if hasHeader {
		header, err := reader.Read()
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		normalized := normalizeColumns(header)
		if len(columnOverride) == 0 {
			if err := s.validateColumnsExist(normalized, tableColumns); err != nil {
				_ = tx.Rollback()
				return 0, err
			}
			insertColumns = normalized
		}
	}

	expectedCols := len(insertColumns)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		if len(record) != expectedCols {
			_ = tx.Rollback()
			return 0, fmt.Errorf("row %d has %d columns; expected %d", inserted+1, len(record), expectedCols)
		}
		placeholders := strings.TrimRight(strings.Repeat("?,", expectedCols), ",")
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(insertColumns, ","), placeholders)
		args := make([]any, expectedCols)
		for i, v := range record {
			args[i] = v
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			_ = tx.Rollback()
			return inserted, err
		}
		inserted++
	}
	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}

// ExportCSV dumps a table to CSV (with header).
func (s *Service) ExportCSV(ctx context.Context, dbName, table, outputPath string) (int, error) {
	if err := s.ValidateName(table); err != nil {
		return 0, err
	}
	dbPath, err := s.databasePath(dbName)
	if err != nil {
		return 0, err
	}
	db, err := s.openDatabase(ctx, dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s", table))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}
	outFile := outputPath
	if outFile == "" {
		base := strings.TrimSuffix(filepath.Base(dbPath), filepath.Ext(dbPath))
		outFile = filepath.Join(s.Config.DatabasePath, fmt.Sprintf("%s_%s_export_%d.csv", base, table, time.Now().Unix()))
	}
	f, err := os.Create(outFile)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()
	if err := writer.Write(cols); err != nil {
		return 0, err
	}

	count := 0
	vals := make([]sql.NullString, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return count, err
		}
		record := make([]string, len(cols))
		for i, v := range vals {
			if v.Valid {
				record[i] = v.String
			} else {
				record[i] = ""
			}
		}
		if err := writer.Write(record); err != nil {
			return count, err
		}
		count++
	}
	return count, rows.Err()
}

// EncryptDatabase encrypts database bytes using AES-GCM with PBKDF2 key derivation.
func (s *Service) EncryptDatabase(dbName, password string) error {
	path, err := s.databasePath(dbName)
	if err != nil {
		return err
	}
	if s.IsEncrypted(path) {
		return errors.New("database appears already encrypted")
	}
	plain, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_ = removeJournalFiles(path)
	cipher, err := encryptBytes(plain, password)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, cipher, s.Config.FilePermissions); err != nil {
		return err
	}
	_ = removeJournalFiles(path)
	return nil
}

func (s *Service) DecryptDatabase(dbName, password string) error {
	path, err := s.databasePath(dbName)
	if err != nil {
		return err
	}
	cipher, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	plain, err := decryptBytes(cipher, password)
	if err != nil {
		return err
	}
	_ = removeJournalFiles(path)
	if err := os.WriteFile(path, plain, s.Config.FilePermissions); err != nil {
		return err
	}
	_ = removeJournalFiles(path)
	return nil
}

func (s *Service) IsEncrypted(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	headerLen := len(sqliteHeader)
	if len(encMagic) > headerLen {
		headerLen = len(encMagic)
	}
	header := make([]byte, headerLen)
	n, err := io.ReadFull(f, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return false
	}
	header = header[:n]

	if len(header) >= len(sqliteHeader) && string(header[:len(sqliteHeader)]) == sqliteHeader {
		return false
	}
	if len(header) >= len(encMagic) && string(header[:len(encMagic)]) == encMagic {
		return true
	}
	return false
}

func (s *Service) Status(ctx context.Context) (*StatusInfo, error) {
	if err := s.Config.EnsureDirectories(); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	version := querySingleText(ctx, db, "select sqlite_version();")
	dbs, err := s.ListDatabases()
	if err != nil {
		return nil, err
	}
	return &StatusInfo{
		Version:       version,
		DatabasePath:  s.Config.DatabasePath,
		DatabaseCount: len(dbs),
		Databases:     dbs,
	}, nil
}

func querySingleInt(ctx context.Context, db *sql.DB, query string) int64 {
	var n int64
	_ = db.QueryRowContext(ctx, query).Scan(&n)
	return n
}

func querySingleText(ctx context.Context, db *sql.DB, query string) string {
	var s string
	_ = db.QueryRowContext(ctx, query).Scan(&s)
	return s
}

func querySingleTextArgs(ctx context.Context, db *sql.DB, query string, args ...any) string {
	var s string
	_ = db.QueryRowContext(ctx, query, args...).Scan(&s)
	return s
}

func hasDBExtension(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".db") || strings.HasSuffix(lower, ".sqlite") || strings.HasSuffix(lower, ".sqlite3")
}

func firstWord(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func copyFile(src, dest string, perm os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dest, data, perm); err != nil {
		return err
	}
	return nil
}

func (s *Service) copyDatabase(ctx context.Context, src, dest string, force bool) error {
	if _, err := os.Stat(dest); err == nil && !force {
		return fmt.Errorf("destination exists: %s", dest)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	db, err := s.openDatabase(ctx, src)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s';", dest))
	if err != nil {
		return err
	}
	return os.Chmod(dest, s.Config.FilePermissions)
}

func (s *Service) countTables(ctx context.Context, path string) (int64, error) {
	db, err := s.openDatabase(ctx, path)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	return querySingleInt(ctx, db, "SELECT COUNT(*) FROM sqlite_master WHERE type='table';"), nil
}

func (s *Service) integrityCheck(ctx context.Context, path string) error {
	db, err := s.openDatabase(ctx, path)
	if err != nil {
		return err
	}
	defer db.Close()
	result := querySingleText(ctx, db, "PRAGMA integrity_check;")
	if strings.ToLower(strings.TrimSpace(result)) != "ok" {
		return fmt.Errorf("integrity_check returned %s", result)
	}
	return nil
}

func removeJournalFiles(base string) error {
	var errs []string
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.Remove(base + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (s *Service) pruneBackups(base string, now time.Time) error {
	entries, err := os.ReadDir(s.Config.BackupPath)
	if err != nil {
		return err
	}
	cutoff := now.Add(-time.Duration(s.Config.BackupRetentionDays) * 24 * time.Hour)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, base+"_") || filepath.Ext(name) != ".db" {
			continue
		}
		parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(name, base+"_"), ".db"), "_")
		if len(parts) == 0 {
			continue
		}
		tsRaw := parts[0]
		parsed, err := time.Parse("20060102", tsRaw)
		if err != nil {
			if len(tsRaw) == len("20060102_150405") {
				parsed, err = time.Parse("20060102_150405", tsRaw)
			}
		}
		if err != nil {
			info, statErr := e.Info()
			if statErr != nil {
				continue
			}
			parsed = info.ModTime()
		}
		if parsed.Before(cutoff) {
			_ = os.Remove(filepath.Join(s.Config.BackupPath, name))
		}
	}
	return nil
}

func (s *Service) tableColumns(ctx context.Context, db *sql.DB, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s);", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

func (s *Service) validateColumnsExist(cols []string, tableCols []string) error {
	tableSet := map[string]struct{}{}
	for _, c := range tableCols {
		tableSet[strings.ToLower(c)] = struct{}{}
	}
	for _, col := range cols {
		if _, ok := tableSet[strings.ToLower(col)]; !ok {
			return fmt.Errorf("column %s not found in table", col)
		}
	}
	return nil
}

func normalizeColumns(cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = strings.TrimSpace(c)
	}
	return out
}

// Simple encryption helpers (AES-GCM with PBKDF2-derived key).
const (
	encMagic     = "SQLENC1"
	sqliteHeader = "SQLite format 3"
)

func encryptBytes(data []byte, password string) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key, iv := deriveKey(password, salt)
	aesgcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	cipher := aesgcm.Seal(nil, iv, data, nil)
	out := append([]byte(encMagic), salt...)
	out = append(out, iv...)
	out = append(out, cipher...)
	return out, nil
}

func decryptBytes(cipher []byte, password string) ([]byte, error) {
	if len(cipher) < len(encMagic)+16+12 || string(cipher[:len(encMagic)]) != encMagic {
		return nil, errors.New("not an encrypted database")
	}
	salt := cipher[len(encMagic) : len(encMagic)+16]
	iv := cipher[len(encMagic)+16 : len(encMagic)+28]
	payload := cipher[len(encMagic)+28:]
	key, _ := deriveKey(password, salt)
	aesgcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	return aesgcm.Open(nil, iv, payload, nil)
}

func deriveKey(password string, salt []byte) (key []byte, iv []byte) {
	key = pbkdf2.Key([]byte(password), salt, 12000, 32+12, sha256.New)
	return key[:32], key[32:]
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
