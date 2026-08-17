// Package library owns the durable, explicitly promoted program library.
// Successful submissions remain corpus evidence; they never become reusable
// library programs without an operator-controlled promotion call.
package library

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shared"
	"program-runtime/internal/sessions"
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

type Seed struct {
	Name, Source, Description string
}

var calledBindingPattern = regexp.MustCompile(`vrooli\.([A-Za-z0-9_-]+)(?:\.([A-Za-z0-9_-]+))?(?:\.([A-Za-z0-9_-]+))?`)

func ExtractCalledBindingIDs(source string) []string {
	seen := make(map[string]struct{})
	for _, match := range calledBindingPattern.FindAllStringSubmatch(source, -1) {
		if len(match) < 4 || match[2] == "" || match[3] == "" {
			continue
		}
		id := strings.ReplaceAll(match[1], "_", "-") + "/" + strings.ReplaceAll(match[2], "_", "-") + "/" + strings.ReplaceAll(match[3], "_", "-")
		seen[id] = struct{}{}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func NewRepository(db SQLExecutor) *Repository { return &Repository{db: db} }

func (r *Repository) EnsureSeeded(ctx context.Context, seeds []Seed, now time.Time) error {
	for _, seed := range seeds {
		if strings.TrimSpace(seed.Name) == "" || strings.TrimSpace(seed.Source) == "" {
			continue
		}
		called, _ := json.Marshal(ExtractCalledBindingIDs(seed.Source))
		_, err := r.db.ExecContext(ctx, `INSERT INTO library_programs (id,name,version,source,description,origin,created_at,source_program_id,promoted_by,promotion_reason,called_binding_ids) SELECT ?,?,?,?,?,?,?,?,?,?,? WHERE NOT EXISTS (SELECT 1 FROM library_programs WHERE name=? AND version=1)`, "lib_seed_"+seed.Name, seed.Name, 1, seed.Source, seed.Description, "seeded", now.UTC().Format(time.RFC3339Nano), "", "", "", string(called), seed.Name)
		if err != nil {
			return fmt.Errorf("seed library %q: %w", seed.Name, err)
		}
		if _, err := r.db.ExecContext(ctx, `UPDATE library_programs SET called_binding_ids=? WHERE name=? AND version=1 AND origin='seeded' AND (called_binding_ids='' OR called_binding_ids='[]')`, string(called), seed.Name); err != nil {
			return fmt.Errorf("refresh seeded library bindings %q: %w", seed.Name, err)
		}
		_, err = r.db.ExecContext(ctx, `INSERT INTO library_current(name,version) SELECT ?,1 WHERE NOT EXISTS (SELECT 1 FROM library_current WHERE name=?)`, seed.Name, seed.Name)
		if err != nil {
			return fmt.Errorf("select seeded library %q: %w", seed.Name, err)
		}
	}
	return nil
}

// RemoveSeededAliases is the one-time migration for the pre-promotion
// library shape. Seeded rows were aliases masquerading as reusable programs;
// deleting only rows explicitly marked origin=seeded leaves promoted history
// intact and makes the durable library match its current contract.
func (r *Repository) RemoveSeededAliases(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM library_current WHERE name IN (SELECT name FROM library_programs WHERE origin='seeded')`); err != nil {
		return fmt.Errorf("remove seeded library selections: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM library_programs WHERE origin='seeded'`); err != nil {
		return fmt.Errorf("remove seeded library aliases: %w", err)
	}
	return nil
}

func (r *Repository) List(ctx context.Context) ([]*sharedv1.LibraryProgram, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT p.id,p.name,p.version,p.source,p.description,p.origin,p.created_at,p.source_program_id,p.promoted_by,p.promotion_reason,p.called_binding_ids,COALESCE(c.name IS NOT NULL,0) FROM library_programs p LEFT JOIN library_current c ON c.name=p.name AND c.version=p.version ORDER BY p.name,p.version`)
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

func (r *Repository) Get(ctx context.Context, name string, version int64) (*sharedv1.LibraryProgram, error) {
	var row rowScanner
	if version > 0 {
		row = r.db.QueryRowContext(ctx, `SELECT p.id,p.name,p.version,p.source,p.description,p.origin,p.created_at,p.source_program_id,p.promoted_by,p.promotion_reason,p.called_binding_ids,COALESCE(c.name IS NOT NULL,0) FROM library_programs p LEFT JOIN library_current c ON c.name=p.name AND c.version=p.version WHERE p.name=? AND p.version=?`, name, version)
	} else {
		row = r.db.QueryRowContext(ctx, `SELECT p.id,p.name,p.version,p.source,p.description,p.origin,p.created_at,p.source_program_id,p.promoted_by,p.promotion_reason,p.called_binding_ids,1 FROM library_programs p JOIN library_current c ON c.name=p.name AND c.version=p.version WHERE p.name=?`, name)
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

func (r *Repository) Promote(ctx context.Context, program *programsv1.Program, name, description, promotedBy, reason string, now time.Time) (*sharedv1.LibraryProgram, error) {
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
	ids, _ := json.Marshal(ExtractCalledBindingIDs(program.GetSource()))
	_, err := r.db.ExecContext(ctx, `INSERT INTO library_programs (id,name,version,source,description,origin,created_at,source_program_id,promoted_by,promotion_reason,called_binding_ids) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, "lib_"+uuid.NewString(), name, version, program.GetSource(), strings.TrimSpace(description), "promoted", now.UTC().Format(time.RFC3339Nano), program.GetId(), promotedBy, reason, string(ids))
	if err != nil {
		return nil, fmt.Errorf("promote library program: %w", err)
	}
	return r.Get(ctx, name, version)
}

func (r *Repository) PromoteByID(ctx context.Context, programID, name, description, promotedBy, reason string, now time.Time) (*sharedv1.LibraryProgram, error) {
	var source, status string
	if err := r.db.QueryRowContext(ctx, `SELECT source,status FROM programs WHERE id=?`, strings.TrimSpace(programID)).Scan(&source, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read source program: %w", err)
	}
	program := &programsv1.Program{Id: programID, Source: source, Status: parseProgramStatus(status)}
	return r.Promote(ctx, program, name, description, promotedBy, reason, now)
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
	if err := row.Scan(&p.Id, &p.Name, &p.Version, &p.Source, &p.Description, &p.Origin, &p.CreatedAt, &p.SourceProgramId, &p.PromotedBy, &p.PromotionReason, &ids, &current); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(ids), &p.CalledBindingIds)
	p.Current = current != 0
	return p, nil
}
