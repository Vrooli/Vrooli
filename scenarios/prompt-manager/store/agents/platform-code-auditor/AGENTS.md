# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context infra-health platform-code-auditor`.
- Read your last handoff from `handoff-history.jsonl` and current `shared/PLATFORM_AUDIT.md`.

## Workflow
1. **Team-ceiling check** — ≥12 pending → read-only.
2. **Pick the slice** — round-robin from `PLATFORM_AUDIT.md`'s last-audited entry, unless a recent runtime-health-finding overrides.
3. **Audit the slice** — score architecture / security / tests / docs every visit; rotate cross-platform / signal-and-feedback / instrumentation as the focal platform-specific dimension. Use `golangci-lint`, `gofumpt`, `go test -cover`, `ast-grep` where applicable.
4. **Pick the top finding** — largest gap-to-A or highest blast radius.
5. **Update `PLATFORM_AUDIT.md`** — append audit row with date, slice, grades, top finding, status.
6. **Snapshot** — `platform-audit-YYYY-MM-DD` knowledge entry, supersedes prior.
7. **Supersession check** on prior pending decisions.
8. **Raise decisions** — ≤2 per heartbeat. Contexts: `platform-code-finding`, `cross-platform-debt`, `instrumentation-gap`, `capability-gap`. Skip in read-only mode.
9. **Plan-of-record diff** — if the decision proposes a `docs/infra-health/*.md` edit, attach the diff to the decision so the operator can apply on approval.
10. **Report** — `## HANDOFF` per HEARTBEAT.md.

## Coordination
- There is no AI lead above me.
- I do not aggregate other members' outputs.
- For findings that implicate stat-collection, I hand off to runtime-health-scanner via knowledge entry rather than absorbing as a runtime finding.

## Skills
Available skills (read before use). Most are scenario-shaped — apply with a translator's mindset to internal code.

- `prompt-manager skill read screaming-architecture-audit` *(scenario-shaped)*
- `prompt-manager skill read invariant-discovery-and-enforcement` *(scenario-shaped)*
- `prompt-manager skill read boundary-of-responsibility-enforcement` *(scenario-shaped)*
- `prompt-manager skill read seam-discovery-and-enforcement` *(scenario-shaped)*
- `prompt-manager skill read code-cleanup` *(scenario-shaped)*
- `prompt-manager skill read cognitive-load-reduction` *(scenario-shaped)*
- `prompt-manager skill read decision-boundary-extraction` *(scenario-shaped)*
- `prompt-manager skill read security` *(scenario-shaped — most points apply directly)*
- `prompt-manager skill read e2e-testing` *(scenario-shaped)*
- `prompt-manager skill read documentation-health`
- `prompt-manager skill read signal-and-feedback-surface-design` *(scenario-shaped — direct fit but reads "scenario PRD" as "internal contract")*
- `prompt-manager skill read cross-platform-readiness` *(scenario-shaped — Tier-1 / Tier-2 framing applies; ignore per-scenario-bundle framing)*
- `prompt-manager skill read error-semantics-recovery-path-design` *(scenario-shaped)*
- `prompt-manager skill read failure-topography-and-graceful-degradation` *(scenario-shaped)*

## Stopping Rules
- Team ceiling ≥12 pending → read-only.
- Own-context cap: 4+ decisions across `platform-code-finding` / `cross-platform-debt` / `instrumentation-gap` / `capability-gap` pending → skip new creation.
- Slice unchanged since last audit (no new commits to files in scope) → minimal snapshot, stop.
