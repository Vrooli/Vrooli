package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"

	"device-sync-hub/internal/auth"
	"device-sync-hub/internal/modules"
	internalrealtime "device-sync-hub/internal/realtime"
	"device-sync-hub/internal/server"

	"github.com/vrooli/api-core/schedule"

	devicesH "device-sync-hub/handlers/devices"
	healthH "device-sync-hub/handlers/health"
	identityH "device-sync-hub/handlers/identity"

	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices"
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices/devices_v1connect"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/identity"
	identityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/identity/identity_v1connect"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

// TestE2E_OwnerFirstRunFlow drives the WHOLE first-run owner bootstrap through
// the real hub server wiring against a stub scenario-authenticator — exactly the
// path a fresh user takes:
//
//	IdentityService.Register (same-origin) → hub forwards to the authenticator
//	  (discovery-resolved) → relays the issued RS256 JWT
//	→ DevicesService.SetupOwnerDevice with that JWT → hub VERIFIES it locally
//	  against the authenticator's JWKS → claims the hub + trusts the caller
//	→ returns the one-time device token.
//
// No browser cross-origin call; no AUTH_SERVICE_URL; no per-request /validate.
// This is the deterministic stand-in for the live walkthrough (the real
// authenticator is currently un-bootable here due to a platform postgres issue).
func TestE2E_OwnerFirstRunFlow(t *testing.T) {
	signingKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Stub scenario-authenticator: issues an RS256 JWT on register/login and
	// publishes its public key as JWKS.
	authStub := newAuthStub(t, signingKey)
	defer authStub.Close()

	// Real hub server wiring (mirrors main.go) pointed at the stub via a static
	// discovery resolver — the one seam we substitute.
	resolver := discovery.NewStaticResolver(authStub.URL)
	logger := log.New(io.Discard, "", 0)
	clk := schedule.System()

	dsn := "file:" + filepath.Join(t.TempDir(), "e2e.db") +
		"?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)"
	db, err := database.Open(context.Background(), database.Config{
		Driver: database.DriverSQLite, DSN: dsn, MaxOpenConns: 1, MaxIdleConns: 1,
	})
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...))

	authClient := auth.NewClient(auth.Config{Resolver: resolver})
	hub := internalrealtime.NewHub(clk)

	srv := server.New(
		server.Deps{Clock: clk, Logger: logger},
		healthH.Module(db, "device-sync-hub-api", "test"),
		devicesH.Module(db, clk, authClient, hub, logger),
		identityH.Module(resolver, logger),
	)
	// Owner middleware injects the verified identity for owner-gated RPCs.
	live := httptest.NewServer(auth.Middleware(authClient, logger)(srv.Handler()))
	defer live.Close()

	identityClient := identityconnect.NewIdentityServiceClient(live.Client(), live.URL)
	devicesClient := devicesconnect.NewDevicesServiceClient(live.Client(), live.URL)
	ctx := context.Background()

	// 1) Create the owner account (same-origin) → the hub relays the JWT.
	reg, err := identityClient.Register(ctx, connect.NewRequest(&identityv1.RegisterRequest{
		Email: "owner@example.com", Password: "Str0ng!pw",
	}))
	require.NoError(t, err, "register through the hub's IdentityService")
	token := reg.Msg.GetToken()
	require.NotEmpty(t, token, "hub must relay the issued owner JWT")
	assert.Equal(t, "owner@example.com", reg.Msg.GetEmail())

	// 2) Make this the first device — owner-authed; the hub verifies the JWT
	//    LOCALLY against the stub's JWKS (no call back to the authenticator).
	setupReq := connect.NewRequest(&devicesv1.SetupOwnerDeviceRequest{
		Profile: &devicesv1.DeviceProfile{DeviceName: "Workstation", Kind: "laptop"},
	})
	setupReq.Header().Set("Authorization", "Bearer "+token)
	setup, err := devicesClient.SetupOwnerDevice(ctx, setupReq)
	require.NoError(t, err, "SetupOwnerDevice with the relayed owner JWT")
	require.NotNil(t, setup.Msg.GetDevice())
	assert.Equal(t, devicesv1.TrustState_TRUST_STATE_TRUSTED, setup.Msg.GetDevice().GetTrustState())
	assert.NotEmpty(t, setup.Msg.GetDeviceToken(), "hub must mint a one-time device token")
	assert.Equal(t, "owner@example.com", setup.Msg.GetDevice().GetOwnerId(), "device keyed to the verified owner")

	// 3) A garbage token must NOT be accepted (local verification is real).
	badReq := connect.NewRequest(&devicesv1.SetupOwnerDeviceRequest{
		Profile: &devicesv1.DeviceProfile{DeviceName: "Intruder"},
	})
	badReq.Header().Set("Authorization", "Bearer not.a.real.jwt")
	_, err = devicesClient.SetupOwnerDevice(ctx, badReq)
	require.Error(t, err, "a forged token must be rejected")
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// --- stub scenario-authenticator ---------------------------------------------

// stubAccounts is a Connect AccountsService stub that issues an RS256 JWT
// (aud-scoped to the default realm) on Register/Login — the real authenticator's
// P0 contract that the hub now forwards to over Connect.
type stubAccounts struct {
	accountsconnect.UnimplementedAccountsServiceHandler
	t   *testing.T
	key *rsa.PrivateKey
}

func (s *stubAccounts) issue(email string) (*accountsv1.Account, *accountsv1.TokenPair) {
	if email == "" {
		email = "owner@example.com"
	}
	return &accountsv1.Account{Id: email, Email: email, Realm: "default"},
		&accountsv1.TokenPair{AccessToken: signOwnerJWT(s.t, s.key, email), RefreshToken: "refresh-xyz"}
}

func (s *stubAccounts) Register(_ context.Context, req *connect.Request[accountsv1.RegisterRequest]) (*connect.Response[accountsv1.RegisterResponse], error) {
	acc, tok := s.issue(req.Msg.GetEmail())
	return connect.NewResponse(&accountsv1.RegisterResponse{Account: acc, Tokens: tok}), nil
}

func (s *stubAccounts) Login(_ context.Context, req *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
	acc, tok := s.issue(req.Msg.GetEmail())
	return connect.NewResponse(&accountsv1.LoginResponse{Account: acc, Tokens: tok}), nil
}

func newAuthStub(t *testing.T, key *rsa.PrivateKey) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	accPath, accHandler := accountsconnect.NewAccountsServiceHandler(&stubAccounts{t: t, key: key})
	mux.Handle(accPath, accHandler)
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		eBytes := big.NewInt(int64(key.PublicKey.E)).Bytes()
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "test",
			"n": base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(eBytes),
		}}})
	})
	return httptest.NewServer(mux)
}

// signOwnerJWT mints an RS256 JWT shaped like scenario-authenticator's (user_id,
// email, roles + issuer), keyed to email so OwnerID is stable across the flow.
func signOwnerJWT(t *testing.T, key *rsa.PrivateKey, email string) string {
	t.Helper()
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	header := enc(map[string]string{"alg": "RS256", "typ": "JWT"})
	payload := enc(map[string]any{
		"user_id": email,
		"email":   email,
		"roles":   []string{"user"},
		"iss":     auth.AuthScenarioSlug,
		"aud":     auth.AuthExpectedAudience,
		"iat":     time.Now().Add(-time.Minute).Unix(),
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signingInput := header + "." + payload
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	require.NoError(t, err)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}
