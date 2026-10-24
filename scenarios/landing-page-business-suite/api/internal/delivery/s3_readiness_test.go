package delivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
