# Agent Inbox Unified Retrieval — Meta-Orchestrator Summary

## Source

Brainstorming session (2026-04-22). Current agent-inbox stuffs *every enabled tool* into every turn's request context. This is token-wasteful, doesn't scale as scenarios add tools, and creates friction for non-technical users (they'd have to manually enable the right tools before each conversation). The user wants a `just talk and the system finds the right surface` experience — which is also the forcing function for the phone-agent UX (voice-only, no tool-picker UI).

This initiative is the consumer-side UX payoff for the three authoring initiatives: `tool-authoring-standard`, `cli-conversational-surface`, `widget-standard`.

## Shared Decisions (apply across all four sibling initiatives)

1. **Proto-first** for all three surface types.
2. **Manifest-free.** Index is built from runtime endpoints + static source extraction, not scenario-declared manifests.
3. **Fewer packages.**
4. **Auditor comparison** enforces consistency across authoring side.
5. **Static embedding extraction** for widgets; runtime discovery for tools and CLI commands; all three land in the same index with a surface-type tag.

## Scope of This Initiative

Replace the current `all enabled tools in context` model with a semantic index and a dual-track retrieval UX.

### Dual-track retrieval

**Track 1 — agent-led (always-on meta-tools)**
Every chat in auto-mode always has three small meta-tools in context:

- `search_tools(query)` — semantic search over proto tools
- `search_commands(query)` — semantic search over CLI commands (via `introspect` data)
- `search_widgets(query)` — semantic search over widget descriptors

These are cheap (3 tool defs total) compared to dumping 30+ scenario tools. The agent invokes them on demand, exactly as a human would use a search box.

**Track 2 — passive (user-turn embedding)**
Each user message is embedded and matched against the unified index. Results surface as UX affordances, not as model context:

- **Tools / CLI commands**: surface as one-click-enable suggestions (a row above or alongside the input). User approves → that tool becomes available for this turn.
- **Widgets**: auto-render inline when confidence is high; show a dismissible `show this?` card when borderline; silent when low.

Passive track runs on the *user* message (embedding-only, no model call) to bound latency. Agent-led runs on the *agent's* choice.

### Auto-mode default

Auto-mode becomes the new default for chats. The current `enable these N tools globally` UI stays as an escape hatch for users who want manual control.

### Unified index

- Qdrant (already in the stack) stores embeddings + metadata for all three surface types.
- Metadata includes: surface type (`tool` | `command` | `widget`), scenario, id, description, category, tags.
- Rebuild triggers: scenario startup (tools, commands), scenario UI build (widgets via static extraction).

### Latency + efficiency budget

- Added first-token latency: **<200ms** over baseline (budget for embedding + Qdrant query).
- Token saved per turn: measurable reduction vs. current `stuff all enabled tools` baseline; target is negotiated during research.
- Recall@K on a routing eval set: target defined during research.

### What we're NOT building

- The surfaces themselves — authored by the three sibling initiatives.
- A new vector store — Qdrant is already adopted.
- Cross-chat learning / feedback loops (future work; defer).
- Non-semantic ranking heuristics beyond what falls out of vector similarity + category filtering.

## Anticipated Items

- `research/unified-retrieval-index-design` — surface-type schema in Qdrant, embedding model choice, re-index triggers, dual-track UX spec, eval set design, latency + recall budgets.
- `execute/agent-inbox-qdrant-retrieval-index` — build the index and the re-indexers.
- `execute/agent-inbox-agent-led-search-tools` — the three `search_*` meta-tools.
- `execute/agent-inbox-passive-suggestion-surface` — UI affordances for tool/command suggestions and widget auto-render.
- `execute/agent-inbox-auto-mode-default` — flip auto-mode to default; document the manual escape hatch.
- `execute/agent-inbox-retrieval-eval-harness` — held-out routing eval set + CI check so recall doesn't regress.

## Cross-Initiative Dependencies

- **Depends on** `tool-authoring-standard`, `cli-conversational-surface`, `widget-standard` for the authored surfaces to index. Research phase can begin in parallel with the sibling research phases; execute phase needs at least one sibling producing real descriptors.
- **Consumed by** `phone-agent` — the phone-agent conversational UX needs auto-mode retrieval and the widget auto-render story to work voice-first.

## Design Intent

The core claim this initiative is trying to prove: **a non-technical user should be able to open agent-inbox, say what they want in plain language, and have the system find the right surface (tool, CLI command, or widget) without ever learning which scenarios exist**. Everything in the retrieval design is in service of that goal — the dual-track structure, the surface-type-agnostic index, the static widget extraction, the 200ms latency budget. Phone-agent is the strictest test case: no UI, no typing, no option to `scroll through enabled tools`. If the retrieval layer is good enough for voice, it's good enough for everything else.

## Open Questions Deferred to Workshop / Research

- **Embedding model**: local (Ollama) or hosted (OpenAI/Voyage/etc.)? Local aligns with the `shared local resources` project vision but may cost recall; research benchmarks both.
- **Re-index triggers**: scenario start, scenario-auditor run, file-watcher on UI source? Minimal-viable is scenario-start plus manual refresh; research refines.
- **Confidence thresholds** for auto-rendering widgets vs. suggesting them vs. staying silent. Needs real-usage calibration.
- **Multiple-widget-per-turn**: if two widgets both match a turn, render both? Pick the higher-confidence one? Research decides with a usability tradeoff framing.
- **Cross-chat personalization**: does the index learn per-user which surfaces to rank higher? Out of scope for v1; flag as future.
- **Agent-led search result quality**: when the agent calls `search_tools("audio settings")`, does it get back descriptors good enough to decide which tool to invoke, or does it need richer result payloads? Research item prototypes.
