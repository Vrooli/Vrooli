# Meta-Orchestrator Summary: GCT Release Pipeline

## Source
Planning conversation covering Git Control Tower enhancement roadmap. This initiative builds agent-powered PR description and release notes generation using the commit-to-initiative information hierarchy.

## Decisions Made
- Information hierarchy compresses at each level: commits → initiatives → PR descriptions → release notes
- PR description generator: collects commits on branch, groups by Vrooli-Initiative trailer, enriches from Swarm Manager if available, generates structured draft (summary, changes by initiative, test plan), presents as editable before PR creation
- Release notes generator: aggregates PR descriptions since last release tag, categorizes (features, fixes, improvements), applies user's preferred format template, supports release image attachment
- The release agent only needs to look at PRs (not raw commits) since each PR already has structured summaries
- Template system for consistent release format stored in GCT settings
- Graceful degradation at every level:
  - With trailers + Swarm Manager: richest output
  - With trailers only: grouped by slug, agent summarizes
  - Without trailers: agent reads all commits (current manual process, automated)

## Cross-Initiative Dependencies
- gct-pr-description-generator depends on gct-github-pr-api (from gct-github-integration)
- gct-release-notes-generator depends on gct-github-release-api (from gct-github-integration) AND gct-pr-description-generator
- Both items are enhanced by gct-trailer-support (from gct-commit-initiative-linking) but work without it
- Both items are enhanced by gct-swarm-manager-integration but work without it

## User's Current Pain Point
- The user's current branch has ~1000 commits
- Generating release notes currently takes hours of manual work with a coding agent
- This friction means releases happen rarely instead of weekly
- The commit "part N" convention (e.g., "swarm-manager qol improvements p3") groups related commits but isn't formalized
- Future commits with trailers will make this dramatically easier; existing backlog must fall back to commit analysis

## Unresolved Questions Deferred To Workshop
- Exact release template format (look at previous releases for the pattern the user likes)
- Whether the PR description should be auto-generated on PR creation or require explicit trigger
- How to handle the existing 1000-commit backlog for the first release using this system
- Whether release notes should include a changelog section separate from the narrative summary
- Rich editor choice for release authoring (Monaco markdown? Dedicated markdown editor with preview?)
