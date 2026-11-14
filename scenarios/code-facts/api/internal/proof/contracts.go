// Package proof owns contract and relationship evidence semantics.
package proof

import (
	"context"
	"time"
)

type Status string

const (
	StatusProven       Status = "proven"
	StatusMissing      Status = "missing"
	StatusContradicted Status = "contradicted"
	StatusUnsupported  Status = "unsupported"
	StatusUnknown      Status = "unknown"
)

type Evidence struct {
	ID         string
	Status     Status
	Path       string
	StartLine  int
	EndLine    int
	SourceHash string
	Generation string
	Analyzer   string
	Message    string
}

type Contract struct {
	ID         string
	Kind       string
	Name       string
	FullName   string
	ParentID   string
	Path       string
	StartLine  int
	EndLine    int
	Comment    string
	SourceHash string
	Digest     string
	Attributes map[string]string
}

type ContractSnapshot struct {
	Digest               string
	DescriptorGeneration uint64
	Contracts            []Contract
	ProvenanceFailures   []string
	LastReloadFailure    string
	LastReloadFailureAt  time.Time
}

type ContractSource interface {
	Snapshot(context.Context) (ContractSnapshot, error)
}

type ProjectionReader interface {
	Relationships(context.Context, string, string, int) ([]Evidence, error)
}
