# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Add the agent-runtime delivery ramp. Scenario to Plugin takes a Vrooli scenario that is already standalone-installable and produces a signed, publishable **Agent Plugin** — an Agent Plugins 1.0.0 folder carrying Agent Skills (`SKILL.md`) and MCP server declarations (`mcp.json`) — then proves that package installs and works in a clean room before any channel receives it. It is the fifth deployment ramp alongside `scenario-to-desktop`, `scenario-to-android`, `scenario-to-ios`, and `scenario-to-cloud`, and it implements the same common ramp contract: consume a target plan, produce artifacts, run target-native validation, emit evidence, and ask `deployment-manager` for the release decision before publishing.
- **Primary users / verticals**: Vrooli scenario owners deciding whether a capability is publishable and preparing it; release operators approving what reaches an external audience; the monetization team measuring the `skill-registries` channel; and — indirectly but decisively — the external coding agents (Claude Code, Codex CLI, Cursor, Copilot, Kiro, Windsurf) that install the published result and must be able to trust it.
- **Deployment surfaces**: Go API (Connect RPC) as the authority; `scenario-to-plugin` CLI as the primary operator surface; React UI for the readiness board, package inspection, evidence review, and publication history; emitted plugin artifacts and attestations as the outward-facing product.
- **Value promise**: Publishing a capability to an agent registry stops being a hand-run supply-chain chore and becomes a gated, evidenced ramp. Every published plugin carries a signature, provenance, an SBOM, a scanner verdict, and — uniquely — a machine-checked guarantee that the commands its skill documents actually exist in the CLI it wraps. In a market where hundreds of thousands of scraped skills carry no review at all, verifiable non-drift is the differentiator that earns registry curation and the trust an external agent needs before it runs anything.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Agent Plugin composition | When an operator requests a package for a scenario that declares plugin content, the ramp shall emit an Agent Plugins 1.0.0 tree whose plugin.json, skills/, and mcp.json validate against the published specification.
- [ ] OT-P0-002 | Skill-to-CLI drift gate | When a package is composed, the ramp shall fail closed if any command shown in a SKILL.md body is absent from the wrapped scenario's pinned cli-manifest.
- [ ] OT-P0-003 | Skill specification conformance | When a SKILL.md is packaged, the ramp shall reject frontmatter that breaks the Agent Skills specification and reject hidden Unicode, bidirectional marks, or angle brackets anywhere in the skill body.
- [ ] OT-P0-004 | Install script safety gate | When a skill ships an install script, the ramp shall reject any download that is not pinned to an immutable reference and verified against a recorded checksum.
- [ ] OT-P0-005 | Supply-chain attestation bundle | When a package passes conformance, the ramp shall produce a Cosign signature, an SLSA provenance attestation, and a CycloneDX SBOM bound to the artifact digest.
- [ ] OT-P0-006 | Clean-room install rehearsal | Before any publication, the ramp shall install the package in an isolated workspace and record a protocol-profile journey proving the declared commands run without the full Vrooli runtime.
- [ ] OT-P0-007 | Gate-bound publication | The ramp shall refuse to publish to any channel until deployment-manager returns a passing release decision for the same source commit and target.
- [ ] OT-P0-008 | Published-version revocation | When an operator revokes a published version, the ramp shall withdraw or flag that artifact in every channel it reached and record the per-channel outcome.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Claude Code distribution adapter | The ramp should publish one composed package to a Claude Code marketplace descriptor without re-authoring any skill content.
- [ ] OT-P1-002 | MCP server declaration | When a scenario declares an MCP server, the ramp should compose mcp.json and record that server's authentication posture in the package report.
- [ ] OT-P1-003 | Per-plugin install attribution | The ramp should attribute installs, registry referrers, and scanner verdicts to a single plugin so channel activation can be decided from evidence rather than argument.
- [ ] OT-P1-004 | Entitlement-aware skills | When a declared skill requires a paid entitlement, the ramp should package a sign-in path that resolves entitlement at run time instead of embedding a credential in the artifact.
- [ ] OT-P1-005 | Fleet publish-readiness board | The ramp should report, for every scenario, which publication prerequisites are met and which one blocks eligibility.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Curated registry submission | The ramp may submit a signed package to curated registries and track each submission's review state.
- [ ] OT-P2-002 | Composite plugins | The ramp may compose one plugin from several scenarios when those scenarios ship as a single external capability.
- [ ] OT-P2-003 | Drift repair proposals | The ramp may propose a SKILL.md edit when a drift check fails against a renamed or removed command.
- [ ] OT-P2-004 | Emerging runtime adapters | The ramp may add distribution adapters for agent packaging standards that reach a stable release after Agent Plugins 1.0.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go API on Connect RPC with proto-first contracts (`packages/proto/schemas/scenario-to-plugin/v1/`); Go CLI on `cli-core` primitives declared in `cli/manifest.json`; React 19 + Vite + Tailwind UI bound to the `vrooli-default` design kit. The ramp implements the `Builder`, `Driver`, and `Distributor` seams of the shared `packages/delivery-ramp-go` spine rather than re-deriving evidence, verdict, or journey semantics.
- Data + storage expectations: SQLite is the scenario-owned store. It holds package records, conformance findings, attestation references, rehearsal journeys, and distribution history. Artifact bytes, signatures, SBOMs, and rehearsal logs stay in the scenario's capture store; the database and every emitted `TargetVerdict` carry references only, never bytes.
- Integration strategy: `deployment-manager` is the governance plane and decides release; this ramp never grants itself release authority. `workspace-sandbox` provides clean-room isolation for install rehearsals. `cli-health` supplies the pinned command surface the drift gate compares against. `scenario-dependency-analyzer` governs every third-party package. `offer-desk` receives channel evidence. `secrets-manager` and the credential authority own registry tokens; the ramp reads references, never literals.
- Non-goals / guardrails: The ramp does not decide whether a capability *should* be published — that judgment stays with the `skill-registries` channel doctrine and the operator. It does not author skill content; skills are owned by the scenario they wrap. It does not make a scenario standalone-installable; that is an upstream prerequisite and this ramp fails closed when it is unmet. It does not implement packaging for other tiers, does not store artifact bytes in `deployment-manager`, and never escalates privilege — every emitted install script targets a user-scoped prefix and no published artifact may request elevation.

## 🤝 Dependencies & Launch Plan

- Required resources: SQLite for scenario-owned state; local filesystem capture store for artifacts and rehearsal logs. No shared Postgres, Redis, Qdrant, or Ollama dependency — a ramp that needed the full resource fleet to publish a standalone capability would contradict its own contract.
- Scenario dependencies: `deployment-manager` (release gate and target verdicts, required); `workspace-sandbox` (clean-room rehearsal isolation, required); `cli-health` (pinned command surface for the drift gate, required); `secrets-manager` (registry credential references, required at publish); `scenario-dependency-analyzer` (package governance, build-time); `offer-desk` (channel evidence, optional); `knowledge-observatory` (documentation indexing of published skills, optional).
- Operational risks: (1) **Trust blast radius** — one flagged plugin damages every other Vrooli plugin's registry standing, so conformance and attestation must fail closed rather than warn. (2) **Standard churn** — Agent Plugins 1.0.0 is weeks old and Claude Code keeps an incompatible native format, so the distributor seam must stay pluggable and no format may be assumed canonical. (3) **Stealth bundling** — a package that quietly pulls the full Vrooli runtime is a trust violation; the rehearsal exists specifically to catch it. (4) **Credential leakage** — registry tokens and any user credential path must never enter an artifact, an SBOM, or a verdict. (5) **Drift decay** — a skill that documents a command the CLI later renames is the most likely long-run failure, which is why the drift gate is P0 rather than a lint.
- Launch sequencing: (1) scenario↔plugin declaration schema and the composition domain; (2) the conformance domain, including the drift gate, so nothing unverified can proceed; (3) the attestation domain over the release-signing authority; (4) the rehearsal domain on `workspace-sandbox` with protocol-profile evidence; (5) the distribution domain with the Agent Plugins adapter and the `deployment-manager` gate handshake; (6) one real pilot published end to end — `workspace-sandbox` or `git-control-tower` — before a second adapter is added; (7) telemetry attribution, which is the `skill-registries` channel's stated activation prerequisite.

## 🎨 UX & Branding

- Look & feel: The `vrooli-default` design kit, unmodified in its binding contract — tokens, type scale, spacing, radius, motion, and status-color semantics. The product tone is a release console, not a dashboard: dense, evidence-first, and calm. The dominant visual idea is a **gate ladder** — composition, conformance, attestation, rehearsal, publication — where every package shows exactly which rung it stands on and precisely what blocks the next one. Status color carries meaning and nothing else: green is earned evidence, amber is a degraded or expiring signal, red is a closed gate, and grey is not-yet-run. A gate never renders green from an absent check.
- Accessibility: WCAG 2.2 AA is the floor and it is tested, not asserted. Every gate state is conveyed by icon and text as well as color, so a red gate is legible to a user who cannot distinguish it from green. All interactive controls are reachable and operable by keyboard with a visible focus ring; the publish and revoke confirmations are focus-trapped and escapable. Live regions announce long-running rehearsal and publication progress. Evidence tables carry real header semantics, and every artifact digest is selectable text rather than an image. Target contrast is 4.5:1 for body text and 3:1 for interface and graphical elements.
- Voice & messaging: Precise and non-promotional. The UI says what was checked, what it found, and what to do next; it never says "secure" or "verified" without naming the evidence behind the word. Refusals explain the closed gate and name the remedy in the same sentence. Nothing is described as published until a channel has confirmed it.
- Branding hooks: Standard Vrooli marks and iconography from the design kit. Published plugin artifacts carry Vrooli provenance identity through the Cosign certificate and SLSA attestation rather than through visual branding — for an agent audience the trust signal is cryptographic, not decorative. The PWA install surface keeps the seeded manifest, service worker, and safe-area tokens valid; product icons are replaced when final branding exists.

## 📎 Appendix

- Agent Plugins 1.0.0 specification — https://agent-plugins.org/specification (published 2026-08-06; TSC from Amazon, Anysphere, Microsoft, OpenAI, Vercel).
- Agent Skills / `SKILL.md` open standard — https://agentskills.io/specification (stewarded by the Agentic AI Foundation).
- `docs/monetization/catalogs/channels/skill-registries.md` — channel hypothesis, activation criteria, anti-patterns, and the recommendation-blindness boundary.
- `scenarios/deployment-manager/docs/decisions/005-governance-plane-boundary.md` — ADR-005, the four-plane split that assigns this scenario the ramp plane.
- `scenarios/deployment-manager/docs/guides/packaging-matrix.md` — the common ramp contract this scenario implements.
- `packages/delivery-ramp-go` — the provider-neutral journey, evidence, verdict, and validation-matrix spine.
- `docs/reference/scenario-to-desktop-evidence-and-tier-contract.md` — the precedent for a root-level cross-plane evidence contract.
