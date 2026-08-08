# Meta-Optimization Manager

Measures how ready the Vrooli project is for **lower-powered (local) coding
agents** to do real software engineering — and points to where to improve next.

It is a thin, **read-mostly aggregator**: it reads each owner scenario's
"intended space" (the denominator) and joins it against that owner's live
registry (the numerator) to produce an honest, per-projection readiness
scoreboard. Its `focus next` board carries both the declared **coverage axis**
and the observed **empirical axis** (trial verdict recurrence and attributed
agent-manager friction), so the team gets one synthesized surface for what is
missing and what is costly. It **surfaces numbers and candidates; it never decides** — substrate,
tiering, and nomination calls stay agentic.

## The Model In One Picture

Readiness is measured as coverage across four **projections**, each owned by
another scenario, plus upstream-generator convergence:

| Projection | Question it answers | Denominator owner | Live numerator source |
|---|---|---|---|
| **Answer** | "Can the project answer architectural questions?" | `search-hub` | `search-hub` provider registry |
| **Validate** | "Is the right thing tested and auto-fixed?" | `test-genie` | `test-genie health` + `fleet status` |
| **Guide** | "Is there a skill for each SWE task?" | `prompt-manager` | `prompt-manager graph health` |
| **Act** | "Can an agent programmatically invoke each operation?" | `program-runtime` | `BindingRegistryService.ResolveActCells` |

Answer/Validate/Guide measure **knowledge** supply; Act measures **effect** supply.
Act is wired end-to-end and reports live callable verdicts with denominator
confidence; an unavailable runtime is still surfaced as `UNAVAILABLE`, never as
`0%` and never silently dropped.

Coverage = `now / total` per projection, **computed live and never stored** (only
short-TTL snapshots are cached). Every coverage number is paired with a
**denominator-confidence** (`authoritative` / `partial` / `sketch`) so the board
can never imply false completeness. The canonical model — the projection
definitions, the `basis × sufficiency` attestation contract, the status legend,
and the Guide→Validate→Answer maturity gradient — lives in
[`docs/concepts/COVERAGE-MODEL.md`](docs/concepts/COVERAGE-MODEL.md).

The denominators are **not owned here** — they live with their owners as
`docs/spaces/<projection>-space.md` and are read through a shared
`space --projection <p> --json` contract (with a doc-parse fallback today; see
[Known Limitations](#known-limitations)).

## Capabilities (Domains)

| Domain | CLI verbs | What it gives you |
|---|---|---|
| **coverage** | `coverage status`, `coverage list-cells`, `coverage explain-cell`, `coverage validate-docs` | The readiness scoreboard, per-cell drill-down with provenance, and the base-document integrity gate (no stale/broken refs; Guide rows map to exactly one skill). |
| **focus** | `focus next` | Ranked next-best gaps (impact × importance) across all projections. |
| **gaps** | `gaps list`, `gaps show`, `gaps note` | The honest, durable gaps registry with notes/approaches/context. |
| **convergence** | `convergence status`, `convergence fitness`, `convergence references`, `convergence trend` | Per-template four-lens fitness and gold-star generated-golden health (OT-P1-002). Lens counts are **filesystem proxies** (LOC, comment-grep) — structural signals, not semantic analysis. |
| **trials** | `trials run`, `trials list`, `trials history`, `trials show`, `trials coverage` | Empirically exercises a local model on fixture SWE tasks via agent-manager's sandboxed runner, scores the produced diff against a deterministic oracle, and records success-rate / tokens / wall-time as a trend (the real proof of readiness). |

All `--json` output is typed proto-JSON. Every cross-scenario read **degrades
gracefully** — an owner being down marks that projection unavailable with an
honest reason, it never false-fails.

## Architecture

Domain-first Go API behind Connect-RPC, a typed Go CLI mirroring each service, and
a React/Vite operator console — all coordinated through generated proto contracts
in `packages/proto/schemas/meta-optimization-manager`. Each domain follows the
canonical layering:

```
handler → Service → { SpaceReader, NumeratorJoiner, SnapshotRepository }
             ↑              ↑ (faked in tests)        ↑
          (proto edge)   live owner reads        short-TTL cache
```

State is minimal and local (SQLite via `api-core/storage`): the qualitative gaps
registry, the trials history time-series, a cached convergence index, and
short-TTL coverage snapshots. See
[`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) and
[`docs/internal/SEAMS.md`](docs/internal/SEAMS.md).

## Running

```bash
make setup   # build API + UI, install deps + scenario CLI (wraps `vrooli scenario setup`)
make start   # start API + UI (wraps `vrooli scenario start`)
make test    # run the suite (wraps `vrooli scenario test`)
make logs    # tail logs
make stop    # stop
```

See [`docs/QUICKSTART.md`](docs/QUICKSTART.md) for the full clone-to-running flow.

## Known Limitations

These are honest, current gaps — tracked in
[`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md):

- **Trials are not yet proven end-to-end on a live local model.** The runner,
  evaluator, and fixtures are real and tested; a single operator live-e2e pass
  (opencode + a local model) is needed to confirm the diff-apply path.

## Documentation Map

| Need | Start Here |
|---|---|
| The coverage model + attestation contract | [`docs/concepts/COVERAGE-MODEL.md`](docs/concepts/COVERAGE-MODEL.md) |
| Architecture & data flow | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Product domains | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Run the scenario | [`docs/QUICKSTART.md`](docs/QUICKSTART.md) |
| UI design language | [`DESIGN.md`](DESIGN.md) |
| CLI commands | [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |
| API endpoints | [`docs/reference/api-endpoints.md`](docs/reference/api-endpoints.md) |
| Env vars, ports, config | [`docs/reference/configuration.md`](docs/reference/configuration.md) |
| Seams & fakes | [`docs/internal/SEAMS.md`](docs/internal/SEAMS.md) |
| Durable decisions | [`docs/internal/DECISIONS.md`](docs/internal/DECISIONS.md) |
| Known issues & deferred work | [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md) |

## Working Rules

1. **Surfaces, does not decide.** Add numbers, candidates, and confidence — never
   bake in substrate/tiering/nomination judgment.
2. **Honest by construction.** Every coverage number ships paired with its
   denominator-confidence; reads degrade gracefully and never false-fail.
3. **Do not own the denominators.** The space docs live with their owners; read
   them through the shared contract, never re-implement an owner's measurement.
4. **Update `PRD.md` and `requirements/`** before feature work — operational
   targets drive code + tests.
5. **Read [`DESIGN.md`](DESIGN.md) before UI work**; preserve the i18n and
   accessibility seams.
6. **Append to [`docs/internal/PROGRESS.md`](docs/internal/PROGRESS.md)** when you
   land work, and record real debt in
   [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md).
7. **Keep boundaries** — only edit within this scenario, except the shared
   `*-space.md` denominators this scenario co-owns by contract.
