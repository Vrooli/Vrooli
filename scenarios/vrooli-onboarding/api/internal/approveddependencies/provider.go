// Package approveddependencies provides a principal-bound, read-only view of
// the repository's owner-reviewed third-party dependency governance records.
// Approval and edits remain owned by scenario-dependency-analyzer's evidence
// and review workflow; this package intentionally has no write path.
package approveddependencies

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const SourcePath = ".vrooli/dependencies/approved-dependencies.json"

var ErrReadOnly = errors.New("approved dependency governance is read-only here; use scenario-dependency-analyzer review workflow")

type Record struct {
	Ecosystem     string `json:"ecosystem"`
	PackageName   string `json:"package_name"`
	VersionRange  string `json:"version_range"`
	State         string `json:"state"`
	RangePolicy   string `json:"range_policy,omitempty"`
	Rationale     string `json:"rationale,omitempty"`
	ApprovedBy    string `json:"approved_by,omitempty"`
	ApprovedDate  string `json:"approved_date,omitempty"`
	LastReviewed  string `json:"last_reviewed,omitempty"`
	ReviewExpires string `json:"review_expires,omitempty"`
	LicenseNotes  string `json:"license_notes,omitempty"`
	SecurityNotes string `json:"security_notes,omitempty"`
}

type Snapshot struct {
	Principal      string   `json:"principal"`
	Source         string   `json:"source"`
	SourceRevision string   `json:"source_revision,omitempty"`
	SourceDigest   string   `json:"source_digest"`
	SchemaVersion  string   `json:"schema_version"`
	Version        string   `json:"version"`
	Mode           string   `json:"mode"`
	Guidance       string   `json:"guidance"`
	Records        []Record `json:"records"`
}

// Provider reads the canonical registry from an explicitly supplied checkout.
// It does not infer approval from the mere presence of a record: callers must
// present a principal and consumers must honor state and policy fields.
type Provider struct{ repoRoot string }

func New(repoRoot string) *Provider { return &Provider{repoRoot: repoRoot} }

func (p *Provider) Read(ctx context.Context, principal string) (Snapshot, error) {
	principal = strings.TrimSpace(principal)
	if principal == "" {
		return Snapshot{}, errors.New("approved dependencies: principal is required")
	}
	if p == nil || strings.TrimSpace(p.repoRoot) == "" {
		return Snapshot{}, errors.New("approved dependencies: repository root is required")
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	path := filepath.Join(p.repoRoot, filepath.FromSlash(SourcePath))
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("approved dependencies: read %s: %w", SourcePath, err)
	}
	var raw struct {
		SchemaVersion string `json:"schema_version"`
		Version       string `json:"version"`
		Guidance      string `json:"guidance"`
		Policy        struct {
			Mode string `json:"mode"`
		} `json:"policy"`
		Records []Record `json:"records"`
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	if err := dec.Decode(&raw); err != nil {
		return Snapshot{}, fmt.Errorf("approved dependencies: invalid JSON: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return Snapshot{}, errors.New("approved dependencies: trailing JSON content")
	}
	if raw.SchemaVersion != "1" || raw.Version == "" || raw.Policy.Mode == "" || raw.Records == nil {
		return Snapshot{}, errors.New("approved dependencies: incomplete or unsupported registry schema")
	}
	for i, r := range raw.Records {
		if strings.TrimSpace(r.Ecosystem) == "" || strings.TrimSpace(r.PackageName) == "" || strings.TrimSpace(r.VersionRange) == "" || strings.TrimSpace(r.State) == "" {
			return Snapshot{}, fmt.Errorf("approved dependencies: record %d is missing identity or state", i)
		}
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	digest := sha256.Sum256(data)
	return Snapshot{Principal: principal, Source: SourcePath, SourceRevision: repositoryRevision(ctx, p.repoRoot), SourceDigest: "sha256:" + hex.EncodeToString(digest[:]), SchemaVersion: raw.SchemaVersion, Version: raw.Version, Mode: raw.Policy.Mode, Guidance: raw.Guidance, Records: raw.Records}, nil
}

// RefuseMutation makes the boundary explicit for generic provider dispatchers.
func (p *Provider) RefuseMutation(context.Context, string, []byte) error { return ErrReadOnly }

func repositoryRevision(ctx context.Context, root string) string {
	cmd := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
