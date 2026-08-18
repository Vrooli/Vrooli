package binaryfetch

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

func isDigest(value string) bool {
	if len(strings.TrimSpace(value)) != 64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimSpace(value))
	return err == nil
}

// Facts is the only input accepted by acquisition target selection. Keys are
// open-ended so adding a host capability does not require changing this
// contract or manifests that do not use the capability.
type Facts map[string]string

// BuildTimeFacts is the fact set available while producing a release. A
// release knows its selected operating system and architecture, but cannot
// know the GPU of the machine that will eventually run it.
var BuildTimeFacts = map[string]struct{}{
	"os":   {},
	"arch": {},
}

// Acquisition declares how an executable or executable tree arrives on disk.
// It is deliberately independent from the artifact launch gate: target SHA256
// authenticates downloaded bytes, while the artifact declaration authenticates
// the bytes that will be executed.
type Acquisition struct {
	Kind       string              `json:"kind"`
	License    string              `json:"license,omitempty"`
	Targets    []AcquisitionTarget `json:"targets"`
	Provenance *Provenance         `json:"provenance,omitempty"`
}

type ComposeStep struct {
	Role     string `json:"role"`
	Kind     string `json:"kind"`
	Dest     string `json:"dest"`
	URL      string `json:"url,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Archive  string `json:"archive,omitempty"`
	BinPath  string `json:"bin_path,omitempty"`
	Mode     string `json:"mode,omitempty"`
	Lockfile string `json:"lockfile,omitempty"`
}

// AcquisitionTarget is one ordered candidate source. The first target whose
// predicate matches wins. An unsupported target is useful as a terminal,
// explicit declaration for a platform the item does not serve.
type AcquisitionTarget struct {
	When           map[string]string `json:"when,omitempty"`
	URL            string            `json:"url,omitempty"`
	Image          string            `json:"image,omitempty"`
	SHA256         string            `json:"sha256,omitempty"`
	ArtifactSHA256 string            `json:"artifact_sha256,omitempty"`
	Archive        string            `json:"archive,omitempty"`
	Layout         string            `json:"layout,omitempty"`
	BinPath        string            `json:"bin_path,omitempty"`
	Mode           string            `json:"mode,omitempty"`
	// Executable is the host command adopted by kind=none. It is deliberately
	// explicit: a managed service may reuse a separately governed host tool,
	// but it may not discover an arbitrary running process or binary.
	Executable  string            `json:"executable,omitempty"`
	RuntimeEnv  map[string]string `json:"runtime_env,omitempty"`
	Unsupported string            `json:"unsupported,omitempty"`
	Compose     []ComposeStep     `json:"compose,omitempty"`
}

// IsDir reports whether the acquired artifact is an executable tree.
func (t AcquisitionTarget) IsDir() bool { return strings.EqualFold(strings.TrimSpace(t.Layout), "dir") }

// Provenance records an optional upstream trust mechanism. kind=none relies
// on the reviewed digest. kind=gpg-checksums verifies a signed checksum file
// when the optional runtime verifier is available.
type Provenance struct {
	Kind              string `json:"kind"`
	KeyURL            string `json:"key_url,omitempty"`
	ChecksumManifest  string `json:"checksum_manifest_url,omitempty"`
	ChecksumSignature string `json:"checksum_signature_url,omitempty"`
	Fingerprint       string `json:"fingerprint,omitempty"`
}

// Validate checks the declaration before it is persisted in a manifest. It
// deliberately does not resolve a target: resolution depends on host facts
// that are unavailable while validating a release or resource manifest.
func (a Acquisition) Validate() error {
	kind := strings.ToLower(strings.TrimSpace(a.Kind))
	switch kind {
	case "url", "oci-image", "none", "composed":
	default:
		return fmt.Errorf("acquisition kind %q is invalid", a.Kind)
	}
	if len(a.Targets) == 0 {
		return errors.New("acquisition targets must not be empty")
	}
	for index, target := range a.Targets {
		if err := validateTargetPredicate(target.When); err != nil {
			return fmt.Errorf("acquisition target %d: %w", index, err)
		}
		unsupported := strings.TrimSpace(target.Unsupported)
		if unsupported != "" {
			if target.URL != "" || target.Image != "" || target.SHA256 != "" || target.ArtifactSHA256 != "" || target.Archive != "" || target.Layout != "" || target.BinPath != "" || target.Executable != "" || len(target.RuntimeEnv) != 0 || len(target.Compose) != 0 {
				return fmt.Errorf("acquisition target %d: unsupported target cannot also declare an artifact", index)
			}
			continue
		}
		switch kind {
		case "url":
			if strings.TrimSpace(target.URL) == "" {
				return fmt.Errorf("acquisition target %d: url is required", index)
			}
			parsed, err := url.Parse(target.URL)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				return fmt.Errorf("acquisition target %d: url must be absolute", index)
			}
			if len(strings.TrimSpace(target.SHA256)) != 64 {
				return fmt.Errorf("acquisition target %d: sha256 must be a 64-character digest", index)
			}
		case "oci-image":
			if strings.TrimSpace(target.Image) == "" {
				return fmt.Errorf("acquisition target %d: image is required", index)
			}
		case "none":
			if strings.TrimSpace(target.Executable) == "" {
				return fmt.Errorf("acquisition target %d: kind none requires executable or unsupported", index)
			}
			if filepath.IsAbs(target.Executable) || filepath.Clean(target.Executable) != target.Executable || strings.ContainsAny(target.Executable, `/\\`) {
				return fmt.Errorf("acquisition target %d: executable must be a PATH command name", index)
			}
		case "composed":
			if target.Layout != "dir" {
				return fmt.Errorf("acquisition target %d: composed acquisition requires layout dir", index)
			}
			if !isDigest(target.ArtifactSHA256) {
				return fmt.Errorf("acquisition target %d: composed acquisition requires artifact_sha256", index)
			}
			if len(target.Compose) == 0 {
				return fmt.Errorf("acquisition target %d: composed acquisition requires compose steps", index)
			}
			for stepIndex, step := range target.Compose {
				if err := step.Validate(); err != nil {
					return fmt.Errorf("acquisition target %d compose step %d: %w", index, stepIndex, err)
				}
			}
		}
		if target.Archive != "" && target.Archive != "tar.gz" && target.Archive != "tar.bz2" && target.Archive != "tar.zst" && target.Archive != "zip" && target.Archive != "none" {
			return fmt.Errorf("acquisition target %d: archive %q is invalid", index, target.Archive)
		}
		if target.Layout != "" && target.Layout != "file" && target.Layout != "dir" {
			return fmt.Errorf("acquisition target %d: layout %q is invalid", index, target.Layout)
		}
		if target.ArtifactSHA256 != "" && !isDigest(target.ArtifactSHA256) {
			return fmt.Errorf("acquisition target %d: artifact_sha256 must be a 64-character digest", index)
		}
	}
	if a.Provenance != nil {
		if a.Provenance.Kind != "none" && a.Provenance.Kind != "gpg-checksums" {
			return fmt.Errorf("acquisition provenance kind %q is invalid", a.Provenance.Kind)
		}
		if a.Provenance.Kind == "gpg-checksums" {
			for label, value := range map[string]string{"key_url": a.Provenance.KeyURL, "checksum_manifest_url": a.Provenance.ChecksumManifest, "checksum_signature_url": a.Provenance.ChecksumSignature} {
				parsed, err := url.Parse(strings.TrimSpace(value))
				if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
					return fmt.Errorf("acquisition provenance %s must be an absolute HTTPS URL", label)
				}
			}
			if strings.TrimSpace(a.Provenance.Fingerprint) == "" {
				return errors.New("acquisition gpg-checksums provenance fingerprint is required")
			}
		}
	}
	return nil
}

func (s ComposeStep) Validate() error {
	if strings.TrimSpace(s.Role) == "" || strings.TrimSpace(s.Kind) == "" || strings.TrimSpace(s.Dest) == "" {
		return errors.New("compose step requires role, kind, and dest")
	}
	if filepath.IsAbs(s.Dest) || strings.HasPrefix(filepath.Clean(s.Dest), "..") {
		return fmt.Errorf("compose step dest %q must stay under the artifact root", s.Dest)
	}
	switch strings.ToLower(strings.TrimSpace(s.Kind)) {
	case "url":
		parsed, err := url.Parse(s.URL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return errors.New("url compose step requires an absolute URL")
		}
		if !isDigest(s.SHA256) {
			return errors.New("url compose step requires sha256")
		}
		if s.Archive != "tar.gz" && s.Archive != "tar.bz2" && s.Archive != "tar.zst" && s.Archive != "zip" {
			return fmt.Errorf("url compose step archive %q is invalid", s.Archive)
		}
	case "python-wheels":
		if strings.TrimSpace(s.Lockfile) == "" {
			return errors.New("python-wheels compose step requires lockfile")
		}
	default:
		return fmt.Errorf("compose step kind %q is invalid", s.Kind)
	}
	return nil
}

func validateTargetPredicate(predicate map[string]string) error {
	for name, requirement := range predicate {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(requirement) == "" {
			return errors.New("target predicates must have non-empty names and values")
		}
	}
	return nil
}

// UnsupportedError reports the explicit reason declared by a matching target.
type UnsupportedError struct{ Reason string }

func (e *UnsupportedError) Error() string { return "binaryfetch: unsupported: " + e.Reason }

// NoMatchingTargetError reports that no ordered candidate matched the host
// facts. Reasons contains the target-specific rejection explanations when
// available, which is useful to both operators and acquisition explain.
type NoMatchingTargetError struct{ Reasons []string }

func (e *NoMatchingTargetError) Error() string {
	if len(e.Reasons) == 0 {
		return "binaryfetch: no acquisition target matches host facts"
	}
	return "binaryfetch: no acquisition target matches host facts: " + strings.Join(e.Reasons, "; ")
}

// CandidateExplanation is the operator-facing result for one target.
type CandidateExplanation struct {
	Index    int
	When     map[string]string
	Matched  bool
	Reason   string
	Selected bool
}

// ResolutionExplanation describes every candidate and the final selection.
type ResolutionExplanation struct {
	Facts      Facts
	Selected   int
	Candidates []CandidateExplanation
}

// Resolve returns the first target whose predicate matches facts. A target
// with an Unsupported reason still wins selection and returns an
// UnsupportedError so callers cannot silently fall through to a less precise
// declaration.
func (a Acquisition) Resolve(facts Facts) (AcquisitionTarget, error) {
	for _, target := range a.Targets {
		matched, err := predicateMatches(target.When, facts)
		if err != nil {
			return AcquisitionTarget{}, err
		}
		if !matched {
			continue
		}
		if reason := strings.TrimSpace(target.Unsupported); reason != "" {
			return target, &UnsupportedError{Reason: reason}
		}
		return target, nil
	}
	return AcquisitionTarget{}, &NoMatchingTargetError{}
}

// Explain evaluates every target in order and records why each candidate was
// rejected. It intentionally never hides an explicit unsupported candidate.
func (a Acquisition) Explain(facts Facts) ResolutionExplanation {
	result := ResolutionExplanation{Facts: cloneFacts(facts), Selected: -1}
	for i, target := range a.Targets {
		matched, err := predicateMatches(target.When, facts)
		candidate := CandidateExplanation{Index: i, When: cloneFacts(target.When), Matched: matched}
		switch {
		case err != nil:
			candidate.Reason = err.Error()
		case matched && strings.TrimSpace(target.Unsupported) != "":
			candidate.Reason = "matched: unsupported — " + strings.TrimSpace(target.Unsupported)
			candidate.Selected = result.Selected < 0
		case matched:
			candidate.Reason = "matched"
			candidate.Selected = result.Selected < 0
		default:
			candidate.Reason = predicateRejectionReason(target.When, facts)
		}
		if candidate.Selected {
			result.Selected = i
		}
		result.Candidates = append(result.Candidates, candidate)
	}
	return result
}

// UsesOnlyBuildTimeFacts reports whether a target can be resolved while
// staging a deterministic release.
func UsesOnlyBuildTimeFacts(target AcquisitionTarget) bool {
	for name := range target.When {
		if _, ok := BuildTimeFacts[name]; !ok {
			return false
		}
	}
	return true
}

func predicateMatches(predicate map[string]string, facts Facts) (bool, error) {
	for name, requirement := range predicate {
		actual, ok := facts[name]
		if !ok {
			return false, nil
		}
		matched, err := compareFact(normalizeFact(name, actual), normalizeFact(name, requirement))
		if err != nil {
			return false, fmt.Errorf("fact %q: %w", name, err)
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

func predicateRejectionReason(predicate map[string]string, facts Facts) string {
	for name, requirement := range predicate {
		actual, ok := facts[name]
		if !ok {
			return fmt.Sprintf("fact %q is absent", name)
		}
		matched, err := compareFact(normalizeFact(name, actual), normalizeFact(name, requirement))
		if err != nil {
			return fmt.Sprintf("fact %q is invalid: %v", name, err)
		}
		if !matched {
			return fmt.Sprintf("fact %q=%q does not satisfy %q", name, actual, requirement)
		}
	}
	return "not selected because an earlier candidate matched"
}

func normalizeFact(name, value string) string {
	if strings.EqualFold(strings.TrimSpace(name), "os") {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "darwin", "mac", "macos":
			return "macos"
		}
	}
	return strings.TrimSpace(value)
}

func compareFact(actual, requirement string) (bool, error) {
	requirement = strings.TrimSpace(requirement)
	for _, operator := range []string{">=", "<=", ">", "<"} {
		if strings.HasPrefix(requirement, operator) {
			want := strings.TrimSpace(strings.TrimPrefix(requirement, operator))
			if want == "" {
				return false, errors.New("comparison has no value")
			}
			gotNumber, gotErr := strconv.ParseFloat(strings.TrimSpace(actual), 64)
			wantNumber, wantErr := strconv.ParseFloat(want, 64)
			if gotErr != nil || wantErr != nil {
				return false, fmt.Errorf("comparison requires numeric values (actual %q, required %q)", actual, want)
			}
			switch operator {
			case ">=":
				return gotNumber >= wantNumber, nil
			case "<=":
				return gotNumber <= wantNumber, nil
			case ">":
				return gotNumber > wantNumber, nil
			case "<":
				return gotNumber < wantNumber, nil
			}
		}
	}
	return actual == requirement, nil
}

func cloneFacts(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
