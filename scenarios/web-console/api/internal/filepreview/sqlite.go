package filepreview

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const MaxSQLiteBytes int64 = 32 << 20

var ErrSQLiteLimit = errors.New("database preview limit")
var ErrSQLiteBusy = errors.New("database preview busy")
var ErrSQLiteInvalid = errors.New("database snapshot unavailable")
var sqliteSlots = make(chan struct{}, 2)

// SQLiteSnapshot opens only an existing, regular, read-only source. A pinned read
// transaction includes committed WAL pages; backup produces a standalone image
// without checkpoints, journal-mode changes, or writes to the source database.
// Never use immutable=1 here: that ignores active journals.
func SQLiteSnapshot(ctx context.Context, filename string, size, mtime int64) (result []byte, resultErr error) {
	defer func() {
		var se *sqlite.Error
		if errors.As(resultErr, &se) && (se.Code()&255 == sqlite3.SQLITE_BUSY || se.Code()&255 == sqlite3.SQLITE_LOCKED) {
			resultErr = ErrSQLiteBusy
		}
	}()
	select {
	case sqliteSlots <- struct{}{}:
		defer func() { <-sqliteSlots }()
	default:
		return nil, ErrSQLiteBusy
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	before, err := os.Stat(filename)
	if err != nil || !before.Mode().IsRegular() {
		return nil, ErrSQLiteInvalid
	}
	if before.Size() != size || before.ModTime().UnixNano() != mtime {
		return nil, &Error{Code: CodeStale, Message: "database changed; reopen preview"}
	}
	if size < 100 {
		return nil, ErrSQLiteInvalid
	}
	if size > MaxSQLiteBytes {
		return nil, ErrSQLiteLimit
	}
	f, err := os.Open(filename)
	if err != nil {
		return nil, ErrSQLiteInvalid
	}
	var header [100]byte
	_, err = f.ReadAt(header[:], 0)
	f.Close()
	if err != nil || string(header[:16]) != "SQLite format 3\x00" {
		return nil, ErrSQLiteInvalid
	}
	dbURI := sqliteReadOnlyURI(filename)
	db, err := sql.Open("sqlite", dbURI)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	for id, value := range map[int]int{sqlite3.SQLITE_LIMIT_LENGTH: 1 << 20, sqlite3.SQLITE_LIMIT_SQL_LENGTH: 65536, sqlite3.SQLITE_LIMIT_COLUMN: 128, sqlite3.SQLITE_LIMIT_EXPR_DEPTH: 32, sqlite3.SQLITE_LIMIT_ATTACHED: 0, sqlite3.SQLITE_LIMIT_VDBE_OP: 100000} {
		if _, err = sqlite.Limit(conn, id, value); err != nil {
			return nil, err
		}
	}
	if _, err = conn.ExecContext(ctx, "BEGIN"); err != nil {
		return nil, err
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")
	// Force an actual read before sizing. BEGIN alone is deferred; pin the
	// same source snapshot used by the subsequent incremental backup.
	var rootPage int64
	if err = conn.QueryRowContext(ctx, "SELECT rootpage FROM sqlite_schema LIMIT 1").Scan(&rootPage); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	var pages, pageSize int64
	if err = conn.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pages); err != nil {
		return nil, err
	}
	if err = conn.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return nil, err
	}
	if pageSize < 512 || pageSize > 65536 || pages < 1 || pages > MaxSQLiteBytes/pageSize {
		return nil, ErrSQLiteLimit
	}
	var image []byte
	err = conn.Raw(func(raw any) error {
		source, ok := raw.(interface {
			NewBackup(string) (*sqlite.Backup, error)
		})
		if !ok {
			return ErrSQLiteInvalid
		}
		backup, e := source.NewBackup(":memory:")
		if e != nil {
			return e
		}
		finished := false
		defer func() {
			if !finished {
				_ = backup.Finish()
			}
		}()
		for copied := int64(0); ; copied += 128 {
			if e = ctx.Err(); e != nil {
				return e
			}
			if copied > MaxSQLiteBytes/pageSize+128 {
				return ErrSQLiteLimit
			}
			more, stepErr := backup.Step(128)
			if stepErr != nil {
				return stepErr
			}
			if !more {
				break
			}
		}
		destination, e := backup.Commit()
		finished = true
		if e != nil {
			return e
		}
		defer destination.Close()
		serializer, ok := destination.(interface {
			driver.Conn
			Serialize() ([]byte, error)
		})
		if !ok {
			return ErrSQLiteInvalid
		}
		image, e = serializer.Serialize()
		return e
	})
	if errors.Is(err, ErrSQLiteLimit) {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	after, err := os.Stat(filename)
	if err != nil || !os.SameFile(before, after) {
		return nil, &Error{Code: CodeStale, Message: "database replaced; reopen preview"}
	}
	if int64(len(image)) > MaxSQLiteBytes {
		return nil, ErrSQLiteLimit
	}
	// Backup is complete and contains all journal pages. Normalize only this
	// private image for read-only in-memory deserialization (SQLite cookbook).
	if len(image) >= 100 {
		image[18], image[19] = 1, 1
	}
	return image, nil
}

// A Windows drive must be a URI path, not its authority: file:///C:/..., never
// file://C:/.... URL encoding also preserves literal question marks and hashes.
func sqliteReadOnlyURI(filename string) string {
	sqlitePath := filepath.ToSlash(filename)
	if len(sqlitePath) >= 3 && sqlitePath[1] == ':' && sqlitePath[2] == '/' {
		sqlitePath = "/" + sqlitePath
	}
	u := url.URL{Scheme: "file", Path: sqlitePath}
	q := u.Query()
	q.Set("mode", "ro")
	q.Add("_pragma", "trusted_schema(0)")
	q.Add("_pragma", "query_only(1)")
	q.Add("_pragma", "busy_timeout(100)")
	q.Add("_pragma", "cache_size(-1024)")
	u.RawQuery = q.Encode()
	return u.String()
}
