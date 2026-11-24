// Package credentialleak provides a bounded scanner for synthetic credential
// sentinels. It reports only sentinel labels, channels, and encoding classes.
package credentialleak

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html"
	"net/url"
	"strings"
	"unicode"
)

const (
	maxObservationBytes = 1 << 20
	fragmentWidth       = 20
	fragmentStep        = 1
)

var ErrObservationTooLarge = errors.New("credential leak observation exceeds scanner bound")

// Family is safe inventory metadata. Name must be stable, and Field/Scope are
// references only; values never belong here.
type Family struct {
	Name     string `json:"name"`
	Identity string `json:"identity"`
	Field    string `json:"field"`
	Scope    string `json:"scope"`
}

// Canary is synthetic test data. Do not serialize Value into evidence.
type Canary struct {
	Family string
	Value  string
}

type Observation struct {
	Channel string
	Data    []byte
}

type Finding struct {
	Family         string `json:"family"`
	Channel        string `json:"channel"`
	Representation string `json:"representation"`
}

type Scanner struct {
	needles []needle
}

type needle struct {
	family string
	value  string
	form   string
}

// SyntheticCanaries produces deterministic, obviously synthetic, high-entropy
// values from family labels. The values are reproducible for artifact scans,
// but remain absent from receipts and scanner findings.
func SyntheticCanaries(families []Family) []Canary {
	canaries := make([]Canary, 0, len(families))
	for _, family := range families {
		seed := sha256.Sum256([]byte("PY1-UNIT2-CREDENTIAL-LEAK-v1/" + family.Name))
		digest := hex.EncodeToString(seed[:])
		canaries = append(canaries, Canary{Family: family.Name, Value: "SYNTHETIC-" + digest[:12] + "-LPBS-" + digest[12:]})
	}
	return canaries
}

func NewScanner(canaries []Canary) (*Scanner, error) {
	if len(canaries) == 0 {
		return nil, errors.New("credential leak scanner requires at least one synthetic canary")
	}
	scanner := &Scanner{}
	seenFamilies := make(map[string]struct{}, len(canaries))
	for _, canary := range canaries {
		if strings.TrimSpace(canary.Family) == "" || len(canary.Value) < 32 {
			return nil, errors.New("credential leak scanner received invalid synthetic canary metadata")
		}
		if _, exists := seenFamilies[canary.Family]; exists {
			return nil, errors.New("credential leak scanner received duplicate family metadata")
		}
		seenFamilies[canary.Family] = struct{}{}
		for _, form := range representations(canary.Value) {
			for _, needleValue := range fragments(form) {
				scanner.needles = append(scanner.needles, needle{family: canary.Family, value: needleValue, form: form.name})
			}
		}
	}
	return scanner, nil
}

// Scan returns metadata-only findings. It fails closed on oversized input so
// an artifact cannot be declared clean after silent truncation.
func (s *Scanner) Scan(observation Observation) ([]Finding, error) {
	if s == nil || len(s.needles) == 0 {
		return nil, errors.New("credential leak scanner is not initialized")
	}
	if strings.TrimSpace(observation.Channel) == "" {
		return nil, errors.New("credential leak observation requires a channel")
	}
	if len(observation.Data) > maxObservationBytes {
		return nil, ErrObservationTooLarge
	}
	forms := searchableForms(string(observation.Data))
	found := make(map[string]Finding)
	for _, form := range forms {
		for _, candidate := range s.needles {
			if strings.Contains(form.value, candidate.value) {
				key := candidate.family + "\x00" + observation.Channel
				found[key] = Finding{Family: candidate.family, Channel: observation.Channel, Representation: candidate.form}
			}
		}
	}
	result := make([]Finding, 0, len(found))
	for _, finding := range found {
		result = append(result, finding)
	}
	return result, nil
}

// RedactForEvidence replaces an entire observation if a sentinel is found.
// It does not attempt partial masking, which could leave an encoded fragment.
func (s *Scanner) RedactForEvidence(observation Observation) ([]byte, []Finding, error) {
	findings, err := s.Scan(observation)
	if err != nil {
		return nil, nil, err
	}
	if len(findings) == 0 {
		return append([]byte(nil), observation.Data...), nil, nil
	}
	return []byte("[redacted: synthetic credential sentinel detected]"), findings, nil
}

type representation struct{ name, value string }

func representations(value string) []representation {
	jsonValue, _ := json.Marshal(value)
	query := url.QueryEscape(value)
	path := url.PathEscape(value)
	forms := []representation{
		{name: "raw", value: value},
		{name: "base64", value: base64.StdEncoding.EncodeToString([]byte(value))},
		{name: "base64-raw", value: base64.RawStdEncoding.EncodeToString([]byte(value))},
		{name: "base64-url", value: base64.URLEncoding.EncodeToString([]byte(value))},
		{name: "hex", value: hex.EncodeToString([]byte(value))},
		{name: "url-query", value: query},
		{name: "url-path", value: path},
		{name: "json", value: strings.Trim(string(jsonValue), `"`)},
		{name: "html", value: html.EscapeString(value)},
	}
	for _, form := range append([]representation(nil), forms...) {
		forms = append(forms,
			representation{name: "case-folded", value: strings.ToLower(form.value)},
			representation{name: "punctuation-normalized", value: normalizeAlphanumeric(form.value)},
		)
	}
	return forms
}

func fragments(form representation) []string {
	value := form.value
	if len(value) < fragmentWidth {
		return []string{value}
	}
	fragments := []string{value, value[:fragmentWidth], value[len(value)-fragmentWidth:]}
	for start := 0; start+fragmentWidth <= len(value); start += fragmentStep {
		fragments = append(fragments, value[start:start+fragmentWidth])
	}
	return fragments
}

func searchableForms(data string) []representation {
	withoutSpace := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, data)
	alnum := normalizeAlphanumeric(data)
	return []representation{
		{name: "raw", value: data},
		{name: "case-folded", value: strings.ToLower(data)},
		{name: "whitespace-collapsed", value: withoutSpace},
		{name: "punctuation-normalized", value: alnum},
	}
}

func normalizeAlphanumeric(data string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.ToLower(data))
}
