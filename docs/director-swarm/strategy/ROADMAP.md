# Roadmap

A thematic lens over Vrooli's goal portfolio. **Swarm Manager is authoritative** for goal status, priority ordering, dependencies, and per-goal detail — consult `swarm-manager goals list` and `swarm-manager goals get --name <name>` for current state.

This doc holds **positioning**: why each goal exists and how it maps to the ranking criteria in [PORTFOLIO_PHILOSOPHY.md](PORTFOLIO_PHILOSOPHY.md). It is grouped by theme rather than ordered, because the portfolio pursues several themes in parallel. Status is never reflected here — check Swarm Manager.

Goal names are `goal:` typed references (`docs/reference/machine-readable-references.md`): swarm-manager owns their existence, this doc owns only their theme and positioning. `portfolio-manager` diffs the live goal list against these references every heartbeat; a goal listed here that no longer exists, or a live goal missing here, is drift and becomes a bounded `goal-portfolio` decision.

Re-baselined 2026-07-24 against the live goal set (operator-approved).

## Themes

Each theme maps to one or more ranking criteria. A healthy portfolio usually has active work across several themes simultaneously.

---

### Theme 1 — Revenue & Delivery

**Goal:** close the distance from zero paying users to first paid apps, with delivery pipelines (desktop, self-host, hosted) that can sustain repeated paid releases.

**Primary criterion:** revenue (1) + quality/auditability co-requirement (2).

| Goal | Positioning |
|---|---|
| `goal:desktop-deploy-v1` | The Tier 1 delivery spine: packaging, release records, approval gating for monetized desktop delivery. |
| `goal:desktop-monetization-assurance` | Stripe-backed paid desktop delivery; payment and fulfillment guardrails. |
| `goal:self-host-v1` | The self-host delivery path — the audience web-console's headliner pitch is accurate for today. |
| `goal:hosted-cloud-tier-foundation` | Tier 3 expansion that opens the phone-anywhere market without invalidating the self-host pitch (decision dec-1777173379756603490, option C). |
| `goal:monetization-v1` | The monetization substrate itself — SKUs, entitlements, purchase flow. |
| `goal:remote-deployment-ops` | Broadens delivery beyond desktop-local once first paid apps prove out. |
| `goal:rapid-approval-flow` | Operator-facing approval velocity — the bottleneck for anything requiring human decisions on releases. |
| `goal:brand-manager-readiness` | Branding for monetized scenarios — required before a paid app reaches real users with a coherent presentation. |
| `goal:landing-page-api-domain-subpackages` | Acquisition-surface plumbing behind the public landing page. |

---

### Theme 2 — Bundle Scenarios (the apps themselves)

**Goal:** get the business bundle's headliner scenarios (`web-console`, `git-control-tower`, `swarm-manager`) to paid-release readiness, and harden the depth-layer capabilities that amplify them.

**Primary criterion:** revenue (1), with indirect quality contribution (2) via scenario hardening.

Cross-reference: [docs/monetization/catalogs/skus/base/business.md](../../monetization/catalogs/skus/base/business.md) for the bundle definition.

| Goal | Positioning |
|---|---|
| `goal:git-control-tower-ai-provenance` | GCT needs AI-provenance surfaces before it's trustworthy as a paid product. |
| `goal:gct-commit-initiative-linking` | Ties GCT to Swarm Manager goals — part of GCT's core headliner story. (Name still carries the retired "initiative" vocabulary; rename candidate.) |
| `goal:gct-github-integration` | Required for GCT to be useful in any real external repo. |
| `goal:gct-merge-and-conflicts` | GCT's most painful current gap. |
| `goal:gct-release-pipeline` | GCT's own release path. |
| `goal:swarm-manager-graph-workspace` | Core UX of the Swarm Manager scenario. |
| `goal:swarm-manager-dashboard` | Surface parity work. |
| `goal:swarm-manager-feature-parity` | Swarm Manager must cover everything the deprecated app-issue-tracker covered. |
| `goal:swarm-manager-quality-gates` | Self-correction for agent-driven Swarm Manager work. |
| `goal:goal-driven-execution-and-estimation` | Goal-scoped execution strategy and estimation — the operator loop's planning depth. |
| `goal:decision-question-visuals` | Visual grounding for decision questions — operator decides faster with mockups than prose. |
| `goal:decision-visual-grounding-propagation` | Propagates decision visuals through the proposal/review loop. |
| `goal:ai-image-generation-foundation` | Image-tools-backed generation plumbing (wrap-not-use) that decision mockups and future image use cases sit on. |
| `goal:initiative-feedback-research-support` | Research entry point for proposal feedback. (Retired-vocabulary name; rename candidate.) |
| `goal:proposal-advanced-diff-ux` | Diff UX for reviewing proposals. (Renamed from retired decision vocabulary.) |
| `goal:workshop-decision-triage` | Workshop decision-sync tail; review against the plan-workshop consolidation before further investment. |
| `goal:audio-reliability-v1` | Shared audio substrate reliability — hands-free mobile use across web-console, swarm-manager, agent-manager. |

---

### Theme 3 — Platform Safety, Auditability & Quality

**Goal:** make the foundation under paid apps reliable, recoverable, and transparent. Every goal here compounds — the fifth paid app ships faster than the first because these are in place.

**Primary criterion:** quality/auditability (2). This theme is how criterion 2 shows up in the portfolio.

| Goal | Positioning |
|---|---|
| `goal:notification-hub-greenfield` | Event-driven notifications — how users learn about releases, issues, and approval requests. |
| `goal:run-level-undo-and-revert` | Recoverability when an agent run produces bad state. |
| `goal:data-backup-manager-v2` | Durable-data protection under every scenario — a paid product cannot lose user data. |
| `goal:persistence-inventory-and-contracts` | Knowing where every byte of state lives is a precondition for backup, migration, and test isolation. |
| `goal:api-surface-manifest-conformance` | Manifest-of-manifests conformance — keeps declared API surfaces honest fleet-wide. |
| `goal:design-language-foundation` | Design-token standard + component library + generation guardrails — quality floor for agent-generated UI. |

---

### Theme 4 — Vrooli Self-Improvement & Outcomes

**Goal:** make Vrooli measurably smarter, faster, and less erring over time. Instrument the system enough that capability gaps surface automatically, not one operator discovery at a time.

**Primary criterion:** meta-optimization (3). Also feeds every other theme by making the agents that execute them more reliable.

| Goal | Positioning |
|---|---|
| `goal:command-center-foundation` | Scaffolding for the outcomes/metrics platform — see [OUTCOMES_CHARTER.md](../evidence/OUTCOMES_CHARTER.md). |
| `goal:command-center-data-layer` | API aggregation + gap-tracking — the `/api/v1/gaps` surface that makes missing capabilities programmatically visible. |
| `goal:command-center-dashboards` | Six themed dashboard pages (Mission Control, The Hive, The Forge, Ledger, Broadcast, Panorama). |
| `goal:director-dashboard-gap-workflow` | Wires director-swarm to monitor Command Center gaps and propose capability backlog items to close them. |
| `goal:dtv-meta-optimization-readiness` | Prep work for the meta-optimization team's readiness milestone. |
| `goal:ecosystem-intelligence-loop` | The recursive-learning-loop plumbing — agents building tools that make agents smarter. |
| `goal:swarm-manager-meta-optimizer` | Swarm-manager's own meta-optimizer team — closes the loop on its execution quality. |
| `goal:prompt-manager-decision-workflow-polish` | Decision-workflow ergonomics in the substrate every team's decisions flow through. |
| `goal:search-hub-corpus-buildout` | AI search over declared capability-work corpora — recall infrastructure for agents. |
| `goal:search-hub-federation-adoption` | Federates existing scenario AI-search features into one retrieval surface. |
| `goal:emulator-platform` | Dedicated emulator platform — future-facing platform play. |

---

### Theme 5 — Ecosystem Apps & Demand Validation

**Goal:** grow the scenario ecosystem beyond the business bundle — consumer/companion apps and cheap market tests that validate demand before deep investment.

**Primary criterion:** revenue (1) via new markets, deliberately staged behind bundle readiness.

| Goal | Positioning |
|---|---|
| `goal:portal` | Chat-first ecosystem front door — the surface that makes the whole ecosystem approachable. |
| `goal:portal-front-door` | Portal's front-door UX slice. |
| `goal:phone-agent` | Voice/phone interface — extends the ecosystem to a hands-free channel. |
| `goal:contact-book-plus` | Relationship-intelligence substrate — a consumer scenario and a data capability other scenarios compose. |
| `goal:inventory-app` | Physical-capability substrate — bridges the ecosystem to real-world objects. |
| `goal:routines-app` | Authority-layer content scenario — recurring-guidance delivery. |
| `goal:bookmark-intelligence-hub-rework-and-ideation` | Idea-candidate pipeline feeding marketing and ideation loops. |
| `goal:lifestyle-demand-validation` | Physical-guide market test — validate demand before building deep. |

---

### Theme 6 — Community Contribution Loop

**Goal:** let outside contributors and outside repos feed work into (and receive work from) the swarm safely — the growth loop that compounds beyond operator capacity.

**Primary criterion:** meta-optimization (3) with a growth flavor; quality gates (2) are the hard co-requirement for anything inbound.

| Goal | Positioning |
|---|---|
| `goal:contribution-outbound-v1-bug-reports` | V1 outbound: file bug reports upstream — the cheapest trust-building contribution. |
| `goal:contribution-outbound-extensions` | Outbound PRs and scenarios — the full outbound loop. |
| `goal:contribution-inbound-triage` | Inbound triage team — safe intake of external contributions. |
| `goal:contribution-verification-isolated` | Isolated verification — untrusted work never touches live state unverified. |
| `goal:contribution-settings` | Settings and onboarding for the contribution loop. |

---

## Meta-notes on this roadmap

- **Drift discipline:** goal rows follow the read-time rule in `docs/agent-system/OPERATING_GRAPHS.md` §"State belongs to scenarios; prose holds judgment" — swarm-manager owns goal existence and status; this doc owns theme and positioning only. The `portfolio-manager` heartbeat diff is the sensor; deltas become `goal-portfolio` decisions.
- **A new goal's theme is decided at proposal time**, as part of the `goal-proposal` decision. If the theme isn't clear, that's a signal the goal isn't well-scoped.
- **Themes can shift.** If a theme becomes vestigial, propose retiring it; if a cluster of unthemed goals emerges, propose a new one. Theme boundaries are a tool, not dogma.
- **Cross-theme dependencies are normal.** The theme tells you why the goal exists; Swarm Manager dependencies tell you what must ship first.

## Updating this doc

Changes go through approved decisions with context `goal-portfolio` or `goal-proposal`. Structural changes (adding/removing themes, re-categorizing many goals at once) warrant an explicit operator review at the vision walk.
