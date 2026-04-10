package report

import (
	"context"

	"development-toolchain-validator/domain/expectation"
	"development-toolchain-validator/domain/skill"
)

// SkillConnectionLister lists skill connections with optional filtering.
// Satisfied by skill.Repository directly.
type SkillConnectionLister interface {
	List(ctx context.Context, opts skill.ListOptions) ([]*skill.Connection, error)
}

// ExpectationLister lists expectations by connection.
// This is a composite interface — use NewExpectationRepoAdapter to create
// an implementation from the separate structural and CLI repositories.
type ExpectationLister interface {
	ListStructural(ctx context.Context, opts expectation.ListOptions) ([]*expectation.StructuralExpectation, error)
	ListCLI(ctx context.Context, opts expectation.ListOptions) ([]*expectation.CLIAssertion, error)
}

// ExpectationRepoAdapter wraps the two expectation repositories into a
// single ExpectationLister for the report service.
type ExpectationRepoAdapter struct {
	structural expectation.StructuralRepository
	cli        expectation.CLIRepository
}

// NewExpectationRepoAdapter creates an adapter from the two expectation repos.
func NewExpectationRepoAdapter(structural expectation.StructuralRepository, cli expectation.CLIRepository) *ExpectationRepoAdapter {
	return &ExpectationRepoAdapter{structural: structural, cli: cli}
}

func (a *ExpectationRepoAdapter) ListStructural(ctx context.Context, opts expectation.ListOptions) ([]*expectation.StructuralExpectation, error) {
	return a.structural.List(ctx, opts)
}

func (a *ExpectationRepoAdapter) ListCLI(ctx context.Context, opts expectation.ListOptions) ([]*expectation.CLIAssertion, error) {
	return a.cli.List(ctx, opts)
}

// ValidationResultReader reads aggregated validation results from storage.
type ValidationResultReader interface {
	// CLIResultsByReference returns CLI assertion results for the latest
	// validation run of the given reference. Each result includes the
	// connection_id via a join on cli_assertions.
	CLIResultsByReference(ctx context.Context, referenceID string) ([]*CLIResultRow, error)
}

// CLIResultRow is a denormalized row joining cli_results with cli_assertions
// to expose the connection_id alongside the result data.
type CLIResultRow struct {
	AssertionID  string `json:"assertion_id"`
	ConnectionID string `json:"connection_id"`
	Command      string `json:"command"`
	JSONPath     string `json:"json_path"`
	Status       string `json:"status"`
	ActualValue  string `json:"actual_value,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}
