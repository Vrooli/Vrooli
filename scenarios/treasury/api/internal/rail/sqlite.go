package rail

import (
	"context"
	"database/sql"
)

type SQLiteRepository struct {
	db interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	}
}

func NewSQLiteRepository(db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
},
) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Registration, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT name, enabled FROM rails ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []Registration
	for rows.Next() {
		var value Registration
		if err := rows.Scan(&value.Name, &value.Enabled); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

var _ Repository = (*SQLiteRepository)(nil)
