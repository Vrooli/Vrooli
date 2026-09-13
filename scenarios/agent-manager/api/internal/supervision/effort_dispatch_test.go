package supervision

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/identity"
	"github.com/google/uuid"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type dispatchFixture struct {
	s            *EffortService
	r            *Repository
	owner        EffortActor
	e            *pb.EffortEnrollment
	req          *api.IssueSupervisorDispatchRequest
	token        string
	now          time.Time
	secret       []byte
	provisionErr error
}

func newDispatchFixture(t *testing.T) *dispatchFixture {
	t.Helper()
	s, r, _ := effortFixture(t)
	f := &dispatchFixture{s: s, r: r, now: time.Now().UTC().Truncate(time.Second), secret: []byte("dispatch-fixture-secret"), owner: EffortActor{ID: "owner", Operator: true, Scopes: []string{"agent-manager:supervise"}}}
	s.now = func() time.Time { return f.now }
	s.ConfigureDispatch(f.secret, func(token string) error {
		if f.provisionErr != nil {
			return f.provisionErr
		}
		f.token = token
		return nil
	}, func(context.Context, string) error { return nil })
	var err error
	f.e, err = s.Enroll(context.Background(), &pb.EnrollEffortRequest{Enrollment: &pb.EffortEnrollment{EffortRef: "service:standing-supervision", SupervisorOwnerSubject: "owner", SupervisorScope: "agent-manager:supervise"}, IdempotencyKey: "anchor"}, f.owner)
	if err != nil {
		t.Fatal(err)
	}
	f.req = &api.IssueSupervisorDispatchRequest{EffortRef: f.e.EffortRef, ExpectedRevision: f.e.Revision, TeamId: "supervisors", MemberId: "leader", ProfileKey: "qualified", ExpiresAt: timestamppb.New(f.now.Add(time.Hour)), MaximumRuns: 3, MinimumIntervalSeconds: 60, IdempotencyKey: "issue"}
	return f
}
func (f *dispatchFixture) issue(t *testing.T) {
	t.Helper()
	var err error
	f.e, err = f.s.IssueDispatch(context.Background(), f.req, f.owner)
	if err != nil {
		t.Fatal(err)
	}
}
func (f *dispatchFixture) wake(key string) *api.CreateSupervisorRunRequest {
	return &api.CreateSupervisorRunRequest{EffortRef: f.e.EffortRef, AuthorizationId: f.e.DispatchAuthorization.AuthorizationId, TeamId: "supervisors", MemberId: "leader", TaskId: uuid.NewString(), IdempotencyKey: key}
}

func TestSupervisorDispatchAnchorDoesNotBecomeItsOwnEffort(t *testing.T) {
	f := newDispatchFixture(t)
	f.issue(t)
	board, err := f.s.Board(t.Context(), &pb.GetEffortBoardRequest{EffortRef: f.e.EffortRef})
	if err != nil || board.ActiveCount != 0 || len(board.Rows) != 1 || board.Rows[0].NextAction != "authorization-only" || board.Rows[0].GetOutcomeStanding().GetState() != "not-applicable" {
		t.Fatal("dispatch-only anchor counted as unfinished work", board, err)
	}
	ordinary := proto.Clone(f.e).(*pb.EffortEnrollment)
	ordinary.DispatchAuthorization = nil
	row := f.s.projectEffort(t.Context(), ordinary, &pb.EffortBoardRow{NextAction: "authorization-only"}, &pb.EffortDiscovery{})
	if row.NextAction == "authorization-only" {
		t.Fatal("untrusted source hid an effort behind authorization-only standing")
	}
}
func TestSupervisorDispatchPurposeProvisioningAndReplay(t *testing.T) {
	f := newDispatchFixture(t)
	f.provisionErr = errors.New("private-store-error")
	if _, err := f.s.IssueDispatch(context.Background(), f.req, f.owner); err == nil || strings.Contains(err.Error(), "private-store-error") {
		t.Fatal("provisioning failure hidden or leaked")
	}
	f.provisionErr = nil
	f.issue(t)
	firstToken, firstID := f.token, f.e.DispatchAuthorization.AuthorizationId
	f.issue(t)
	if f.token != firstToken || f.e.DispatchAuthorization.AuthorizationId != firstID {
		t.Fatal("replayed issuance minted new authority")
	}
	claims, err := identity.VerifyToken(f.token, f.secret)
	if err != nil || claims.Purpose != SupervisorDispatchPurpose || len(claims.Scopes) != 0 || claims.RunID != uuid.Nil {
		t.Fatal("dispatcher obtained ordinary run capabilities", err)
	}
	rows, err := f.r.db.QueryxContext(context.Background(), `SELECT result_json FROM supervision_effort_transitions`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(value, f.token) {
			t.Fatal("transition persisted bearer")
		}
	}
}

func TestSupervisorDispatchBindsOwnerScopeProfileExpiryAndDispatcher(t *testing.T) {
	for _, variant := range []string{"wrong-owner", "wrong-scope", "expired", "too-long", "profile-unavailable"} {
		t.Run(variant, func(t *testing.T) {
			f := newDispatchFixture(t)
			switch variant {
			case "wrong-owner":
				f.owner.ID = "other"
			case "wrong-scope":
				f.owner.Scopes = []string{"agent-manager:read"}
			case "expired":
				f.req.ExpiresAt = timestamppb.New(f.now)
			case "too-long":
				f.req.ExpiresAt = timestamppb.New(f.now.Add(31 * 24 * time.Hour))
			case "profile-unavailable":
				f.s.dispatchProfile = func(context.Context, string) error { return errors.New("missing") }
			}
			if _, err := f.s.IssueDispatch(context.Background(), f.req, f.owner); err == nil || f.token != "" {
				t.Fatal("invalid issuance accepted")
			}
		})
	}
	f := newDispatchFixture(t)
	f.issue(t)
	for _, variant := range []string{"team", "member", "authorization", "effort", "forged", "ordinary-token"} {
		t.Run(variant, func(t *testing.T) {
			req := f.wake("wake")
			token := f.token
			switch variant {
			case "team":
				req.TeamId = "other"
			case "member":
				req.MemberId = "other"
			case "authorization":
				req.AuthorizationId = uuid.NewString()
			case "effort":
				req.EffortRef = "business:other"
			case "forged":
				token += "x"
			case "ordinary-token":
				token, _ = identity.GenerateToken(&identity.Claims{Subject: "owner", ExpiresAt: f.now.Add(time.Hour).Unix()}, f.secret)
			}
			if _, err := f.s.AdmitDispatch(context.Background(), req, token); err == nil {
				t.Fatal("unbound dispatcher accepted")
			}
		})
	}
}

func TestSupervisorDispatchIssuanceReportsSafeRefusalPredicate(t *testing.T) {
	for _, tc := range []struct {
		predicate string
		change    func(*dispatchFixture)
	}{
		{"signer_unavailable", func(f *dispatchFixture) { f.s.dispatchSecret = nil }},
		{"credential_provisioner_unavailable", func(f *dispatchFixture) { f.s.dispatchProvision = nil }},
		{"profile_verifier_unavailable", func(f *dispatchFixture) { f.s.dispatchProfile = nil }},
		{"enrollment_withdrawn", func(f *dispatchFixture) { f.e.Withdrawn = true }},
		{"enrollment_owner_mismatch", func(f *dispatchFixture) { f.owner.ID = "private-other-owner" }},
		{"supervisor_owner_mismatch", func(f *dispatchFixture) { f.e.SupervisorOwnerSubject = "private-other-owner" }},
		{"enrollment_scope_mismatch", func(f *dispatchFixture) { f.e.SupervisorScope = "private-service:scope" }},
		{"owner_scope_missing", func(f *dispatchFixture) { f.owner.Scopes = []string{"agent-manager:write", "private-service:scope"} }},
		{"expiry_not_future", func(f *dispatchFixture) { f.req.ExpiresAt = timestamppb.New(f.now) }},
		{"expiry_exceeds_maximum", func(f *dispatchFixture) { f.req.ExpiresAt = timestamppb.New(f.now.Add(31 * 24 * time.Hour)) }},
		{"expiry_exceeds_enrollment", func(f *dispatchFixture) { f.e.AuthorityExpiresAt = timestamppb.New(f.now.Add(time.Minute)) }},
	} {
		t.Run(tc.predicate, func(t *testing.T) {
			f := newDispatchFixture(t)
			tc.change(f)
			_, observation, err := f.r.GetEffort(t.Context(), f.e.EffortRef)
			if err != nil {
				t.Fatal(err)
			}
			expected := f.e.Revision
			f.e.Revision++
			if err := f.r.SaveEffort(t.Context(), f.e, observation, expected, "diagnostic-fixture", "fixture"); err != nil {
				t.Fatal(err)
			}
			f.req.ExpectedRevision = f.e.Revision
			_, err = f.s.IssueDispatch(t.Context(), f.req, f.owner)
			want := ErrDispatchAuthority.Error() + ": issue-dispatch predicate=" + tc.predicate
			if !errors.Is(err, ErrDispatchAuthority) || err.Error() != want {
				t.Fatalf("refusal must retain the sentinel and only a fixed predicate: got %v, want %q", err, want)
			}
			stored, _, err := f.r.GetEffort(t.Context(), f.e.EffortRef)
			if err != nil || stored.Revision != f.req.ExpectedRevision || stored.DispatchAuthorization != nil || f.token != "" {
				t.Fatal("refused issuance changed authorization or provisioned a credential", err)
			}
		})
	}
}

func TestSupervisorDispatchIssuanceRequiresExistingScopeCeiling(t *testing.T) {
	for _, scope := range []string{SupervisorDispatchScope, "*", "agent-manager:*", "*:supervise", "agent-manager:write", "agent-manager.read", "agent-manager:read"} {
		t.Run(scope, func(t *testing.T) {
			f := newDispatchFixture(t)
			f.owner.Scopes = []string{scope}
			enrollment, err := f.s.IssueDispatch(t.Context(), f.req, f.owner)
			allowed := scope == SupervisorDispatchScope || scope == "*" || scope == "agent-manager:*" || scope == "*:supervise"
			if !allowed {
				if !errors.Is(err, ErrDispatchAuthority) || f.token != "" {
					t.Fatal("operator or coarse write permission widened into supervisor authority", err)
				}
				return
			}
			if err != nil || len(enrollment.GetDispatchAuthorization().GetScopes()) != 1 || enrollment.DispatchAuthorization.Scopes[0] != SupervisorDispatchScope {
				t.Fatal("existing ceiling must attenuate to the exact supervisor scope", err)
			}
			if len(f.owner.Scopes) != 1 || f.owner.Scopes[0] != scope {
				t.Fatal("issuance mutated the owner's scopes")
			}
		})
	}
}

func TestSupervisorDispatchFutureWakesKeepRateAllowanceAcrossRestart(t *testing.T) {
	f := newDispatchFixture(t)
	f.issue(t)
	ctx := context.Background()
	first := f.wake("first")
	for i := 0; i < 2; i++ {
		if a, err := f.s.AdmitDispatch(ctx, first, f.token); err != nil || a.DispatchedRuns != 1 {
			t.Fatal("replay spent or lost admission", err)
		}
	}
	changed := proto.Clone(first).(*api.CreateSupervisorRunRequest)
	changed.TaskId = uuid.NewString()
	if _, err := f.s.AdmitDispatch(ctx, changed, f.token); err == nil {
		t.Fatal("replay substituted task")
	}
	if _, err := f.s.AdmitDispatch(ctx, f.wake("early"), f.token); err == nil {
		t.Fatal("rate limit bypassed")
	}
	for i := 0; i < 2; i++ {
		f.now = f.now.Add(time.Minute)
		restarted := NewEffortService(f.r, nil, nil, EffortDiscoveryConfig{})
		restarted.now = f.s.now
		restarted.ConfigureDispatch(f.secret, nil, nil)
		if a, err := restarted.AdmitDispatch(ctx, f.wake(uuid.NewString()), f.token); err != nil || a.DispatchedRuns != uint32(i+2) {
			t.Fatal("restart lost bounded allowance", err)
		}
	}
	f.now = f.now.Add(time.Minute)
	if _, err := f.s.AdmitDispatch(ctx, f.wake("exhausted"), f.token); err == nil {
		t.Fatal("cumulative allowance widened")
	}
}

func TestSupervisorDispatchRevocationWithdrawalAndExpiryInvalidateChildren(t *testing.T) {
	for _, variant := range []string{"revoke", "withdraw", "amend", "expire"} {
		t.Run(variant, func(t *testing.T) {
			f := newDispatchFixture(t)
			f.issue(t)
			ctx := context.Background()
			req := f.wake("admitted")
			if _, err := f.s.AdmitDispatch(ctx, req, f.token); err != nil {
				t.Fatal(err)
			}
			a := f.e.DispatchAuthorization
			claims := &identity.Claims{Subject: "owner", Scopes: []string{"agent-manager:supervise"}, ProfileKey: "qualified", ExpiresAt: a.ExpiresAt.AsTime().Unix(), DispatchEffortRef: f.e.EffortRef, DispatchAuthorizationID: a.AuthorizationId}
			if err := f.s.CheckDispatchIdentity(ctx, claims); err != nil {
				t.Fatal(err)
			}
			e, _, err := f.r.GetEffort(ctx, f.e.EffortRef)
			if err != nil {
				t.Fatal(err)
			}
			switch variant {
			case "revoke":
				_, err = f.s.RevokeDispatch(ctx, &api.RevokeSupervisorDispatchRequest{EffortRef: e.EffortRef, AuthorizationId: a.AuthorizationId, ExpectedRevision: e.Revision, Reason: "operator withdrawal", IdempotencyKey: "revoke"}, f.owner)
			case "withdraw":
				_, err = f.s.Withdraw(ctx, &pb.WithdrawEffortRequest{EffortRef: e.EffortRef, ExpectedRevision: e.Revision, Reason: "retired", IdempotencyKey: "withdraw"}, f.owner)
			case "amend":
				_, err = f.s.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "amend"}, f.owner)
			case "expire":
				f.now = a.ExpiresAt.AsTime()
			}
			if err != nil {
				t.Fatal(err)
			}
			if f.s.CheckDispatchIdentity(ctx, claims) == nil {
				t.Fatal("child authority survived termination")
			}
			if _, err = f.s.AdmitDispatch(ctx, req, f.token); err == nil {
				t.Fatal("replay resurrected terminated authority")
			}
			if _, err = f.s.IssueDispatch(ctx, f.req, f.owner); err == nil {
				t.Fatal("issuance replay resurrected revoked/expired grant")
			}
		})
	}
}
