# Meta-Orchestrator Summary

## Source
Planning session comparing app-issue-tracker and swarm-manager feature sets to identify gaps, then designing a backlog plan to close those gaps and eventually deprecate app-issue-tracker.

## Decisions Made
- Agent identity will be a cross-scenario standard built into agent-manager, not a swarm-manager-only field
- Identity tokens will be injected as env vars (same pattern as VROOLI_SANDBOX_* vars) and verified via agent-manager API
- Concurrency guardrails are independent of identity work and can proceed in parallel
- Sidebar search is pure UI work, independent of backend changes
- Initiative-level agent research is intentionally deferred until identity + guardrails + sidebar are further along
- App-issue-tracker deprecation is terminal and depends on all three execute items
- One initiative groups all items since the unifying goal is feature parity for deprecation

## Key Findings from Investigation

### Agent-Manager Current State
- Already injects VROOLI_SANDBOX_ID, VROOLI_SANDBOX_MERGED, VROOLI_SANDBOX_SCOPE env vars for sandbox-aware scenarios
- Has CLAUDE_CODE_AGENT_TAG for process discovery via /proc inspection
- CreatedBy strings on tasks/runs are unverified (any process could claim any identity)
- Profile system uses EnsureProfile with stable profile_key for idempotent management
- Env var injection path: RunExecutor.executeAgent() → SandboxEnvVars() + sanitizedBaseEnv() + appendEnvMap()
- Sanitization denylist removes API_PORT, VROOLI_* system vars, CLAUDECODE, CLAUDE_CODE_AGENT_TAG
- Consumer pattern established in packages/cli-core/cliutil/sandbox.go (DetectSandbox, ScenarioInScope, ResolveMergedPath)

### Swarm-Manager Settings System
- 21 settings across 4 tab groups (General, Execution, Workshop, Review)
- Well-structured for extension: proto → handler normalization → frontend service → UI form
- Missing: concurrency limits, rate limiting, circuit breaker, cost awareness
- Hardcoded semaphore of 5 for graph projection parallel ops (not configurable)

### Swarm-Manager Sidebar Current State
- Pure activity feed (Sidebar.tsx) built from captures + backlog via lib/feed.ts
- No search, filtering, sorting, or tabs
- Mobile: 85vw overlay, max 320px - not full viewport
- Feed priority: classifying captures → failed captures → attention items → regular backlog → blocked items
- Graph-level filtering exists in SettingsDrawer but not in sidebar

### App-Issue-Tracker Features NOT Needed in Swarm-Manager
- Semantic/vector search via Qdrant (premature, text search sufficient for now)
- GitHub PR integration (delegated to executing agents and GCT review)
- Component/target catalog (covered by scenarios endpoint)
- Watchers/notification subscriptions (single-operator for now)
- Key-value labels (tags array sufficient)
- Structured error_context fields on issues (description + files sufficient)
- Confidence scores on results (GCT review dimensions cover this)

## Dependency Notes
- research/agent-identity-standard has no deps - can start immediately
- execute/swarm-manager-concurrency-guardrails has no deps - can start immediately  
- execute/swarm-manager-sidebar-search has no deps - can start immediately
- The identity chain is the longest path: research → agent-manager impl → swarm-manager adoption
- chore/app-issue-tracker-deprecation gates on all three execute items

## Unresolved Questions Deferred To Workshop
- JWT vs custom HMAC token format (research/agent-identity-standard)
- gRPC vs REST for verification endpoint (research/agent-identity-standard)
- Token refresh strategy for long-running agents (research/agent-identity-standard)
- Exact cost-per-turn estimate model for budget caps (execute/swarm-manager-concurrency-guardrails)
- Whether initiative-level agent should auto-trigger or be manual-only (research/initiative-level-agent-design)
