package mfa

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"

	"scenario-authenticator/internal/accounts"
)

type memoryCustodian struct {
	mu     sync.Mutex
	values map[string]string
}

func (c *memoryCustodian) Put(userID, slot, secret string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.values == nil {
		c.values = map[string]string{}
	}
	c.values[userID+"/"+slot] = secret
	return nil
}

func (c *memoryCustodian) Resolve(userID, slot string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, ok := c.values[userID+"/"+slot]
	if !ok {
		return "", sql.ErrNoRows
	}
	return value, nil
}

func (c *memoryCustodian) Delete(userID, slot string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.values, userID+"/"+slot)
	return nil
}

func TestStoreEncryptsSeedsAndConsumesFactorsOnce(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(accounts.Schema),
		apidb.SchemaProviderFunc(Schema),
	))
	key := []byte("01234567890123456789012345678901")
	store, err := NewStore(db, key, nil)
	require.NoError(t, err)

	ctx := context.Background()
	enrollment, err := store.BeginEnrollment(ctx, "user-1", "default", "owner@example.com")
	require.NoError(t, err)
	parsed, err := url.Parse(enrollment.ProvisioningURI)
	require.NoError(t, err)
	secret := parsed.Query().Get("secret")
	require.NotEmpty(t, secret)
	code, err := Code(secret, time.Now().UTC())
	require.NoError(t, err)

	recoveryCodes, err := store.ConfirmEnrollment(ctx, "user-1", enrollment.ID, code)
	require.NoError(t, err)
	require.Len(t, recoveryCodes, recoveryCount)
	enrolled, err := store.Enrolled(ctx, "user-1")
	require.NoError(t, err)
	require.True(t, enrolled)

	var ciphertext string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT secret_ciphertext FROM mfa_enrollments WHERE user_id=?`, "user-1").Scan(&ciphertext))
	require.NotContains(t, ciphertext, secret)
	var storedHash string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT code_hash FROM mfa_recovery_codes WHERE user_id=? LIMIT 1`, "user-1").Scan(&storedHash))
	require.NotContains(t, storedHash, recoveryCodes[0])

	challenge, _, err := store.StartChallenge(ctx, "user-1")
	require.NoError(t, err)
	valid, err := store.VerifyLogin(ctx, "user-1", challenge, code, "")
	require.NoError(t, err)
	require.True(t, valid)
	valid, err = store.VerifyLogin(ctx, "user-1", challenge, code, "")
	require.ErrorIs(t, err, ErrChallengeInvalid)
	require.False(t, valid)

	recoveryChallenge, _, err := store.StartChallenge(ctx, "user-1")
	require.NoError(t, err)
	valid, err = store.VerifyLogin(ctx, "user-1", recoveryChallenge, "", recoveryCodes[0])
	require.NoError(t, err)
	require.True(t, valid)
	replayChallenge, _, err := store.StartChallenge(ctx, "user-1")
	require.NoError(t, err)
	valid, err = store.VerifyLogin(ctx, "user-1", replayChallenge, "", recoveryCodes[0])
	require.ErrorIs(t, err, ErrCodeInvalid)
	require.False(t, valid)

	require.NoError(t, store.RemoveEnrollment(ctx, "user-1"))
	enrolled, err = store.Enrolled(ctx, "user-1")
	require.NoError(t, err)
	require.False(t, enrolled)
	require.True(t, strings.Contains(enrollment.ProvisioningURI, "otpauth://totp/"))
}

func TestStoreWithCustodianKeepsSeedsOutOfDatabase(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(accounts.Schema),
		apidb.SchemaProviderFunc(Schema),
	))
	custodian := &memoryCustodian{values: map[string]string{}}
	store, err := NewStoreWithCustodian(db, custodian, nil)
	require.NoError(t, err)
	ctx := context.Background()
	enrollment, err := store.BeginEnrollment(ctx, "user-2", "default", "owner@example.com")
	require.NoError(t, err)
	parsed, err := url.Parse(enrollment.ProvisioningURI)
	require.NoError(t, err)
	code, err := Code(parsed.Query().Get("secret"), time.Now().UTC())
	require.NoError(t, err)
	_, err = store.ConfirmEnrollment(ctx, "user-2", enrollment.ID, code)
	require.NoError(t, err)

	var ciphertext, secretRef string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT secret_ciphertext,secret_ref FROM mfa_enrollments WHERE user_id=?`, "user-2").Scan(&ciphertext, &secretRef))
	require.Empty(t, ciphertext)
	require.NotEmpty(t, secretRef)
	custodian.mu.Lock()
	require.Len(t, custodian.values, 1)
	custodian.mu.Unlock()
	require.NoError(t, store.RemoveEnrollment(ctx, "user-2"))
	custodian.mu.Lock()
	require.Empty(t, custodian.values)
	custodian.mu.Unlock()
}
