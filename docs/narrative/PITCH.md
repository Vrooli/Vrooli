# Pitch — Canonical Lines

This file is the source of truth for how Vrooli is described in any external context. Everything else (dev logs, blog posts, landing pages, family-explainer conversations, pitch decks, press kits) pulls from here.

**Write rule:** operator-curated via accepted `brand-guideline-update` decisions. Agents propose; do not edit directly.

---

## Motto

> **Software that builds itself.**

Universal — anchors the recursive-intelligence thesis without requiring familiarity. Pairs with the README's existing values tagline ("Your code. Your data. Your hardware. Your control.") — motto plays the philosophical lead, the README tagline plays the values lead.

---

## Universal one-line pitch

> **Vrooli is a self-improving software foundry that runs on your own hardware. Agents build the apps; the apps make the agents better; you keep all of it.**

Works for any audience as a stand-alone description. When in doubt, lead with this.

---

## Audience-tailored leads

When the audience is known and a tailored lead lands harder, swap the universal lead for one of these. The body can stay shared.

### For technical / OSS-contributor audiences

> **Vrooli is a recursive-intelligence platform: AI agents build scenarios (full apps with UI, API, CLI, tests), each scenario becomes a permanent capability the next agent can compose, and the system gets compoundingly better at building software the more you use it. Open source, self-hostable, AGPL.**

Anchors: recursive intelligence, agents-as-builders, scenario tech-tree, OSS framing.

### For control / sovereignty-seeking audiences

> **Vrooli is a self-hosted AI software foundry. Your code runs on your hardware, your data never leaves it, and the agents that automate your work are auditable and replaceable. Cloud LLMs are optional, not required.**

Anchors: local sovereignty, no vendor lock-in, audit-ability.

### For capability-focused audiences (comparing to other agent frameworks)

> **Vrooli is a multi-agent platform where agents don't just call tools — they build scenarios, full apps with their own APIs and CLIs that future agents compose. Compared to OpenClaw, Hermes, OpenHands, or Cline: Vrooli's output is shippable apps, not just executed tasks. Each scenario is both a tool the agent uses and a product the system can sell.**

Anchors: dual-purpose architecture (tool + product), tech-tree compounding, monetization-aware.

### For family / personal-bundle audiences

> **Vrooli is a personal AI server that runs on your own machine. It handles your tasks — calendar, scheduling, automating chores, building little tools you wish existed — and it gets better at helping you the more you use it. Like having a programmer in your pocket who never forgets.**

Anchors: personal automation, accessible framing, "programmer in your pocket" mental model, household relevance.

---

## 30-second pitch

> Vrooli is a self-improving software foundry. AI agents build apps on your own hardware — apps that solve your specific problems and become permanent tools the system can use forever. Every problem solved makes the next problem easier. It's open source, self-hostable, and the apps you build stay yours.

Used for: dev-log openers, podcast intros, casual conversations after the one-liner lands.

---

## 2-minute pitch

> Most AI agent platforms today are ephemeral: you ask them to do something, they do it, and the work disappears. Vrooli is different. When a Vrooli agent solves a problem, the solution gets crystallized as a *scenario* — a full app with its own UI, API, CLI, and tests. The next agent can compose that scenario as a building block. The system grows a tech tree of software it has built and can keep building on.
>
> Three things make this work. First, **recursive intelligence**: agents build the tools that make the next agents smarter. Second, **local sovereignty**: everything runs on your hardware — your code, your data, your control. Cloud LLMs are optional, not required. Third, **agents-as-builders**: the agents themselves are visible and auditable; you can read their decisions, replace them, or watch them improve over time.
>
> What this looks like today: a Go-native control plane (`vrooli`) on Linux/macOS/Windows-via-WSL2; a library of scenarios for everything from web automation to deployment to multi-agent coordination; a local resource layer (Postgres, Redis, Ollama, Qdrant, Browserless) that scenarios compose. The platform is in active pre-1.0 development, AGPL-licensed, monetized through themed app bundles (developer tools, lifestyle/household, and more) plus optional managed-deployment subscriptions for people who don't want to run their own infrastructure.
>
> The long-term arc is bigger — but you don't need to buy into it to find Vrooli useful today. Useful today: a personal AI software-builder that runs on your machine and gets compoundingly better. Useful tomorrow: an open-source platform where any user's improvements can be shared with every other user.

Used for: blog post intros, partner conversations, customer onboarding, longer dev-logs.

---

## Key positioning lines

Stable building blocks. Each is short enough to drop into copy verbatim.

| Line | Use when |
|---|---|
| Software that builds itself. | Motto / universal hook |
| Your code. Your data. Your hardware. Your control. | Sovereignty values tagline (existing on README) |
| Agents build the apps; the apps make the agents better. | Recursive intelligence summary |
| The system literally cannot forget how to solve problems — only get better at solving them. | Compounding-intelligence framing |
| Every scenario is both a tool the agent uses and a product the system can sell. | Dual-purpose architecture |
| Shippable apps, not just executed tasks. | Differentiator vs other agent frameworks |
| Open source. Self-hostable. AGPL. | OSS-credibility one-liner |
| Cloud LLMs are optional, not required. | Sovereignty / lock-in framing |

---

## What Vrooli is NOT

Equally important. These framings keep the pitch grounded and prevent audience confusion.

- **Not a chat product.** Vrooli isn't an LLM front-end like ChatGPT or Claude.ai. It uses LLMs as a substrate; it isn't one.
- **Not competing with cloud LLM APIs.** Vrooli runs on top of Anthropic, OpenAI, OpenRouter, Ollama, and any model endpoint you point it at. It's complementary to LLM providers, not an alternative.
- **Not a no-code automation tool** (n8n, Zapier, Make). Those wire up existing services. Vrooli's agents *build* services from scratch when needed, then compose them.
- **Not just an agent framework** (LangChain, AutoGen, OpenHands, Cline). Those let agents call tools. Vrooli's agents build *scenarios* — full apps with UI/API/CLI/tests that can ship as products.
- **Not a cloud SaaS-first business.** Vrooli's primary delivery target is the user's own hardware. Managed-deployment is a convenience option, not the product.
- **Not a research prototype.** Active pre-1.0 development with a real product roadmap and revenue lines, not a paper or demo.

---

## Notes on usage

- For first-publish content (where the audience hasn't heard of Vrooli before), pair the audience-tailored lead with at least the 30-second pitch — pure motto/tagline isn't enough to ground a stranger.
- Always pair revenue claims with `pending-telemetry` or `aspirational` honesty flags per `STRATEGY.md`. Pre-launch revenue numbers are projections, not measured.
- The post-labor / DAO / peaceful-revolution narrative does NOT live in this file. It lives in [`NARRATIVE.md`](NARRATIVE.md)'s deep-vision bracketed section, marked for vision-aligned audiences only. Do not lead with it externally without the audience already being bought in.
- When competing-framework comparisons land in the conversation (OpenClaw, Hermes, OpenHands, Cline), use the capability-focused lead and emphasize "shippable apps, not just executed tasks" — that's the structural difference.
