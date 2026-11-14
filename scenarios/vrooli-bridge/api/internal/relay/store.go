package relay

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MemoryCommandStore is useful for isolated tests and development. Production
// wiring uses SQLiteCommandStore so command state survives a Bridge restart.
type MemoryCommandStore struct {
	mu      sync.Mutex
	records map[string]CommandRecord
}

func NewMemoryCommandStore() *MemoryCommandStore {
	return &MemoryCommandStore{records: make(map[string]CommandRecord)}
}

func (s *MemoryCommandStore) Reserve(_ context.Context, request Request) (CommandRecord, error) {
	request = normalizeRequest(request)
	if request.CommandID == "" {
		return CommandRecord{}, ErrInvalidRequest
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.records[request.CommandID]; ok {
		if requestFingerprint(existing.Request) != requestFingerprint(request) {
			return CommandRecord{}, ErrCommandConflict
		}
		if existing.State == StateNotAdmitted {
			existing.State = StateReserved
			existing.UpdatedAt = now
			s.records[request.CommandID] = existing
		}
		return cloneRecord(existing), nil
	}
	record := CommandRecord{CommandID: request.CommandID, Request: cloneRequest(request), State: StateReserved, UpdatedAt: now}
	s.records[request.CommandID] = record
	return cloneRecord(record), nil
}

func (s *MemoryCommandStore) Update(_ context.Context, commandID, nodeID string, state CommandState, response Response) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[commandID]
	if !ok || record.Request.NodeID != nodeID {
		return ErrCommandNotFound
	}
	record.State = state
	record.Response = cloneResponse(response)
	record.UpdatedAt = time.Now().UTC()
	s.records[commandID] = record
	return nil
}

func (s *MemoryCommandStore) Get(_ context.Context, commandID, nodeID string) (CommandRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[commandID]
	if !ok || record.Request.NodeID != nodeID {
		return CommandRecord{}, ErrCommandNotFound
	}
	return cloneRecord(record), nil
}

// SQLExecutor is the narrow database contract shared by Bridge's routed
// database and sqlite test fixtures.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteCommandStore struct{ db SQLExecutor }

func NewSQLiteCommandStore(db SQLExecutor) (*SQLiteCommandStore, error) {
	if db == nil {
		return nil, errors.New("relay command store database is required")
	}
	return &SQLiteCommandStore{db: db}, nil
}

func (s *SQLiteCommandStore) Reserve(ctx context.Context, request Request) (CommandRecord, error) {
	request = normalizeRequest(request)
	if request.CommandID == "" {
		return CommandRecord{}, ErrInvalidRequest
	}
	argsJSON, err := json.Marshal(request.Args)
	if err != nil {
		return CommandRecord{}, fmt.Errorf("encode relay args: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO relay_commands
(command_id, fingerprint, correlation_id, actor, node_id, scenario, command, args_json, timeout_seconds, max_response_bytes, state, response_kind, response_data, response_reason, response_exit_code, response_total_bytes, route_name, route_cost_units, route_latency_ms, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', NULL, '', 0, 0, '', 0, 0, ?)`,
		request.CommandID, requestFingerprint(request), request.CorrelationID, request.Actor,
		request.NodeID, request.Scenario, request.Command, string(argsJSON), request.TimeoutSeconds,
		request.MaxResponseBytes, string(StateReserved), now)
	if err != nil {
		return CommandRecord{}, fmt.Errorf("reserve relay command: %w", err)
	}
	record, err := s.get(ctx, request.CommandID, request.NodeID)
	if err != nil {
		return CommandRecord{}, err
	}
	if requestFingerprint(record.Request) != requestFingerprint(request) {
		return CommandRecord{}, ErrCommandConflict
	}
	if record.State == StateNotAdmitted {
		_, err = s.db.ExecContext(ctx, `UPDATE relay_commands SET state = ?, updated_at = ? WHERE command_id = ? AND node_id = ? AND state = ?`, string(StateReserved), now, request.CommandID, request.NodeID, string(StateNotAdmitted))
		if err != nil {
			return CommandRecord{}, fmt.Errorf("re-reserve relay command: %w", err)
		}
		record.State = StateReserved
		record.UpdatedAt, _ = time.Parse(time.RFC3339Nano, now)
	}
	return record, nil
}

func (s *SQLiteCommandStore) Update(ctx context.Context, commandID, nodeID string, state CommandState, response Response) error {
	result, err := s.db.ExecContext(ctx, `UPDATE relay_commands
SET state = ?, response_kind = ?, response_data = ?, response_reason = ?, response_exit_code = ?, response_total_bytes = ?, route_name = ?, route_cost_units = ?, route_latency_ms = ?, updated_at = ?
WHERE command_id = ? AND node_id = ?`, string(state), response.Kind, response.Data, response.Reason,
		response.ExitCode, response.TotalBytes, response.Route, response.RouteCostUnits, response.RouteLatencyMS, time.Now().UTC().Format(time.RFC3339Nano), commandID, nodeID)
	if err != nil {
		return fmt.Errorf("update relay command: %w", err)
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		if err != nil {
			return fmt.Errorf("check relay command update: %w", err)
		}
		return ErrCommandNotFound
	}
	return nil
}

func (s *SQLiteCommandStore) Get(ctx context.Context, commandID, nodeID string) (CommandRecord, error) {
	return s.get(ctx, commandID, nodeID)
}

func (s *SQLiteCommandStore) get(ctx context.Context, commandID, nodeID string) (CommandRecord, error) {
	var record CommandRecord
	var request Request
	var argsJSON, state, responseKind, responseReason, routeName, updated string
	var responseData []byte
	var routeCost, routeLatency uint64
	err := s.db.QueryRowContext(ctx, `SELECT command_id, correlation_id, actor, node_id, scenario, command, args_json, timeout_seconds, max_response_bytes, state, response_kind, response_data, response_reason, response_exit_code, response_total_bytes, route_name, route_cost_units, route_latency_ms, updated_at
FROM relay_commands WHERE command_id = ? AND node_id = ?`, commandID, nodeID).Scan(
		&record.CommandID, &request.CorrelationID, &request.Actor, &request.NodeID, &request.Scenario, &request.Command,
		&argsJSON, &request.TimeoutSeconds, &request.MaxResponseBytes, &state, &responseKind, &responseData, &responseReason,
		&record.Response.ExitCode, &record.Response.TotalBytes, &routeName, &routeCost, &routeLatency, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return CommandRecord{}, ErrCommandNotFound
	}
	if err != nil {
		return CommandRecord{}, fmt.Errorf("read relay command: %w", err)
	}
	if err := json.Unmarshal([]byte(argsJSON), &request.Args); err != nil {
		return CommandRecord{}, fmt.Errorf("decode relay args: %w", err)
	}
	request.CommandID = record.CommandID
	record.Request = request
	record.State = CommandState(state)
	record.Response.CorrelationID = request.CorrelationID
	record.Response.Kind, record.Response.Data, record.Response.Reason = responseKind, append([]byte(nil), responseData...), responseReason
	record.Response.Route, record.Response.RouteCostUnits, record.Response.RouteLatencyMS = routeName, routeCost, routeLatency
	record.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return CommandRecord{}, fmt.Errorf("parse relay command timestamp: %w", err)
	}
	return record, nil
}

func requestFingerprint(request Request) string {
	args := request.Args
	if args == nil {
		args = []string{}
	}
	value := struct {
		Actor, NodeID, Scenario, Command string
		Args                             []string
		TimeoutSeconds, MaxResponseBytes int64
	}{request.Actor, request.NodeID, request.Scenario, request.Command, args, request.TimeoutSeconds, int64(request.MaxResponseBytes)}
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func cloneRequest(request Request) Request {
	request.Args = append([]string(nil), request.Args...)
	return request
}

func cloneRecord(record CommandRecord) CommandRecord {
	record.Request = cloneRequest(record.Request)
	record.Response = cloneResponse(record.Response)
	return record
}

func terminalState(state CommandState) bool {
	return state == StateCompleted || state == StateFailed || state == StateOutcomeUnknown
}

func stateFromResponse(response Response) CommandState {
	switch response.Kind {
	case KindCompleted:
		return StateCompleted
	case KindOutcomeUnknown:
		return StateOutcomeUnknown
	case KindFailed, KindTerminated:
		return StateFailed
	default:
		return StateSubmitted
	}
}

func responseForRecord(record CommandRecord) Response {
	response := cloneResponse(record.Response)
	if response.CorrelationID == "" {
		response.CorrelationID = record.Request.CorrelationID
	}
	if response.Kind == "" {
		response.Kind = string(record.State)
	}
	return response
}

func normalizeCommandID(value string) string { return strings.TrimSpace(value) }
