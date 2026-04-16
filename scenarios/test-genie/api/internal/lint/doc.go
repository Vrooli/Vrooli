// Package lint provides component-based static analysis validation for scenarios.
//
// It discovers top-level components, matches them to supported lint handlers,
// runs the matched handler, and applies policy to unmatched components.
//
// Current handlers:
//   - go_module: golangci-lint (preferred) or go vet (fallback)
//   - node_package: tsc for type checking, eslint for linting
//   - python_project: ruff (preferred) or flake8 (fallback), mypy for type checking
//
// Severity handling:
//   - type errors fail the phase
//   - lint warnings fail only when strict mode is enabled
//   - policy errors for unmatched components fail the phase
package lint
