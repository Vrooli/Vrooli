// Package lint provides static analysis validation for scenarios.
//
// It orchestrates linting and type checking across multiple languages:
//   - Go: golangci-lint (preferred) or go vet (fallback)
//   - TypeScript/JavaScript: tsc for type checking, eslint for linting
//   - Python: ruff (preferred) or flake8 (fallback), mypy for type checking
//
// The package follows the same patterns as the dependencies package,
// with a Runner that coordinates language-specific linters and
// aggregates their results into a unified report.
//
// Severity Handling:
//   - Type errors (tsc, mypy, go vet, typecheck): Fail the phase
//   - Lint warnings (eslint, ruff, stylistic linters): Pass with warnings
package lint
