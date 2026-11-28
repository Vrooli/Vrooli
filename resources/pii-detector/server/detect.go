package main

// detect.go holds the deterministic regex + checksum ("Presidio-style") pass.
//
// This pass is cheap, deterministic, and high-precision on *structured*
// identifiers (email, phone, SSN, credit-card, IBAN). It runs independently of
// the NER model and its spans are merged with the model's spans (see merge.go).
// Keeping it separate from the model means a host with no model artifact still
// gets reliable structured-PII detection.

import (
	"regexp"
	"strings"
)

// Span is one detected PII occurrence over the request text. Offsets are byte
// offsets into the original UTF-8 text (start inclusive, end exclusive) so a
// caller can mask/redact the exact substring without re-tokenizing.
type Span struct {
	Category   string  `json:"category"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	// Source records which pass produced the span ("regex" or "model"); it is
	// informational and lets the merge logic prefer one source deterministically.
	Source string `json:"source,omitempty"`
}

// Category constants are Presidio-aligned names, shared with the model-policy
// category vocabulary so downstream redaction can key off a stable set.
const (
	CategoryEmail      = "EMAIL_ADDRESS"
	CategoryPhone      = "PHONE_NUMBER"
	CategorySSN        = "US_SSN"
	CategoryCreditCard = "CREDIT_CARD"
	CategoryIBAN       = "IBAN_CODE"
)

// regexRule pairs a compiled pattern with its category and an optional
// validator (a checksum/structural test). A rule with a validator only emits a
// span when the validator passes, which is what keeps credit-card and IBAN
// precision high (a 16-digit number that fails Luhn is not a card).
type regexRule struct {
	category  string
	pattern   *regexp.Regexp
	validate  func(match string) bool
	baseScore float64
}

// regexRules is the ordered deterministic rule set. Order is not significant to
// correctness because spans are de-overlapped in merge.go, but email/SSN are
// listed before the looser phone rule so that, on a tie, the more specific
// category is preferred by the stable sort there.
var regexRules = []regexRule{
	{
		category:  CategoryEmail,
		pattern:   regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		baseScore: 0.99,
	},
	{
		category:  CategorySSN,
		pattern:   regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		validate:  validateSSN,
		baseScore: 0.95,
	},
	{
		category:  CategoryCreditCard,
		pattern:   regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`),
		validate:  validateLuhn,
		baseScore: 0.95,
	},
	{
		category: CategoryIBAN,
		// Country code + 2 check digits + 11–30 BBAN chars, allowing the single
		// spaces humans use to group IBANs in fours (e.g. "GB82 WEST 1234 …").
		// validateIBAN strips the spaces before the mod-97 checksum.
		pattern:   regexp.MustCompile(`\b[A-Z]{2}\d{2}(?: ?[A-Z0-9]){11,30}\b`),
		validate:  validateIBAN,
		baseScore: 0.95,
	},
	{
		category: CategoryPhone,
		// \b anchors stop the loose pattern from grabbing a prefix of a longer
		// pure-digit run (an order id / timestamp), which both false-positives and
		// truncates the span. A real phone number is bounded by non-word chars.
		pattern:   regexp.MustCompile(`\b(?:\+?\d{1,3}[ .\-]?)?(?:\(\d{2,4}\)[ .\-]?)?\d{3}[ .\-]?\d{3,4}(?:[ .\-]?\d{2,4})?\b`),
		validate:  validatePhone,
		baseScore: 0.75,
	},
}

// RegexDetect runs the deterministic pass over text and returns all validated
// structured-PII spans. It never returns overlapping spans within a single
// rule, but spans from different rules may overlap; overlap resolution is
// merge.go's job.
func RegexDetect(text string) []Span {
	var spans []Span
	for _, rule := range regexRules {
		for _, loc := range rule.pattern.FindAllStringIndex(text, -1) {
			match := text[loc[0]:loc[1]]
			if rule.validate != nil && !rule.validate(match) {
				continue
			}
			spans = append(spans, Span{
				Category:   rule.category,
				Start:      loc[0],
				End:        loc[1],
				Text:       match,
				Confidence: rule.baseScore,
				Source:     "regex",
			})
		}
	}
	return spans
}

// digitsOnly strips every non-digit rune. Used by the numeric validators so
// that spacing/dashes in the matched text do not affect the checksum.
func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteByte(byte(r))
		}
	}
	return b.String()
}

// validateLuhn implements the Luhn mod-10 checksum used by payment cards.
func validateLuhn(match string) bool {
	digits := digitsOnly(match)
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	sum := 0
	double := false
	for i := len(digits) - 1; i >= 0; i-- {
		d := int(digits[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

// validateIBAN implements the ISO 13616 mod-97 check. It moves the first four
// characters to the end, maps letters to two-digit numbers (A=10 … Z=35), and
// checks the resulting integer is congruent to 1 mod 97.
func validateIBAN(match string) bool {
	s := strings.ToUpper(strings.ReplaceAll(match, " ", ""))
	if len(s) < 15 || len(s) > 34 {
		return false
	}
	// Move first four chars to the end.
	rearranged := s[4:] + s[:4]
	remainder := 0
	for _, r := range rearranged {
		var val int
		switch {
		case r >= '0' && r <= '9':
			val = int(r - '0')
		case r >= 'A' && r <= 'Z':
			val = int(r-'A') + 10
		default:
			return false
		}
		// Fold the (1- or 2-digit) value into the running mod-97 remainder.
		if val >= 10 {
			remainder = (remainder*100 + val) % 97
		} else {
			remainder = (remainder*10 + val) % 97
		}
	}
	return remainder == 1
}

// validateSSN rejects the structurally-invalid US SSN groups (area 000/666/9xx,
// group 00, serial 0000) so a random 3-2-4 number is not flagged.
func validateSSN(match string) bool {
	digits := digitsOnly(match)
	if len(digits) != 9 {
		return false
	}
	area := digits[0:3]
	group := digits[3:5]
	serial := digits[5:9]
	if area == "000" || area == "666" || area[0] == '9' {
		return false
	}
	if group == "00" || serial == "0000" {
		return false
	}
	return true
}

// validatePhone requires at least 10 digits so the loose phone pattern does not
// flag short numeric runs. A credit-card-length run that passes Luhn is better
// served by the card rule; the merge step prefers the more specific category.
func validatePhone(match string) bool {
	digits := digitsOnly(match)
	return len(digits) >= 10 && len(digits) <= 15
}
