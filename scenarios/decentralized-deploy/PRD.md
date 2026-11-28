# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario decentralized-deploy`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- Purpose: Provide a reusable scenario-to-decentralized-storage-and-naming deployment pipeline with explicit effect gating, renewal, teardown, and operation evidence.
- Primary users/verticals: Vrooli operators and scenarios that need portable decentralized publication.
- Deployment surfaces: CLI and API first; direct UI for lifecycle visibility; composition by other scenarios through stable CLI/API contracts.
- Value promise: Turn a service artifact into a devnet deployment that can be resolved, renewed, removed, and audited without provider-specific caller code.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Devnet deployment lifecycle | One CLI operation deploys a reference site and small API through storage and naming adapters on local devnets; resolve, renew and teardown complete and leave no scenario-owned artifacts.
- [ ] OT-P0-002 | Effect gate and refusal behavior | Every funded, public or identity-bound adapter effect is denied in stub mode; live mode remains unavailable until an operator runbook grants it, and injected provider refusal/timeouts are surfaced with recoverable receipts.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Free sandbox upload and ephemeral key closure | A free-sandbox storage path accepts synthetic content through an ephemeral signing key; exact retrieval is verified and the private key is removed from its task-owned path.
- [ ] OT-P1-002 | Replaceable decentralized adapters | Kubo/IPFS, Arweave/arlocal/Turbo, ENS/anvil and ArNS are expressed behind replaceable storage and naming contracts with local devnets and a fixture-backed provider/path catalogue.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Cross-provider recovery and monitoring handoff | Deployment references exportable content, naming and recovery metadata; a future CI2/CI7 consumer can score linkage, provider loss, substitution and restore time.

## 🧱 Tech Direction Snapshot

- Preferred stacks/frameworks: Governed React-Vite scenario template with Go API and CLI; deterministic adapters and machine-readable provider/path catalogue.
- Data and storage expectations: SQLite for deployment records; content files and keys remain in task-scoped paths, with exportable content identifiers.
- Integration strategy: Storage and naming adapter interfaces; local Kubo, arlocal and anvil devnets; free Turbo sandbox path; Compute Manager supplies a compute reference and owns compute providers.
- Non-goals/guardrails: No compute provider interface, funded leases, name purchases, mainnet payments, public launches, provider contact, account creation, provider terms acceptance, or external anonymity claims. One shared stub/live gate blocks every funded, public or identity-bound effect.

## 🤝 Dependencies & Launch Plan

- Required resources: Pinned host Kubo tool for local IPFS devnet; local anvil and arlocal runtimes must be declared and lifecycle-managed by this scenario when added.
- Scenario dependencies: Compute Manager owns compute and supplies only a compute reference; Deployment Manager governs the deployment operation.
- Operational risks: Provider shutdown/refusal, incomplete CAR roots, gateway/resolver dependency, key cleanup failures, hidden provider identity requirements, and test/devnet port/resource contention.
- Launch sequencing: Establish generated scenario contract; implement Kubo storage deploy/renew/teardown; add ENS/anvil and ArNS/arlocal; implement shared effect gate and fault hooks; add Arweave/Turbo and provider catalogue; verify cleanup and Test Genie phases. Only synthetic devnet/free sandbox effects are in this work.

## 🎨 UX & Branding

- Look and feel: An operator console for named deployment operations, provider readiness, lifecycle state, and retained evidence; keep the UI secondary to the reusable CLI/API.
- Accessibility: Keyboard-complete controls, WCAG AA contrast, clear focus, status announced semantically, and loading/error/empty states.
- Voice and messaging: Concise, factual lifecycle receipts that distinguish stubbed, devnet, sandbox and live effects; do not describe privacy or permanence beyond observed evidence.
- Branding hooks: Use the generated Vrooli design system and replace all placeholder surface copy with deployment-specific language.

## 📎 Appendix

Mission: crypto-infrastructure/MISSION.md, CI1. Prior evidence: runs/research-renewal-20261027T043008Z/FINAL.md and runs/research-renewal-20261026T131100Z/evidence/E1,E4/build.py. Scenario boundary: storage and naming only; Compute Manager owns compute providers.
