# Architecture

## Purpose Of This Document

The system map for **meta-optimization-manager**: its shape, boundaries, contracts, and extension rules. Read this before changing structure. The defining property is that this scenario is a **thin, read-mostly aggregator** — it measures the project's readiness for local coding agents by reading other scenarios' typed outputs, and it never re-implements measurement, makes the improvement, or makes a judgment call.

## Scenario Shape

- **Surfaces**: Go API (Connect-RPC), Go CLI (cli-core `ScenarioApp`, typed proto-JSON output), React + Vite + Tailwind UI (operator console, P2).
- **Domains**: `coverage`, `convergence`, `focus`, `trials` (see [DOMAINS.md](DOMAINS.md)).
- **Owned state**: minimal — SQLite (via `api-core/storage`) holding the gaps registry, the trials history time-series, a cached fitness-audit index, and short-TTL coverage snapshots. The denominators (space docs) and the numerators (live coverage) are **not** owned here.
- **Altitude**: this is "ecosystem-manager one level up" — EM measures + steers a single scenario's maturity; this measures + steers the *fleet's self-optimization readiness*.

## System Boundaries

**In scope (measure + surface + route):** aggregate per-projection coverage with denominator-confidence; validate base-document integrity; measure template/reference convergence; maintain the gaps registry and rank focus; run empirical local-model trials and track their trend.

**Out of scope (kept elsewhere on purpose):**
- **Re-implementing measurement** — test-genie, prompt-manager graph-health, completeness-scoring, code-facts, scenario-auditor stay the owners.
- **Doing the improvement** — improving skills/scenarios/templates stays agentic, *directed by* focus.
- **Owning the denominators** — the space docs live with their owner scenarios; this scenario reads them via the `space` verb.
- **Making judgment calls** — it surfaces numbers + candidates, never the substrate-extraction / tiering / nomination / root-cause decision.
- **The judgment-heavy meta-optimization themes** — skill/action lifecycle, friction intake, contrarian review, team/agent structural audits stay agentic; programmatizing them would weaken them.

## Contracts And Data Flow

- **Upstream read contract — `space --projection <p> --json`**: every denominator owner (search-hub → Answer, test-genie → Validate, prompt-manager → Guide) exposes this verb; `coverage` reads it as the denominator. This is the scenario's primary shared-contract dependency.
- **Consumed — the attestation contract** (`AttestedAnswer` on search-hub `SearchHit`): answers carry basis × sufficiency; this scenario consumes attested answers and re-exposes its own (P2).
- **Read clients (typed CLI/RPC, never re-run):** `test-genie health` + `fleet status`, `prompt-manager graph health`, `completeness-scoring GetScore`, `search-hub providers list`, `code-facts` + `architecture-cartographer` (convergence structure), `scenario-auditor` (clean scans), `agent-manager` + `workspace-sandbox` (trials).
- **Data flow (status):** read each projection's denominator (space verb) → read the live numerator (registries) → join → compute coverage + denominator-confidence → attach latest trial trend → return one scoreboard. Every read degrades gracefully; a down source yields "unavailable", never a failed snapshot.

See [DATA.md](DATA.md) for the persisted shapes and [FLOWS.md](FLOWS.md) for the step-by-step flows.

## Shared Infrastructure

- `api/internal/server/` — composition root + HTTP server wiring.
- The **space-reader client** + per-scenario **read clients** — typed adapters over each upstream's CLI/RPC, each with a test double (see [../internal/SEAMS.md](../internal/SEAMS.md)).
- `api-core/storage` — SQLite access for the owned state.
- `packages/proto/schemas/meta-optimization-manager/v1/<domain>/` — the wire contracts.
- cli-core `ScenarioApp` + `api-core` — CLI/HTTP plumbing shared across the fleet.

## Extension Rules

- A new domain measures by **reading**; it never re-implements an owner's measurement.
- A new read source is a **typed client with graceful degradation** — never a hard dependency that can fail the snapshot.
- **Never persist the numerator** — coverage is always computed live.
- **Always pair a coverage number with its denominator-confidence** — the scoreboard must not imply false completeness.
- Business logic in `api/internal/<domain>/`; wire contracts in proto; CLI/UI are translation layers.

## Architecture Maturity

**Draft (documentation-first).** The charter (PRD), requirements registry, domain map, and this architecture are authored; no domain code exists yet. The first real vertical slice (Gate 6) is `coverage`. The example `notes` domain still ships and will be removed via `vrooli scenario detemplate` once `coverage` is green.

## Intentional Deviations

- **New `spaces` docs section** added to search-hub / test-genie / prompt-manager to host the denominator space docs (a net-new artifact class).
- **Convergence coordinated-edit walkthrough** is a genuinely new mechanization (no existing engine); its findings are marked lower-confidence until proven.
- **Stewardship / intake / contrarian / team-audit themes excluded** by design — judgment, not measurement (see System Boundaries).

## Documentation Architecture

- Charter: `../../PRD.md`; design contract: `../../DESIGN.md`.
- Concepts: this file, [DOMAINS.md](DOMAINS.md), [COVERAGE-MODEL.md](COVERAGE-MODEL.md) (the keystone the three space docs reference), [DATA.md](DATA.md), [FLOWS.md](FLOWS.md), [INTEGRATIONS.md](INTEGRATIONS.md).
- Internal memory: [../internal/SEAMS.md](../internal/SEAMS.md), [../internal/TESTING.md](../internal/TESTING.md), `../internal/DECISIONS.md`, `../internal/PROBLEMS.md`.

## Cross-References

- [DOMAINS.md](DOMAINS.md) · [FLOWS.md](FLOWS.md) · [DATA.md](DATA.md) · [INTEGRATIONS.md](INTEGRATIONS.md)
- [../internal/SEAMS.md](../internal/SEAMS.md) · [../internal/TESTING.md](../internal/TESTING.md)
- [COVERAGE-MODEL.md](COVERAGE-MODEL.md) — the canonical model + legend.
