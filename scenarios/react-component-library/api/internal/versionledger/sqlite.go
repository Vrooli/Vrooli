package versionledger

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Repository struct {
	db interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	}
	sourceRoot string
}

func NewRepository(db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}, sourceRoot string,
) *Repository {
	return &Repository{db: db, sourceRoot: sourceRoot}
}

// openSourceRoot anchors library filesystem operations to the configured
// library directory. Paths read from the ledger are still validated by
// sourceRelativePath before they are passed to the root.
func (r *Repository) openSourceRoot() (*os.Root, error) {
	if strings.TrimSpace(r.sourceRoot) == "" {
		return nil, fmt.Errorf("library source root is empty")
	}
	return os.OpenRoot(r.sourceRoot)
}

func sourceRelativePath(name string) (string, error) {
	name = filepath.FromSlash(name)
	if name == "" || filepath.IsAbs(name) {
		return "", fmt.Errorf("refusing path outside library root")
	}
	clean := filepath.Clean(name)
	if clean == "." || !filepath.IsLocal(clean) {
		return "", fmt.Errorf("refusing path outside library root")
	}
	return clean, nil
}

func readSourceFile(root *os.Root, name string) ([]byte, error) {
	relative, err := sourceRelativePath(name)
	if err != nil {
		return nil, err
	}
	return root.ReadFile(relative)
}

func sourceFileMode(root *os.Root, name string) (os.FileMode, error) {
	relative, err := sourceRelativePath(name)
	if err != nil {
		return 0, err
	}
	info, err := root.Lstat(relative)
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf("refusing to replace non-regular library file %q", name)
	}
	return info.Mode().Perm(), nil
}

func (v VersionLedger) String() string { return fmt.Sprintf("%s@%s", v.LibraryID, v.Version) }
