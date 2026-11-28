package utilizationhistory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type SQLStore interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Sample struct {
	ObservedAt     time.Time           `json:"observed_at"`
	Complete       bool                `json:"complete"`
	ProviderRunIDs map[string][]string `json:"provider_run_ids"`
	UnmappedRunIDs []string            `json:"unmapped_run_ids"`
	Unknown        []string            `json:"unknown,omitempty"`
}

type Repository struct{ db SQLStore }

func NewRepository(db SQLStore) *Repository { return &Repository{db: db} }

func (r *Repository) Record(ctx context.Context, sample Sample) error {
	if r == nil || r.db == nil {
		return errors.New("utilization history store unavailable")
	}
	if sample.ObservedAt.IsZero() {
		return errors.New("utilization sample timestamp is required")
	}
	providers, err := json.Marshal(sample.ProviderRunIDs)
	if err != nil {
		return err
	}
	unmapped, err := json.Marshal(sample.UnmappedRunIDs)
	if err != nil {
		return err
	}
	unknown, err := json.Marshal(sample.Unknown)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO utilization_history_samples
		(id,observed_at,physical_complete,provider_run_ids_json,unmapped_run_ids_json,unknown_json)
		VALUES(?,?,?,?,?,?)`, uuid.NewString(), sample.ObservedAt.UTC().Format(time.RFC3339Nano), sample.Complete, string(providers), string(unmapped), string(unknown))
	return err
}

func (r *Repository) Read(ctx context.Context, since, until time.Time) ([]Sample, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("utilization history store unavailable")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT observed_at,physical_complete,provider_run_ids_json,unmapped_run_ids_json,unknown_json
		FROM utilization_history_samples WHERE observed_at>=? AND observed_at<=? ORDER BY observed_at`, since.UTC().Format(time.RFC3339Nano), until.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Sample{}
	for rows.Next() {
		var at, providers, unmapped, unknown string
		var complete bool
		if err := rows.Scan(&at, &complete, &providers, &unmapped, &unknown); err != nil {
			return nil, err
		}
		sample := Sample{Complete: complete}
		if sample.ObservedAt, err = time.Parse(time.RFC3339Nano, at); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(providers), &sample.ProviderRunIDs); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(unmapped), &sample.UnmappedRunIDs); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(unknown), &sample.Unknown); err != nil {
			return nil, err
		}
		result = append(result, sample)
	}
	return result, rows.Err()
}

// Prune bounds local history while retaining the full quota calibration horizon.
func (r *Repository) Prune(ctx context.Context, before time.Time) error {
	if r == nil || r.db == nil {
		return errors.New("utilization history store unavailable")
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM utilization_history_samples WHERE observed_at < ?`, before.UTC().Format(time.RFC3339Nano))
	return err
}
