# Effort workspace contract

The workspace is the leader's durable source and review desk. Keep it outside scenario source and outside generated native skills. Resolve `runtime_home.dir_name` and `runtime_home.entries.plan_artifacts.path` from `.vrooli/repo-contract.json`; use `<runtime-home>/<plan-artifacts>/efforts/<slug>/`. This existing protected class retains sources cited by plans. A Plan Manager supplied artifact location takes precedence when one already owns this effort. Do not create a new runtime-home class or store irreplaceable intent in cache.

Use the canonical helper for a new folder or a structural review:

```bash
python3 scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/scripts/workspace.py init --repo <repo> --slug <slug>
python3 scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/scripts/workspace.py validate <effort-path>
python3 scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/scripts/workspace.py report <effort-path>
```

The helper manages local review artifacts only. It does not dispatch agents, enforce runtime grants or decide that remote tests passed. Its derived circuit report is an admission input until the owner implementation enforces that policy.

Preserve this class during manual cleanup and disk-pressure automation. A broad cleanup of a parent directory must not remove the protected descendants. Qualification uses temporary fixtures and read-only proposals; never test protection by deleting the real effort folder. Protection is not a backup: retain the platform's independent backup/custody references when available.

## Minimum useful contents

| Artifact | Meaning and writer |
|---|---|
| `README.md` | Human entrypoint: outcome, current stage, review order, authoritative links and next action. Leader-owned. |
| `effort.json` | Identity, repository, authority status, policy and owner references. Leader-owned; approved authority changes require the actual authority. |
| `sources/` | Preserved user intent, accepted decisions and immutable evidence snapshots. Retain a content digest at approval. |
| `requirements.json` | Stable source requirements, deliverables, acceptance, owner plan references, and evidence assessments. An assessment is not a remote plan status. |
| `findings/` | Bounded investigations; label fact, hypothesis and recommendation. Link source and capture time. |
| `capabilities.json` | Needed operations and last qualification evidence. This is an effort observation, not a replacement global capability registry. |
| `recovery.jsonl` | Append-only repair intent/outcome events, keyed by attempt, component and fingerprint. Keep owner incident/issue references. |
| `plans/` | Owner receipts, review exports or clearly labelled candidates. Edit authoritative plans through Plan Manager. |
| `team/` | Proposed or actual team/member references, leader mandate and prompt inputs. Draft configurations do not imply activation. |
| `programs/` | Selected program references, qualification evidence and candidate contracts. Deployed program source belongs to its owner. |
| `handoffs/`, `review/`, `evidence/` | Next-action checkpoints, independent assessments and links/receipts from evidence producers. |

Create optional directories only when they have content. Do not put credentials or private identity tokens into this workspace. Source preservation does not require copying transcripts with unrelated private content.

`requirements.json` rows have `id`, `source`, `statement`, `deliverable`, `acceptance`, `owner_plan`, `assessment`, `evidence`, and optionally `depends_on`. Assessment is `unverified`, `met`, `unmet`, or `waived`; `met` requires evidence, and `waived` requires an actual user decision reference. Keep recommendations separate from user requirements.

Use one coordinator writer for aggregate artifacts. Workers write their own result paths and owner records. A fallback coordinator must prove exclusive ownership before editing shared control state. A local JSON file is not a distributed lock. Record migration to an owner state API explicitly; after migration, retain only its reference/projection here.

## Planner tree and economical assignments

Record a root planner and optional narrower planner branches. A worker has one assigning parent and one bounded assignment. Represent parent/task/attempt IDs and the selected context in the owner APIs; the folder retains references. Review is a transient worker assignment, not a permanent judge role. The root retains overall acceptance responsibility and delegates substantive implementation instead of becoming a second writer on each child's files.

Set one maximum tree depth, one aggregate active-agent ceiling (including subplanners and reviewers), and a separate scarce-model ceiling. All descendants reserve the same effort allowance. A planner does not receive another copy of its parent's budget. Add branches only where distinct ownership and sufficient ready work justify the coordination cost.

The model policy records preferred economical worker profiles, permitted effort levels, stronger planning/escalation profiles, capability requirements, allowed provider/runner fallbacks and qualification receipts. Keep credentials in their owner store; use only credential-pool references here. A configured profile, a catalog listing, authenticated access and observed accepted output are separate evidence states. Avoid baking model names or tariffs into this reusable skill; resolve them in the effort's policy and owner catalog.

Worker handoffs retain task/attempt/parent identity, result disposition, source revision, changed boundaries/artifact references, validation evidence, findings/deviations, remaining work, actual or unknown usage, and pending owner operations. Store one immutable handoff identity; delivery retry does not create another result. Progress/limit events can update owner state before that handoff. A quota-interrupted attempt remains resumable or pending, not falsely complete. The parent chooses the next assignment after reading the evidence.

Parent-directed handoff governs work coordination, not custody of the produced material. A worker may write its assigned campaign/artifact/code/evidence records through their owner APIs. It may not assign sibling work, change another worker's claims, or mutate the family admission policy. The assigning parent and deterministic owner admission mechanism retain those decisions.

## Approval and recurring team policy

Keep planning authorization separate from execution authorization. For review-stage efforts, `execution.status` remains `not-approved` and `schedule.enabled` remains false. Record the requested cadence, overlap rule, completion predicate and selected route as proposed policy. The leader's wake prompt must reference this folder and the authoritative family rather than contain another task ledger.

Approval binds the destination/acceptance revision, allowed effects, exclusions, resource budget, component repair caps, fallback routes and final disposition. Preserve a digest of that approved material. The leader may improve working strategy inside that boundary without another approval. New requirements or excluded effects require an amendment; capability outages alone do not erase the boundary.

The finite team contains a root planner, optional subplanners and temporary bounded workers. Independent review uses a worker with the required review context. Reuse suitable agent identities and member composition rather than clone permanent marketing-team members. Configure the shortest resume cadence useful for the approved workload; cadence belongs to effort policy, not this reusable skill. Skip ticks for queued, running, parked or uncertain leader runs. A parked owner watch should wake the existing run. Quota-blocked work is reconsidered only when its owner eligibility condition changes.

Before closure, verify the requirement evidence, validate retained limitations against the actual acceptance policy, and disable recurring work. A failed retirement operation leaves closure pending. Keep live marketing capability/artifact state with Content Desk and related owners after the temporary effort retires.
