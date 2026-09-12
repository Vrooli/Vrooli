package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"backdrop-studio/internal/vector/authoring"
)

// The authored-generator store.
//
// An authored generator is catalog data: it arrives from a model, is admitted
// by validation, and is then bound to by styles exactly as a built-in preset
// is. Storing it here rather than in a file is what makes it upgradeable,
// listable and refusable — and what lets a style's binding be checked before a
// render rather than during one.

// PutAuthoredGenerator stores a validated generator.
//
// A generator whose report did not pass is refused rather than stored with a
// flag, because a stored generator is one a later code path may reasonably
// assume was checked. The place to keep a failed attempt is the operator's
// screen, where the refusals are actionable.
func (s *Store) PutAuthoredGenerator(ctx context.Context, g authoring.Generator) error {
	if strings.TrimSpace(g.ID) == "" {
		return fmt.Errorf("catalog: an authored generator needs an id")
	}
	if !g.Validation.Passed {
		return fmt.Errorf("catalog: generator %q did not pass validation and is not stored; refusals: %s",
			g.ID, strings.Join(g.Validation.Refusals, "; "))
	}
	if strings.TrimSpace(g.ModelID) == "" {
		return fmt.Errorf("catalog: generator %q names no authoring model; an asset drawn by it could not be disclosed", g.ID)
	}
	if strings.TrimSpace(g.Prompt) == "" {
		return fmt.Errorf("catalog: generator %q carries no authoring prompt; it could never be re-authored or reviewed", g.ID)
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO backdrop_authored_generators (id, name, template, params, inks, prompt, model_id, validation, validated, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?)
ON CONFLICT(id) DO UPDATE SET
  name=excluded.name, template=excluded.template, params=excluded.params, inks=excluded.inks,
  prompt=excluded.prompt, model_id=excluded.model_id, validation=excluded.validation, validated=1`,
		g.ID, g.Name, g.Template, mustJSON(g.Params), mustJSON(g.Inks), g.Prompt, g.ModelID,
		mustJSON(g.Validation), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("catalog: store authored generator %q: %w", g.ID, err)
	}
	return nil
}

// AuthoredGenerator reads one by id. A generator that is absent or was stored
// unvalidated is reported as missing, so a caller cannot render from one.
func (s *Store) AuthoredGenerator(ctx context.Context, id string) (authoring.Generator, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, template, params, inks, prompt, model_id, validation
FROM backdrop_authored_generators WHERE id=? AND validated=1`, id)
	var (
		g                     authoring.Generator
		params, inks, verdict string
	)
	if err := row.Scan(&g.ID, &g.Name, &g.Template, &params, &inks, &g.Prompt, &g.ModelID, &verdict); err != nil {
		if err == sql.ErrNoRows {
			return authoring.Generator{}, &UnknownGeneratorError{ID: id}
		}
		return authoring.Generator{}, fmt.Errorf("catalog: read authored generator %q: %w", id, err)
	}
	_ = json.Unmarshal([]byte(params), &g.Params)
	_ = json.Unmarshal([]byte(inks), &g.Inks)
	_ = json.Unmarshal([]byte(verdict), &g.Validation)
	return g, nil
}

// ListAuthoredGenerators returns every validated generator, id-ordered.
func (s *Store) ListAuthoredGenerators(ctx context.Context) ([]authoring.Generator, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, template, params, inks, prompt, model_id, validation
FROM backdrop_authored_generators WHERE validated=1 ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("catalog: list authored generators: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []authoring.Generator
	for rows.Next() {
		var (
			g                     authoring.Generator
			params, inks, verdict string
		)
		if err := rows.Scan(&g.ID, &g.Name, &g.Template, &params, &inks, &g.Prompt, &g.ModelID, &verdict); err != nil {
			return nil, fmt.Errorf("catalog: scan authored generator: %w", err)
		}
		_ = json.Unmarshal([]byte(params), &g.Params)
		_ = json.Unmarshal([]byte(inks), &g.Inks)
		_ = json.Unmarshal([]byte(verdict), &g.Validation)
		out = append(out, g)
	}
	return out, rows.Err()
}

// DeleteAuthoredGenerator removes one.
func (s *Store) DeleteAuthoredGenerator(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM backdrop_authored_generators WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("catalog: delete authored generator %q: %w", id, err)
	}
	return nil
}

// UnknownGeneratorError reports a style bound to a generator that is not stored
// and validated.
//
// It is a distinct type because it maps to FailedPrecondition at the handler
// edge and because the fix is specific: author and store the generator, or
// change the style's binding. A retry never helps.
type UnknownGeneratorError struct{ ID string }

func (e *UnknownGeneratorError) Error() string {
	return fmt.Sprintf(
		"catalog: no validated authored generator named %q; author and store it, or bind the style to a built-in preset", e.ID)
}
