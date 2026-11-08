package programs

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"program-runtime/internal/localworkbridge"
)

// ServiceAdmissionBoundary is a trusted in-process owner boundary. Within must
// hold the same serialization used by authority revocation and broker session
// invalidation until fn finishes. It must invoke fn exactly once, synchronously,
// and honor ctx. A mutex unrelated to those authorities does not meet this
// contract. No production implementation or route registration is supplied here.
type ServiceAdmissionBoundary interface {
	Within(context.Context, func(context.Context) error) error
}

type servicePermitKey struct{}
type serviceAdmission struct {
	boundary ServiceAdmissionBoundary
	deadline string
	recheck  func(context.Context) error
	identity func(string) (localworkbridge.SourceProgramIdentity, error)
	check    func(context.Context, localworkbridge.SourceProgramIdentity) error
}

// WithSourcePermit is reserved for a trusted finite owner adapter. HTTP request
// attribution cannot construct a SourcePermit. This does not verify an agent or
// grant access to generic owner routes. Callers still resolve the pinned source,
// exact inputs and one-shot session through their canonical owners.
func WithSourcePermit(ctx context.Context, permit *localworkbridge.SourcePermit, boundary ServiceAdmissionBoundary) (context.Context, error) {
	if permit == nil || boundary == nil {
		return nil, localworkbridge.ErrSourceAuthority
	}
	if err := permit.Recheck(ctx); err != nil {
		return nil, err
	}
	return context.WithValue(ctx, servicePermitKey{}, &serviceAdmission{boundary: boundary, deadline: permit.Call().AdmissionDeadline, recheck: permit.Recheck, identity: permit.AdmissionIdentity, check: permit.CheckProgram}), nil
}

func serviceAdmissionFrom(ctx context.Context) *serviceAdmission {
	a, _ := ctx.Value(servicePermitKey{}).(*serviceAdmission)
	return a
}

type serviceOwnerKey struct{}

func serviceOwnerJSON(ctx context.Context, createOnly bool) string {
	if !createOnly {
		return ""
	}
	value, _ := ctx.Value(serviceOwnerKey{}).(string)
	return value
}

func createServiceAdmission(ctx context.Context, p *programsv1.Program, create func(context.Context) (bool, error)) (bool, error) {
	a := serviceAdmissionFrom(ctx)
	if a == nil {
		return create(ctx)
	}
	owner, err := a.identity(p.GetId())
	if err != nil {
		return false, err
	}
	// Service authority never masquerades as verified agent attribution.
	if p.GetCallerIdentityVerified() || p.GetRequestDigest() == "" || owner.Name != p.GetProgramName() || owner.Digest != p.GetProgramDigest() || submissionID(Identity{ProgramName: owner.Name, IdempotencyKey: owner.RequestKey}) != p.GetId() {
		return false, localworkbridge.ErrSourceAuthority
	}
	data, err := json.Marshal(owner)
	if err != nil {
		return false, err
	}
	var created bool
	err = a.boundary.Within(ctx, func(inner context.Context) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := a.check(inner, owner); err != nil {
			return err
		}
		deadline, err := time.Parse(time.RFC3339Nano, a.deadline)
		if err != nil || !time.Now().Before(deadline) {
			return ErrRequestExpired
		}
		if err := inner.Err(); err != nil {
			return err
		}
		// Clip the actual storage call, including SQLite lock waits, to the
		// fixed admission deadline. A prior wall-clock check is insufficient.
		commitCtx, cancel := context.WithDeadline(inner, deadline)
		defer cancel()
		if err := a.check(commitCtx, owner); err != nil {
			return err
		}
		if err := commitCtx.Err(); err != nil {
			return err
		}
		created, err = create(context.WithValue(commitCtx, serviceOwnerKey{}, string(data)))
		return err
	})
	return created, err
}

// GetServiceAdmission returns only owner-retained identity with the canonical
// program row. The two reads run under the composite boundary; identity is
// immutable on Save. It is intentionally absent from the generic Repository
// interface, so unsupported repositories fail closed rather than lose ownership.
func (r *sqliteRepository) GetServiceAdmission(ctx context.Context, id string) (*programsv1.Program, localworkbridge.SourceProgramIdentity, error) {
	return r.withServiceAdmission(ctx, id, nil)
}

func (r *sqliteRepository) withServiceAdmission(ctx context.Context, id string, visit func(context.Context, *programsv1.Program, localworkbridge.SourceProgramIdentity) error) (*programsv1.Program, localworkbridge.SourceProgramIdentity, error) {
	var p *programsv1.Program
	var owner localworkbridge.SourceProgramIdentity
	a := serviceAdmissionFrom(ctx)
	if a == nil {
		return nil, owner, localworkbridge.ErrSourceAuthority
	}
	err := a.boundary.Within(ctx, func(inner context.Context) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		var err error
		p, err = r.get(inner, id)
		if err != nil {
			if errors.Is(err, ErrProgramNotFound) {
				if checkErr := a.recheck(inner); checkErr != nil {
					return checkErr
				}
			}
			return err
		}
		var raw string
		if err = r.db.QueryRowContext(inner, "SELECT service_owner_json FROM programs WHERE id = ?", id).Scan(&raw); err != nil {
			return err
		}
		if raw == "" || json.Unmarshal([]byte(raw), &owner) != nil || owner.ID != p.GetId() || owner.Name != p.GetProgramName() || owner.Digest != p.GetProgramDigest() || submissionID(Identity{ProgramName: owner.Name, IdempotencyKey: owner.RequestKey}) != id {
			return localworkbridge.ErrSourceAuthority
		}
		if err = a.check(inner, owner); err != nil {
			return err
		}
		if err := inner.Err(); err != nil {
			return err
		}
		if visit != nil {
			return visit(inner, p, owner)
		}
		return nil
	})
	if err != nil {
		return nil, localworkbridge.SourceProgramIdentity{}, err
	}
	return p, owner, nil
}

// SourceProgram reads durable ownership rather than synthesizing a response
// identity from a caller or permit. There is no in-memory ownership fallback.
func (s *Service) SourceProgram(ctx context.Context, id string) (localworkbridge.SourceBrokerProgram, error) {
	r, ok := s.repo.(*sqliteRepository)
	if !ok {
		return localworkbridge.SourceBrokerProgram{}, localworkbridge.ErrSourceAuthority
	}
	p, owner, err := r.GetServiceAdmission(ctx, id)
	if errors.Is(err, ErrProgramNotFound) {
		return localworkbridge.SourceBrokerProgram{}, nil
	}
	if err != nil {
		return localworkbridge.SourceBrokerProgram{}, err
	}
	return sourceProgramResult(p, owner)
}

// SourceAdmissionID uses the same existing durable execution identity as normal
// declared admission. An ID identifies a row; it is never read authority.
func SourceAdmissionID(key string) string {
	return submissionID(Identity{ProgramName: localworkbridge.SourceCollectorName, IdempotencyKey: key})
}

// WithSourceProgram keeps the trusted authority boundary held during an owned
// read and synchronous callback. Callbacks must not re-enter this boundary.
// Missing rows are reported as ErrProgramNotFound, never terminal evidence.
func (s *Service) WithSourceProgram(ctx context.Context, id string, fn func(context.Context, localworkbridge.SourceBrokerProgram) error) error {
	r, ok := s.repo.(*sqliteRepository)
	if !ok || fn == nil {
		return localworkbridge.ErrSourceAuthority
	}
	_, _, err := r.withServiceAdmission(ctx, id, func(inner context.Context, p *programsv1.Program, owner localworkbridge.SourceProgramIdentity) error {
		result, err := sourceProgramResult(p, owner)
		if err != nil {
			return err
		}
		return fn(inner, result)
	})
	return err
}

func sourceProgramResult(p *programsv1.Program, owner localworkbridge.SourceProgramIdentity) (localworkbridge.SourceBrokerProgram, error) {
	result := localworkbridge.SourceBrokerProgram{Found: true, Identity: owner, Status: p.GetStatus().String()}
	if p.GetStatus() == programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED {
		if len(p.GetStdout()) > 65536 {
			return localworkbridge.SourceBrokerProgram{}, localworkbridge.ErrSourceAuthority
		}
		result.Stdout = p.GetStdout()
	}
	return result, nil
}

// ServiceRecoveryBoundary is a trusted lifecycle owner seam, absent by default.
// WithStoppedExecution must establish that all prior execution of this exact
// durable ID has stopped and joined, and exclude replacement execution through
// fn. It invokes fn once synchronously, respecting context. A caller assertion,
// mutex unrelated to execution, or source grant does not establish this proof.
// Lock order is stopped execution, source authority, then session invalidation.
type ServiceRecoveryBoundary interface {
	WithStoppedExecution(context.Context, string, func(context.Context) error) error
}

// RecoverSourceAfterStop interrupts only the permit-owned durable admission.
// Recovery is explicit and requires the lifecycle owner's stop/join boundary.
// No construction path invokes it and no row is retried or re-executed.
func (s *Service) RecoverSourceAfterStop(ctx context.Context, id string, stopped ServiceRecoveryBoundary, cleanup func(context.Context, string) error) error {
	r, ok := s.repo.(*sqliteRepository)
	if !ok || stopped == nil || cleanup == nil {
		return localworkbridge.ErrSourceAuthority
	}
	return stopped.WithStoppedExecution(ctx, id, func(recovery context.Context) error {
		sessionID := ""
		_, _, err := r.withServiceAdmission(recovery, id, func(inner context.Context, p *programsv1.Program, _ localworkbridge.SourceProgramIdentity) error {
			sessionID = p.GetSessionId()
			if IsTerminal(p.GetStatus()) {
				return nil
			}
			if p.GetStatus() != programsv1.ProgramStatus_PROGRAM_STATUS_ACCEPTED && p.GetStatus() != programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING {
				return localworkbridge.ErrSourceAuthority
			}
			_, err := r.db.ExecContext(inner, `UPDATE programs SET status='failed', completed_at=?, failure_shape='runtime_interrupted', failure_cause=?, failure_detail=?, failure_detail_receipt_format=0 WHERE id=? AND status IN ('accepted','running')`, s.clock().UTC().Format(time.RFC3339Nano), programsv1.FailureCause_FAILURE_CAUSE_RUNTIME_INTERRUPTED.String(), interruptedDetail, id)
			return err
		})
		if err != nil {
			return err
		}
		if sessionID != "" {
			return cleanup(recovery, sessionID)
		}
		return nil
	})
}
