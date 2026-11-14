# Problems — Music Tools

## Completed song-model integration and retained limits — 2026-10-17

Work shape: bounded convergence task. W0 target extension is authorized by the
operator's acceptance of MiniMax and HeartMuLa; older categorical rejections in
the model registry and decision log are historical, not current prohibitions.
Implementation (W3) must meet the new target in
`../concepts/ARCHITECTURE.md#accepted-song-model-integration-target--2026-10-17`.

Harness handoff: work until both selected models run through the durable music
pipeline and the lifecycle/retention acceptance criteria in that target are
proven by focused resource/provider/control-plane tests and fresh managed
generation receipts. Continue through diagnosable friction. Track findings here;
run two fresh adversarial reviews after checks pass. Do not widen into UI
redesign or unrelated host services; never stop coding agents. No native harness
goal or separate plan is required for this bounded assignment.

Operator feedback (verbatim):

> ACE take2 wins. It's not quite the  quality I'm hoping for. It still sounds like AI and isn't as good as a real KiiiKiii song. But it does have the same vibe as that artist and sounds sorta like them. The "MiniMax quantized output-head trial — seed170104" one may win in terms of song quality. It sounds the least like AI and might even fool most people. But it doesn't sound like the type of song we are going for, nor does the voice match. Not sure if that's how you created it or some limitation with minimax. But based on these results, I think it would he worth properly adding minimax to vrooli the same way we did with ace and that other music model we added. Not sure about the other models, but maybe? We should definitely make sure that our music code properly handles cleaning them up and stuff so that ideally only one runs at a time. And in general the code should probably be more mature and professional and reliable and all that so that we can use these models in a consistent, powerful way to get high quality music.
>
> We can add HeartMuLa too if you want. Might as well, as long as we do in fact clean it up properly. Probably need a better way to decide when to do a disk cleanup of these models. I think our resource capacity stuff can turn off heavy things like ai models when other code needs the gpu and such, but it's largely the responsibility of scenarios to decide how to manage their resources' disk usage. We don't want to have to download these models often, but at the same time we should probably remove them when not in use after a while.
>
> And if you have more ideas for making the music generation code more professional and clean and reliable and powerful and all that, go right ahead

Implemented and proven: pinned managed acquisition; isolated MiniMax/HeartMuLa
adapters; declared `song-v1` provider dispatch; immutable queued model snapshots;
actual sampler/seed/version/audio receipt validation; control-plane exclusive
music groups; owner-fenced last-use retention and pins; cancellation/reaping;
and fresh native Music jobs for both profiles. Twenty resource regressions
include partial deletion followed by reacquisition. The final cleanup/recovery patches
keep the qualified inference inputs unchanged; the artifact delta is recorded.

Native proof: HeartMuLa job `6e823005-56fb-4d00-8927-746831dd834e` and MiniMax job
`4d6c302e-eb38-4f48-b149-402d4bcf193d` produced verified 35-second takes. Music
Test Genie `20261017-225733-5e968194` passed unit and API phases. The earlier
proto run `20261017-220838-76c3abbe` retains four unrelated capabilities/cleanup
payload declaration errors; the touched model declarations were repaired.
QA record: `knw-1792274731037093159`. Broader resource baseline archive-fixture
and Kokoro parity failures are recorded as `knw-1792275244610914537`.

Operational limits: the host capacity policy is advisory. Ollama's model API
understates observed CUDA overhead (`knw-1792275243130357759`); a music worker
refuses insufficient free VRAM without publishing a take. The qualification
used a temporary Ollama CPU rung and restored its normal configuration. A
lightweight Songgen supervisor retains a logical resource reservation until
normal stop, although each worker releases physical model allocations on exit.
Stop the session's resource when finished. Model verification and cold loading
can take minutes; these are bounded quality-oriented recipes, not a throughput
claim. Neither new profile accepts a native reference voice.

Retention is eligibility, not a background timer: a new Songgen job sweeps
expired peer profiles, another music-member activation sweeps eligible legacy
bundles, and owner previews/pins remain explicit. Generated ACE support files
are declared separately from acquired weights. Unknown files and unknown
lifetimes remain protected. Scratch storage points to its actual data directory
and uses a one-day age floor above the maximum worker lifetime; active stages
are not exposed to a byte-pressure deletion policy.
On-demand data groups prevent ordinary restart from undoing model eviction.
`vrooli resource acquisition data songgen --group PROFILE` is the sole explicit
weight acquisition route; it verifies only the declared group under lifecycle
and data fences. Startup requires one complete group and fails promptly with
that recovery command when none is available. The regression removes a tiny
fixture bundle and verifies that routine setup does not download it again.
Vendored weight files publish through the shared atomic ownership helper after
a streamed copy and flush; they are not read back wholesale into RAM.
The final scoped control-plane run passed 39 regression cases; Songgen passed
20 Python regressions and its CLI checks. The passing Music unit phase still
reports an injectable-seam warning and coverage advisories, so passing phases
do not establish debt-free testing. All music resources are stopped at handoff,
with no music capacity claims or music GPU processes remaining.


Action-promotion follow-up: the cache-preview Action was not registered because
Prompt Manager returned unvalidated root-command governance despite the declared
leaf (`knw-1792276423714974589`). A Swarm capture attempt returned 401 verified
agent caller required; the durable work record is
`9cf51466-9753-422d-8293-7a6f7c9e9294`, and canonical skill revision 8 is synced.
Existing scenario edits from other work are preserved. Human listening feedback
does not imply a calibrated quality or identity score.

Final activation/cache preview, restoration and review evidence:
`/home/matthalloran8/.vrooli/plan-artifacts/efforts/video-toolchain-reliability-2026-10-16/production/continuation/music/song-model-integration-v1/verification.json`.


Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-09-19 — Whole-scenario validation is not green

**Symptom:** The earlier business-phase Test Genie runs
`20260919-085527-8b8fb018` and `20260919-091248-fa56f603` terminated
`provider_unavailable`. After registry repair, business run
`20260919-092125-70160574` passed with zero ERROR/BLOCKER findings. The latest
broad run `20260919-092140-17c536c3` completed with 16 phases passed and 11
failed.

**Root cause:** The broad collection exposes existing UI shell/viewport debt, proto
declaration debt, security and quality findings, and other scaffold-level findings
outside the ACE-Step and composition implementation slice.

**Workaround:** Use the passing business phase, focused Go/API/CLI tests, and the
captured live resource receipts for the implemented composition slice. Treat the
whole-scenario scorecard as non-green until the unrelated findings are repaired.

**Real fix:** Restore the unavailable providers and separately repair the broad
scenario findings; then rerun the business and whole-scenario collections.

**Owner:** scenario maintainers / owning platform providers.

**Refs:** Plan artifact validation receipt; Test Genie runs listed above.

### 2026-09-19 — Kyutai artifact drift remains after governed reacquisition

**Symptom:** `kyutai-stt` is stopped after the governed reclaim path detected an
artifact checksum mismatch. Reacquisition failed with the same mismatch.

**Root cause:** The staged artifact hash is
`52d4150654482c6c6940d03e435339cabbb5517c26152dc41a24a06c97a8c5d4`, while the
manifest expects
`c66384203a0dd4d59a95e8fe026de3b56e070f61d0c470e7dc54707903a625f1`.

**Workaround:** Leave Kyutai stopped and do not edit the manifest or bypass governed
acquisition. Kokoro, Whisper, and ACE-Step remain healthy.

**Real fix:** Repair the upstream governed artifact source or publish a new
checksum-governed Kyutai artifact.

**Owner:** resource/artifact maintainers.

**Refs:** `vrooli resource install kyutai-stt --reacquire`; Plan Manager finding
`ebf5fcb4-9c1b-4a9d-99a9-e524f4a510a7`.

### 2026-08-19 — Declared resources do not exist yet (partially superseded)

**2026-10-16 reconciliation:** ACE-Step now exists and is enabled/required in
service.json. Composition, jobs, pool and storage are implemented. music-mir and
the broader analysis runtime remain unbuilt. The historical symptom below is
not a current instruction to recreate ACE-Step.

**Symptom:** `PRD.md`, `ARCHITECTURE.md`, and `INTEGRATIONS.md` describe `ace-step`
and `music-mir` as required managed-service resources. Neither exists under
`resources/`, and `.vrooli/service.json` declares no dependencies at all.

**Root cause:** Documentation-first sequencing. The docs describe the intended
target; the wiring has not been created.

**Workaround:** None needed — but do not read the docs as a description of running
software. Nothing in this scenario executes a model today.

**Real fix:** Create both resources via `template-manager resource-template
generate`, then declare them in `.vrooli/service.json` alongside `qdrant`.

**Owner:** unassigned.

**Refs:** `.vrooli/service.json`, `docs/concepts/ARCHITECTURE.md`.

### 2026-08-19 — Upstream disagrees with itself about track separation

**Symptom:** The composition model's README feature matrix marks track separation,
layer addition, and continuation as supported on all DiT variants. Its inference
reference states these are base-model only. The variant that fits the reference
host's free VRAM is the turbo variant.

**Root cause:** Unresolved upstream documentation inconsistency.

**Workaround:** Treat separation as unavailable from the composition runtime.
`music-mir` owns it unconditionally.

**Real fix:** Test both variants directly and record the measured answer here. The
dedicated separator stays regardless — it is also the better separator.

**Owner:** unassigned.

**Refs:** `docs/reference/model-registry.md` — Operation availability.

### 2026-08-19 — The best structure-and-beats tool declares no licence

**Symptom:** The joint structure, beat, and downbeat model has no licence statement
in its repository.

**Root cause:** Upstream omission.

**Workaround:** Default-to-restricted places it in the non-commercial lane, so a
permissive-lane build already excludes it. No action needed for personal use.

**Real fix:** Obtain a licence statement upstream, or find a permissively licensed
equivalent. Until then the permissive lane has no structure analysis at all, which
is a real capability gap in a commercial build.

**Owner:** unassigned.

**Refs:** `docs/reference/model-registry.md` — Supporting tools.

### 2026-08-19 — Host capacity ledger under-reports GPU consumers

**Symptom:** `vrooli capacity reconcile` reports an unclaimed GPU consumer holding
~3.79 GiB, and four resources that declare GPU usage while holding no active claim.

**Root cause:** Pre-existing host hygiene issue, not caused by this scenario.

**Workaround:** Treat free-VRAM figures as advisory and verify with
`vrooli capacity reconcile` before sizing an operation.

**Real fix:** Bring the existing GPU resources into the claim ledger. This matters
here because this scenario will be the largest GPU consumer on the host, and its
admission verdicts inherit the ledger's accuracy.

**Owner:** unassigned — control-plane concern.

**Refs:** `vrooli capacity reconcile`, `docs/concepts/ARCHITECTURE.md` — Residency policy.

### 2026-08-19 — No throughput figure has been measured on target hardware

**Partially resolved 2026-09-18 — composition only.**

**Symptom:** Every performance number in this scenario's documentation is labelled
`vendor` or `estimated`. None was produced on the reference host.

**Root cause:** No implementation exists to measure.

**Workaround:** Treat `docs/internal/PERFORMANCE.md` budgets as hypotheses.

**Real fix:** Run the publisher's own profiling entrypoint and a structure-analysis
pass over a sample of real tracks, then replace the estimates and relabel them
`measured`.

**Progress (2026-09-18):** a throwaway spike measured the composition tier on the
reference host — 17.9 s for 45 s of audio with the DiT resident, 55.2 s offloaded,
~7.0 GiB peak. Those figures are now in `PERFORMANCE.md` and
`../reference/model-registry.md`, labelled `measured`.

**Still open, and the larger share:** analysis, embedding, structure-and-beats, and
separation have no measurement at all. Composition was the easiest tier to measure
and the least representative — it takes an exclusive lease and runs alone, whereas
the embedding pool is the always-on tier whose cost is paid continuously. Do not
let one measured tier read as a profiled scenario.

**Owner:** unassigned.

**Refs:** `docs/internal/PERFORMANCE.md`.

### 2026-09-18 — The planner LM silently rewrites the caption and invents the tempo

**Symptom:** Generated music ignores the prompt. A brief asking for "144 BPM, cold
detuned metallic lead, menacing and restless" produced a slow, bland, piano-tinged
track. Tempo requests were ignored across six generations.

**Root cause:** `GenerationParams` defaults `use_cot_caption=True` and
`use_cot_metas=True`, so the planner LM rewrites the caption and generates its own
metadata *before* the DiT sees anything. At 0.6B the rewrite was measured replacing
`bpm: 144` with `bpm: 65` and the brief above with "atmospheric... shimmering synth
pads... melancholic arpeggio... ethereal pads for reflection... a surprising
melodic piano-like figure". The DiT never received the authored prompt. Passing BPM
inside the caption string rather than the `bpm` field compounds this — free text is
exactly what the rewrite discards.

**Workaround:** Set `use_cot_caption=False` and `use_cot_metas=False`, and pass
`bpm` as a field. Measured effect at fixed seed and prompt: tempo honoured, hi-hat
band energy 19.1% -> 27.5%, onset density 6.2/s -> 9.6/s.

**Real fix:** The composition operation must treat an authored caption as authored.
Whether the planner may rewrite it is a caller-visible contract decision, not a
default to inherit — and if a rewrite happens, the rewritten caption belongs in the
operation's report alongside the output, so the caller can see what was actually
generated from.

**Owner:** unassigned.

**Refs:** `acestep/inference.py` (`use_cot_caption`, `use_cot_metas`, ~line 825);
`../reference/model-registry.md` (Composition).

### 2026-09-18 — Advisory capacity broker makes composition VRAM non-deterministic

**Symptom:** Identical generation runs alternately succeed and die with
`torch.OutOfMemoryError`, depending on what other resources happen to be warm.

**Root cause:** The control-plane broker runs `enforce = advisory` with
`preempt_enabled = false` on the reference host, so claims are recorded but never
actuated. A claim for 7 GiB returned `verdict: queue — waiting for 5 reclaimable
claim(s) to idle`, which never resolves because nothing reclaims. Residency is
first-come, first-served: kokoro warming from 1.58 to 2.47 GiB, and ollama taking
3.7 GiB mid-run, each broke a generation.

**Workaround:** `offload_dit_to_cpu=True` fits under ~6 GiB, at a measured 3.1x
cost (17.9 s -> 55.2 s for 45 s of audio). Poll for free VRAM before starting.

**Real fix:** `OT-P0-003` as written assumes the broker actuates. It does not today.
Either composition claims with enforcement enabled, or the operation declares an
exclusive lease and the degradation rungs in the PRD are real rungs rather than
advisory labels.

**Owner:** unassigned.

**Refs:** `internal/capacity/decide.go` (idle-yield rule), `vrooli capacity policy get`.

### 2026-09-18 — Nothing in the platform can schedule idle-time work

> **Scope corrected 2026-09-18.** This entry previously read as though the missing
> scheduler blocked the inventory feature. It does not. A pool with
> replenish-on-consume needs no scheduler: when a reservation drops depth below
> the threshold, enqueue a batch. What the missing scheduler blocks is
> *idle-time* replenishment specifically — an optimisation over when the work
> runs. The distinction matters because the previous framing made a shippable
> `OT-P1-005` look platform-blocked.

**Symptom:** The scenario is expected to top its take inventory up when the GPU is
otherwise unused. There is no mechanism to trigger that, and no definition of
"otherwise unused" to trigger on. A batch can only run because something asked for
it — either a caller directly, or a consumption event that dropped the pool below
its threshold.

**Root cause:** Two gaps, one in the platform and one in the definition.

1. **No scheduler exists.** `internal/` has no cron, timer, or recurring-work
   package. `compute-manager` acquires *remote* capacity and tracks its cost; it
   does not schedule local work. The nearest existing trigger is prompt-manager's
   `heartbeat`, which schedules agent teams rather than jobs — usable as a driver,
   but it means an agent must be running for inventory to be maintained.
2. **"Idle" has no definition to act on.** The capacity broker's `yield_when_idle`
   and `idle_unload_ttl_seconds` are contention-driven: they describe when a claim
   should give way, not when the host is free enough to start optional work. On
   this host they are advisory and never actuate (see the entry above), so even
   that signal is inert.

**Why it matters more than it looks:** surplus is not an optimisation here. The
2026-09-18 batch decision records that usable output comes from selecting among
takes, so a caller who wants one good track needs several generated. Generating
them on demand is minutes of GPU time in the caller's path, and — as the spike
found repeatedly — competing for VRAM with resident tenants is exactly when
generation fails. Idle-time topping-up dodges contention, which is the real
constraint, rather than latency.

**Workaround — and the v1, not a stopgap:** replenish on consume. The scenario
owns depth policy (target depth per style, replenish threshold) and enqueues a
batch when a reservation drops the pool below the threshold. This delivers the
whole user-visible promise — takes are ready when wanted, and regenerate as they
are used — with no scheduler present. The only thing it does not do is *choose a
good moment*, so a replenish can land on a contended card and degrade or wait.

**Real fix, for the idle half only:** not this scenario's to make alone. It needs
a platform answer to "the host is idle enough to start optional work", plus a
durable trigger that does not require an agent session to be alive. The nearest
existing driver is prompt-manager's `heartbeat`, which schedules agent teams
rather than jobs — usable, but it means inventory is only maintained while an
agent runs, which is the wrong dependency for a background capability.

Until then the trigger stays **pluggable**, and the depth policy must be testable
with no trigger present at all.

**Owner:** unassigned. The platform half belongs to the control plane, not here.

**Refs:** `internal/capacity/admission.go` (`YieldWhenIdle`, `IdleUnloadTTLSeconds`),
`docs/internal/DECISIONS.md` (2026-09-18 batch and inventory-ownership decisions), `PRD.md` `OT-P1-005` (the target this blocks).

### 2026-09-19 — Long composition waits and exact-take reservation (resolved)

**Symptoms:** A ten-take composition job outlived the CLI's default HTTP timeout,
and `takes reserve <id>` could return `409 take is not available` even when the
requested take was reported as available.

**Root causes:** `jobs wait` inherited cli-core's ordinary 120-second request
deadline even though the job continues server-side. The reservation handler first
looked up the requested take, then called the pool's style-level `Draw`; when
multiple takes shared a style, `Draw` could atomically reserve a different take
and make the requested-id comparison fail.

**Fix:** The CLI wait command now uses cli-core's server-owned no-timeout client,
preserving authentication and invocation provenance. The pool now exposes an
atomic exact-id `Reserve` transition, and the HTTP handler uses it instead of
style-level selection. Regression tests cover both behaviors; the live CLI path
was verified against a persisted launch-trap take.

**Remaining limitation:** A no-timeout wait still depends on caller cancellation
or the server reaching a terminal state; the CLI does not yet expose a progress
stream or reconnecting watch command.

**Refs:** `cli/domains/domains.go`, `api/internal/composition/pool.go`,
`api/handlers/composition/module.go`, and their regression tests.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Template contract | The generated `react-vite` manifest requires README headings (`What You Get`, `Customize Safely`) and reference-doc headings (`Notes (CRUD reference)`) that the same template version does not emit. `audio-tools` shows the identical mismatch, so this is template-wide, not local. | Fails doc validation on every newly generated scenario. | Upstream: reconcile `templates/scenarios/react-vite` manifest `requiredHeadings` with what the template emits. Locally the README headings were aligned; the `notes` example headings were left alone because `detemplate` removes them. |
| Resource wiring | ACE-Step exists and is enabled/required. music-mir remains planned. | Full analysis scope remains unimplemented; composition is available independently. | Build music-mir only under its authorized analysis scope; do not recreate ACE-Step. |
| Domain inventory | Composition/styles/jobs/pool/storage/model/cleanup exist; the full declared analysis/transform inventory does not. | Partial implementation, not whole-scenario readiness. | Reconcile each unbuilt domain with its PRD obligation before implementation; no blanket scaffold rebuild. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
- [`../reference/model-registry.md`](../reference/model-registry.md) — model evidence and confidence labels


## Work ladder

### 2026-10-15 — Reduced capacity admission rejected (W3, repaired)

A localized implementation defect: `internal/capacity/types.go` admits both `grant` and `degrade`, but the Music Tools CLI adapter accepted only `grant`. Live composition jobs failed with “granted step offload-dit” presented as a denial. The focused regression reproduced this before the fix and now covers full/reduced grants plus queue/deny rejection. Profile metadata and the release callback remain intact.

`go test -v ./internal/capacity ./handlers/composition` passed. Governed scenario restart healthy. Two live two-take batches succeeded under the reduced rung without stopping shared tenants. Scoped Test Genie unit run `20261015-185612-97316999` remains FAIL in the unrelated UI `test:coverage` command; no W0–W2 or full-suite certification claimed. Evidence and live job IDs: `/home/matthalloran8/.vrooli/plan-artifacts/efforts/product-video-research-2026-10-15/revision-03/music/capacity-repair.md`.

## 2026-10-16 launch-video delivery repair

Scoped W3 behavior: local PCM arrangement replaces the repeatedly shortened
FFmpeg filter graph, and the Audition page loads immutable audio blobs before
playback, records cut-specific choices without discarding takes, and exports an
operator attestation bound to those same bytes. Focused CLI/UI/type evidence and
the 34.119-second retained-source rehearsal live in
`~/.vrooli/plan-artifacts/efforts/product-video-research-2026-10-15/process-updates/`.
No new model inference or shared resource lifecycle action was needed.

Test Genie unit run `20261016-134901-7ccbd488` remained non-green: the UI coverage
command reported functions 74.19% and branches 79.78%, below the retained 85% floor.
Focused assertions pass; this is not whole-scenario certification. Earlier QA
`knw-1792091248952297950` tracks broader validation failure. Subsequent source/UI
changes need their own focused receipts; this run is not exact-current evidence.

## 2026-10-16 video-toolchain reliability repairs

Owner changes replace whole-timeline arrangement allocation with snapshotted,
bounded disk spools and chunked exact PCM mixing. Complete output bundles stage
before publication under a cooperating-writer lock. Cancellation covers copy,
decode and mix. The lock is not a portable fence against arbitrary external
writers; hard-killed processes may require explicit owned-scratch cleanup.

Composition now uses one durable engine. Pool load/mutations surface persistence
errors before exposing changes. Terminal jobs publish only after save; failed
finalization retains the result and Get/Wait retry that save without rerunning
model work. Process restart still marks interrupted jobs failed and retains
already completed pool takes; no automatic inference resumption is claimed.
Constructor DDL/ignored ALTER was retired. Older incompatible dev schema needs
an explicit owner maintenance step; it is not silently mutated at runtime.

Generated WAV ingestion is bounded/streamed, verifies PCM framing/declared bytes,
and records actual resource model/rung identity. Ownership receipts restore byte
accounting with protection until the pool reapplies state. Historical outputs
without receipts are counted and protected conservatively, not silently evicted.
Legacy adoption/cleanup remains explicit. Capacity operation claims heartbeat;
loss cancels the owned job, and release has a bounded diagnostic path. A resource
requiring restart to apply degradation is refused; Music performs no host repair.

Audition is a bounded candidate session with job filtering, unload, hash-bound
findings, explicit draft save/restore and generated-schema provenance parsing.
Encoded-byte bounds do not assert a browser heap hard cap. Drafts and exported
reviews remain operator attestations, not proof an agent heard audio.

Fresh source validation and remaining production qualification are recorded in
`~/.vrooli/plan-artifacts/efforts/video-toolchain-reliability-2026-10-16/implementation/music.md`.
Live services/resources were not restarted by this implementation. Older running
ACE-Step builds lack required output-identity headers and will be refused until
an authorized managed activation; do not bypass identity checks to get footage.

### Independent review corrections (2026-10-16)

Cleanup now conditionally deletes an exact eligible ownership/content/access
snapshot under the same storage lock as reservation persistence. A deletion
that wins first makes reservation refuse missing audio; the pool persists that
take as unavailable, excludes it from ready depth and requests refill under its
existing threshold policy. Provenance is retained. Pool targets are 1–100;
unsupported persisted inventory policy fails loudly instead of admitting an
impossible batch. Terminal job cache is bounded (default 256 persisted terminal
entries) without deleting durable rows, pending failed finalizations or active
subscribers. Historical Get/Wait/Subscribe retain durable recovery.

ACE-Step refuses capacity acknowledgement while loading and validates the
admitted expected rung before inference. Historical DB failures no longer become
404. Saturated subscribers retain a terminal event. Draft restore reconciles
playback/attestation hashes and cancels in-flight loads before replacing state.
Evidence and remaining qualification limits: effort implementation/music-review-fixes.md.

### Lyrics, effective seeds and partial-profile honesty (2026-10-16)

Style-owned lyrics now round-trip through the portable decoder and durable
composition snapshots; absent legacy lyrics remain instrumental. Launch consumers
must check original source-job intent and listen for actual lyrics/vocals. A style
ID alone cannot establish instrumentality, especially after delete/recreate.
English remains the configured resource language; no vocal-quality or multilingual
qualification is claimed.

Fresh compose admissions choose a base seed once unless an explicit signed-int64
base is provided (zero is valid). The queued snapshot retains it. Per-take seeds
are uint32 and ingestion requires an actual matching upstream receipt plus bounded
profile headers. Historical planned seeds are not actual-seed proof; old output
rows are not rewritten. Optional shift is sampling control, not vocal-mode control.

A newly discovered durable bug allowed a late worker to overwrite a recovered
terminal job. Active-state conditional writes now adopt the first authoritative
terminal result, including pending-save retries. Completed partial profile facts
survive failed/cancelled batches only if their worker wins that write; late facts
can remain unknown. This is not a distributed running-worker lease, automatic
inference recovery or style-versioning repair. The source changes require normal
paired Music/ACE activation; source tests do not establish current live adoption.
Evidence: effort `implementation/music-lyrics-seed-profile.md` and its retained
red/green receipts. Broad UI coverage debt and whole-scenario qualification remain
as documented above.
