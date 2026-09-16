package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type fakeAWSError struct {
	code      string
	message   string
	status    int
	requestID string
}

func (e fakeAWSError) Error() string            { return e.code + ": " + e.message }
func (e fakeAWSError) ErrorCode() string        { return e.code }
func (e fakeAWSError) ErrorMessage() string     { return e.message }
func (e fakeAWSError) HTTPStatusCode() int      { return e.status }
func (e fakeAWSError) ServiceRequestID() string { return e.requestID }

func TestClassifyS3ErrorCategories(t *testing.T) {
	cases := []struct {
		name        string
		err         error
		operation   string
		wantCode    DiagnosticCode
		wantRetry   bool
		wantRequest string
	}{
		{
			name:      "authentication rejected",
			err:       fakeAWSError{code: "SignatureDoesNotMatch", message: "signature mismatch", status: 403},
			operation: "PutObject",
			wantCode:  CodeAuthenticationRejected,
		},
		{
			name:      "access key inactive",
			err:       fakeAWSError{code: "InvalidAccessKeyId", message: "The AWS Access Key Id you provided is currently inactive.", status: 403},
			operation: "PutObject",
			wantCode:  CodeAccessKeyInactive,
		},
		{
			name:      "session token expired",
			err:       fakeAWSError{code: "ExpiredToken", message: "token expired", status: 400},
			operation: "GetObject",
			wantCode:  CodeSessionTokenInvalid,
			wantRetry: true,
		},
		{
			name:      "encryption permission denied",
			err:       fakeAWSError{code: "KMS.AccessDeniedException", message: "not authorized to use kms key", status: 403},
			operation: "PutObject",
			wantCode:  CodeEncryptionPermission,
		},
		{
			name:      "list denied",
			err:       fakeAWSError{code: "AccessDenied", message: "Access Denied", status: 403},
			operation: "ListBucket",
			wantCode:  CodeListBucketDenied,
		},
		{
			name:      "get denied",
			err:       fakeAWSError{code: "AccessDenied", message: "Access Denied", status: 403},
			operation: "GetObject",
			wantCode:  CodeGetObjectDenied,
		},
		{
			name:      "put denied",
			err:       fakeAWSError{code: "AccessDenied", message: "Access Denied", status: 403},
			operation: "PutObject",
			wantCode:  CodePutObjectDenied,
		},
		{
			name:      "delete denied",
			err:       fakeAWSError{code: "AccessDenied", message: "Access Denied", status: 403},
			operation: "DeleteObject",
			wantCode:  CodeDeleteObjectDenied,
		},
		{
			name:      "wrong region",
			err:       fakeAWSError{code: "PermanentRedirect", message: "wrong region", status: 301},
			operation: "HeadBucket",
			wantCode:  CodeBucketWrongRegion,
		},
		{
			name:      "timeout",
			err:       context.DeadlineExceeded,
			operation: "HeadBucket",
			wantCode:  CodeRequestTimeout,
			wantRetry: true,
		},
		{
			name:      "network failure",
			err:       &url.Error{Op: "Get", URL: "https://s3.example.com", Err: errors.New("dial tcp: connection refused")},
			operation: "HeadBucket",
			wantCode:  CodeNetworkFailure,
			wantRetry: true,
		},
		{
			name:        "request id preserved",
			err:         fakeAWSError{code: "InternalError", message: "boom", status: 500, requestID: "req-123"},
			operation:   "PutObject",
			wantCode:    CodeOperationFailed,
			wantRetry:   true,
			wantRequest: "req-123",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostic := ClassifyS3Error(tc.err, tc.operation, "vrooli-bucket", "us-east-1")
			if diagnostic == nil {
				t.Fatal("expected diagnostic")
			}
			if diagnostic.Code != tc.wantCode {
				t.Fatalf("code = %s, want %s (summary: %s)", diagnostic.Code, tc.wantCode, diagnostic.Summary)
			}
			if diagnostic.Retryable != tc.wantRetry {
				t.Fatalf("retryable = %v, want %v", diagnostic.Retryable, tc.wantRetry)
			}
			if tc.wantRequest != "" && diagnostic.RequestID != tc.wantRequest {
				t.Fatalf("request id = %q, want %q", diagnostic.RequestID, tc.wantRequest)
			}
			if diagnostic.Bucket != "vrooli-bucket" || diagnostic.Region != "us-east-1" {
				t.Fatalf("diagnostic lost addressing context: %#v", diagnostic)
			}
			if diagnostic.Remediation == "" {
				t.Fatalf("diagnostic for %s has no remediation", diagnostic.Code)
			}
		})
	}
}

func TestClassifyS3ErrorNeverLeaksSecrets(t *testing.T) {
	secret := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	err := fakeAWSError{code: "SignatureDoesNotMatch", message: "The request signature we calculated does not match " + secret, status: 403}
	diagnostic := ClassifyS3Error(err, "PutObject", "vrooli-bucket", "us-east-1")
	if strings.Contains(diagnostic.Error(), secret) || strings.Contains(diagnostic.Remediation, secret) {
		t.Fatalf("diagnostic leaked secret material: %#v", diagnostic)
	}
}

func TestCredentialPresenceStates(t *testing.T) {
	cases := []struct {
		name      string
		read      func(context.Context, string) (string, error)
		wantState CredentialState
		wantSet   bool
	}{
		{"configured", func(context.Context, string) (string, error) { return "AKIAEXAMPLE", nil }, CredentialConfigured, true},
		{"missing", func(context.Context, string) (string, error) { return "", credentialauthority.ErrUnconfigured }, CredentialMissing, false},
		{"unavailable", func(context.Context, string) (string, error) { return "", credentialauthority.ErrProviderUnavailable }, CredentialUnavailable, false},
		{"absent", func(context.Context, string) (string, error) { return "", credentialauthority.ErrProviderAbsent }, CredentialUnavailable, false},
		{"authority error", func(context.Context, string) (string, error) { return "", errors.New("boom") }, CredentialAuthorityError, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService(nil).WithCredentialIO(CredentialIO{Read: tc.read})
			presence := service.CredentialPresence(context.Background())
			if presence.AccessKeyID != tc.wantState {
				t.Fatalf("access key state = %s, want %s", presence.AccessKeyID, tc.wantState)
			}
			if presence.AccessKeyIDSet != tc.wantSet {
				t.Fatalf("access key set = %v, want %v", presence.AccessKeyIDSet, tc.wantSet)
			}
			if tc.wantState == CredentialUnavailable || tc.wantState == CredentialAuthorityError {
				if presence.CredentialsSource == CredentialSourceAuthority {
					t.Fatalf("expected non-authority source for %s, got %s", tc.wantState, presence.CredentialsSource)
				}
			}
		})
	}
}

func TestCredentialPresenceWithoutAuthorityIsUnavailable(t *testing.T) {
	presence := NewService(nil).CredentialPresence(context.Background())
	if presence.AccessKeyID != CredentialUnavailable || presence.SecretAccessKey != CredentialUnavailable {
		t.Fatalf("expected unavailable presence without authority, got %#v", presence)
	}
	if presence.CredentialsSource != "unavailable" {
		t.Fatalf("credentials source = %q, want unavailable", presence.CredentialsSource)
	}
}

func TestBuildProvisioningGuideSubstitutesBucketAndWarns(t *testing.T) {
	guide := BuildProvisioningGuide("vrooli-bucket", "us-east-1")
	if guide.Bucket != "vrooli-bucket" || guide.Region != "us-east-1" {
		t.Fatalf("guide addressing wrong: %#v", guide)
	}
	if !strings.Contains(guide.PolicyDocument, "arn:aws:s3:::vrooli-bucket/*") {
		t.Fatalf("policy object ARN not substituted: %s", guide.PolicyDocument)
	}
	if !strings.Contains(guide.PolicyDocument, `"Resource": "arn:aws:s3:::vrooli-bucket"`) {
		t.Fatalf("policy bucket ARN not substituted: %s", guide.PolicyDocument)
	}
	if len(guide.Warnings) < 5 {
		t.Fatalf("expected the full safety warning set, got %d", len(guide.Warnings))
	}
	joined := strings.Join(guide.Warnings, "\n")
	for _, fragment := range []string{"displayed only once", "root user", "AdministratorAccess", "two access keys"} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("warnings missing %q: %s", fragment, joined)
		}
	}
	if !strings.Contains(strings.Join(guide.LocalCommands, "\n"), "delivery-s3-secret-access-key") {
		t.Fatalf("local commands missing provision command: %#v", guide.LocalCommands)
	}
	if guide.PrivateBucketNote == "" || guide.SessionTokenNote == "" {
		t.Fatalf("guide missing notes: %#v", guide)
	}
}

func TestSettingsSnapshotNeverSerializesSecretValues(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectQuery("SELECT id, bundle_key, provider, bucket, region, endpoint, force_path_style, default_prefix,").
		WithArgs("business_suite").
		WillReturnRows(storageSettingsRows())

	const secret = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	service := NewService(db).WithCredentialIO(CredentialIO{
		Read: func(_ context.Context, field string) (string, error) {
			if field == CredentialFieldSecretAccessKey {
				return secret, nil
			}
			return "", credentialauthority.ErrUnconfigured
		},
	})
	snapshot, err := service.SettingsSnapshot(context.Background(), "business_suite")
	if err != nil {
		t.Fatalf("SettingsSnapshot: %v", err)
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	if strings.Contains(string(encoded), secret) {
		t.Fatalf("snapshot serialized the secret value: %s", encoded)
	}
	if !snapshot.SecretAccessKeySet || snapshot.SecretAccessKeyState != string(CredentialConfigured) {
		t.Fatalf("expected present secret state, got %#v", snapshot)
	}
}

func TestMissingCredentialsDiagnosticNamesMissingField(t *testing.T) {
	presence := CredentialPresence{
		AccessKeyID:     CredentialConfigured,
		SecretAccessKey: CredentialMissing,
	}
	diagnostic := MissingCredentialsDiagnostic(presence, "vrooli-bucket", "us-east-1")
	if diagnostic.Code != CodeSecretAccessKeyMissing {
		t.Fatalf("code = %s, want %s", diagnostic.Code, CodeSecretAccessKeyMissing)
	}
	if !strings.Contains(diagnostic.Summary, "secret") {
		t.Fatalf("summary does not name the missing field: %s", diagnostic.Summary)
	}
}

func storageSettingsRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "bundle_key", "provider", "bucket", "region", "endpoint", "force_path_style",
		"default_prefix", "signed_url_ttl_seconds", "public_base_url", "created_at", "updated_at",
	})
}

func TestValidateStorageReportsMissingCredentialsWithGuide(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	now := time.Now()
	mock.ExpectQuery("SELECT id, bundle_key, provider, bucket, region, endpoint, force_path_style, default_prefix,").
		WithArgs("business_suite").
		WillReturnRows(storageSettingsRows().AddRow(1, "business_suite", "s3", "vrooli-bucket", "us-east-1", nil, false, "artifacts", 900, nil, now, now))

	service := NewService(db).WithCredentialIO(CredentialIO{
		Read: func(context.Context, string) (string, error) { return "", credentialauthority.ErrUnconfigured },
	})
	result, err := service.ValidateStorage(context.Background(), "business_suite")
	if err == nil {
		t.Fatal("expected validation failure for missing credentials")
	}
	diagnostic, ok := FromDiagnostic(err)
	if !ok {
		t.Fatalf("error %v is not a structured diagnostic", err)
	}
	if diagnostic.Code != CodeAccessKeyIDMissing {
		t.Fatalf("code = %s, want %s", diagnostic.Code, CodeAccessKeyIDMissing)
	}
	if result == nil || result.Ready {
		t.Fatalf("expected not-ready result, got %#v", result)
	}
	if result.Guide == nil || result.Guide.Bucket != "vrooli-bucket" {
		t.Fatalf("expected provisioning guide for configured bucket, got %#v", result.Guide)
	}
	if result.Presence.AccessKeyIDSet || result.Presence.SecretAccessKeySet {
		t.Fatalf("expected absent presence flags, got %#v", result.Presence)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestValidateStorageReportsAuthorityUnavailable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	now := time.Now()
	mock.ExpectQuery("SELECT id, bundle_key, provider, bucket, region, endpoint, force_path_style, default_prefix,").
		WithArgs("business_suite").
		WillReturnRows(storageSettingsRows().AddRow(1, "business_suite", "s3", "vrooli-bucket", "us-east-1", nil, false, "artifacts", 900, nil, now, now))

	service := NewService(db).WithCredentialIO(CredentialIO{
		Read: func(context.Context, string) (string, error) { return "", credentialauthority.ErrProviderUnavailable },
	})
	result, err := service.ValidateStorage(context.Background(), "business_suite")
	if err == nil {
		t.Fatal("expected validation failure when the authority is unavailable")
	}
	diagnostic, _ := FromDiagnostic(err)
	if diagnostic.Code != CodeAuthorityUnavailable {
		t.Fatalf("code = %s, want %s", diagnostic.Code, CodeAuthorityUnavailable)
	}
	if result == nil || result.Presence.CredentialsSource != "unavailable" {
		t.Fatalf("expected unavailable source, got %#v", result)
	}
}

func TestValidateStorageReportsMissingBucketWhenUnconfigured(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectQuery("SELECT id, bundle_key, provider, bucket, region, endpoint, force_path_style, default_prefix,").
		WithArgs("business_suite").
		WillReturnRows(storageSettingsRows())

	service := NewService(db).WithCredentialIO(CredentialIO{
		Read: func(context.Context, string) (string, error) { return "AKIAEXAMPLE", nil },
	})
	result, err := service.ValidateStorage(context.Background(), "business_suite")
	if err == nil {
		t.Fatal("expected validation failure without a bucket")
	}
	diagnostic, _ := FromDiagnostic(err)
	if diagnostic.Code != CodeBucketNameMissing {
		t.Fatalf("code = %s, want %s", diagnostic.Code, CodeBucketNameMissing)
	}
	if result == nil || result.Guide == nil {
		t.Fatalf("expected a guide with the missing-bucket failure, got %#v", result)
	}
}
