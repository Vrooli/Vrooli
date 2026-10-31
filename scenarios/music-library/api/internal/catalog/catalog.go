package catalog

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

//go:embed schema.sql
var schema string

func Schema() string { return schema }

var (
	hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	ErrInvalid  = errors.New("invalid catalog input")
	ErrNotFound = errors.New("track not found")
)

type (
	Use struct {
		Project  string
		Revision string
		Status   string
	}
	Track struct {
		SHA256      string
		Title       string
		SourcePath  string
		TakeID      string
		ReceiptPath string
		Tags        []string
		Approval    string
		Feedback    string
		Uses        []Use
	}
	Filter struct {
		Tag        string
		UnusedOnly bool
		Project    string
	}
	Repository interface {
		Put(context.Context, Track) (Track, error)
		List(context.Context, Filter) ([]Track, error)
		RecordUse(context.Context, string, Use) (Track, error)
	}
	SQLExecutor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	Store struct{ db SQLExecutor }
)

func NewStore(db SQLExecutor) *Store { return &Store{db: db} }

func validate(t Track) error {
	if !hashPattern.MatchString(t.SHA256) || strings.TrimSpace(t.Title) == "" || strings.TrimSpace(t.SourcePath) == "" {
		return fmt.Errorf("%w: sha256, title and source path required", ErrInvalid)
	}
	if t.Approval != "liked" && t.Approval != "unreviewed" && t.Approval != "rejected" {
		return fmt.Errorf("%w: approval must be liked, unreviewed or rejected", ErrInvalid)
	}
	if len(t.Tags) > 32 || len(t.Uses) > 0 {
		return fmt.Errorf("%w: too many tags or unexpected uses", ErrInvalid)
	}
	for _, x := range t.Tags {
		if strings.TrimSpace(x) != x || x == "" || len(x) > 80 {
			return fmt.Errorf("%w: invalid tag", ErrInvalid)
		}
	}
	return nil
}

func (s *Store) Put(ctx context.Context, t Track) (Track, error) {
	if err := validate(t); err != nil {
		return Track{}, err
	}
	b, err := json.Marshal(t.Tags)
	if err != nil {
		return Track{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO music_catalog_tracks (sha256,title,source_path,take_id,receipt_path,tags_json,approval,feedback) VALUES (?,?,?,?,?,?,?,?) ON CONFLICT(sha256) DO UPDATE SET title=excluded.title,source_path=excluded.source_path,take_id=excluded.take_id,receipt_path=excluded.receipt_path,tags_json=excluded.tags_json,approval=excluded.approval,feedback=excluded.feedback`, t.SHA256, t.Title, t.SourcePath, t.TakeID, t.ReceiptPath, string(b), t.Approval, t.Feedback)
	if err != nil {
		return Track{}, fmt.Errorf("save track: %w", err)
	}
	return s.get(ctx, t.SHA256)
}

func (s *Store) get(ctx context.Context, id string) (Track, error) {
	var t Track
	var tags string
	err := s.db.QueryRowContext(ctx, `SELECT sha256,title,source_path,take_id,receipt_path,tags_json,approval,feedback FROM music_catalog_tracks WHERE sha256=?`, id).Scan(&t.SHA256, &t.Title, &t.SourcePath, &t.TakeID, &t.ReceiptPath, &tags, &t.Approval, &t.Feedback)
	if errors.Is(err, sql.ErrNoRows) {
		return Track{}, ErrNotFound
	}
	if err != nil {
		return Track{}, err
	}
	if err = json.Unmarshal([]byte(tags), &t.Tags); err != nil {
		return Track{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT project,revision,status FROM music_catalog_uses WHERE sha256=? ORDER BY project,revision`, id)
	if err != nil {
		return Track{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var u Use
		if err = rows.Scan(&u.Project, &u.Revision, &u.Status); err != nil {
			return Track{}, err
		}
		t.Uses = append(t.Uses, u)
	}
	return t, rows.Err()
}

func (s *Store) List(ctx context.Context, f Filter) ([]Track, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT sha256 FROM music_catalog_tracks ORDER BY title,sha256`)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []Track{}
	for _, id := range ids {
		t, e := s.get(ctx, id)
		if e != nil {
			return nil, e
		}
		if f.Tag != "" {
			found := false
			for _, tag := range t.Tags {
				if tag == f.Tag {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		used := false
		project := f.Project == ""
		for _, u := range t.Uses {
			if u.Status == "accepted" {
				used = true
			}
			if u.Project == f.Project {
				project = true
			}
		}
		if f.UnusedOnly && used {
			continue
		}
		if !project {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *Store) RecordUse(ctx context.Context, id string, u Use) (Track, error) {
	if !hashPattern.MatchString(id) || strings.TrimSpace(u.Project) == "" || strings.TrimSpace(u.Revision) == "" || (u.Status != "draft" && u.Status != "accepted") {
		return Track{}, fmt.Errorf("%w: project, revision and draft/accepted status required", ErrInvalid)
	}
	if _, err := s.get(ctx, id); err != nil {
		return Track{}, err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO music_catalog_uses (sha256,project,revision,status) VALUES (?,?,?,?) ON CONFLICT(sha256,project,revision) DO UPDATE SET status=CASE WHEN music_catalog_uses.status='accepted' THEN 'accepted' ELSE excluded.status END`, id, u.Project, u.Revision, u.Status)
	if err != nil {
		return Track{}, err
	}
	return s.get(ctx, id)
}
