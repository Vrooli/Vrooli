package main

import "testing"

// Named fixtures shared across the detector and HTTP tests so the intent of each
// row is readable and the literal is defined once. Their invalid twins carry the
// reason they must not match.
const (
	validEmail      = "bob@example.com"
	validEmailPlus  = "alice.smith+tag@example.co.uk"
	validSSN        = "123-45-6789"
	validVisa       = "4111 1111 1111 1111" // passes Luhn
	validVisaNoSep  = "4111111111111111"
	validAmex       = "340000000000009"     // 15-digit Amex test number
	card13          = "4222222222222"       // 13-digit Visa test (lower Luhn boundary)
	card19          = "4111111111111111110" // 19-digit (upper Luhn boundary)
	cardDashed      = "4111-1111-1111-1111" // dashed separators, same Visa
	validIBAN       = "GB82WEST12345698765432"
	validIBANSpaced = "GB82 WEST 1234 5698 7654 32" // same IBAN, human grouping

	nonLuhn16  = "1234567812345678"       // 16 digits, fails Luhn → not a card
	badIBAN    = "GB82WEST12345698765433" // bad check digit
	ssnArea000 = "000-45-6789"            // structurally invalid SSN area
	longDigits = "123456789012345"        // 15-digit id → must NOT be a phone
)

func TestValidateLuhn(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{validVisaNoSep, true},
		{validVisa, true},
		{"5500005555555559", true}, // Mastercard test number
		{validAmex, true},          // Amex test number (15 digits)
		{"4111111111111112", false},
		{"1234567812345670", true}, // constructed valid Luhn
		{nonLuhn16, false},
		{"", false},
		{"42", false}, // too short
	}
	for _, c := range cases {
		if got := validateLuhn(c.in); got != c.want {
			t.Errorf("validateLuhn(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestValidateIBAN(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{validIBAN, true},
		{validIBANSpaced, true}, // validator strips spaces
		{"DE89370400440532013000", true},
		{"FR1420041010050500013M02606", true},
		{badIBAN, false},
		{"GB00WEST12345698765432", false}, // bad check digits
		{"XX", false},                     // too short
	}
	for _, c := range cases {
		if got := validateIBAN(c.in); got != c.want {
			t.Errorf("validateIBAN(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestValidateSSN(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{validSSN, true},
		{ssnArea000, false},    // area 000
		{"666-45-6789", false}, // area 666
		{"900-45-6789", false}, // area 9xx
		{"123-00-6789", false}, // group 00
		{"123-45-0000", false}, // serial 0000
	}
	for _, c := range cases {
		// validateSSN strips non-digits itself, so the dashed fixture is passed
		// through directly.
		if got := validateSSN(c.in); got != c.want {
			t.Errorf("validateSSN(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestRegexDetectMatrix is the single positive/negative matrix for the regex
// pass. Each row names the text, the category under test, and whether a span of
// that category is expected; wantText (when set) pins the exact matched
// substring so span-truncation and boundary bugs are caught. This replaces the
// five per-category hand-written funcs and folds in the P1-2/P1-3 regressions.
func TestRegexDetectMatrix(t *testing.T) {
	cases := []struct {
		name     string
		text     string
		cat      string
		want     bool
		wantText string // optional exact-span assertion
	}{
		// Email
		{"email plus-subaddr multi-label TLD", "Contact " + validEmailPlus + " for details.", CategoryEmail, true, validEmailPlus},
		{"email trailing period stays out of span", "write " + validEmail + ".", CategoryEmail, true, validEmail},
		{"email in sentence", "mail me at " + validEmail + " now", CategoryEmail, true, validEmail},
		{"email missing @ not flagged", "alice at example dot com", CategoryEmail, false, ""},
		{"email missing TLD not flagged", "a@b", CategoryEmail, false, ""},
		{"email bare @ not flagged", "@nope", CategoryEmail, false, ""},

		// Credit card (Luhn-gated) — boundary lengths and separators.
		{"card spaced groups", "card " + validVisa + " on file", CategoryCreditCard, true, validVisa},
		{"card amex 15 digits", "amex " + validAmex + " charged", CategoryCreditCard, true, validAmex},
		{"card 13-digit boundary", "card " + card13 + " ok", CategoryCreditCard, true, card13},
		{"card 19-digit boundary", "card " + card19 + " ok", CategoryCreditCard, true, card19},
		{"card dashed separators", "charge " + cardDashed + " now", CategoryCreditCard, true, cardDashed},
		{"card non-Luhn 16 digits not flagged", "order " + nonLuhn16 + " shipped", CategoryCreditCard, false, ""},

		// IBAN (mod-97 gated) — the spaced form is the P1-3 regression.
		{"iban compact", "pay to " + validIBAN + " please", CategoryIBAN, true, validIBAN},
		{"iban spaced human grouping", "pay to " + validIBANSpaced + " please", CategoryIBAN, true, validIBANSpaced},
		{"iban bad check digit not flagged", "ref " + badIBAN + " invalid", CategoryIBAN, false, ""},

		// SSN (dash-structured, validated)
		{"ssn dashed", "SSN " + validSSN + " on form", CategorySSN, true, validSSN},
		{"ssn invalid area not flagged", "id " + ssnArea000, CategorySSN, false, ""},
		{"ssn no-dash run not matched", "num 123456789 here", CategorySSN, false, ""},

		// Phone — the long-digit-run negative is the P1-2 regression.
		{"phone 10-digit", "call 5551234567 now", CategoryPhone, true, "5551234567"},
		{"phone long numeric id not flagged", "ref " + longDigits + " in system", CategoryPhone, false, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spans := RegexDetect(c.text)
			if got := hasCategory(spans, c.cat, c.wantText); got != c.want {
				t.Errorf("text %q: category %s present=%v want=%v (spans=%+v)", c.text, c.cat, got, c.want, spans)
			}
		})
	}
}

// TestRegexDetectMultipleMatchesInOneString locks that the regex pass returns
// every occurrence, not just the first (the matrix only asserts presence).
func TestRegexDetectMultipleMatchesInOneString(t *testing.T) {
	text := "cards " + validVisa + " and " + validAmex + " on file"
	n := 0
	for _, s := range RegexDetect(text) {
		if s.Category == CategoryCreditCard {
			n++
		}
	}
	if n != 2 {
		t.Errorf("expected 2 credit-card spans, got %d (%+v)", n, RegexDetect(text))
	}

	emails := "to " + validEmail + " and " + validEmailPlus
	m := 0
	for _, s := range RegexDetect(emails) {
		if s.Category == CategoryEmail {
			m++
		}
	}
	if m != 2 {
		t.Errorf("expected 2 email spans, got %d", m)
	}
}

// TestRegexSpanOffsetsMatchText asserts byte offsets slice back to the matched
// text, including after a multibyte (UTF-8) prefix so rune/byte confusion in the
// offsets would surface.
func TestRegexSpanOffsetsMatchText(t *testing.T) {
	for _, text := range []string{
		"mail me at " + validEmail + " now",
		"naïve café " + validSSN + " and " + validEmail, // multibyte prefix
	} {
		spans := RegexDetect(text)
		if len(spans) == 0 {
			t.Fatalf("expected at least one span in %q", text)
		}
		for _, s := range spans {
			if text[s.Start:s.End] != s.Text {
				t.Errorf("span offsets %d:%d do not slice to %q (got %q) in %q", s.Start, s.End, s.Text, text[s.Start:s.End], text)
			}
		}
	}
}

func hasCategory(spans []Span, cat, text string) bool {
	for _, s := range spans {
		if s.Category == cat && (text == "" || s.Text == text) {
			return true
		}
	}
	return false
}
