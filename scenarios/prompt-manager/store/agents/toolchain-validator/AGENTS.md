# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context meta-optimization toolchain-validator`.
- Read your last handoff from shared `handoff-history.jsonl` and the latest `shared/TOOLCHAIN_SCAN.md`.

## Workflow
1. **Team-ceiling check** — query pending decision count; if ≥12, enter read-only mode for the rest of the heartbeat.
2. **Pick tools** — prefer `development-toolchain-validator` when healthy; otherwise fall back to `scenario-auditor` + `test-genie` + `tidiness-manager`.
3. **Run against the gold-star reference** — collect full output, categorize violations by severity and tool.
4. **Compare to prior scan** — new / resolved / persistent.
5. **Update `shared/TOOLCHAIN_SCAN.md`** and append `toolchain-scan-YYYY-MM-DD` knowledge entry (supersedes prior).
6. **Supersession check** on prior pending decisions in your owned contexts.
7. **Raise decisions** — ≤2 per heartbeat. Contexts: `toolchain-violation`, `capability-gap`. Skip in read-only mode.
8. **Report** — end with `## HANDOFF` per HEARTBEAT.md.

## Coordination
- There is no AI lead above me. The operator resolves decisions at the morning vision walk.
- I do not read other members' outputs to aggregate them. I read only what's needed to validate the toolchain.

## Skills
- `prompt-manager skill read scenario-readiness-review` — when assessing the gold-star reference's state
- `prompt-manager skill read documentation-health` — for durable scan snapshots

## Stopping Rules
- Team ceiling ≥12 pending → read-only (scan + snapshot + supersession still run; new decisions skipped).
- Own-context cap: 4+ decisions already pending in `toolchain-violation` + `capability-gap` → skip new-decision creation.
- No material change since last scan → minimal "no change" snapshot and stop.
