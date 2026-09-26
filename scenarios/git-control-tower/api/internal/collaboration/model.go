// Package collaboration owns host-neutral collaboration contracts. Provider
// authentication and SDKs stay outside Git Control Tower.
package collaboration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type HostKind string

const (
	HostGitHub    HostKind = "github"
	HostGitLab    HostKind = "gitlab"
	HostBitbucket HostKind = "bitbucket"
)

type Capability string

const (
	CapabilityReadChange Capability = "read_change"
	CapabilityComments   Capability = "comments"
	CapabilityReviews    Capability = "reviews"
	CapabilityChecks     Capability = "checks"
	CapabilityReleases   Capability = "releases"
	CapabilityPublish    Capability = "publish"
)

type CapabilityStanding string

const (
	StandingUnconfigured CapabilityStanding = "unconfigured"
	StandingAvailable    CapabilityStanding = "available"
	StandingUnsupported  CapabilityStanding = "unsupported"
	StandingDenied       CapabilityStanding = "denied"
	StandingDisconnected CapabilityStanding = "disconnected"
	StandingRevoked      CapabilityStanding = "revoked"
	StandingRateLimited  CapabilityStanding = "rate_limited"
)

type HostIdentity struct {
	Kind           HostKind `json:"kind"`
	InstanceURL    string   `json:"instanceUrl"`
	InstallationID string   `json:"installationId,omitempty"`
	AccountID      string   `json:"accountId,omitempty"`
	RepositoryID   string   `json:"repositoryId"`
}

type CapabilityStatus struct {
	Capability Capability         `json:"capability"`
	Standing   CapabilityStanding `json:"standing"`
	Scopes     []string           `json:"scopes,omitempty"`
	Reason     string             `json:"reason,omitempty"`
}

type Change struct {
	Host         HostIdentity       `json:"host"`
	Number       string             `json:"number"`
	Title        string             `json:"title"`
	BaseRevision string             `json:"baseRevision"`
	HeadRevision string             `json:"headRevision"`
	State        string             `json:"state"`
	Capabilities []CapabilityStatus `json:"capabilities"`
}

type DraftPublication struct {
	DraftDigest       string `json:"draftDigest"`
	Destination       string `json:"destination"`
	ExpectedRevision  string `json:"expectedRevision"`
	AuthorityStanding string `json:"authorityStanding"`
}

func (h HostIdentity) Validate() error {
	if h.Kind != HostGitHub && h.Kind != HostGitLab && h.Kind != HostBitbucket {
		return fmt.Errorf("unsupported host kind %q", h.Kind)
	}
	if strings.TrimSpace(h.InstanceURL) == "" || strings.TrimSpace(h.RepositoryID) == "" {
		return fmt.Errorf("host instance and repository identity are required")
	}
	return nil
}

func (c Change) Validate() error {
	if err := c.Host.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(c.Number) == "" || strings.TrimSpace(c.HeadRevision) == "" {
		return fmt.Errorf("change number and head revision are required")
	}
	return nil
}

func (p DraftPublication) Validate() error {
	if strings.TrimSpace(p.DraftDigest) == "" || strings.TrimSpace(p.Destination) == "" || strings.TrimSpace(p.ExpectedRevision) == "" {
		return fmt.Errorf("publication requires exact draft, destination, and revision")
	}
	if p.AuthorityStanding != "human_verified" {
		return fmt.Errorf("publication requires human_verified authority")
	}
	return nil
}

func (c Change) IdentityDigest() (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}
	value := strings.Join([]string{string(c.Host.Kind), c.Host.InstanceURL, c.Host.RepositoryID, c.Number, c.BaseRevision, c.HeadRevision}, "\x00")
	h := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(h[:]), nil
}
