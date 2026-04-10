# Meta-Orchestrator Summary: GCT GitHub Integration

## Source
Planning conversation covering Git Control Tower enhancement roadmap. This initiative adds GitHub API integration for PR management, tags, and release publishing.

## Decisions Made
- GitHub client must be behind an interface for full mockability — zero real GitHub API calls in tests
- Auth: SSH handles git transport, but GitHub API calls need token-based auth (PAT or app token)
- Tokens stored in GCT's existing credential storage system alongside SSH keys
- PR endpoints: create, list, view, merge
- Tag/release endpoints: list tags, create tag, draft release notes, publish release, list releases
- PR descriptions can auto-populate from commit analysis (enhanced by trailers from gct-commit-initiative-linking initiative when available)
- Release authoring includes: template system for consistent format, rich markdown editor, release image attachment, one-click publish

## Dependency Notes
- Research item should evaluate go-github library
- PR API and release API both depend on the research item but are independent of each other (can be built in parallel)
- Release authoring UI depends on the release API
- The PR description generator (in gct-release-pipeline initiative) depends on the PR API from this initiative
- Testing architecture is critical — mock interface design during research phase sets the pattern for all execution items

## Unresolved Questions Deferred To Workshop
- go-github vs raw HTTP client evaluation
- GitHub App token vs PAT trade-offs (Apps have better rate limits but more setup)
- What PR metadata to surface in the UI (reviews, checks, labels, assignees?)
- Whether to support GitLab/Bitbucket in the future (affects interface design breadth)
- How review panel results should feed into PR descriptions (automatic section? manual inclusion?)
