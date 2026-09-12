package research

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

// ResearchQuestion is an independently answerable part of a request.
type ResearchQuestion struct {
	ID       string
	Prompt   string
	Required bool
}

// AssessmentDisposition deliberately has an explicit unknown value. Unknown
// evidence is never promoted to supported by a structural conversion.
type AssessmentDisposition string

const (
	AssessmentSupported    AssessmentDisposition = "supported"
	AssessmentContradicted AssessmentDisposition = "contradicted"
	AssessmentUnresolved   AssessmentDisposition = "unresolved"
	AssessmentUnknown      AssessmentDisposition = "unknown"
)

type EvidencePassageRef struct {
	ReceiptID          string
	PassageID          string
	ContentHash        string
	ExtractionRevision string
}

type ClaimAssessment struct {
	AssessmentID   string
	ClaimID        string
	Disposition    AssessmentDisposition
	Evidence       []EvidencePassageRef
	PolicyRevision string
	Reason         string
}

type QuestionCoverage struct {
	QuestionID       string
	Status           string
	ClaimIDs         []string
	UnresolvedReason string
}

// EvaluateQuestionCoverage turns independently assessed claims into a
// question-level result. A question with no supported claim is never marked
// supported, even when another question in the same request is supported.
func EvaluateQuestionCoverage(questions []ResearchQuestion, assessments []ClaimAssessment, claimsByQuestion map[string][]string) []QuestionCoverage {
	byClaim := make(map[string]AssessmentDisposition, len(assessments))
	for _, assessment := range assessments {
		byClaim[assessment.ClaimID] = assessment.Disposition
	}
	out := make([]QuestionCoverage, 0, len(questions))
	for _, question := range questions {
		claimIDs := append([]string(nil), claimsByQuestion[question.ID]...)
		coverage := QuestionCoverage{QuestionID: question.ID, ClaimIDs: claimIDs, Status: "unknown"}
		if len(claimIDs) == 0 {
			coverage.Status = "unresolved"
			coverage.UnresolvedReason = "no_claim_assigned"
		} else {
			unknown := false
			for _, id := range claimIDs {
				switch byClaim[id] {
				case AssessmentSupported:
					coverage.Status = "supported"
				case AssessmentContradicted, AssessmentUnresolved:
					if coverage.Status != "supported" {
						coverage.Status = "unresolved"
					}
					coverage.UnresolvedReason = "claim_not_supported"
				default:
					unknown = true
				}
			}
			if coverage.Status == "unknown" && unknown {
				coverage.UnresolvedReason = "assessment_unknown"
			}
		}
		out = append(out, coverage)
	}
	return out
}

// ValidateContractBounds validates the shared request contract before any
// network or model work. The limits are intentionally small and stable so
// direct calls and delegated calls have the same admission behavior.
func ValidateContractBounds(query string, questions []ResearchQuestion, p EvidencePolicy) error {
	if strings.TrimSpace(query) == "" || len(query) > 4096 {
		return fmt.Errorf("query must contain 1..4096 bytes")
	}
	if len(questions) > 20 {
		return fmt.Errorf("at most 20 research questions are allowed")
	}
	seen := make(map[string]struct{}, len(questions))
	for _, q := range questions {
		if strings.TrimSpace(q.Prompt) == "" || len(q.Prompt) > 2048 {
			return fmt.Errorf("research question prompt must contain 1..2048 bytes")
		}
		if len(q.ID) > 128 || (q.ID != "" && !isContractToken(q.ID)) {
			return fmt.Errorf("research question id is invalid")
		}
		if q.ID != "" {
			if _, ok := seen[q.ID]; ok {
				return fmt.Errorf("duplicate research question id %q", q.ID)
			}
			seen[q.ID] = struct{}{}
		}
	}
	if p.MaxEvidenceBytes < 0 || p.MaxEvidenceBytes > 4<<20 {
		return fmt.Errorf("max evidence bytes must be within 0..4194304")
	}
	return nil
}

func isContractToken(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return s != ""
}

func (d AssessmentDisposition) Supported() bool { return d == AssessmentSupported }

func (d AssessmentDisposition) Valid() bool {
	switch d {
	case AssessmentSupported, AssessmentContradicted, AssessmentUnresolved, AssessmentUnknown:
		return true
	default:
		return false
	}
}

// AssessClaimSupport applies the bounded, inspectable support rule used by
// recorded evaluations. A citation reference without retained passage text
// is unknown, never supported. The rule is intentionally conservative: every
// normalized claim term must occur in retained evidence, so an unsupported
// sentence cannot be laundered by a supported sentence in the same summary.
func AssessClaimSupport(claim string, passages []string) AssessmentDisposition {
	claimTerms := contractTerms(claim)
	if len(claimTerms) == 0 || len(passages) == 0 {
		return AssessmentUnknown
	}
	evidence := make(map[string]struct{})
	for _, passage := range passages {
		for term := range contractTerms(passage) {
			evidence[term] = struct{}{}
		}
	}
	matched := 0
	for term := range claimTerms {
		if _, ok := evidence[term]; ok {
			matched++
		}
	}
	if matched == len(claimTerms) {
		return AssessmentSupported
	}
	return AssessmentUnresolved
}

func contractTerms(s string) map[string]struct{} {
	out := map[string]struct{}{}
	var word []rune
	flush := func() {
		if len(word) >= 3 {
			out[strings.ToLower(string(word))] = struct{}{}
		}
		word = word[:0]
	}
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word = append(word, r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

// IndependentPublisherCount counts publisher registrable domains rather than
// hostnames. Subdomains of one publisher therefore cannot manufacture source
// independence; malformed or non-web URLs contribute nothing.
func IndependentPublisherCount(rawURLs []string) int {
	seen := map[string]struct{}{}
	for _, raw := range rawURLs {
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
			continue
		}
		parts := strings.Split(strings.ToLower(strings.TrimSuffix(u.Hostname(), ".")), ".")
		if len(parts) < 2 {
			continue
		}
		seen[strings.Join(parts[len(parts)-2:], ".")] = struct{}{}
	}
	return len(seen)
}
