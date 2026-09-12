# FAQ — Canonical Q&A

Approved answers to the questions Vrooli actually gets — from family, friends, technical readers, customers, journalists, partners. Pulled from by advertisers and the operator when answering external questions.

**Write rule:** operator-curated via accepted `brand-guideline-update` decisions. Agents propose new entries when a question pattern emerges across drafts or conversations; operator approves wording.

---

## What is Vrooli?

Vrooli is a self-improving software foundry that runs on your own hardware. AI agents build apps (we call them *scenarios*), each scenario becomes a permanent capability future scenarios can compose, and the system gets compoundingly better at building software the more it builds. Open source, self-hostable, AGPL.

Short version: software that builds itself, on your machine, and you keep all of it.

## How does it actually work?

Three layers running locally:

1. **Resources** — local services that provide raw capability (Postgres, Redis, Ollama, Qdrant, Searxng, etc.).
2. **Scenarios** — full apps composed from resources and other scenarios. Each has its own UI, API, CLI, tests.
3. **Agents and teams** — AI agents build, review, and improve scenarios, coordinated through structured team patterns. The operator steers via a daily vision walk; agents execute.

A Go-native control plane (`vrooli`) orchestrates everything. You install once, and the system runs as a local foundry.

## How will Vrooli make money?

Multiple revenue lines, in priority order:

1. **Themed app bundles.** Sets of related scenarios sold as SKUs — first the developer/solopreneur bundle (Landing Page Business Suite, Git Control Tower, Web Console, etc.), next the lifestyle/household bundle, with more candidates queued. Buyers get the apps, the source, and the right to extend.
2. **Managed deployment subscriptions.** For users who want the apps but don't want to run their own infrastructure — Vrooli runs the stack on their behalf. The shovels we sell are the shovels we use.
3. **Direct scenario deployments.** Custom builds for specific business needs, packaged from existing scenarios.
4. **Hardware (long-term).** Specialized servers for engineering, science, finance, household — appliances that run a Vrooli stack out of the box.

The principle: subscription is convenience, not a paywall. Every scenario is self-hostable for free. Free / self-host users are brand credibility, not lost revenue.

See [`path:docs/monetization/`](../monetization/) for the full plan.

## How long until it makes money?

Pre-revenue today. The developer/solopreneur bundle is the first active SKU effort; LPBS is its hero app. Concrete commercial timing depends on the bundle reaching shippable maturity (one or more apps in the bundle deployable end-to-end, marketing infrastructure ready, payment + onboarding wired). Until that ships, all revenue numbers in any document carry `pending-telemetry` or `aspirational` honesty flags — no claimed-but-not-measured numbers.

Realistic horizons: first paying revenue from the developer bundle is the near-term milestone. Lifestyle bundle is the next candidate. The platform's compounding-intelligence property is structurally designed to make each new bundle cheaper to ship than the last; we'll know whether that compounding is real once the second and third bundles ship.

## Is it open source?

Yes. AGPLv3 license. The full source is on GitHub. The license is deliberate: improvements stay visible, networked deployments remain accountable, community capability cannot be silently enclosed.

Self-hosting is supported and encouraged. Subscription buyers get the same source — they're paying for convenience and managed infrastructure, not for code access.

## Can I use it now? What's the maturity level?

Active pre-1.0 development. Tier 1 (full local stack with secure remote access) is production-ready for developers and operators willing to run their own infrastructure on Linux, macOS, or Windows-via-WSL2. Mobile, desktop, and managed-cloud deployment tiers are in varying stages of maturity.

Useful today: as a local development and orchestration environment. As a personal AI software-builder. As a substrate for building custom scenarios.

Not yet useful for: non-technical end users (no managed-cloud SKU yet), production-critical business workflows without operator-level technical comfort, mobile-first users.

See [`docs/QUICKSTART.md`](../QUICKSTART.md) and [`path:docs/deployment/`](../deployment/) for current operating reality.

## How is Vrooli different from agent frameworks like OpenClaw, Hermes, OpenHands, or Cline?

These are all real, capable, well-designed tools. Vrooli is structurally different in three ways:

1. **Output shape.** Other agent frameworks let agents *call tools and execute tasks*. Vrooli's agents *build scenarios* — full apps with UI, API, CLI, tests. The output is shippable software, not just task completion.
2. **Compounding via tech tree.** Each scenario is also a building block future scenarios can compose. The system grows a tech tree of software it has built. Other frameworks have skill systems (Hermes especially has a notable learning loop), but the unit of accumulation is "agent capabilities," not "shippable apps."
3. **Monetization-aware architecture.** Each scenario is simultaneously a product (something you can sell), a capability (something an agent can use), and a test (it validates the underlying stack). Other frameworks treat monetization as out-of-scope.

Vrooli is also coordinated multi-team (director-swarm, marketing-crew, monetization, meta-optimization, scenario-qa) with structured decision flows and human-in-the-loop vision walks — not a single-agent or simple multi-agent loop.

You could plausibly use OpenClaw or Hermes alongside Vrooli — they're not strict competitors. But if you want shippable apps as your output, Vrooli's substrate is built for it.

## Do I need to bring my own LLM, or does Vrooli include one?

Vrooli is model-agnostic. Bring your own. Cloud LLMs (Anthropic, OpenAI, Google) work; local models via Ollama / LM Studio / any OpenAI-compatible endpoint work; routing through OpenRouter works.

Cloud LLMs are optional, not required. If you want all inference on your hardware, point Vrooli at local models only. If you want the speed/quality of frontier cloud models for some tasks and local for others, that's also supported.

## How does Vrooli handle privacy and data sovereignty?

Default posture: your code, your data, your hardware, your control. Vrooli's primary delivery target is your own machine. Data doesn't leave it unless you configure a cloud LLM or external service (and even then, only the prompts you send to it).

For users who choose managed-deployment, the same privacy posture applies — Vrooli runs the stack on infrastructure you can audit, with the same source you'd self-host.

## Who's behind it?

Vrooli is the project of a solo builder (Matt Halloran) — an engineer who's always had more ideas than time to build them and is building toward software that turns ideas into shipped apps. Pre-launch; pre-team in the traditional sense; agents handle most of the day-to-day execution under operator steering.

This is intentional, not a constraint. The point is to demonstrate that a small operator with the right substrate can run a software foundry at the throughput of a much larger company. Once the bundle ships and the model is proven, scaling looks different than a typical startup — more like a federation of contributors than an employee headcount.

## Why "Vrooli"? Is it named after something?

The name is invented; not an acronym. The logo is a rabbit (after the operator's pet rabbit, Jeff) shaped from the letters V-R-O-O-L-I — speed (rabbits are fast, the rabbit appears to be zooming) plus a small visual easter egg. The motto: "Software that builds itself."

## What's the long-term vision — the really ambitious version?

The near-term arc is concrete: ship bundles, validate the model, grow the OSS contributor base, expand into more domains and deployment tiers.

The longer-term arc is a thesis about how an open-source, self-hosted, locally-running automation platform could help society navigate AI-driven labor displacement more peacefully than the default trajectory — by giving anyone the tools to share automation back into a global ecosystem rather than only large companies capturing the gains. That arc is captured in [`NARRATIVE.md`](NARRATIVE.md)'s deep-vision section, gated for vision-aligned audiences. Most users don't need it to find Vrooli useful.

## Can I contribute? How?

Yes — the project is open source and contributions are central to the long-term plan. The high-leverage areas today: scenario quality and completeness, resource integrations, deployment intelligence, testing and validation infrastructure, documentation and operator workflows, core control plane improvements.

Start with [`docs/CONTRIBUTING.md`](../CONTRIBUTING.md) and the [`docs/README.md`](../README.md) hub. Run the project locally; pick a scenario or area that resonates; the rest is normal OSS contribution.

---

## Adding to this FAQ

Trigger conditions for the brand-manager (member) to propose a new entry:
- Same question shows up in ≥3 drafts, conversations, or external interactions
- New SKU shipping creates a new common question
- Operator explicitly flags a question as canon-worthy

Process:
1. Brand-manager raises a `brand-guideline-update` decision proposing the new Q&A entry, citing where the question pattern was observed.
2. Operator accepts / refines / rejects.
3. On acceptance, the entry is added here in the appropriate section. Commit message cites the decision id.
