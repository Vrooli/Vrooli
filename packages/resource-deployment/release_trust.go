package resourcedeployment

// This file is deliberately dependency-free: the same release contract is
// used by the source stager and the desktop admission boundary.

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/binaryfetch"
)

type ArtifactTrustMode string

const (
	ArtifactTrustDevelopmentLocal ArtifactTrustMode = "development-local"
	ArtifactTrustProduction       ArtifactTrustMode = "production"
)

func (m ArtifactTrustMode) Validate() error {
	if m != ArtifactTrustDevelopmentLocal && m != ArtifactTrustProduction {
		return fmt.Errorf("artifact trust mode must be development-local or production")
	}
	return nil
}

type ReleaseManifest struct {
	SchemaVersion string            `json:"schema_version"`
	Artifacts     []ReleaseArtifact `json:"artifacts"`
}
type ReleaseArtifact struct {
	Name               string `json:"name"`
	SHA256             string `json:"sha256"`
	Role               string `json:"role"`
	OS                 string `json:"os,omitempty"`
	Arch               string `json:"arch,omitempty"`
	UpstreamProvenance string `json:"upstream_provenance"`
}
type ReleaseSignature struct {
	SchemaVersion string `json:"schema_version"`
	KeyID         string `json:"key_id"`
	Algorithm     string `json:"algorithm"`
	Signature     string `json:"signature"`
}

// ReleaseManifestSigner is implemented by the external release authority.
// It receives canonical public manifest bytes only; private keys never enter
// the repository, local configuration, or desktop pipeline.
type ReleaseManifestSigner interface {
	SignReleaseManifest(canonicalManifest []byte) (ReleaseSignature, error)
}

func (m ReleaseManifest) CanonicalBytes() ([]byte, error) {
	if m.SchemaVersion != "v1" {
		return nil, fmt.Errorf("unsupported release manifest schema %q", m.SchemaVersion)
	}
	a := append([]ReleaseArtifact(nil), m.Artifacts...)
	sort.Slice(a, func(i, j int) bool { return a[i].Name < a[j].Name })
	if len(a) == 0 {
		return nil, fmt.Errorf("release manifest has no artifacts")
	}
	seen := map[string]bool{}
	for _, x := range a {
		if !IsSafeArtifactName(x.Name) || seen[x.Name] || len(x.SHA256) != 64 || strings.TrimSpace(x.Role) == "" || strings.TrimSpace(x.UpstreamProvenance) == "" {
			return nil, fmt.Errorf("invalid release artifact %q", x.Name)
		}
		if _, e := hex.DecodeString(x.SHA256); e != nil {
			return nil, fmt.Errorf("invalid release artifact digest %q", x.Name)
		}
		seen[x.Name] = true
	}
	return json.Marshal(struct {
		SchemaVersion string            `json:"schema_version"`
		Artifacts     []ReleaseArtifact `json:"artifacts"`
	}{m.SchemaVersion, a})
}

func LoadReleaseManifest(root string) (ReleaseManifest, error) {
	b, e := os.ReadFile(filepath.Join(root, "release-manifest.json"))
	if e != nil {
		return ReleaseManifest{}, fmt.Errorf("read release manifest: %w", e)
	}
	var m ReleaseManifest
	if e = json.Unmarshal(b, &m); e != nil {
		return m, fmt.Errorf("parse release manifest: %w", e)
	}
	if _, e = m.CanonicalBytes(); e != nil {
		return m, e
	}
	return m, nil
}

func VerifyReleaseDirectory(root string, mode ArtifactTrustMode, publicKeyPath string) (ReleaseManifest, *ReleaseSignature, error) {
	if e := mode.Validate(); e != nil {
		return ReleaseManifest{}, nil, e
	}
	m, e := LoadReleaseManifest(root)
	if e != nil {
		return m, nil, e
	}
	canonical, e := m.CanonicalBytes()
	if e != nil {
		return m, nil, e
	}
	for _, a := range m.Artifacts {
		artifactPath := filepath.Join(root, a.Name)
		info, statErr := os.Stat(artifactPath)
		if statErr != nil {
			return m, nil, fmt.Errorf("read staged artifact %s: %w", a.Name, statErr)
		}
		got := ""
		if info.IsDir() {
			got, statErr = binaryfetch.TreeDigest(artifactPath)
		} else {
			b, readErr := os.ReadFile(artifactPath)
			statErr = readErr
			if statErr == nil {
				sum := sha256.Sum256(b)
				got = hex.EncodeToString(sum[:])
			}
		}
		if statErr != nil {
			return m, nil, fmt.Errorf("hash staged artifact %s: %w", a.Name, statErr)
		}
		if got != a.SHA256 {
			return m, nil, fmt.Errorf("staged artifact %s hash mismatch", a.Name)
		}
	}
	if mode == ArtifactTrustDevelopmentLocal {
		return m, nil, nil
	}
	b, e := os.ReadFile(filepath.Join(root, "release-manifest.sig.json"))
	if e != nil {
		return m, nil, fmt.Errorf("production release signature missing: configure the Vrooli release authority and provide release-manifest.sig.json: %w", e)
	}
	var s ReleaseSignature
	if e = json.Unmarshal(b, &s); e != nil {
		return m, nil, fmt.Errorf("parse release signature: %w", e)
	}
	if s.SchemaVersion != "v1" || s.Algorithm != "rsa-pkcs1v15-sha256" || strings.TrimSpace(s.KeyID) == "" {
		return m, nil, fmt.Errorf("invalid production release signature envelope")
	}
	sig, e := base64.StdEncoding.DecodeString(s.Signature)
	if e != nil {
		return m, nil, fmt.Errorf("decode release signature: %w", e)
	}
	pemData, e := os.ReadFile(publicKeyPath)
	if e != nil {
		return m, nil, fmt.Errorf("read release trust key: %w", e)
	}
	block, _ := pem.Decode(pemData)
	if block == nil {
		return m, nil, fmt.Errorf("parse release trust key: invalid PEM")
	}
	pub, e := x509.ParsePKIXPublicKey(block.Bytes)
	if e != nil {
		return m, nil, e
	}
	key, ok := pub.(*rsa.PublicKey)
	if !ok {
		return m, nil, fmt.Errorf("release trust key is not RSA")
	}
	digest := sha256.Sum256(canonical)
	if e = rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); e != nil {
		return m, nil, fmt.Errorf("production release signature invalid: %w", e)
	}
	return m, &s, nil
}
