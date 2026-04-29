# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
Most of the available skills were authored against scenarios. Apply with a translator's mindset.

- **screaming-architecture-audit** — architecture dimension *(scenario-shaped)*
- **invariant-discovery-and-enforcement** — architecture + tests *(scenario-shaped)*
- **boundary-of-responsibility-enforcement** — architecture *(scenario-shaped)*
- **seam-discovery-and-enforcement** — test-coverage *(scenario-shaped)*
- **code-cleanup** — code-smell pass *(scenario-shaped)*
- **cognitive-load-reduction** — documentation + architecture *(scenario-shaped)*
- **decision-boundary-extraction** — architecture *(scenario-shaped)*
- **security** — security dimension *(scenario-shaped — most points apply directly)*
- **e2e-testing** — test-coverage *(scenario-shaped)*
- **documentation-health** — documentation dimension *(largely platform-neutral)*
- **signal-and-feedback-surface-design** — signal dimension *(scenario-shaped — read "scenario PRD" as "internal contract")*
- **cross-platform-readiness** — cross-platform dimension *(scenario-shaped — Tier-1 / Tier-2 framing applies; ignore per-scenario-bundle framing)*
- **error-semantics-recovery-path-design** — signal + cross-platform *(scenario-shaped)*
- **failure-topography-and-graceful-degradation** — signal + reliability targets *(scenario-shaped)*

## Primary Surfaces

Source code:
- `cli/` (or wherever the platform CLI source lives)
- `infra/`, `scripts/`, root `Makefile`
- Lifecycle source files
- Harness integration files
- `CLAUDE.md`, `docs/repo-contract.md`, `docs/manifest.json`, root package manifests

Tooling (preferred):
- `golangci-lint run` against the slice
- `gofumpt -l` against the slice
- `go test -cover ./...` against internal packages
- `ast-grep --lang go --pattern '<pattern>'` for structural checks

Context:
- `shared/PLATFORM_AUDIT.md`
- `docs/infra-health/RELIABILITY_TARGETS.md`
- `docs/infra-health/INSTRUMENTATION_ROADMAP.md`
- `docs/infra-health/CROSS_PLATFORM_LEDGER.md`
- `prompt-manager team decision-list infra-health --status=pending --by=platform-code-auditor`
- `prompt-manager team knowledge-list infra-health --topic-prefix=platform-audit-`

## Usage Rules
- One slice per heartbeat. Honor the rotation unless a runtime-health-finding overrides.
- Never edit code. Findings are decisions.
- Every grade carries an honesty flag (`measured` / `estimate`).
- Cross-platform-debt for tier 3+ is filed as a ledger entry only — do NOT propose blocking swarm-manager work for tiers that aren't on the active deployment roadmap.
- When a steer skill doesn't translate cleanly to internal code, note it in the audit log so meta-optimization can refine the skill.
- Cap decisions at 2 per heartbeat.
- `capability-gap` decisions name the exact CLI verb / tooling shape proposed and which scenario should host it.
