package scenarioruntime

import (
	"context"
	"database/sql"
	"fmt"
)

const instanceSelectSQL = `
SELECT instance_id, scenario, generation, scope_path, sandbox_id, status, phase,
  started_at, updated_at, last_heartbeat_at, heartbeat_deadline_at, stopped_at,
  stop_reason, owner_kind, owner_pid, working_dir, host_boot_id, host_session_id,
  supervisor_id, supervised_at, last_reconciled_at, reconciliation_status,
  reconciliation_reason, supervision_policy, schema_version
FROM runtime_instances`

const supervisorSessionSelectSQL = `
SELECT supervisor_id, host_boot_id, host_session_id, pid, status, started_at,
  last_heartbeat_at, heartbeat_deadline_at, stopped_at, stop_reason, version, metadata_json
FROM runtime_supervisor_sessions`

const portClaimSelectSQL = `
SELECT claim_id, instance_id, scenario, port_name, env_var, port, bind_host, url,
  status, created_at, updated_at, expires_at, last_bound_at
FROM runtime_port_claims`

const processRefSelectSQL = `
SELECT ref_id, instance_id, pid, pgid, process_id, step, command, log_file,
  status, started_at, ended_at, host_boot_id
FROM runtime_process_refs`

type scanner interface {
	Scan(dest ...any) error
}

func getInstanceTx(ctx context.Context, tx *sql.Tx, instanceID string) (Instance, error) {
	return scanInstance(tx.QueryRowContext(ctx, instanceSelectSQL+` WHERE instance_id = ?`, instanceID))
}

func scanInstance(row scanner) (Instance, error) {
	var in Instance
	var startedAt, updatedAt string
	var lastHeartbeatAt, heartbeatDeadlineAt, stoppedAt, supervisedAt, lastReconciledAt sql.NullString
	var ownerPID sql.NullInt64
	err := row.Scan(
		&in.InstanceID, &in.Scenario, &in.Generation, &in.ScopePath, &in.SandboxID, &in.Status, &in.Phase,
		&startedAt, &updatedAt, &lastHeartbeatAt, &heartbeatDeadlineAt, &stoppedAt,
		&in.StopReason, &in.OwnerKind, &ownerPID, &in.WorkingDir, &in.HostBootID, &in.HostSessionID,
		&in.SupervisorID, &supervisedAt, &lastReconciledAt, &in.ReconciliationStatus,
		&in.ReconciliationReason, &in.SupervisionPolicy, &in.SchemaVersion,
	)
	if err != nil {
		return Instance{}, mapRowErr(err)
	}
	parsedStartedAt, err := parseRequiredTime(startedAt)
	if err != nil {
		return Instance{}, fmt.Errorf("parse instance started_at: %w", err)
	}
	parsedUpdatedAt, err := parseRequiredTime(updatedAt)
	if err != nil {
		return Instance{}, fmt.Errorf("parse instance updated_at: %w", err)
	}
	in.StartedAt = parsedStartedAt
	in.UpdatedAt = parsedUpdatedAt
	if in.LastHeartbeatAt, err = parseOptionalTime(lastHeartbeatAt); err != nil {
		return Instance{}, fmt.Errorf("parse instance last_heartbeat_at: %w", err)
	}
	if in.HeartbeatDeadlineAt, err = parseOptionalTime(heartbeatDeadlineAt); err != nil {
		return Instance{}, fmt.Errorf("parse instance heartbeat_deadline_at: %w", err)
	}
	if in.StoppedAt, err = parseOptionalTime(stoppedAt); err != nil {
		return Instance{}, fmt.Errorf("parse instance stopped_at: %w", err)
	}
	if in.SupervisedAt, err = parseOptionalTime(supervisedAt); err != nil {
		return Instance{}, fmt.Errorf("parse instance supervised_at: %w", err)
	}
	if in.LastReconciledAt, err = parseOptionalTime(lastReconciledAt); err != nil {
		return Instance{}, fmt.Errorf("parse instance last_reconciled_at: %w", err)
	}
	in.OwnerPID = ptrInt(ownerPID)
	return in, nil
}

func getSupervisorSessionTx(ctx context.Context, tx *sql.Tx, supervisorID string) (SupervisorSession, error) {
	return scanSupervisorSession(tx.QueryRowContext(ctx, supervisorSessionSelectSQL+` WHERE supervisor_id = ?`, supervisorID))
}

func scanSupervisorSession(row scanner) (SupervisorSession, error) {
	var session SupervisorSession
	var pid sql.NullInt64
	var startedAt, lastHeartbeatAt, heartbeatDeadlineAt string
	var stoppedAt sql.NullString
	err := row.Scan(
		&session.SupervisorID, &session.HostBootID, &session.HostSessionID, &pid, &session.Status,
		&startedAt, &lastHeartbeatAt, &heartbeatDeadlineAt, &stoppedAt, &session.StopReason,
		&session.Version, &session.MetadataJSON,
	)
	if err != nil {
		return SupervisorSession{}, mapRowErr(err)
	}
	parsedStartedAt, err := parseRequiredTime(startedAt)
	if err != nil {
		return SupervisorSession{}, fmt.Errorf("parse supervisor started_at: %w", err)
	}
	parsedLastHeartbeatAt, err := parseRequiredTime(lastHeartbeatAt)
	if err != nil {
		return SupervisorSession{}, fmt.Errorf("parse supervisor last_heartbeat_at: %w", err)
	}
	parsedHeartbeatDeadlineAt, err := parseRequiredTime(heartbeatDeadlineAt)
	if err != nil {
		return SupervisorSession{}, fmt.Errorf("parse supervisor heartbeat_deadline_at: %w", err)
	}
	session.StartedAt = parsedStartedAt
	session.LastHeartbeatAt = parsedLastHeartbeatAt
	session.HeartbeatDeadlineAt = parsedHeartbeatDeadlineAt
	if session.StoppedAt, err = parseOptionalTime(stoppedAt); err != nil {
		return SupervisorSession{}, fmt.Errorf("parse supervisor stopped_at: %w", err)
	}
	session.PID = ptrInt(pid)
	return session, nil
}

func scanSupervisorSessions(rows *sql.Rows) ([]SupervisorSession, error) {
	var out []SupervisorSession
	for rows.Next() {
		session, err := scanSupervisorSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runtime supervisor sessions: %w", err)
	}
	return out, nil
}

func scanInstances(rows *sql.Rows) ([]Instance, error) {
	var out []Instance
	for rows.Next() {
		in, err := scanInstance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runtime instances: %w", err)
	}
	return out, nil
}

func getPortClaimTx(ctx context.Context, tx *sql.Tx, claimID string) (PortClaim, error) {
	return scanPortClaim(tx.QueryRowContext(ctx, portClaimSelectSQL+` WHERE claim_id = ?`, claimID))
}

func scanPortClaim(row scanner) (PortClaim, error) {
	var claim PortClaim
	var createdAt, updatedAt string
	var expiresAt, lastBoundAt sql.NullString
	err := row.Scan(
		&claim.ClaimID, &claim.InstanceID, &claim.Scenario, &claim.PortName, &claim.EnvVar, &claim.Port,
		&claim.BindHost, &claim.URL, &claim.Status, &createdAt, &updatedAt, &expiresAt, &lastBoundAt,
	)
	if err != nil {
		return PortClaim{}, mapRowErr(err)
	}
	parsedCreatedAt, err := parseRequiredTime(createdAt)
	if err != nil {
		return PortClaim{}, fmt.Errorf("parse claim created_at: %w", err)
	}
	parsedUpdatedAt, err := parseRequiredTime(updatedAt)
	if err != nil {
		return PortClaim{}, fmt.Errorf("parse claim updated_at: %w", err)
	}
	claim.CreatedAt = parsedCreatedAt
	claim.UpdatedAt = parsedUpdatedAt
	if claim.ExpiresAt, err = parseOptionalTime(expiresAt); err != nil {
		return PortClaim{}, fmt.Errorf("parse claim expires_at: %w", err)
	}
	if claim.LastBoundAt, err = parseOptionalTime(lastBoundAt); err != nil {
		return PortClaim{}, fmt.Errorf("parse claim last_bound_at: %w", err)
	}
	return claim, nil
}

func scanPortClaims(rows *sql.Rows) ([]PortClaim, error) {
	var out []PortClaim
	for rows.Next() {
		claim, err := scanPortClaim(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, claim)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runtime port claims: %w", err)
	}
	return out, nil
}

func scanHealthSnapshot(row scanner) (HealthSnapshot, error) {
	var snapshot HealthSnapshot
	var readiness, schemaValid sql.NullBool
	var checkedAt sql.NullString
	var latencyMillis sql.NullInt64
	err := row.Scan(
		&snapshot.InstanceID, &snapshot.Scenario, &snapshot.Status, &readiness, &checkedAt,
		&latencyMillis, &snapshot.Error, &snapshot.ResponseJSON, &schemaValid,
	)
	if err != nil {
		return HealthSnapshot{}, mapRowErr(err)
	}
	var parseErr error
	if snapshot.CheckedAt, parseErr = parseOptionalTime(checkedAt); parseErr != nil {
		return HealthSnapshot{}, fmt.Errorf("parse health checked_at: %w", parseErr)
	}
	snapshot.Readiness = ptrBool(readiness)
	snapshot.LatencyMillis = ptrInt64(latencyMillis)
	snapshot.SchemaValid = ptrBool(schemaValid)
	return snapshot, nil
}

func getProcessRefTx(ctx context.Context, tx *sql.Tx, refID string) (ProcessRef, error) {
	return scanProcessRef(tx.QueryRowContext(ctx, processRefSelectSQL+` WHERE ref_id = ?`, refID))
}

func scanProcessRef(row scanner) (ProcessRef, error) {
	var ref ProcessRef
	var pid, pgid sql.NullInt64
	var startedAt string
	var endedAt sql.NullString
	err := row.Scan(
		&ref.RefID, &ref.InstanceID, &pid, &pgid, &ref.ProcessID, &ref.Step,
		&ref.Command, &ref.LogFile, &ref.Status, &startedAt, &endedAt, &ref.HostBootID,
	)
	if err != nil {
		return ProcessRef{}, mapRowErr(err)
	}
	parsedStartedAt, err := parseRequiredTime(startedAt)
	if err != nil {
		return ProcessRef{}, fmt.Errorf("parse process ref started_at: %w", err)
	}
	ref.StartedAt = parsedStartedAt
	if ref.EndedAt, err = parseOptionalTime(endedAt); err != nil {
		return ProcessRef{}, fmt.Errorf("parse process ref ended_at: %w", err)
	}
	ref.PID = ptrInt(pid)
	ref.PGID = ptrInt(pgid)
	return ref, nil
}

func scanProcessRefs(rows *sql.Rows) ([]ProcessRef, error) {
	var out []ProcessRef
	for rows.Next() {
		ref, err := scanProcessRef(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runtime process refs: %w", err)
	}
	return out, nil
}
