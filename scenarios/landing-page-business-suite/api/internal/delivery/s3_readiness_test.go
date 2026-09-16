package delivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// readinessS3Server emulates the subset of the S3 API that VerifyOperations
// touches. The handler ignores object paths and signatures: the test varies
// only the status code per operation so a bucket that is discoverable but not
// writable is reproducible without MinIO or real credentials.
func readinessS3Server(headStatus, putStatus, getStatus, deleteStatus int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(headStatus)
		case http.MethodPut:
			w.WriteHeader(putStatus)
		case http.MethodGet:
			if getStatus == http.StatusOK {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("vrooli-s3-readiness-v1"))
			} else {
				w.WriteHeader(getStatus)
			}
		case http.MethodDelete:
			w.WriteHeader(deleteStatus)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
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

func TestVerifyOperationsProvesWriteReadDelete(t *testing.T) {
	server := readinessS3Server(http.StatusOK, http.StatusOK, http.StatusOK, http.StatusNoContent)
	defer server.Close()

	if err := readinessS3Verifier(t, server.URL).VerifyOperations(context.Background(), "readiness-bucket", "artifacts"); err != nil {
		t.Fatalf("expected complete permission set to pass, got %v", err)
	}
}

func TestVerifyOperationsRejectsDiscoverableButUnwritableBucket(t *testing.T) {
	// HeadBucket succeeds, so a shallow TestConnection would report ready even
	// though the bucket cannot accept a released artifact.
	server := readinessS3Server(http.StatusOK, http.StatusForbidden, http.StatusOK, http.StatusNoContent)
	defer server.Close()

	err := readinessS3Verifier(t, server.URL).VerifyOperations(context.Background(), "readiness-bucket", "artifacts")
	if err == nil || !strings.Contains(err.Error(), "write readiness object") {
		t.Fatalf("expected denied write to fail readiness, got %v", err)
	}
}

func TestVerifyOperationsRejectsUnreadableBucket(t *testing.T) {
	server := readinessS3Server(http.StatusOK, http.StatusOK, http.StatusForbidden, http.StatusNoContent)
	defer server.Close()

	err := readinessS3Verifier(t, server.URL).VerifyOperations(context.Background(), "readiness-bucket", "artifacts")
	if err == nil || !strings.Contains(err.Error(), "read readiness object") {
		t.Fatalf("expected denied read to fail readiness, got %v", err)
	}
}

func TestVerifyOperationsRejectsUncleanableBucket(t *testing.T) {
	server := readinessS3Server(http.StatusOK, http.StatusOK, http.StatusOK, http.StatusForbidden)
	defer server.Close()

	err := readinessS3Verifier(t, server.URL).VerifyOperations(context.Background(), "readiness-bucket", "artifacts")
	if err == nil || !strings.Contains(err.Error(), "delete readiness object") {
		t.Fatalf("expected denied delete to fail readiness, got %v", err)
	}
}
