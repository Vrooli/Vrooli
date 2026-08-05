# deployment-manager documentation

This is the maintained entry point for deployment-manager. The scenario owns
deployment policy, target fitness, approval gates, release records, and the
shared evidence contract; target ramps own build, packaging, execution, and
artifact publication.

## Start here

- [Architecture](concepts/ARCHITECTURE.md) — boundaries and proto-first data flow
- [Evidence contract](guides/evidence-contract.md) — the reviewable release evidence model
- [Desktop workflow](workflows/desktop-deployment.md) — scenario-to-desktop release path
- [Tier 2 desktop](tiers/tier-2-desktop.md) — desktop target requirements
- [CLI overview](cli/overview-commands.md) — current command surface
- [API reference](api/bundles.md) — representative Connect/API contracts
- [Product requirements](../PRD.md) — operational targets and traceability

## Reference by task

| Task | Document |
| --- | --- |
| Create and inspect profiles | [Profile commands](cli/profile-commands.md) |
| Analyze dependencies and fitness | [CLI overview](cli/overview-commands.md) and [fitness guide](guides/fitness-scoring.md) |
| Review target evidence | [Evidence contract](guides/evidence-contract.md) |
| Assemble a bundle | [Bundle manifest schema](guides/bundle-manifest-schema.md) |
| Sign a release | [Code signing](guides/code-signing.md) |
| Diagnose a failed release | [Troubleshooting](workflows/troubleshooting.md) |

## Engineering references

- [Architecture decision records](decisions/006-greenfield-storage.md)
- [Shared evidence decision](decisions/007-shared-evidence-contract.md)
- [Current problems and seams](internal/PROBLEMS.md) and [test seams](internal/SEAMS.md)
- [Requirements registry](../requirements/README.md)

The navigation in this file is intentionally composed of maintained files,
rather than deleted category index pages. `manifest.json` registers the
durable documentation set and is the source of truth for documentation audit.
