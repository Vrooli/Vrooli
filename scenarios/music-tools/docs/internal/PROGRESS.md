# Progress — Music Tools

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/music-tools/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
Copied July 7 template history was removed from this scenario’s log.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append concise milestones when work lands. Keep detailed execution receipts
with the owning plan.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-09-18 | agent | done | Composition tier proven outside the scenario. ACE-Step 1.5 turbo measured on the reference host at 17.9 s per 45 s of audio (55.2 s CPU-offloaded, ~7.0 GiB peak), validating the 2026-08-19 model choice. Found and fixed the planner caption-rewrite defect. No scenario code was written — the spike ran in a throwaway venv and the recipe lives in the `music-generation` core-pack skill until it can move here. |
| 2026-09-18 | agent | done | Closed the last P1 orphan: `requirements/10-delivery-ops` covers `OT-P1-003` with standards-conformant loudness measurement, platform-target normalisation with true-peak ceiling, and CPU-only reference mastering. Scoped to the delta beyond MUS-P0-002, which asserted only that deterministic ops need no model — not that their output is correct. Business-phase intent debt 3 → 2; remaining orphans are both P2. |
| 2026-09-18 | agent | done | PRD revised on operator instruction (the template's read-only policy was overridden deliberately). Batch-of-takes made the defining contract; added OT-P0-007 (style library, promoted from OT-P1-004), OT-P0-008 (batch and selection), OT-P0-009 (provenance), OT-P1-005 (maintained inventory); launch sequencing inverted to composition-first on spike evidence; the two out-of-scenario preconditions named. New `requirements/09-batch-selection` covers all four, cutting PRD orphans from 4 to 3. |
| 2026-09-18 | agent | done | Doc pass folding the spike back in: recorded four decisions (batch-of-takes contract, caption ownership, no hosted rung, batch/inventory scope), added `takes` to the schema map, replaced the empty measurements table, marked the CLI and API references as unimplemented with their planned surfaces, and filed the idle-scheduling platform gap. |
| 2026-09-19 | agent | partial | Shipped and validated the governed ACE-Step managed resource: checksum-pinned offline closure, activity-edge compose boundary, full/offload lifecycle rungs, eager readiness, stable artifact digest, and live offload WAV receipt. Expanded composition provenance/API/CLI transitions, added styles/audition/pool UI routes with tests, and retired the manual music-generation setup skill. Remaining plan work is infrastructure-domain completion, SFT measurement, and experience/UI-health cleanup. |
| 2026-09-19 | agent | partial | Measured ten SFT jobs through the shipped pipeline with preserved briefs/seeds and valid WAV receipts; turbo remains default because no grounded quality advantage was established. Reconciled capacity with ACE-Step claimed and healthy. Business and broad Test Genie validation were rerun, but the provider was unavailable for business and the broad suite retained pre-existing UI/proto/quality/provider findings; these remain explicit audit limitations. |
| 2026-09-19 | agent | done | Closed the composition-slice intent spine: shipped requirement references now resolve to real tests with valid registry statuses, explicitly deferred domains use roadmap/manual validations, and P2 targets have coverage. Business phase passed with zero ERROR/BLOCKER findings. The broad collection passed 16/27 phases; remaining failures are logged as unrelated scaffold/provider debt. |

## 2026-10-17 — Managed MiniMax Music3 and HeartMuLa

Integrated explicit `minimax-music3-nf4` and `heartmula-3b` variants into the
existing style → durable job → take → owned blob route. One Songgen resource
shares transport, fencing, receipts and retention while keeping the pinned
inference dependencies isolated. New jobs freeze exact provider/runtime/style
identity and refuse receipt or actual sampling mismatches. Defaults remain
SA3 for instrumentals and ACE for implicit lyric requests. Updated canonical
music-generation skill revision 8 preserves the listener's separate realism
and style judgments and does not claim reference-voice support.

Both native 35-second verification jobs succeeded with matching exported WAV
hashes. HeartMuLa peak Torch allocation was 8,577,632,256 bytes; MiniMax NF4 was
7,403,018,240 bytes. Workers exited between jobs and no Songgen GPU process
remained afterward. The final retention/recovery patches have focused reacquisition
regressions and recorded unchanged inference-input hashes; it does not rerender
the already-qualified inference code. Exact jobs, takes, receipts, source/preview
audio and the artifact delta are retained in the protected effort directory
`production/continuation/music/song-model-integration-v1`.

Control-plane work adds music lifecycle exclusivity, refuses shared-broker
bypasses, protects running runtime artifacts from pruning, and gives ACE/SA3
last-use cache ownership. Corrected proxy stop/port-release ordering, explicit
idle placement, and serving CPU-rung startup without hiding drift or failed
liveness probes. Cache cleanup preserves pins, new downloads, unexpected files,
operator exports and concurrent acquisitions; interrupted deletion remains
retryable. Explicit data-group acquisition prevents ordinary setup from
redownloading a profile retired by the unused-model policy. Startup checks
complete group presence and expected sizes before process launch. Superseded
local candidate runtime trees are removed, with their
receipts and source snapshots retained.

Focused Go/resource/storage checks passed. Music Test Genie
`20261017-225733-5e968194` passed unit and API phases. The final scoped
control-plane run passed 39 regression cases, and Songgen passed 20 Python
regressions plus its Go CLI tests. Streaming atomic publication avoids reading
entire vendored weight files into RAM. Earlier broad baseline
findings and advisory-capacity limits are explicitly retained in PROBLEMS.md;
this work does not claim whole-repository certification or listener approval of
new tracks. All music resources are stopped after verification; selected weights
remain cached and Ollama's normal configuration is restored. The final evidence
manifest records managed adoption, cleanup, and two explicit self-review passes.

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
