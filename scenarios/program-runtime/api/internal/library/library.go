// Package library owns the durable, versioned program library. Promotion is
// explicit and only current versions are callable.
package library

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"program-runtime/internal/sessions"
	"program-runtime/internal/shapes"

	"github.com/google/uuid"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shared"
)

//go:embed schema.sql
var schema string

func Schema() string { return schema }

type SQLExecutor = sessions.SQLExecutor

var (
	ErrNotFound        = errors.New("library program not found")
	ErrSourceFailed    = errors.New("only a succeeded program may be promoted")
	ErrNameRequired    = errors.New("library name is required")
	ErrVersionRequired = errors.New("library version is required")
)

type Repository struct{ db SQLExecutor }

func NewRepository(db SQLExecutor) *Repository { return &Repository{db: db} }

func EnsureCompatibility(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(library_programs)")
	if err != nil {
		return fmt.Errorf("inspect library schema: %w", err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("scan library schema: %w", err)
		}
		if name == "tier" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !found {
		if _, err := db.ExecContext(ctx, "ALTER TABLE library_programs ADD COLUMN tier TEXT NOT NULL DEFAULT 'promoted'"); err != nil {
			return fmt.Errorf("add library tier: %w", err)
		}
	}
	for _, column := range []struct{ name, definition string }{
		{"declared_inputs", "TEXT NOT NULL DEFAULT '[]'"},
		{"declared_outputs", "TEXT NOT NULL DEFAULT '[]'"},
		{"coverage", "TEXT NOT NULL DEFAULT ''"},
		{"validated_at", "TEXT NOT NULL DEFAULT ''"},
	} {
		var exists bool
		probe, err := db.QueryContext(ctx, "PRAGMA table_info(library_programs)")
		if err != nil {
			return fmt.Errorf("inspect library column %s: %w", column.name, err)
		}
		for probe.Next() {
			var cid, notNull, primaryKey int
			var name, columnType string
			var defaultValue any
			if err := probe.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				probe.Close()
				return err
			}
			if name == column.name {
				exists = true
			}
		}
		probe.Close()
		if !exists {
			if _, err := db.ExecContext(ctx, "ALTER TABLE library_programs ADD COLUMN "+column.name+" "+column.definition); err != nil {
				return fmt.Errorf("add library.%s: %w", column.name, err)
			}
		}
	}
	if _, err := db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_library_programs_tier ON library_programs(tier, created_at)"); err != nil {
		return fmt.Errorf("index library tier: %w", err)
	}
	var invalid int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM library_programs WHERE tier <> 'promoted' OR tier IS NULL`).Scan(&invalid); err != nil {
		return fmt.Errorf("check library tiers: %w", err)
	}
	if invalid > 0 {
		return fmt.Errorf("library contains %d rows with a non-promoted tier", invalid)
	}
	return nil
}

func (r *Repository) List(ctx context.Context, pagination ...int) ([]*sharedv1.LibraryProgram, error) {
	limit, offset := -1, 0
	if len(pagination) > 0 && pagination[0] > 0 {
		limit = pagination[0]
	}
	if len(pagination) > 1 && pagination[1] > 0 {
		offset = pagination[1]
	}
	query := `SELECT p.id,p.name,p.version,p.source,p.description,p.origin,p.created_at,p.source_program_id,p.promoted_by,p.promotion_reason,p.called_binding_ids,p.tier,p.declared_inputs,p.declared_outputs,p.coverage,p.validated_at,COALESCE(c.name IS NOT NULL,0) FROM library_programs p LEFT JOIN library_current c ON c.name=p.name AND c.version=p.version ORDER BY CASE p.tier WHEN 'promoted' THEN 0 ELSE 1 END,p.name,p.version LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list library: %w", err)
	}
	defer rows.Close()
	var out []*sharedv1.LibraryProgram
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListCallable returns only the library versions selected for lib.<name>().
func (r *Repository) ListCallable(ctx context.Context) ([]*sharedv1.LibraryProgram, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT p.id,p.name,p.version,p.source,p.description,p.origin,p.created_at,p.source_program_id,p.promoted_by,p.promotion_reason,p.called_binding_ids,p.tier,p.declared_inputs,p.declared_outputs,p.coverage,p.validated_at,1 FROM library_programs p INNER JOIN library_current c ON c.name=p.name AND c.version=p.version ORDER BY p.name,p.version`)
	if err != nil {
		return nil, fmt.Errorf("list callable library: %w", err)
	}
	defer rows.Close()
	var out []*sharedv1.LibraryProgram
	for rows.Next() {
		program, scanErr := scan(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, program)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, name string, version int64) (*sharedv1.LibraryProgram, error) {
	var row rowScanner
	if version > 0 {
		row = r.db.QueryRowContext(ctx, `SELECT p.id,p.name,p.version,p.source,p.description,p.origin,p.created_at,p.source_program_id,p.promoted_by,p.promotion_reason,p.called_binding_ids,p.tier,p.declared_inputs,p.declared_outputs,p.coverage,p.validated_at,COALESCE(c.name IS NOT NULL,0) FROM library_programs p LEFT JOIN library_current c ON c.name=p.name AND c.version=p.version WHERE p.name=? AND p.version=?`, name, version)
	} else {
		row = r.db.QueryRowContext(ctx, `SELECT p.id,p.name,p.version,p.source,p.description,p.origin,p.created_at,p.source_program_id,p.promoted_by,p.promotion_reason,p.called_binding_ids,p.tier,p.declared_inputs,p.declared_outputs,p.coverage,p.validated_at,1 FROM library_programs p JOIN library_current c ON c.name=p.name AND c.version=p.version WHERE p.name=?`, name)
	}
	p, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get library %q: %w", name, err)
	}
	return p, nil
}

func (r *Repository) Promote(ctx context.Context, program *programsv1.Program, name, description, promotedBy, reason, coverage string, declaredInputs, declaredOutputs []string, now time.Time) (*sharedv1.LibraryProgram, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrNameRequired
	}
	if program == nil || program.GetStatus() != programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED {
		return nil, ErrSourceFailed
	}
	var version int64
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0)+1 FROM library_programs WHERE name=?`, name).Scan(&version); err != nil {
		return nil, fmt.Errorf("next library version: %w", err)
	}
	calledIDs, err := shapes.Derive(ctx, r.db, program.GetId())
	if err != nil {
		return nil, fmt.Errorf("derive successful bindings: %w", err)
	}
	ids, _ := json.Marshal(calledIDs)
	if len(declaredInputs) == 0 {
		declaredInputs = []string{"session_id"}
	}
	if len(declaredOutputs) == 0 {
		declaredOutputs = []string{"bounded projection"}
	}
	if strings.TrimSpace(coverage) == "" {
		coverage = "successful governed program"
	}
	inputs, _ := json.Marshal(declaredInputs)
	outputs, _ := json.Marshal(declaredOutputs)
	_, err = r.db.ExecContext(ctx, `INSERT INTO library_programs (id,name,version,source,description,origin,created_at,source_program_id,promoted_by,promotion_reason,called_binding_ids,tier,declared_inputs,declared_outputs,coverage,validated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, "lib_"+uuid.NewString(), name, version, program.GetSource(), strings.TrimSpace(description), "promoted", now.UTC().Format(time.RFC3339Nano), program.GetId(), promotedBy, reason, string(ids), "promoted", string(inputs), string(outputs), coverage, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("promote library program: %w", err)
	}
	return r.Get(ctx, name, version)
}

func (r *Repository) PromoteByID(ctx context.Context, programID, name, description, promotedBy, reason, coverage string, declaredInputs, declaredOutputs []string, now time.Time) (*sharedv1.LibraryProgram, error) {
	var source, status string
	if err := r.db.QueryRowContext(ctx, `SELECT source,status FROM programs WHERE id=?`, strings.TrimSpace(programID)).Scan(&source, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read source program: %w", err)
	}
	program := &programsv1.Program{Id: programID, Source: source, Status: parseProgramStatus(status)}
	return r.Promote(ctx, program, name, description, promotedBy, reason, coverage, declaredInputs, declaredOutputs, now)
}

func parseProgramStatus(status string) programsv1.ProgramStatus {
	if strings.EqualFold(strings.TrimSpace(status), "succeeded") {
		return programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED
	}
	return programsv1.ProgramStatus_PROGRAM_STATUS_FAILED
}

func (r *Repository) SetCurrent(ctx context.Context, name string, version int64) (*sharedv1.LibraryProgram, error) {
	p, err := r.Get(ctx, name, version)
	if err != nil {
		return nil, err
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO library_current(name,version) VALUES(?,?) ON CONFLICT(name) DO UPDATE SET version=excluded.version`, name, version); err != nil {
		return nil, fmt.Errorf("set current library version: %w", err)
	}
	p.Current = true
	return p, nil
}

func (r *Repository) CurrentStamp(ctx context.Context) string {
	rows, err := r.db.QueryContext(ctx, `SELECT name,version FROM library_current ORDER BY name`)
	if err != nil {
		return ""
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var name string
		var version int64
		if rows.Scan(&name, &version) == nil {
			parts = append(parts, fmt.Sprintf("%s@%d", name, version))
		}
	}
	return strings.Join(parts, ",")
}

type rowScanner interface{ Scan(...any) error }

func scan(row rowScanner) (*sharedv1.LibraryProgram, error) {
	p := &sharedv1.LibraryProgram{}
	var ids string
	var current int
	var inputs, outputs string
	if err := row.Scan(&p.Id, &p.Name, &p.Version, &p.Source, &p.Description, &p.Origin, &p.CreatedAt, &p.SourceProgramId, &p.PromotedBy, &p.PromotionReason, &ids, &p.Tier, &inputs, &outputs, &p.Coverage, &p.ValidatedAt, &current); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(ids), &p.CalledBindingIds)
	_ = json.Unmarshal([]byte(inputs), &p.DeclaredInputs)
	_ = json.Unmarshal([]byte(outputs), &p.DeclaredOutputs)
	p.Current = current != 0
	return p, nil
}
