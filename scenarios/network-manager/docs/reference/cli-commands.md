# CLI Commands — Network Manager

The scenario CLI is a thin Go wrapper over the API. Commands should not contain business logic.

## Global flags (provided by cli-core)

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override API endpoint. |
| `--auto-start` | Start the scenario if the API is unreachable. |
| `--json` | Emit machine-readable JSON. |
| `--no-color` | Disable color. |
| `--color` | Force color. |
| `--help`, `-h` | Show help. |
| `--version`, `-v` | Show version. |

## Built-in commands (auto-provided by `cli-core`)

### `network-manager status`

Calls `GET /health` and renders scenario readiness.

```bash
network-manager status
network-manager status --json
```

### `network-manager configure <key> <value>`

Persists CLI config such as `api_base` or token values.

## Scenario commands — Network Manager

Implemented command groups:

| Command | Purpose | Requirement |
|---|---|---|
| `network-manager snapshot run` | Run a read-only health snapshot. | `NM-P0-001` |
| `network-manager snapshot list` | List stored snapshots. | `NM-P0-001` |
| `network-manager snapshot get <id>` | Get a stored snapshot. | `NM-P0-001` |
| `network-manager snapshot export <id>` | Export a health report. | `NM-P0-001` |
| `network-manager resolver status` | Show AdGuard Home and DNS backend health. | `NM-P0-002` |
| `network-manager resolver configure-adguard` | Configure the first resolver backend. | `NM-P0-002` |
| `network-manager resolver upstreams` | Preview or update resolver upstreams. | `NM-P0-002` |
| `network-manager resolver health` | Run resolver health checks. | `NM-P0-002` |
| `network-manager policy preview` | Preview filtering changes. | `NM-P0-003` |
| `network-manager policy apply` | Apply an approved policy change. | `NM-P0-003` |
| `network-manager policy rollback <id>` | Roll back a policy change. | `NM-P0-003` |
| `network-manager policy pause` | Pause filtering for a target. | `NM-P0-003` |
| `network-manager policy resume` | Resume filtering for a target. | `NM-P0-003` |
| `network-manager policy profiles` | List household policy profiles. | `NM-P1-001` |
| `network-manager policy profile-set` | Store household policy profile intent. | `NM-P1-001` |
| `network-manager policy schedule <profile-id>` | Evaluate a profile schedule for a target without live mutation. | `NM-P1-002` |
| `network-manager policy bypass-guidance` | Diagnose IPv6, DoT, DoQ, and DoH bypass guidance without live mutation. | `NM-P1-004` |
| `network-manager policy doh-guidance` | Generate endpoint/browser DoH policy guidance without invasive inspection. | `NM-P1-008` |
| `network-manager devices refresh` | Refresh LAN-visible device inventory. | `NM-P0-004` |
| `network-manager devices list` | List devices with identity confidence. | `NM-P0-004` |
| `network-manager devices group <id>` | Assign a device group. | `NM-P0-004` |
| `network-manager devices explain <id>` | Explain identity confidence. | `NM-P0-004` |
| `network-manager monitoring schedules` | List recurring monitoring schedules. | `NM-P1-007` |
| `network-manager monitoring schedule-set` | Create or update a baseline-anchored monitoring schedule. | `NM-P1-007` |
| `network-manager monitoring run <schedule-id>` | Run a monitoring check and compare against baseline evidence. | `NM-P1-007` |
| `network-manager monitoring alerts` | List regression alerts. | `NM-P1-007` |
| `network-manager optimize run` | Run a baseline/candidate optimization experiment. | `NM-P0-005` |
| `network-manager optimize candidate <run-id>` | Run one optimization candidate. | `NM-P0-005` |
| `network-manager optimize score <run-id>` | Score optimization candidates. | `NM-P0-005` |
| `network-manager optimize approve <run-id>` | Approve a candidate apply. | `NM-P0-005` |
| `network-manager optimize rollback <run-id>` | Roll back an optimization run. | `NM-P0-005` |
| `network-manager adapters capabilities` | List supported host/resolver/router actions. | `NM-P0-006` |
| `network-manager adapters explain <action>` | Explain unsupported actions. | `NM-P0-006` |
| `network-manager adapters platform` | Show detected platform summary. | `NM-P0-006` |
| `network-manager privacy retention` | Show retention defaults. | `NM-P0-008` |
| `network-manager privacy retention-set` | Update retention defaults. | `NM-P0-008` |
| `network-manager privacy visibility` | Show visibility defaults. | `NM-P0-008` |
| `network-manager home actions` | List Home Automation actions. | `NM-P0-007` |
| `network-manager home invoke <name>` | Invoke a Home Automation action. | `NM-P0-007` |
| `network-manager home events` | List Home Automation events. | `NM-P0-007` |

Current behavior is mixed implementation state. Snapshot, adapters, resolver,
policy, inventory, privacy, optimization, and Home Automation commands call service-backed APIs with persisted state.
Policy live resolver writes still fail closed through the conservative adapter
until a governed AdGuard Home policy client confirms support. Inventory
production discovery reports unsupported until a governed resolver client can
provide client evidence, but identity reconciliation and storage are implemented.
Household policy profiles are persisted and schedules can be evaluated, but
automatic enforcement returns manual-required intent until resolver/router
capabilities prove safe live changes.
IPv6/encrypted-DNS bypass and endpoint/browser DoH guidance commands are
read-only. They produce manual steps and adapter-preview instructions where
appropriate, but they do not mutate router, firewall, browser, or endpoint
state.
Continuous monitoring commands persist baseline-anchored schedules and alerts.
Checks are operator-triggered/advisory in this slice; autonomous background
scheduling is deferred to a lifecycle-aware scheduler.
Optimization can create baseline-backed experiment ledgers, run read-only
candidate snapshots, score reliability-first evidence, require approval, and
record apply/rollback outcomes. Production persistent optimization apply returns
`manual_required` until a real adapter can prove rollback support. Home
Automation write actions require approval and return `manual_required` when no
supported publisher/adapter path can safely mutate network state.

## Output contracts

| Contract | Used by | Structure |
|---|---|---|
| Operational | `status`, `resolver status`, `adapters capabilities` | Status -> Triage -> Next Steps |
| Data Retrieval | `snapshot list`, `devices list` | Summary -> Results -> Retrieval Hints |
| Mutation | `policy apply`, `optimize approve`, `rollback` | Result -> What Changed -> Next Command |

## Adding a new command

1. Add or confirm the API endpoint first.
2. Add a command entry to `cli/manifest.json`.
3. Implement the handler as an API call and report renderer.
4. Add command tests tagged with the relevant `[REQ:ID]`.
5. Update this document.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md)
- [`configuration.md`](configuration.md)
- [`../internal/TESTING.md`](../internal/TESTING.md)
