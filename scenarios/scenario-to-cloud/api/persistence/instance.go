package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"scenario-to-cloud/instance"
)

func (r *Repository) CreateInstance(ctx context.Context, value instance.Instance) (instance.Instance, error) {
	command, err := json.Marshal(value.Command)
	if err != nil {
		return instance.Instance{}, fmt.Errorf("encode instance command: %w", err)
	}
	if value.ID == "" {
		row := r.db.QueryRowContext(ctx, `
			INSERT INTO cloud_instances (name, provider, state, image, workdir, pid, command, created_at, updated_at, last_error)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			RETURNING id`, value.Name, value.Provider, value.State, value.Image, value.Workdir, value.PID, command, value.CreatedAt, value.UpdatedAt, value.LastError)
		if err := row.Scan(&value.ID); err != nil {
			return instance.Instance{}, fmt.Errorf("create instance: %w", err)
		}
		return value, nil
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO cloud_instances (id,name,provider,state,image,workdir,pid,command,created_at,updated_at,last_error) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, value.ID, value.Name, value.Provider, value.State, value.Image, value.Workdir, value.PID, command, value.CreatedAt, value.UpdatedAt, value.LastError)
	if err != nil {
		return instance.Instance{}, fmt.Errorf("create instance: %w", err)
	}
	return value, nil
}

func (r *Repository) GetInstance(ctx context.Context, id string) (*instance.Instance, error) {
	var value instance.Instance
	var command []byte
	err := r.db.QueryRowContext(ctx, `SELECT id,name,provider,state,image,workdir,pid,command,created_at,updated_at,last_error FROM cloud_instances WHERE id=$1`, id).Scan(&value.ID, &value.Name, &value.Provider, &value.State, &value.Image, &value.Workdir, &value.PID, &command, &value.CreatedAt, &value.UpdatedAt, &value.LastError)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get instance: %w", err)
	}
	if err := json.Unmarshal(command, &value.Command); err != nil {
		return nil, fmt.Errorf("decode instance command: %w", err)
	}
	return &value, nil
}

func (r *Repository) UpdateInstanceState(ctx context.Context, id string, state instance.State, pid int, lastError string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE cloud_instances SET state=$1,pid=$2,last_error=$3,updated_at=NOW() WHERE id=$4`, state, pid, lastError, id)
	if err != nil {
		return fmt.Errorf("update instance state: %w", err)
	}
	if count, err := result.RowsAffected(); err == nil && count == 0 {
		return instance.ErrInstanceNotFound
	}
	return nil
}

func (r *Repository) DeleteInstance(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM cloud_instances WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete instance: %w", err)
	}
	if count, err := result.RowsAffected(); err == nil && count == 0 {
		return instance.ErrInstanceNotFound
	}
	return nil
}
