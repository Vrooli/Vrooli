# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **scientific-debugging** — sharpen "is this finding actually load-bearing?" reasoning
- **documentation-health** — aging-scan and contrarian-snapshot writeups
- **assumption-mapping-and-hardening** — surface which assumptions a finding rests on *(scenario-shaped — translate to internal-code framing)*
- **change-axis-and-evolution-resilience-audit** — spot polishing dressed as reliability work *(scenario-shaped)*

## Primary Surfaces
- `prompt-manager team decision-list infra-health --status=pending --json`
- `prompt-manager team decision-get infra-health <decision-id>` for full context
- `shared/RUNTIME_LESSONS.md`
- `shared/PLATFORM_AUDIT.md`
- `shared/AGING_SCAN.md`
- `docs/infra-health/RELIABILITY_TARGETS.md`
- `docs/infra-health/INSTRUMENTATION_ROADMAP.md`
- `docs/infra-health/CROSS_PLATFORM_LEDGER.md`
- Prior `infra-contrarian-*` knowledge entries

## Usage Rules
- Cap review at 5 decisions per heartbeat. Beyond that, defer.
- Cap challenges at 2 per heartbeat (own-context cap).
- Cap `framework-meta` at ONE per calendar month.
- Every challenge names a specific failure mode + at least one specific evidence point.
- "Challenged-and-passed" is a valid outcome and must be recorded — silent passes erase the discipline.
- I never edit code or plan-of-record docs.
- I never raise `runtime-health-finding`, `platform-code-finding`, `instrumentation-gap`, `cross-platform-debt`, `capability-gap`, or `reliability-target-update`. Only `decision-rejection-proposed` and `framework-meta`.
