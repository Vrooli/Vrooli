package mfa

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/url"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	mfav1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/mfa"

	"scenario-authenticator/handlers/auth"
	"scenario-authenticator/internal/accounts"
	"scenario-authenticator/internal/audit"
	"scenario-authenticator/internal/authcrypto"
	intmfa "scenario-authenticator/internal/mfa"
	"scenario-authenticator/internal/realm"
	"scenario-authenticator/internal/redisstate"
	"scenario-authenticator/internal/sessions"
)

func TestEnrollmentAndLoginRequireTheSecondFactor(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(accounts.Schema),
		apidb.SchemaProviderFunc(audit.Schema),
		apidb.SchemaProviderFunc(intmfa.Schema),
	))
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keys := authcrypto.NewKeysFromPair(privateKey, &privateKey.PublicKey)
	clk := schedule.System()
	repo := accounts.NewSQLiteRepository(db, clk)
	auditLogger := audit.NewSQLiteLogger(db, clk)
	mfaStore, err := intmfa.NewStore(db, keys.StorageKey("mfa-seeds"), clk)
	require.NoError(t, err)
	svc := accounts.NewService(accounts.ServiceConfig{
		Repo: repo, Signer: authcrypto.NewSigner(keys, authcrypto.SignerConfig{Issuer: realm.Issuer}),
		Sessions: sessions.NewManager(redisstate.NewMemory(), nil), Audit: auditLogger,
		MachineBindings: repo.(accounts.MachineBindingStore), MFA: mfaStore, Clock: clk,
	})
	authHandler := auth.NewConnectHandler(auth.Deps{Service: svc})
	mfaHandler := NewConnectHandler(Deps{Accounts: svc, Store: mfaStore, Audit: auditLogger})
	ctx := context.Background()

	registered, err := authHandler.Register(ctx, connect.NewRequest(&accountsv1.RegisterRequest{Email: "mfa@example.com", Password: "Passw0rd"}))
	require.NoError(t, err)
	accessToken := registered.Msg.Tokens.AccessToken

	started, err := mfaHandler.BeginEnrollment(ctx, connect.NewRequest(&mfav1.BeginEnrollmentRequest{AccessToken: accessToken}))
	require.NoError(t, err)
	parsed, err := url.Parse(started.Msg.ProvisioningUri)
	require.NoError(t, err)
	code, err := intmfa.Code(parsed.Query().Get("secret"), clk.Now())
	require.NoError(t, err)
	confirmed, err := mfaHandler.ConfirmEnrollment(ctx, connect.NewRequest(&mfav1.ConfirmEnrollmentRequest{
		AccessToken: accessToken, EnrollmentId: started.Msg.EnrollmentId, TotpCode: code,
	}))
	require.NoError(t, err)
	require.Len(t, confirmed.Msg.RecoveryCodes, 10)

	challengeResponse, err := authHandler.Login(ctx, connect.NewRequest(&accountsv1.LoginRequest{Email: "mfa@example.com", Password: "Passw0rd"}))
	require.NoError(t, err)
	require.True(t, challengeResponse.Msg.MfaRequired)
	require.NotEmpty(t, challengeResponse.Msg.MfaChallenge)
	require.Empty(t, challengeResponse.Msg.Tokens.AccessToken)

	completed, err := authHandler.Login(ctx, connect.NewRequest(&accountsv1.LoginRequest{
		Email: "mfa@example.com", Password: "Passw0rd", MfaChallenge: challengeResponse.Msg.MfaChallenge, TotpCode: code,
	}))
	require.NoError(t, err)
	require.False(t, completed.Msg.MfaRequired)
	require.NotEmpty(t, completed.Msg.Tokens.AccessToken)

	_, err = authHandler.Login(ctx, connect.NewRequest(&accountsv1.LoginRequest{
		Email: "mfa@example.com", Password: "Passw0rd", MfaChallenge: challengeResponse.Msg.MfaChallenge, TotpCode: code,
	}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	recoveryChallenge, err := authHandler.Login(ctx, connect.NewRequest(&accountsv1.LoginRequest{Email: "mfa@example.com", Password: "Passw0rd"}))
	require.NoError(t, err)
	_, err = authHandler.Login(ctx, connect.NewRequest(&accountsv1.LoginRequest{
		Email: "mfa@example.com", Password: "Passw0rd", MfaChallenge: recoveryChallenge.Msg.MfaChallenge, RecoveryCode: confirmed.Msg.RecoveryCodes[0],
	}))
	require.NoError(t, err)

	_, err = authHandler.Login(ctx, connect.NewRequest(&accountsv1.LoginRequest{Email: "mfa@example.com", Password: "Passw0rd"}))
	require.NoError(t, err)
	_, err = authHandler.Login(ctx, connect.NewRequest(&accountsv1.LoginRequest{
		Email: "mfa@example.com", Password: "Passw0rd", MfaChallenge: recoveryChallenge.Msg.MfaChallenge, RecoveryCode: confirmed.Msg.RecoveryCodes[0],
	}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = mfaHandler.RemoveEnrollment(ctx, connect.NewRequest(&mfav1.RemoveEnrollmentRequest{AccessToken: accessToken}))
	require.NoError(t, err)
	startedEvents, err := auditLogger.List(ctx, audit.Filter{Action: "mfa.enrollment.started"})
	require.NoError(t, err)
	require.Len(t, startedEvents, 1)
	confirmedEvents, err := auditLogger.List(ctx, audit.Filter{Action: "mfa.enrollment.confirmed"})
	require.NoError(t, err)
	require.Len(t, confirmedEvents, 1)
	removedEvents, err := auditLogger.List(ctx, audit.Filter{Action: "mfa.enrollment.removed"})
	require.NoError(t, err)
	require.Len(t, removedEvents, 1)

	_, err = db.ExecContext(ctx, `UPDATE realms SET mfa_required=1 WHERE id=?`, realm.DefaultID)
	require.NoError(t, err)
	_, err = authHandler.Register(ctx, connect.NewRequest(&accountsv1.RegisterRequest{Email: "mandatory@example.com", Password: "Passw0rd"}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	_, err = authHandler.Login(ctx, connect.NewRequest(&accountsv1.LoginRequest{Email: "mandatory@example.com", Password: "Passw0rd"}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}
