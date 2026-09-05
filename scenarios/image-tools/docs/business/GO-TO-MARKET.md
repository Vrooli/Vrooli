# Go To Market — Image Tools

This document records launch strategy, positioning, channels, and
validation experiments for the scenario. Everything below is an early,
pre-implementation hypothesis derived from `PRD.md`, not a committed plan.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

- **Audience:** internal Vrooli scenarios and agents first (the near-term
  consumers), then prosumers/creators, small marketing teams,
  developers/automation builders, and privacy-sensitive users (legal,
  healthcare, enterprise).
- **One-line positioning:** "A complete, local-first image toolbox — pro
  editing plus on-demand AI (generate, upscale, restore, remove objects,
  analyze) that runs on your hardware and via your own keys; every operation
  scriptable from the CLI."
- **Against cloud editors:** private by default (images never leave the
  machine), no per-operation lock-in, and fully automatable. Cloud editors
  exfiltrate source images and meter usage; image-tools does neither.
- **Against heavyweight desktop suites:** lighter, headless-complete, and
  composable — usable from a CLI/API and embeddable inside other scenarios,
  not a GUI-only monolith.
- **Main claim (to be tested):** other scenarios and agents adopt image-tools
  as their image layer instead of reimplementing it.
- **Proof needed:** demonstrated internal consumption plus working
  local-first AI ops on commodity hardware with a graceful BYOK fallback.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Internal scenario reuse | Marketing/content scenarios compose image-tools as their image layer | Stable internal API contract, embeddable UI component, integration examples | # of consuming scenarios and call volume |
| Vrooli platform / scenario catalog | Listed as a reusable `*-tools` capability for the fleet | Scenario listing, capability labels, demo recipes | Discovery + first-use by new consumers |
| Developer / automation audience | CLI + API attract pipeline and watch-folder users | CLI docs, API reference, batch/watch examples | CLI/API invocation counts, recipe creation |
| Desktop app + open standards | Local-first toolbox packaged standalone; Claude Skill publication teaches external agents to use it | Desktop/PWA build, published SKILL.md | External installs / skill usage (later) |

## Launch Motion

1. Land the prerequisite (extend the root CLI host-inventory contract for
   GPU/VRAM/CPU/os + build the image-tools `capabilities` seam consuming it via
   `packages/vrooli-cli-go`) and the deterministic op core + storage + job
   queue + model-registry spine.
2. Ship as an **internal capability** consumed by marketing scenarios
   (campaign-content-studio, landing-page-business-suite) and palette-gen —
   dogfood before any external push.
3. Bring core AI ops online via standalone backends with hardware-aware
   selection, CPU fallback, and BYOK tier; prove local-first AI on commodity
   hardware.
4. Expand breadth (P1 ops, recipes, batch, watch-folder), then provenance and
   extras, surfacing the toolbox to developer/automation audiences via CLI + API.
5. Only after the role and adoption are clear, package **standalone**
   (desktop/PWA/SaaS edge) and publish open-standard assets (Claude Skill).
   Sequence mirrors the PRD launch sequencing.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| Private by default — your images never leave your machine | Privacy-sensitive users, prosumers | Local-first deterministic + local-model AI execution | hypothesis |
| Headless-complete and automatable — every op scriptable from the CLI/API | Developers, automation builders | Full CLI parity (OT-P0-013), batch/watch-folder | hypothesis |
| Hardware-aware — runs on a laptop or a GPU box | All segments | CPU-capable defaults + GPU selection + fallback chain | hypothesis |
| No vendor lock-in — own your outputs and your keys | Teams, enterprise, prosumers | User-owned storage, BYOK with pass-through pricing | hypothesis |
| AI on your terms — local models or your own cloud keys | Prosumers, teams | Local → CPU → BYOK fallback with pre-op cost estimate | hypothesis |
| Build once, reuse forever — one image layer the whole fleet composes | Internal scenarios/agents | Embeddable component + stable API contract | hypothesis |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Internal-consumption adoption | Internal scenario reuse | ≥1 marketing scenario consuming image-tools in production paths | If met, prioritize the embeddable-component / API contract; if not, revisit the primitive's ergonomics |
| Op usage telemetry | CLI + API | Sustained op volume with a clear deterministic-vs-AI mix | Invest in the most-used op families; deprioritize unused breadth |
| Local-vs-BYOK split | AI ops | Measurable share of AI ops run locally vs. BYOK cloud | High local share validates local-first messaging; high BYOK share signals hosted-GPU demand |
| Demand for paid/hosted tiers | Developer + prosumer feedback | Repeated requests for managed catalog / hosted GPU / team features | If met, build the corresponding paid tier per `MONETIZATION.md`; otherwise keep deferred |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
