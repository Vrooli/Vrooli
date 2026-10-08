# Audio Tools

Shared speech-to-text, text-to-speech, summarization, audio processing, and
provider routing for Vrooli applications. The Go API, Go CLI, React UI, and
shared browser capture package expose different surfaces of the same capability.

For existing-scenario work, read the [product targets](PRD.md),
[architecture](docs/concepts/ARCHITECTURE.md), and
[known limitations](docs/internal/PROBLEMS.md). Use
[START-HERE](docs/START-HERE.md) for initialization work, not as evidence that
existing product domains are placeholders.

## Capability and development status

Local and BYOK adapters, streaming transports, Dictation Studio, experiments,
and shared consumer integration exist. Their presence does not qualify every
provider, device, or recovery path. The
[integration inventory](docs/concepts/INTEGRATIONS.md) identifies current wiring;
the [testing contract](docs/internal/TESTING.md) defines the evidence needed.

The owned subscription/credits route is an implementation target, not a working
fallback for users without local models or keys. See
[monetization](docs/business/MONETIZATION.md) for its gaps and owner boundaries.

Audio Tools pilots [contract-driven scenario development](../../docs/agent-system/SCENARIO_DEVELOPMENT.md).
Read the [portable voice target](docs/concepts/ARCHITECTURE.md#development-target-portable-streaming-voice)
and [pilot decision sheet](docs/internal/TESTING.md#pilot-decision-sheet-and-safe-first-slice).
The `portable-voice-v1` PRD now covers the full intended voice capability. Its
scope is documented; numeric SLOs, release cohorts, billing policy details, and
execution allowances still need the decisions listed in the testing contract.
The historical local-only proposal is a subset, not authority or full completion.
Use `prompt-manager skill read audio-tools` for operation selection and
`prompt-manager skill read audio-tools-improve` for authorized development.
The declared `audio-tools.setpoint-read` program reads engine/health inventory and
an optional selected experiment. Its `ok` result means collection succeeded, not
that streaming, device, quality, or billing targets passed. See the
[measurement contract](.vrooli/program-runtime/setpoint-read.json).
Version 2 lists all 15 PRD targets and keeps every outcome unknown. It still
lacks owner-backed acceptance joins; target visibility is not target achievement.

For the full development goal, start with the
[mandate and completion contract](docs/internal/TESTING.md#full-mandate-and-completion-contract).
One approved engagement can perform successive in-scope repairs; it does not need
a new backlog item for each experiment. This documentation is not a launched
engagement, a spending grant, or proof that Swarm execution is qualified.

## Running The Scenario

```bash
# Start API + UI in the background
make start   # wraps `vrooli scenario start`
```

See [`docs/QUICKSTART.md`](docs/QUICKSTART.md) for the full clone-to-running flow.

Select validation scope under [the testing policy](../../docs/TESTING.md):

```bash
vrooli scenario test audio-tools --phases unit
```

Use the runner's returned wait command once. A broad suite is not the default
for a focused change. Dependency installation follows
[package governance](../../docs/package-governance.md), not raw package-manager commands.

## Documentation Map

| Need | Start Here |
|---|---|
| Initialize after generation | [`docs/START-HERE.md`](docs/START-HERE.md) |
| Establish UI design language | `DESIGN.md` at this scenario's root |
| Run the scenario | [`docs/QUICKSTART.md`](docs/QUICKSTART.md) |
| Understand the architecture | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Map product domains | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Track workflows, data, and integrations | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md), [`docs/concepts/DATA.md`](docs/concepts/DATA.md), [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| Capture monetization and launch strategy | [`docs/business/MONETIZATION.md`](docs/business/MONETIZATION.md), [`docs/business/GO-TO-MARKET.md`](docs/business/GO-TO-MARKET.md) |
| Prepare deployment and operations | [`docs/operations/DEPLOYMENT.md`](docs/operations/DEPLOYMENT.md), [`docs/operations/RUNBOOK.md`](docs/operations/RUNBOOK.md), [`docs/operations/OBSERVABILITY.md`](docs/operations/OBSERVABILITY.md) |
| Write tests | [`docs/internal/TESTING.md`](docs/internal/TESTING.md) |
| Add or update seams/fakes | [`docs/internal/SEAMS.md`](docs/internal/SEAMS.md) |
| Configure env vars, ports, CLI config | [`docs/reference/configuration.md`](docs/reference/configuration.md) |
| Add API endpoints | [`docs/reference/api-endpoints.md`](docs/reference/api-endpoints.md) |
| Add CLI commands | [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |

## Working Rules

1. **Identify the active contract and authority.** A development mandate permits successive in-scope repairs; a review request does not.
2. **Use `make orient` for initialization status.** It does not establish product readiness or reset completed product work.
3. **Preserve approved targets.** Route PRD/requirement changes through their owners; do not rewrite targets to match current code.
4. **Read root `DESIGN.md` before UI work.** Tokens, motion, and status semantics are binding; specific component lists in the design are illustrative — implement everything your scenario actually needs.
5. **Update the owning design documents** when an authorized change affects their contracts.
6. **Keep `docs/manifest.json` accurate.** Durable docs should be registered there with a truthful maturity value.
7. **Record work in its existing engagement or progress log.** Follow shared Memory's learning contract; do not create a second per-repair ledger.
8. **Read declared dependencies** in `.vrooli/service.json`; optional resources do not guarantee an available provider.
9. **Repair at the owning boundary.** Shared capture, resource, proto, or billing changes require the active grant; do not copy them into this scenario to evade scope.
