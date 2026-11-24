package administration

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/vrooli/api-core/consumeridentity"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type rotationAuthorityStore struct {
	mu                   sync.Mutex
	values               map[string]string
	failCandidateDeletes bool
}

func (s *rotationAuthorityStore) Put(service, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[service+"/"+key] = value
	return nil
}
func (s *rotationAuthorityStore) Get(service, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", credentialauthority.ErrNotFound, service, key)
	}
	return value, nil
}
func (s *rotationAuthorityStore) Delete(service, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failCandidateDeletes && strings.HasPrefix(key, "candidate/") {
		return errors.New("synthetic candidate cleanup outage")
	}
	delete(s.values, service+"/"+key)
	return nil
}

func (s *rotationAuthorityStore) putLegacyActive(identity credentialauthority.Identity, field, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values["vrooli.credentials.v1/"+string(identity)+":"+field] = value
}

func TestOperatorCredentialWriteUsesAuthorityCandidateActivation(t *testing.T) {
	store := &rotationAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	SetCredentialAuthorityProvider(func() (*credentialauthority.Authority, error) { return authority, nil })
	t.Cleanup(func() { SetCredentialAuthorityProvider(nil) })
	if err := PutAuthorityCredential("SENDGRID_API_KEY", "synthetic-first-version"); err != nil {
		t.Fatal(err)
	}
	first, err := ResolveAuthorityCredential("SENDGRID_API_KEY")
	if err != nil || first != "synthetic-first-version" {
		t.Fatalf("first active credential = %q, %v", first, err)
	}
	if err := PutAuthorityCredential("SENDGRID_API_KEY", "synthetic-rotated-version"); err != nil {
		t.Fatal(err)
	}
	second, err := ResolveAuthorityCredential("SENDGRID_API_KEY")
	if err != nil || second != "synthetic-rotated-version" {
		t.Fatalf("rotated credential = %q, %v", second, err)
	}
}

func TestVersionBoundCredentialRotationReturnsMetadataOnlyReceipt(t *testing.T) {
	store := &rotationAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	SetCredentialAuthorityProvider(func() (*credentialauthority.Authority, error) { return authority, nil })
	t.Cleanup(func() { SetCredentialAuthorityProvider(nil) })
	first, err := PutAuthorityCredentialIfVersion("MAILGUN_API_KEY", "sentinel-provider-value-v1", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Version == "" || first.Scope != "mail:transactional:send" || first.Identity != credentialAuthorityIdentity || first.Field != "mailgun-api-key" {
		t.Fatalf("initial metadata receipt = identity:%q field:%q version-present:%t scope:%q", first.Identity, first.Field, first.Version != "", first.Scope)
	}
	rotated, err := PutAuthorityCredentialIfVersion("MAILGUN_API_KEY", "sentinel-provider-value-v2", first.Version)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.Version == first.Version || rotated.CleanupPending {
		t.Fatalf("rotation receipt version changed:%t cleanup_pending:%t", rotated.Version != first.Version, rotated.CleanupPending)
	}
	if _, err := PutAuthorityCredentialIfVersion("MAILGUN_API_KEY", "sentinel-stale-write", first.Version); !errors.Is(err, credentialauthority.ErrVersionConflict) {
		t.Fatalf("stale version write error = %v, want version conflict", err)
	}
	binding, err := ResolveCredentialBinding("MAILGUN_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if binding.Version != rotated.Version || binding.Value != "sentinel-provider-value-v2" {
		t.Fatalf("active binding version matches:%t value is latest:%t", binding.Version == rotated.Version, binding.Value == "sentinel-provider-value-v2")
	}
	encoded, err := json.Marshal(rotated)
	if err != nil {
		t.Fatal(err)
	}
	for _, sentinel := range []string{"sentinel-provider-value-v1", "sentinel-provider-value-v2", "sentinel-stale-write"} {
		if strings.Contains(string(encoded), sentinel) {
			t.Fatalf("rotation receipt leaked secret sentinel %q", sentinel)
		}
	}
}

func TestCredentialBindingsAcrossCustodyCategoriesCarryVersionAndScope(t *testing.T) {
	store := &rotationAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	SetCredentialAuthorityProvider(func() (*credentialauthority.Authority, error) { return authority, nil })
	t.Cleanup(func() { SetCredentialAuthorityProvider(nil) })
	cases := []struct{ key, scope string }{
		{"STRIPE_TEST_SECRET_KEY", "stripe:test:payments"},
		{"STRIPE_LIVE_SECRET_KEY", "stripe:live:payments"},
		{"DELIVERY_S3_SECRET_ACCESS_KEY", "s3:delivery:object-read-write"},
		{"SENDGRID_API_KEY", "mail:transactional:send"},
		{"SESSION_SECRET", "signing:session-cookie"},
		{"CONSUMER_AUTH_PRIVATE_KEY", "signing:consumer-token"},
		{"LPBS_SERVICE_SECRET", "service:lpbs-bootstrap"},
	}
	for i, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			synthetic := fmt.Sprintf("fixture-value-%d-never-log-or-export", i)
			if err := PutAuthorityCredential(tc.key, synthetic); err != nil {
				t.Fatal(err)
			}
			binding, err := ResolveCredentialBinding(tc.key)
			if err != nil {
				t.Fatal(err)
			}
			if binding.Identity != credentialAuthorityIdentity || binding.Field != AuthorityFieldForKey(tc.key) || binding.Version == "" || binding.Scope != tc.scope || binding.Value != synthetic {
				t.Fatalf("binding metadata = identity:%q field:%q version-present:%t scope:%q value-match:%t", binding.Identity, binding.Field, binding.Version != "", binding.Scope, binding.Value == synthetic)
			}
			before := binding.Version
			if err := PutAuthorityCredential(tc.key, synthetic+"-rotated"); err != nil {
				t.Fatal(err)
			}
			after, err := ResolveCredentialBinding(tc.key)
			if err != nil {
				t.Fatal(err)
			}
			if after.Version == before || after.Value != synthetic+"-rotated" || after.Scope != tc.scope {
				t.Fatalf("rotation did not activate scoped candidate (version changed %t, value changed %t)", after.Version != before, after.Value != synthetic+"-rotated")
			}
		})
	}
}

func TestCredentialBindingOutageRefusesAndRecoversWithoutLoggingValue(t *testing.T) {
	store := &rotationAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	SetCredentialAuthorityProvider(func() (*credentialauthority.Authority, error) { return authority, nil })
	if err := PutAuthorityCredential("MAILGUN_API_KEY", "synthetic-mail-secret"); err != nil {
		t.Fatal(err)
	}
	unavailable, err := credentialauthority.Unavailable("synthetic fixture outage")
	if err != nil {
		t.Fatal(err)
	}
	SetCredentialAuthorityProvider(func() (*credentialauthority.Authority, error) { return unavailable, nil })
	if binding, err := ResolveCredentialBinding("MAILGUN_API_KEY"); err == nil || binding.Value != "" || strings.Contains(err.Error(), "synthetic-mail-secret") {
		t.Fatalf("outage resolution = value-present:%t error:%v; want refusal without material", binding.Value != "", err)
	}
	SetCredentialAuthorityProvider(func() (*credentialauthority.Authority, error) { return authority, nil })
	var logs []string
	if got := ResolveSecret("MAILGUN_API_KEY", func(_ string, fields map[string]interface{}) { logs = append(logs, fmt.Sprint(fields)) }); got != "synthetic-mail-secret" {
		t.Fatal("authority recovery did not resolve the active fixture")
	}
	for _, line := range logs {
		if strings.Contains(line, "synthetic-mail-secret") {
			t.Fatal("secret value appeared in resolver log")
		}
	}
}

func TestIssuedServicePrincipalBindingIsExactAndVersioned(t *testing.T) {
	store := &rotationAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := credentialauthority.ParseIdentity("scenario-authenticator/service-principal/landing-page-business-suite-money-ledger-writer/money-ledger")
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := authority.PutCandidate(identity, "credential", "synthetic-issued-principal")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(candidate); err != nil {
		t.Fatal(err)
	}
	binding, err := ResolveBoundCredentialFromAuthority(authority, identity, "credential", "money-ledger:write")
	if err != nil {
		t.Fatal(err)
	}
	if binding.Identity != identity || binding.Field != "credential" || binding.Scope != "money-ledger:write" || binding.Version != candidate.Version || binding.Value != "synthetic-issued-principal" {
		t.Fatalf("issued-principal binding mismatch: identity=%t field=%q scope=%q version-match=%t value-match=%t", binding.Identity == identity, binding.Field, binding.Scope, binding.Version == candidate.Version, binding.Value == "synthetic-issued-principal")
	}
}

func TestConsumerSigningRotationActivatesVersionedKeyWithJWKSOverlap(t *testing.T) {
	store := &rotationAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	oldSigner, err := consumeridentity.GenerateSigner("initial", "lpbs.test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	oldPEM, err := consumeridentity.MarshalPrivateKeyPEM(oldSigner.Private)
	if err != nil {
		t.Fatal(err)
	}
	first, err := authority.PutCandidate(credentialAuthorityIdentity, "consumer-auth-private-key", string(oldPEM))
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(first); err != nil {
		t.Fatal(err)
	}
	oldToken, _, err := oldSigner.Sign(consumeridentity.Claims{Subject: "user-1", UserID: "user-1", Email: "u@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	rotatedAt := time.Now().UTC()
	maxLeaseTTL := 7 * 24 * time.Hour
	result, err := RotateConsumerSigningCredential(authority, ConsumerSigningRotationOptions{Identity: credentialAuthorityIdentity, Field: "consumer-auth-private-key", Scope: "signing:consumer-token", ExpectedVersion: first.Version, LegacyKeyID: "initial", Issuer: "lpbs.test", AccessTTL: time.Minute, MaxTokenTTL: maxLeaseTTL, Now: rotatedAt})
	if err != nil {
		t.Fatal(err)
	}
	receipt := result.Receipt
	if receipt.PreviousKeyID != "initial" || receipt.ActiveKeyID == "" || receipt.ActiveKeyID == receipt.PreviousKeyID || receipt.Version == first.Version || receipt.OverlapKeyCount != 1 {
		t.Fatalf("unexpected rotation receipt: %+v", receipt)
	}
	if want := rotatedAt.Add(maxLeaseTTL + ConsumerSigningDefaultClockSkew + ConsumerSigningJWKSCacheMaxAge); receipt.PreviousKeysExpireAt.Before(want.Add(-time.Second)) || receipt.PreviousKeysExpireAt.After(want.Add(time.Second)) {
		t.Fatalf("overlap expiry %s not aligned with access TTL + skew + JWKS cache age (%s)", receipt.PreviousKeysExpireAt, want)
	}
	active, err := ResolveBoundCredentialFromAuthority(authority, credentialAuthorityIdentity, "consumer-auth-private-key", "signing:consumer-token")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := ParseConsumerSigningCredential(active.Value, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.KeyID != receipt.ActiveKeyID || strings.Contains(fmt.Sprintf("%+v", receipt), bundle.PrivateKeyPEM) {
		t.Fatal("receipt contains private material or active key ID does not match")
	}
	service := NewUserAuthService(UserAuthServiceOptions{JWTIssuer: "lpbs.test", ConsumerSigningKeyPEM: string(oldPEM), ConsumerSigningKeyID: "initial"})
	result.ApplyTo(service)
	if _, err := service.ValidateAccessToken(oldToken); err != nil {
		t.Fatalf("old key overlap rejected token: %v", err)
	}
	newToken, _, err := service.GenerateAccessToken("user-2", "new@example.test", "session-2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateAccessToken(newToken); err != nil {
		t.Fatalf("new active key rejected token: %v", err)
	}
	if expiredSet := service.consumerKeySetAt(receipt.PreviousKeysExpireAt); len(expiredSet.Keys) != 1 {
		t.Fatalf("expired overlap keys remained in verifier/JWKS set: count=%d", len(expiredSet.Keys))
	}
	if strings.Contains(active.Value, string(oldPEM)) {
		t.Fatal("retired private key was retained instead of its public overlap key")
	}
}

func TestConsumerSigningRotationFailsClosedOnAuthorityOutage(t *testing.T) {
	store := &rotationAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := consumeridentity.GenerateSigner("initial", "lpbs.test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM, err := consumeridentity.MarshalPrivateKeyPEM(signer.Private)
	if err != nil {
		t.Fatal(err)
	}
	first, err := authority.PutCandidate(credentialAuthorityIdentity, "consumer-auth-private-key", string(privatePEM))
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(first); err != nil {
		t.Fatal(err)
	}
	unavailable, err := credentialauthority.Unavailable("synthetic outage")
	if err != nil {
		t.Fatal(err)
	}
	_, err = RotateConsumerSigningCredential(unavailable, ConsumerSigningRotationOptions{Identity: credentialAuthorityIdentity, Field: "consumer-auth-private-key", Scope: "signing:consumer-token", ExpectedVersion: first.Version, LegacyKeyID: "initial", Issuer: "lpbs.test", AccessTTL: time.Minute, Now: time.Now().UTC()})
	if err == nil {
		t.Fatal("rotation succeeded while authority unavailable")
	}
	active, err := ResolveBoundCredentialFromAuthority(authority, credentialAuthorityIdentity, "consumer-auth-private-key", "signing:consumer-token")
	if err != nil || active.Version != first.Version {
		t.Fatalf("active credential changed during outage: version-match=%t err=%v", active.Version == first.Version, err)
	}
}

func TestConsumerSigningRotationMigratesLegacyUnversionedSignerAndRejectsStaleVersion(t *testing.T) {
	store := &rotationAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	legacySigner, err := consumeridentity.GenerateSigner("legacy", "lpbs.test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	legacyPEM, err := consumeridentity.MarshalPrivateKeyPEM(legacySigner.Private)
	if err != nil {
		t.Fatal(err)
	}
	store.putLegacyActive(credentialAuthorityIdentity, "consumer-auth-private-key", string(legacyPEM))
	started := time.Now().UTC()
	result, err := RotateConsumerSigningCredential(authority, ConsumerSigningRotationOptions{
		Identity: credentialAuthorityIdentity, Field: "consumer-auth-private-key", Scope: "signing:consumer-token",
		ExpectedVersion: "", LegacyKeyID: "legacy", Issuer: "lpbs.test", AccessTTL: time.Minute,
		MaxTokenTTL: 7 * 24 * time.Hour, Now: started,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt.PreviousKeyID != "legacy" || result.Receipt.Version == "" {
		t.Fatalf("legacy rotation receipt omitted metadata: %+v", result.Receipt)
	}
	active := authority.Status(credentialAuthorityIdentity, "consumer-auth-private-key")
	if !active.Configured || active.Version != result.Receipt.Version {
		t.Fatalf("versioned legacy replacement not active: configured=%t version-match=%t", active.Configured, active.Version == result.Receipt.Version)
	}
	_, err = RotateConsumerSigningCredential(authority, ConsumerSigningRotationOptions{
		Identity: credentialAuthorityIdentity, Field: "consumer-auth-private-key", Scope: "signing:consumer-token",
		ExpectedVersion: "stale-version", LegacyKeyID: "legacy", Issuer: "lpbs.test", AccessTTL: time.Minute,
	})
	if !errors.Is(err, credentialauthority.ErrVersionConflict) {
		t.Fatalf("stale expected version error = %v, want conflict", err)
	}
	if after := authority.Status(credentialAuthorityIdentity, "consumer-auth-private-key"); after.Version != result.Receipt.Version {
		t.Fatal("stale rotation attempt changed the active version")
	}
}

func TestConsumerSigningRotationTreatsCandidateCleanupAsCommitted(t *testing.T) {
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
	service := NewUserAuthService(UserAuthServiceOptions{JWTIssuer: "lpbs.test", ConsumerSigningKeyPEM: string(initialPEM), ConsumerSigningKeyID: "initial", ConsumerSigningVersion: initial.Version})
	store.failCandidateDeletes = true
	rotation, err := RotateConsumerSigningCredential(authority, ConsumerSigningRotationOptions{
		Identity: credentialAuthorityIdentity, Field: consumerSigningAuthorityField, Scope: ConsumerSigningAuthorityScope,
		ExpectedVersion: initial.Version, LegacyKeyID: "initial", Issuer: "lpbs.test", AccessTTL: time.Minute,
		MaxTokenTTL: 7 * 24 * time.Hour, Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("rotation reported failure after committed activation: %v", err)
	}
	if !rotation.Receipt.CleanupPending || rotation.Receipt.Version == "" {
		t.Fatalf("rotation receipt omitted committed cleanup state: %+v", rotation.Receipt)
	}
	if status := authority.Status(credentialAuthorityIdentity, consumerSigningAuthorityField); !status.Configured || status.Version != rotation.Receipt.Version {
		t.Fatal("receipt version does not match committed authority state")
	}
	rotation.ApplyTo(service)
	newToken, _, err := service.GenerateAccessToken("rotated-user", "u@example.test", "session")
	if err != nil {
		t.Fatalf("committed signing key was not applied locally: %v", err)
	}
	if _, err := service.ValidateAccessToken(newToken); err != nil {
		t.Fatalf("locally activated key rejected new token: %v", err)
	}
}

func TestConsumerSigningRotationCASAllowsOnlyOneConcurrentExpectedVersion(t *testing.T) {
	store := &rotationAuthorityStore{}
	firstAuthority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	secondAuthority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := consumeridentity.GenerateSigner("initial", "lpbs.test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM, err := consumeridentity.MarshalPrivateKeyPEM(signer.Private)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := firstAuthority.PutCandidate(credentialAuthorityIdentity, "consumer-auth-private-key", string(privatePEM))
	if err != nil {
		t.Fatal(err)
	}
	if err := firstAuthority.ActivateCandidate(initial); err != nil {
		t.Fatal(err)
	}

	type outcome struct {
		result ConsumerSigningRotationResult
		err    error
	}
	start := make(chan struct{})
	outcomes := make(chan outcome, 2)
	for _, authority := range []*credentialauthority.Authority{firstAuthority, secondAuthority} {
		authority := authority
		go func() {
			<-start
			result, rotateErr := RotateConsumerSigningCredential(authority, ConsumerSigningRotationOptions{
				Identity: credentialAuthorityIdentity, Field: "consumer-auth-private-key", Scope: "signing:consumer-token",
				ExpectedVersion: initial.Version, LegacyKeyID: "initial", Issuer: "lpbs.test", AccessTTL: time.Minute,
				MaxTokenTTL: 7 * 24 * time.Hour, Now: time.Now().UTC(),
			})
			outcomes <- outcome{result: result, err: rotateErr}
		}()
	}
	close(start)
	first, second := <-outcomes, <-outcomes
	wins, conflicts := 0, 0
	for _, attempt := range []outcome{first, second} {
		switch {
		case attempt.err == nil:
			wins++
			if attempt.result.Receipt.Version == "" || attempt.result.Receipt.Version == initial.Version {
				t.Fatalf("successful concurrent rotation has invalid version receipt: %+v", attempt.result.Receipt)
			}
			if strings.Contains(fmt.Sprintf("%+v", attempt.result.Receipt), string(privatePEM)) {
				t.Fatal("rotation receipt exposed signing private material")
			}
		case errors.Is(attempt.err, credentialauthority.ErrVersionConflict):
			conflicts++
		default:
			t.Fatalf("concurrent rotation error = %v, want success or version conflict", attempt.err)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("concurrent expected-version rotations: wins=%d conflicts=%d, want one each", wins, conflicts)
	}
	active := firstAuthority.Status(credentialAuthorityIdentity, "consumer-auth-private-key")
	if !active.Configured || active.Version == initial.Version {
		t.Fatalf("successful rotation was not the sole active version: configured=%t changed=%t", active.Configured, active.Version != initial.Version)
	}
}

func TestCustodyManifestDoesNotInjectLPBSAuthoredSecretsIntoEnvironment(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "..", ".vrooli", "service.json")
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Credentials struct {
			Descriptors []struct {
				LogicalID string `json:"logical_id"`
				Field     string `json:"field"`
				Env       string `json:"env"`
			} `json:"descriptors"`
			Consumers []struct {
				LogicalID string `json:"logical_id"`
				Field     string `json:"field"`
				Reason    string `json:"reason"`
			} `json:"consumers"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	fields := map[string]bool{
		"stripe-test-secret-key": true, "stripe-live-secret-key": true,
		"delivery-s3-secret-access-key": true, "smtp-password": true,
		"sendgrid-api-key": true, "mailgun-api-key": true,
		"session-secret": true, "consumer-auth-private-key": true,
		"service-secret": true, "admin-default-password": true, "admin-mfa-encryption-key": true,
	}
	descriptors := make(map[string]bool)
	for _, descriptor := range manifest.Credentials.Descriptors {
		if !fields[descriptor.Field] {
			continue
		}
		if descriptor.LogicalID != string(credentialAuthorityIdentity) || descriptor.Env != "" {
			t.Errorf("%s descriptor identity=%q env-present=%t; LPBS-authored secrets must resolve by identity, not env", descriptor.Field, descriptor.LogicalID, descriptor.Env != "")
		}
		descriptors[descriptor.Field] = true
	}
	for field := range fields {
		if !descriptors[field] {
			t.Errorf("manifest has no descriptor for %s", field)
		}
	}
	consumers := make(map[string]bool)
	for _, consumer := range manifest.Credentials.Consumers {
		if !fields[consumer.Field] {
			continue
		}
		if consumer.LogicalID != string(credentialAuthorityIdentity) || !strings.Contains(consumer.Reason, "scope=") || !strings.Contains(consumer.Reason, "active-version metadata") || !strings.Contains(consumer.Reason, "candidate activation") {
			t.Errorf("%s consumer lacks identity, scope or rotatability contract", consumer.Field)
		}
		consumers[consumer.Field] = true
	}
	for field := range fields {
		if !consumers[field] {
			t.Errorf("manifest has no custody consumer for %s", field)
		}
	}
}

func TestRetiredStripeCredentialNamesFailClosed(t *testing.T) {
	t.Setenv("LPBS_TEST_CREDENTIAL_FALLBACK", "1")
	for _, key := range []string{"STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET", "STRIPE_PUBLISHABLE_KEY"} {
		t.Setenv(key, "fixture-secret-must-not-resolve")
		if value, err := ResolveAuthorityCredential(key); err == nil || value != "" {
			t.Fatalf("ResolveAuthorityCredential(%s) = %q, %v; want a retired-field error and no value", key, value, err)
		}
		if err := PutAuthorityCredential(key, "fixture-secret-must-not-write"); err == nil {
			t.Fatalf("PutAuthorityCredential(%s) unexpectedly accepted a retired Stripe field", key)
		}
	}
}

type witnessAuthorityStore struct {
	mu    sync.Mutex
	value string
}

func (s *witnessAuthorityStore) Put(_, _ string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = value
	return nil
}

func (s *witnessAuthorityStore) Get(_, _ string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.value == "" {
		return "", credentialauthority.ErrNotFound
	}
	return s.value, nil
}

func (s *witnessAuthorityStore) Delete(_, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = ""
	return nil
}

func newCredentialWitnessTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE credential_mint_witness (
		logical_id TEXT NOT NULL,
		field TEXT NOT NULL,
		minted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (logical_id, field)
	)`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestCredentialWitnessRecordsAndReadsOnlyMintMetadata(t *testing.T) {
	db := newCredentialWitnessTestDB(t)
	witness := NewCredentialWitness(db)
	identity, err := credentialauthority.ParseIdentity("vrooli/landing-page-business-suite")
	if err != nil {
		t.Fatal(err)
	}

	minted, err := witness.Minted(identity, "service-secret")
	if err != nil || minted {
		t.Fatalf("initial Minted() = %v, %v; want false, nil", minted, err)
	}
	if err := witness.RecordMint(identity, "service-secret"); err != nil {
		t.Fatal(err)
	}
	minted, err = witness.Minted(identity, "service-secret")
	if err != nil || !minted {
		t.Fatalf("recorded Minted() = %v, %v; want true, nil", minted, err)
	}
	if err := witness.RecordMint(identity, "service-secret"); err != nil {
		t.Fatalf("duplicate RecordMint() must be idempotent: %v", err)
	}

	var columns string
	if err := db.QueryRow(`SELECT group_concat(name, ',') FROM pragma_table_info('credential_mint_witness')`).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if columns != "logical_id,field,minted_at" {
		t.Fatalf("witness table columns = %q; credential values must not be stored", columns)
	}
}

func TestCredentialWitnessPropagatesDatabaseErrors(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	witness := NewCredentialWitness(db)
	identity, err := credentialauthority.ParseIdentity("vrooli/landing-page-business-suite")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := witness.Minted(identity, "field"); err == nil {
		t.Fatal("Minted() must propagate a witness-store error")
	}
	if err := witness.RecordMint(identity, "field"); err == nil {
		t.Fatal("RecordMint() must propagate a witness-store error")
	}
}

func TestCredentialWitnessRefusesRemintAfterStoreLoss(t *testing.T) {
	db := newCredentialWitnessTestDB(t)
	witness := NewCredentialWitness(db)
	identity, err := credentialauthority.ParseIdentity("vrooli/landing-page-business-suite")
	if err != nil {
		t.Fatal(err)
	}
	store := &witnessAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	mints := 0
	if _, err := authority.ResolveOrMint(identity, "service-secret", witness, func() (string, error) {
		mints++
		return "minted-once", nil
	}); err != nil {
		t.Fatalf("first resolve-or-mint: %v", err)
	}
	if mints != 1 {
		t.Fatalf("mint count after virgin start = %d, want 1", mints)
	}
	if err := store.Delete(string(identity), "service-secret"); err != nil {
		t.Fatal(err)
	}
	_, err = authority.ResolveOrMint(identity, "service-secret", witness, func() (string, error) {
		mints++
		return "replacement", nil
	})
	var refused *credentialauthority.MintRefusedError
	if !errors.As(err, &refused) {
		t.Fatalf("second resolve-or-mint error = %v, want MintRefusedError", err)
	}
	if mints != 1 {
		t.Fatalf("mint count after store loss = %d, want 1", mints)
	}
	if !strings.Contains(err.Error(), "vrooli credentials") || !strings.Contains(err.Error(), "--accept-credential-loss") {
		t.Fatalf("refusal error lacks recovery guidance: %v", err)
	}
}

func TestResolveSecretRetiredStripeFallbackLogsNoValue(t *testing.T) {
	t.Setenv("LPBS_TEST_CREDENTIAL_FALLBACK", "1")
	t.Setenv("STRIPE_SECRET_KEY", "fixture-retired-stripe-secret")
	var event string
	var fields map[string]interface{}
	value := ResolveSecret("STRIPE_SECRET_KEY", func(name string, details map[string]interface{}) {
		event, fields = name, details
	})
	if value != "" {
		t.Fatal("retired Stripe env fallback unexpectedly returned a value")
	}
	if event != "credential_authority_unavailable" {
		t.Fatalf("event = %q; want authority refusal", event)
	}
	for _, field := range fields {
		if strings.Contains(fmt.Sprint(field), "fixture-retired-stripe-secret") {
			t.Fatal("retired Stripe value appeared in log fields")
		}
	}
}

func TestResolveSecretDoesNotUseLegacyEnvironmentWhenAuthorityIsUnavailable(t *testing.T) {
	t.Setenv("LPBS_TEST_CREDENTIAL_FALLBACK", "0")
	SetCredentialAuthorityProvider(func() (*credentialauthority.Authority, error) { return nil, errors.New("isolated authority outage") })
	t.Cleanup(func() { SetCredentialAuthorityProvider(nil) })
	for _, key := range []string{"SENDGRID_API_KEY", "MAILGUN_API_KEY", "SMTP_PASSWORD", "STRIPE_TEST_SECRET_KEY", "DELIVERY_S3_SECRET_ACCESS_KEY", "LPBS_SERVICE_SECRET"} {
		t.Setenv(key, "synthetic-legacy-environment-value")
		value := ResolveSecret(key, nil)
		if value != "" {
			t.Fatalf("ResolveSecret(%s) returned a value while authority was unavailable", key)
		}
	}
}
