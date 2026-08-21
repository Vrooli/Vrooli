// Package analysis owns analyzer brokering and normalized graph projections.
package analysis

import "context"

type Request struct {
	Path       string
	Language   string
	SourceHash string
	Families   []string
}

type Fact struct {
	ID         string
	Family     string
	Kind       string
	Subject    string
	Predicate  string
	Object     string
	Path       string
	SourceHash string
	Proof      string
	Analyzer   string
	Version    string
	Attributes map[string]string
}

type Result struct {
	Analyzer string
	Version  string
	Facts    []Fact
	Warnings []string
}

type Analyzer interface {
	Analyze(context.Context, Request) (Result, error)
}

type ProjectionStore interface {
	Replace(context.Context, string, string, []Fact) error
	Expand(context.Context, string, string, []string, int) ([]Fact, error)
}
