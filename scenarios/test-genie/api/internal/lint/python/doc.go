// Package python provides Python-specific linting using ruff, flake8, and mypy.
//
// Tool priority for linting:
//  1. ruff check (fast, modern)
//  2. flake8 (fallback)
//
// Type checking:
//   - mypy (if mypy.ini or pyproject.toml has mypy config)
//
// Type errors from mypy are treated as errors that fail the phase.
// Linting issues are treated as warnings that don't fail the phase.
package python
