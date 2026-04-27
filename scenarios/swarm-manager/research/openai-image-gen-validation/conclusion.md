# Research Conclusion: Validate OpenAI gpt-image-2 for UI Mockup Generation

## Research Question
Is OpenAI `gpt-image-2` (accessed directly via the OpenAI API) a suitable backend for Swarm Manager decision-question UI mockups, and does its cost, quality, and style-consistency behavior justify going OpenAI-direct over the OpenRouter `openai/gpt-5.4-image-2` composite?

Specifically:
1. Can the target environment (Swarm Manager + agent sandboxes) reach the OpenAI images endpoint with an API key?
2. What is the real per-image cost at the quality we would ship at for decision mockups?
3. Does `input_fidelity='high'` with a reference image produce style-consistent outputs suitable for the ui-mockup-generation skill?
4. Across 3-5 real-world decision scenarios (pure text-to-mockup, no screenshot grounding), does the end-to-end loop produce mockups that would change a user's decision answer?
5. How does OpenAI-direct compare to the OpenRouter composite on cost and quality for the same prompts?

## Summary
Workshop has fully locked the validation protocol; empirical execution is still pending. The protocol is: verify the gpt-image-2 API surface against current OpenAI docs, then run 3-5 historically-derived prompts at medium quality with a Vrooli screenshot as style anchor, score them against the would-change-the-answer gate (self-rated, ≥60% pass bar), spot-check 1-2 prompts on OpenRouter for cost comparison, persist outputs to local filesystem under `archive/`, and self-stop at $5 cumulative API spend. The conclusion's go/no-go and final cost/comparison numbers will be filled in by the execution run; everything needed to start that run is now specified.

## Methodology
Locked from round 1 decisions (d1=A, d2=A, d3=A, d4=A) and round 2 decisions (d1=A, d2=A, d3=A, d4=A):

**Round 1 — what to test:**
- **Test prompts (r1.d1=A):** Pull 3-5 UI-shaped decisions from recent Swarm Manager workshop rounds and convert each to an image-prompt. Selection rule: decisions whose options describe layout, palette, density, or interface-element placement — not naming or backend choices. Prompts derived from the actual decision `topic` + `options[].label` text so the test reflects the real distribution the skill will see.
- **Default quality tier (r1.d2=A):** Validate **medium** quality as the shipping default; render **high** for one prompt to characterize the cost/quality delta as an opt-in escape hatch.
- **OpenRouter comparison (r1.d3=A):** Minimal spot-check — run **1-2** of the test prompts through the OpenRouter `openai/gpt-5.4-image-2` composite at matching (medium) quality, compare $/image and a side-by-side qualitative read.
- **Reference image (r1.d4=A):** Use a single existing Vrooli UI screenshot as the style anchor across all `input_fidelity='high'` test prompts. Style consistency judged by palette/typography/density agreement across outputs sharing that anchor — not layout faithfulness (grounding-propagation's job).

**Round 2 — how to execute:**
- **API verification (r2.d1=A):** Before any live call, read current OpenAI Images API docs + the cookbook UI-mockup notebook and record exact endpoint path, model ID string, parameter name/values for the quality tier, parameter name/values for `input_fidelity`, and the request shape for attaching a reference image into the conclusion. Then issue **one** cheapest-tier call from the target environment as a connectivity smoke test before burning budget on the real prompts.
- **Scoring protocol (r2.d2=A):** For each test prompt, the executor writes down their pre-image option pick first, then renders the mockup, then records whether seeing the mockup would have changed (or meaningfully confirmed-with-new-info) that pick. The conclusion reports the count. **Pass bar = ≥60% changed-or-meaningfully-confirmed.** Below that, the recommendation is no-go (or scope-narrow) on the downstream skill execute item.
- **Image persistence (r2.d3=A):** Write one PNG per `(prompt, vendor, quality)` combination to `archive/generated/` under this backlog item's folder, plus an `index.json` mapping each PNG to its prompt, vendor, quality tier, token counts, and dollar cost. This doubles as a deliberate prototype answer for the foundation-level open question on image storage (local FS vs MinIO vs embedded data URIs).
- **Spend cap (r2.d4=A):** Hard cap of **$5 USD** cumulative API spend across OpenAI + OpenRouter combined. The harness must track running spend after each call and abort with partial-results reporting if the cap would be exceeded by the next call.

Every per-call record persisted into `archive/generated/index.json` includes: input/output token counts, derived dollar cost, vendor, quality tier, whether a reference image was attached, and the score-protocol outcome (pre-pick → post-pick → changed?).

## Findings

### Finding 1: Working vendor assumption is OpenAI-direct, unverified
Per `initiative:ai-image-generation-foundation/orchestration-summary.md`, the initiative's current working assumption is that OpenAI-direct is cheaper than OpenRouter's `openai/gpt-5.4-image-2` composite because the composite bundles GPT-5.4 reasoning tokens. Quoted prices: $8 input / $30 output per 1M tokens; ~$0.03-0.05 medium-quality image, ~$0.13-0.20 high-quality. Round 1 d3=A locks the verification protocol: a 1-2 prompt spot-check at matching quality, enough to confirm or overturn the assumption without full-matrix spend.

### Finding 2: Research scope is narrower than the surface description suggests
Screenshot grounding (using a Vrooli page capture as a reference image to pin layout) is explicitly handled by the downstream `decision-visual-grounding-propagation` initiative. This research is **text-to-mockup only**, plus a style-consistency check of `input_fidelity='high'` — not a layout-faithfulness check. A no-go on style consistency blocks only the current initiative; it does not itself block grounding.

### Finding 3: Toggle-gate and CLI-helper boundary shape the test harness
The initiative doc is emphatic that the EnableImageGeneration toggle lives inside `swarm-manager image generate`, not per-skill. Since this research predates that helper (the helper is `execute/swarm-manager-image-cli-helper`, a downstream item), the research harness must call the OpenAI API directly or via a throwaway wrapper. Findings must not assume the CLI helper exists.

### Finding 4: API surface must be schema-verified before any live call
The initiative doc and this item's spec name `gpt-image-2`, `quality` tiers, and `input_fidelity='high'` but cite no endpoint URL, no parameter casing, and no response-shape contract. Round 2 d1=A locks the mitigation: verify endpoint path, model ID string, quality-tier parameter name/values, `input_fidelity` parameter name/values, and the reference-image attachment shape against current docs *before* issuing the smoke test, then record the pinned values into this conclusion before live runs. This converts an unverified-API single-point-of-failure into a fail-fast 20-minute step.

### Finding 5: Prompt-template rules are already partially specified upstream
The initiative's orchestration summary already records the cookbook-derived rules: present-tense "as if shipped"; no concept-art vocabulary ("sketch", "stylized", "concept"); structure layout → hierarchy → spacing → real interface elements → constraints; device framing when applicable; reference-image channel holds style consistency, text holds content. The deliverable's "prompt-template sketch" should treat these as accepted inputs and add only what falls out of empirical testing (e.g. specific phrasings that demonstrably moved outputs in the test runs).

### Finding 6: The "would-change-the-answer" gate is the implicit success criterion, now scored
The orchestration summary defines the agent-side generation gate: *invoke image gen only when a reader could pick the wrong option from text alone, but the right one from an image.* Round 2 d2=A operationalises this for the research itself: each test prompt is scored by recording the executor's pre-image option pick, rendering the mockup, then recording whether the mockup would have changed (or meaningfully confirmed-with-new-info) the pick. Pass bar is ≥60%. Below that the recommendation is no-go (or scope-narrow) on `execute/ui-mockup-generation-skill`.

### Finding 7: Image storage format — local filesystem, prototyped here for foundation
The initiative's "Unresolved Questions Deferred To Workshop" section explicitly flagged image storage location as an open question to be *"resolved by the research item + handoff-propagation item together."* Round 2 d3=A makes a deliberate choice: this research persists outputs to the local filesystem under `archive/generated/` as one PNG per `(prompt, vendor, quality)` combination plus an `index.json`. Foundation can adopt this pattern, evolve it (e.g. swap in MinIO when scale demands), or reject it — but the precedent is now an explicit, defensible decision rather than an accidental side-effect of research mechanics. Rationale: cheapest, inspectable, ports cleanly into a filesystem-backed production answer, doesn't pre-commit foundation to a server-side blob store.

### Finding 8: Reference screenshot is not present in the sandbox working tree
A scan of the agent-manager sandbox working tree found no PNG/JPG assets and no `screenshots/` or `assets/` directories. The d4=A reference image must be sourced from the actual Vrooli scenario repo or captured at execution time. Surfacing this now so the executor doesn't assume the asset is checked in here and waste cycles searching for it.

### Finding 9: Spend has a hard cap so the harness self-stops
Round 2 d4=A sets a $5 cumulative API spend cap across OpenAI + OpenRouter combined. Back-of-envelope expected spend (~$0.50-$2.00) leaves 3-5x headroom for retries, OpenRouter price surprises, and the single high-quality render, while protecting against parameter mistakes that could re-render everything at the high tier without anyone noticing.

## Limitations
- This research does not prove out screenshot-grounded mockups — that is deferred to the downstream `decision-visual-grounding-propagation` initiative.
- Per-image cost measured here reflects the prompts we happen to test. Real production distribution (prompt lengths, whether reference images are attached) may shift the cost envelope.
- API key plumbing is not in scope; the research only needs ad-hoc API access, not the production path.
- The OpenRouter spot-check is a 1-2 prompt sample (per r1.d3=A). It can confirm a large gap but not reliably catch a small one — the conclusion's cost-justification will state confidence accordingly.
- Style-consistency judgement against the Vrooli reference image is a single-rater visual call, not an inter-rater study.
- The would-change-the-answer scoring is self-rated by the executor (per r2.d2=A) — single rater, possibly 5 prompts. The 60% pass bar is a defensible heuristic but not a statistically robust threshold.
- Empirical findings (live cost numbers, OpenRouter comparison, style-consistency outcome, scoring tally) do **not yet exist** in this conclusion. The Methodology and pass criteria are locked; the actual go/no-go call and updates to sibling-item priorities below are predicated on the execution run completing.
- The reference Vrooli screenshot needed for `input_fidelity='high'` testing is not present in this repo (Finding 8) and must be sourced at execution time.

## Actions

### Action 1: Execute the locked research protocol
- **Owner:** the research-execution agent for this backlog item
- **What:** Run the validation per the Methodology section above. Concretely: (1) verify the OpenAI Images API surface and pin endpoint/model-ID/parameter shapes into this conclusion before any live call; (2) issue one cheapest-tier connectivity smoke test; (3) source one Vrooli UI screenshot as the style anchor; (4) curate 3-5 historical UI-shaped decisions and convert to image-prompts; (5) render at medium quality with the reference image, plus one high-quality render of one prompt; (6) spot-check 1-2 prompts on OpenRouter `openai/gpt-5.4-image-2` at matching quality; (7) score each prompt against the would-change-the-answer gate (pre-pick → render → post-pick → changed?); (8) persist all PNGs and an `index.json` to `archive/generated/`; (9) abort and report partial results if cumulative OpenAI + OpenRouter spend would exceed $5.
- **Output:** Update this `conclusion.md` with: pinned API surface details, real per-image cost numbers, OpenRouter comparison numbers, the scoring tally, the prompt-template sketch derived from what actually moved outputs, and an explicit go / scope-narrow / no-go call.
- **Reason:** The protocol is fully specified; the empirical run is the deliverable.

### Action 2: Update backlog item — execute/openai-api-key-plumbing on go-decision
- **Kind:** execute
- **Name:** openai-api-key-plumbing
- **Trigger:** Action 1 produces a "go" or "go-with-caveats" call.
- **Changes (conditional on outcome):** No metadata change required on a clean go — the existing `depends_on: research/openai-image-gen-validation` will be satisfied. On a no-go, change `status` to `archived` and document the no-go reason in the description.
- **Reason:** This item is currently blocked on this research; its disposition flips entirely on Action 1's outcome. Calling it out explicitly so the orchestrator does not need to re-derive the implication.

### Action 3: Update backlog item — execute/ui-mockup-generation-skill with prompt-template sketch and scope
- **Kind:** execute
- **Name:** ui-mockup-generation-skill
- **Trigger:** Action 1 produces a go or scope-narrow call.
- **Changes:** After Action 1 completes, update this item's `description` to attach (or link) the prompt-template sketch produced by the research and to record the validated quality tier (medium) plus the `input_fidelity='high'` finding. On a scope-narrow outcome (e.g. style consistency fails at medium), also narrow the description to the supported subset.
- **Reason:** This item depends on the research and on the CLI helper; once the protocol's empirical outputs exist, the skill's description should carry the validated prompting rules forward instead of re-deriving them from the cookbook.

### Action 4: Update initiative — record image-storage decision on ai-image-generation-foundation
- **Name:** ai-image-generation-foundation
- **Changes:** After Action 1 completes, update the initiative's description (or its `orchestration-summary.md`) to mark the previously-open image-storage question as resolved with the local-filesystem prototype from Finding 7, citing this research as the precedent. If foundation later chooses MinIO or another store, that becomes a separate, conscious decision to override — not a quiet drift.
- **Reason:** Finding 7 closes a foundation-level open question. Recording the resolution at the initiative level prevents the same question from re-surfacing during downstream planning.

### Action 5: No new sibling backlog items
- **Reason:** The initiative has 4 members in the right shape (this research, key plumbing, CLI helper, mockup-generation skill) with the right `depends_on` topology. The empirical execution is the deliverable of this research itself, not a separate item; the API-surface verification, smoke test, scoring, and storage-format prototype all fit inside Action 1's scope. Spinning up a new item would fragment a single executable protocol across two backlog entries with no benefit.
