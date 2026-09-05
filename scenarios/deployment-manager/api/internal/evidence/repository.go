package evidence

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"deployment-manager/shared"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
)

// Repository is the storage seam for the evidence contract. Implementations
// persist references and verdict metadata only; producers retain the bytes.
type Repository interface {
	Save(context.Context, string, string, *commonv1.TargetVerdict) error
	List(context.Context, string, string, int) ([]*commonv1.TargetVerdict, error)
}

// SQLRepository stores the contract in two normalized tables. The dialect is
// currently either "postgres" ($1 placeholders) or "sqlite" (? placeholders).
// Keeping the placeholder choice explicit makes the storage seam portable while
// the deployment-manager re-platform proceeds.
type SQLRepository struct {
	db          shared.RoutedDBTX
	placeholder func(int) string
}

//go:embed schema.sql
var schemaSQL string

func NewSQLRepository(db shared.RoutedDBTX, dialect string) *SQLRepository {
	placeholder := func(i int) string { return fmt.Sprintf("$%d", i) }
	if strings.EqualFold(dialect, "sqlite") {
		placeholder = func(int) string { return "?" }
	}
	return &SQLRepository{db: db, placeholder: placeholder}
}

func Schema() string { return schemaSQL }

func (r *SQLRepository) Save(ctx context.Context, profileID, commit string, verdict *commonv1.TargetVerdict) error {
	if r == nil || r.db == nil {
		return errors.New("evidence repository is not configured")
	}
	if verdict == nil || verdict.Target == nil {
		return errors.New("target verdict and target are required")
	}
	if profileID == "" || commit == "" || verdict.RunId == "" {
		return errors.New("profile, commit, and run_id are required")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin evidence transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	id := uuid.NewString()
	now := time.Now().UTC()
	p := r.placeholder
	_, err = tx.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO deployment_evidence_verdicts
			(id, profile_id, git_commit_hash, target_ramp, target_platform, target_os,
			 device_kind, bridge_node_id, bridge_job_id, disposition, run_id, detail, created_at)
			VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)`,
			p(1), p(2), p(3), p(4), p(5), p(6), p(7), p(8), p(9), p(10), p(11), p(12), p(13)),
		id, profileID, commit, verdict.Target.Ramp, verdict.Target.Platform, verdict.Target.Os,
		int32(verdict.Target.DeviceKind), optionalString(verdict.Target.BridgeNodeId),
		optionalString(verdict.Target.BridgeJobId), int32(verdict.Disposition), verdict.RunId,
		verdict.Detail, now,
	)
	if err != nil {
		return fmt.Errorf("insert evidence verdict: %w", err)
	}
	for _, ref := range verdict.Refs {
		if ref == nil {
			return errors.New("evidence references cannot be nil")
		}
		_, err = tx.ExecContext(ctx,
			fmt.Sprintf(`INSERT INTO deployment_evidence_refs
				(verdict_id, producer, artifact_id, kind, checksum, size_bytes, created_at)
				VALUES (%s,%s,%s,%s,%s,%s,%s)`, p(1), p(2), p(3), p(4), p(5), p(6), p(7)),
			id, ref.Producer, ref.ArtifactId, ref.Kind, ref.Checksum, ref.SizeBytes,
			ref.CreatedAt.AsTime(),
		)
		if err != nil {
			return fmt.Errorf("insert evidence reference: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit evidence transaction: %w", err)
	}
	return nil
}

func (r *SQLRepository) List(ctx context.Context, profileID, commit string, limit int) ([]*commonv1.TargetVerdict, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("evidence repository is not configured")
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	p := r.placeholder
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT id, target_ramp, target_platform,
		target_os, device_kind, bridge_node_id, bridge_job_id, disposition, run_id, detail, created_at
		FROM deployment_evidence_verdicts WHERE profile_id = %s AND git_commit_hash = %s
		ORDER BY created_at ASC LIMIT %s`, p(1), p(2), p(3)), profileID, commit, limit)
	if err != nil {
		return nil, fmt.Errorf("list evidence verdicts: %w", err)
	}
	defer rows.Close()

	var out []*commonv1.TargetVerdict
	for rows.Next() {
		var id, ramp, platform, osName, runID, detail string
		var deviceKind, disposition int32
		var nodeID, jobID sql.NullString
		var created time.Time
		if err := rows.Scan(&id, &ramp, &platform, &osName, &deviceKind, &nodeID, &jobID,
			&disposition, &runID, &detail, &created); err != nil {
			return nil, fmt.Errorf("scan evidence verdict: %w", err)
		}
		refs, err := r.listRefs(ctx, id)
		if err != nil {
			return nil, err
		}
		target := &commonv1.EvidenceTarget{
			Ramp: ramp, Platform: platform, Os: osName,
			DeviceKind: commonv1.DeviceKind(deviceKind),
		}
		if nodeID.Valid {
			target.BridgeNodeId = &nodeID.String
		}
		if jobID.Valid {
			target.BridgeJobId = &jobID.String
		}
		out = append(out, &commonv1.TargetVerdict{
			Target: target, Disposition: commonv1.Disposition(disposition), Refs: refs,
			RunId: runID, Detail: detail,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evidence verdicts: %w", err)
	}
	return out, nil
}

func (r *SQLRepository) listRefs(ctx context.Context, verdictID string) ([]*commonv1.EvidenceRef, error) {
	p := r.placeholder
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT producer, artifact_id, kind,
		checksum, size_bytes, created_at FROM deployment_evidence_refs WHERE verdict_id = %s
		ORDER BY created_at ASC`, p(1)), verdictID)
	if err != nil {
		return nil, fmt.Errorf("list evidence references: %w", err)
	}
	defer rows.Close()
	var refs []*commonv1.EvidenceRef
	for rows.Next() {
		var producer, artifactID, kind, checksum string
		var size int64
		var created time.Time
		if err := rows.Scan(&producer, &artifactID, &kind, &checksum, &size, &created); err != nil {
			return nil, fmt.Errorf("scan evidence reference: %w", err)
		}
		refs = append(refs, &commonv1.EvidenceRef{
			Producer: producer, ArtifactId: artifactID,
			Kind: kind, Checksum: checksum, SizeBytes: size, CreatedAt: timestamppb.New(created),
		})
	}
	return refs, rows.Err()
}

func optionalString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
