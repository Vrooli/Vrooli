# Meta-Orchestrator Summary: Project Wiki

## Source
Planning session workshopping how to build an AI-maintained project-level wiki for Vrooli. Inspired by Andrej Karpathy's LLM Wiki concept (https://gist.githubusercontent.com/karpathy/442a6bf555914893e9891c11519de94f/raw/ac46de1ad27f92b28ac95459c782c07f6b8c964a/llm-wiki.md). The companion initiative (team-organizational-maturity) covers the team schema upgrade that enables contributions declarations.

## Core Vision
Agents handle the maintenance burden that makes humans abandon knowledge bases. The wiki is the institutional memory layer between raw sources (docs/, scenario docs, external inputs) and semantic search (Knowledge Observatory). Knowledge compounds over time rather than being rediscovered with each query.

## Decisions Made

### Architecture
- New standalone scenario (project-wiki), not an extension of Knowledge Observatory or Prompt Manager.
- KO stays as search/indexing backend (observer). Wiki is the content authoring/maintenance layer.
- Prompt Manager team knowledge (JSONL) stays as operational working memory. Wiki is institutional long-term memory. Different lifecycles, different audiences.
- Wiki pages are plain markdown on disk — readable, diffable, version-controllable. Not in a database.
- Three core operations from Karpathy's model: Ingest, Query, Lint.
- index.md (catalog) and log.md (audit trail) for navigation, per Karpathy's design.

### Content Structure
Organized by question, not by team:
- identity/ — What Vrooli IS (vision, pitch, glossary, brand voice)
- strategy/ — Where we're GOING (roadmap, priorities, initiatives/, decisions)
- capabilities/ — What we CAN DO (scenarios/, resources/, deployment)
- operations/ — How we WORK (teams, architecture, patterns, lessons learned)
- revenue/ — How we EARN (model, opportunities/, metrics)
- marketing/ — How we COMMUNICATE (campaigns/, audience, content calendar)
- quality/ — How HEALTHY we are (platform health, scenario quality/, tech debt)

Multiple teams feed the same sections. Scenario wiki pages capture business context (revenue potential, deployment status, who uses it), not code docs (KO handles those). Initiative pages are living documents updated as swarm-manager items complete.

### Information Sources — Two Parallel Streams (Neither Is Primary)
- **Swarm Manager**: Authoritative for engineering/capability changes. Completed backlog items, research conclusions, initiative progress. Dominant signal stream NOW because most work goes through backlog items.
- **Team heartbeat outputs**: Authoritative for operational actions outside Swarm Manager. Marketing campaigns, deployment decisions, strategic shifts. From handoff-history.jsonl, decisions.jsonl, knowledge.jsonl. INCREASINGLY IMPORTANT as system matures and teams do more work autonomously through scenario CLIs.
- **Repo/deployment state**: Ground truth for what's actually deployed and current.
- Wiki agent routes signals to wiki sections based on each team's declared contributions in team.json v2.

### Maintenance Model
- Dedicated prompt-manager team (wiki-maintenance), NOT distributed across all teams.
- Single-process, autonomous decision mode (wiki updates don't need approval; reviewable via git diff).
- One agent (Wiki Curator) runs full cycle: scan signals → determine updates → apply → lint → publish to KO → handoff.
- Teams don't need to know about the wiki. They declare contributions in team.json; wiki agent reads those declarations and scans the right signal streams.

### Integration Points
- Wiki → Knowledge Observatory: publishes pages for semantic search/embedding via KO's existing ingest API (POST /api/v1/knowledge/documents/ingest, nomic-embed-text → Qdrant 768-dim)
- Wiki ← Swarm Manager: reads completed items and initiative status via swarm-manager CLI
- Wiki ← Prompt Manager: reads team handoffs/decisions/knowledge via team shared files
- Wiki ← Team.json v2: reads contributions declarations to know what signals to scan for

### Existing System Context
- Knowledge Observatory: Fully built semantic search + doc validation. Has embedding pipeline (Ollama → Qdrant), 15 doc-type schema enforcement, deep search via agents, project-level search. All 44 Qdrant collections currently empty. It's an observer/indexer, not a content author.
- Prompt Manager team knowledge: Simple JSONL append-only stores scoped to individual teams. No cross-team sharing. Max 100 entries with 180-day pruning. This is operational working memory.
- Existing docs/: Well-structured hub-and-spoke (README.md, VISION.md, docs/strategy/roadmap.md, etc.). No living state dashboard, no unified glossary, no scenario inventory with business context.

## Dependency Notes
- This research depends on research/team-schema-v2-design (needs to know the contributions field structure)
- execute/project-wiki-maintenance-team depends on BOTH core-runtime AND team-schema-v2-migration
- ko-integration and initial-content-seed can proceed in parallel after core runtime
- Initial content seed is significant: 22 initiatives need individual pages, all scenarios need business-context pages, existing docs need synthesis (not mechanical copy)

## Unresolved Questions For This Research
- Exact heartbeat cadence for wiki-maintenance team (daily vs. twice-daily)
- Whether wiki ingest should be fully automated or produce proposed changes for review
- How to handle contradictions between wiki content and new signals (overwrite? flag for human?)
- Whether the wiki should have a UI (React dashboard) or if CLI + KO search UI is sufficient for v1
- Specific filtering heuristics for what's wiki-worthy beyond general guidelines
- Exact KO namespace convention and metadata schema for wiki records
- Whether log.md should have a max size / rotation policy
