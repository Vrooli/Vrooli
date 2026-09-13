package supervision

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// EffortRepository hides storage from observation, authorization and delivery.
type EffortRepository interface {
	GetEffort(context.Context, string) (*pb.EffortEnrollment, *pb.EffortBoardRow, error)
	ListEfforts(context.Context, string, int) ([]*pb.EffortEnrollment, error)
	SaveEffort(context.Context, *pb.EffortEnrollment, *pb.EffortBoardRow, uint64, string, string) error
	RefreshEffortObservation(context.Context, string, uint64, *pb.EffortBoardRow) error
	ReplayEffortOperation(context.Context, string, string, proto.Message) (bool, error)
	GetEffortDirective(context.Context, string) (*pb.EffortDirective, error)
	HadUncertainEffortDelivery(context.Context, string) (bool, error)
	ListEffortDirectives(context.Context, string, string, int) ([]*pb.EffortDirective, error)
	SaveEffortDirective(context.Context, *pb.EffortDirective, uint64, string, string) error
	GetEffortDiscovery(context.Context) (*pb.EffortDiscovery, error)
	SaveEffortDiscovery(context.Context, *pb.EffortDiscovery) error
	EffortRegistryOffset(context.Context) (int, error)
	SaveEffortRegistryOffset(context.Context, int) error
	SaveEffortAssessment(context.Context, *pb.EffortAssessment, string, string) error
	LatestEffortAssessment(context.Context, string) (*pb.EffortAssessment, error)
}

var effortJSON = protojson.MarshalOptions{UseProtoNames: true}

func (r *Repository) EffortRegistryOffset(ctx context.Context) (int, error) {
	var offset int
	err := r.db.QueryRowContext(ctx, `SELECT run_offset FROM supervision_effort_registry_cursor WHERE singleton=1`).Scan(&offset)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return offset, err
}
func (r *Repository) SaveEffortRegistryOffset(ctx context.Context, offset int) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO supervision_effort_registry_cursor(singleton,run_offset) VALUES(1,?) ON CONFLICT(singleton) DO UPDATE SET run_offset=excluded.run_offset`, offset)
	return err
}

func (r *Repository) RefreshEffortObservation(ctx context.Context, ref string, revision uint64, o *pb.EffortBoardRow) error {
	b, err := effortJSON.Marshal(o)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `UPDATE supervision_efforts SET observation_json=? WHERE effort_ref=? AND revision=?`, string(b), ref, revision)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	return nil
}

func effortDigest(m proto.Message) string {
	b, _ := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	return bytesDigest(b)
}
func bytesDigest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func (r *Repository) GetEffort(ctx context.Context, ref string) (*pb.EffortEnrollment, *pb.EffortBoardRow, error) {
	var enrollment, observation string
	err := r.db.QueryRowContext(ctx, `SELECT enrollment_json,observation_json FROM supervision_efforts WHERE effort_ref=?`, ref).Scan(&enrollment, &observation)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	e, o := &pb.EffortEnrollment{}, &pb.EffortBoardRow{}
	if err = protojson.Unmarshal([]byte(enrollment), e); err != nil {
		return nil, nil, err
	}
	err = protojson.Unmarshal([]byte(observation), o)
	return e, o, err
}
func (r *Repository) ListEfforts(ctx context.Context, after string, limit int) ([]*pb.EffortEnrollment, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT enrollment_json FROM supervision_efforts WHERE effort_ref>? ORDER BY effort_ref LIMIT ?`, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*pb.EffortEnrollment
	for rows.Next() {
		var b string
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		e := &pb.EffortEnrollment{}
		if err = protojson.Unmarshal([]byte(b), e); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
func (r *Repository) ReplayEffortOperation(ctx context.Context, key, digest string, out proto.Message) (bool, error) {
	var stored, b string
	err := r.db.QueryRowContext(ctx, `SELECT request_digest,result_json FROM supervision_effort_transitions WHERE operation_key=?`, key).Scan(&stored, &b)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if stored != digest {
		return false, ErrConflict
	}
	return true, protojson.Unmarshal([]byte(b), out)
}
func (r *Repository) SaveEffort(ctx context.Context, e *pb.EffortEnrollment, o *pb.EffortBoardRow, expected uint64, key, digest string) error {
	b, err := effortJSON.Marshal(e)
	if err != nil {
		return err
	}
	ob, err := effortJSON.Marshal(o)
	if err != nil {
		return err
	}
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var result sql.Result
	if expected == 0 {
		result, err = tx.ExecContext(ctx, `INSERT INTO supervision_efforts(effort_ref,revision,enrollment_json,observation_json) VALUES(?,?,?,?) ON CONFLICT(effort_ref) DO NOTHING`, e.GetEffortRef(), e.GetRevision(), string(b), string(ob))
	} else {
		result, err = tx.ExecContext(ctx, `UPDATE supervision_efforts SET revision=?,enrollment_json=?,observation_json=? WHERE effort_ref=? AND revision=?`, e.GetRevision(), string(b), string(ob), e.GetEffortRef(), expected)
	}
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	if err = r.recordEffortTransition(ctx, tx, key, digest, b); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) recordEffortTransition(ctx context.Context, tx *sql.Tx, key, digest string, b []byte) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO supervision_effort_transitions(operation_key,request_digest,result_json,occurred_at) VALUES(?,?,?,?)`, key, digest, string(b), formatTime(r.now().UTC()))
	return err
}
func (r *Repository) GetEffortDirective(ctx context.Context, id string) (*pb.EffortDirective, error) {
	var b string
	err := r.db.QueryRowContext(ctx, `SELECT directive_json FROM supervision_effort_directives WHERE directive_id=?`, id).Scan(&b)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	d := &pb.EffortDirective{}
	err = protojson.Unmarshal([]byte(b), d)
	return d, err
}

// Legacy terminal dispositions can conceal an earlier uncertain dispatch. Read
// only this directive's bounded transition history; never infer it from prose.
func (r *Repository) HadUncertainEffortDelivery(ctx context.Context, id string) (bool, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT result_json FROM supervision_effort_transitions WHERE operation_key LIKE ? ORDER BY occurred_at LIMIT 1001`, "delivery:"+id+":%")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
		if count > 1000 {
			return false, errors.New("directive transition coverage incomplete; owner history reconciliation required")
		}
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return false, err
		}
		prior := &pb.EffortDirective{}
		if err := protojson.Unmarshal([]byte(raw), prior); err != nil {
			return false, err
		}
		if prior.DirectiveId == id && prior.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN {
			return true, nil
		}
	}
	return false, rows.Err()
}
func (r *Repository) ListEffortDirectives(ctx context.Context, ref, after string, limit int) ([]*pb.EffortDirective, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT directive_json FROM supervision_effort_directives WHERE (?='' OR effort_ref=?) AND directive_id>? ORDER BY directive_id LIMIT ?`, ref, ref, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*pb.EffortDirective
	for rows.Next() {
		var b string
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		d := &pb.EffortDirective{}
		if err = protojson.Unmarshal([]byte(b), d); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}
func (r *Repository) SaveEffortDirective(ctx context.Context, d *pb.EffortDirective, expected uint64, key, digest string) error {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	reserve := expected == 0 && d.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_PENDING
	if reserve {
		// Lock before checking the complete durable reservation set. Refused
		// requests cannot hide accepted effects behind a bounded display page.
		revision := d.GetSourceSnapshot().GetEnrollment().GetRevision()
		locked, lockErr := tx.ExecContext(ctx, `UPDATE supervision_efforts SET revision=revision WHERE effort_ref=? AND revision=?`, d.EffortRef, revision)
		if lockErr != nil {
			return lockErr
		}
		n, _ := locked.RowsAffected()
		if n != 1 {
			return ErrConflict
		}
		var raw string
		if err = tx.QueryRowContext(ctx, `SELECT enrollment_json FROM supervision_efforts WHERE effort_ref=?`, d.EffortRef).Scan(&raw); err != nil {
			return err
		}
		e := &pb.EffortEnrollment{}
		if err = protojson.Unmarshal([]byte(raw), e); err != nil {
			return err
		}
		var count int
		var latest sql.NullString
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*),MAX(reserved_at) FROM supervision_effort_directive_reservations WHERE effort_ref=?`, d.EffortRef).Scan(&count, &latest); err != nil {
			return err
		}
		reason := ""
		if e.Withdrawn || e.TargetRevision != d.TargetRevision {
			reason = "enrollment no longer authorizes request"
		}
		if e.MaximumDirectives == 0 || count >= int(e.MaximumDirectives) {
			reason = "directive allowance exhausted or unavailable"
		}
		if latest.Valid && e.CooldownSeconds > 0 {
			stamp, parseErr := time.Parse(time.RFC3339Nano, latest.String)
			if parseErr != nil {
				return parseErr
			}
			if d.CreatedAt.AsTime().Before(stamp.Add(time.Duration(e.CooldownSeconds) * time.Second)) {
				reason = "directive cooldown is active"
			}
		}
		if reason != "" {
			reserve = false
			d.Delivery = pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED
			d.DeliveryReason = reason
		}
	}
	b, err := effortJSON.Marshal(d)
	if err != nil {
		return err
	}
	var result sql.Result
	if expected == 0 {
		result, err = tx.ExecContext(ctx, `INSERT INTO supervision_effort_directives(directive_id,effort_ref,revision,directive_json) VALUES(?,?,?,?) ON CONFLICT(directive_id) DO NOTHING`, d.GetDirectiveId(), d.GetEffortRef(), d.GetRevision(), string(b))
	} else {
		result, err = tx.ExecContext(ctx, `UPDATE supervision_effort_directives SET revision=?,directive_json=? WHERE directive_id=? AND revision=?`, d.GetRevision(), string(b), d.GetDirectiveId(), expected)
	}
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	if reserve {
		if _, err = tx.ExecContext(ctx, `INSERT INTO supervision_effort_directive_reservations(directive_id,effort_ref,reserved_at) VALUES(?,?,?)`, d.DirectiveId, d.EffortRef, formatTime(d.CreatedAt.AsTime())); err != nil {
			return err
		}
	}
	if err = r.recordEffortTransition(ctx, tx, key, digest, b); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) GetEffortDiscovery(ctx context.Context) (*pb.EffortDiscovery, error) {
	var b string
	err := r.db.QueryRowContext(ctx, `SELECT discovery_json FROM supervision_effort_discovery WHERE singleton=1`).Scan(&b)
	if errors.Is(err, sql.ErrNoRows) {
		return &pb.EffortDiscovery{Partial: true, Findings: []*pb.EffortDiscoveryFinding{{Code: "not_scanned", Reason: "discovery has not completed a scan"}}}, nil
	}
	if err != nil {
		return nil, err
	}
	d := &pb.EffortDiscovery{}
	err = protojson.Unmarshal([]byte(b), d)
	return d, err
}
func (r *Repository) SaveEffortDiscovery(ctx context.Context, d *pb.EffortDiscovery) error {
	b, err := effortJSON.Marshal(d)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO supervision_effort_discovery(singleton,discovery_json) VALUES(1,?) ON CONFLICT(singleton) DO UPDATE SET discovery_json=excluded.discovery_json`, string(b))
	return err
}
