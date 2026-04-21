# Roadmap

A thematic lens over Vrooli's initiative portfolio. **Swarm Manager is authoritative** for initiative status, priority ordering, dependencies, and per-initiative detail — consult `swarm-manager initiatives list` and `swarm-manager initiatives get --name <name>` for current state.

This doc holds **positioning**: which initiatives exist, why they exist, and how they map to the goals in [PORTFOLIO_PHILOSOPHY.md](PORTFOLIO_PHILOSOPHY.md). It is grouped by theme rather than ordered, because the portfolio pursues several themes in parallel. Status is not reflected here — check Swarm Manager.

## Themes

The portfolio organizes into four themes. Each maps to one or more of the ranking criteria in the philosophy. A healthy portfolio usually has active work in at least the first three simultaneously.

---

### Theme 1 — Revenue & Desktop Delivery

**Goal:** close the distance from zero paying users to first paid apps, with a delivery pipeline that can sustain repeated paid releases.

**Primary criterion:** revenue (1) + quality/auditability co-requirement (2).

| Initiative | Positioning |
|---|---|
| `desktop-runtime-interop` | Desktop apps must coexist and override local swarm equivalents cleanly — required for the business bundle's Tier 1 delivery to work at all. |
| `desktop-release-governance` | Approval gating, release records, LPBS release contracts. The spine of monetized desktop delivery. |
| `desktop-monetization-assurance` | Stripe-backed paid desktop delivery; payment and fulfillment guardrails. |
| `rapid-approval-flow` | Operator-facing approval velocity — the bottleneck for anything requiring human decisions on releases. |
| `remote-deployment-ops` | Broadens delivery beyond desktop-local once first paid apps prove out. |
| `brand-manager-readiness` | Branding for monetized scenarios — required before a paid app reaches real users with a coherent presentation. |

---

### Theme 2 — Bundle Scenarios (the apps themselves)

**Goal:** get the business bundle's headliner scenarios (`web-console`, `git-control-tower`) to paid-release readiness, and harden the depth-layer scenarios that amplify them.

**Primary criterion:** revenue (1), with indirect quality contribution (2) via scenario hardening.

Cross-reference: [docs/monetization/catalog/base/business.md](../monetization/catalog/base/business.md) for the bundle definition.

| Initiative | Positioning |
|---|---|
| `git-control-tower-ai-provenance` | GCT needs AI-provenance surfaces before it's trustworthy as a paid product. |
| `gct-commit-initiative-linking` | Ties GCT to Swarm Manager initiatives — part of GCT's core headliner story. |
| `gct-github-integration` | Required for GCT to be useful in any real external repo. |
| `gct-pre-commit-security` | Paid release gate for GCT. |
| `gct-merge-and-conflicts` | GCT's most painful current gap. |
| `gct-release-pipeline` | GCT's own release path. |
| `swarm-manager-graph-workspace` | Core UX of the Swarm Manager scenario. |
| `swarm-manager-dashboard` | Surface parity work. |
| `swarm-manager-feature-parity` | Swarm Manager must cover everything the deprecated app-issue-tracker covered. |
| `swarm-manager-quality-gates` | Self-correction for agent-driven Swarm Manager work. |
| `continuous-audio-platform` | Shared audio-tools scenario — hands-free mobile use across web-console, swarm-manager, agent-manager. |
| `trusted-node-bridge` | Enables bundle scenarios to be trusted across devices. |

---

### Theme 3 — Platform Safety, Auditability & Quality

**Goal:** make the foundation under paid apps reliable, recoverable, and transparent. Every initiative here compounds — fifth paid app is easier to ship than the first because these are in place.

**Primary criterion:** quality/auditability (2). This theme is how criterion 2 shows up in the portfolio.

| Initiative | Positioning |
|---|---|
| `protected-agent-sandboxing` | Agents must not have unconstrained filesystem/network access. Foundational. |
| `agent-sandbox-audit-foundation` | Auditability of what agents actually did in sandbox — prerequisite for trusting autonomous work. |
| `vrooli-events` | Central event bus and policy engine — unifies observability across scenarios. |
| `notification-hub-greenfield` | Event-driven notifications — how users learn about releases, issues, and approval requests. |
| `run-level-undo-and-revert` | Recoverability when an agent run produces bad state. |

---

### Theme 4 — Vrooli Self-Improvement & Outcomes

**Goal:** make Vrooli measurably smarter, faster, and less erring over time. Instrument the system enough that gaps in capability are surfaced automatically, not discovered by the operator one at a time.

**Primary criterion:** meta-optimization (3). Also feeds back into all other themes by making agents that execute them more reliable.

| Initiative | Positioning |
|---|---|
| `command-center-foundation` | Scaffolding for the outcomes/metrics platform — see [OUTCOMES_CHARTER.md](OUTCOMES_CHARTER.md). |
| `command-center-data-layer` | API aggregation + gap-tracking system. The `/api/v1/gaps` surface that makes missing capabilities programmatically visible. |
| `command-center-dashboards` | Six themed dashboard pages (Mission Control, The Hive, The Forge, Ledger, Broadcast, Panorama). |
| `director-dashboard-gap-workflow` | Wires director-swarm to monitor Command Center gaps and propose capability backlog items to close them. |
| `dtv-meta-optimization-readiness` | Prep work for the meta-optimization team's readiness milestone. |
| `ecosystem-intelligence-loop` | The recursive-learning-loop plumbing — agents building tools that make agents smarter. |
| `emulator-platform` | Dedicated emulator platform — future-facing platform play. |

---

## Meta-notes on this roadmap

- **When an initiative is added or removed in Swarm Manager**, `portfolio-manager` should propose an update to this doc assigning the new initiative to a theme or removing the retired one. Don't let the roadmap drift from the live initiative list for long.
- **Themes can shift.** If a theme becomes vestigial (e.g., all its initiatives retire), propose retiring the theme. If a new theme emerges, propose adding one. Theme boundaries are a tool, not dogma.
- **Cross-theme dependencies are normal.** An initiative's Swarm Manager dependencies may point into a different theme; that's fine. The theme tells you why the initiative exists; the dependencies tell you what must ship first.
- **A new initiative's theme is decided at proposal time**, as part of the `initiative-proposal` decision. If the theme isn't clear, that's a signal the initiative isn't well-scoped.

## Updating this doc

Changes go through approved decisions with context `initiative-portfolio` or `initiative-proposal`. Structural changes (adding/removing themes, re-categorizing many initiatives at once) warrant an explicit operator review at the vision walk.
