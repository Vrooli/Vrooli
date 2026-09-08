package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/localprincipal"

	apidb "github.com/vrooli/api-core/database"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"

	"scenario-authenticator/internal/accounts"
	"scenario-authenticator/internal/audit"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/authorization"
	"scenario-authenticator/internal/localexchange"
	"scenario-authenticator/internal/realm"
	"scenario-authenticator/internal/redisstate"
	"scenario-authenticator/internal/sessions"

	dbtest "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"
)

type harness struct {
	h      *connectHandler
	svc    *accounts.Service
	signer *authcrypto.Signer
	audit  audit.Logger
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	d := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(accounts.Schema),
		apidb.SchemaProviderFunc(audit.Schema),
		apidb.SchemaProviderFunc(authorization.Schema),
	); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	keys := authcrypto.NewKeysFromPair(priv, &priv.PublicKey)
	signer := authcrypto.NewSigner(keys, authcrypto.SignerConfig{Issuer: realm.Issuer})
	clk := schedule.System()
	repo := accounts.NewSQLiteRepository(d, clk)
	auditLogger := audit.NewSQLiteLogger(d, clk)
	svc := accounts.NewService(accounts.ServiceConfig{
		Repo:              repo,
		Signer:            signer,
		Sessions:          sessions.NewManager(redisstate.NewMemory(), nil),
		Audit:             auditLogger,
		Authorization:     authorization.NewService(repo.(authorization.ScopeStore), auditLogger),
		MachineBindings:   repo.(accounts.MachineBindingStore),
		Clock:             clk,
		ResourceAudiences: map[string]string{"bridge": "scenario-authenticator:bridge"},
	})
	return &harness{h: NewConnectHandler(Deps{Service: svc}), svc: svc, signer: signer, audit: auditLogger}
}

func (h *harness) register(t *testing.T, email, pw string) *accountsv1.RegisterResponse {
	t.Helper()
	resp, err := h.h.Register(context.Background(), connect.NewRequest(&accountsv1.RegisterRequest{Email: email, Password: pw}))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return resp.Msg
}

func TestRegisterLoginValidateRoundTrip(t *testing.T) {
	h := newHarness(t)
	reg := h.register(t, "a@b.co", "Passw0rd")
	if reg.Account.Id == "" || reg.Tokens.AccessToken == "" || reg.Tokens.RefreshToken == "" {
		t.Fatalf("incomplete register response: %+v", reg)
	}
	if reg.Account.Realm != realm.DefaultID {
		t.Fatalf("realm = %q", reg.Account.Realm)
	}
	if len(reg.Account.Roles) != 1 || reg.Account.Roles[0] != "admin" {
		t.Fatalf("first account roles = %v, want [admin]", reg.Account.Roles)
	}

	login, err := h.h.Login(context.Background(), connect.NewRequest(&accountsv1.LoginRequest{Email: "a@b.co", Password: "Passw0rd"}))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	val, err := h.h.Validate(context.Background(), connect.NewRequest(&accountsv1.ValidateRequest{AccessToken: login.Msg.Tokens.AccessToken}))
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !val.Msg.Valid || val.Msg.UserId != reg.Account.Id || val.Msg.Realm != realm.DefaultID {
		t.Fatalf("unexpected validate: %+v", val.Msg)
	}
}

func TestLoginSetsHttpOnlySameOriginCookies(t *testing.T) {
	h := newHarness(t)
	t.Setenv("VROOLI_AUTH_COOKIE_SECURE", "true")
	h.register(t, "cookie@b.co", "Passw0rd")
	req := connect.NewRequest(&accountsv1.LoginRequest{Email: "cookie@b.co", Password: "Passw0rd"})
	req.Header().Set("X-Forwarded-Proto", "https")
	response, err := h.h.Login(context.Background(), req)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	cookies := (&http.Response{Header: response.Header()}).Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies=%v, want access and refresh cookies", cookies)
	}
	for _, cookie := range cookies {
		if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatalf("cookie %q lacks browser session protections: %#v", cookie.Name, cookie)
		}
	}
}

func TestRegisteredResourceAudienceIsStampedAndPreservedOnRefresh(t *testing.T) {
	h := newHarness(t)
	registered, err := h.h.Register(context.Background(), connect.NewRequest(&accountsv1.RegisterRequest{
		Email: "resource@b.co", Password: "Passw0rd", Resource: "bridge",
	}))
	if err != nil {
		t.Fatalf("resource register: %v", err)
	}
	if registered.Msg.Tokens.Audience != "scenario-authenticator:bridge" {
		t.Fatalf("issued audience = %q", registered.Msg.Tokens.Audience)
	}
	validated, err := h.h.Validate(context.Background(), connect.NewRequest(&accountsv1.ValidateRequest{AccessToken: registered.Msg.Tokens.AccessToken}))
	if err != nil || !validated.Msg.Valid || validated.Msg.Audience != "scenario-authenticator:bridge" {
		t.Fatalf("resource validation = %#v, err=%v", validated.Msg, err)
	}
	refreshed, err := h.h.Refresh(context.Background(), connect.NewRequest(&accountsv1.RefreshRequest{RefreshToken: registered.Msg.Tokens.RefreshToken}))
	if err != nil {
		t.Fatalf("resource refresh: %v", err)
	}
	if refreshed.Msg.Tokens.Audience != "scenario-authenticator:bridge" {
		t.Fatalf("refreshed audience = %q", refreshed.Msg.Tokens.Audience)
	}

	_, err = h.h.Register(context.Background(), connect.NewRequest(&accountsv1.RegisterRequest{
		Email: "unknown-resource@b.co", Password: "Passw0rd", Resource: "missing",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unknown resource error = %v, want invalid argument", err)
	}
}

func TestDuplicateEmailAlreadyExists(t *testing.T) {
	h := newHarness(t)
	h.register(t, "dup@b.co", "Passw0rd")
	_, err := h.h.Register(context.Background(), connect.NewRequest(&accountsv1.RegisterRequest{Email: "dup@b.co", Password: "Passw0rd"}))
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("want AlreadyExists, got %v", err)
	}
}

func TestOnlyAdminMayManageAnotherPrincipalScopes(t *testing.T) {
	h := newHarness(t)
	admin := h.register(t, "admin@b.co", "Passw0rd")
	member := h.register(t, "member@b.co", "Passw0rd")
	target := h.register(t, "target@b.co", "Passw0rd")

	granted, err := h.h.GrantScope(context.Background(), connect.NewRequest(&accountsv1.GrantScopeRequest{
		AccessToken: admin.Tokens.AccessToken, PrincipalId: member.Account.Id, Scope: "demo:write",
	}))
	if err != nil || len(granted.Msg.Scopes) != 1 || granted.Msg.Scopes[0] != "demo:write" {
		t.Fatalf("admin grant = %#v, %v", granted, err)
	}
	_, err = h.h.GrantScope(context.Background(), connect.NewRequest(&accountsv1.GrantScopeRequest{
		AccessToken: member.Tokens.AccessToken, PrincipalId: target.Account.Id, Scope: "demo:write",
	}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("ordinary principal cross-target grant error = %v, want unauthenticated", err)
	}
}

func TestAdminRoleManagementAndLastAdminProtection(t *testing.T) {
	h := newHarness(t)
	admin := h.register(t, "roles-admin@b.co", "Passw0rd")
	member := h.register(t, "roles-member@b.co", "Passw0rd")

	updated, err := h.h.SetRoles(context.Background(), connect.NewRequest(&accountsv1.SetRolesRequest{
		AccessToken: admin.Tokens.AccessToken, PrincipalId: member.Account.Id, Roles: []string{"admin", "user", "admin"},
	}))
	if err != nil {
		t.Fatalf("admin set roles: %v", err)
	}
	if len(updated.Msg.Roles) != 2 || updated.Msg.Roles[0] != "admin" || updated.Msg.Roles[1] != "user" {
		t.Fatalf("normalized roles = %v", updated.Msg.Roles)
	}

	_, err = h.h.SetRoles(context.Background(), connect.NewRequest(&accountsv1.SetRolesRequest{
		AccessToken: member.Tokens.AccessToken, PrincipalId: admin.Account.Id, Roles: []string{"user"},
	}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("ordinary principal role-management error = %v, want unauthenticated", err)
	}

	_, err = h.h.SetRoles(context.Background(), connect.NewRequest(&accountsv1.SetRolesRequest{
		AccessToken: admin.Tokens.AccessToken, PrincipalId: admin.Account.Id, Roles: []string{"user"},
	}))
	if err != nil {
		t.Fatalf("removing one of two admins: %v", err)
	}
	memberAdmin, err := h.h.Login(context.Background(), connect.NewRequest(&accountsv1.LoginRequest{Email: "roles-member@b.co", Password: "Passw0rd"}))
	if err != nil {
		t.Fatalf("member admin login: %v", err)
	}
	_, err = h.h.SetRoles(context.Background(), connect.NewRequest(&accountsv1.SetRolesRequest{
		AccessToken: memberAdmin.Msg.Tokens.AccessToken, PrincipalId: member.Account.Id, Roles: []string{"user"},
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("last admin removal error = %v, want failed precondition", err)
	}
	auditRecords, err := h.audit.List(context.Background(), audit.Filter{Action: "account.roles.updated"})
	if err != nil || len(auditRecords) < 2 {
		t.Fatalf("role audit records = %v, err=%v", auditRecords, err)
	}
}

func TestWeakPasswordAndBadEmail(t *testing.T) {
	h := newHarness(t)
	_, err := h.h.Register(context.Background(), connect.NewRequest(&accountsv1.RegisterRequest{Email: "x@y.co", Password: "weak"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument for weak pw, got %v", err)
	}
	_, err = h.h.Register(context.Background(), connect.NewRequest(&accountsv1.RegisterRequest{Email: "not-an-email", Password: "Passw0rd"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument for bad email, got %v", err)
	}
}

func TestLoginAntiEnumeration(t *testing.T) {
	h := newHarness(t)
	h.register(t, "real@b.co", "Passw0rd")

	// Wrong password and unknown account must yield the SAME code + message.
	_, errWrong := h.h.Login(context.Background(), connect.NewRequest(&accountsv1.LoginRequest{Email: "real@b.co", Password: "Wrong0rd!"}))
	_, errUnknown := h.h.Login(context.Background(), connect.NewRequest(&accountsv1.LoginRequest{Email: "ghost@b.co", Password: "Passw0rd"}))
	if connect.CodeOf(errWrong) != connect.CodeUnauthenticated || connect.CodeOf(errUnknown) != connect.CodeUnauthenticated {
		t.Fatalf("codes differ: %v / %v", errWrong, errUnknown)
	}
	if errWrong.Error() != errUnknown.Error() {
		t.Fatalf("anti-enumeration leak: %q vs %q", errWrong.Error(), errUnknown.Error())
	}
}

func TestChangePasswordRehashesAndRevokesSessions(t *testing.T) {
	h := newHarness(t)
	reg := h.register(t, "change@b.co", "Passw0rd")
	second, err := h.h.Login(context.Background(), connect.NewRequest(&accountsv1.LoginRequest{Email: "change@b.co", Password: "Passw0rd"}))
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if sessions, err := h.svc.ListSessions(context.Background(), reg.Tokens.AccessToken); err != nil || len(sessions) != 2 {
		t.Fatalf("sessions before change = %d, err=%v", len(sessions), err)
	}

	changed, err := h.h.ChangePassword(context.Background(), connect.NewRequest(&accountsv1.ChangePasswordRequest{
		AccessToken: reg.Tokens.AccessToken, CurrentPassword: "Passw0rd", NewPassword: "Newpass9",
	}))
	if err != nil {
		t.Fatalf("change password: %v", err)
	}
	if changed.Msg.RevokedSessions != 2 {
		t.Fatalf("revoked sessions = %d, want 2", changed.Msg.RevokedSessions)
	}
	if _, err := h.svc.ListSessions(context.Background(), second.Msg.Tokens.AccessToken); !errors.Is(err, accounts.ErrInvalidCredentials) {
		t.Fatalf("revoked access token still listed sessions: %v", err)
	}
	if _, err := h.h.Login(context.Background(), connect.NewRequest(&accountsv1.LoginRequest{Email: "change@b.co", Password: "Passw0rd"})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("old password accepted: %v", err)
	}
	if _, err := h.h.Login(context.Background(), connect.NewRequest(&accountsv1.LoginRequest{Email: "change@b.co", Password: "Newpass9"})); err != nil {
		t.Fatalf("new password rejected: %v", err)
	}
}

func TestRefreshRotationAndReuseDetection(t *testing.T) {
	h := newHarness(t)
	reg := h.register(t, "r@b.co", "Passw0rd")
	first := reg.Tokens.RefreshToken

	rot, err := h.h.Refresh(context.Background(), connect.NewRequest(&accountsv1.RefreshRequest{RefreshToken: first}))
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if rot.Msg.Tokens.RefreshToken == first {
		t.Fatal("refresh token not rotated")
	}
	// Replaying the first (now rotated-out) token is reuse → rejected, and the
	// whole family is revoked, so the rotated token also stops working.
	if _, err := h.h.Refresh(context.Background(), connect.NewRequest(&accountsv1.RefreshRequest{RefreshToken: first})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("reuse not rejected: %v", err)
	}
	if _, err := h.h.Refresh(context.Background(), connect.NewRequest(&accountsv1.RefreshRequest{RefreshToken: rot.Msg.Tokens.RefreshToken})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("family not revoked after reuse: %v", err)
	}
}

func TestRevokeAllRejectsExistingAccessAndRefreshTokens(t *testing.T) {
	h := newHarness(t)
	reg := h.register(t, "revoke-all@b.co", "Passw0rd")

	if _, err := h.svc.RevokeAllSessions(context.Background(), reg.Tokens.AccessToken, accounts.RequestMeta{}); err != nil {
		t.Fatalf("revoke all: %v", err)
	}
	validated, err := h.h.Validate(context.Background(), connect.NewRequest(&accountsv1.ValidateRequest{
		AccessToken: reg.Tokens.AccessToken,
	}))
	if err != nil || validated.Msg.Valid {
		t.Fatalf("revoked access token validation = valid=%v, err=%v", validated.Msg.Valid, err)
	}
	if _, err := h.h.Refresh(context.Background(), connect.NewRequest(&accountsv1.RefreshRequest{
		RefreshToken: reg.Tokens.RefreshToken,
	})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("revoked refresh token error = %v, want unauthenticated", err)
	}
}

func TestLogoutBlacklistsToken(t *testing.T) {
	h := newHarness(t)
	reg := h.register(t, "lo@b.co", "Passw0rd")
	access := reg.Tokens.AccessToken

	if _, err := h.h.Logout(context.Background(), connect.NewRequest(&accountsv1.LogoutRequest{AccessToken: access})); err != nil {
		t.Fatalf("logout: %v", err)
	}
	val, err := h.h.Validate(context.Background(), connect.NewRequest(&accountsv1.ValidateRequest{AccessToken: access}))
	if err != nil {
		t.Fatalf("validate after logout: %v", err)
	}
	if val.Msg.Valid {
		t.Fatal("blacklisted token still validates")
	}
}

// TestCrossAudienceRejected covers OT-P0-008 at the handler level: a token
// minted for a different realm aud is rejected even though only the default
// realm exists.
func TestCrossAudienceRejected(t *testing.T) {
	h := newHarness(t)
	crossTok, err := h.signer.Sign(authcrypto.TokenInput{UserID: "u1", Audience: "scenario-authenticator:other-realm"})
	if err != nil {
		t.Fatalf("sign cross-aud: %v", err)
	}
	val, err := h.h.Validate(context.Background(), connect.NewRequest(&accountsv1.ValidateRequest{AccessToken: crossTok}))
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if val.Msg.Valid {
		t.Fatal("cross-aud token accepted — cross-tenant leak")
	}
}

func TestAccountLockout(t *testing.T) {
	d := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(accounts.Schema), apidb.SchemaProviderFunc(audit.Schema)); err != nil {
		t.Fatalf("schemas: %v", err)
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer := authcrypto.NewSigner(authcrypto.NewKeysFromPair(priv, &priv.PublicKey), authcrypto.SignerConfig{Issuer: realm.Issuer})
	clk := schedule.System()
	svc := accounts.NewService(accounts.ServiceConfig{
		Repo: accounts.NewSQLiteRepository(d, clk), Signer: signer,
		Sessions: sessions.NewManager(redisstate.NewMemory(), nil),
		Audit:    audit.NewSQLiteLogger(d, clk), Clock: clk, LockThreshold: 3,
	})
	h := NewConnectHandler(Deps{Service: svc})
	if _, err := h.Register(context.Background(), connect.NewRequest(&accountsv1.RegisterRequest{Email: "lock@b.co", Password: "Passw0rd"})); err != nil {
		t.Fatalf("register: %v", err)
	}
	var lastErr error
	for i := 0; i < 3; i++ {
		_, lastErr = h.Login(context.Background(), connect.NewRequest(&accountsv1.LoginRequest{Email: "lock@b.co", Password: "Wrong0rd!"}))
	}
	if connect.CodeOf(lastErr) != connect.CodeUnauthenticated {
		t.Fatalf("pre-lock want Unauthenticated, got %v", lastErr)
	}
	// Now even the CORRECT password is locked out.
	_, err := h.Login(context.Background(), connect.NewRequest(&accountsv1.LoginRequest{Email: "lock@b.co", Password: "Passw0rd"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("want PermissionDenied (locked), got %v", err)
	}
}

func TestMachinePrincipalExchangeBoundAndUnbound(t *testing.T) {
	h := newHarness(t)
	reg := h.register(t, "machine@b.co", "Passw0rd")
	const (
		machineID = "linux-workstation"
		bound     = "unix:1000"
		unbound   = "unix:1001"
	)

	linked, err := h.h.LinkMachineAccount(context.Background(), connect.NewRequest(&accountsv1.LinkMachineAccountRequest{
		AccessToken:    reg.Tokens.AccessToken,
		MachineId:      machineID,
		LocalPrincipal: bound,
		IsDefault:      true,
	}))
	if err != nil {
		t.Fatalf("link machine account: %v", err)
	}
	if linked.Msg.MachineId != machineID || linked.Msg.LocalPrincipal != bound {
		t.Fatalf("unexpected binding: %+v", linked.Msg)
	}

	ctx := localexchange.WithPeerPrincipal(context.Background(), localprincipal.Principal(bound))
	exchanged, err := h.h.ExchangeMachinePrincipal(ctx, connect.NewRequest(&accountsv1.ExchangeMachinePrincipalRequest{MachineId: machineID}))
	if err != nil {
		t.Fatalf("bound exchange: %v", err)
	}
	if exchanged.Msg.Account.Id != reg.Account.Id || exchanged.Msg.Tokens.AccessToken == "" {
		t.Fatalf("unexpected exchange response: %+v", exchanged.Msg)
	}
	revoked, err := h.h.RevokeMachineAccount(context.Background(), connect.NewRequest(&accountsv1.RevokeMachineAccountRequest{
		AccessToken: reg.Tokens.AccessToken, MachineId: machineID, LocalPrincipal: bound,
	}))
	if err != nil || revoked.Msg.RevokedCount != 1 {
		t.Fatalf("revoke machine binding = %#v, err=%v", revoked.Msg, err)
	}

	_, err = h.h.ExchangeMachinePrincipal(localexchange.WithPeerPrincipal(context.Background(), localprincipal.Principal(unbound)), connect.NewRequest(&accountsv1.ExchangeMachinePrincipalRequest{MachineId: machineID}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unbound exchange code = %v, want unauthenticated", err)
	}
	accepted, err := h.audit.List(context.Background(), audit.Filter{Action: "machine.exchange.accepted"})
	if err != nil || len(accepted) != 1 || !accepted[0].Success {
		t.Fatalf("accepted exchange audit = %+v, err=%v", accepted, err)
	}
	refused, err := h.audit.List(context.Background(), audit.Filter{Action: "machine.exchange.refused"})
	if err != nil || len(refused) != 1 || refused[0].Success {
		t.Fatalf("refused exchange audit = %+v, err=%v", refused, err)
	}
}
