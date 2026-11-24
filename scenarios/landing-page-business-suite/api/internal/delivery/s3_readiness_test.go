package delivery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const readinessPayload = "vrooli-s3-readiness-v1"

// readinessScenario describes the status code each S3 operation returns. The
// handler ignores object paths and signatures: tests vary only the response per
// operation so a bucket that is discoverable but not writable, or writable but
// not cleanable, is reproducible without MinIO or real credentials.
type readinessScenario struct {
	headStatus   int
	headRegion   string
	listStatus   int
	putStatus    int
	getStatus    int
	getBody      string
	deleteStatus int
}

func (s readinessScenario) server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			if s.headRegion != "" {
				w.Header().Set("x-amz-bucket-region", s.headRegion)
			}
			w.WriteHeader(s.headStatus)
		case http.MethodPut:
			w.WriteHeader(s.putStatus)
		case http.MethodGet:
			if r.URL.Query().Get("list-type") == "2" {
				if s.listStatus == http.StatusOK {
					w.Header().Set("Content-Type", "application/xml")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult><Name>readiness-bucket</Name><IsTruncated>false</IsTruncated></ListBucketResult>`))
					return
				}
				w.WriteHeader(s.listStatus)
				return
			}
			if s.getStatus == http.StatusOK {
				w.WriteHeader(http.StatusOK)
				body := s.getBody
				if body == "" {
					body = readinessPayload
				}
				_, _ = w.Write([]byte(body))
			} else {
				w.WriteHeader(s.getStatus)
			}
		case http.MethodDelete:
			w.WriteHeader(s.deleteStatus)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
}

// readinessS3Server preserves the original four-code helper used by the
// permission tests, with listing and region successful.
func readinessS3Server(headStatus, putStatus, getStatus, deleteStatus int) *httptest.Server {
	return readinessScenario{
		headStatus:   headStatus,
		listStatus:   http.StatusOK,
		putStatus:    putStatus,
		getStatus:    getStatus,
		deleteStatus: deleteStatus,
	}.server()
}

func readinessS3Verifier(t *testing.T, endpoint string) OperationVerifier {
	t.Helper()
	provider := S3StorageProvider{ResolveCredential: func(context.Context, string) (string, error) {
		return "readiness-test-credential", nil
	}}
	storage, err := provider.New(context.Background(), StorageSettings{
		Provider:       "s3",
		Bucket:         "readiness-bucket",
		Region:         "us-east-1",
		Endpoint:       endpoint,
		ForcePathStyle: true,
		DefaultPrefix:  "artifacts",
	})
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	verifier, ok := storage.(OperationVerifier)
	if !ok {
		t.Fatalf("S3 storage does not implement OperationVerifier")
	}
	return verifier
}

func verifyScenario(t *testing.T, scenario readinessScenario) error {
	t.Helper()
	server := scenario.server()
	defer server.Close()
	return readinessS3Verifier(t, server.URL).VerifyOperations(context.Background(), "readiness-bucket", "artifacts")
}

func assertDiagnostic(t *testing.T, err error, wantCode DiagnosticCode, contains string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected failure with code %s, got nil", wantCode)
	}
	diagnostic, ok := FromDiagnostic(err)
	if !ok {
		t.Fatalf("error %v is not a structured diagnostic", err)
	}
	if diagnostic.Code != wantCode {
		t.Fatalf("diagnostic code = %s, want %s (summary: %s)", diagnostic.Code, wantCode, diagnostic.Summary)
	}
	if contains != "" && !strings.Contains(err.Error(), contains) {
		t.Fatalf("error %q does not contain %q", err.Error(), contains)
	}
}

func TestVerifyOperationsProvesWriteReadDelete(t *testing.T) {
	err := verifyScenario(t, readinessScenario{
		headStatus:   http.StatusOK,
		listStatus:   http.StatusOK,
		putStatus:    http.StatusOK,
		getStatus:    http.StatusOK,
		deleteStatus: http.StatusNoContent,
	})
	if err != nil {
		t.Fatalf("expected complete permission set to pass, got %v", err)
	}
}

func TestVerifyOperationsRejectsDiscoverableButUnwritableBucket(t *testing.T) {
	// HeadBucket succeeds, so a shallow TestConnection would report ready even
	// though the bucket cannot accept a released artifact.
	err := verifyScenario(t, readinessScenario{
		headStatus:   http.StatusOK,
		listStatus:   http.StatusOK,
		putStatus:    http.StatusForbidden,
		getStatus:    http.StatusOK,
		deleteStatus: http.StatusNoContent,
	})
	assertDiagnostic(t, err, CodePutObjectDenied, "write readiness object")
}

func TestVerifyOperationsRejectsUnreadableBucket(t *testing.T) {
	err := verifyScenario(t, readinessScenario{
		headStatus:   http.StatusOK,
		listStatus:   http.StatusOK,
		putStatus:    http.StatusOK,
		getStatus:    http.StatusForbidden,
		deleteStatus: http.StatusNoContent,
	})
	assertDiagnostic(t, err, CodeGetObjectDenied, "read readiness object")
}

func TestVerifyOperationsRejectsUncleanableBucket(t *testing.T) {
	err := verifyScenario(t, readinessScenario{
		headStatus:   http.StatusOK,
		listStatus:   http.StatusOK,
		putStatus:    http.StatusOK,
		getStatus:    http.StatusOK,
		deleteStatus: http.StatusForbidden,
	})
	assertDiagnostic(t, err, CodeDeleteObjectDenied, "delete readiness object")
}

func TestVerifyOperationsRejectsListDenied(t *testing.T) {
	err := verifyScenario(t, readinessScenario{
		headStatus:   http.StatusOK,
		listStatus:   http.StatusForbidden,
		putStatus:    http.StatusOK,
		getStatus:    http.StatusOK,
		deleteStatus: http.StatusNoContent,
	})
	assertDiagnostic(t, err, CodeListBucketDenied, "list readiness prefix")
}

func TestVerifyOperationsRejectsWrongRegion(t *testing.T) {
	err := verifyScenario(t, readinessScenario{
		headStatus:   http.StatusOK,
		headRegion:   "eu-west-1",
		listStatus:   http.StatusOK,
		putStatus:    http.StatusOK,
		getStatus:    http.StatusOK,
		deleteStatus: http.StatusNoContent,
	})
	assertDiagnostic(t, err, CodeBucketWrongRegion, "eu-west-1")
}

func TestVerifyOperationsRejectsMissingBucket(t *testing.T) {
	err := verifyScenario(t, readinessScenario{
		headStatus:   http.StatusNotFound,
		listStatus:   http.StatusOK,
		putStatus:    http.StatusOK,
		getStatus:    http.StatusOK,
		deleteStatus: http.StatusNoContent,
	})
	assertDiagnostic(t, err, CodeBucketNotFound, "")
}

func TestS3CredentialResolutionFailsClosedWithoutAuthorityPair(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "ambient-fixture-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "ambient-fixture-secret")
	calls := []string{}
	provider := S3StorageProvider{ResolveCredential: func(_ context.Context, field string) (string, error) {
		calls = append(calls, field)
		return "", credentialauthority.ErrUnconfigured
	}}
	_, err := provider.New(context.Background(), StorageSettings{Provider: "s3", Bucket: "unused", Region: "us-east-1"})
	if err == nil || !strings.Contains(err.Error(), "required credential") {
		t.Fatalf("storage error = %v; want a fail-closed missing authority credential error", err)
	}
	if strings.Contains(err.Error(), "ambient-fixture") {
		t.Fatalf("storage error leaked ambient fixture data: %v", err)
	}
	if len(calls) != 3 || calls[0] != CredentialFieldAccessKeyID || calls[1] != CredentialFieldSecretAccessKey || calls[2] != CredentialFieldSessionToken {
		t.Fatalf("resolved fields = %#v; want only required authority fields before refusing", calls)
	}
}

func TestS3CredentialResolutionPreservesProviderOutageAndDoesNotUseAmbientEnv(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "ambient-fixture-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "ambient-fixture-secret")
	providerErr := errors.New("isolated authority unavailable")
	provider := S3StorageProvider{ResolveCredential: func(_ context.Context, field string) (string, error) {
		if field == CredentialFieldAccessKeyID {
			return "", providerErr
		}
		return "synthetic-secret", nil
	}}
	_, err := provider.New(context.Background(), StorageSettings{Provider: "s3", Bucket: "unused", Region: "us-east-1"})
	if !errors.Is(err, providerErr) {
		t.Fatalf("storage error = %v; want provider outage", err)
	}
	if strings.Contains(err.Error(), "ambient-fixture") || strings.Contains(err.Error(), "synthetic-secret") {
		t.Fatalf("storage error leaked credential material: %v", err)
	}
}

func TestS3ProviderWithoutAuthorityResolverRefusesAmbientCredentialChain(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "ambient-fixture-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "ambient-fixture-secret")
	_, err := (S3StorageProvider{}).New(context.Background(), StorageSettings{Provider: "s3", Bucket: "unused", Region: "us-east-1"})
	if err == nil || !strings.Contains(err.Error(), "ambient AWS credential sources are disabled") {
		t.Fatalf("provider error = %v; want disabled ambient-chain refusal", err)
	}
}

type rotatingAuthorityStore struct {
	mu        sync.Mutex
	values    map[string]string
	available bool
}

func (s *rotatingAuthorityStore) Put(service, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.available {
		return errors.New("isolated provider unavailable")
	}
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[service+"/"+key] = value
	return nil
}
func (s *rotatingAuthorityStore) Get(service, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.available {
		return "", errors.New("isolated provider unavailable")
	}
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", credentialauthority.ErrNotFound, service, key)
	}
	return value, nil
}
func (s *rotatingAuthorityStore) Delete(service, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.available {
		return errors.New("isolated provider unavailable")
	}
	delete(s.values, service+"/"+key)
	return nil
}

func TestS3AuthorityVersionRotationAndOutageRecovery(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "ambient-fixture-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "ambient-fixture-secret")
	store := &rotatingAuthorityStore{available: true}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := credentialauthority.ParseIdentity("vrooli/landing-page-business-suite")
	if err != nil {
		t.Fatal(err)
	}
	activate := func(field, value string) credentialauthority.CandidateRef {
		t.Helper()
		candidate, err := authority.PutCandidate(identity, field, value)
		if err != nil {
			t.Fatalf("stage candidate for %s: %v", field, err)
		}
		if err := authority.ActivateCandidate(candidate); err != nil {
			t.Fatalf("activate candidate for %s: %v", field, err)
		}
		return candidate
	}
	firstAccess := activate(CredentialFieldAccessKeyID, "synthetic-access-v1")
	firstSecret := activate(CredentialFieldSecretAccessKey, "synthetic-secret-v1")
	resolve := func(ctx context.Context, field string) (string, error) { return authority.Require(identity, field) }
	newStorage := func() *s3Storage {
		t.Helper()
		storage, err := newS3Storage(context.Background(), StorageSettings{Provider: "s3", Bucket: "unused", Region: "us-east-1", Endpoint: "http://127.0.0.1:1", ForcePathStyle: true}, resolve)
		if err != nil {
			t.Fatalf("create storage from authority: %v", err)
		}
		return storage
	}
	readConfiguredKeys := func(storage *s3Storage) (string, string) {
		t.Helper()
		creds, err := storage.client.Options().Credentials.Retrieve(context.Background())
		if err != nil {
			t.Fatalf("retrieve isolated SDK credential fixture: %v", err)
		}
		return creds.AccessKeyID, creds.SecretAccessKey
	}
	beforeAccess, beforeSecret := readConfiguredKeys(newStorage())
	if beforeAccess != "synthetic-access-v1" || beforeSecret != "synthetic-secret-v1" {
		t.Fatal("S3 client did not bind to initial authority version")
	}
	activeAccessV1 := authority.Status(identity, CredentialFieldAccessKeyID).Version
	activeSecretV1 := authority.Status(identity, CredentialFieldSecretAccessKey).Version
	if activeAccessV1 != firstAccess.Version || activeSecretV1 != firstSecret.Version {
		t.Fatal("authority status version did not match activated candidates")
	}

	candidateAccessV2, err := authority.PutCandidate(identity, CredentialFieldAccessKeyID, "synthetic-access-v2")
	if err != nil {
		t.Fatal(err)
	}
	candidateSecretV2, err := authority.PutCandidate(identity, CredentialFieldSecretAccessKey, "synthetic-secret-v2")
	if err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.available = false
	store.mu.Unlock()
	if err := authority.ActivateCandidate(candidateAccessV2); !errors.Is(err, credentialauthority.ErrProviderUnavailable) {
		t.Fatalf("outage activation = %v; want provider unavailable", err)
	}
	if _, err := resolve(context.Background(), CredentialFieldSecretAccessKey); !errors.Is(err, credentialauthority.ErrProviderUnavailable) {
		t.Fatalf("outage resolution = %v; want provider unavailable", err)
	}
	store.mu.Lock()
	store.available = true
	store.mu.Unlock()
	stillAccess, err := authority.Require(identity, CredentialFieldAccessKeyID)
	if err != nil || stillAccess != "synthetic-access-v1" {
		t.Fatal("failed activation replaced the prior active credential")
	}
	if err := authority.ActivateCandidate(candidateAccessV2); err != nil {
		t.Fatalf("activate recovered candidate: %v", err)
	}
	if err := authority.ActivateCandidate(candidateSecretV2); err != nil {
		t.Fatalf("activate recovered secret candidate: %v", err)
	}
	rotatedAccess, rotatedSecret := readConfiguredKeys(newStorage())
	if rotatedAccess != "synthetic-access-v2" || rotatedSecret != "synthetic-secret-v2" {
		t.Fatal("new S3 client did not bind to rotated authority versions")
	}
	if authority.Status(identity, CredentialFieldAccessKeyID).Version != candidateAccessV2.Version || authority.Status(identity, CredentialFieldSecretAccessKey).Version != candidateSecretV2.Version {
		t.Fatal("rotated authority version metadata does not match candidates")
	}
}
