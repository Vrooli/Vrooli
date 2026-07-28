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
| SQLite | embedded storage | yes | journal, facets, forest, harness | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be started through lifecycle commands. |
| ai-gateway | scenario | yes | journal (classify, embed), forest (summarize) | scenario CLI/API | **Writes degrade, never fail.** An entry is appended unclassified and queued; compaction pauses until inference is available. |
| search-hub | scenario | yes (for federated reach) | federation | `.vrooli/search.json` descriptor + `RegisterProvider` | Local `recall` keeps working; only cross-corpus federated query loses the memory provider. |
| vrooli-events | scenario | no | harness | api-core receipt publication (automatic) | Run correlation is absent; writes and recall are unaffected. |
| swarm-manager | scenario | no | harness (work-record migration) | records read for one-time import | Migration deferred; work-record memories still write normally. |
| Coding-agent resources | local resource | no | harness (projection, capture, import) | `resources/<agent>/` — projection path, prompt block, and hook install ride the existing resource install/update machinery | Projection is not written and native writes are not captured for that runtime; memory keeps working for every other harness. |

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
| Claude Code | `~/.claude/projects/<slug>/memory/MEMORY.md` and fact files | file-per-fact plus index | native memory tool | `PreToolUse` and `PostToolUse` | hook, then diff recovery | Live Vrooli index: 19,492 bytes. Installed changelog confirms an over-limit index write returns an explicit error, not silent truncation. |
| Gemini CLI | `~/.gemini/GEMINI.md`, `## Gemini Added Memories` | append-under-section | `save_memory`, `/memory add` | no hook command or hook surface appeared in the installed CLI help or resource | store diff | Live file exists. |
| Codex | global `~/.codex/AGENTS.md`/`AGENTS.override.md`, project `AGENTS.md`; internal memories also exist at `~/.codex/memories/` | instruction file is a single blob; internal memory is runtime-owned | no dedicated durable-memory write command | `PreToolUse` lifecycle hook | store diff | Current official manual: `AGENTS.md` injection defaults to 32,768 bytes and hooks support `PreToolUse`. The limit clips injected context; it does not rewrite the source file. |
| opencode | `~/.config/opencode/AGENTS.md` and project `AGENTS.md` | single blob | none native | no hook surface in installed CLI help; resource states no hook backstop | store diff | Resource config defaults `instructions` to `AGENTS.md`. |
| grok | `~/.grok/memory/MEMORY.md` and `<workspace>/MEMORY.md` | Markdown files plus SQLite FTS5/vec0 index | `/remember`, `/flush`, and natural-language remember | `PreToolUse` and `PostToolUse` | hook, then diff recovery | Installed Grok memory and hooks manuals describe both paths. The directory is absent while experimental memory is disabled. |
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

### Projection size ceiling

The generated projection must stay under whatever size each harness tolerates,
or the runtime edits the projection itself — producing writes this scenario
caused and must then chase.

| Harness | Measured ceiling | At-ceiling behavior | Projection rule |
|---|---|---|---|
| Codex | 32,768 bytes by default for `AGENTS.md` injection; configurable | Codex clips instruction bytes supplied to the first turn. It does not alter the file. | Emit pins first. Keep the projection at or below 32 KiB unless the active Codex configuration proves a higher limit. |
| Claude Code | No numeric limit exposed by the installed runtime; live Vrooli index is 19,492 bytes | Installed changelog: an index write over the runtime read limit returns an explicit error. | Treat projection overflow as an error. Do not truncate or rewrite native memory. |
| Gemini CLI | No documented projection limit found in the installed CLI | Not measured; no native file mutation was performed. | Bound output to the scenario default and report overflow. |
| opencode | No documented projection limit found in the installed CLI | Not measured; no native file mutation was performed. | Bound output to the scenario default and report overflow. |
| grok | No documented projection limit found in the installed local manual | Not measured; experimental memory is disabled on this host. | Bound output to the scenario default and report overflow. |
| antigravity | No documented projection-file contract | Not applicable: the runtime owns its `brain/` files. | Do not project into `brain/`; use only an approved prompt/instruction surface when one is established. |

No live harness memory file was modified for this measurement. The scratch-file
tests originally proposed for Codex and Claude would not prove their runtime
limits: Codex limits context injection rather than writes, and Claude's numeric
read limit is not externally configurable. The documented runtime behavior is
therefore the authoritative evidence used by projection code.

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
| ai-gateway | required | Facet classification, facet-text derivation, embedding, and compaction summarization. | Scenario CLI/API. Model policy stays in the gateway. |
| search-hub | required | Federated retrieval. Memory registers a provider descriptor; the router holds no memory content and no vectors. | `.vrooli/search.json` validated against `.vrooli/schemas/search.schema.json`; boot self-registration. |
| vrooli-events | automatic | Run correlation for memories written inside an agent run. **No integration work** — `api-core/server.go` wraps every handler in `eventbus.AutomaticRuntime`, so receipts carrying run id, workflow execution id, actor kind, and identity token are published for every endpoint already. | Memory stores correlation ids only and never copies run payloads. |
| swarm-manager | migration-only | Work records are absorbed as a memory kind (`VMEM-P1-001`). One-time import of existing records; the `records create` write path is retired from agent-facing docs. | Read-only import. |

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
| search-hub | registration failure at boot | Scenario starts and serves local recall; registration retries. Federated query loses the memory provider until it succeeds. | `VMEM-P0-009` |
| vrooli-events | receipt not published, or null run id | Memory is written with no correlation. Documented as expected: `run_id` is nullable for heartbeat-spawned agents. | `VMEM-P1-002` |
| Harness store unreadable | path missing, permission denied, or SQLite locked | Import skips that harness and reports it; other harnesses import normally. A partial sweep is never recorded as complete. | `VMEM-P0-011` |
| Harness store format changed | adapter extraction yields zero items from a non-empty source | **Import aborts for that adapter rather than importing nothing silently.** A vendor changing storage layout must surface as a failure, not as an empty diff that looks like "nothing new". | `VMEM-P0-011` |
| Hook install unsupported | runtime exposes no usable pre-write surface | Falls back to store diff. Capture latency degrades from real-time to sweep interval; no memory is lost. | `VMEM-P1-008` |
| Projection exceeds harness cap | generated file over the runtime's documented limit | Projection is truncated **at a pin-safe boundary** and the overflow is reported. Pinned standing rules are emitted first so a truncation can never drop one. | `VMEM-P0-010` |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
