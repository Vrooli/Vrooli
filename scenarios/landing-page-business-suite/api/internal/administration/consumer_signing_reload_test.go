package administration

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/consumeridentity"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func TestConsumerSigningIssuerUsesDefaultWhenUnset(t *testing.T) {
	if got := ConsumerSigningIssuer("  "); got != "landing-page-business-suite" {
		t.Fatalf("consumer signing issuer = %q, want the runtime default", got)
	}
	if got := ConsumerSigningIssuer(" issuer.example "); got != "issuer.example" {
		t.Fatalf("consumer signing issuer = %q, want trimmed configured issuer", got)
	}
}

func TestConsumerSigningAuthorityReloadUpdatesPeerReplicaAndExternalVerifier(t *testing.T) {
	store := &rotationAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	initialSigner, err := consumeridentity.GenerateSigner("initial", "lpbs.test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	initialPEM, err := consumeridentity.MarshalPrivateKeyPEM(initialSigner.Private)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := authority.PutCandidate(credentialAuthorityIdentity, consumerSigningAuthorityField, string(initialPEM))
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(initial); err != nil {
		t.Fatal(err)
	}
	source, err := NewAuthorityConsumerSigningSource(authority, credentialAuthorityIdentity, consumerSigningAuthorityField, ConsumerSigningAuthorityScope, "initial", "", "lpbs.test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	makeReplica := func() *UserAuthService {
		return NewUserAuthService(UserAuthServiceOptions{
			JWTIssuer: "lpbs.test", ConsumerSigningKeyPEM: string(initialPEM), ConsumerSigningKeyID: "initial",
			ConsumerSigningVersion: initial.Version, ConsumerSigningSource: source, AccessTTL: time.Minute,
		})
	}
	rotatingReplica := makeReplica()
	peerReplica := makeReplica()
	oldToken, _, err := rotatingReplica.GenerateAccessToken("old-user", "old@example.test", "old-session")
	if err != nil {
		t.Fatal(err)
	}

	rotatedAt := time.Now().UTC()
	rotation, err := RotateConsumerSigningCredential(authority, ConsumerSigningRotationOptions{
		Identity: credentialAuthorityIdentity, Field: consumerSigningAuthorityField, Scope: ConsumerSigningAuthorityScope,
		ExpectedVersion: initial.Version, LegacyKeyID: "initial", Issuer: "lpbs.test", AccessTTL: time.Minute,
		MaxTokenTTL: 7 * 24 * time.Hour, Now: rotatedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	rotation.ApplyTo(rotatingReplica)

	reload, err := peerReplica.RefreshConsumerSigningKeyset()
	if err != nil {
		t.Fatalf("peer replica failed to reload the activated authority version: %v", err)
	}
	if !reload.Changed || reload.Version != rotation.Receipt.Version || reload.ActiveKeyID != rotation.Receipt.ActiveKeyID || reload.OverlapKeyCount != 1 {
		t.Fatalf("unexpected metadata reload receipt: %+v", reload)
	}
	newToken, _, err := peerReplica.GenerateAccessToken("new-user", "new@example.test", "new-session")
	if err != nil {
		t.Fatalf("peer replica failed to sign with reloaded key: %v", err)
	}
	if _, err := peerReplica.ValidateAccessToken(oldToken); err != nil {
		t.Fatalf("peer replica did not retain old-key overlap: %v", err)
	}

	// Exercise the public JWKS with an independent verifier construction, as a
	// relying party does, rather than using the LPBS service's internal keyset.
	publicJWKS, err := peerReplica.PublicKeySet()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(publicJWKS), "PRIVATE KEY") || strings.Contains(string(publicJWKS), string(initialPEM)) {
		t.Fatal("published JWKS contained private signing material")
	}
	keyset, err := consumeridentity.ParseJWKS(publicJWKS)
	if err != nil {
		t.Fatalf("external verifier could not parse JWKS: %v", err)
	}
	externalVerifier := consumeridentity.NewVerifier(keyset, "lpbs.test", 30*time.Second)
	for label, token := range map[string]string{"overlap": oldToken, "active": newToken} {
		if _, err := externalVerifier.Verify(token); err != nil {
			t.Fatalf("independent verifier rejected %s token using published JWKS: %v", label, err)
		}
	}
	encodedReceipt, err := json.Marshal(reload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encodedReceipt), string(initialPEM)) || strings.Contains(string(encodedReceipt), "PRIVATE KEY") {
		t.Fatal("reload receipt exposed private signing material")
	}
}

func TestConsumerSigningAuthorityReloadFailsClosedOnProviderOutage(t *testing.T) {
	authority, err := credentialauthority.Unavailable("synthetic outage")
	if err != nil {
		t.Fatal(err)
	}
	source, err := NewAuthorityConsumerSigningSource(authority, credentialAuthorityIdentity, consumerSigningAuthorityField, ConsumerSigningAuthorityScope, "initial", "", "lpbs.test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM, err := consumeridentity.MarshalPrivateKeyPEM(key)
	if err != nil {
		t.Fatal(err)
	}
	service := NewUserAuthService(UserAuthServiceOptions{
		JWTIssuer: "lpbs.test", ConsumerSigningKeyPEM: string(privatePEM), ConsumerSigningKeyID: "cached",
		ConsumerSigningVersion: "cached-version", ConsumerSigningSource: source, AccessTTL: time.Minute,
	})
	if _, _, err := service.GenerateAccessToken("user", "user@example.test", "session"); err == nil || strings.Contains(err.Error(), string(privatePEM)) {
		t.Fatalf("signing with unavailable authority = %v, want sanitized fail-closed error", err)
	}
	if _, err := service.PublicKeySet(); err == nil || strings.Contains(err.Error(), string(privatePEM)) {
		t.Fatalf("JWKS reload with unavailable authority = %v, want sanitized failure", err)
	}
	service.consumerKeyMu.RLock()
	version, keyID := service.consumerSigningVersion, service.consumerSigner.KeyID
	service.consumerKeyMu.RUnlock()
	if version != "cached-version" || keyID != "cached" {
		t.Fatalf("outage replaced last active in-memory key: version=%q keyID=%q", version, keyID)
	}
}
