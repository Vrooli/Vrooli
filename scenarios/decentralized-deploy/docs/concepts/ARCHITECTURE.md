# Architecture — Decentralized Deploy

## Scenario Purpose

Record this scenario's actual system shape and durable architecture decisions. Describe the shared React-Vite pattern only where it affects this scenario. Read the scenario's [service declaration](../../.vrooli/service.json) before describing runtime behavior.

## Scenario Shape

The scenario is an external-integration adapter for decentralized storage and naming. Its reusable surface is programmatic (CLI plus API); the generated direct UI is not yet scenario-specific. The first working code is the Go deployment domain in `api/internal/deployment/`: storage/naming contracts, a shared gated-effect policy, fault hooks, orchestration and the Kubo local adapter. It is not wired into API handlers or CLI commands yet.

## System Boundaries

CI1 owns decentralized storage and naming adapters, their devnet lifecycle, operation receipts and the provider/path catalogue. It consumes a compute reference supplied by its caller. Compute Manager owns compute providers; Deployment Manager governs the operation. This scenario must not add a compute-provider or provider-target interface.

The Kubo adapter uses only the exact managed Kubo executable passed to it and an isolated task-owned repository. Repository initialization disables bootstrap peers, swarm listeners and routing. It never opens the operator's default IPFS repository. ENS/anvil, ArNS/arlocal, Turbo, and the shared effect policy's operator approval path remain unimplemented.

## Contracts And Data Flow

The current vertical slice accepts an artifact directory, recursively adds it with pinned Kubo, explicitly pins the returned root, and retrieves a declared relative entry by CID. Renewal re-pins the root. Teardown removes the recursive pin, runs local Kubo GC and removes the repository only when its ownership marker matches. The scenario must expose this as a receipt-bearing operation through its CLI/API before the user-facing lifecycle claim is made.

Provider acceptance and dead ends are read from `data/provider-path-catalog.json`; it is a recorded-fixture catalogue and must not poll provider listings. Unknowns such as BitLaunch remain explicitly unqualified.

## Shared Infrastructure

List shared infrastructure this scenario uses and its boundary. Keep product behavior with its domain owner.

## Extension Rules

Describe where future changes should land. Keep capability, workflow, and persistence ownership in `DOMAINS.md`, `FLOWS.md`, and `DATA.md`.

## Architecture Maturity

Package tests currently pass for gate refusal, provider faults, compensation, catalogue validation and Kubo lifecycle. One test ran managed Kubo 0.43.1 against a task-scoped offline repository and proved exact retrieval of a 22-byte synthetic payload after pinning, followed by renewal and repo cleanup. This proves one local storage adapter. It does not prove API/CLI wiring, name resolution, Arweave/Turbo, provider loss, public IPFS retrieval or the CI1 showcase.

## Ecosystem Fit

- Role: external integration. Interfaces: programmatic CLI/API (other scenarios consume it) and operator UI for lifecycle state and gate readiness. Programmatic completion requires a stable, receipt-bearing deploy/renew/teardown contract; UI completion requires production loading, error and empty states.
- Compound-value seam: replaceable storage and naming interfaces, plus a machine-readable provider/path catalogue; Compute Manager's compute reference stays an input seam.
- Objective: I1 Capability compounding, served by Director Swarm and Scenario QA. No LLM step is needed in the operation path.
- Retirement claim: nothing is currently retireable. Identify exact repository scripts only after the CLI lifecycle is green, then delete those scripts in the same change that replaces them.
- Bundle/capability posture: self-hosted operation remains free/BYOK; any provider-funded action remains gated and is not enabled by this assignment. Monetization choices stay with operator canon.

## Intentional Deviations

Record durable deviations from the shared guide, with a reason and revisit trigger.

## Documentation Architecture

Keep one local source for each concern and link details rather than copying inventories.

## Cross-References

- [Scenario integrations](INTEGRATIONS.md)
- [Scenario data](DATA.md)
