# Infra Health Editing

## Authority

`docs/infra-health/` is operator-curated plan-of-record canon for the `infra-health` team. Agents read it every heartbeat, but they do not edit it directly during normal team operation.

The team-owned runtime state lives under `scenarios/prompt-manager/store/teams/infra-health/`. Rolling snapshots and logs in that store are working state, not durable PoR canon.

The team contract's `operatingContract.documents.planOfRecord.paths` lists the *consumed-canon subset* agents read each heartbeat (README and the operating model). The reliability targets, instrumentation roadmap and cross-platform ledger that were once in that subset are retired: each is now a computed surface, and the paths that pointed at them were dropped rather than repointed. The governance docs here — `editing.md`, `adoption-validation.md`, `changelog.md` — are required by [`manifest.json`](../manifest.json) for PoR structure but are intentionally omitted from `planOfRecord.paths`, matching the fleet convention (e.g. meta-optimization).

## Change Flow

1. An infra-health member observes a material reliability, instrumentation, platform-code, or portability signal.
2. The member writes evidence to the appropriate knowledge topic or rolling snapshot.
3. The member raises the smallest relevant work type.
4. The operator accepts, rejects, or requests revision.
5. The operator applies accepted PoR edits and cites the decision id.

Common edit contexts:

None of the four evidence decisions targets a file in this folder any more. Each writes
to the surface that computes the thing it is about, which is why an approved decision can
no longer create a hand-maintained list that disagrees with its own grid.

| Context | Target | Not a PoR edit because |
|---|---|---|
| `reliability-target-update` | `scenarios/infrastructure-manager/setpoint/reliability-setpoint.json` | The bar is checked-in operator data graded live; a prose copy could disagree with it. |
| `instrumentation-gap` | The owning layer's `docs/spaces/<projection>-space.md` — open a cell as `MISSING` with its `gap_opened_on` | The gap list is the computed open-loop set (`coverage open-loop`), not a second list. |
| `cross-platform-debt` | The team's knowledge topics, read against `vrooli capability ledger`; the durable judgment half moves to the instrument's `portability` domain when it ships | Per-capability, per-OS state is computed from the manifests. |
| `capability-work` | Downstream Swarm Manager work | Never was a PoR edit. |
| `framework-meta` | `operating/OPERATING_MODEL.md`, `governance/`, or manifest updates | The only context that still edits this folder. |

## Direct Edits

Direct agent edits to plan-of-record canon are not allowed during normal heartbeats. The only direct-edit infra-health state is team runtime state under `scenarios/prompt-manager/store/teams/infra-health/`.
