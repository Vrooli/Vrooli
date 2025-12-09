# Architecture Decision Records

> This directory documents significant architectural decisions made in deployment-manager.
> ADRs help future contributors understand *why* things are designed the way they are.

## What is an ADR?

An Architecture Decision Record (ADR) captures a single architectural decision along with its context and consequences. ADRs are immutable once accepted - if a decision changes, a new ADR supersedes the old one.

## Decision Index

| ID | Decision | Status | Date |
|----|----------|--------|------|
| [001](001-tiered-deployment.md) | Tiered Deployment Model | Accepted | Dec 2024 |
| [002](002-manifest-driven-bundles.md) | Manifest-Driven Bundle Architecture | Accepted | Dec 2024 |
| [003](003-runtime-supervisor.md) | Go Runtime Supervisor | Accepted | Dec 2024 |
| [004](004-secrets-classification.md) | Four-Class Secrets Model | Accepted | Dec 2024 |

## ADR Template

When adding a new decision, use this template:

```markdown
# ADR-XXX: Title

## Status

Proposed | Accepted | Deprecated | Superseded by [ADR-YYY](YYY-title.md)

## Context

What is the issue that we're seeing that motivates this decision?

## Decision

What is the change that we're proposing and/or doing?

## Consequences

What becomes easier or harder because of this decision?

### Positive

- Benefit 1
- Benefit 2

### Negative

- Drawback 1
- Drawback 2

### Neutral

- Trade-off 1

## Alternatives Considered

What other options were evaluated?

### Alternative 1

Why it was rejected.

### Alternative 2

Why it was rejected.
```

## When to Write an ADR

Write an ADR when:

1. **Introducing new architecture** - Major components, patterns, or approaches
2. **Choosing between alternatives** - When multiple valid solutions exist
3. **Breaking from convention** - Doing something differently than expected
4. **Making trade-offs** - Accepting downsides for specific benefits
5. **Deprecating existing patterns** - Explaining why old approaches are replaced

## Related Documentation

- [Roadmap](../ROADMAP.md) - Implementation status
- [Technical Guides](../guides/) - How things work
- [Bundled Runtime Plan](/docs/plans/bundled-desktop-runtime-plan.md) - Detailed implementation plan
