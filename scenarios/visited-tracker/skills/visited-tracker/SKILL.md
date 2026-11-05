---
name: "visited-tracker"
description: "Choose revision-aware review work, coordinate claims, and record completion without confusing visits with correctness."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["visited-tracker", "attention", "composition"]
  icon: "route"
  status: "active"
  revision: 2
  createdAt: "2026-09-06T00:00:00Z"
  updatedAt: "2026-09-06T13:45:00Z"
  requires: {"scenarios": ["visited-tracker", "program-runtime", "vrooli-memory", "prompt-manager"], "commands": ["visited-tracker attention preview", "program-runtime library run", "prompt-manager skill read"]}
  origin: {"kind": "authored"}
  learning: {"scope": "visited-tracker-usage", "capture": "every attempt"}
---
## Tools focus: Visited Tracker

Choose revision-aware review work. A visit records attention, not correctness. A claim reserves work, not permission to edit it. The caller keeps its objective, heuristics, and repair authority.

Read `prompt-manager skill read vrooli-memory` for the learning mechanics. Role and rung definitions are in `path:docs/agent-system/SKILL_AUTHORING.md` §Scenario skill sets.

Before acting, run `vrooli-memory recall wake --scope visited-tracker-usage`. Apply prior evidence about the campaign scope and review purpose.

| Need | Next step |
|---|---|
| Establish an authorized directory and review purpose | `visited-tracker attention ensure-campaign` with location, tag, patterns, and a bounded file limit. Reuse the same scope; incompatible settings return a conflict. [S1] |
| Rank candidate files without consuming work | Run `visited-tracker.attention-select` with campaign_id and limit. Use candidates only when status is ok. [S3] |
| Coordinate concurrent workers | `visited-tracker attention claim` with campaign-id, worker, and a stable request-id. Retain the returned claim ID, path, revision, and expiry. [S1] |
| Finish the exact claimed revision with recorded evidence | `visited-tracker attention complete` with outcome reviewed and an evidence reference. [S1] |
| Stop without finishing | `visited-tracker attention complete` with outcome incomplete or skipped. Neither grants review credit. [S1] |
| Record legacy browsing history | `visited-tracker visit`. Do not use visit totals as review evidence. [S1] |
| Repair this capability or its programs | Read `prompt-manager skill read visited-tracker-improve`. [S0] |

Compose selection as `lib.visited_tracker.attention_select(campaign_id=..., limit=...)` from any authorized scenario program. Check the child envelope and artifact identity before acting. Claim through the typed `visited_tracker.attention.claim(...)` binding before parallel work. Complete through `visited_tracker.attention.complete(...)` only after the caller has evidence. Inspect the claim even on a request replay: it can already be completed or expired. No-work is a present observation, not a permanent exhausted verdict.

A new content revision regains attention. Successful review reduces its score; age raises it again. Every fourth new claim selects the least recently attempted eligible file. For a fixed set of N continuously eligible files, each receives an attempt within 4N successful new claims, including when higher-priority work repeatedly ends incomplete. Replays do not advance this cursor. This is an attempt bound within one campaign; callers still own mandatory checks, cross-campaign scheduling, and completion evidence. Increase file priority only from evidence about impact or a known problem; do not lower it to hide unresolved work.

These one-operation CLI leaves stay S1: the state transition belongs to the scenario. Promote a recurring multi-operation shape after recording repeatable evidence. Agents may edit scenario skills and programs within authorized scope. Follow `skill-set-authoring` and Program Runtime's construction and contract guides.

| Symptom | In-use setting move | Capture |
|---|---|---|
| Review cannot finish in the current reservation | Use an appropriate ttl-seconds on a new claim, at most 3600. Do not complete an expired claim. | Duration and evidence of the timeout |
| Preview is too broad | Reduce limit, at most 20 through the program. | Selected scope and omitted work |
| Scan exceeds its file limit | Create a narrower purpose tag and patterns through ensure-campaign. | Why the partition preserves required coverage |

After acting, always run `vrooli-memory journal note --scope visited-tracker-usage --kind task-record` with trigger, approach, claim/program identity, evidence, and outcome. Capture cross-run lessons; the claim ledger already owns per-file completion. Use machine JSON only when passing IDs or envelopes into another tool; use the human CLI view for inspection.

### Troubleshooting & Edge Cases

| Symptom | First check | Response |
|---|---|---|
| Program unavailable | `vrooli scenario status visited-tracker` | Start through the lifecycle within scope; unknown is not zero. |
| Program refused | Envelope error class and exact binding grant | Use the governed session grant path; do not bypass with a private HTTP call. |
| Completion conflicts | Worker, claim revision, current file, exclusion | Re-read; changed or excluded work cannot receive stale credit. Release incomplete work where still valid. |
| Lost response | Original request-id, worker, and TTL | Replay the same request. It returns original ownership, including expired/completed claims. |
| Scan error | Root availability, permissions, symlinks, file size | Restore a readable authorized scope; failed scans do not apply deletions. |
| Claim history grows | Campaign purpose and retained evidence | Terminal claims are automatically moved to a bounded replay archive. Replays remain stable while retained; create a new purpose-scoped campaign when a ledger has outgrown its retained history. |
| Legacy write returns HTTP 409 | Current campaign revision | Reload before retrying the intended mutation; never overwrite a newer detached value. |
| Request expires while another writer owns storage | Cancellation or deadline result | Lock waits honor the request lifetime. A cancelled waiter does not write; retain the original request ID when retrying a claim whose response was lost. |

Legacy sync patterns extend the campaign scope; they do not narrow it. Create a new purpose-scoped campaign to narrow future work.

Files are hashed with a 64 MiB per-file and 128 MiB per-scan bound. Snapshot history retains 100 entries. Unambiguous content-preserving renames retain identity; ambiguous renames and deleted-file recreation get new identities. Do not import review credit: imported visits are history, and imported files require fresh review.
