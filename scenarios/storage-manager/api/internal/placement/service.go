package placement

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/vrooli/api-core/database"
	corestorage "github.com/vrooli/api-core/storage"
)

//go:embed schema.sql
var schemaSQL string

type Plan struct {
	ID          string    `json:"id"`
	Entry       string    `json:"entry"`
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
}

type Audit struct {
	ID              string    `json:"id"`
	PlanID          string    `json:"plan_id"`
	OccurredAt      time.Time `json:"occurred_at"`
	Event           string    `json:"event"`
	Source          string    `json:"source"`
	Destination     string    `json:"destination"`
	Bytes           int64     `json:"bytes"`
	Verified        bool      `json:"verified"`
	SourcePreserved bool      `json:"source_preserved"`
	Message         string    `json:"message,omitempty"`
}

type Verification struct {
	Kind           string `json:"kind"`
	Owner          string `json:"owner"`
	Entry          string `json:"entry"`
	Platform       string `json:"platform"`
	Applicable     bool   `json:"applicable"`
	DeclaredAbsent bool   `json:"declared_absent"`
	Path           string `json:"path,omitempty"`
	Error          string `json:"error,omitempty"`
}

type Service struct{ store Store }

type Store interface {
	SavePlan(context.Context, Plan) error
	GetPlan(context.Context, string) (Plan, bool, error)
	SaveAudit(context.Context, Audit) error
	ListAudit(context.Context, int) ([]Audit, error)
}

func New(db *database.RoutedDB) *Service {
	if db != nil {
		return &Service{store: &sqlStore{db: db}}
	}
	return &Service{store: &memoryStore{plans: map[string]Plan{}}}
}

func Schema() string { return schemaSQL }

// Verify resolves every declaration for a target platform without touching
// the filesystem. A typed NotApplicable result is a correct declared absence,
// while every other resolution error remains visible as unresolvable.
func (s *Service) Verify(_ context.Context, repoRoot string, owners []corestorage.OwnerManifest, platform corestorage.Platform) []Verification {
	result := make([]Verification, 0)
	for _, owner := range owners {
		for _, entry := range owner.StorageEntries {
			row := Verification{Kind: string(owner.Kind), Owner: owner.ID, Entry: entry.Name, Platform: string(platform)}
			path, err := corestorage.ResolveOwnerStoragePath(repoRoot, owner, entry, platform, corestorage.PlatformSeams{})
			if err == nil {
				row.Applicable = true
				row.Path = path
			} else {
				var absent *corestorage.NotApplicable
				if errors.As(err, &absent) {
					row.DeclaredAbsent = true
				} else {
					row.Error = err.Error()
				}
			}
			result = append(result, row)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Kind != result[j].Kind {
			return result[i].Kind < result[j].Kind
		}
		if result[i].Owner != result[j].Owner {
			return result[i].Owner < result[j].Owner
		}
		return result[i].Entry < result[j].Entry
	})
	return result
}

func (s *Service) Preview(ctx context.Context, entry, source, destination string) (Plan, error) {
	if entry == "" || !filepath.IsAbs(source) || !filepath.IsAbs(destination) {
		return Plan{}, fmt.Errorf("placement preview requires entry and absolute source/destination")
	}
	if source == destination {
		return Plan{}, fmt.Errorf("placement preview source and destination must differ")
	}
	if _, err := os.Stat(source); err != nil {
		return Plan{}, fmt.Errorf("placement preview source: %w", err)
	}
	if _, err := os.Stat(destination); err == nil {
		return Plan{}, fmt.Errorf("placement preview destination already exists")
	} else if !os.IsNotExist(err) {
		return Plan{}, fmt.Errorf("placement preview destination: %w", err)
	}
	now := time.Now().UTC()
	hash := sha256.Sum256([]byte(entry + "\x00" + source + "\x00" + destination))
	plan := Plan{ID: "placement-" + hex.EncodeToString(hash[:8]), Entry: entry, Source: source, Destination: destination, CreatedAt: now, Status: "preview"}
	if err := s.store.SavePlan(ctx, plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func (s *Service) Migrate(ctx context.Context, planID string, approved bool) (Audit, error) {
	plan, ok, err := s.store.GetPlan(ctx, planID)
	if err != nil {
		return Audit{}, err
	}
	if !ok {
		return Audit{}, fmt.Errorf("placement plan %q not found", planID)
	}
	if !approved {
		return Audit{}, fmt.Errorf("placement migration requires explicit approval")
	}
	result, migrateErr := corestorage.MigrateVerified(plan.ID, plan.Source, plan.Destination, true)
	audit := Audit{ID: plan.ID + "/" + time.Now().UTC().Format("20060102T150405.000000000Z"), PlanID: plan.ID, OccurredAt: time.Now().UTC(), Event: "migration.completed", Source: plan.Source, Destination: plan.Destination, Bytes: result.Bytes, Verified: result.Verified, SourcePreserved: result.SourcePreserved}
	if migrateErr != nil {
		audit.Event = "migration.failed"
		audit.Message = migrateErr.Error()
		audit.SourcePreserved = true
	}
	if err := s.store.SaveAudit(ctx, audit); err != nil {
		return audit, err
	}
	if migrateErr != nil {
		return audit, migrateErr
	}
	return audit, nil
}

func (s *Service) Audit(ctx context.Context, limit int) ([]Audit, error) {
	return s.store.ListAudit(ctx, limit)
}

type memoryStore struct {
	plans  map[string]Plan
	audits []Audit
}

func (m *memoryStore) SavePlan(_ context.Context, p Plan) error { m.plans[p.ID] = p; return nil }
func (m *memoryStore) GetPlan(_ context.Context, id string) (Plan, bool, error) {
	p, ok := m.plans[id]
	return p, ok, nil
}

func (m *memoryStore) SaveAudit(_ context.Context, a Audit) error {
	m.audits = append(m.audits, a)
	return nil
}

func (m *memoryStore) ListAudit(_ context.Context, limit int) ([]Audit, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	start := len(m.audits) - limit
	if start < 0 {
		start = 0
	}
	out := append([]Audit(nil), m.audits[start:]...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

type sqlStore struct{ db *database.RoutedDB }

func (s *sqlStore) SavePlan(ctx context.Context, p Plan) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO placement_plans (id,entry,source,destination,created_at,status) VALUES (?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET status=excluded.status`, p.ID, p.Entry, p.Source, p.Destination, p.CreatedAt.Format(time.RFC3339Nano), p.Status)
	return err
}

func (s *sqlStore) GetPlan(ctx context.Context, id string) (Plan, bool, error) {
	var p Plan
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT id,entry,source,destination,created_at,status FROM placement_plans WHERE id=?`, id).Scan(&p.ID, &p.Entry, &p.Source, &p.Destination, &created, &p.Status)
	if err == sql.ErrNoRows {
		return Plan{}, false, nil
	}
	if err != nil {
		return Plan{}, false, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return p, true, nil
}

func (s *sqlStore) SaveAudit(ctx context.Context, a Audit) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO placement_audit (id,plan_id,occurred_at,event,source,destination,bytes,verified,source_preserved,message) VALUES (?,?,?,?,?,?,?,?,?,?)`, a.ID, a.PlanID, a.OccurredAt.Format(time.RFC3339Nano), a.Event, a.Source, a.Destination, a.Bytes, boolInt(a.Verified), boolInt(a.SourcePreserved), a.Message)
	return err
}

func (s *sqlStore) ListAudit(ctx context.Context, limit int) ([]Audit, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,plan_id,occurred_at,event,source,destination,bytes,verified,source_preserved,message FROM placement_audit ORDER BY occurred_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Audit{}
	for rows.Next() {
		var a Audit
		var stamp string
		var verified, preserved int
		if err := rows.Scan(&a.ID, &a.PlanID, &stamp, &a.Event, &a.Source, &a.Destination, &a.Bytes, &verified, &preserved, &a.Message); err != nil {
			return nil, err
		}
		a.OccurredAt, _ = time.Parse(time.RFC3339Nano, stamp)
		a.Verified = verified != 0
		a.SourcePreserved = preserved != 0
		out = append(out, a)
	}
	return out, rows.Err()
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
