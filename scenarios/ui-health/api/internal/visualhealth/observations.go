package visualhealth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ObservationRepository hides persistence from the browser-evidence service.
// Reports are metadata only, capped at 16KiB each and 256 records globally.
var ErrInvalidObservation = errors.New("bounded scenario, client_id and valid layout_json object are required")

type ObservationRepository interface {
	Record(context.Context, *pb.BrowserObservation) (*pb.BrowserObservation, error)
	List(context.Context, string, string, int) ([]*pb.BrowserObservation, error)
}

type SQLObservations struct{ db *sql.DB }

func NewSQLObservations(db *sql.DB) *SQLObservations { return &SQLObservations{db: db} }

func ObservationSchema() string {
	return `
 CREATE TABLE IF NOT EXISTS browser_layout_observations (
  id TEXT PRIMARY KEY, scenario TEXT NOT NULL, client_id TEXT NOT NULL,
  received_at TEXT NOT NULL, payload TEXT NOT NULL
 );
 CREATE INDEX IF NOT EXISTS browser_layout_observations_scenario_time
 ON browser_layout_observations(scenario, received_at);
`
}

func (s *SQLObservations) Record(ctx context.Context, input *pb.BrowserObservation) (*pb.BrowserObservation, error) {
	if input == nil || len(input.GetLayoutJson()) > 16384 || len(input.GetScenario()) > 100 ||
		len(input.GetClientId()) > 100 || len(input.GetBuildId()) > 200 || len(input.GetUserAgent()) > 1000 ||
		strings.TrimSpace(input.GetScenario()) == "" || strings.TrimSpace(input.GetClientId()) == "" ||
		!json.Valid([]byte(input.GetLayoutJson())) || !strings.HasPrefix(strings.TrimSpace(input.GetLayoutJson()), "{") {
		return nil, ErrInvalidObservation
	}
	row := proto.Clone(input).(*pb.BrowserObservation)
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	row.Id = hex.EncodeToString(id)
	now := time.Now().UTC()
	row.ReceivedAt = now.Format(time.RFC3339Nano)
	row.CaptureKind = "browser_geometry"
	row.Findings = browserObservationFindings(row)
	payload, err := protojson.Marshal(row)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO browser_layout_observations (id, scenario, client_id, received_at, payload) VALUES (?,?,?,?,?)`, row.Id, row.Scenario, row.ClientId, row.ReceivedAt, string(payload)); err != nil {
		return nil, err
	}
	// Bound both age and total size. Timestamp and id give deterministic ordering.
	if _, err = tx.ExecContext(ctx, `DELETE FROM browser_layout_observations WHERE received_at < ? OR id NOT IN (SELECT id FROM browser_layout_observations ORDER BY received_at DESC, id DESC LIMIT 256)`, now.Add(-24*time.Hour).Format(time.RFC3339Nano)); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *SQLObservations) List(ctx context.Context, scenario, clientID string, limit int) ([]*pb.BrowserObservation, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT payload FROM browser_layout_observations WHERE scenario = ? AND (? = '' OR client_id = ?) AND received_at >= ? ORDER BY received_at DESC, id DESC LIMIT ?`, scenario, clientID, clientID, time.Now().UTC().Add(-24*time.Hour).Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*pb.BrowserObservation, 0)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		row := &pb.BrowserObservation{}
		if err := protojson.Unmarshal([]byte(payload), row); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
