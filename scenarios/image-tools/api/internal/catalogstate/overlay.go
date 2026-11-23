package catalogstate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// An override is the operator's (or an agent's) explicit statement that a catalog
// entry is enabled or disabled, whatever the seed says. The seed ships in the
// binary and changes with it; an override is durable state that outlives the
// binary, so without provenance it silently shadows a seed that has since moved
// on (a 2026-11-10 agent's "disabled" kept RestoreFormer++ off after the seed
// enabled it).
//
// The rule, applied by Reconcile at every boot (the only time the seed can
// change):
//
//   - Every override records who set it, when, why, and the seed's enabled flag
//     at that moment (SeedEnabled).
//   - A pinned override always wins: it is a deliberate, standing choice.
//   - An unpinned override wins only while the seed still says what it said when
//     the override was written. When the seed's flag has changed since, the seed
//     change is newer than the override and wins: the override is marked
//     superseded (kept for the record, no longer applied).
//   - An override written before provenance existed has no recorded baseline. It
//     keeps its effect (never silently flipped) and is adopted: the current seed
//     becomes its baseline and its author is "unknown (before provenance)", so
//     the next seed change can supersede it. Where it contradicts the seed it is
//     reported so a human can pin it or clear it.

// UnknownActor attributes overrides written before provenance was recorded (the
// migration that created the override tables writes this literal).
const UnknownActor = "unknown (before provenance)"

// Override is one stored enabled-state override with its provenance.
type Override struct {
	ID      string
	Enabled bool
	// SetBy is who wrote the override: a declared actor (an agent or "operator"),
	// not an authenticated identity.
	SetBy  string
	SetAt  time.Time
	Reason string
	Pinned bool
	// SeedEnabled is the seed's enabled flag when the override was written; nil
	// until adopted for a pre-provenance row.
	SeedEnabled *bool
	// SupersededAt/SupersededNote are set when a later seed change replaced the
	// override. A superseded override is not applied.
	SupersededAt   time.Time
	SupersededNote string
}

// Active reports whether the override still applies.
func (o Override) Active() bool { return o.SupersededAt.IsZero() }

// Contradicts reports whether an active override disagrees with the seed's
// current flag.
func (o Override) Contradicts(seedEnabled bool) bool { return o.Active() && o.Enabled != seedEnabled }

// SeedState reports a catalog entry's current seed flag; known=false for an id
// that is no longer in the catalog.
type SeedState func(id string) (enabled, known bool)

// Reconciliation is one change Reconcile made, for the boot log.
type Reconciliation struct {
	ID     string
	Action string // "adopted" or "superseded"
	Detail string
}

// Overlay persists the enabled-state overrides of one catalog.
type Overlay struct{ table }

// NewOverlay opens the override store for a catalog kind.
func NewOverlay(db SQLExecutor, kind Kind) *Overlay {
	return &Overlay{table{db, kind.table("override")}}
}

// SetOverride writes (replacing any earlier override for the id, superseded or
// not) an override with its provenance. seedEnabled is the seed's current flag,
// recorded as the baseline the supersession rule compares against.
func (s *Overlay) SetOverride(ctx context.Context, o Override, seedEnabled bool, now time.Time) error {
	if strings.TrimSpace(o.ID) == "" {
		return fmt.Errorf("%s: id is required", s.name)
	}
	setBy := strings.TrimSpace(o.SetBy)
	if setBy == "" {
		return fmt.Errorf("%s: an override must say who set it", s.name)
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO "+s.name+` (id, enabled, set_by, set_at, reason, pinned, seed_enabled, superseded_at, superseded_note)
VALUES (?,?,?,?,?,?,?,'','') ON CONFLICT(id) DO UPDATE SET enabled=excluded.enabled, set_by=excluded.set_by,
set_at=excluded.set_at, reason=excluded.reason, pinned=excluded.pinned, seed_enabled=excluded.seed_enabled,
superseded_at='', superseded_note=''`,
		o.ID, o.Enabled, setBy, now.UTC().Format(time.RFC3339), strings.TrimSpace(o.Reason), o.Pinned, boolInt(seedEnabled))
	return err
}

// ClearOverride deletes an override so the entry follows the seed again. It
// reports whether a row existed.
func (s *Overlay) ClearOverride(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM "+s.name+" WHERE id=?", id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

const overrideColumns = "id, enabled, set_by, set_at, reason, pinned, seed_enabled, superseded_at, superseded_note"

func scanOverride(scan func(...any) error) (o Override, err error) {
	var setAt, supersededAt string
	var seed int
	if err = scan(&o.ID, &o.Enabled, &o.SetBy, &setAt, &o.Reason, &o.Pinned, &seed, &supersededAt, &o.SupersededNote); err != nil {
		return
	}
	o.SetAt, _ = time.Parse(time.RFC3339, setAt)
	o.SupersededAt, _ = time.Parse(time.RFC3339, supersededAt)
	if seed >= 0 {
		known := seed == 1
		o.SeedEnabled = &known
	}
	return
}

// LoadOverrides returns every stored override, superseded ones included, keyed
// by id. Views use it to show provenance.
func (s *Overlay) LoadOverrides(ctx context.Context) (map[string]Override, error) {
	rows, err := list(ctx, s.table, overrideColumns, scanOverride)
	if err != nil {
		return nil, err
	}
	out := make(map[string]Override, len(rows))
	for _, o := range rows {
		out[o.ID] = o
	}
	return out, nil
}

// LoadOverlay returns the enabled flag of every override that still applies.
// This is the map selection and the wire shape overlay on the seed, so a
// superseded override can never leak into either.
func (s *Overlay) LoadOverlay(ctx context.Context) (map[string]bool, error) {
	overrides, err := s.LoadOverrides(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(overrides))
	for id, o := range overrides {
		if o.Active() {
			out[id] = o.Enabled
		}
	}
	return out, nil
}

// GetOverride returns one override.
func (s *Overlay) GetOverride(ctx context.Context, id string) (Override, bool, error) {
	o, err := scanOverride(s.db.QueryRowContext(ctx, "SELECT "+overrideColumns+" FROM "+s.name+" WHERE id=?", id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Override{}, false, nil
	}
	return o, err == nil, err
}

// Reconcile applies the supersession rule against the current seed. It is
// idempotent and is called at boot, after the seed that shipped with this binary
// is known. Ids the seed no longer lists are left alone.
func (s *Overlay) Reconcile(ctx context.Context, seed SeedState, now time.Time) ([]Reconciliation, error) {
	overrides, err := s.LoadOverrides(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(overrides))
	for id := range overrides {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var changes []Reconciliation
	for _, id := range ids {
		o := overrides[id]
		current, known := seed(id)
		if !known || !o.Active() {
			continue
		}
		switch {
		case o.SeedEnabled == nil:
			if _, err := s.db.ExecContext(ctx, "UPDATE "+s.name+" SET seed_enabled=? WHERE id=?", boolInt(current), id); err != nil {
				return nil, err
			}
			detail := "recorded the current seed flag as its baseline"
			if o.Enabled != current {
				detail += fmt.Sprintf("; it contradicts the seed (seed enabled=%t, override enabled=%t)", current, o.Enabled)
			}
			changes = append(changes, Reconciliation{ID: id, Action: "adopted", Detail: detail})
		case *o.SeedEnabled != current && !o.Pinned:
			note := fmt.Sprintf("seed enabled changed %t to %t after %s set enabled=%t", *o.SeedEnabled, current, o.SetBy, o.Enabled)
			if !o.SetAt.IsZero() {
				note += " on " + o.SetAt.Format("2006-01-02")
			}
			if _, err := s.db.ExecContext(ctx, "UPDATE "+s.name+" SET superseded_at=?, superseded_note=? WHERE id=?", now.UTC().Format(time.RFC3339), note, id); err != nil {
				return nil, err
			}
			changes = append(changes, Reconciliation{ID: id, Action: "superseded", Detail: note})
		}
	}
	return changes, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
