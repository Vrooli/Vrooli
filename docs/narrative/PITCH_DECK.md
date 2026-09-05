# Pitch Deck — Outline

Slide-by-slide outline for Vrooli pitch decks. The structure is **family-explainer-first** (the operator's stated near-term need), built so the same outline also serves partner conversations and early-customer onboarding. A fundraising-shaped variant is flagged but not encoded — author it when fundraising is the actual context.

**Write rule:** operator-curated via accepted `brand-guideline-update` decisions for the *outline*. **Slide content is operator-authored** — voice, judgment, and stakes (fundraising, family-explainer) are too personal to delegate.

**Status:** Outline only. Slide content is TBD until the operator authors the actual deck.

---

## Variants

| Variant | Use when | Diff from base |
|---|---|---|
| Base | Family, friends, partners, early customers, general explanation | Default outline below |
| Fundraising | Talking to investors / funds | Heavier emphasis on traction (slide 7), market size (slide 6), financial model (slide 8); explicit Ask slide (slide 11). Author this variant only when actually raising. |
| Customer | Specific customer / SKU buyer conversations | Lead with the SKU's value prop; slides 1-4 reordered to put their problem first. Skip slides 9-11. |

Default to the **base** variant unless context demands otherwise.

---

## Base outline (12 slides)

### Slide 1 — Title / Hook

**Goal:** establish what this is in one beat.

**Content slots:**
- Logo (use `assets/public/Vrooli-motto-shadow.png` or higher-res variant)
- Motto: *Software that builds itself.*
- One-line description: *A self-improving software foundry that runs on your own hardware.*
- Operator name + role
- Date / context (which audience this deck was prepared for)

**Voice notes:** Short. Don't explain yet. Let the title carry the weight.

---

### Slide 2 — The Problem

**Goal:** the problem Vrooli is solving, in audience-relevant terms.

**Content slots (operator picks the framing that fits the audience):**
- *Family-explainer framing:* I have more ideas than time. Most software gets built by hand, slowly. AI can write code now, but the work disappears after each task. Nothing accumulates.
- *Builder framing:* AI agent frameworks today are ephemeral — agents call tools, do work, and the work doesn't crystallize into anything reusable.
- *Sovereignty framing:* The most powerful AI runs in someone else's cloud, on someone else's terms, with someone else's view of your data.

**Voice notes:** Pick one framing per audience. Don't list all three.

---

### Slide 3 — The Vision

**Goal:** what would the world look like if this problem were solved?

**Content slots:**
- *What if software built itself?*
- *What if every problem solved became a permanent tool the system could use forever?*
- *What if your AI ran on your hardware, with your data, under your control — and got compoundingly better the more you used it?*

Pull from [`PITCH.md` § Universal one-line pitch](PITCH.md#universal-one-line-pitch) and the relevant audience-tailored lead.

**Voice notes:** Short, evocative. Stay grounded; don't promise the moon yet — that's slide 9-10.

---

### Slide 4 — Vrooli (the answer)

**Goal:** introduce the product as the answer to slides 2-3.

**Content slots:**
- *Vrooli: a self-improving software foundry on your own hardware.*
- Three-bullet summary:
  1. AI agents build *scenarios* — full apps with UI, API, CLI, tests
  2. Each scenario becomes a permanent capability future scenarios can compose
  3. The system gets compoundingly better the more it builds
- License: open source, AGPL.
- Status: pre-1.0, active development.

**Voice notes:** Plain. This is the moment the audience says "ohhh, that."

---

### Slide 5 — How It Works

**Goal:** demystify the architecture in one slide.

**Content slots:**
- Three layers (visual diagram preferred):
  1. Resources (local services: Postgres, Redis, Ollama, Qdrant, etc.)
  2. Scenarios (full apps composed from resources + other scenarios)
  3. Agents and teams (build, review, coordinate via vision walks)
- Optional: a recursive-arrow diagram showing scenarios being composed by future scenarios.

**Voice notes:** Visual-heavy. Don't overload text.

---

### Slide 6 — What Makes Vrooli Different

**Goal:** differentiation against the competitive landscape.

**Content slots:**
- vs other agent frameworks (OpenClaw, Hermes, OpenHands, Cline, AutoGen): *output is shippable apps, not just executed tasks.*
- vs no-code (n8n, Zapier, Make): *agents build the apps, not just wire existing services.*
- vs cloud LLM products (ChatGPT, Claude.ai): *runs on your hardware; uses LLMs as a substrate, isn't one.*
- vs research prototypes: *active development, real roadmap, real revenue lines.*

Pull from [`FAQ.md` § How is Vrooli different](FAQ.md#how-is-vrooli-different-from-agent-frameworks-like-openclaw-hermes-openhands-or-cline) and [`PITCH.md` § What Vrooli is NOT](PITCH.md#what-vrooli-is-not).

**Voice notes:** Acknowledge competitors are real and capable. Don't punch down. The differentiator is structural, not "we're better." Operators reading this should think: "ah, different category."

---

### Slide 7 — Where It Is Today

**Goal:** show concrete current reality, not promises.

**Content slots:**
- Working: Tier 1 local stack on Linux / macOS / WSL2; library of scenarios; resource layer integrated; multi-team agent coordination
- Specific scenarios that work end-to-end (operator picks 3-5 to highlight, e.g. LPBS, Web Console, Swarm Manager, Browser Automation Studio)
- Repository link, recent commit cadence (or stat: e.g. "scenarios shipped in last N months")
- Honest gaps: what's not yet ready (mobile-first, managed-cloud SKU, etc.)

**Voice notes:** Honest. The point of this slide is to ground everything that follows. If you over-claim here, the audience disengages.

---

### Slide 8 — Business Model

**Goal:** how Vrooli makes money. Different audiences want different depth here.

**Content slots:**
- Themed app bundles (developer/solopreneur first; lifestyle/household next; etc.)
- Managed deployment subscriptions (convenience for non-self-host users)
- Direct scenario deployments (custom builds)
- Hardware (long-term)
- Subscription is convenience, not a paywall — every scenario is self-hostable for free.
- Pre-revenue today; first bundle in flight.

For family-explainer: keep it simple — "we sell themed sets of apps as products, plus subscriptions for people who don't want to run their own infrastructure."
For partner / customer: emphasize the SKU model and the bundle they're closest to.
For fundraising: emphasize the compounding-cost-per-bundle thesis.

**Voice notes:** Honesty flags everywhere. No claimed revenue numbers without `aspirational` or `pending-telemetry` framing.

See [`path:docs/monetization/`](../monetization/) for the full plan.

---

### Slide 9 — The 1-Year Arc

**Goal:** what's next, concretely.

**Content slots:**
- Ship the developer/solopreneur bundle
- Validate the bundle-monetization model with first paying revenue
- Grow OSS contributor base
- Expand deployment tier maturity (mobile, managed-cloud)

**Voice notes:** Concrete and falsifiable. If nothing here ships in 12 months, the story breaks.

---

### Slide 10 — The 3-10 Year Arc

**Goal:** the bigger picture, audience-calibrated.

**Content slots:**
- Bundle saturation (developer + lifestyle + household + specialized)
- Domain-specialized servers (engineering, science, finance) — see VISION.md Phase 3
- Hardware appliance line for households / businesses
- OSS ecosystem grows: contributors share scenarios across instances; global tech tree compounds
- Distributed automation simulation tooling matures

**Audience calibration:**
- For mainstream / family / customer audiences: stop here. The arc is exciting; the deep-vision (post-labor, peaceful-revolution) is **not** for these audiences.
- For futurist / AI-aligned / post-labor-curious audiences: add the deep-vision summary from [`NARRATIVE.md` § Deep vision](NARRATIVE.md#deep-vision-bracketed--vision-aligned-audiences-only). For everyone else: leave it off.

**Voice notes:** Operator must decide audience-fit before populating this slide. The deep-vision content is gated for a reason — see NARRATIVE.md.

---

### Slide 11 — Ask / What's Next

**Goal:** what does the operator want from the audience?

**Content slots (operator picks based on context):**
- *Family / friends:* "I want you to understand what I'm working on; I'm not asking for anything beyond that today."
- *Partner / customer:* "Try the developer bundle when it ships; here's how to get early access."
- *Contributor:* "Pick a scenario, run it, contribute. Here's how."
- *Investor (fundraising variant only):* specific raise amount, terms, use of funds, milestones tied to the 1-year arc.

**Voice notes:** Always have an ask, even if it's small. Open conversations need a follow-up step.

---

### Slide 12 — Contact

**Goal:** how the audience follows up.

**Content slots:**
- Repository: https://github.com/Vrooli/Vrooli
- Website: https://vrooli.com
- Operator email: matthalloran8@gmail.com
- Social: @VrooliOfficial (X), @vrooli (YouTube)
- Where to start (link to QUICKSTART or onboarding)

**Voice notes:** Include the easiest possible next step.

---

## Cross-references

- All copy pulls from [`PITCH.md`](PITCH.md), [`NARRATIVE.md`](NARRATIVE.md), and [`FAQ.md`](FAQ.md).
- Visual assets from [`docs/marketing/strategy/ASSETS.md`](../marketing/strategy/ASSETS.md).
- Brand image style for slide backgrounds / visual elements: [`docs/marketing/strategy/IMAGE_STYLE.md`](../marketing/strategy/IMAGE_STYLE.md).
- Long-arc content: [`VISION.md`](../../VISION.md) (operator-authored manifesto).
- Architecture detail (for technical audiences asking "how does this really work"): [`docs/concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md).

## Changelog discipline

When the outline structure changes (new slide, removed slide, reordering), the change is proposed via `brand-guideline-update` decision and accepted by the operator. The operator then revises actual deck content in whatever tool they're authoring it in (Keynote, Google Slides, Pitch, etc.) — this markdown file is the *outline of record*, not the deck itself.
