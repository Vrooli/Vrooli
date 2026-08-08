# Integrations — Vrooli Memory

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes (harness state only) | harness projections, import runs, retry state | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. The journal corpus is now authoritative in source-ledger; this database is retained for harness-local state during cutover and rollback. |
| source-ledger | scenario | yes for live wake/recall after phase 16 | journal, recall, forest, facets | generated Connect clients with startup-resolved base URL | Interactive recall fails with a typed unavailable error when the ledger is down; the last successful managed projection remains on disk and continues to provide ambient memory. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be started through lifecycle commands. |
| ai-gateway | scenario | yes (through source-ledger after phase 16) | journal (classify, embed), forest (summarize) | scenario CLI/API | **Writes degrade, never fail.** An entry is appended unclassified and queued; compaction pauses until inference is available. |
| search-hub | scenario | transitive through source-ledger | source-ledger federation | source-ledger `.vrooli/search.json` + bounded `RegisterProvider` | Local `recall` and wake keep working; only cross-corpus federation loses the source-ledger providers. |
| vrooli-events | scenario | no | harness | api-core receipt publication (automatic) | Run correlation is absent; writes and recall are unaffected. |
| swarm-manager | scenario | no | harness (work-record import) | local records adapter, content-addressed import keys | Import is replay-safe and optional; imported records become `kind=work-record` memories and normal writes remain available. |
| Coding-agent resources | local resource | no | harness (projection, capture, import) | `resources/<agent>/` owns the runtime formats; the Go-native `vrooli-memory hooks` reconciler writes only the documented Claude/Grok hook config and never edits agent binaries | Projection/import remain available for every adapter; hook-capable runtimes capture writes in real time and others use the scheduled store sweep. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| SQLite (embedded) | active | Journal, facet, and tree tables are single-writer, local, and modest in size. No shared resource is warranted. | If memory becomes multi-host or the vector index outgrows embedded storage. |
| ollama | indirect | Reached **through ai-gateway**, never directly. Embedding and summarization are inference calls, and ai-gateway owns model policy, routing, and capacity. | Never call ollama directly from this scenario — that would duplicate policy the gateway owns. |
| qdrant | not-applicable | Vector storage goes through the shared `aisearch-go` package used by every existing search provider, keeping this scenario's index shape identical to the rest of the fleet. | If corpus size makes an embedded index untenable. |
| claude-code, codex, gemini, grok, opencode, antigravity | active (integration target) | The six coding-agent harnesses this scenario unifies. Each already owns install/update and a unified allow/deny permissions layer under `resources/<agent>/cli/internal/permissions/`, which is the correct home for projection writes, prompt-block install, and hook install. **This scenario never edits agent binaries** — it extends the resource that already configures them. | If a new coding-agent resource is added, or Cursor gains one. |

## Harness Capability Matrix

The harness domain integrates with coding-agent runtimes, each of which is a
Vrooli resource under `resources/<agent>/`. **Reads are uniform** — every
harness gets the generated projection written to the file it already loads.
**Writes are not**, because each runtime stores memory differently. This matrix
is the contract the import adapters and the capture strategy are built from.

Measured 2026-07-27 by inspecting the installed runtime, its local manual, and
the resource-owned integration source. A runtime is never treated as supported
from a sibling runtime's behavior. “Store diff” is the safe capture floor when
the runtime has no documented hook that identifies a native-memory write.

| Harness | Memory location | Storage shape | Native write tool | Hook API | Capture channel | Evidence |
|---|---|---|---|---|---|---|
| Claude Code | `~/.claude/projects/<slug>/memory/MEMORY.md` and fact files | file-per-fact plus index | native memory tool | `PreToolUse` and `PostToolUse` | installed `vrooli-memory` PreToolUse hook, then diff recovery | `vrooli-memory hooks --action install --runtime claude-code` reconciles one managed entry in `~/.claude/settings.json`; the Go-native `vrooli-memory hook --runtime claude-code` filters structured payloads and swallows capture failures. |
| Gemini CLI | `~/.gemini/GEMINI.md`, `## Gemini Added Memories` | append-under-section | `save_memory`, `/memory add` | no hook command or hook surface appeared in the installed CLI help or resource | store diff | Live file exists. |
| Codex | global `~/.codex/AGENTS.md`/`AGENTS.override.md`, project `AGENTS.md`; internal memories also exist at `~/.codex/memories/` | instruction file is a single blob; internal memory is runtime-owned | no dedicated durable-memory write command | none: current installed CLI help exposes no hook surface and the resource adapter's `RenderHook` is explicitly a no-op | store diff | Current installed runtime exposes `~/.codex/memories/`; `AGENTS.md` injection defaults to 32,768 bytes and clips supplied context without rewriting the source file. |
| opencode | `~/.config/opencode/AGENTS.md` and project `AGENTS.md` | single blob | none native | no hook surface in installed CLI help; resource states no hook backstop | store diff | Resource config defaults `instructions` to `AGENTS.md`. |
| grok | `~/.grok/memory/MEMORY.md` and `<workspace>/MEMORY.md` | Markdown files plus SQLite FTS5/vec0 index | `/remember`, `/flush`, and natural-language remember | `PreToolUse` and `PostToolUse` | installed `vrooli-memory` PreToolUse hook, then diff recovery | `vrooli-memory hooks --action install --runtime grok` writes the resource-native hook document under `~/.grok/hooks/`; failures remain non-blocking. |
| antigravity | `~/.gemini/antigravity/brain/` | per-task Markdown and metadata files; conversation protobufs | none exposed | no user-writable hook contract | store diff | Live `brain/` contains task and plan Markdown; resource documentation records compiled-in hooks only. |
| Cursor | `state.vscdb` SQLite, `ItemTable` key `aicontext.personalContext`; project `.cursor/rules/` | SQLite BLOB | yes | n/a (IDE, not CLI) | out of scope | No Vrooli resource exists, so this plan does not support it. |

### What the shapes imply

Storage shape, not vendor, determines the capture strategy:

| Shape | Import difficulty | Natural key | Capture strategy |
|---|---|---|---|
| file-per-fact | trivial | file path | Store diff is sufficient; hooks optional. |
| append-under-section | easy | position within the known section | Store diff scoped to the section header. |
| single blob | **hard** | none — whole-file rewrite carries no item identity | Hooks matter; diffing a rewritten blob into discrete memories is unreliable. |
| SQLite BLOB | **hard** | one row, opaque value | Parse and segment inside the BLOB, or hook. |

Capture channels, in priority order:

1. **Hook at write time** — precise and real-time, where the runtime exposes one.
2. **Store diff** — the universal floor. Works everywhere, idempotent by content
   hash (see [`DATA.md`](DATA.md) → Import / Export).
3. **Transcript mining** — most complete but most coupled to internal formats.
   Claude Code memory writes are tool calls and therefore appear in
   `~/.claude/projects/<slug>/<uuid>.jsonl`. Reserve for harnesses the first two
   cannot cover.

### Projection ceilings

The generated projection must stay under whatever size each harness tolerates,
or the runtime edits the projection itself — producing writes this scenario
caused and must then chase.

The projection service enforces two ceilings on the complete consumer file for
every configured runtime: 32,768 bytes and 200 rendered text lines. The line
ceiling is deliberately shared because Claude Code previously truncated a
200-line projection; the other runtimes use the same conservative contract
until a runtime-specific limit is measured. The live dry-run sweep on
2026-08-06 measured the rendered files at 9,532–20,241 bytes, all below the
byte ceiling. Every projection subtracts the existing curated region — the
bytes and rendered lines outside the managed wake markers — before asking the
ledger for a wake budget. The final splice is checked again, including marker
lines, so a multiline excerpt cannot pass an approximate entry-cost check.

If the curated region already exceeds a ceiling, it is preserved byte-for-byte
and the generated wake block is empty; the result is reported as overflow. A
non-pinned entry that would exceed the remaining capacity is omitted and the
projection reports overflow honestly. Pinned entries are emitted first and a
pinned overflow fails before any file write. Native runtime behavior beyond
these scenario guards is intentionally not exercised by writing oversized
production files.

| Harness | Byte ceiling | Rendered-line ceiling | At-ceiling behavior | Projection rule |
|---|---|---|---|---|
| Codex | 32,768 bytes by default for `AGENTS.md` injection; configurable | 200 | Codex clips instruction bytes supplied to the first turn. It does not alter the file. | Emit pins first; keep the complete file at or below both scenario ceilings. |
| Claude Code | 32,768 bytes (scenario guard); live dry-run 20,241 bytes | 200 | Scenario returns overflow before writing; native over-cap behavior was not invoked. | Emit pins first and fail closed before a pinned item would be truncated. |
| Gemini CLI | 32,768 bytes (scenario guard); live dry-run 13,686 bytes | 200 | Scenario returns overflow before writing; native over-cap behavior was not invoked. | Emit pins first and fail closed before a pinned item would be truncated. |
| opencode | 32,768 bytes (scenario guard); live dry-run 13,686 bytes | 200 | Scenario returns overflow before writing; native over-cap behavior was not invoked. | Emit pins first and fail closed before a pinned item would be truncated. |
| grok | 32,768 bytes (scenario guard); live dry-run 13,686 bytes | 200 | Scenario returns overflow before writing; native over-cap behavior was not invoked. | Emit pins first and fail closed before a pinned item would be truncated. |
| antigravity | 32,768 bytes (scenario guard); live dry-run 9,532 bytes | 200 | Scenario returns overflow before writing; native over-cap behavior was not invoked. | Emit pins first and fail closed before a pinned item would be truncated. |

No live harness memory file was modified for this measurement. Codex's native
injection clipping remains documented separately; the other installed runtimes
expose no stable, user-configurable numeric limit. The scenario-owned guard is
therefore the authoritative safety boundary used by projection code.

### Known gaps in this matrix

Recorded rather than guessed:

- **Cursor is named across product docs but has no `resources/cursor`.** Until a
  resource exists, Cursor is out of operational scope; the SQLite row above is
  research, not an integration.
- **OpenCode has no native memory feature.** It remains projection-only until a
  plugin supplies a governed native-memory contract.

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| source-ledger | required | Sole authority for journal, facets, forest, recall, rules, and scopes. `vrooli-memory` uses generated Go Connect clients and retains only harness-specific import/projection/maintenance state. | Source-ledger Connect API; startup discovery resolves the endpoint once. Recall fails with typed `unavailable` when the authority is down, while the last successful projection remains readable. |
| ai-gateway | transitive through source-ledger | Facet classification, facet-text derivation, embedding, and compaction summarization remain source-ledger responsibilities. | Source-ledger owns the gateway dependency and model policy. |
| search-hub | transitive through source-ledger | Federated retrieval and per-scope provider registration remain source-ledger responsibilities. | Source-ledger owns the Search Hub descriptor and registration; this consumer has no provider descriptor. |
| vrooli-events | automatic | Run correlation for memories written inside an agent run. **No integration work** — `api-core/server.go` wraps every handler in `eventbus.AutomaticRuntime`, so receipts carrying run id, workflow execution id, actor kind, and identity token are published for every endpoint already. | Memory stores correlation ids only and never copies run payloads. |
| swarm-manager | import adapter | Work records are absorbed as a memory kind (`VMEM-P1-001`). The adapter is idempotent and records source provenance; the `records create` write path is retired from agent-facing docs. | Read-only source sweep; failures are reported without claiming completion. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | Every inference call routes through ai-gateway, and all storage is local. This scenario reaches no external network service. | Add only if a hosted embedding or summarization provider is ever introduced — which would go through ai-gateway, not here. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| ai-gateway (classify) | call error or timeout | Entry is appended with an unclassified facet and enqueued for retry. **A write is never lost to an inference failure.** | `VMEM-P0-002` |
| ai-gateway (summarize) | call error or timeout | Compaction pass aborts cleanly; the frontier stays over target until inference returns. No partial summary is written. | `VMEM-P0-007` |
| search-hub | source-ledger registration failure at boot or scope creation | Memory starts and serves local recall; source-ledger retries registration. Federated query loses only the affected source-ledger provider until registration succeeds. | `VMEM-P0-009` |
| vrooli-events | receipt not published, or null run id | Memory is written with no correlation. Documented as expected: `run_id` is nullable for heartbeat-spawned agents. | `VMEM-P1-002` |
| Harness store unreadable | path missing, permission denied, or SQLite locked | Import skips that harness and reports it; other harnesses import normally. A partial sweep is never recorded as complete. | `VMEM-P0-011` |
| Harness store format changed | adapter extraction yields zero items from a non-empty source | A dry-run returns a successful zero-source observation for an absent or empty declared store; malformed extraction still fails, and a durable import records failure rather than claiming completion. | `VMEM-P0-011` |
| Hook install unsupported | runtime exposes no usable pre-write surface | Falls back to store diff. Capture latency degrades from real-time to sweep interval; no memory is lost. | `VMEM-P1-008` |
| Projection exceeds harness cap | generated file over the runtime's documented limit | Projection fails closed and reports overflow; no native file is rewritten or truncated. | `VMEM-P0-010` |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
