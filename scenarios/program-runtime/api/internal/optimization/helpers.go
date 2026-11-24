package optimization

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strings"
)

func digestCases(caseIDs []string) string {
	encoded, _ := json.Marshal(caseIDs)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func candidateID(value any) string {
	v, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

func cloneEvidence(in []Evidence) []Evidence {
	out := make([]Evidence, len(in))
	copy(out, in)
	for i := range out {
		if in[i].Details != nil {
			out[i].Details = make(map[string]any, len(in[i].Details))
			for key, value := range in[i].Details {
				out[i].Details[key] = value
			}
		}
		if in[i].Proof != nil {
			proof := *in[i].Proof
			out[i].Proof = &proof
		}
	}
	return out
}

func findCandidate(candidates []Candidate, id string) *Candidate {
	for i := range candidates {
		if candidates[i].ID == id {
			return &candidates[i]
		}
	}
	return nil
}

// SubjectDigest returns the canonical rights/disk admission subject hash used
// by owner adapters for the pinned request and candidate cohort.
func SubjectDigest(request Request) string {
	expected := make(map[string]string, len(request.Candidates))
	for _, candidate := range request.Candidates {
		expected[candidate.ID] = candidate.ArtifactDigest
	}
	return subjectDigest(request, expected)
}

// AcquiredSubjectDigest binds owner proofs to the actual content digests
// returned by the acquisition owner. Candidate.ArtifactDigest is only an
// optional pre-acquisition expectation; it may be unknown for a registry tag.
func AcquiredSubjectDigest(request Request, evidence []Evidence) string {
	return subjectDigest(request, acquiredDigests(evidence))
}

func subjectDigest(request Request, artifactDigests map[string]string) string {
	body := struct {
		Target          string            `json:"target"`
		Subject         map[string]any    `json:"subject"`
		Candidates      []Candidate       `json:"candidates"`
		ArtifactDigests map[string]string `json:"artifact_digests"`
		PredecessorID   string            `json:"predecessor_id,omitempty"`
		SuiteID         string            `json:"suite_id"`
		CaseIDs         []string          `json:"case_ids"`
		MinimumCases    int               `json:"minimum_cases"`
	}{request.Target, request.Subject, request.Candidates, artifactDigests, request.PredecessorID, request.SuiteID, request.CaseIDs, request.MinimumCases}
	encoded, _ := json.Marshal(body)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func acquiredDigests(evidence []Evidence) map[string]string {
	for i := len(evidence) - 1; i >= 0; i-- {
		if evidence[i].Stage != StageAcquire || evidence[i].Details == nil {
			continue
		}
		if digests, ok := stringMapFromAny(evidence[i].Details["candidate_artifact_digests"]); ok {
			return digests
		}
	}
	return map[string]string{}
}

func cloneRequest(request Request) (Request, error) {
	encoded, err := json.Marshal(request)
	if err != nil {
		return Request{}, err
	}
	var cloned Request
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return Request{}, err
	}
	return cloned, nil
}

func integerFromAny(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), int64(int(v)) == v
	case float64:
		return int(v), float64(int(v)) == v
	case json.Number:
		n, err := v.Int64()
		return int(n), err == nil && int64(int(n)) == n
	default:
		return 0, false
	}
}

func stringMapFromAny(value any) (map[string]string, bool) {
	switch v := value.(type) {
	case map[string]string:
		return v, true
	case map[string]any:
		out := make(map[string]string, len(v))
		for key, item := range v {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			out[key] = text
		}
		return out, true
	default:
		return nil, false
	}
}

func mapFromAny(value any) (map[string]any, bool) { v, ok := value.(map[string]any); return v, ok }

func isFiniteNumber(value any) bool {
	switch v := value.(type) {
	case float64:
		return !math.IsNaN(v) && !math.IsInf(v, 0)
	case float32:
		return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case json.Number:
		n, err := v.Float64()
		return err == nil && !math.IsNaN(n) && !math.IsInf(n, 0)
	default:
		return false
	}
}
