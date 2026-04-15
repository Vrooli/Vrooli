# Director Dashboard Gap Workflow — Orchestration Context

## Source
Brainstorming session on 2026-04-07. This initiative wires the Director Swarm to use command-center for both outcome signals and gap monitoring.

## Team Decision
Director Swarm owns gap detection AND outcome monitoring. NOT Meta Optimization (that's about optimizing prompt-manager entities). NOT Feature team (they get DEPLOYED to fill gaps, but signal detection is director-level).

Reasoning:
- Director's charter: "surface blockers, dependencies, under-specified work" — gaps are exactly that
- Director already monitors stats/overview every heartbeat — this extends naturally
- Fits deployment model: Director → proposes backlog → human approves → Feature team executes

## Two Complementary Lenses
- Swarm Manager = the WORK — what's being built, blocked, queued, velocity, execution details. "Inside the factory."
- Command Center = the OUTCOMES — subscribers growing? Revenue moving? Scenarios healthy? System better? "Looking out the window."

The director needs both. Swarm Manager says "we completed 12 items." Command Center says "and subscriber count is flat, so maybe shift toward marketing."

Command-center COMPLEMENTS swarm-manager, does NOT replace it. Swarm-manager gives granular work detail. Command-center gives outcome context and cross-system visibility.

## Implementation Plan
1. Add command-center dashboard endpoints to Director heartbeat (alongside existing swarm-manager overview/stats calls)
2. Intelligence-officer pulls outcome signals for Now/Near/Far prioritization
3. Add /api/v1/gaps check to heartbeat
4. Intelligence-officer briefing includes "Dashboard Gaps" section
5. When gaps found: evaluate if filling would help active teams → propose backlog if yes
6. Gap priority: Panorama/Mission Control gaps > secondary page gaps; weight by active team needs and effort

## Files to Modify
- Director Swarm TEAM.md — add command-center to heartbeat checklist
- Intelligence officer instructions — add outcome + gap reporting to briefing
- Document the gap → backlog → execution workflow
