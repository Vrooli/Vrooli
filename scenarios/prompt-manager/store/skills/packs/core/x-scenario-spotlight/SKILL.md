## Tools focus: Scenario Spotlight Generator

Generate a scenario-spotlight draft pitching **one Vrooli scenario** as a useful tool / app / product to its target user. Asset-led. Conversion-rung-aware. Contrarian-checked against type-level failure modes before handoff.

> **Status:** v1 (thin). The strategic canon — purpose, audience model, conversion goal, asset requirements, failure modes — lives in [`docs/marketing/post-types/text/scenario-spotlight.md`](../../../../../../docs/marketing/post-types/text/scenario-spotlight.md). This skill is the executable spec; that file is the reasoning.

---

### **1. Focus Statement**

Produce a draft scenario-spotlight (X thread / LinkedIn post / blog body) for a specified scenario, targeted at a specified audience persona at a specified conversion rung. Output is a structured draft plus an asset specification plus honesty flags — never a published artifact. Operator decides; publisher posts.

---

### **2. When to Use / When Not to Use**

| Use when | Don't use when |
|----------|----------------|
| Pitching one scenario as an end-user tool | Building project-wide progress narrative (use `x-dev-log`) |
| Driving a click-through / trial / sign-up CTA | Pitching Vrooli as a developer framework (use `x-oss-framework` once authored) |
| Coordinating a multi-asset campaign launch (use as one component) | Posting without contrarian review |
| Producing a recommendation-style spotlight when a genuine third-party wrote about the scenario | Producing a recommendation-style spotlight without genuine third-party basis (failure mode) |

---

### **3. Required Reading**

Before drafting, read in order:

```bash
prompt-manager skill read swarm-manager-initiative-context  # how to pull scenario state
```

Required file reads (every run):

- `docs/marketing/post-types/text/scenario-spotlight.md` — strategic canon (purpose, audience, conversion goal, asset requirements, contrarian failure modes). **Load-bearing.**
- `docs/marketing/STRATEGY.md` — voice canon (Voice section, Voice samples, Anti-patterns).
- `docs/marketing/post-techniques/essay-shape.md`
- `docs/marketing/post-techniques/hook-vs-body-asymmetry.md`
- `docs/marketing/post-techniques/intro-on-first-mention.md`
- `docs/marketing/post-techniques/inter-post-linkage.md`
- `docs/marketing/post-techniques/no-internal-numbering-externally.md`
- `docs/marketing/AUDIENCES.md` — to resolve the audience-persona input.
- `docs/marketing/CHANNELS.md` — sanitization rules and per-platform formatting.
- `docs/marketing/ASSETS.md` and `docs/marketing/IMAGE_STYLE.md` — brand consistency rules for the asset.
- The target scenario's `PRD.md` and `README.md` — verifiable claims about what the scenario actually does.
- `docs/monetization/TIERS.md` and `docs/monetization/scenario-sku-map.json` — for tier-alignment of demo'd features against the CTA's tier.
- `scenarios/prompt-manager/store/teams/marketing-crew/shared/published-scenario-mentions.jsonl` (filtered by audience) — to decide whether to apply intro-on-first-mention or use a one-line refresher.

Optional reads:
- `docs/marketing/notebook/SCENARIO_SPOTLIGHT_CRAFT.md` — emerging craft patterns (file may not yet exist on first run).
- `docs/marketing/notebook/AUDIENCE_OBSERVATIONS.md` — observations about how audiences respond.

---

### **4. Inputs**

The caller (typically `subscription-advertiser` member, occasionally direct operator) provides:

| Input | Required | Example |
|-------|----------|---------|
| `scenario` | Yes | `"social-media-scheduler"` |
| `audience_persona` | Yes | `"subscription-buyer"` (matches a persona in `AUDIENCES.md`) |
| `conversion_rung` | Yes | `"click-through"` \| `"trial"` \| `"sign-up"` |
| `format` | Yes | `"x-thread"` \| `"linkedin-post"` \| `"blog"` |
| `variant` | No (default `"direct"`) | `"direct"` \| `"recommendation"` \| `"problem-led"` \| `"comparison"` |
| `comparison_target` | If variant=comparison | The named alternative tool |
| `third_party_basis` | If variant=recommendation | Concrete reference: external user write-up, contributor name, or other genuine third party |
| `series_id` | No | If this is a follow-up spotlight in a series |

---

### **5. Process**

Run the steps in order. Do not skip the verifiability passes — they are what distinguishes a real spotlight from theater.

1. **Pull verifiable scenario state.** Read the scenario's `PRD.md`, `README.md`, and any `docs/concepts/` or `docs/guides/` material. Build a list of every claim that could appear in the post and tag each: `verified` (cross-checked against PRD/README) or `uncertain`. Discard `uncertain` claims before drafting.

2. **Tier-align the demo'd features.** For every feature you might demo, look up its tier in `scenarios/<scenario>/.vrooli/service.json` and `docs/monetization/scenario-sku-map.json`. Mark any feature not reachable from the `conversion_rung`'s tier as `tier-mismatch` and exclude it.

3. **Resolve audience familiarity.** Filter `published-scenario-mentions.jsonl` by `audience_persona`. If the scenario has not been mentioned to this audience before, plan a full intro (one sentence: what it is, why it exists, what it does at a high level). If mentioned ≥1 time, use a one-line refresher.

4. **Draft the hook.** Concrete friction or moment of recognition. No abstract value-prop language. Length asymmetric to the body per `STRATEGY.md`.

5. **Draft the body around the asset.** Specify the asset (screen recording with timestamps, screenshot sequence with captions). Body text wraps the asset with what the reader is seeing and why it matters for *their* workflow.

6. **Draft the CTA at the specified conversion rung.** One link. One verb. If the spotlight could plausibly target multiple rungs, pick one — do not blur.

7. **Apply variant-specific shape.** For recommendation variant, frame in third-party voice and cite the `third_party_basis` (in production, not in the post copy itself unless natural). For problem-led, lead with the friction in the audience's voice. For comparison, name the comparison target and the concrete differentiator.

8. **Run the contrarian self-check.** For each failure mode in `post-types/text/scenario-spotlight.md`'s contrarian table, produce a one-line answer:
   - Capability inflation? (cite evidence: every claim verified in step 1)
   - Demo theater? (would a fresh audience-persona user replicate the demo without operator help?)
   - Pricing/tier confusion? (every demo'd feature reachable from CTA tier — verified in step 2)
   - Brand-asset drift? (asset spec aligned with `ASSETS.md` + `IMAGE_STYLE.md`)
   - Internal-vocabulary leakage? (scan copy and asset spec for `p\d+`, `round-`, batch ids, paths, agent-internal names)
   - Recommendation-framing without basis? (only if variant=recommendation: cite the third-party basis)
   - Underclaim? (does the post communicate why the scenario is interesting?)
   - Conversion-rung blur? (one verb in the CTA)

9. **Emit the output contract** (see §6). Do not publish; the publisher member submits a `content-publish-proposal` decision and the operator decides at the vision walk.

---

### **6. Output Contract**

End your response with a fenced ` ```json ` block with this shape:

```json
{
  "scenario": "social-media-scheduler",
  "audience_persona": "subscription-buyer",
  "conversion_rung": "click-through",
  "format": "x-thread",
  "variant": "direct",
  "draft": {
    "hook": "...",
    "intro": "...",
    "body": [
      { "kind": "text", "content": "..." },
      { "kind": "asset_reference", "asset_id": "asset-1", "caption": "..." },
      { "kind": "text", "content": "..." }
    ],
    "conclusion": "...",
    "cta": {
      "verb": "Try it",
      "url_placeholder": "<insert-scenario-landing-page-url>"
    },
    "tweets_or_paragraphs": [
      "Tweet 1 / paragraph 1 text...",
      "Tweet 2 / paragraph 2 text..."
    ]
  },
  "asset_spec": {
    "asset_id": "asset-1",
    "kind": "screen-recording",
    "substrate": "bas",
    "duration_seconds_target": 30,
    "scenes": [
      { "at_seconds": 0, "description": "..." },
      { "at_seconds": 8, "description": "..." }
    ],
    "brand_overlay_required": true,
    "sanitization_pass": "required"
  },
  "honesty_flags": {
    "feature_claims": "measured",
    "demo_authenticity": "replicable",
    "tier_alignment": "verified",
    "third_party_basis": "self-authored"
  },
  "contrarian_self_check": {
    "capability_inflation": "no — every claim cross-checked against PRD",
    "demo_theater": "no — fresh user can replicate per scenes 1-3",
    "pricing_tier_confusion": "no — features map to free-tier per scenario-sku-map",
    "brand_asset_drift": "no — overlay matches IMAGE_STYLE palette",
    "internal_vocabulary_leakage": "no — scanned for p\\d+, round-, batch ids",
    "recommendation_without_basis": "n/a — variant=direct",
    "underclaim": "no — body shows the differentiated capability",
    "conversion_rung_blur": "no — single CTA: click through"
  },
  "verified_claims": [
    "Claim X — cited in PRD section Y",
    "Claim Z — cited in README"
  ],
  "discarded_claims": [
    "Claim A — uncertain, not in PRD"
  ]
}
```

---

### **7. Anti-Patterns**

Defer to `post-types/text/scenario-spotlight.md`'s contrarian table for the canonical list. Common slips:

- **Drafting the post before pulling scenario state.** You'll discover you've claimed something the scenario doesn't do; the rewrite is harder than starting from verified claims.
- **Burying the asset.** A spotlight is asset-led. If the body reads as a coherent post without the asset, the body is doing too much; the asset is doing too little.
- **Producing the asset spec but not sanity-checking it against `ASSETS.md` / `IMAGE_STYLE.md`.** Asset drift catches the contrarian; pre-empt it.
- **Writing recommendation-variant copy without `third_party_basis`.** Reject your own draft and switch variants.

---

### **8. Promotion Path for Patterns You Notice**

If you observe a recurring pattern across runs (e.g., "audience X responds better to comparison variant," "scenarios with CLI-only surfaces benefit from asciinema rather than BAS recording"), append it to `docs/marketing/notebook/SCENARIO_SPOTLIGHT_CRAFT.md` (create the file on first observation). Tag the entry with the pattern and a target promotion location:

- Skill edit (this file's process / output contract) — most observations land here.
- `docs/marketing/post-types/text/scenario-spotlight.md` — observations changing strategic canon (audience model, new variant, new failure mode).
- `docs/marketing/post-techniques/<name>.md` — observations that turn out cross-cutting and apply to other post types.

`brand-manager` curates the promotion path via `notebook-promotion` decisions.

---

### **9. Cross-References**

- Strategic canon: [`docs/marketing/post-types/text/scenario-spotlight.md`](../../../../../../docs/marketing/post-types/text/scenario-spotlight.md).
- Voice canon: [`docs/marketing/STRATEGY.md`](../../../../../../docs/marketing/STRATEGY.md).
- Sibling post-type skill: `x-dev-log` (project-wide narrative — for contrast).
- Asset substrate: [Browser Automation Studio scenario](../../../../../../scenarios/bas/) — known constraint: BAS `recordVideo` gray-bar requires CDP workaround (see BAS scenario docs).
- Publishing plumbing (planned): `social-media-scheduler` scenario, initiative-proposal `dec-1777312920606447957`.
