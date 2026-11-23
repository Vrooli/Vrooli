package verdicts

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

//go:embed evidence.seed.json
var seedBytes []byte

// AllOperations is the claim `operation` value meaning "every operation the
// model serves"; a claim naming one operation overrides it for that operation.
const AllOperations = "*"

// Claim is one curated finding: what a qualification run or comparison
// established about a model on an operation.
type Claim struct {
	ModelID   string `json:"model_id"`
	Operation string `json:"operation"`
	Grade     Grade  `json:"grade"`
	// Reason is the one-line why, written for the reader of `models list`.
	Reason string `json:"reason"`
	Kind   string `json:"kind"`
	// Ref points at the artifact that proves it. A "research:" prefix is the
	// Image Tools research topic (~/.vrooli/research/topics/image-tools-foundation/).
	Ref  string `json:"ref"`
	Date string `json:"date"`
}

// Seed is the parsed evidence.seed.json.
type Seed struct {
	SchemaVersion int     `json:"schema_version"`
	Claims        []Claim `json:"claims"`
}

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

var evidenceKinds = map[string]bool{
	KindQualification: true, KindJudgedComparison: true, KindLiveRun: true,
	KindProviderEvidence: true, KindOperatorAssessment: true,
}

// LoadSeed parses and validates the embedded evidence seed.
func LoadSeed() (*Seed, error) { return ParseSeed(seedBytes) }

// ParseSeed validates raw seed JSON: every claim names a model, a grade, a reason,
// an evidence kind, a ref and a date, and the seed never claims "untested" (the
// absence of evidence is that grade).
func ParseSeed(raw []byte) (*Seed, error) {
	var s Seed
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("verdicts: parse evidence seed: %w", err)
	}
	if s.SchemaVersion != 1 {
		return nil, fmt.Errorf("verdicts: evidence seed schema_version %d, want 1", s.SchemaVersion)
	}
	for i, c := range s.Claims {
		where := fmt.Sprintf("claim %d (%s/%s)", i, c.ModelID, c.Operation)
		switch {
		case strings.TrimSpace(c.ModelID) == "":
			return nil, fmt.Errorf("verdicts: %s: model_id is required", where)
		case strings.TrimSpace(c.Operation) == "":
			return nil, fmt.Errorf("verdicts: %s: operation is required (use %q for every operation)", where, AllOperations)
		case !c.Grade.Valid() || c.Grade == Untested:
			return nil, fmt.Errorf("verdicts: %s: grade %q must be one of recommended, works, experimental, not_recommended, broken", where, c.Grade)
		case strings.TrimSpace(c.Reason) == "":
			return nil, fmt.Errorf("verdicts: %s: a one-line reason is required", where)
		case strings.ContainsAny(c.Reason, "\r\n"):
			return nil, fmt.Errorf("verdicts: %s: reason must be one line", where)
		case !evidenceKinds[c.Kind]:
			return nil, fmt.Errorf("verdicts: %s: kind %q is not a curated evidence kind", where, c.Kind)
		case strings.TrimSpace(c.Ref) == "":
			return nil, fmt.Errorf("verdicts: %s: an evidence ref is required", where)
		case !datePattern.MatchString(c.Date):
			return nil, fmt.Errorf("verdicts: %s: date %q must be YYYY-MM-DD", where, c.Date)
		}
	}
	return &s, nil
}

// ClaimsFor returns the claims that apply to (model, operation): claims naming
// the operation if there are any, else the model's whole-model claims.
func (s *Seed) ClaimsFor(modelID, operation string) []Claim {
	var specific, general []Claim
	for _, c := range s.Claims {
		if c.ModelID != modelID {
			continue
		}
		switch c.Operation {
		case operation:
			specific = append(specific, c)
		case AllOperations:
			general = append(general, c)
		}
	}
	if len(specific) > 0 {
		return specific
	}
	return general
}
