package jwks

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/realm"
)

// TestJWKSServesKeyAndHubStyleVerifierValidates reproduces device-sync-hub's
// local verification path: fetch the JWKS, rebuild the RSA key from (n,e), and
// verify a freshly-minted RS256 token's signature. This is the cross-scenario
// contract — if it breaks, the hub can no longer verify owner tokens.
func TestJWKSServesKeyAndHubStyleVerifierValidates(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	keys := authcrypto.NewKeysFromPair(priv, &priv.PublicKey)
	signer := authcrypto.NewSigner(keys, authcrypto.SignerConfig{Issuer: realm.Issuer})
	tok, err := signer.Sign(authcrypto.TokenInput{UserID: "u1", Email: "a@b.co", Audience: realm.DefaultAudience})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	srv := httptest.NewServer(NewHandler(keys))
	defer srv.Close()
	resp, err := http.Get(srv.URL + Path)
	if err != nil {
		t.Fatalf("get jwks: %v", err)
	}
	defer resp.Body.Close()
	if cc := resp.Header.Get("Cache-Control"); cc != "public, max-age=300" {
		t.Fatalf("cache-control = %q", cc)
	}

	var set struct {
		Keys []struct {
			Kty, Alg, Kid, N, E string
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		t.Fatalf("decode jwks: %v", err)
	}
	if len(set.Keys) != 1 || set.Keys[0].Kty != "RSA" || set.Keys[0].Alg != "RS256" {
		t.Fatalf("unexpected jwks: %+v", set)
	}
	if set.Keys[0].Kid != keys.KID() {
		t.Fatalf("kid mismatch: %q vs %q", set.Keys[0].Kid, keys.KID())
	}

	pub := rebuildRSA(t, set.Keys[0].N, set.Keys[0].E)
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("token not a JWS: %q", tok)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode sig: %v", err)
	}
	hashed := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed[:], sig); err != nil {
		t.Fatalf("hub-style verify failed: %v", err)
	}
}

func TestJWKSNilKeyUnavailable(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Path, nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rec.Code)
	}
}

func rebuildRSA(t *testing.T, nB64, eB64 string) *rsa.PublicKey {
	t.Helper()
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		t.Fatalf("decode n: %v", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		t.Fatalf("decode e: %v", err)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: int(new(big.Int).SetBytes(eBytes).Int64())}
}
