# Meta-Orchestrator Summary: swarm-manager-quality-gates

## Source
Planning session that started with designing vrooli-events and notification-hub-greenfield initiatives, then identified structural gaps in the swarm-manager pipeline when the vrooli-events item executed poorly — the agent didn't use scenario templates, omitted the UI, and produced misaligned PRDs. This led to designing three quality improvements.

## What Went Wrong (the motivation)
1. An idea backlog item for vrooli-events was workshopped and finalized, but the executing agent:
   - Did NOT use the scenario template system (`vrooli scenario create`) — built everything from scratch
   - Omitted the UI entirely because it assumed API-only was sufficient
   - Produced a PRD and requirements that didn't fully reflect the planning conversation's decisions
2. A second agent was spawned to fix these issues but also drifted from the original intent
3. The orchestration-summary.md uploaded to the initiative was too static — it captured decisions but couldn't be used for validation or conversation continuity

## Decisions Made

### Item 1: Workshop Finalization Validation Gates
- Programmatic (not agentic) scanning of plan.md before and after finalization
- Pre-finalization: scan for 13 mandatory sections, prompt-manager skill read commands, greenfield declaration, idea-specific template usage → feed gaps as context to finalize agent
- Post-finalization: re-scan → store results → surface in API/UI → block auto-queue if validation fails
- Skip for research items (they produce conclusion.md with different structure)
- Insert points: pre-gate in `api/internal/backlog/research.go:209-233`, post-gate after finalize agent completes in `workshop_save.go`
- The 13 mandatory sections come from the implementation-plan-authoring skill
- For idea items: must check for `scenario-generation` skill reference and `vrooli scenario create` template usage
- All plans must include: Required Reading with `prompt-manager skill read`, greenfield declaration, final cleanup step with `vrooli scenario restart`

### Item 2: Meta-Orchestration Native Sessions
- Dual-mode capture button: quick capture (existing) vs meta-orchestration conversation (new)
- Reuse existing ClarificationPanel/ClarificationMessages components for conversation UI
- Meta-orchestration runs stored as first-class entities with persistent conversations
- Created initiatives/items carry meta_orchestration_id for attribution
- Resume/fork capability so planning conversations can be revisited
- Agent spawning via existing SpawnBacklog() + ContinueRun() with meta-orchestrator skill
- Environment variable VROOLI_META_ORCHESTRATION_ID (or similar) for attribution propagation
- Reuse existing agent identity system (X-Agent-Identity-Token, Provenance) for tracking

### Item 3: Initiative-Level Oversight
- Research item that waits for items 1 and 2 to complete
- Goal: identify remaining gaps after validation gates and meta-orchestration are in place
- Key questions: automatic vs on-demand, write access vs advisory-only, what data agents need

## Key Architecture Context

### Finalization Flow
- Entry: `workshop_save.go:WorkshopSave()` → `spawnWorkshopAsync(item, ResearchModeFinalize)`
- Pre-validation: `research.go:209-233` (round exists, decisions answered, readiness >= 3)
- Finalize skill: `swarm-manager-workshop-finalize` (non-research) or `swarm-manager-workshop-research-finalize`
- Finalize agent writes finalize round with mode="finalize", zero decisions, only info items

### Capture/Chat UI
- Capture input: `ui/src/components/capture/quick-capture-input.tsx`
- Chat panel: `ui/src/components/backlog/clarification-panel.tsx` (FloatingPanel, draggable)
- Messages: `ui/src/components/backlog/clarification-messages.tsx` (markdown + images)
- State: Zustand store, 3s polling, 90s staleness warning, localStorage draft persistence

### Agent Attribution
- Identity: `api/internal/identity/` (X-Agent-Identity-Token → Provenance)
- Activities: `api/internal/agentactivity/` (owner_type, purpose, metadata with skill_id)
- Env vars: VROOLI_ prefix, passed via CreateRunRequest.environment
- Current: VROOLI_SPAWN_SOURCE for backlog attribution

## Unresolved Questions Deferred To Workshop
- Exact storage format for meta-orchestration sessions
- How the capture button mode switch works in the UI
- How conversation continuity works across sessions (ContinueRun vs new agent with history)
- Whether meta-orchestration gets its own page or lives in existing pages
- What the resume/fork UX looks like
- For initiative agents: triggers, capabilities, data access, write permissions
