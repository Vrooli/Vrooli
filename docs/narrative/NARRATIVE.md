# Narrative — The Vrooli Story

This is the canonical project description at multiple depths. Advertisers pull from here when grounding drafts. Different audiences and contexts get different depths.

**Write rule:** operator-curated via accepted `brand-guideline-update` decisions. Agents propose; do not edit directly.

---

## 1-line

Vrooli is a self-improving software foundry that runs on your hardware: AI agents build apps, the apps make the agents smarter, you keep all of it.

---

## 1-paragraph

Vrooli is the project of a builder who's always had more ideas than time to build them. The aim: software that takes ideas in plain language and turns them into running software, then keeps getting better at building software the more it builds. Today that looks like a Go-native control plane (`vrooli`) on your own machine, a library of *scenarios* (full apps with UIs, APIs, CLIs, tests), and a local resource layer (databases, inference, automation) that scenarios compose. Each scenario becomes a permanent capability future scenarios can build on — a tech tree of software the system grows. Open source, AGPL-licensed, monetized through themed app bundles for developers, solopreneurs, and households.

---

## 1-page

### The premise

Most software is built by people writing code, ticket by ticket. Most "AI agents" are ephemeral — you ask them to do something, they do it, and the work disappears. Vrooli is different. When a Vrooli agent solves a problem, the solution gets crystallized as a *scenario* — a full app with its own UI, API, CLI, and tests. The next agent can compose that scenario as a building block. The system grows a tech tree of software it has built and can keep building on.

This produces compounding intelligence. The system literally cannot forget how to solve problems — only get better at solving them.

### How it works

Three layers, each running on your own hardware:

1. **Resources.** Local services that provide raw capability — Postgres, Redis, Qdrant, Ollama, Browserless, secret stores, browser automation, inference. Vrooli orchestrates these without requiring you to wire them up by hand.

2. **Scenarios.** Full apps composed from resources and other scenarios. Each scenario simultaneously serves as a *product* (something a customer or you-personally can use), a *capability* (a tool future agents can compose), and a *test* (it validates that the underlying resources work together correctly).

3. **Agents and teams.** Agents build scenarios, review each other's work, surface decisions for the operator, and coordinate through structured team patterns (director-swarm, marketing-crew, monetization, meta-optimization). The operator steers via a daily morning vision walk; agents handle execution.

### Why this is different

Existing AI agent frameworks (OpenClaw, Hermes, OpenHands, Cline, Aider, AutoGen, LangChain) let agents call tools. Vrooli's agents build *scenarios* — full apps with their own APIs that future agents compose. The output is shippable apps, not just executed tasks. Each scenario is both a tool the agent uses and a product the system can sell.

This unlocks three things existing frameworks don't: a tech tree of compounding capability, a monetization model where every solved problem becomes a potential SKU, and a path where individual user contributions could one day be shared back into a global open-source ecosystem.

### Who it's for

- **Builders / OSS contributors** who want a self-hosted recursive-intelligence platform to extend, fork, and contribute to.
- **Developers and solopreneurs** who want a tool that turns their ideas into shipped software faster than they could build by hand.
- **Sovereignty-conscious users** who want their code, their data, and their automation infrastructure to stay on their own machines.
- **Households** (eventually) who want a personal AI server that handles calendar, scheduling, household automation, and small custom tools without sending their lives to a cloud.

### Where Vrooli is today

Active pre-1.0 development. Public open-source on GitHub (AGPL). Working scenarios for web automation, deployment, agent coordination, backlog management, browser automation, multi-team governance, and dozens more. Local resource layer integrated. Tier 1 (full local stack with secure remote access) is the production-ready path; mobile, desktop, and managed-cloud deployment tiers are in varying stages of maturity. Monetization is structured but pre-revenue — the first themed bundle (developer tools) is in flight; lifestyle and other bundles are next.

### Where Vrooli is going

The near-term arc: the developer/solopreneur bundle ships, generates revenue, validates the bundle-monetization model. Lifestyle bundle follows. The platform improves the platform: agents build more capable scenarios, scenarios make the system more capable, the cycle accelerates.

The long-term arc is bigger and is captured below in the deep-vision section. You don't need to buy into it to find Vrooli useful today.

### Pointer to the operator-authored manifesto

For the philosophical thesis (recursive intelligence, evolution timeline, compound-intelligence effect, the shift from "humans write code" to "AI builds software"), see [`VISION.md`](../../VISION.md) at the repo root. That document is operator-curated and substantively authored.

---

## Deep vision (bracketed — vision-aligned audiences only)

> ⚠️ **Audience-fit warning.** The narrative below is the long-term arc as the operator articulates it privately. It includes post-labor, accelerationist, and conditionally-Marxist-but-decentralized framing. **Do not lead with this externally** for general audiences (technical buyers, family, customers, mainstream press). It will land jarringly with audiences who haven't already been thinking about AI-driven labor displacement seriously, and it can undercut the simpler near-term pitches that resonate broadly today.
>
> **Use this section only when the audience has already self-identified as:** AI futurists, post-labor / UBI advocates, decentralization / DAO enthusiasts, or operators wrestling with how to navigate large-scale automation. For everyone else, stay in the 1-page version above. As the wider conversation around AI-driven unemployment matures (likely 1-3 years out per current accelerated trajectory), this material may become safe for first-tier publication. Until then, gate it.

### The bigger picture

Vrooli is also a thesis about how to navigate a society where AI displaces labor faster than political systems can adapt.

The default trajectory looks like this: AI capabilities advance faster than coordination. Companies automate work and lay off employees, one company at a time, with no shared plan for what happens next. Unemployment among knowledge workers grows. Civil unrest follows, because no one has offered a credible answer to "what do people do when the work goes away?"

Vrooli's wager is that there's a better path: **outpace the disruption with a coordinated alternative.**

The mechanism: open-source, self-hosted, locally-running automation that anyone can extend and share back. As more users contribute scenarios, the global ecosystem grows. Eventually, with the right simulation infrastructure, you can model society-level automation — what would it look like if every sector were automated, who would own what, how would wealth be distributed — and propose tested alternatives to existing economic structures.

The framing is conditionally Marxist in goal (post-capitalist, post-labor) but libertarian in method (decentralize, don't seize). Rather than a centralized authority taking over the means of production, the means of production become so distributed and so cheap that ownership concentrates *less*, not more. DAOs (or their successors) coordinate sectors and distribute the surplus. Universal income emerges as a mechanical property of automation reaching saturation, not as a policy granted from above.

What Vrooli contributes specifically: an open-source platform where the automation work is auditable, shareable, and accumulates. Every scenario shipped reduces the cost of the next one. Every user who contributes makes the global tech tree richer. The platform itself can simulate society-scale automation as the simulation tooling matures.

### The peaceful-revolution framing

Most disruption ends violently. Marxist revolutions notably failed because they had no concrete plan for *what comes next* — centralized powers took over "temporarily" and never let go. Vrooli's wager is that with a credible, simulation-validated, fully-decentralized alternative ready in advance, the transition can happen peacefully: most people would prefer "here's a tested system that distributes the surplus and keeps your living standard intact" over "ride into the streets and hope for the best."

This is not a near-term marketing claim. It's a north-star — what success looks like if everything works. The day-to-day work of Vrooli (shipping scenarios, building bundles, generating revenue, growing OSS contributions) is what makes the long-term plausible. Without those, the deep arc is a manifesto. With them, it's a credible path.

### Why this is private-by-default today

Three reasons:
1. The audience for it is small today. Most potential users don't want a revolution narrative attached to the personal AI server they're considering.
2. There's no proof-of-arc yet. The bundle-monetization step needs to ship and validate before the deeper arc earns the right to be claimed.
3. The framing is intentionally provocative when stated plainly. "Conditionally Marxist accelerationist libertarianism" is a thicket of triggering words; the audience needs context before it lands.

When the wider AI-displacement conversation matures and proof-of-arc exists, this material moves up to first-tier publication. Until then, it lives here, available to advertisers and operators who know how to deploy it for the right audience.

---

## Pointer table

| Need | Use |
|---|---|
| Tweet, dev-log opener, podcast intro | 1-line or 30-second from [`PITCH.md`](PITCH.md) |
| Blog post lede, customer conversation | 1-paragraph |
| Landing page, partner intro, journalist briefing | 1-page |
| Long-form essay aligned with futurist / post-labor audience | Deep vision (gated) |
| Philosophical / aspirational anchor | [`VISION.md`](../../VISION.md) |
| Technical "how it actually works" | [`docs/concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) |
