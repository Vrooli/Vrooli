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

Verified 2026-07-27. Claude Code rows are from direct filesystem inspection;
the rest are from vendor documentation (linked in
[`../../../../docs/`](../../README.md) research notes) and are marked where
unconfirmed.

| Harness | Memory location | Storage shape | Native write tool | Hook API | Resource |
|---|---|---|---|---|---|
| Claude Code | `~/.claude/projects/<slug>/memory/*.md` + `MEMORY.md` index | **file-per-fact** | yes | `PreToolUse` / `PostToolUse` — **already used** by `resources/claude-code/cli/internal/permissions/pretooluse-bash-deny.sh` | `resources/claude-code` |
| Gemini CLI | `~/.gemini/GEMINI.md`, section `## Gemini Added Memories` | **append-under-section** | `save_memory`, `/memory add` | unknown | `resources/gemini` |
| Codex | `~/.codex/AGENTS.md`, `AGENTS.override.md`, project `AGENTS.md` walked from repo root | **single blob** | none dedicated | unknown | `resources/codex` |
| opencode | `~/.config/opencode/AGENTS.md`, project `AGENTS.md` | **single blob** | **none native** — community plugins only | unknown | `resources/opencode` |
| grok | unknown | unknown | unknown | ships `pretooluse-bash-deny.sh`, so a Claude-Code-compatible surface is *inferred, not verified* | `resources/grok` |
| antigravity | unknown | unknown | unknown | permissions layer present, no hook script | `resources/antigravity` |
| Cursor | `state.vscdb` SQLite, `ItemTable` key `aicontext.personalContext`; project `.cursor/rules/` | **SQLite BLOB** | yes | n/a (IDE, not CLI) | **none — not a Vrooli resource** |

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

| Harness | Documented limit | Notes |
|---|---|---|
| Codex | **32 KiB** default for `AGENTS.md`, configurable | The binding constraint. |
| Claude Code | none documented | Current `MEMORY.md` measures 18,245 bytes — already over half the Codex cap. |

**The failure mode at the cap is unverified and may be worse than assumed.** If
a harness *silently truncates* rather than prompting for a trim, a pinned
standing rule can drop out of context with no signal — the failure
[`REPLACEMENT.md`](REPLACEMENT.md) identifies as this scenario's
highest-consequence error. Determining actual at-cap behaviour per harness is
empirical work, not a documentation task, and is tracked by `VMEM-P0-010`.

### Known gaps in this matrix

Recorded rather than guessed:

- **grok and antigravity** memory locations are unknown. Grok's hook script is
  suggestive of a Claude-Code-compatible surface but is not evidence.
- **Gemini, Codex, and opencode hook APIs** are unverified. This determines
  whether each is capture-by-hook or capture-by-diff.
- **`~/.codex/memories/`** was referenced by one source but not confirmed to
  exist alongside `AGENTS.md`.
- **Cursor is named across product docs but has no `resources/cursor`.** Until a
  resource exists, Cursor is out of operational scope; the SQLite row above is
  research, not an integration.
- **opencode has no native memory feature at all.** It reads `AGENTS.md`;
  memory is supplied by community plugins. "Unify opencode's memory" therefore
  means projection-only until a plugin is in play — a materially different
  proposition than for the other runtimes.

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
