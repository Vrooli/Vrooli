package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

type Corpus struct {
	Version     string       `json:"version"`
	ContentHash string       `json:"content_hash"`
	Cases       []CorpusCase `json:"cases"`
}
type CorpusCase struct {
	ID             string           `json:"id"`
	Split          string           `json:"split"`
	Task           string           `json:"task"`
	Sources        []CorpusSource   `json:"sources"`
	Questions      []CorpusQuestion `json:"questions"`
	ExpectedClaims []ExpectedClaim  `json:"expected_claims"`
	SourceChange   bool             `json:"source_change"`
	Provenance     string           `json:"provenance"`
}
type CorpusSource struct {
	ID          string `json:"id"`
	Revision    string `json:"revision"`
	Content     string `json:"content"`
	ContentHash string `json:"content_hash"`
}
type CorpusQuestion struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Required bool   `json:"required"`
}
type ExpectedClaim struct {
	ClaimID     string `json:"claim_id"`
	Text        string `json:"text"`
	Disposition string `json:"disposition"`
	SourceID    string `json:"source_id"`
	StartByte   int    `json:"start_byte"`
	EndByte     int    `json:"end_byte"`
}

func (c Corpus) Validate() error {
	if strings.TrimSpace(c.Version) == "" || strings.TrimSpace(c.ContentHash) == "" {
		return fmt.Errorf("corpus version and content hash are required")
	}
	if len(c.Cases) == 0 {
		return fmt.Errorf("corpus requires cases")
	}
	if c.ContentHash != c.ManifestHash() {
		return fmt.Errorf("corpus content hash does not match its frozen manifest")
	}
	caseIDs, contents := map[string]bool{}, map[string]string{}
	for _, item := range c.Cases {
		if item.ID == "" || caseIDs[item.ID] {
			return fmt.Errorf("case IDs must be present and unique")
		}
		caseIDs[item.ID] = true
		if item.Split != "development" && item.Split != "holdout" && item.Split != "live" {
			return fmt.Errorf("case %q has invalid split", item.ID)
		}
		if item.Provenance != "fixture" && item.Provenance != "permitted_snapshot" {
			return fmt.Errorf("case %q has invalid provenance", item.ID)
		}
		if len(item.Sources) == 0 || len(item.Questions) == 0 {
			return fmt.Errorf("case %q requires sources and questions", item.ID)
		}
		sourceIDs := map[string]bool{}
		for _, source := range item.Sources {
			if source.ID == "" || source.Revision == "" || source.Content == "" || sourceIDs[source.ID] {
				return fmt.Errorf("case %q has invalid source", item.ID)
			}
			sourceIDs[source.ID] = true
			h := sha256.Sum256([]byte(source.Content))
			hash := hex.EncodeToString(h[:])
			if source.ContentHash != hash {
				return fmt.Errorf("source %q has incorrect content hash", source.ID)
			}
			if previous, exists := contents[hash]; exists && previous != item.Split {
				return fmt.Errorf("source content is duplicated across splits")
			}
			contents[hash] = item.Split
		}
		questions := map[string]bool{}
		for _, q := range item.Questions {
			if q.ID == "" || strings.TrimSpace(q.Text) == "" || questions[q.ID] {
				return fmt.Errorf("case %q has invalid questions", item.ID)
			}
			questions[q.ID] = true
		}
		seenClaims := map[string]bool{}
		for _, claim := range item.ExpectedClaims {
			if claim.ClaimID == "" || seenClaims[claim.ClaimID] || !questions[claim.ClaimID] && len(item.ExpectedClaims) > 0 {
				return fmt.Errorf("case %q has invalid expected claim identity", item.ID)
			}
			seenClaims[claim.ClaimID] = true
			if claim.Disposition != "supported" && claim.Disposition != "contradicted" && claim.Disposition != "unresolved" && claim.Disposition != "unknown" {
				return fmt.Errorf("case %q has invalid claim disposition", item.ID)
			}
			if claim.SourceID != "" {
				source, ok := sourceByID(item.Sources, claim.SourceID)
				if !ok || claim.StartByte < 0 || claim.EndByte <= claim.StartByte || claim.EndByte > len([]byte(source.Content)) || !utf8.ValidString(source.Content[claim.StartByte:claim.EndByte]) {
					return fmt.Errorf("case %q claim %q has invalid support span", item.ID, claim.ClaimID)
				}
			}
		}
		for _, q := range item.Questions {
			if q.Required && !seenClaims[q.ID] {
				return fmt.Errorf("case %q is missing expected coverage for question %q", item.ID, q.ID)
			}
		}
		if item.SourceChange && len(item.Sources) < 2 {
			return fmt.Errorf("source-change case %q requires two source revisions", item.ID)
		}
	}
	return nil
}

func (c Corpus) ManifestHash() string {
	copy := c
	copy.ContentHash = ""
	b, _ := json.Marshal(copy)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func sourceByID(sources []CorpusSource, id string) (CorpusSource, bool) {
	for _, source := range sources {
		if source.ID == id {
			return source, true
		}
	}
	return CorpusSource{}, false
}

// AnswerMatches requires the reviewed claim text, so a keyword occurrence
// cannot pass as a correct answer.
func AnswerMatches(expected, actual string) bool {
	return strings.TrimSpace(expected) != "" && strings.TrimSpace(expected) == strings.TrimSpace(actual)
}
