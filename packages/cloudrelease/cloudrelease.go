// Package cloudrelease is the one definition of a cloud release identity.
//
// The cloud side (scenario-to-cloud, its own Go module) builds a release
// artifact set and publishes a manifest; the target side (internal/cloudtarget,
// run as `vrooli cloud-target release verify|stage`) verifies that manifest
// before anything is extracted. Go's internal rule forbids a scenario module
// from importing internal/, so the manifest shape, the canonical JSON form and
// the digest rule live here, in the root module, and both sides import them.
// Two implementations of "canonical JSON" would be two ways to compute the
// same digest and, eventually, two digests.
//
// # Identity
//
// release_digest = hex(sha256(CanonicalJSON(ReleaseIdentity))) where the
// canonical JSON is sorted-key compact encoding/json output:
//
//	{"bundle_sha256":"…","closure_digest":"…","configuration_digest":"…","native_cli":{"goarch":"…","goos":"…","sha256":"…"}}
//
// Provenance and limits are outside the identity on purpose: they describe how
// the release was produced and how it may be unpacked, not what it is.
package cloudrelease

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
)

// ManifestSchemaVersion is the only manifest schema a target accepts.
const ManifestSchemaVersion = 1

// ArchiveFormat is the only bundle format a target accepts.
const ArchiveFormat = "tar.gz"

// Default bounds applied when a manifest omits a limit.
const (
	DefaultMaxEntries       int64 = 200_000
	DefaultMaxExpandedBytes int64 = 8 << 30
	DefaultMaxEntryBytes    int64 = 2 << 30
)

// File names inside a release directory (cloud store and target store alike).
const (
	// BundleFileName is the deterministic mini-Vrooli tar.gz.
	BundleFileName = "bundle.tar.gz"
	// ManifestFileName is the release manifest this package defines.
	ManifestFileName = "release-manifest.json"
	// InputsFileName is the cloud-side record of every material build input.
	InputsFileName = "inputs.json"
	// CompleteMarker is written last; a directory without it is not a release.
	CompleteMarker = ".complete"
	// ProvenanceDirName holds the signing stage (resource-deployment release
	// manifest, a copy of the cloud manifest and the detached signature).
	ProvenanceDirName = "provenance"
	// ProvenanceManifestCopy is the name the cloud manifest is copied under in
	// the signing stage, so the release authority's artifact list can pin it.
	ProvenanceManifestCopy = "cloud-release-manifest.json"
)

// Provenance policies. A release always says which one produced it; an
// unsigned release is a declared choice, never a silent absence.
const (
	PolicyDevelopmentLocalUnsigned = "development-local-unsigned"
	PolicyProductionSigned         = "production-signed"
)

// NativeCLIFileName is the file name of the control-plane binary for a platform.
func NativeCLIFileName(goos, goarch string) string {
	return "vrooli-" + goos + "-" + goarch
}

// Manifest is the release manifest the cloud side publishes beside a bundle.
// It carries references and digests only; a credential value never belongs
// here and the verifier does not look for one.
type Manifest struct {
	SchemaVersion       int        `json:"schema_version"`
	ReleaseDigest       string     `json:"release_digest"`
	BundleSHA256        string     `json:"bundle_sha256"`
	NativeCLI           NativeCLI  `json:"native_cli"`
	ClosureDigest       string     `json:"closure_digest"`
	ConfigurationDigest string     `json:"configuration_digest"`
	Provenance          Provenance `json:"provenance"`
	Limits              Limits     `json:"limits"`
}

// NativeCLI binds the release to one control-plane binary build.
type NativeCLI struct {
	SHA256 string `json:"sha256"`
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
}

// Provenance names the builder and the policy it was verified under.
type Provenance struct {
	Builder string `json:"builder"`
	Policy  string `json:"policy"`
}

// Limits are the archive budgets declared by the release.
type Limits struct {
	MaxEntries       int64 `json:"max_entries"`
	MaxExpandedBytes int64 `json:"max_expanded_bytes"`
	MaxEntryBytes    int64 `json:"max_entry_bytes"`
}

// ReleaseIdentity is the exact value set the release digest covers.
type ReleaseIdentity struct {
	BundleSHA256        string    `json:"bundle_sha256"`
	NativeCLI           NativeCLI `json:"native_cli"`
	ClosureDigest       string    `json:"closure_digest"`
	ConfigurationDigest string    `json:"configuration_digest"`
}

// Identity extracts the digest-covered values of a manifest.
func (m Manifest) Identity() ReleaseIdentity {
	return ReleaseIdentity{BundleSHA256: m.BundleSHA256, NativeCLI: m.NativeCLI, ClosureDigest: m.ClosureDigest, ConfigurationDigest: m.ConfigurationDigest}
}

// ComputeReleaseDigest returns the digest a manifest must carry.
func ComputeReleaseDigest(m Manifest) (string, error) {
	return CanonicalDigest(m.Identity())
}

// DefaultLimits are the bounds a release declares unless it narrows them.
func DefaultLimits() Limits {
	return Limits{MaxEntries: DefaultMaxEntries, MaxExpandedBytes: DefaultMaxExpandedBytes, MaxEntryBytes: DefaultMaxEntryBytes}
}

// ExtractOptions returns the bounded-extraction options a manifest declares,
// filling defaults for any omitted limit.
func (m Manifest) ExtractOptions(deadline time.Time) binaryfetch.ExtractOptions {
	opts := binaryfetch.ExtractOptions{MaxEntries: m.Limits.MaxEntries, MaxExpandedBytes: m.Limits.MaxExpandedBytes, MaxEntryBytes: m.Limits.MaxEntryBytes, Deadline: deadline}
	if opts.MaxEntries <= 0 {
		opts.MaxEntries = DefaultMaxEntries
	}
	if opts.MaxExpandedBytes <= 0 {
		opts.MaxExpandedBytes = DefaultMaxExpandedBytes
	}
	if opts.MaxEntryBytes <= 0 {
		opts.MaxEntryBytes = DefaultMaxEntryBytes
	}
	return opts
}

var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

// IsDigest reports whether value is a lowercase sha256 hex string.
func IsDigest(value string) bool { return digestPattern.MatchString(value) }

// ErrManifestInvalid wraps every shape failure ParseManifest reports.
var ErrManifestInvalid = errors.New("release manifest invalid")

// ParseManifest decodes and shape-checks a manifest without verifying it.
// Every failure wraps ErrManifestInvalid.
func ParseManifest(raw []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return Manifest{}, fmt.Errorf("%w: parse: %v", ErrManifestInvalid, err)
	}
	if m.SchemaVersion != ManifestSchemaVersion {
		return Manifest{}, fmt.Errorf("%w: schema_version %d is not %d", ErrManifestInvalid, m.SchemaVersion, ManifestSchemaVersion)
	}
	for _, field := range []struct{ name, value string }{
		{"release_digest", m.ReleaseDigest}, {"bundle_sha256", m.BundleSHA256}, {"native_cli.sha256", m.NativeCLI.SHA256},
	} {
		if !IsDigest(field.value) {
			return Manifest{}, fmt.Errorf("%w: %s is not a lowercase sha256 hex string", ErrManifestInvalid, field.name)
		}
	}
	if strings.TrimSpace(m.NativeCLI.GOOS) == "" || strings.TrimSpace(m.NativeCLI.GOARCH) == "" {
		return Manifest{}, fmt.Errorf("%w: native_cli must name goos and goarch", ErrManifestInvalid)
	}
	if strings.TrimSpace(m.ClosureDigest) == "" || strings.TrimSpace(m.ConfigurationDigest) == "" {
		return Manifest{}, fmt.Errorf("%w: closure_digest and configuration_digest are required", ErrManifestInvalid)
	}
	return m, nil
}

// CanonicalDigest returns the sha256 hex of the canonical JSON encoding of
// value.
func CanonicalDigest(value any) (string, error) {
	canonical, err := CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// CanonicalJSON encodes value as sorted-key compact JSON: encoding/json with
// object keys sorted, no insignificant whitespace, UTF-8, and encoding/json's
// default escaping. A struct is first round-tripped through a generic value so
// its keys sort. The same bytes are produced by Python's
// json.dumps(value, sort_keys=True, separators=(",", ":")) for ASCII content.
func CanonicalJSON(value any) ([]byte, error) {
	if value == nil {
		return []byte("null"), nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	return json.Marshal(sortKeys(generic))
}

func sortKeys(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(typed))
		for _, key := range keys {
			out[key] = sortKeys(typed[key])
		}
		return out
	case []any:
		for i := range typed {
			typed[i] = sortKeys(typed[i])
		}
		return typed
	default:
		return value
	}
}
