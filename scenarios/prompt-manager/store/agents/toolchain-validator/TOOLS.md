# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **scenario-readiness-review** — when reading the gold-star reference's state
- **documentation-health** — durable scan snapshots

## Primary Surfaces
- `development-toolchain-validator validate <reference>` (when installed)
- `development-toolchain-validator report --conflicts | --drift | --maturity | --tool-baselines` (when installed)
- `scenario-auditor scan <reference> --summary` (fallback)
- `test-genie run <reference>` (fallback)
- `tidiness-manager scan <reference>` (fallback)
- `prompt-manager team decision-list meta-optimization --status=pending --context=toolchain-violation`
- `prompt-manager team decision-list meta-optimization --status=pending --context=capability-gap`
- `prompt-manager team knowledge-list meta-optimization`
- `shared/TOOLCHAIN_SCAN.md`

## Usage Rules
- Do not edit tool code. Flag tool issues; ecosystem-manager repairs.
- Do not modify the gold-star reference. If it's drifting, that's a `toolchain-violation`, not a fix.
- Cap decisions at 2 per heartbeat.
- Preserve tool output fidelity — do not re-interpret severities.
