# Infra Health Editing

## Authority

`docs/infra-health/` is operator-curated plan-of-record canon for the `infra-health` team. Agents read it every heartbeat, but they do not edit it directly during normal team operation.

The team-owned runtime state lives under `scenarios/prompt-manager/store/teams/infra-health/`. Rolling snapshots and logs in that store are working state, not durable PoR canon.

The team contract's `operatingContract.documents.planOfRecord.paths` lists the *consumed-canon subset* agents read each heartbeat (README, operating model, reliability targets, instrumentation roadmap, cross-platform ledger). The governance docs here — `editing.md`, `adoption-validation.md`, `changelog.md` — are required by [`manifest.json`](../manifest.json) for PoR structure but are intentionally omitted from `planOfRecord.paths`, matching the fleet convention (e.g. meta-optimization).

## Change Flow

1. An infra-health member observes a material reliability, instrumentation, platform-code, or portability signal.
2. The member writes evidence to the appropriate knowledge topic or rolling snapshot.
3. The member raises the smallest relevant work type.
4. The operator accepts, rejects, or requests revision.
5. The operator applies accepted PoR edits and cites the decision id.

Common edit contexts:

| Context | Typical PoR target |
|---|---|
| `reliability-target-update` | `strategy/RELIABILITY_TARGETS.md` |
| `instrumentation-gap` | `evidence/INSTRUMENTATION_ROADMAP.md` |
| `cross-platform-debt` | `evidence/CROSS_PLATFORM_LEDGER.md` |
| `capability-work` | `evidence/INSTRUMENTATION_ROADMAP.md` or downstream Swarm Manager work |
| `framework-meta` | `operating/OPERATING_MODEL.md`, `governance/`, or future manifest updates |

## Direct Edits

Direct agent edits to plan-of-record canon are not allowed during normal heartbeats. The only direct-edit infra-health state is team runtime state under `scenarios/prompt-manager/store/teams/infra-health/`.
