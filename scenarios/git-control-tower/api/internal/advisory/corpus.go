package advisory

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed fixtures/grounding-corpus.json
var groundingCorpusBytes []byte

type GroundingCorpus struct {
	CorpusID string          `json:"corpus_id"`
	Version  string          `json:"version"`
	Cases    []GroundingCase `json:"cases"`
}

type GroundingCase struct {
	ID               string   `json:"id"`
	SubjectKind      string   `json:"subject_kind"`
	Labels           []string `json:"labels"`
	Facts            []string `json:"facts"`
	Prohibited       []string `json:"prohibited_claims"`
	RequiredUnknown  []string `json:"required_unknowns"`
	KnownDefects     []string `json:"known_defects"`
	ExpectedFindings []string `json:"expected_findings"`
}

func LoadGroundingCorpus() (GroundingCorpus, error) {
	var corpus GroundingCorpus
	if err := json.Unmarshal(groundingCorpusBytes, &corpus); err != nil {
		return GroundingCorpus{}, fmt.Errorf("decode grounding corpus: %w", err)
	}
	if strings.TrimSpace(corpus.CorpusID) == "" || strings.TrimSpace(corpus.Version) == "" || len(corpus.Cases) == 0 {
		return GroundingCorpus{}, fmt.Errorf("grounding corpus requires id, version, and cases")
	}
	seen := map[string]bool{}
	for _, item := range corpus.Cases {
		if strings.TrimSpace(item.ID) == "" || seen[item.ID] {
			return GroundingCorpus{}, fmt.Errorf("grounding corpus has invalid or duplicate case %q", item.ID)
		}
		seen[item.ID] = true
		if item.SubjectKind != string(SubjectCurrent) && item.SubjectKind != string(SubjectStaged) && item.SubjectKind != string(SubjectCommit) && item.SubjectKind != string(SubjectRange) && item.SubjectKind != string(SubjectPullRequest) {
			return GroundingCorpus{}, fmt.Errorf("case %q has unsupported subject kind %q", item.ID, item.SubjectKind)
		}
		if len(item.Facts) == 0 || len(item.Prohibited) == 0 || len(item.RequiredUnknown) == 0 {
			return GroundingCorpus{}, fmt.Errorf("case %q must label facts, prohibited claims, and unknowns", item.ID)
		}
		if len(item.KnownDefects) > 0 && len(item.ExpectedFindings) == 0 {
			return GroundingCorpus{}, fmt.Errorf("case %q has defects but no expected findings", item.ID)
		}
	}
	return corpus, nil
}
