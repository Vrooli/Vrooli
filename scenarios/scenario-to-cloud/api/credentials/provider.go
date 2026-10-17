package credentials

import (
	"context"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/domain"
)

// PrepareRequest asks a provider to make a new version acceptable.
type PrepareRequest struct {
	Target      Target
	Binding     domain.CredentialBinding
	Version     domain.CredentialVersion
	Value       string
	OperationID string
}

// PrepareResult reports how the provider accepted the version.
type PrepareResult struct {
	// DualAccept is true when the provider honours both the predecessor and
	// the new version until the predecessor is revoked. PostgreSQL does not:
	// its rollout is a maintenance ordering documented in credential-lifecycle.md.
	DualAccept bool
	Details    map[string]any
	// ExpiresAt is optional provider evidence for the new version. Providers
	// that do not expose expiry leave it nil; the cloud owner never invents a
	// deadline.
	ExpiresAt *time.Time
}

// VerifyRequest asks a provider to confirm consumers use the new version.
type VerifyRequest struct {
	Target    Target
	Binding   domain.CredentialBinding
	Version   domain.CredentialVersion
	Consumers []string
	Acks      []domain.CredentialAck
}

// VerifyResult names which consumers verified and which did not.
type VerifyResult struct {
	Verified    []string
	Unverified  map[string]string
	Unreachable []string
	Details     map[string]any
}

// RevokePredecessorRequest retires the version being replaced.
type RevokePredecessorRequest struct {
	Target      Target
	Binding     domain.CredentialBinding
	Predecessor domain.CredentialVersion
	Successor   domain.CredentialVersion
	OperationID string
}

// RevokePredecessorResult is either done, deferred until a time, or handed to
// an operator with a durable reference.
type RevokePredecessorResult struct {
	Done        bool
	ResumeAfter *time.Time
	Handoff     *domain.OperatorHandoff
	Details     map[string]any
}

// Provider is the per-class hook set the lifecycle drives.
type Provider interface {
	Class() domain.CredentialClass
	Prepare(ctx context.Context, req PrepareRequest) (PrepareResult, error)
	Verify(ctx context.Context, req VerifyRequest) (VerifyResult, error)
	RevokePredecessor(ctx context.Context, req RevokePredecessorRequest) (RevokePredecessorResult, error)
}

// RotationRefusal is returned by a provider that must not rotate locally.
type RotationRefusal struct {
	Reason string
	Owner  string
}

func (r *RotationRefusal) Error() string { return r.Reason }

// ackVerify is the shared consumer-acknowledgement rule: every consumer must
// have acknowledged exactly the new version.
func ackVerify(req VerifyRequest) VerifyResult {
	acked := map[string]int64{}
	for _, ack := range req.Acks {
		acked[ack.Consumer] = ack.Version
	}
	out := VerifyResult{Unverified: map[string]string{}}
	for _, consumer := range req.Consumers {
		version, ok := acked[consumer]
		switch {
		case !ok:
			out.Unverified[consumer] = "no acknowledgement recorded"
			out.Unreachable = append(out.Unreachable, consumer)
		case version != req.Version.Number:
			out.Unverified[consumer] = fmt.Sprintf("acknowledged version %d, expected %d", version, req.Version.Number)
			out.Unreachable = append(out.Unreachable, consumer)
		default:
			out.Verified = append(out.Verified, consumer)
		}
	}
	return out
}

// DatabaseOwner is the declared owner argv for the database resource. The
// argv reads SQL on standard input; the SQL carries a SCRAM verifier, not the
// password.
type DatabaseOwner struct {
	Argv []string
	Role string
}

// DefaultDatabaseOwner is the postgres resource's owner command.
func DefaultDatabaseOwner(role string) DatabaseOwner {
	if strings.TrimSpace(role) == "" {
		role = "postgres"
	}
	return DatabaseOwner{Argv: []string{"psql", "-X", "-q", "-v", "ON_ERROR_STOP=1", "-d", "postgres", "-f", "-"}, Role: role}
}

// DatabasePasswordProvider updates the database role through the owner argv
// on the target, then relies on consumer restart and acknowledgement.
type DatabasePasswordProvider struct {
	Runner ArgvRunner
	Label  string
	Owner  func(binding domain.CredentialBinding) DatabaseOwner
	// Salt is a test seam; nil means a random salt per rotation.
	Salt []byte
}

func (p *DatabasePasswordProvider) Class() domain.CredentialClass {
	return domain.CredentialClassGeneratedDatabasePassword
}

func (p *DatabasePasswordProvider) Prepare(ctx context.Context, req PrepareRequest) (PrepareResult, error) {
	owner := DefaultDatabaseOwner("")
	if p.Owner != nil {
		owner = p.Owner(req.Binding)
	}
	if strings.ContainsAny(owner.Role, `" ;`) {
		return PrepareResult{}, fmt.Errorf("database role %q is not a plain identifier", owner.Role)
	}
	verifier, err := ScramSHA256Verifier(req.Value, p.Salt)
	if err != nil {
		return PrepareResult{}, err
	}
	sql := fmt.Sprintf("ALTER ROLE \"%s\" WITH PASSWORD '%s';\n", owner.Role, verifier)
	if _, err := p.Runner.Run(ctx, p.Label, owner.Argv, strings.NewReader(sql)); err != nil {
		return PrepareResult{}, fmt.Errorf("update database role %s: %w", owner.Role, err)
	}
	return PrepareResult{DualAccept: false, Details: map[string]any{"role": owner.Role, "verifier": "scram-sha-256", "ordering": "maintenance"}}, nil
}

func (p *DatabasePasswordProvider) Verify(_ context.Context, req VerifyRequest) (VerifyResult, error) {
	return ackVerify(req), nil
}

func (p *DatabasePasswordProvider) RevokePredecessor(_ context.Context, _ RevokePredecessorRequest) (RevokePredecessorResult, error) {
	// ALTER ROLE replaced the verifier; the predecessor stopped working at
	// Prepare. Nothing further is needed at the provider.
	return RevokePredecessorResult{Done: true, Details: map[string]any{"predecessor": "replaced_at_prepare"}}, nil
}

// ExternalAPIProvider validates scope with a declared probe and revokes the
// predecessor only when the provider declares an API for it.
type ExternalAPIProvider struct {
	Name string
	// Probe validates the new value's scope. Required.
	Probe func(ctx context.Context, value string) error
	// RevokeAPI revokes a predecessor version at the provider; nil declares
	// that the provider has no revocation API and hands off to an operator.
	RevokeAPI func(ctx context.Context, binding domain.CredentialBinding, predecessor domain.CredentialVersion) error
	// Instruction is the operator handoff text when RevokeAPI is nil.
	Instruction string
}

func (p *ExternalAPIProvider) Class() domain.CredentialClass {
	return domain.CredentialClassExternalAPICredential
}

func (p *ExternalAPIProvider) Prepare(ctx context.Context, req PrepareRequest) (PrepareResult, error) {
	if p.Probe == nil {
		return PrepareResult{}, fmt.Errorf("external credential %s declares no scope probe; refusing to distribute an unvalidated value", p.Name)
	}
	if err := p.Probe(ctx, req.Value); err != nil {
		return PrepareResult{}, fmt.Errorf("scope probe rejected the new credential: %w", err)
	}
	return PrepareResult{DualAccept: true, Details: map[string]any{"probe": "passed", "provider": p.Name}}, nil
}

func (p *ExternalAPIProvider) Verify(_ context.Context, req VerifyRequest) (VerifyResult, error) {
	return ackVerify(req), nil
}

func (p *ExternalAPIProvider) RevokePredecessor(ctx context.Context, req RevokePredecessorRequest) (RevokePredecessorResult, error) {
	if p.RevokeAPI == nil {
		instruction := p.Instruction
		if instruction == "" {
			instruction = "revoke the predecessor credential in the provider console, then resume this rotation"
		}
		return RevokePredecessorResult{Handoff: &domain.OperatorHandoff{Reference: req.OperationID + "/revoke-predecessor", Provider: p.Name, Instruction: instruction, ResumeWith: "POST /api/v1/deployments/{id}/credentials/rotations/" + req.OperationID + "/resume {\"operator_confirmed\":true}"}}, nil
	}
	if err := p.RevokeAPI(ctx, req.Binding, req.Predecessor); err != nil {
		return RevokePredecessorResult{}, err
	}
	return RevokePredecessorResult{Done: true, Details: map[string]any{"provider": p.Name, "revoked_version": req.Predecessor.Number}}, nil
}

// SigningKeyProvider keeps the predecessor verifiable for an overlap window
// so tokens signed before the rotation still verify.
type SigningKeyProvider struct {
	OverlapWindow time.Duration
	Now           func() time.Time
}

func (p *SigningKeyProvider) Class() domain.CredentialClass { return domain.CredentialClassSigningKey }

func (p *SigningKeyProvider) Prepare(_ context.Context, _ PrepareRequest) (PrepareResult, error) {
	return PrepareResult{DualAccept: true, Details: map[string]any{"overlap_window": p.OverlapWindow.String()}}, nil
}

func (p *SigningKeyProvider) Verify(_ context.Context, req VerifyRequest) (VerifyResult, error) {
	return ackVerify(req), nil
}

func (p *SigningKeyProvider) RevokePredecessor(_ context.Context, req RevokePredecessorRequest) (RevokePredecessorResult, error) {
	now := time.Now().UTC()
	if p.Now != nil {
		now = p.Now()
	}
	retireAt := req.Successor.CreatedAt.Add(p.OverlapWindow)
	if now.Before(retireAt) {
		return RevokePredecessorResult{ResumeAfter: &retireAt, Details: map[string]any{"reason": "verification_overlap"}}, nil
	}
	return RevokePredecessorResult{Done: true, Details: map[string]any{"overlap_elapsed": true}}, nil
}

// MachineEnrollmentProvider refuses local rotation: enrollment identity is
// owned by Bridge and rotated/revoked there.
type MachineEnrollmentProvider struct{}

func (MachineEnrollmentProvider) Class() domain.CredentialClass {
	return domain.CredentialClassMachineEnrollment
}

func (MachineEnrollmentProvider) Prepare(context.Context, PrepareRequest) (PrepareResult, error) {
	return PrepareResult{}, &RotationRefusal{Reason: "machine enrollment credentials are owned by vrooli-bridge; rotate or revoke the node identity there", Owner: "vrooli-bridge"}
}

func (MachineEnrollmentProvider) Verify(context.Context, VerifyRequest) (VerifyResult, error) {
	return VerifyResult{}, &RotationRefusal{Reason: "machine enrollment credentials are owned by vrooli-bridge", Owner: "vrooli-bridge"}
}

func (MachineEnrollmentProvider) RevokePredecessor(context.Context, RevokePredecessorRequest) (RevokePredecessorResult, error) {
	return RevokePredecessorResult{}, &RotationRefusal{Reason: "machine enrollment credentials are owned by vrooli-bridge", Owner: "vrooli-bridge"}
}

// EncryptionRecoveryKeyProvider retires a predecessor only after a rewrap
// and a restore against the recovery point that references the new version.
type EncryptionRecoveryKeyProvider struct {
	// RewrapAndRestore rewraps the recovery point under the new key version
	// and proves a restore; it receives the version reference only.
	RewrapAndRestore func(ctx context.Context, ref domain.CredentialVersionRef) error
}

func (p *EncryptionRecoveryKeyProvider) Class() domain.CredentialClass {
	return domain.CredentialClassEncryptionRecoveryKey
}

func (p *EncryptionRecoveryKeyProvider) Prepare(_ context.Context, _ PrepareRequest) (PrepareResult, error) {
	return PrepareResult{DualAccept: true}, nil
}

func (p *EncryptionRecoveryKeyProvider) Verify(ctx context.Context, req VerifyRequest) (VerifyResult, error) {
	if p.RewrapAndRestore == nil {
		return VerifyResult{}, fmt.Errorf("no rewrap/restore verifier is declared; refusing to retire recovery key material")
	}
	if err := p.RewrapAndRestore(ctx, domain.CredentialVersionRef{BindingID: req.Binding.ID, Version: req.Version.Number}); err != nil {
		return VerifyResult{}, fmt.Errorf("rewrap and restore under the new key failed: %w", err)
	}
	result := ackVerify(req)
	if result.Details == nil {
		result.Details = map[string]any{}
	}
	result.Details["rewrap_restore"] = "verified"
	return result, nil
}

func (p *EncryptionRecoveryKeyProvider) RevokePredecessor(_ context.Context, _ RevokePredecessorRequest) (RevokePredecessorResult, error) {
	return RevokePredecessorResult{Done: true}, nil
}

// SharedDependencyProvider has no provider-side preparation; completion is
// entirely the consumers' acknowledgement of the version.
type SharedDependencyProvider struct{}

func (SharedDependencyProvider) Class() domain.CredentialClass {
	return domain.CredentialClassSharedDependency
}

func (SharedDependencyProvider) Prepare(context.Context, PrepareRequest) (PrepareResult, error) {
	return PrepareResult{DualAccept: false}, nil
}

func (SharedDependencyProvider) Verify(_ context.Context, req VerifyRequest) (VerifyResult, error) {
	return ackVerify(req), nil
}

func (SharedDependencyProvider) RevokePredecessor(context.Context, RevokePredecessorRequest) (RevokePredecessorResult, error) {
	return RevokePredecessorResult{Done: true}, nil
}

// Providers is the class → hook registry.
type Providers map[domain.CredentialClass]Provider

// DefaultProviders wires the hooks that need no external declaration and
// leaves external API and recovery-key providers to be declared explicitly.
func DefaultProviders(runner ArgvRunner, label string) Providers {
	return Providers{
		domain.CredentialClassGeneratedDatabasePassword: &DatabasePasswordProvider{Runner: runner, Label: label},
		domain.CredentialClassSigningKey:                &SigningKeyProvider{OverlapWindow: 24 * time.Hour},
		domain.CredentialClassMachineEnrollment:         MachineEnrollmentProvider{},
		domain.CredentialClassSharedDependency:          SharedDependencyProvider{},
	}
}
