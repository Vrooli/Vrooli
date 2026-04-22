# Initiative Context: AI Image Generation Foundation

## Strategic Rationale

User spends 80-90% of project time answering Swarm Manager decision questions. Reading a dozen sentences per option for hundreds of decisions per day is the biggest remaining UX friction. UI decisions in particular have high text-to-visual information asymmetry — "clean sidebar with muted palette" maps to dozens of real UIs. Adding AI-generated mockups to decision options cuts per-decision time and, more importantly, propagates an approved visual target forward to the executor and ecosystem-manager (compound value, not just reading-speed).

## Cross-Item Decisions

### Vendor: OpenAI direct, not OpenRouter
- OpenRouter only exposes gpt-image-2 via a GPT-5.4 composite (`openai/gpt-5.4-image-2`) which bundles reasoning tokens; not cost-efficient for pure image gen.
- Direct OpenAI API pricing: $8 input / $30 output per 1M tokens, ~$0.03-0.05 per medium-quality image, ~$0.13-0.20 high.
- Research item must still compare the composite on real workloads before finalizing — this is the current working assumption, not a locked decision.

### Toggle is load-bearing; enforced in one place
- Image gen is expensive and must be user-toggleable from Swarm Manager settings.
- The toggle is checked inside the `swarm-manager image generate` CLI helper — NOT in each skill.
- Skills invoke the helper unconditionally; helper short-circuits when setting is off.
- This is why the foundation initiative has a dedicated `swarm-manager-image-cli-helper` item separate from the skill.

### Skill scope: UI mockups only, initially
- Diagrams, charts, and other image-gen use cases are deferred. Prove the UI loop works first.
- Future sibling skills (e.g. `diagram-generation`) can be added once the UI case is validated.

### Prompt template rules (from OpenAI cookbook UI-mockup guidance)
- Present-tense "as if shipped" — not "a concept for" or "a design sketch"
- No concept-art vocabulary ("sketch", "stylized", "concept")
- Structure: layout → hierarchy → spacing → real interface elements → constraints
- Device framing (browser/iPhone frame) when applicable
- Reference-image channel holds style consistency; text holds content

### Generation gate (the "when to use" rule)
- Agents invoke image gen only when: *a reader could pick the wrong option from text alone, but the right one from an image*.
- This framing blocks the decoration failure mode (generating images that look nice but don't change the answer).

## Sequencing Notes

1. Research item runs first — it gates all three execute items and can invalidate the vendor/cost assumptions.
2. Key plumbing must land before the CLI helper (the helper loads the key).
3. The skill depends on both the research conclusion AND the CLI helper (it invokes the helper and uses research-derived prompt templates).
4. Everything in `decision-question-visuals` can proceed in parallel except `workshop-skill-image-guidance`, which is the integration point and depends on schema + toggle + skill all existing.
5. Grounding (initiative 3) is intentionally deferred — initiatives 1+2 ship pure text-to-mockup first, which is enough value to validate the feature without touching BAS.

## Unresolved Questions Deferred To Workshop

- Exact storage location for generated images (local filesystem vs MinIO vs embedded data URIs in round JSON). Likely resolved by the research item + handoff-propagation item together.
- How image cost appears in the existing stats dashboard, if at all.
- Whether failures in image gen should degrade gracefully (text-only option) or be surfaced as errors to the agent that can retry.
