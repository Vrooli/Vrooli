package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/localprincipal"

	apidb "github.com/vrooli/api-core/database"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"

	dbtest "github.com/vrooli/api-core/databasetest"
	"scenario-authenticator/internal/accounts"
	"scenario-authenticator/internal/audit"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/localexchange"
	"scenario-authenticator/internal/realm"
	"scenario-authenticator/internal/redisstate"
	"scenario-authenticator/internal/sessions"

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
		Repo:            repo,
		Signer:          signer,
		Sessions:        sessions.NewManager(redisstate.NewMemory(), nil),
		Audit:           auditLogger,
		MachineBindings: repo.(accounts.MachineBindingStore),
		Clock:           clk,
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

func TestDuplicateEmailAlreadyExists(t *testing.T) {
	h := newHarness(t)
	h.register(t, "dup@b.co", "Passw0rd")
	_, err := h.h.Register(context.Background(), connect.NewRequest(&accountsv1.RegisterRequest{Email: "dup@b.co", Password: "Passw0rd"}))
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("want AlreadyExists, got %v", err)
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
	if sessions, err := h.svc.ListSessions(context.Background(), second.Msg.Tokens.AccessToken); err != nil || len(sessions) != 0 {
		t.Fatalf("sessions after change = %d, err=%v", len(sessions), err)
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
