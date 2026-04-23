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
<!-- TBD — will be filled once findings converge. Working assumption (from initiative context): OpenAI-direct wins on cost because OpenRouter only offers gpt-image-2 via a GPT-5.4 composite that bundles reasoning tokens. Research must verify this on real workloads before it becomes a locked decision. -->

## Methodology
Planned (subject to workshop decisions):
- Read the OpenAI Images API docs and the cookbook UI-mockup guidance referenced by the initiative to lock down endpoint, parameter names, and supported quality tiers for `gpt-image-2`.
- From Swarm Manager's target environment, issue one no-op / cheapest-tier call against the OpenAI Images API to confirm connectivity and key plumbing shape (the actual key plumbing is a downstream execute item; this research only needs the ability to call the API).
- Pick 3-5 real decision-question prompts from historical Swarm Manager rounds (see Decision d1) and render each at the candidate quality tiers.
- For the reference-image (`input_fidelity='high'`) channel, render the same prompts with a Vrooli UI reference image attached and compare style consistency.
- Re-issue the same prompts through the OpenRouter `openai/gpt-5.4-image-2` composite and compare cost-per-image and perceived mockup quality.
- Record per-call input/output token counts and derived dollar cost.

## Findings

### Finding 1: Working vendor assumption is OpenAI-direct, unverified
Per `initiative:ai-image-generation-foundation/orchestration-summary.md`, the initiative's current working assumption is that OpenAI-direct is cheaper than OpenRouter's `openai/gpt-5.4-image-2` composite because the composite bundles GPT-5.4 reasoning tokens. Quoted prices: $8 input / $30 output per 1M tokens; ~$0.03-0.05 medium-quality image, ~$0.13-0.20 high-quality. This research item owns the job of either confirming that assumption against real workloads or overturning it.

### Finding 2: Research scope is narrower than the surface description suggests
Screenshot grounding (using a Vrooli page capture as a reference image to pin layout) is explicitly handled by the downstream `decision-visual-grounding-propagation` initiative. This research is **text-to-mockup only**, plus a style-consistency check of `input_fidelity='high'` — not a layout-faithfulness check. A no-go on style consistency blocks only the current initiative; it does not itself block grounding.

### Finding 3: Toggle-gate and CLI-helper boundary shape the test harness
The initiative doc is emphatic that the EnableImageGeneration toggle lives inside `swarm-manager image generate`, not per-skill. Since this research predates that helper (the helper is `execute/swarm-manager-image-cli-helper`, a downstream item), the research harness must call the OpenAI API directly or via a throwaway wrapper. Findings must not assume the CLI helper exists.

<!-- Further findings will accrue as rounds progress. -->

## Limitations
- This research does not prove out screenshot-grounded mockups — that is deferred to `research/bas-grounded-mockup-flow`.
- Per-image cost measured here reflects the prompts we happen to test. Real production distribution (prompt lengths, whether reference images are attached) may shift the cost envelope.
- API key plumbing is not in scope; the research only needs ad-hoc API access, not the production path.

## Actions
<!-- TBD — will be populated when findings converge. Expected shape:
- Update or confirm the initiative's working vendor assumption.
- Go/no-go call on the downstream execute items (openai-api-key-plumbing, ui-mockup-generation-skill).
- Handoff of prompt-template sketch to the ui-mockup-generation-skill execute item.
- Possibly update cost-related fields or descriptions on sibling items if the cost envelope differs materially from the initiative's working numbers.
-->
