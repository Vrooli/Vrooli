// Package bridgeadmin owns LPBS-admin scoped scenario dispatch and its durable
// caller/effect receipt. Bridge remains authoritative for node grants and CLI
// manifest scope; this package never reaches a node directly.
package bridgeadmin

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/nodereach"
)

type Request struct {
	RequestID string   `json:"request_id"`
	Scenario  string   `json:"scenario"`
	Operation string   `json:"operation"`
	NodeID    string   `json:"node_id"`
	Tier      string   `json:"tier"`
	Args      []string `json:"args,omitempty"`
}
type Receipt struct {
	RequestID string    `json:"request_id"`
	Key       string    `json:"-"`
	Caller    string    `json:"caller"`
	Scenario  string    `json:"scenario"`
	Operation string    `json:"operation"`
	NodeID    string    `json:"node_id"`
	Tier      string    `json:"tier"`
	Status    string    `json:"status"`
	RunID     string    `json:"run_id,omitempty"`
	Replayed  bool      `json:"replayed,omitempty"`
	Detail    string    `json:"detail,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
type Dispatcher interface {
	Dispatch(context.Context, nodereach.DispatchRequest) (nodereach.DispatchResponse, error)
}
type Store interface {
	Begin(context.Context, Receipt) (Receipt, bool, error)
	Finish(context.Context, string, string, string, string, bool) error
	Get(context.Context, string) (Receipt, error)
}

func (s *Service) Get(ctx context.Context, caller, requestID string) (Receipt, error) {
	caller = strings.TrimSpace(caller)
	requestID = strings.TrimSpace(requestID)
	if caller == "" || requestID == "" || len(requestID) > 128 {
		return Receipt{}, errors.New("caller and request id are required")
	}
	key := sha256.Sum256([]byte(caller + "\x00" + requestID))
	receipt, err := s.store.Get(ctx, hex.EncodeToString(key[:]))
	if err != nil {
		return Receipt{}, err
	}
	if receipt.Caller != caller {
		return Receipt{}, errors.New("dispatch receipt not found")
	}
	return receipt, nil
}

type Service struct {
	dispatch Dispatcher
	store    Store
	allowed  map[string]map[string]bool
	now      func() time.Time
}

var ErrOutOfScope = errors.New("scenario or operation is outside LPBS dispatch scope")
var ErrBridgeRefused = errors.New("Bridge refused dispatch before target effects")

func New(dispatch Dispatcher, store Store, allowed map[string]map[string]bool) *Service {
	return &Service{dispatch: dispatch, store: store, allowed: allowed, now: time.Now}
}

// Submit records the caller's intent before Bridge dispatch. Requests outside
// the configured scenario/operation set are durably refused without calling
// Bridge. Stable request IDs are namespaced to the admin caller before they
// cross the shared service-principal boundary.
func (s *Service) Submit(ctx context.Context, caller string, req Request) (Receipt, error) {
	caller = strings.TrimSpace(caller)
	req.RequestID = strings.TrimSpace(req.RequestID)
	req.Scenario = strings.TrimSpace(req.Scenario)
	req.Operation = strings.TrimSpace(req.Operation)
	req.NodeID = strings.TrimSpace(req.NodeID)
	req.Tier = strings.TrimSpace(req.Tier)
	if caller == "" || req.RequestID == "" || len(req.RequestID) > 128 || req.Scenario == "" || req.Operation == "" || req.NodeID == "" || !validTier(req.Tier) {
		return Receipt{}, errors.New("caller, request id, scenario, operation, node and supported tier are required")
	}
	receipt := Receipt{RequestID: req.RequestID, Caller: caller, Scenario: req.Scenario, Operation: req.Operation, NodeID: req.NodeID, Tier: req.Tier, Status: "requested", CreatedAt: s.now().UTC()}
	receiptKey := sha256.Sum256([]byte(caller + "\x00" + req.RequestID))
	receipt.Key = hex.EncodeToString(receiptKey[:])
	prior, replayed, err := s.store.Begin(ctx, receipt)
	if err != nil {
		return Receipt{}, err
	}
	if replayed && prior.Status != "requested" && prior.Status != "uncertain" {
		if prior.Status == "refused" {
			return prior, ErrOutOfScope
		}
		return prior, nil
	}
	// A non-nil policy is available to focused owner deployments that need a
	// narrower pilot set. Production leaves it nil: Bridge's registry grants
	// and CLI manifests are the authoritative any-scenario/operation boundary.
	if s.allowed != nil && !s.allowed[req.Scenario][req.Operation] {
		_ = s.store.Finish(ctx, receipt.Key, "refused", "", "scenario or operation is not in the LPBS dispatch scope", false)
		receipt.Status = "refused"
		receipt.Detail = ErrOutOfScope.Error()
		return receipt, ErrOutOfScope
	}
	bridgeID := receipt.Key
	result, err := s.dispatch.Dispatch(ctx, nodereach.DispatchRequest{NodeID: req.NodeID, Scenario: req.Scenario, Verb: req.Operation, Args: append([]string(nil), req.Args...), DeploymentTier: req.Tier, RequestID: bridgeID, Timeout: 30 * time.Second})
	if err != nil {
		if nodereach.IsPreEffectDispatchRefusal(err) {
			if finishErr := s.store.Finish(ctx, receipt.Key, "refused", "", err.Error(), false); finishErr != nil {
				return Receipt{}, fmt.Errorf("persist Bridge refusal: %w", finishErr)
			}
			receipt.Status = "refused"
			receipt.Detail = err.Error()
			return receipt, ErrBridgeRefused
		}
		_ = s.store.Finish(ctx, receipt.Key, "uncertain", "", err.Error(), false)
		receipt.Status = "uncertain"
		receipt.Detail = err.Error()
		return receipt, err
	}
	if strings.TrimSpace(result.RunID) == "" {
		err = fmt.Errorf("Bridge accepted dispatch without a run receipt")
		_ = s.store.Finish(ctx, receipt.Key, "uncertain", "", err.Error(), false)
		receipt.Status = "uncertain"
		receipt.Detail = err.Error()
		return receipt, err
	}
	status := "accepted"
	if result.Replayed {
		status = "replayed"
	}
	if err = s.store.Finish(ctx, receipt.Key, status, result.RunID, "", result.Replayed); err != nil {
		return Receipt{}, err
	}
	receipt.Status = status
	receipt.RunID = result.RunID
	receipt.Replayed = result.Replayed
	return receipt, nil
}
func validTier(t string) bool { return t == "local" || t == "vps" || t == "test-device" }

//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }

type SQLDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type SQLStore struct{ db SQLDB }

func NewSQLStore(db SQLDB) *SQLStore { return &SQLStore{db: db} }
func (s *SQLStore) Begin(ctx context.Context, r Receipt) (Receipt, bool, error) {
	result, err := s.db.ExecContext(ctx, `INSERT INTO bridge_admin_dispatch_receipts(request_id,external_request_id,caller,scenario,operation,node_id,target_tier,status,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,'requested',$8,$8) ON CONFLICT(request_id) DO NOTHING`, r.Key, r.RequestID, r.Caller, r.Scenario, r.Operation, r.NodeID, r.Tier, r.CreatedAt)
	if err != nil {
		return Receipt{}, false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return Receipt{}, false, err
	}
	if n > 0 {
		return r, false, nil
	}
	var prior Receipt
	err = s.db.QueryRowContext(ctx, `SELECT request_id,external_request_id,caller,scenario,operation,node_id,target_tier,status,run_id,replayed,detail,created_at FROM bridge_admin_dispatch_receipts WHERE request_id=$1`, r.Key).Scan(&prior.Key, &prior.RequestID, &prior.Caller, &prior.Scenario, &prior.Operation, &prior.NodeID, &prior.Tier, &prior.Status, &prior.RunID, &prior.Replayed, &prior.Detail, &prior.CreatedAt)
	if err != nil {
		return Receipt{}, false, err
	}
	if prior.Caller != r.Caller || prior.Scenario != r.Scenario || prior.Operation != r.Operation || prior.NodeID != r.NodeID || prior.Tier != r.Tier {
		return Receipt{}, false, errors.New("request id was already used for a different caller or target")
	}
	return prior, true, nil
}
func (s *SQLStore) Finish(ctx context.Context, id, status, runID, detail string, replayed bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE bridge_admin_dispatch_receipts SET status=$2,run_id=$3,replayed=$4,detail=$5,updated_at=NOW() WHERE request_id=$1`, id, status, runID, replayed, detail)
	return err
}

func (s *SQLStore) Get(ctx context.Context, id string) (Receipt, error) {
	var receipt Receipt
	err := s.db.QueryRowContext(ctx, `SELECT request_id,external_request_id,caller,scenario,operation,node_id,target_tier,status,run_id,replayed,detail,created_at FROM bridge_admin_dispatch_receipts WHERE request_id=$1`, id).Scan(&receipt.Key, &receipt.RequestID, &receipt.Caller, &receipt.Scenario, &receipt.Operation, &receipt.NodeID, &receipt.Tier, &receipt.Status, &receipt.RunID, &receipt.Replayed, &receipt.Detail, &receipt.CreatedAt)
	return receipt, err
}
