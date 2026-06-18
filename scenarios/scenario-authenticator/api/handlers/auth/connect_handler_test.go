package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"connectrpc.com/connect"

	apidb "github.com/vrooli/api-core/database"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"

	"scenario-authenticator/internal/accounts"
	"scenario-authenticator/internal/audit"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/clock"
	"scenario-authenticator/internal/realm"
	"scenario-authenticator/internal/redisstate"
	"scenario-authenticator/internal/sessions"
	dbtest "scenario-authenticator/internal/testutil/db"
)

type harness struct {
	h      *connectHandler
	svc    *accounts.Service
	signer *authcrypto.Signer
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
	clk := clock.System{}
	svc := accounts.NewService(accounts.ServiceConfig{
		Repo:     accounts.NewSQLiteRepository(d, clk),
		Signer:   signer,
		Sessions: sessions.NewManager(redisstate.NewMemory(), nil),
		Audit:    audit.NewSQLiteLogger(d, clk),
		Clock:    clk,
	})
	return &harness{h: NewConnectHandler(Deps{Service: svc}), svc: svc, signer: signer}
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
	clk := clock.System{}
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
