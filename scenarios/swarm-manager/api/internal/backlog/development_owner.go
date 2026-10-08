package backlog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/effortauthority"
	sharedidentity "github.com/vrooli/api-core/identity"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"swarm-manager/internal/identity"
)

var ErrDevelopmentRefused = errors.New("development decision refused")

const developmentMaxArtifactBytes = 512 * 1024
const developmentMaxSources = 64
const developmentMaxDecisions = 1024
const developmentMaxProposalBytes = 128 * 1024

var developmentRequestID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

// DevelopmentPrincipalBinding is an exact installed identity, not a subject
// alias or a grant. Provider/source, issuer and realm are never inferred.
type DevelopmentPrincipalBinding struct {
	Kind    sharedidentity.ActorKind
	Subject string
	Source  sharedidentity.AuthSource
	Issuer  string
	Realm   string
}

// Personal-local proof intentionally has no issuer in the shared provider.
// Preserve that verified human mode through an exact source/realm/subject
// binding; other installed modes retain their verified issuer binding.
func developmentValidBinding(b DevelopmentPrincipalBinding) bool {
	if b.Subject == "" {
		return false
	}
	if b.Kind == sharedidentity.ActorAgent {
		switch b.Source {
		case sharedidentity.SourceAgentProvenance:
			return b.Issuer == "" && b.Realm == ""
		case sharedidentity.SourceScenarioAuthenticator:
			return b.Issuer != ""
		default:
			return false
		}
	}
	if b.Kind != sharedidentity.ActorHuman {
		return false
	}
	switch b.Source {
	case sharedidentity.SourcePersonalLocal:
		return b.Realm != "" && b.Issuer == ""
	case sharedidentity.SourceCloudflareAccess, sharedidentity.SourceScenarioAuthenticator:
		return b.Issuer != ""
	default:
		return false
	}
}

func (b DevelopmentPrincipalBinding) matches(p sharedidentity.Principal) bool {
	return p.IsVerified() && p.Kind == b.Kind && p.Subject == b.Subject && p.Source == b.Source && p.Issuer == b.Issuer && p.Realm == b.Realm
}

type DevelopmentArtifactBinding struct {
	ID, RelativePath, SHA256, MediaType, SnapshotDigest string
	SizeBytes                                           int64
	CapturedAt                                          time.Time
}

// DevelopmentContract is owner-installed immutable setup. It does not grant
// credentials or expand existing finite authority. New bindings need a new digest.
type DevelopmentContract struct {
	Subject          effortauthority.CommissionSubject
	Owner            DevelopmentPrincipalBinding
	Readers          []DevelopmentPrincipalBinding
	RequiredCriteria []string
	Effects          []string
	Artifacts        []DevelopmentArtifactBinding
	SourceRoot       string
	RetainedRoot     string
}

// DevelopmentEvidenceOwner resolves actual immutable owner receipts and the
// current product state. Commit must hold concrete product/evidence owner exclusion through the canonical save.
// Ordinary hashes, source stability, SQL/test leases and run pins are not that fence.
// An unavailable owner is refusal, never caller-provided evidence or success.
type DevelopmentRetainedEvidence struct {
	ID, Producer, Digest string
	Binding, Bytes       []byte
}

type DevelopmentEvidenceSnapshot struct {
	Receipts                                  []DevelopmentRetainedEvidence
	ProductDigest, EvidenceSetDigest, Version string
	Criteria                                  []*api.DevelopmentCriterionEvidence
}
type DevelopmentEvidenceOwner interface {
	ReadDevelopmentEvidence(context.Context, *api.DevelopmentReference, []string) (DevelopmentEvidenceSnapshot, error)
	CommitDevelopmentEvidence(context.Context, *api.DevelopmentReference, DevelopmentEvidenceSnapshot, func(func() error) error) error
}
type DevelopmentOwner struct {
	finite    *FiniteCommissionOwner
	contracts map[string]DevelopmentContract
	evidence  DevelopmentEvidenceOwner
}

func NewDevelopmentOwner(finite *FiniteCommissionOwner, contracts map[string]DevelopmentContract, evidence DevelopmentEvidenceOwner) (*DevelopmentOwner, error) {
	if finite == nil || finite.store == nil || len(contracts) == 0 {
		return nil, ErrDevelopmentRefused
	}
	// Copy caller-owned maps/slices before retaining configuration.
	raw, err := json.Marshal(contracts)
	if err != nil {
		return nil, err
	}
	var frozen map[string]DevelopmentContract
	if json.Unmarshal(raw, &frozen) != nil {
		return nil, ErrDevelopmentRefused
	}
	for id, c := range frozen {
		if !filepath.IsAbs(c.SourceRoot) || filepath.Clean(c.SourceRoot) != c.SourceRoot || !filepath.IsAbs(c.RetainedRoot) || filepath.Clean(c.RetainedRoot) != c.RetainedRoot || c.Subject.Validate() != nil || c.Subject.Effort != id || c.Owner.Kind != sharedidentity.ActorHuman || c.Owner.Subject != c.Subject.Owner || !developmentValidBinding(c.Owner) || len(c.RequiredCriteria) == 0 || len(c.RequiredCriteria) > 256 || len(c.Artifacts) > developmentMaxSources {
			return nil, ErrDevelopmentRefused
		}
		if _, ok := finite.targets[id]; !ok {
			return nil, ErrDevelopmentRefused
		}
		criteria := map[string]bool{}
		for _, key := range c.RequiredCriteria {
			if strings.TrimSpace(key) == "" || criteria[key] {
				return nil, ErrDevelopmentRefused
			}
			criteria[key] = true
		}
		artifacts := map[string]bool{}
		for _, a := range c.Artifacts {
			if a.ID == "" || artifacts[a.ID] || !developmentSafePath(a.RelativePath) || !developmentValidDigest(a.SHA256) || !developmentValidDigest(a.SnapshotDigest) || a.SizeBytes < 0 || a.SizeBytes > developmentMaxArtifactBytes || a.CapturedAt.IsZero() {
				return nil, ErrDevelopmentRefused
			}
			artifacts[a.ID] = true
		}
		for _, b := range c.Readers {
			if b.Subject == "" || !developmentValidBinding(b) || (b.Kind != sharedidentity.ActorHuman && b.Kind != sharedidentity.ActorAgent) {
				return nil, ErrDevelopmentRefused
			}
		}
	}
	return &DevelopmentOwner{finite: finite, contracts: frozen, evidence: evidence}, nil
}
func developmentValidDigest(value string) bool {
	raw, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && len(raw) == 32 && strings.ToLower(value) == value
}
func developmentSafePath(value string) bool {
	return value != "" && value != "." && !strings.Contains(value, "\\") && !strings.HasPrefix(value, "/") && path.Clean(value) == value && value != ".." && !strings.HasPrefix(value, "../")
}
func developmentDigest(value any) string { return "sha256:" + effortauthority.Digest(value) }
func developmentBytesDigest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func (o *DevelopmentOwner) authorize(ctx context.Context, id string, write bool) (DevelopmentContract, sharedidentity.Principal, error) {
	cap := "swarm-manager:read"
	if write {
		cap = "swarm-manager:write"
	}
	p, err := authn.RequireCapability(ctx, cap)
	if err != nil || ctx.Err() != nil || (!p.ExpiresAt.IsZero() && !p.ExpiresAt.After(time.Now())) {
		return DevelopmentContract{}, p, ErrDevelopmentRefused
	}
	if write {
		if _, err = authn.RequireHuman(ctx); err != nil {
			return DevelopmentContract{}, p, ErrDevelopmentRefused
		}
	}
	if o == nil || o.finite == nil {
		return DevelopmentContract{}, p, ErrDevelopmentRefused
	}
	c, ok := o.contracts[id]
	if !ok {
		return c, p, ErrDevelopmentRefused
	}
	if c.Owner.matches(p) {
		return c, p, nil
	}
	if !write {
		for _, b := range c.Readers {
			if b.matches(p) {
				return c, p, nil
			}
		}
	}
	return c, p, ErrDevelopmentRefused
}
func developmentActor(p sharedidentity.Principal) *api.DevelopmentActor {
	return &api.DevelopmentActor{Subject: p.Subject, Provider: p.Issuer, Realm: p.Realm, Source: string(p.Source)}
}
func (o *DevelopmentOwner) current(ctx context.Context, id string, write bool) (identity.EffortControl, DevelopmentContract, sharedidentity.Principal, *api.DevelopmentReference, error) {
	c, p, err := o.authorize(ctx, id, write)
	if err != nil {
		return identity.EffortControl{}, c, p, nil, err
	}
	control, _, err := o.finite.current(ctx, c.Subject)
	if err != nil {
		return control, c, p, nil, err
	}
	if control.Validate() != nil {
		return control, c, p, nil, ErrDevelopmentRefused
	}
	ref := &api.DevelopmentReference{EffortId: id, Revision: control.Revision, AuthorityDigest: control.AuthorityDigest(), ContractDigest: developmentDigest(c), CommissionSubjectDigest: developmentDigest(c.Subject)}
	return control, c, p, ref, nil
}
func (o *DevelopmentOwner) addressed(ctx context.Context, supplied *api.DevelopmentReference, write bool) (identity.EffortControl, DevelopmentContract, sharedidentity.Principal, *api.DevelopmentReference, error) {
	if supplied == nil {
		return identity.EffortControl{}, DevelopmentContract{}, sharedidentity.Principal{}, nil, ErrDevelopmentRefused
	}
	c, target, p, ref, err := o.current(ctx, supplied.EffortId, write)
	if err != nil || !proto.Equal(ref, supplied) {
		return c, target, p, ref, ErrDevelopmentRefused
	}
	return c, target, p, ref, nil
}
func developmentArtifacts(c DevelopmentContract) []*api.DevelopmentArtifact {
	out := make([]*api.DevelopmentArtifact, 0, len(c.Artifacts))
	for _, a := range c.Artifacts {
		out = append(out, &api.DevelopmentArtifact{Id: a.ID, RelativePath: a.RelativePath, Sha256: a.SHA256, SizeBytes: a.SizeBytes, MediaType: a.MediaType, CapturedAt: a.CapturedAt.UTC().Format(time.RFC3339Nano), SnapshotDigest: a.SnapshotDigest})
	}
	return out
}
func (o *DevelopmentOwner) GetDevelopment(ctx context.Context, id string) (*api.DevelopmentView, error) {
	c, target, p, ref, err := o.current(ctx, id, false)
	if err != nil {
		return nil, err
	}
	l := c.AggregateLimits
	out := &api.DevelopmentView{Reference: ref, WorkShape: string(o.finite.targets[id].WorkShape), OwnerSubject: target.Owner.Subject, Owner: &api.DevelopmentActor{Subject: target.Owner.Subject, Provider: target.Owner.Issuer, Realm: target.Owner.Realm, Source: string(target.Owner.Source)}, ScopeAllow: c.Scope.Allow, ScopeDeny: c.Scope.Deny, Limits: &api.DevelopmentLimits{MaxWorkers: int32(l.MaxWorkers), MaxConcurrency: int32(l.MaxConcurrency), MaxDepth: int32(l.MaxDepth), MaxActiveDescendants: int32(l.MaxActiveDescendants), MaxPremiumDescendants: int32(l.MaxPremiumDescendants), MaxTokens: l.MaxTokens, MaxChargeMicroUsd: l.MaxChargeMicroUSD, MaxWallSeconds: l.MaxWallSeconds, MaxWaitSeconds: int32(l.MaxWaitSeconds), Deadline: l.Deadline}, Artifacts: developmentArtifacts(target), RequiredCriterionIds: append([]string(nil), target.RequiredCriteria...), AccountingStatus: "unavailable", Accounting: &api.DevelopmentAccounting{Status: "unavailable"}, LaunchBlockers: []string{"development decisions do not qualify or enable runtime execution", "canonical accounting observation is unavailable", "current caller migration and physical carrier isolation are unqualified"}}
	_ = p
	if r := c.FiniteCommission; r != nil {
		out.Generation = r.Generation
		live, item, frontierErr := o.finite.current(ctx, target.Subject)
		planHash := ""
		if item.PlanAcceptance != nil {
			planHash = item.PlanAcceptance.PlanContentHash
		}
		out.Approved = frontierErr == nil && live.AuthorityDigest() == c.AuthorityDigest() && r.WorkShape == string(o.finite.targets[id].WorkShape) && r.Actor == target.Subject.Owner && !r.AcceptedAt.IsZero() && !r.AcceptedAt.After(time.Now()) && r.Generation > 0 && r.PlanSubjectVersion == finitePlanSubjectVersion(item) && r.PlanContentHash == planHash && effortauthority.Digest(r.Subject) == effortauthority.Digest(target.Subject) && r.AuthorityDigest == c.AuthorityDigest() && r.DevelopmentContractDigest == ref.ContractDigest
		out.Revoked = r.Revoked
	}
	if !out.Approved || out.Revoked {
		out.LaunchBlockers = append(out.LaunchBlockers, "current finite commission is not eligible")
	}
	if o.evidence == nil {
		out.LaunchBlockers = append(out.LaunchBlockers, "product evidence owner is unavailable")
	}
	if d := c.Development; d != nil {
		out.DispositionVersion = d.DispositionVersion
		out.OutcomeAccepted = d.AcceptedProductDigest != "" && d.AcceptedReferenceDigest == developmentDigest(ref)
		if out.OutcomeAccepted {
			out.TestedProductDigest = d.AcceptedProductDigest
			out.EvidenceSetDigest = d.EvidenceSetDigest
		}
		for key, record := range d.Decisions {
			receipt, err := developmentDecodeReceipt(record)
			if err != nil || receipt.RequestId != key {
				return nil, ErrDevelopmentRefused
			}
			out.RetainedDecisions = append(out.RetainedDecisions, receipt)
		}
		sort.Slice(out.RetainedDecisions, func(i, j int) bool { return out.RetainedDecisions[i].RequestId < out.RetainedDecisions[j].RequestId })
		if out.OutcomeAccepted && len(d.Criteria) > 0 {
			var rows []*api.DevelopmentCriterionEvidence
			if json.Unmarshal(d.Criteria, &rows) != nil {
				return nil, ErrDevelopmentRefused
			}
			out.Criteria = rows
		}
	}
	return out, nil
}

// developmentReadFile uses a server-selected root and rejects every symlink,
// traversal, nonregular file and excessive read before exposing bytes.
func developmentReadFile(rootPath, relative string) ([]byte, error) {
	if !developmentSafePath(relative) {
		return nil, ErrDevelopmentRefused
	}
	info, err := os.Lstat(rootPath)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrDevelopmentRefused
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	parts := strings.Split(relative, "/")
	for i := range parts {
		info, err = root.Lstat(strings.Join(parts[:i+1], "/"))
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil, ErrDevelopmentRefused
		}
	}
	if !info.Mode().IsRegular() || info.Size() > developmentMaxArtifactBytes {
		return nil, ErrDevelopmentRefused
	}
	f, err := root.Open(relative)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, ErrDevelopmentRefused
	}
	raw, err := io.ReadAll(io.LimitReader(f, developmentMaxArtifactBytes+1))
	if err != nil || len(raw) > developmentMaxArtifactBytes {
		return nil, ErrDevelopmentRefused
	}
	return raw, nil
}
func (o *DevelopmentOwner) GetDevelopmentArtifact(ctx context.Context, ref *api.DevelopmentReference, id string) (*api.GetDevelopmentArtifactResponse, error) {
	if ref == nil {
		return nil, ErrDevelopmentRefused
	}
	ref = proto.Clone(ref).(*api.DevelopmentReference)
	// Resolve already retained versions independently of the latest frontier.
	if _, record, contract, err := o.historical(ctx, ref, false); err == nil {
		for _, a := range contract.Artifacts {
			if a.ID == id {
				return &api.GetDevelopmentArtifactResponse{Reference: ref, Artifact: developmentArtifacts(DevelopmentContract{Artifacts: []DevelopmentArtifactBinding{a}})[0], Content: append([]byte(nil), record.ArtifactBytes[id]...)}, nil
			}
		}
		return nil, ErrDevelopmentRefused
	} else if !errors.Is(err, errDevelopmentHistoryMissing) {
		return nil, ErrDevelopmentRefused
	}
	_, target, _, address, err := o.addressed(ctx, ref, false)
	if err != nil {
		return nil, err
	}
	// Current, not-yet-published retained inputs remain bounded exact reads.
	for _, a := range target.Artifacts {
		if a.ID == id {
			raw, err := developmentReadFile(target.RetainedRoot, a.RelativePath)
			if err != nil || int64(len(raw)) != a.SizeBytes || strings.TrimPrefix(developmentBytesDigest(raw), "sha256:") != strings.TrimPrefix(a.SHA256, "sha256:") {
				return nil, ErrDevelopmentRefused
			}
			return &api.GetDevelopmentArtifactResponse{Reference: address, Artifact: developmentArtifacts(DevelopmentContract{Artifacts: []DevelopmentArtifactBinding{a}})[0], Content: raw}, nil
		}
	}
	return nil, ErrDevelopmentRefused
}
func developmentDecodeReceipt(record identity.DevelopmentDecisionRecord) (*api.DevelopmentDecisionResponse, error) {
	if developmentBytesDigest(record.Receipt) != record.ReceiptDigest {
		return nil, ErrDevelopmentRefused
	}
	receipt := new(api.DevelopmentDecisionResponse)
	if protojson.Unmarshal(record.Receipt, receipt) != nil {
		return nil, ErrDevelopmentRefused
	}
	if !developmentValidDigest(record.RequestDigest) || !developmentRequestID.MatchString(receipt.RequestId) || receipt.Reference == nil || receipt.Reference.EffortId == "" || receipt.Reference.Revision < 1 || !developmentValidDigest(receipt.Reference.AuthorityDigest) || !developmentValidDigest(receipt.Reference.ContractDigest) || !developmentValidDigest(receipt.Reference.CommissionSubjectDigest) || receipt.Actor == nil || receipt.Actor.Subject == "" || receipt.Actor.Subject != receipt.ActorSubject || !developmentValidBinding(DevelopmentPrincipalBinding{Kind: sharedidentity.ActorHuman, Subject: receipt.Actor.Subject, Source: sharedidentity.AuthSource(receipt.Actor.Source), Issuer: receipt.Actor.Provider, Realm: receipt.Actor.Realm}) || receipt.Generation == 0 {
		return nil, ErrDevelopmentRefused
	}
	if _, err := time.Parse(time.RFC3339Nano, receipt.DecidedAt); err != nil {
		return nil, ErrDevelopmentRefused
	}
	fingerprint := proto.Clone(receipt).(*api.DevelopmentDecisionResponse)
	fingerprint.ReceiptDigest = ""
	if developmentDigest(fingerprint) != receipt.ReceiptDigest {
		return nil, ErrDevelopmentRefused
	}
	switch receipt.Action {
	case "approve", "revoke":
		if receipt.TestedProductDigest != "" || receipt.EvidenceSetDigest != "" || len(receipt.Criteria) != 0 {
			return nil, ErrDevelopmentRefused
		}
	case "accept":
		if receipt.DispositionVersion == 0 || !developmentValidDigest(receipt.TestedProductDigest) || !developmentValidDigest(receipt.EvidenceSetDigest) || len(receipt.Criteria) == 0 {
			return nil, ErrDevelopmentRefused
		}
	default:
		return nil, ErrDevelopmentRefused
	}
	return receipt, nil
}
func (o *DevelopmentOwner) decide(ctx context.Context, ref *api.DevelopmentReference, generation *uint64, version *uint64, key, action, expectedProduct string) (*api.DevelopmentDecisionResponse, error) {
	if generation == nil || !developmentRequestID.MatchString(key) {
		return nil, ErrDevelopmentRefused
	}
	if ref != nil {
		ref = proto.Clone(ref).(*api.DevelopmentReference)
	}
	g := *generation
	generation = &g
	if version != nil {
		v := *version
		version = &v
	}
	if ref == nil {
		return nil, ErrDevelopmentRefused
	}
	_, principal, err := o.authorize(ctx, ref.EffortId, true)
	if err != nil {
		return nil, err
	}
	address := ref
	if action == "accept" && (version == nil || !developmentValidDigest(expectedProduct)) {
		return nil, ErrDevelopmentRefused
	}
	requestDigest := developmentDigest(struct {
		Reference            *api.DevelopmentReference
		Generation           *uint64
		Version              *uint64
		Key, Action, Product string
		Actor                *api.DevelopmentActor
	}{address, generation, version, key, action, expectedProduct, developmentActor(principal)})
	if replay, found, err := o.replayHistorical(ctx, address, key, action, requestDigest); err != nil || found {
		return replay, err
	}
	_, target, principal, currentAddress, err := o.addressed(ctx, address, true)
	if err != nil {
		return nil, err
	}
	address = currentAddress
	var receipt *api.DevelopmentDecisionResponse
	err = o.finite.store.withFiniteLock(address.EffortId, func() error {
		current, err := o.finite.store.Load(address.EffortId)
		if err != nil {
			return err
		}
		// Re-resolve authored work and current target under the existing owner lock.
		live, _, _, again, err := o.current(ctx, address.EffortId, true)
		if err != nil || !proto.Equal(address, again) || developmentDigest(live) != developmentDigest(current) {
			return ErrDevelopmentRefused
		}
		if current.Development != nil {
			if record, ok := current.Development.Decisions[key]; ok {
				if record.RequestDigest != requestDigest {
					return ErrDevelopmentRefused
				}
				receipt, err = developmentDecodeReceipt(record)
				if err == nil && (receipt.RequestId != key || receipt.Action != action || !proto.Equal(receipt.Reference, address) || !proto.Equal(receipt.Actor, developmentActor(principal))) {
					return ErrDevelopmentRefused
				}
				return err
			}
		}
		gen := uint64(0)
		if current.FiniteCommission != nil {
			gen = current.FiniteCommission.Generation
		}
		if gen != *generation {
			return ErrDevelopmentRefused
		}
		next := current
		var snapshot DevelopmentEvidenceSnapshot
		switch action {
		case "approve":
			_, item, err := o.finite.current(ctx, target.Subject)
			if err != nil {
				return err
			}
			next, err = o.finite.acceptFiniteTransform(ctx, target.Subject, *generation, current, item)
			if err != nil {
				return err
			}
			next.FiniteCommission.DevelopmentContractDigest = address.ContractDigest
		case "revoke":
			r := current.FiniteCommission
			if r == nil || r.DevelopmentContractDigest != address.ContractDigest || r.Revoked || r.Generation == ^uint64(0) || effortauthority.Digest(r.Subject) != effortauthority.Digest(target.Subject) {
				return ErrDevelopmentRefused
			}
			copy := *r
			copy.Revoked = true
			copy.Generation++
			copy.RevokedAt = time.Now().UTC()
			next.FiniteCommission = &copy
		case "accept":
			approved := current.FiniteCommission
			_, approvedItem, frontierErr := o.finite.current(ctx, target.Subject)
			planHash := ""
			if approvedItem.PlanAcceptance != nil {
				planHash = approvedItem.PlanAcceptance.PlanContentHash
			}
			if frontierErr != nil || approved == nil || approved.DevelopmentContractDigest != address.ContractDigest || approved.WorkShape != string(o.finite.targets[address.EffortId].WorkShape) || approved.PlanSubjectVersion != finitePlanSubjectVersion(approvedItem) || approved.PlanContentHash != planHash || approved.Generation == 0 || approved.Actor != target.Subject.Owner || approved.AcceptedAt.IsZero() || approved.AcceptedAt.After(time.Now()) || approved.AuthorityDigest != current.AuthorityDigest() || effortauthority.Digest(approved.Subject) != effortauthority.Digest(target.Subject) {
				return ErrDevelopmentRefused
			}
			// Revocation closes launch admission; it does not erase the exact
			// historically commissioned work or its retained outcome obligations.
			standing := uint64(0)
			if current.Development != nil {
				standing = current.Development.DispositionVersion
			}
			if standing != *version || standing == ^uint64(0) || o.evidence == nil {
				return ErrDevelopmentRefused
			}
			snapshot, err = o.evidence.ReadDevelopmentEvidence(ctx, proto.Clone(address).(*api.DevelopmentReference), append([]string(nil), target.RequiredCriteria...))
			if err != nil || snapshot.ProductDigest != expectedProduct || !developmentValidDigest(snapshot.EvidenceSetDigest) || snapshot.Version == "" || len(snapshot.Criteria) != len(target.RequiredCriteria) {
				return ErrDevelopmentRefused
			}
			// Capture provider-owned rows before checks and retained hashing.
			snapshot = developmentCloneSnapshot(snapshot)
			frozenRows := make([]*api.DevelopmentCriterionEvidence, len(snapshot.Criteria))
			for i, row := range snapshot.Criteria {
				if row != nil {
					frozenRows[i] = proto.Clone(row).(*api.DevelopmentCriterionEvidence)
				}
			}
			snapshot.Criteria = frozenRows
			rows := map[string]bool{}
			for _, row := range snapshot.Criteria {
				if row == nil || !row.Satisfied || row.UnavailableReason != "" || row.TestedProductDigest != expectedProduct || len(row.ReceiptIds) == 0 || !developmentValidDigest(row.ReceiptSetDigest) || rows[row.CriterionId] {
					return ErrDevelopmentRefused
				}
				rows[row.CriterionId] = true
			}
			for _, id := range target.RequiredCriteria {
				if !rows[id] {
					return ErrDevelopmentRefused
				}
			}

		default:
			return ErrDevelopmentRefused
		}
		var d identity.DevelopmentStanding
		if current.Development != nil {
			raw, _ := json.Marshal(current.Development)
			if json.Unmarshal(raw, &d) != nil {
				return ErrDevelopmentRefused
			}
		}
		if d.History == nil {
			d.History = map[string]identity.DevelopmentHistoryRecord{}
		}
		historyKey := developmentDigest(address)
		if existing, ok := d.History[historyKey]; ok {
			if _, err := developmentDecodeHistory(address, existing); err != nil {
				return err
			}
		} else {
			if len(d.History) >= developmentMaxHistory {
				return ErrDevelopmentRefused
			}
			retained, err := developmentCaptureHistory(address, target)
			if err != nil {
				return err
			}
			d.History[historyKey] = retained
			raw, err := json.Marshal(d.History)
			if err != nil || len(raw) > developmentMaxHistoryBytes {
				return ErrDevelopmentRefused
			}
		}
		if d.Decisions == nil {
			d.Decisions = map[string]identity.DevelopmentDecisionRecord{}
		}
		if len(d.Decisions) >= developmentMaxDecisions {
			return ErrDevelopmentRefused
		}
		if action == "accept" {
			d.DispositionVersion++
			d.AcceptedReferenceDigest = developmentDigest(address)
			d.AcceptedProductDigest = snapshot.ProductDigest
			d.EvidenceSetDigest = snapshot.EvidenceSetDigest
			d.Criteria, _ = json.Marshal(snapshot.Criteria)
			if d.EvidenceReceipts == nil {
				d.EvidenceReceipts = map[string]identity.DevelopmentEvidenceReceipt{}
			}
			totalBytes := 0
			for _, retained := range d.EvidenceReceipts {
				totalBytes += len(retained.Bytes) + len(retained.Binding)
			}
			seenReceipts := map[string]bool{}
			for _, r := range snapshot.Receipts {
				if r.ID == "" || r.Producer != "test-genie" || !developmentValidDigest(r.Digest) || developmentBytesDigest(r.Bytes) != r.Digest || len(r.Binding) == 0 {
					return ErrDevelopmentRefused
				}
				if seenReceipts[r.ID] {
					return ErrDevelopmentRefused
				}
				seenReceipts[r.ID] = true
				value := identity.DevelopmentEvidenceReceipt{Producer: r.Producer, Digest: r.Digest, Binding: r.Binding, Bytes: r.Bytes}
				if retained, exists := d.EvidenceReceipts[r.ID]; exists {
					if !reflect.DeepEqual(retained, value) {
						return ErrDevelopmentRefused
					}
					continue
				}
				totalBytes += len(r.Bytes) + len(r.Binding)
				if totalBytes > developmentMaxHistoryBytes {
					return ErrDevelopmentRefused
				}
				d.EvidenceReceipts[r.ID] = value
			}
			if len(d.EvidenceReceipts) == 0 {
				return ErrDevelopmentRefused
			}
		}
		afterGen := uint64(0)
		if next.FiniteCommission != nil {
			afterGen = next.FiniteCommission.Generation
		}
		receipt = &api.DevelopmentDecisionResponse{RequestId: key, Action: action, Reference: address, Generation: afterGen, ActorSubject: principal.Subject, Actor: developmentActor(principal), DecidedAt: time.Now().UTC().Format(time.RFC3339Nano), DispositionVersion: d.DispositionVersion, EvidenceSetDigest: snapshot.EvidenceSetDigest, TestedProductDigest: snapshot.ProductDigest, Criteria: snapshot.Criteria}
		receipt.ReceiptDigest = developmentDigest(receipt)
		raw, err := protojson.Marshal(receipt)
		if err != nil {
			return err
		}
		d.Decisions[key] = identity.DevelopmentDecisionRecord{RequestDigest: requestDigest, ReceiptDigest: developmentBytesDigest(raw), Receipt: raw}
		next.Development = &d
		// Only the two owner runtime records may change. Completion and authored
		// limits/scopes/candidate bindings are preserved exactly.
		a, b := current, next
		a.FiniteCommission = nil
		b.FiniteCommission = nil
		a.Development = nil
		b.Development = nil
		if !reflect.DeepEqual(a, b) {
			return ErrDevelopmentRefused
		}
		if _, _, _, fenced, err := o.current(ctx, address.EffortId, true); err != nil || !proto.Equal(address, fenced) {
			return ErrDevelopmentRefused
		}
		if action == "accept" {
			calls := 0
			var commitErr error
			fenceErr := o.evidence.CommitDevelopmentEvidence(ctx, proto.Clone(address).(*api.DevelopmentReference), developmentCloneSnapshot(snapshot), func(guard func() error) error {
				if guard == nil {
					return ErrDevelopmentRefused
				}
				calls++
				if calls != 1 {
					return ErrDevelopmentRefused
				}
				if _, _, _, fenced, err := o.current(ctx, address.EffortId, true); err != nil || !proto.Equal(address, fenced) {
					commitErr = ErrDevelopmentRefused
					return commitErr
				}
				commitErr = o.finite.store.saveUnlockedGuarded(next, guard)
				return commitErr
			})
			if fenceErr != nil {
				return fenceErr
			}
			if calls != 1 {
				return ErrDevelopmentRefused
			}
			return commitErr
		}
		return o.finite.store.saveUnlocked(next)
	})
	if err != nil {
		return nil, err
	}
	return receipt, nil
}
func (o *DevelopmentOwner) ApproveDevelopment(ctx context.Context, r *api.ApproveDevelopmentRequest) (*api.DevelopmentDecisionResponse, error) {
	if r == nil {
		return nil, ErrDevelopmentRefused
	}
	return o.decide(ctx, r.Reference, r.ExpectedGeneration, nil, r.RequestId, "approve", "")
}
func (o *DevelopmentOwner) RevokeDevelopment(ctx context.Context, r *api.RevokeDevelopmentRequest) (*api.DevelopmentDecisionResponse, error) {
	if r == nil {
		return nil, ErrDevelopmentRefused
	}
	return o.decide(ctx, r.Reference, r.ExpectedGeneration, nil, r.RequestId, "revoke", "")
}
func (o *DevelopmentOwner) AcceptDevelopment(ctx context.Context, r *api.AcceptDevelopmentRequest) (*api.DevelopmentDecisionResponse, error) {
	if r == nil {
		return nil, ErrDevelopmentRefused
	}
	return o.decide(ctx, r.Reference, r.ExpectedGeneration, r.ExpectedDispositionVersion, r.RequestId, "accept", r.TestedProductDigest)
}
func (o *DevelopmentOwner) PreviewDevelopment(ctx context.Context, r *api.PreviewDevelopmentRequest) (*api.PreviewDevelopmentResponse, error) {
	if r == nil || r.Proposal == nil {
		return nil, ErrDevelopmentRefused
	}
	r = proto.Clone(r).(*api.PreviewDevelopmentRequest)
	control, target, _, ref, err := o.addressed(ctx, r.Reference, false)
	if err != nil {
		return nil, err
	}
	proposal := proto.Clone(r.Proposal).(*api.DevelopmentProposal)
	if proto.Size(proposal) > developmentMaxProposalBytes || !utf8.ValidString(proposal.Title) || !utf8.ValidString(proposal.Description) || len(proposal.SourceRelativePaths) > developmentMaxSources || len(proposal.RequiredCriterionIds) > 256 || len(proposal.ScopeAllow) > 256 || len(proposal.ScopeDeny) > 256 || len(proposal.ProposedEffects) > 256 {
		return nil, ErrDevelopmentRefused
	}
	out := &api.PreviewDevelopmentResponse{Reference: ref, Proposal: proposal, LaunchBlockers: []string{"preview grants no commission, dispatch or product acceptance"}}
	if strings.TrimSpace(proposal.Title) == "" {
		out.CompletenessFindings = append(out.CompletenessFindings, "title is missing")
	}
	if len(proposal.RequiredCriterionIds) == 0 {
		out.CompletenessFindings = append(out.CompletenessFindings, "required criteria are missing")
	}
	if proposal.WorkShape != string(o.finite.targets[ref.EffortId].WorkShape) {
		out.Conflicts = append(out.Conflicts, "work shape differs from installed target")
	}
	if proposal.TargetSubjectRef == "" {
		out.CompletenessFindings = append(out.CompletenessFindings, "target subject reference is missing")
	} else if proposal.TargetSubjectRef != ref.CommissionSubjectDigest {
		out.Conflicts = append(out.Conflicts, "target subject reference differs from installed target")
	}
	// Proposed patterns cannot establish containment merely through matching a
	// sample path. Retain a conservative exact authored scope check.
	if !reflect.DeepEqual(proposal.ScopeAllow, control.Scope.Allow) || !reflect.DeepEqual(proposal.ScopeDeny, control.Scope.Deny) {
		out.Conflicts = append(out.Conflicts, "scope differs from installed reviewed ceiling")
	}
	if proposal.Limits == nil {
		out.CompletenessFindings = append(out.CompletenessFindings, "limits are missing")
	} else {
		l := proposal.Limits
		proposed := identity.AggregateLimits{MaxWorkers: int(l.MaxWorkers), MaxConcurrency: int(l.MaxConcurrency), MaxDepth: int(l.MaxDepth), MaxActiveDescendants: int(l.MaxActiveDescendants), MaxPremiumDescendants: int(l.MaxPremiumDescendants), MaxTokens: l.MaxTokens, MaxChargeMicroUSD: l.MaxChargeMicroUsd, MaxWallSeconds: l.MaxWallSeconds, MaxWaitSeconds: int(l.MaxWaitSeconds), Deadline: l.Deadline}
		ceiling := control.AggregateLimits
		if proposed.Validate() != nil {
			out.Conflicts = append(out.Conflicts, "limits are invalid")
		}
		if proposed.MaxWorkers > ceiling.MaxWorkers || proposed.MaxConcurrency > ceiling.MaxConcurrency || proposed.MaxDepth > ceiling.MaxDepth || proposed.MaxActiveDescendants > ceiling.MaxActiveDescendants || proposed.MaxPremiumDescendants > ceiling.MaxPremiumDescendants {
			out.Conflicts = append(out.Conflicts, "descendant limits exceed installed ceiling")
		}
		for _, bound := range [][2]int64{{proposed.MaxTokens, ceiling.MaxTokens}, {proposed.MaxChargeMicroUSD, ceiling.MaxChargeMicroUSD}, {proposed.MaxWallSeconds, ceiling.MaxWallSeconds}, {int64(proposed.MaxWaitSeconds), int64(ceiling.MaxWaitSeconds)}} {
			if bound[0] < 0 || (bound[1] > 0 && (bound[0] == 0 || bound[0] > bound[1])) {
				out.Conflicts = append(out.Conflicts, "resource limits exceed installed ceiling")
				break
			}
		}
		if ceiling.Deadline != "" {
			c, ce := time.Parse(time.RFC3339, ceiling.Deadline)
			p, pe := time.Parse(time.RFC3339, proposed.Deadline)
			if ce != nil || pe != nil || p.After(c) {
				out.Conflicts = append(out.Conflicts, "deadline exceeds installed ceiling")
			}
		} else if proposed.Deadline != "" {
			if _, err := time.Parse(time.RFC3339, proposed.Deadline); err != nil {
				out.Conflicts = append(out.Conflicts, "deadline is invalid")
			}
		}
	}
	allowed := map[string]bool{}
	for _, effect := range target.Effects {
		allowed[effect] = true
	}
	for _, effect := range proposal.ProposedEffects {
		if !allowed[effect] {
			out.Conflicts = append(out.Conflicts, "effect exceeds installed ceiling: "+effect)
		}
	}
	seen := map[string]bool{}
	for _, relative := range proposal.SourceRelativePaths {
		if seen[relative] || !developmentSafePath(relative) || !developmentReviewableSource(relative) || !control.Scope.Allows(relative) {
			return nil, ErrDevelopmentRefused
		}
		seen[relative] = true
		raw, err := developmentReadFile(target.SourceRoot, relative)
		if err != nil || !utf8.Valid(raw) {
			return nil, fmt.Errorf("preview source unavailable: %w", ErrDevelopmentRefused)
		}
		out.ObservedSources = append(out.ObservedSources, &api.DevelopmentSourceObservation{RelativePath: relative, Sha256: developmentBytesDigest(raw), SizeBytes: int64(len(raw)), MediaType: "application/octet-stream"})
	}
	out.ProposalDigest = developmentDigest(struct {
		Reference *api.DevelopmentReference
		Proposal  *api.DevelopmentProposal
		Sources   []*api.DevelopmentSourceObservation
	}{ref, proposal, out.ObservedSources})
	return out, nil
}

// Preview is bounded to reviewable text source; work scope alone does not
// authorize reading credentials, runtime databases or arbitrary binary files.
func developmentReviewableSource(relative string) bool {
	for _, part := range strings.Split(relative, "/") {
		if strings.HasPrefix(part, ".") || part == "node_modules" || part == "data" {
			return false
		}
	}
	switch strings.ToLower(path.Ext(relative)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".md", ".proto", ".css", ".scss", ".html":
		return true
	}
	return false
}

func developmentCloneSnapshot(snapshot DevelopmentEvidenceSnapshot) DevelopmentEvidenceSnapshot {
	out := snapshot
	out.Receipts = make([]DevelopmentRetainedEvidence, len(snapshot.Receipts))
	for i, r := range snapshot.Receipts {
		out.Receipts[i] = r
		out.Receipts[i].Binding = append([]byte(nil), r.Binding...)
		out.Receipts[i].Bytes = append([]byte(nil), r.Bytes...)
	}
	out.Criteria = make([]*api.DevelopmentCriterionEvidence, len(snapshot.Criteria))
	for i, row := range snapshot.Criteria {
		if row != nil {
			out.Criteria[i] = proto.Clone(row).(*api.DevelopmentCriterionEvidence)
		}
	}
	return out
}
