package facts

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
)

type candidateReceiptURLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

type candidateReceiptMatchRequest struct {
	ReceiptID string                                `json:"receiptId"`
	Candidate candidateReceiptMatchRequestCandidate `json:"candidate"`
}

type candidateReceiptMatchRequestCandidate struct {
	CandidateID      string `json:"candidateId"`
	Scenario         string `json:"scenario"`
	RelativePath     string `json:"relativePath"`
	SourceLine       uint32 `json:"sourceLine"`
	SourceFileSHA256 string `json:"sourceFileSha256"`
}

type candidateReceiptMatchResponse struct {
	Status    string                                `json:"status"`
	ReceiptID string                                `json:"receiptId"`
	Candidate candidateReceiptMatchRequestCandidate `json:"candidate"`
	Reason    string                                `json:"reason"`
}

// CandidateReceiptMatchHTTPHandler checks one caller-selected Test Genie
// receipt against one Code Facts candidate. The owner remains authoritative;
// transport failures are unavailable, while owner UNKNOWN stays an explicit
// response and never becomes a no-result or enforcement claim.
func CandidateReceiptMatchHTTPHandler(resolver candidateReceiptURLResolver, httpClient connect.HTTPClient) http.HandlerFunc {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
		decoder.DisallowUnknownFields()
		var request candidateReceiptMatchRequest
		if err := decoder.Decode(&request); err != nil {
			var maxBytes *http.MaxBytesError
			if errors.As(err, &maxBytes) {
				http.Error(w, "request body exceeds limit", http.StatusRequestEntityTooLarge)
			} else {
				http.Error(w, "invalid request body", http.StatusBadRequest)
			}
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			http.Error(w, "request body must contain one JSON object", http.StatusBadRequest)
			return
		}
		request.ReceiptID = strings.TrimSpace(request.ReceiptID)
		request.Candidate.CandidateID = strings.TrimSpace(request.Candidate.CandidateID)
		request.Candidate.Scenario = strings.TrimSpace(request.Candidate.Scenario)
		request.Candidate.RelativePath = strings.TrimSpace(request.Candidate.RelativePath)
		request.Candidate.SourceFileSHA256 = strings.TrimSpace(request.Candidate.SourceFileSHA256)
		if request.ReceiptID == "" || len(request.ReceiptID) > 256 || request.Candidate.CandidateID == "" || len(request.Candidate.CandidateID) > 512 ||
			request.Candidate.Scenario == "" || request.Candidate.RelativePath == "" || request.Candidate.SourceLine == 0 || request.Candidate.SourceFileSHA256 == "" {
			http.Error(w, "receipt ID and complete candidate source identity are required", http.StatusBadRequest)
			return
		}
		if resolver == nil {
			http.Error(w, "Test Genie URL resolver unavailable", http.StatusServiceUnavailable)
			return
		}
		baseURL, err := resolver.ResolveScenarioURLDefault(r.Context(), "test-genie")
		if err != nil || strings.TrimSpace(baseURL) == "" {
			http.Error(w, "Test Genie receipt service unavailable", http.StatusServiceUnavailable)
			return
		}
		client := validationconnect.NewValidationServiceClient(httpClient, strings.TrimRight(baseURL, "/"))
		ownerResponse, err := client.GetCandidateReceiptMatch(r.Context(), connect.NewRequest(&validationv1.GetCandidateReceiptMatchRequest{
			ReceiptId: request.ReceiptID,
			Candidate: &validationv1.CandidateEvidenceReference{
				CandidateId: request.Candidate.CandidateID, Scenario: request.Candidate.Scenario, RelativePath: request.Candidate.RelativePath,
				SourceLine: request.Candidate.SourceLine, SourceFileSha256: request.Candidate.SourceFileSHA256,
			},
		}))
		if err != nil {
			if connect.CodeOf(err) == connect.CodeInvalidArgument {
				http.Error(w, "Test Genie rejected the candidate or receipt reference", http.StatusBadRequest)
			} else {
				http.Error(w, "Test Genie receipt service unavailable", http.StatusServiceUnavailable)
			}
			return
		}
		if ownerResponse == nil || ownerResponse.Msg == nil {
			http.Error(w, "Test Genie returned no receipt match", http.StatusServiceUnavailable)
			return
		}
		status := "UNKNOWN"
		if ownerResponse.Msg.GetStatus() == validationv1.CandidateReceiptMatchStatus_CANDIDATE_RECEIPT_MATCH_STATUS_EXACT_MATCH {
			status = "EXACT_MATCH"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(candidateReceiptMatchResponse{
			Status: status, ReceiptID: ownerResponse.Msg.GetReceiptId(), Candidate: request.Candidate, Reason: ownerResponse.Msg.GetReason(),
		})
	}
}
