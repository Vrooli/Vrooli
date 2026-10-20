package projects

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func (r *sqliteRepository) CreateSourceMedia(ctx context.Context, media SourceMedia) (SourceMedia, bool, error) {
	query := `INSERT INTO project_source_media(id,project_id,sha256,size_bytes,media_type,width,height,orientation,color_model,color_profile_status,original_filename,imported_at,import_method,decoder,storage_key)
	VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(project_id,sha256) DO NOTHING
	RETURNING id,project_id,sha256,size_bytes,media_type,width,height,orientation,color_model,color_profile_status,original_filename,imported_at,import_method,decoder,storage_key`
	created, err := scanSourceMedia(r.db.QueryRowContext(ctx, query,
		media.ID, media.ProjectID, media.SHA256, media.SizeBytes, media.MediaType, media.Width, media.Height,
		media.Orientation, media.ColorModel, media.ColorProfileStatus, media.OriginalFilename,
		media.ImportedAt.UTC().Format(timeFormat), media.ImportMethod, media.Decoder, media.storageKey))
	if err == nil {
		return created, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return SourceMedia{}, false, fmt.Errorf("insert source media: %w", err)
	}
	winner, err := r.GetSourceMediaByDigest(ctx, media.ProjectID, media.SHA256)
	if err != nil {
		return SourceMedia{}, false, err
	}
	return winner, false, nil
}

func (r *sqliteRepository) GetSourceMedia(ctx context.Context, projectID, id string) (SourceMedia, error) {
	media, err := scanSourceMedia(r.db.QueryRowContext(ctx, `SELECT id,project_id,sha256,size_bytes,media_type,width,height,orientation,color_model,color_profile_status,original_filename,imported_at,import_method,decoder,storage_key FROM project_source_media WHERE project_id=? AND id=?`, projectID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return SourceMedia{}, ErrSourceMediaNotFound{ID: id}
	}
	if err != nil {
		return SourceMedia{}, fmt.Errorf("get source media: %w", err)
	}
	return media, nil
}

func (r *sqliteRepository) GetSourceMediaByDigest(ctx context.Context, projectID, digest string) (SourceMedia, error) {
	media, err := scanSourceMedia(r.db.QueryRowContext(ctx, `SELECT id,project_id,sha256,size_bytes,media_type,width,height,orientation,color_model,color_profile_status,original_filename,imported_at,import_method,decoder,storage_key FROM project_source_media WHERE project_id=? AND sha256=?`, projectID, digest))
	if errors.Is(err, sql.ErrNoRows) {
		return SourceMedia{}, ErrSourceMediaNotFound{ID: digest}
	}
	if err != nil {
		return SourceMedia{}, fmt.Errorf("get source media by digest: %w", err)
	}
	return media, nil
}

func (r *sqliteRepository) ListSourceMedia(ctx context.Context, projectID string, limit int, after *SourceMedia) ([]SourceMedia, error) {
	query := `SELECT id,project_id,sha256,size_bytes,media_type,width,height,orientation,color_model,color_profile_status,original_filename,imported_at,import_method,decoder,storage_key FROM project_source_media WHERE project_id=? ORDER BY imported_at DESC,id DESC LIMIT ?`
	args := []any{projectID, limit}
	if after != nil {
		query = `SELECT id,project_id,sha256,size_bytes,media_type,width,height,orientation,color_model,color_profile_status,original_filename,imported_at,import_method,decoder,storage_key FROM project_source_media WHERE project_id=? AND (imported_at<? OR (imported_at=? AND id<?)) ORDER BY imported_at DESC,id DESC LIMIT ?`
		cursorTime := after.ImportedAt.UTC().Format(timeFormat)
		args = []any{projectID, cursorTime, cursorTime, after.ID, limit}
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list source media: %w", err)
	}
	defer rows.Close()
	out := make([]SourceMedia, 0)
	for rows.Next() {
		media, scanErr := scanSourceMedia(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, media)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate source media: %w", err)
	}
	return out, nil
}

type sourceMediaScanner interface{ Scan(...any) error }

func scanSourceMedia(row sourceMediaScanner) (SourceMedia, error) {
	var media SourceMedia
	var imported string
	err := row.Scan(&media.ID, &media.ProjectID, &media.SHA256, &media.SizeBytes, &media.MediaType, &media.Width, &media.Height,
		&media.Orientation, &media.ColorModel, &media.ColorProfileStatus, &media.OriginalFilename, &imported,
		&media.ImportMethod, &media.Decoder, &media.storageKey)
	if err != nil {
		return SourceMedia{}, err
	}
	media.ImportedAt, err = time.Parse(parseTimeFormat, imported)
	if err != nil {
		return SourceMedia{}, fmt.Errorf("parse source media imported_at: %w", err)
	}
	return media, nil
}
