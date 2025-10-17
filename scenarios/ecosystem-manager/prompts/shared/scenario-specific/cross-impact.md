# Cross-Scenario Impact

Scenarios rarely stand alone; they call each other's APIs, share databases, and rely on the same resource pool. Before modifying behavior, take a moment to note who consumes the capability you are touching and confirm they can tolerate the change. Small updates can cascade, so choose the safest option that keeps dependents working.

## API Version Bumps
- Add a new version (e.g., `/api/v2/...`) whenever you introduce a breaking change in payload shape, response codes, authentication, or timing guarantees.
- Keep the previous version running with a deprecation note until downstream scenarios have migrated; do not remove it without proof the callers have switched.
- Surface the migration steps in the PRD/README and link to them in commit notes so future improvers can finish the rollout.
