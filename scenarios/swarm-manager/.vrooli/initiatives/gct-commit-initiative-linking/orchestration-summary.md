# Meta-Orchestrator Summary: GCT Commit-Initiative Linking

## Source
Planning conversation covering Git Control Tower enhancement roadmap. This initiative creates a namespaced git trailer system linking commits to Swarm Manager initiatives.

## Decisions Made
- Use Vrooli-namespaced git trailers to avoid collisions: Vrooli-Initiative, Vrooli-Backlog, Vrooli-Continues
- Git trailers are machine-parseable via `git log --format='%(trailers:key=Vrooli-Initiative,valueonly)'`
- Trailers live in the commit message itself — no external database needed for linkage
- The "continue" feature in GCT (incrementing part number) should also: copy Vrooli-* trailers from previous commit, add Vrooli-Continues trailer with previous commit hash
- Swarm Manager integration is optional with graceful degradation:
  - Trailer present + Swarm Manager available → rich initiative context
  - Trailer present + Swarm Manager unavailable → group by trailer value, summarize commits
  - No trailer → standard commit message only
- Same optional-dependency pattern as other GCT integrations (tidiness-manager, test-genie, etc.)

## Key Technical Context: Git Trailers
- Standard git feature since Git 2.1 (~2014)
- Added via `git commit --trailer "Key: Value"`
- Queried via `git log --format='%(trailers:key=Key,valueonly)'`
- Survive rebases, cherry-picks, format-patches
- Multiple values per key supported
- GitHub renders them in commit view

## Dependency Notes
- This initiative is independent but enhances gct-release-pipeline
- Research is small (effort: S) since the trailer convention is straightforward
- Swarm Manager integration follows the same client pattern as other GCT optional deps

## Unresolved Questions Deferred To Workshop
- Final trailer key names (Vrooli-Initiative vs Vrooli-initiative vs other casing)
- Whether to auto-suggest initiative slugs from Swarm Manager API or just from recent trailer history
- How the trailer editor should look in the commit panel (always visible? collapsible section? advanced mode?)
- Whether Vrooli-Continues should be auto-populated silently or shown to the user for confirmation
