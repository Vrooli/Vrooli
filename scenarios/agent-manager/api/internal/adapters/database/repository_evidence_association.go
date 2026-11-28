package database

// This owner is deliberately unregistered. No production route, grant or adapter
// composes it. Associations are evidence only, never Work identity or authority.
import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"agent-manager/internal/runmodel"
	"github.com/vrooli/api-core/eventbus"
)

type AssociationNativeLocator struct {
	Schema                    string `json:"schema"`
	OwnerID                   string `json:"owner_id"`
	RunID                     string `json:"run_id"`
	SourceInstance            string `json:"source_instance"`
	SourceEpoch               string `json:"source_epoch"`
	LifecycleVersion          string `json:"lifecycle_version"`
	TerminalObservationSHA256 string `json:"terminal_observation_sha256"`
}
type EvidenceAssociationProposal struct {
	Schema               string                    `json:"schema"`
	AssociationID        string                    `json:"association_id"`
	OwnerID              string                    `json:"owner_id"`
	EnvironmentID        string                    `json:"environment_id"`
	WorkID               string                    `json:"work_id"`
	WorkVersion          int64                     `json:"work_version"`
	ScopeVersion         string                    `json:"scope_version"`
	ComponentID          string                    `json:"component_id"`
	OperationID          string                    `json:"operation_id"`
	ExecutorReference    TerminalEvidenceReference `json:"executor_reference"`
	Native               AssociationNativeLocator  `json:"native"`
	TerminalEventID      string                    `json:"terminal_event_id"`
	CanonicalEventSHA256 string                    `json:"canonical_event_sha256"`
	Purpose              string                    `json:"purpose"`
	Bridge               struct {
		SourceInstance string `json:"source_instance"`
		ScopeEpoch     string `json:"scope_epoch"`
		ScopeDigest    string `json:"scope_digest"`
	} `json:"bridge"`
}
type EvidenceAssociationAcknowledgment struct {
	Schema                    string `json:"schema"`
	AssociationID             string `json:"association_id"`
	ProposalSHA256            string `json:"proposal_sha256"`
	OwnerID                   string `json:"owner_id"`
	SourceInstance            string `json:"source_instance"`
	SourceEpoch               string `json:"source_epoch"`
	RunID                     string `json:"run_id"`
	LifecycleVersion          string `json:"lifecycle_version"`
	TerminalObservationSHA256 string `json:"terminal_observation_sha256"`
	TerminalEventID           string `json:"terminal_event_id"`
	CanonicalEventSHA256      string `json:"canonical_event_sha256"`
	NativeRevision            int    `json:"native_revision"`
	EffectAuthority           string `json:"effect_authority"`
}

// Authority is trusted composition code, not a caller assertion. It must check
// the entire proposal's exact owner, Work, scope, component, operation, executor,
// bridge and native identity against independently held current authorization.
// WithAuthority MUST serialize this exact proposal/action's current authority
// and its revocation across the synchronous callback, including local DB COMMIT.
// The callback must run exactly once or not at all on denial. recheck checks the
// same held authority without reacquiring its lock. A boolean pre/post check is
// NOT an implementation of this boundary. No default production gate is supplied.
// This covers only native local commit, never a distributed transaction.
type EvidenceAssociationAuthority interface {
	WithAuthority(context.Context, EvidenceAssociationProposal, string, func(recheck func() error) error) error
}

type EvidenceAssociationOwner struct {
	runs      *runRepository
	authority EvidenceAssociationAuthority
}

func NewEvidenceAssociationOwner(ctx context.Context, repos *Repositories, authority EvidenceAssociationAuthority) (*EvidenceAssociationOwner, error) {
	if ctx == nil || authority == nil {
		return nil, errors.New("trusted association authority required")
	}
	reader, err := NewTerminalEvidenceReader(repos, func(context.Context, TerminalEvidenceSelector) error { return nil })
	if err != nil {
		return nil, err
	}
	// Bounded additive ledger on the existing owner DB, not a Work registry.
	_, err = reader.runs.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS run_evidence_associations (owner_id TEXT NOT NULL, association_id TEXT NOT NULL, proposal_json TEXT NOT NULL, proposal_sha256 TEXT NOT NULL, acknowledgment_json TEXT NOT NULL, revision INTEGER NOT NULL CHECK(revision IN (1,2)), state TEXT NOT NULL CHECK(state IN ('acknowledged','revoked')), PRIMARY KEY (owner_id,association_id))`)
	if err != nil {
		return nil, err
	}
	for _, statement := range []string{
		`CREATE TRIGGER IF NOT EXISTS run_evidence_associations_immutable BEFORE UPDATE ON run_evidence_associations WHEN NEW.owner_id IS NOT OLD.owner_id OR NEW.association_id IS NOT OLD.association_id OR NEW.proposal_json IS NOT OLD.proposal_json OR NEW.proposal_sha256 IS NOT OLD.proposal_sha256 OR NEW.acknowledgment_json IS NOT OLD.acknowledgment_json OR NOT (OLD.revision=1 AND OLD.state='acknowledged' AND NEW.revision=2 AND NEW.state='revoked') BEGIN SELECT RAISE(ABORT,'immutable evidence association'); END`,
		`CREATE TRIGGER IF NOT EXISTS run_evidence_associations_no_delete BEFORE DELETE ON run_evidence_associations BEGIN SELECT RAISE(ABORT,'immutable evidence association history'); END`,
		`CREATE TRIGGER IF NOT EXISTS run_evidence_associations_no_replace BEFORE INSERT ON run_evidence_associations WHEN EXISTS(SELECT 1 FROM run_evidence_associations WHERE owner_id=NEW.owner_id AND association_id=NEW.association_id AND (proposal_json IS NOT NEW.proposal_json OR proposal_sha256 IS NOT NEW.proposal_sha256 OR acknowledgment_json IS NOT NEW.acknowledgment_json OR revision IS NOT NEW.revision OR state IS NOT NEW.state)) BEGIN SELECT RAISE(ABORT,'conflicting evidence association'); END`,
	} {
		if _, err = reader.runs.db.ExecContext(ctx, statement); err != nil {
			return nil, err
		}
	}
	return &EvidenceAssociationOwner{reader.runs, authority}, nil
}

var associationScopedID = regexp.MustCompile(`^[A-Za-z0-9_-]{3,100}$`)
var associationToken = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$`)

func (p EvidenceAssociationProposal) selector() TerminalEvidenceSelector {
	return TerminalEvidenceSelector{Schema: p.Native.Schema, OwnerID: p.Native.OwnerID, SourceInstance: p.Native.SourceInstance, SourceEpoch: p.Native.SourceEpoch, RunID: p.Native.RunID, LifecycleVersion: p.Native.LifecycleVersion, TerminalObservationSHA256: p.Native.TerminalObservationSHA256, TerminalEventID: p.TerminalEventID, CanonicalEventSHA256: p.CanonicalEventSHA256}
}

// CanonicalAssociationJSON uses recursively sorted keys and JSON.stringify
// compatible escaping. Proposal values are ASCII strings; no number ambiguity.
func CanonicalAssociationJSON(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	var obj any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&obj); err != nil {
		return "", err
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err = enc.Encode(obj); err != nil {
		return "", err
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}
func (p EvidenceAssociationProposal) canonical() (string, string, error) {
	for _, v := range []string{p.AssociationID, p.OwnerID, p.EnvironmentID, p.WorkID, p.ScopeVersion, p.ComponentID, p.OperationID, p.ExecutorReference.Version, p.Bridge.SourceInstance, p.Bridge.ScopeEpoch} {
		if !associationToken.MatchString(v) {
			return "", "", errors.New("bounded association identity required")
		}
	}
	for _, v := range []string{p.AssociationID, p.WorkID, p.ComponentID, p.OperationID} {
		if !associationScopedID.MatchString(v) {
			return "", "", errors.New("bounded scoped association ID required")
		}
	}
	for _, v := range []string{p.EnvironmentID, p.ScopeVersion, p.ExecutorReference.Version} {
		if len(v) > 200 {
			return "", "", errors.New("association field exceeds 200 bytes")
		}
	}
	if len(p.Bridge.SourceInstance) > 64 || len(p.Bridge.ScopeEpoch) > 64 {
		return "", "", errors.New("bridge scope exceeds 64 bytes")
	}
	if p.WorkVersion < 1 || p.WorkVersion > 9007199254740991 {
		return "", "", errors.New("safe positive Work version required")
	}
	if p.Schema != "flywheel.native-evidence-association-proposal.v1" || p.Purpose != "evidence-association-only" || !terminalEvidenceDigest.MatchString(p.Bridge.ScopeDigest) {
		return "", "", errors.New("invalid association contract")
	}
	u, err := url.Parse(p.ExecutorReference.URI)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" || len(p.ExecutorReference.URI) > 2000 {
		return "", "", errors.New("exact HTTPS executor reference required")
	}
	for _, c := range p.ExecutorReference.URI {
		if c < 33 || c > 126 {
			return "", "", errors.New("executor reference must be bounded ASCII")
		}
	}
	s, err := CanonicalAssociationJSON(p)
	return s, runmodel.SHA256([]byte(s)), err
}

// Decode requires canonical bytes: this rejects duplicates, unknown properties,
// trailing values, case aliases and ambiguous alternative encodings.
func DecodeEvidenceAssociationProposal(raw string) (EvidenceAssociationProposal, error) {
	var p EvidenceAssociationProposal
	d := json.NewDecoder(strings.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return p, err
	}
	if d.Decode(new(any)) != io.EOF {
		return p, errors.New("trailing proposal data")
	}
	canonical, _, err := p.canonical()
	if err != nil {
		return p, err
	}
	if canonical != raw {
		return p, errors.New("canonical proposal bytes required")
	}
	return p, nil
}
func (o *EvidenceAssociationOwner) withAuthority(ctx context.Context, p EvidenceAssociationProposal, action string, fn func(func() error) error) error {
	if o == nil || o.runs == nil || o.authority == nil || ctx == nil {
		return errors.New("association owner unavailable")
	}
	if _, _, err := p.canonical(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	calls := 0
	err := o.authority.WithAuthority(ctx, p, action, func(recheck func() error) error {
		calls++
		if calls != 1 || recheck == nil {
			return errors.New("invalid association authority boundary")
		}
		check := func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := recheck(); err != nil {
				return err
			}
			return ctx.Err()
		}
		if err := check(); err != nil {
			return err
		}
		return fn(check)
	})
	if err != nil {
		return err
	}
	if calls != 1 {
		return errors.New("authority boundary did not execute")
	}
	return nil
}
func associationAck(p EvidenceAssociationProposal, digest string) EvidenceAssociationAcknowledgment {
	return EvidenceAssociationAcknowledgment{Schema: "agent-manager.evidence-association-ack.v1", AssociationID: p.AssociationID, ProposalSHA256: digest, OwnerID: p.Native.OwnerID, SourceInstance: p.Native.SourceInstance, SourceEpoch: p.Native.SourceEpoch, RunID: p.Native.RunID, LifecycleVersion: p.Native.LifecycleVersion, TerminalObservationSHA256: p.Native.TerminalObservationSHA256, TerminalEventID: p.TerminalEventID, CanonicalEventSHA256: p.CanonicalEventSHA256, NativeRevision: 1, EffectAuthority: "none"}
}

// Acknowledge accepts only expected revision 0; an identical, still-authorized
// replay returns the original ack. Revoked or conflicting IDs never reactivate.
func (o *EvidenceAssociationOwner) Acknowledge(ctx context.Context, p EvidenceAssociationProposal, expectedRevision int) (EvidenceAssociationAcknowledgment, error) {
	return o.access(ctx, p, expectedRevision, "acknowledge", nil)
}

// Select is a fresh authorized selected read, not a transferable grant. Callers
// must also freshly check cloud and Events owners; partial states stay held.
func (o *EvidenceAssociationOwner) Select(ctx context.Context, p EvidenceAssociationProposal) (EvidenceAssociationAcknowledgment, error) {
	return o.access(ctx, p, 1, "select", nil)
}
func (o *EvidenceAssociationOwner) access(ctx context.Context, p EvidenceAssociationProposal, expected int, action string, selected *SelectedEvidenceAssociation) (EvidenceAssociationAcknowledgment, error) {
	var result EvidenceAssociationAcknowledgment
	err := o.withAuthority(ctx, p, action, func(check func() error) error {
		var err error
		result, err = o.accessAuthorized(ctx, p, expected, action, selected, check)
		return err
	})
	if err != nil {
		return EvidenceAssociationAcknowledgment{}, err
	}
	return result, nil
}
func (o *EvidenceAssociationOwner) accessAuthorized(ctx context.Context, p EvidenceAssociationProposal, expected int, action string, selected *SelectedEvidenceAssociation, check func() error) (EvidenceAssociationAcknowledgment, error) {
	var empty EvidenceAssociationAcknowledgment
	canonical, digest, err := p.canonical()
	if err != nil {
		return empty, err
	}
	if (action == "acknowledge" && expected != 0) || (action == "select" && expected != 1) {
		return empty, errors.New("association revision conflict")
	}
	if err = check(); err != nil {
		return empty, err
	}
	reader := TerminalEvidenceReader{runs: o.runs, authority: func(c context.Context, _ TerminalEvidenceSelector) error { return check() }}
	if _, err = reader.Select(ctx, p.selector()); err != nil {
		return empty, err
	}
	tx, err := o.runs.db.BeginTxx(ctx, nil)
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	// Re-pin the immutable receipt and Events ack within the transaction. The
	// complete reader above verified their semantic relationship. Exact retained
	// bytes are compared here so concurrent replacement cannot slip through.
	var observation, intent, ackJSON, instance, epoch string
	if err = tx.QueryRowContext(ctx, `SELECT r.observation_json,r.event_intent_json,a.acknowledgment_json,s.source_instance,s.source_epoch FROM run_terminal_receipts r JOIN run_terminal_event_acknowledgments a ON a.event_intent_id=r.event_intent_id CROSS JOIN run_terminal_receipt_source s WHERE r.run_id=? AND r.lifecycle_version=? AND r.source_instance=? AND r.source_epoch=? AND r.observation_sha256=? AND r.event_intent_id=? AND s.singleton=1`, p.Native.RunID, p.Native.LifecycleVersion, p.Native.SourceInstance, p.Native.SourceEpoch, p.Native.TerminalObservationSHA256, p.TerminalEventID).Scan(&observation, &intent, &ackJSON, &instance, &epoch); err != nil {
		return empty, err
	}
	var eventAck eventbus.DomainEventAcknowledgment
	if err = json.Unmarshal([]byte(ackJSON), &eventAck); err != nil {
		return empty, err
	}
	if err = eventbus.ValidateDomainEventAcknowledgment(intent, p.TerminalEventID, eventAck); err != nil {
		return empty, err
	}
	if runmodel.SHA256([]byte(observation)) != p.Native.TerminalObservationSHA256 || runmodel.SHA256([]byte(intent)) != p.TerminalEventID || eventAck.EventID != p.TerminalEventID || eventAck.CanonicalSHA256 != p.CanonicalEventSHA256 || instance != p.Native.SourceInstance || epoch != p.Native.SourceEpoch {
		return empty, errors.New("association receipt fence conflict")
	}
	ack := associationAck(p, digest)
	encoded, err := CanonicalAssociationJSON(ack)
	if err != nil {
		return empty, err
	}
	if action == "acknowledge" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO run_evidence_associations (owner_id,association_id,proposal_json,proposal_sha256,acknowledgment_json,revision,state) VALUES (?,?,?,?,?,1,'acknowledged') ON CONFLICT (owner_id,association_id) DO NOTHING`, p.OwnerID, p.AssociationID, canonical, digest, encoded); err != nil {
			return empty, err
		}
	}
	var stored, storedDigest, storedAck, state string
	var revision int
	if err = tx.QueryRowContext(ctx, `SELECT proposal_json,proposal_sha256,acknowledgment_json,revision,state FROM run_evidence_associations WHERE owner_id=? AND association_id=?`, p.OwnerID, p.AssociationID).Scan(&stored, &storedDigest, &storedAck, &revision, &state); err != nil {
		return empty, err
	}
	if stored != canonical || storedDigest != digest || storedAck != encoded || !((revision == 1 && state == "acknowledged") || (selected != nil && revision == 2 && state == "revoked")) {
		return empty, errors.New("association conflict or revocation")
	}
	if err = check(); err != nil {
		return empty, err
	}
	if err = tx.Commit(); err != nil {
		return empty, err
	}
	if selected != nil {
		*selected = SelectedEvidenceAssociation{Schema: "agent-manager.selected-evidence-association.v1", Proposal: p, Ack: ack, State: state, Revision: revision}
	}
	return ack, nil
}
func (o *EvidenceAssociationOwner) Revoke(ctx context.Context, p EvidenceAssociationProposal, expectedRevision int) error {
	return o.withAuthority(ctx, p, "revoke", func(check func() error) error { return o.revokeAuthorized(ctx, p, expectedRevision, check) })
}
func (o *EvidenceAssociationOwner) revokeAuthorized(ctx context.Context, p EvidenceAssociationProposal, expectedRevision int, check func() error) error {
	canonical, digest, err := p.canonical()
	if err != nil {
		return err
	}
	if expectedRevision != 1 {
		return errors.New("association revision conflict")
	}
	if err = check(); err != nil {
		return err
	}
	tx, err := o.runs.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE run_evidence_associations SET revision=2,state='revoked' WHERE owner_id=? AND association_id=? AND proposal_json=? AND proposal_sha256=? AND revision=1 AND state='acknowledged'`, p.OwnerID, p.AssociationID, canonical, digest)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("association revoke fence: %w", sql.ErrNoRows)
	}
	if err = check(); err != nil {
		return err
	}
	return tx.Commit()
}

// SelectedEvidenceAssociation separates immutable acknowledgment from current
// revocation state. It conveys no authority and no lifecycle or commissioning fact.
type SelectedEvidenceAssociation struct {
	Schema   string                            `json:"schema"`
	Proposal EvidenceAssociationProposal       `json:"proposal"`
	Ack      EvidenceAssociationAcknowledgment `json:"ack"`
	State    string                            `json:"state"`
	Revision int                               `json:"revision"`
}

func (o *EvidenceAssociationOwner) SelectAssociation(ctx context.Context, p EvidenceAssociationProposal) (SelectedEvidenceAssociation, error) {
	var selected SelectedEvidenceAssociation
	_, err := o.access(ctx, p, 1, "select", &selected)
	if err != nil {
		return SelectedEvidenceAssociation{}, err
	}
	return selected, nil
}
