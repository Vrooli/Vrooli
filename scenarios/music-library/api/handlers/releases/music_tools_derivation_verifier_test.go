package releases

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	domain "music-library/internal/releases"
)

func validMusicToolsReceipt() musicToolsDerivationReceipt {
	return musicToolsDerivationReceipt{
		SchemaVersion: "1", JobID: "job-1", Kind: "sample",
		SourceTrackSHA256:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		DerivedTrackSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		SourceStartSeconds: 1, SourceEndSeconds: 5, DerivedStartSeconds: 8, DerivedEndSeconds: 12,
		ResultRef: "outputs/job-1/derived.wav", OperationVersion: "1", ToolName: "ffmpeg",
		ToolVersion: "7.0", InheritedRightsScope: "personal_only",
	}
}

func validMusicLibraryDerivation() domain.Derivation {
	return domain.Derivation{
		ID: "sample-1", Kind: "sample", SourceTrackSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		DerivedTrackSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		OperationOwner:     "music-tools", OperationID: "job-1", SourceStartSeconds: 1, SourceEndSeconds: 5,
		DerivedStartSeconds: 8, DerivedEndSeconds: 12, InheritedRights: "personal_only", EvidenceRef: "music-tools:job-1",
	}
}

func TestMusicToolsDerivationVerifierBindsReceiptToReleaseLineage(t *testing.T) {
	receipt := validMusicToolsReceipt()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/jobs/job-1/derivation-receipt", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(receipt)
	}))
	defer server.Close()

	verifier := &musicToolsDerivationVerifier{
		baseURL: func(context.Context) (string, error) { return server.URL, nil },
		client:  server.Client(),
	}
	verified, err := verifier.VerifyDerivation(context.Background(), validMusicLibraryDerivation())
	require.NoError(t, err)
	require.True(t, verified)
}

func TestMusicToolsDerivationVerifierRejectsUnmatchedAndUnreviewableReceipts(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		mutate     func(*musicToolsDerivationReceipt)
		body       string
		wantVerify bool
	}{
		{name: "wrong source", status: http.StatusOK, mutate: func(r *musicToolsDerivationReceipt) {
			r.SourceTrackSHA256 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
		}},
		{name: "wrong output", status: http.StatusOK, mutate: func(r *musicToolsDerivationReceipt) {
			r.DerivedTrackSHA256 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
		}},
		{name: "wrong range", status: http.StatusOK, mutate: func(r *musicToolsDerivationReceipt) { r.SourceEndSeconds = 8 }},
		{name: "rights widening", status: http.StatusOK, mutate: func(r *musicToolsDerivationReceipt) { r.InheritedRightsScope = "commercial" }},
		{name: "job not successful or receipt absent", status: http.StatusPreconditionFailed, body: "job has no verified derivation receipt"},
		{name: "invalid JSON", status: http.StatusOK, body: "{"},
		{name: "trailing JSON", status: http.StatusOK, body: `{} {}`},
		{name: "oversized", status: http.StatusOK, body: strings.Repeat("x", (64<<10)+1)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				if tc.body != "" {
					_, _ = w.Write([]byte(tc.body))
					return
				}
				receipt := validMusicToolsReceipt()
				tc.mutate(&receipt)
				_ = json.NewEncoder(w).Encode(receipt)
			}))
			defer server.Close()
			verifier := &musicToolsDerivationVerifier{
				baseURL: func(context.Context) (string, error) { return server.URL, nil },
				client:  server.Client(),
			}
			verified, err := verifier.VerifyDerivation(context.Background(), validMusicLibraryDerivation())
			require.NoError(t, err)
			require.False(t, verified)
		})
	}
}

func TestMusicToolsDerivationVerifierFailsClosedForInvalidOwnerOrUnavailableService(t *testing.T) {
	verifier := &musicToolsDerivationVerifier{baseURL: func(context.Context) (string, error) { return "", errors.New("owner unavailable") }}
	item := validMusicLibraryDerivation()
	verified, err := verifier.VerifyDerivation(context.Background(), item)
	require.NoError(t, err)
	require.False(t, verified)

	item.OperationOwner = "persona"
	verified, err = verifier.VerifyDerivation(context.Background(), item)
	require.NoError(t, err)
	require.False(t, verified)
}
