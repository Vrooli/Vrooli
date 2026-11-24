package capture

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
)

var ErrRequestTokenConflict = errors.New("capture request token was already used with different input")

var requestTokenPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type CaptureRequestRecord struct {
	RequestToken string `db:"request_token"`
	RequestHash  string `db:"request_hash"`
	RequestJSON  string `db:"request_json"`
	ExecutionID  string `db:"execution_id"`
	State        string `db:"state"`
	ResponseJSON string `db:"response_json"`
	Error        string `db:"error"`
}

// CaptureRequestStore durably binds a caller token to one normalized request
// and execution identity. Implementations must make Create atomic across API
// processes.
type CaptureRequestStore interface {
	Create(context.Context, CaptureRequestRecord) (record CaptureRequestRecord, created bool, err error)
	Get(context.Context, string) (CaptureRequestRecord, error)
	Complete(context.Context, string, string) error
	Fail(context.Context, string, string) error
}

type SQLCaptureRequestStore struct{ DB *sqlx.DB }

func NewSQLCaptureRequestStore(db *sqlx.DB) SQLCaptureRequestStore {
	return SQLCaptureRequestStore{DB: db}
}

func NormalizeRequestToken(raw string) (string, error) {
	token := strings.TrimSpace(raw)
	if token == "" || !requestTokenPattern.MatchString(token) {
		return "", fmt.Errorf("request token must contain 1 to 128 ASCII letters, digits, dot, underscore, colon, or hyphen")
	}
	return token, nil
}

func (s SQLCaptureRequestStore) Create(ctx context.Context, input CaptureRequestRecord) (CaptureRequestRecord, bool, error) {
	if s.DB == nil {
		return CaptureRequestRecord{}, false, errors.New("capture request database is unavailable")
	}
	result, err := s.DB.ExecContext(ctx, `
		INSERT OR IGNORE INTO capture_requests
		(request_token, request_hash, request_json, execution_id, state)
		VALUES (?, ?, ?, ?, 'in_progress')`, input.RequestToken, input.RequestHash, input.RequestJSON, input.ExecutionID)
	if err != nil {
		return CaptureRequestRecord{}, false, fmt.Errorf("reserve capture request token: %w", err)
	}
	record, err := s.Get(ctx, input.RequestToken)
	if err != nil {
		return CaptureRequestRecord{}, false, err
	}
	if record.RequestHash != input.RequestHash {
		return record, false, ErrRequestTokenConflict
	}
	inserted, _ := result.RowsAffected()
	return record, inserted == 1, nil
}

func (s SQLCaptureRequestStore) Get(ctx context.Context, token string) (CaptureRequestRecord, error) {
	if s.DB == nil {
		return CaptureRequestRecord{}, errors.New("capture request database is unavailable")
	}
	var record CaptureRequestRecord
	err := s.DB.GetContext(ctx, &record, `SELECT request_token, request_hash, request_json, execution_id, state, response_json, error FROM capture_requests WHERE request_token = ?`, token)
	if errors.Is(err, sql.ErrNoRows) {
		return CaptureRequestRecord{}, sql.ErrNoRows
	}
	if err != nil {
		return CaptureRequestRecord{}, fmt.Errorf("load capture request token: %w", err)
	}
	return record, nil
}

func (s SQLCaptureRequestStore) Complete(ctx context.Context, token, responseJSON string) error {
	result, err := s.DB.ExecContext(ctx, `UPDATE capture_requests SET state='completed', response_json=?, error='', updated_at=CURRENT_TIMESTAMP WHERE request_token=? AND state='in_progress'`, responseJSON, token)
	if err != nil {
		return fmt.Errorf("persist capture response: %w", err)
	}
	if n, _ := result.RowsAffected(); n != 1 {
		record, lookupErr := s.Get(ctx, token)
		if lookupErr != nil {
			return lookupErr
		}
		if record.State == "completed" && record.ResponseJSON == responseJSON {
			return nil
		}
		return fmt.Errorf("capture request %q is no longer in progress", token)
	}
	return nil
}

func (s SQLCaptureRequestStore) Fail(ctx context.Context, token, detail string) error {
	if len(detail) > 1000 {
		detail = detail[:1000]
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE capture_requests SET state='failed', error=?, updated_at=CURRENT_TIMESTAMP WHERE request_token=? AND state='in_progress'`, detail, token)
	if err != nil {
		return fmt.Errorf("persist capture failure: %w", err)
	}
	return nil
}
