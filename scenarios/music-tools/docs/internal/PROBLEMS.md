# Problems — Music Tools

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

### 2026-08-19 — Declared resources do not exist yet

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

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Template contract | The generated `react-vite` manifest requires README headings (`What You Get`, `Customize Safely`) and reference-doc headings (`Notes (CRUD reference)`) that the same template version does not emit. `audio-tools` shows the identical mismatch, so this is template-wide, not local. | Fails doc validation on every newly generated scenario. | Upstream: reconcile `templates/scenarios/react-vite` manifest `requiredHeadings` with what the template emits. Locally the README headings were aligned; the `notes` example headings were left alone because `detemplate` removes them. |
| Resource wiring | Docs declare two required resources; `.vrooli/service.json` declares none. | Blocks `implemented`. | Create the resources and declare them. |
| Domain scaffolding | `DOMAINS.md` declares twelve domains; the generated `notes` example domain is still present and none of the twelve exist in code. | Expected at `generated`; blocks `implemented`. | Detemplate, then build domains in the launch order in `PRD.md`. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
- [`../reference/model-registry.md`](../reference/model-registry.md) — model evidence and confidence labels
