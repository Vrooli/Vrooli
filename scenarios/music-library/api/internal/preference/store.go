package preference

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
)

var ErrJudgmentNotFound = errors.New("reversed judgment not found")
var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Store struct{ db database.SQLTxExecutor }

func NewStore(db database.SQLTxExecutor) *Store { return &Store{db: db} }

func normalizeContext(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "default"
	}
	if len(value) > 80 || strings.ContainsAny(value, "\r\n\x00") {
		return "", ErrInvalid
	}
	return value, nil
}

func newID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", id[:]), nil
}

func (s *Store) Get(ctx context.Context, contextName string) (Profile, error) {
	name, err := normalizeContext(contextName)
	if err != nil {
		return Profile{}, err
	}
	return s.load(ctx, s.db, name)
}

func (s *Store) SetControl(ctx context.Context, contextName string, control Control) (Profile, error) {
	name, err := normalizeContext(contextName)
	if err != nil {
		return Profile{}, err
	}
	validation := NewProfile()
	validation.Controls = []Control{control}
	if err := validation.Validate(); err != nil {
		return Profile{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Profile{}, err
	}
	defer tx.Rollback()
	p, err := s.load(ctx, tx, name)
	if err != nil {
		return Profile{}, err
	}
	found := false
	for i := range p.Controls {
		if p.Controls[i].Attribute == control.Attribute {
			p.Controls[i] = control
			found = true
			break
		}
	}
	if !found {
		p.Controls = append(p.Controls, control)
	}
	p = fitEvidence(p)
	if err := s.appendProfile(ctx, tx, &p); err != nil {
		return Profile{}, err
	}
	if err := appendAudit(ctx, tx, name, "control_set", control); err != nil {
		return Profile{}, err
	}
	if err := tx.Commit(); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (s *Store) RecordJudgment(ctx context.Context, contextName string, j Judgment) (Profile, error) {
	name, err := normalizeContext(contextName)
	if err != nil {
		return Profile{}, err
	}
	j.Context, j.Source = name, "operator"
	j.At = time.Now().UTC().Format(time.RFC3339Nano)
	if j.ID == "" {
		j.ID, err = newID()
		if err != nil {
			return Profile{}, err
		}
	}
	if err := validateStoredJudgment(j); err != nil {
		return Profile{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Profile{}, err
	}
	defer tx.Rollback()
	p, err := s.load(ctx, tx, name)
	if err != nil {
		return Profile{}, err
	}
	if j.Reverses != "" {
		var exists int
		err = tx.QueryRowContext(ctx, `SELECT 1 FROM music_taste_judgments WHERE judgment_id=? AND context=?`, j.Reverses, name).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return Profile{}, ErrJudgmentNotFound
		}
		if err != nil {
			return Profile{}, err
		}
	}
	encoded, err := json.Marshal(j)
	if err != nil {
		return Profile{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO music_taste_judgments (judgment_id,context,left_track_sha256,right_track_sha256,choice,judgment_json,judged_at) VALUES (?,?,?,?,?,?,?)`, j.ID, name, j.LeftID, j.RightID, string(j.Choice), string(encoded), j.At)
	if err != nil {
		return Profile{}, fmt.Errorf("append taste judgment: %w", err)
	}
	p.Judgments = append(p.Judgments, j)
	p = fitEvidence(p)
	if err := s.appendProfile(ctx, tx, &p); err != nil {
		return Profile{}, err
	}
	if err := appendAudit(ctx, tx, name, "judgment_recorded", j); err != nil {
		return Profile{}, err
	}
	if err := tx.Commit(); err != nil {
		return Profile{}, err
	}
	return p, nil
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func (s *Store) load(ctx context.Context, q queryer, name string) (Profile, error) {
	p := NewProfile()
	p.Context = name
	var raw string
	err := q.QueryRowContext(ctx, `SELECT profile_json FROM music_taste_profile_versions WHERE context=? ORDER BY revision DESC LIMIT 1`, name).Scan(&raw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Profile{}, err
	}
	if err == nil {
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			return Profile{}, err
		}
		if p.FeatureSpace == "catalog-tags-v1" || p.FeatureSpace == "" {
			p.FeatureSpace = FeatureSpaceVersion
		}
	}
	p.Judgments = nil
	rows, err := q.QueryContext(ctx, `SELECT judgment_json FROM music_taste_judgments WHERE context=? ORDER BY judged_at,judgment_id`, name)
	if err != nil {
		return Profile{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var encoded string
		var j Judgment
		if err := rows.Scan(&encoded); err != nil {
			return Profile{}, err
		}
		if err := json.Unmarshal([]byte(encoded), &j); err != nil {
			return Profile{}, err
		}
		p.Judgments = append(p.Judgments, j)
	}
	if err := rows.Err(); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (s *Store) appendProfile(ctx context.Context, q queryer, p *Profile) error {
	p.Revision++
	p.UpdatedAt = time.Now().UTC()
	encoded, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `INSERT INTO music_taste_profile_versions (context,revision,profile_json,created_at) VALUES (?,?,?,?)`, p.Context, p.Revision, string(encoded), p.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("append taste profile version: %w", err)
	}
	return nil
}

func appendAudit(ctx context.Context, q queryer, name, eventType string, value any) error {
	id, err := newID()
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `INSERT INTO music_taste_audit_events (event_id,context,event_type,event_json,created_at) VALUES (?,?,?,?,?)`, id, name, eventType, string(encoded), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("append taste audit event: %w", err)
	}
	return nil
}

func validateStoredJudgment(j Judgment) error {
	if !hashPattern.MatchString(j.LeftID) || !hashPattern.MatchString(j.RightID) || j.LeftID == j.RightID || j.Choice == "" || len(j.Reason) > 1000 || len(j.AttributeScope) > 120 {
		return ErrInvalid
	}
	if j.Choice != ChooseLeft && j.Choice != ChooseRight && j.Choice != ChooseTie && j.Choice != ChooseNeither && j.Choice != ChooseSkip {
		return ErrInvalid
	}
	if j.Reverses != "" && len(j.Reverses) > 80 {
		return ErrInvalid
	}
	if !validEvidence(j.LeftID, j.LeftEvidence) || !validEvidence(j.RightID, j.RightEvidence) {
		return ErrInvalid
	}
	return nil
}

func validEvidence(trackID string, features []Feature) bool {
	if len(features) > 256 {
		return false
	}
	for _, f := range features {
		if strings.TrimSpace(f.Attribute) != f.Attribute || f.Attribute == "" || len(f.Attribute) > 120 || f.Value < -1 || f.Value > 1 || math.IsNaN(f.Value) || math.IsInf(f.Value, 0) || f.Confidence <= 0 || f.Confidence > 1 || math.IsNaN(f.Confidence) || math.IsInf(f.Confidence, 0) || !validFeatureSource(trackID, f.Source) {
			return false
		}
	}
	return true
}

func validFeatureSource(trackID, source string) bool {
	if source == "catalog:"+trackID {
		return true
	}
	prefix := "decomposition:" + trackID + ":"
	if !strings.HasPrefix(source, prefix) {
		return false
	}
	jobID := strings.TrimPrefix(source, prefix)
	return jobID != "" && len(jobID) <= 160 && strings.TrimSpace(jobID) == jobID && !strings.ContainsAny(jobID, "\r\n\x00")
}

func fitEvidence(p Profile) Profile {
	// Refit from the fixed prior so each new profile revision is a deterministic
	// replay of the append-only evidence rather than an accidental second fit.
	p.Weights = map[string]Weight{}
	candidates := make([]Candidate, 0, len(p.Judgments)*2)
	for _, j := range p.Judgments {
		candidates = append(candidates, Candidate{ID: j.LeftID, Features: j.LeftEvidence}, Candidate{ID: j.RightID, Features: j.RightEvidence})
	}
	return FitPairwise(p, candidates, 50)
}
