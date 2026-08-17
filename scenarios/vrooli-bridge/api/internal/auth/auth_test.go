package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedsession "github.com/vrooli/api-core/operatorsession"
	"github.com/vrooli/api-core/trustposture"
	localenrollment "vrooli-bridge/internal/operatorsession"
)

// fakeBase64URL JSON-encodes v and base64url's it (no padding) — the JWS
// segment encoding.
func seg(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(b)
}

// TestValidate_RejectsNonRS256 proves the algorithm lock: a token whose header
// alg is anything but RS256 (here "none", the classic confusion attack) is
// rejected before any key is consulted.
func TestValidate_RejectsNonRS256(t *testing.T) {
	header := seg(t, map[string]string{"alg": "none", "typ": "JWT"})
	claims := seg(t, map[string]any{"user_id": "u1"})
	token := header + "." + claims + "." + "c2ln" // bogus sig

	c := NewClient(Config{}) // nil resolver — but alg check fires first
	_, err := c.Validate(context.Background(), token)
	require.ErrorIs(t, err, ErrUnauthenticated)
}

// TestValidate_EmptyToken is a fast fail-closed.
func TestValidate_EmptyToken(t *testing.T) {
	c := NewClient(Config{})
	_, err := c.Validate(context.Background(), "   ")
	require.ErrorIs(t, err, ErrUnauthenticated)
}

// TestValidate_NoResolverIsUnavailable proves fail-closed-but-distinguishable:
// a well-formed RS256 token with no way to fetch the signing key yields
// ErrAuthUnavailable (retryable), not ErrUnauthenticated.
func TestValidate_NoResolverIsUnavailable(t *testing.T) {
	header := seg(t, map[string]string{"alg": "RS256", "typ": "JWT"})
	claims := seg(t, map[string]any{"user_id": "u1"})
	token := header + "." + claims + "." + base64.RawURLEncoding.EncodeToString([]byte("sig"))

	c := NewClient(Config{}) // nil resolver → cannot obtain key
	_, err := c.Validate(context.Background(), token)
	require.ErrorIs(t, err, ErrAuthUnavailable)
}

func TestRequireOwner_FailsClosedWithoutIdentity(t *testing.T) {
	_, err := RequireOwner(context.Background())
	require.ErrorIs(t, err, ErrUnauthenticated)

	ctx := WithIdentity(context.Background(), Identity{OwnerID: "owner-1"})
	id, err := RequireOwner(ctx)
	require.NoError(t, err)
	require.Equal(t, "owner-1", id.OwnerID)
}

func TestBearerToken(t *testing.T) {
	require.Equal(t, "abc", BearerToken("Bearer abc"))
	require.Equal(t, "abc", BearerToken("bearer abc"))
	require.Equal(t, "", BearerToken("Basic abc"))
	require.Equal(t, "", BearerToken("abc"))
	require.Equal(t, "", BearerToken(""))
}

func TestAudienceUnmarshal_StringOrArray(t *testing.T) {
	var single audience
	require.NoError(t, json.Unmarshal([]byte(`"a"`), &single))
	require.True(t, single.contains("a"))

	var many audience
	require.NoError(t, json.Unmarshal([]byte(`["a","b"]`), &many))
	require.True(t, many.contains("b"))
}

func TestIdentityCarriesVerifiedScopes(t *testing.T) {
	id, err := (ownerClaims{
		UserID: "owner-1", Iss: AuthScenarioSlug, Aud: audience{AuthExpectedAudience},
		Scopes: []string{"vrooli-bridge:write"}, Exp: time.Now().Add(time.Hour).Unix(),
	}).toIdentity(time.Now())
	require.NoError(t, err)
	require.Equal(t, []string{"vrooli-bridge:write"}, id.Scopes)
}

func TestBreakGlassValidatorIsOfflineAndDistinct(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	now := time.Unix(1000, 0).UTC()
	token, err := trustposture.Issue(private, trustposture.BreakGlassClaims{
		Subject: "owner-1", Audience: AuthExpectedAudience,
		Target: "host-a",
		Scopes: []string{"vrooli-bridge:read"}, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Minute).Unix(),
	})
	require.NoError(t, err)
	c := NewClient(Config{Now: func() time.Time { return now }, BreakGlassPublicKey: public, BreakGlassTarget: "host-a"})
	id, err := c.ValidateBreakGlass(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, "owner-1", id.OwnerID)
	require.Equal(t, AuthMethodBreakGlass, id.AuthMethod)
	_, err = c.ValidateBreakGlass(context.Background(), token+"x")
	require.ErrorIs(t, err, ErrUnauthenticated)
}

type localStore struct{ record localenrollment.Record }

func (s localStore) Lookup(context.Context, string) (localenrollment.Record, error) { return s.record, nil }

func TestValidateLocalDoesNotNeedAuthenticator(t *testing.T) {
	private, err := sharedsession.GenerateKey()
	require.NoError(t, err)
	public, err := sharedsession.PublicKey(private)
	require.NoError(t, err)
	now := time.Unix(2_000, 0).UTC()
	token, err := sharedsession.Mint(private, "enrollment-1", "owner-1", []string{"vrooli-bridge:read"}, now, time.Minute)
	require.NoError(t, err)
	c := NewClient(Config{Now: func() time.Time { return now }, LocalSessions: localStore{record: localenrollment.Record{Reference: "enrollment-1", OperatorID: "owner-1", Mode: sharedsession.ModePersonal, PublicKey: public, Scopes: []string{"vrooli-bridge:read"}, EnrolledAt: now}}})
	id, err := c.ValidateLocal(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, "owner-1", id.OwnerID)
	require.Equal(t, AuthMethodEnrolled, id.AuthMethod)
}
