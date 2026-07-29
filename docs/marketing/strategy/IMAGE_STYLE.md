# Image Style — AI Image Generation Guide

Style directives for AI-generated images (blog hero images, social cards, dev-log accompanying visuals, presentation imagery, scenario UI illustrations). Pulled from by the producer, and any agent generating visual content via OpenRouter image models or similar.

**Write rule:** operator-curated via accepted `brand-guideline-update` decisions. Agents propose; do not edit directly.

**Status:** Markdown stand-in. **Eventually subsumed by the `brand-manager` scenario** when it ships — that scenario will provide structured palette / typography / image-style storage. Until then, this file is the prompt directive of record.

---

## Visual identity at a glance

| Aspect | Value |
|---|---|
| Aesthetic | Futuristic, abstract, neon |
| Mood | Confident builder energy — capable without being aggressive; ambitious without being self-serious |
| Primary palette | Dark blue / deep purple base, with accent neon green |
| Secondary cues | Speed, automation, recursive / compounding shapes (loops, fractals, tech trees) |
| Avoid | Photorealism, stock-photo people, corporate-marketer vibes, overused "AI brain" visual clichés |

---

## Palette

### Primary

| Role | Color descriptor (use in prompts) | Notes |
|---|---|---|
| Base | deep blue / dark navy / midnight | Anchors the composition; most surface area |
| Mid | deep purple / royal purple / electric violet | Transition tone, often used in gradients |
| Accent | neon green / electric green / phosphor green | Highlight / glow / emphasis points only — do not flood |

The base + mid create a dark foundation reminiscent of nighttime / deep-tech / "futuristic dashboard." The accent green pulls focus to a small number of specific elements (a glowing edge, a key shape, a line of motion).

### What this is NOT

- Not pastel
- Not warm-tone (no orange/red/amber leads — only as small contrast points if absolutely needed)
- Not flat / minimalist-monochrome
- Not corporate-blue alone — the purple matters; without it, the palette reads as generic-tech

---

## Style descriptors (for prompts)

When prompting an image model, lead with these descriptors. Approximate prompt template:

```
<subject of the image>, abstract neon style, dark blue and deep purple base
with electric green accent highlights, futuristic mood, glow effects, sense of
speed and motion, no photorealism, no stock-photo people, no corporate clipart
```

Specific descriptor phrases that work:
- "abstract neon"
- "futuristic, glowing accents"
- "dark blue / deep purple base, neon green highlights"
- "sense of speed and motion"
- "compounding / recursive shapes" (when the image is conceptual)
- "fractal-like, nested, tech-tree imagery" (for recursive-intelligence themes)
- "high contrast, dark background"

Phrases to avoid:
- "photorealistic" — Vrooli's visual identity is abstract
- "stock photo," "people in suits," "office workers" — wrong vibe
- "brain made of circuits," "AI robot face," "neural network sphere" — generic AI clichés
- "watercolor," "hand-drawn," "pastel" — wrong palette/style
- "minimalist white background" — wrong foundation

---

## Composition guidance

- **Dark backgrounds preferred.** The palette is built around a dark base; light backgrounds break it.
- **Glow / bloom effects** on accent elements pull focus the right way.
- **Motion lines / streaks** invoke speed (the rabbit-zoom logo motif). Use sparingly.
- **Recursive / nested shapes** for conceptual imagery about self-improvement, tech-tree growth, compounding. Examples: a fractal that contains itself; a network of nodes that builds outward; a tree where branches contain smaller versions of the tree.
- **Negative space matters.** The composition should breathe; don't fill every pixel.

---

## Examples by content type

### Dev-log header image

> *"Abstract neon visualization of an agent system: dark blue and deep purple background, glowing green nodes connected by motion-line edges, a central recursive shape that suggests self-improvement, no human figures, high contrast"*

### Blog hero image (technical)

> *"Futuristic dashboard imagery, dark navy base with deep purple gradient, neon green data streams, sense of speed and accumulation, abstract not photorealistic"*

### Bundle landing page hero

> *"Conceptual representation of a tech tree of software: dark blue background, branching nodes in purple connected by glowing green threads, sense of compounding capability, abstract"*

### Social card (Twitter / OG image)

> *"Bold abstract neon composition, dark blue and purple base with a single bright green accent, sense of motion, room for overlay text in the upper third, no photorealism"*

### Presentation slide background (deck imagery)

> *"Subtle dark gradient (navy to deep purple), faint glow accents in neon green, room for foreground text and diagrams, low visual noise"*

---

## When to use AI imagery vs. existing assets

- **Use existing brand assets** ([`ASSETS.md`](ASSETS.md)) for: logos, favicons, OG image (default), icons, anywhere identity matters.
- **Use AI-generated imagery** for: dev-log headers, blog hero images, bundle-specific or campaign-specific OG variants, presentation slides, conceptual illustrations of recursive intelligence / agent systems.
- **Never replace the logo** with an AI-generated variant. The rabbit + V-R-O-O-L-I letterforms are canon.

---

## Honesty and humans

- **No fake photo-style human imagery.** If an image needs to convey "operator using Vrooli," use abstract or hand-drawn-illustration approaches, not AI-generated photorealism. Photorealistic AI humans tend toward uncanny / corporate-stock and undercut builder voice.
- **Real photos of the operator or contributors** can be used, but go through normal photo channels — not AI generation.

---

## When IMAGE_STYLE.md is updated

Trigger conditions for the brand-manager (member) to propose a `brand-guideline-update`:
- The visual direction shifts (palette change, aesthetic shift)
- The `brand-manager` *scenario* ships, in which case this file becomes a pointer at the scenario's structured storage
- Specific prompt phrases produce repeatedly-poor or off-brand results across multiple drafts (drift signal — propose phrasing fix)

## Cross-references

- [`docs/marketing/strategy/ASSETS.md`](ASSETS.md) — canonical brand assets (logos, fonts, etc.)
- [`docs/marketing/strategy/BRAND.md`](BRAND.md) — visual identity overview
- [`docs/marketing/strategy/STRATEGY.md`](STRATEGY.md) — voice canon (the linguistic counterpart to this file's visual canon)
- [`scenarios/prompt-manager/store/skills/packs/core/brand-manager/SKILL.md`](../../../scenarios/prompt-manager/store/skills/packs/core/brand-manager/SKILL.md) — planned scenario that subsumes this file
