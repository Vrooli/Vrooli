# API Health

API Health is Vrooli's delegated provider for scenario API readiness. It validates whether a scenario's API surface is lifecycle-compatible, exposes a trustworthy health endpoint, follows low-ambiguity HTTP semantics, and avoids API-runtime footguns.

The scenario was generated from `react-vite` with the `vrooli-default` design kit. It keeps the standard full-stack shape:

## What You Get

- Go API (`api/`)
- Go CLI (`cli/`)
- React + TypeScript + Vite UI (`ui/`)
- Proto contracts relocated to `packages/proto/schemas/api-health/`
- Lifecycle metadata in `.vrooli/service.json`
- Provider maturity contract in `.vrooli/maturity.json`
- Requirements registry in `requirements/`

## What It Owns

- The shared `ScenarioValidationService.ValidateScenario` provider surface for API readiness.
- API lifecycle checks: service health metadata, `api-core/preflight`, and `api-core/server`.
- Static and live `/health` validation against the `api-core/health` response contract.
- HTTP response semantics for supported first-party patterns: status codes, content types, and versioned REST feature endpoints.
- API-runtime hygiene checks: outbound HTTP timeouts, response body close discipline, request context propagation, cancellable long-running goroutines, and structured API logging.
- Deterministic fix preview/apply for unambiguous local repairs.

## What It Does Not Own

- Static lint/type policy: `quality-health`.
- Security headers, CORS, vulnerability scanning, and dependency security: `security-health`.
- Proto breaking-change validation: `proto-health`.
- CLI manifest correctness: `cli-health`.
- UI runtime/render/interop behavior: `ui-health`.
- Storage isolation and migration safety: `storage-manager`.
- Load/performance benchmarking: `performance-health`.

## Local Commands

```bash
make orient
make test
vrooli scenario requirements validate api-health
```

Planned CLI surface:

```bash
api-health validate scenario <target>
api-health probe health <target>
api-health fix preview <target>
api-health fix apply <target>
```

## Documentation Map

| Need | Start Here |
|---|---|
| Product contract | [`PRD.md`](PRD.md) |
| Domain ownership | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| System architecture | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Workflows | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md) |
| Data/storage | [`docs/concepts/DATA.md`](docs/concepts/DATA.md) |
| Integrations | [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| API endpoints | [`docs/reference/api-endpoints.md`](docs/reference/api-endpoints.md) |
| CLI commands | [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |
| Progress and known gaps | [`docs/internal/PROGRESS.md`](docs/internal/PROGRESS.md), [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md) |

## Working Rules

1. Preserve provider boundaries. If a check belongs to another health provider, link to it instead of duplicating it.
2. Validate from first principles. Scenario-auditor API rules are migration input, not implementation to copy.
3. Keep findings mapped in `.vrooli/maturity.json` before emitting them.
4. Declare fixability honestly. Auto-fix only local, deterministic repairs.
5. Use lifecycle for live probes. Never run target binaries directly.

## Customize Safely

Treat this scenario as a provider contract first and an implementation scaffold second. When adding checks, start from the maturity ladder in `.vrooli/maturity.json`, define the finding and fixability class, then implement the detector or fixer against current Vrooli lifecycle contracts.
