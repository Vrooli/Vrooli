package programs

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

var ErrRequestConflict = errors.New("declared program request key is bound to different intent")
var ErrInvalidRequestKey = errors.New("invalid declared program request identity")
var ErrRequestExpired = errors.New("declared program admission deadline expired; inspect owner state, do not renew the same attempt")

// Retention MUST preserve keyed records for at least this long. A caller pins
// the deadline before its first request; retry cannot renew admission authority.
const DeclaredAdmissionWindow = 24 * time.Hour

func submissionID(identity Identity) string {
	if identity.IdempotencyKey == "" {
		return "prog_" + uuid.NewString()
	}
	return "prog_" + uuid.NewSHA1(uuid.NameSpaceURL, []byte("declared-program\x00"+identity.ProgramName+"\x00"+identity.IdempotencyKey)).String()
}

// GetDeclaredExecution observes an admission without allocating a session or
// executing code. Missing evidence is distinct from an expired retry refusal.
func (s *Service) GetDeclaredExecution(ctx context.Context, name, key string) (*programsv1.Program, error) {
	if name == "" || strings.TrimSpace(name) != name || key == "" || len(key) > 128 || strings.TrimSpace(key) != key || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return nil, ErrInvalidRequestKey
	}
	return s.repo.Get(ctx, submissionID(Identity{ProgramName: name, IdempotencyKey: key}))
}

// CloseDeclaredAdmission races on the same unique program row as submission.
// If closure wins, the terminal record prevents even an in-flight submission
// from executing. If submission wins, its original handle must be drained; a
// closure response does not claim that already-admitted effects have stopped.
// No session, runner, second ledger or clock-based absence inference is used.
func (s *Service) CloseDeclaredAdmission(ctx context.Context, source string, provenance programsv1.Provenance, includeMaterialized bool, identity Identity, caller Caller) (*programsv1.Program, error) {
	if identity.IdempotencyKey == "" {
		return nil, ErrInvalidRequestKey
	}
	existing, err := s.ReplayDeclared(ctx, source, provenance, includeMaterialized, identity, caller)
	if existing != nil || (err != nil && !errors.Is(err, ErrRequestExpired)) {
		return existing, err
	}
	now := s.clock().UTC().Format(time.RFC3339Nano)
	p := &programsv1.Program{Id: submissionID(identity), Source: source, Provenance: provenance, CreatedAt: now, CompletedAt: now, Status: programsv1.ProgramStatus_PROGRAM_STATUS_CANCELLED, ProgramName: identity.ProgramName, ProgramDigest: identity.ProgramDigest, CallerRunId: caller.RunID, CallerAgentProfile: caller.AgentProfile, CallerSkillId: caller.SkillID, CallerHarness: caller.Harness, FailureShape: "admission_closed", FailureDetail: "Admission closed before execution; no session or effects were started", RequestDigest: declaredRequestDigest(source, provenance, includeMaterialized, identity, caller)}
	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	if created {
		s.notifyTerminal(p.Id)
		return clone(p), nil
	}
	return s.ReplayDeclared(ctx, source, provenance, includeMaterialized, identity, caller)
}

// ReplayDeclared reads the original execution before allocating a new session.
// It intentionally includes failed/interrupted results: observation never replays
// domain effects. Source includes the resolved input preamble and pinned program.
func (s *Service) ReplayDeclared(ctx context.Context, source string, provenance programsv1.Provenance, includeMaterialized bool, identity Identity, caller Caller) (*programsv1.Program, error) {
	if identity.IdempotencyKey == "" {
		if identity.AdmissionDeadline != "" {
			return nil, fmt.Errorf("%w: admission_deadline requires idempotency_key", ErrInvalidRequestKey)
		}
		return nil, nil
	}
	if len(identity.IdempotencyKey) > 128 || strings.TrimSpace(identity.IdempotencyKey) != identity.IdempotencyKey || strings.IndexFunc(identity.IdempotencyKey, unicode.IsControl) >= 0 || identity.ProgramName == "" || identity.ProgramDigest == "" {
		return nil, fmt.Errorf("%w: key requires a pinned declared program and 1..128 non-padded bytes without control characters", ErrInvalidRequestKey)
	}
	deadline, err := time.Parse(time.RFC3339Nano, identity.AdmissionDeadline)
	if err != nil {
		return nil, fmt.Errorf("%w: admission_deadline must be RFC3339", ErrInvalidRequestKey)
	}
	p, err := s.repo.Get(ctx, submissionID(identity))
	if errors.Is(err, ErrProgramNotFound) {
		return nil, s.checkAdmissionDeadline(deadline)
	}
	if err != nil {
		return nil, err
	}
	if p.RequestDigest != declaredRequestDigest(source, provenance, includeMaterialized, identity, caller) {
		return nil, ErrRequestConflict
	}
	return p, nil
}

func (s *Service) checkAdmissionDeadline(deadline time.Time) error {
	now := s.clock()
	if !deadline.After(now) {
		return ErrRequestExpired
	}
	if deadline.After(now.Add(DeclaredAdmissionWindow)) {
		return fmt.Errorf("%w: admission_deadline must be within 24h", ErrInvalidRequestKey)
	}
	return nil
}

// Store a digest rather than comparing historical source. Bearer receipts in
// source are deliberately unreadable after restart; matching intent must not
// depend on recovering that authority or retaining its plaintext.
func declaredRequestDigest(source string, provenance programsv1.Provenance, includeMaterialized bool, identity Identity, caller Caller) string {
	deadline, _ := time.Parse(time.RFC3339Nano, identity.AdmissionDeadline)
	// Keep this persisted format explicit: adding unrelated struct fields must
	// not invalidate an already admitted request after an upgrade.
	intent := []any{"declared-request-v1", identity.ProgramName, identity.ProgramDigest, identity.IdempotencyKey, deadline.UTC().Format(time.RFC3339Nano), source, int32(provenance), includeMaterialized, caller.RunID, caller.AgentProfile, caller.SkillID, caller.Harness}
	if len(identity.Grants) > 0 {
		grants := slices.Clone(identity.Grants)
		slices.Sort(grants)
		intent[0] = "declared-request-v2"
		intent = append(intent, slices.Compact(grants))
	}
	encoded, _ := json.Marshal(intent)
	return fmt.Sprintf("%x", sha256.Sum256(encoded))
}

func (s *Service) admitProgram(ctx context.Context, p *programsv1.Program, includeMaterialized bool, identity Identity, caller Caller) (*programsv1.Program, bool, error) {
	if identity.IdempotencyKey != "" {
		// Preflight may outlive admission. Recheck before the only atomic insert.
		if replay, err := s.ReplayDeclared(ctx, p.Source, p.Provenance, includeMaterialized, identity, caller); err != nil || replay != nil {
			return replay, false, err
		}
		p.RequestDigest = declaredRequestDigest(p.Source, p.Provenance, includeMaterialized, identity, caller)
	}
	created, err := s.repo.Create(ctx, p)
	if err != nil || created {
		return p, created, err
	}
	existing, err := s.ReplayDeclared(ctx, p.Source, p.Provenance, includeMaterialized, identity, caller)
	if err == nil && existing == nil {
		err = fmt.Errorf("existing program admission is unavailable")
	}
	return existing, false, err
}
