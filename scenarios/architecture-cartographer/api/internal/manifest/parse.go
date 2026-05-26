package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ContentType is the manifest source encoding. Detection falls back to
// inspecting the first non-whitespace byte when the caller leaves it
// blank.
type ContentType string

const (
	ContentTypeUnspecified ContentType = ""
	ContentTypeYAML        ContentType = "application/yaml"
	ContentTypeJSON        ContentType = "application/json"
)

// rawManifest mirrors ManifestDefinition with yaml/json tags so we can
// decode either encoding through the same path. Time/hash fields are
// populated by Parse, not by the source document.
type rawManifest struct {
	ManifestVersion string             `yaml:"manifest_version" json:"manifest_version"`
	Scenario        string             `yaml:"scenario"         json:"scenario"`
	Domains         []rawDomain        `yaml:"domains"          json:"domains"`
	SharedSubstrate []string           `yaml:"shared_substrate" json:"shared_substrate"`
	SignalWeights   map[string]float64 `yaml:"signal_weights"   json:"signal_weights"`
	Thresholds      []rawThreshold     `yaml:"thresholds"       json:"thresholds"`
	Transitional    []rawTransitional  `yaml:"transitional"     json:"transitional"`
}

type rawDomain struct {
	Name                  string             `yaml:"name"                    json:"name"`
	Paths                 []string           `yaml:"paths"                   json:"paths"`
	AllowedDependencies   []string           `yaml:"allowed_dependencies"    json:"allowed_dependencies"`
	Glossary              []string           `yaml:"glossary"                json:"glossary"`
	SignalWeightOverrides map[string]float64 `yaml:"signal_weight_overrides" json:"signal_weight_overrides"`
}

type rawThreshold struct {
	Tier     string  `yaml:"tier"      json:"tier"`
	MinValue float64 `yaml:"min_value" json:"min_value"`
}

type rawTransitional struct {
	ID          string `yaml:"id"           json:"id"`
	Kind        string `yaml:"kind"         json:"kind"`
	Locator     string `yaml:"locator"      json:"locator"`
	Rationale   string `yaml:"rationale"    json:"rationale"`
	ExpiresWhen string `yaml:"expires_when" json:"expires_when"`
}

// Parse decodes a manifest source. detectedType is the resolved
// content type; diagnostics carry parser-level findings (syntax
// errors, unknown version, etc.). When err is non-nil the manifest
// could not be decoded at all and diagnostics contains a single
// error-severity entry pinpointing the source location.
func Parse(source []byte, hint ContentType) (m ManifestDefinition, detectedType ContentType, diagnostics []Diagnostic, err error) {
	detectedType = resolveContentType(source, hint)

	var raw rawManifest
	switch detectedType {
	case ContentTypeJSON:
		dec := json.NewDecoder(bytes.NewReader(source))
		dec.DisallowUnknownFields()
		if e := dec.Decode(&raw); e != nil {
			d := Diagnostic{
				Severity: DiagnosticSeverityError,
				Path:     "",
				Message:  fmt.Sprintf("manifest JSON parse error: %s", e.Error()),
				Code:     "MANIFEST_PARSE_ERROR",
			}
			return ManifestDefinition{}, detectedType, []Diagnostic{d}, e
		}
	case ContentTypeYAML:
		dec := yaml.NewDecoder(bytes.NewReader(source))
		dec.KnownFields(true)
		if e := dec.Decode(&raw); e != nil {
			d := Diagnostic{
				Severity: DiagnosticSeverityError,
				Path:     "",
				Message:  fmt.Sprintf("manifest YAML parse error: %s", e.Error()),
				Code:     "MANIFEST_PARSE_ERROR",
				Line:     yamlErrorLine(e),
			}
			return ManifestDefinition{}, detectedType, []Diagnostic{d}, e
		}
	default:
		// Empty input is treated as a JSON object {} would be — but explicitly flag it.
		d := Diagnostic{
			Severity: DiagnosticSeverityError,
			Path:     "",
			Message:  "manifest source is empty",
			Code:     "MANIFEST_EMPTY_SOURCE",
		}
		return ManifestDefinition{}, detectedType, []Diagnostic{d}, fmt.Errorf("manifest source is empty")
	}

	m = ManifestDefinition{
		Version:         resolveVersion(raw.ManifestVersion, &diagnostics),
		Scenario:        raw.Scenario,
		SharedSubstrate: append([]string(nil), raw.SharedSubstrate...),
		SignalWeights:   SignalWeights{Weights: copyFloatMap(raw.SignalWeights)},
		ContentHash:     hashContent(source),
	}
	for _, d := range raw.Domains {
		m.Domains = append(m.Domains, DomainSpec{
			Name:                  d.Name,
			Paths:                 append([]string(nil), d.Paths...),
			AllowedDependencies:   append([]string(nil), d.AllowedDependencies...),
			Glossary:              append([]string(nil), d.Glossary...),
			SignalWeightOverrides: SignalWeights{Weights: copyFloatMap(d.SignalWeightOverrides)},
		})
	}
	for _, t := range raw.Thresholds {
		m.Thresholds = append(m.Thresholds, Threshold(t))
	}
	for _, t := range raw.Transitional {
		m.Transitional = append(m.Transitional, TransitionalDeclaration(t))
	}
	return m, detectedType, diagnostics, nil
}

func resolveContentType(source []byte, hint ContentType) ContentType {
	if hint != ContentTypeUnspecified {
		return hint
	}
	trimmed := bytes.TrimLeft(source, " \t\r\n")
	if len(trimmed) == 0 {
		return ContentTypeUnspecified
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return ContentTypeJSON
	}
	return ContentTypeYAML
}

func resolveVersion(raw string, diagnostics *[]Diagnostic) ManifestVersion {
	switch strings.TrimSpace(raw) {
	case "", "v1", "V1":
		return ManifestVersionV1
	default:
		*diagnostics = append(*diagnostics, Diagnostic{
			Severity: DiagnosticSeverityError,
			Path:     "manifest_version",
			Message:  fmt.Sprintf("unknown manifest_version %q (expected v1)", raw),
			Code:     "MANIFEST_UNKNOWN_VERSION",
		})
		return ManifestVersion(raw)
	}
}

func copyFloatMap(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func hashContent(source []byte) string {
	sum := sha256.Sum256(source)
	return hex.EncodeToString(sum[:])
}

// yamlErrorLine extracts the 1-based line number from a yaml.v3 typed
// error so diagnostics can carry source position. Returns 0 when the
// error doesn't expose location info.
func yamlErrorLine(err error) int {
	var te *yaml.TypeError
	if errorsAs(err, &te) && len(te.Errors) > 0 {
		// yaml.v3 prefixes line numbers as "line N: ..."; parse without
		// pulling in regexp.
		first := te.Errors[0]
		const prefix = "line "
		if i := strings.Index(first, prefix); i >= 0 {
			rest := first[i+len(prefix):]
			n := 0
			for _, r := range rest {
				if r < '0' || r > '9' {
					break
				}
				n = n*10 + int(r-'0')
			}
			return n
		}
	}
	return 0
}

// errorsAs is a tiny shim so we don't pull "errors" in just for one
// type-assertion in yamlErrorLine.
func errorsAs(err error, target any) bool {
	switch t := target.(type) {
	case **yaml.TypeError:
		te, ok := err.(*yaml.TypeError)
		if !ok {
			return false
		}
		*t = te
		return true
	}
	return false
}
