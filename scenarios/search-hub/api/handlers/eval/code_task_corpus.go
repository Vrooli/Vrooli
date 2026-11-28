package eval

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"search-hub/internal/corpusgen"
)

// CodeTaskCorpusBuilder is the Search Hub production-call seam. It combines
// owner verification, immutable snapshot sealing and the metadata-only
// evaluation projection; implementations must not return task or patch bodies.
type CodeTaskCorpusBuilder interface {
	BuildAndProject(context.Context, []corpusgen.CodeTaskCandidate, string) (corpusgen.CodeTaskCorpusSnapshot, []corpusgen.CodeTaskEvaluationCase, error)
}

// OwnerBoundCodeTaskCorpusBuilder is the only production implementation. Its
// constructor requires the complete source/identity/route/rights/privacy/repo/
// objective/label owner bundle before a request can reach projection.
type OwnerBoundCodeTaskCorpusBuilder struct {
	owners *corpusgen.CodeTaskEvidenceOwnerAdapter
}

func NewOwnerBoundCodeTaskCorpusBuilder(clients corpusgen.CodeTaskEvidenceOwnerClients) (*OwnerBoundCodeTaskCorpusBuilder, error) {
	owners, err := corpusgen.NewCodeTaskEvidenceOwnerAdapter(clients)
	if err != nil {
		return nil, err
	}
	return &OwnerBoundCodeTaskCorpusBuilder{owners: owners}, nil
}

func (b *OwnerBoundCodeTaskCorpusBuilder) BuildAndProject(ctx context.Context, candidates []corpusgen.CodeTaskCandidate, splitVersion string) (corpusgen.CodeTaskCorpusSnapshot, []corpusgen.CodeTaskEvaluationCase, error) {
	if b == nil || b.owners == nil {
		return corpusgen.CodeTaskCorpusSnapshot{}, nil, errors.New("code-task evidence owners are unavailable")
	}
	snapshot := corpusgen.BuildCodeTaskCorpus(ctx, candidates, splitVersion, b.owners)
	if !snapshot.Sealed {
		return snapshot, nil, errors.New("code-task corpus is not sealed; projection refused")
	}
	projected, err := snapshot.EvaluationCases()
	if err != nil {
		return snapshot, nil, err
	}
	return snapshot, projected, nil
}

type codeTaskCorpusBuildRequest struct {
	SplitVersion string                        `json:"split_version"`
	Candidates   []corpusgen.CodeTaskCandidate `json:"candidates"`
}

type codeTaskCorpusBuildResponse struct {
	Snapshot corpusgen.CodeTaskCorpusSnapshot   `json:"snapshot"`
	Cases    []corpusgen.CodeTaskEvaluationCase `json:"cases,omitempty"`
	Error    string                             `json:"error,omitempty"`
}

// BuildCodeTaskCorpus is mounted on the existing Search Hub eval module. The
// request schema contains references and hashes only; strict decoding rejects
// attempts to smuggle source/task/patch bodies into this path. A missing owner
// bundle refuses before reading the request or projecting any case.
func (h *connectHandler) BuildCodeTaskCorpus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.deps.CodeTaskCorpus == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(codeTaskCorpusBuildResponse{Error: "code-task evidence owners are unavailable"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request codeTaskCorpusBuildRequest
	if err := decoder.Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(codeTaskCorpusBuildResponse{Error: "invalid body-free code-task corpus request"})
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(codeTaskCorpusBuildResponse{Error: "request must contain one JSON object"})
		return
	}
	request.SplitVersion = strings.TrimSpace(request.SplitVersion)
	snapshot, projected, err := h.deps.CodeTaskCorpus.BuildAndProject(r.Context(), request.Candidates, request.SplitVersion)
	if err != nil {
		// Never disclose partially accepted source lineage. Only the rejection
		// codes and aggregate coverage gaps may leave this caller before a complete
		// immutable denominator exists.
		snapshot.Cases = nil
		snapshot.CaseIDs = nil
		snapshot.DenominatorSHA256 = ""
		snapshot.CorpusSHA256 = ""
		snapshot.Sealed = false
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(codeTaskCorpusBuildResponse{Snapshot: snapshot, Error: "code-task corpus refused before projection"})
		return
	}
	_ = json.NewEncoder(w).Encode(codeTaskCorpusBuildResponse{Snapshot: snapshot, Cases: projected})
}
