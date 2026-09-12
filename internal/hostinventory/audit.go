package hostinventory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
)

type AuditKind string

const (
	AuditUndeclaredMutation AuditKind = "undeclared_mutation"
	AuditMissingDeclaration AuditKind = "missing_declared_artifact"
	AuditContentDrift       AuditKind = "content_drift"
)

type DeclaredArtifact struct {
	Owner       string
	Path        string
	ContentHash string
}

type ObservedArtifact struct {
	Path        string
	ContentHash string
}

type AuditFinding struct {
	Kind   AuditKind `json:"kind"`
	Path   string    `json:"path"`
	Owner  string    `json:"owner,omitempty"`
	Reason string    `json:"reason"`
}

type AuditReport struct {
	Findings []AuditFinding `json:"findings,omitempty"`
}

func AuditArtifacts(declared []DeclaredArtifact, observed []ObservedArtifact) AuditReport {
	byPath := make(map[string]DeclaredArtifact, len(declared))
	for _, item := range declared {
		byPath[item.Path] = item
	}
	seen := make(map[string]bool, len(observed))
	report := AuditReport{}
	for _, item := range observed {
		declaredItem, ok := byPath[item.Path]
		if !ok {
			report.Findings = append(report.Findings, AuditFinding{Kind: AuditUndeclaredMutation, Path: item.Path, Reason: "observed host artifact has no declaration"})
			continue
		}
		seen[item.Path] = true
		if declaredItem.ContentHash != "" && item.ContentHash != declaredItem.ContentHash {
			report.Findings = append(report.Findings, AuditFinding{Kind: AuditContentDrift, Path: item.Path, Owner: declaredItem.Owner, Reason: "observed content hash differs from declaration"})
		}
	}
	for _, item := range declared {
		if !seen[item.Path] {
			report.Findings = append(report.Findings, AuditFinding{Kind: AuditMissingDeclaration, Path: item.Path, Owner: item.Owner, Reason: "declared host artifact was not observed"})
		}
	}
	sort.Slice(report.Findings, func(i, j int) bool { return report.Findings[i].Path < report.Findings[j].Path })
	return report
}

func HashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
