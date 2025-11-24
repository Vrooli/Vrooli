# Refactor Mode

Focus on code quality and maintainability. Reduce technical debt, improve code organization, and ensure adherence to project standards.

## Key Areas
- Simplify complex code paths
- Improve naming consistency
- Remove dead code and unused dependencies
- Increase test coverage for refactored areas
- Align with project style and lint rules

## Approach
- Identify hotspots using metrics and lints
- Apply small, safe refactors with tests
- Keep behavior identical; avoid feature changes
- Document notable design decisions

## Success Criteria
- Tidiness score improves
- Standards violations decrease
- No regression in operational targets or tests
- Code is more maintainable

## Recommended Tools
- tidiness-manager (identify issues, validate improvements)
- golangci-lint / eslint (automated quality checks)
- ast-grep (structural code analysis and refactoring)
