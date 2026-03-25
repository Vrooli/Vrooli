# Implementation Plan: Effort Test

## Required Reading
```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

## Purpose
<!-- TBD — The title "Effort Test" suggests either (a) a scenario for effort estimation/tracking, (b) a test harness for measuring agent effort, or (c) a placeholder item. Blocked on workshop round-001 decision d1. -->

## Problem Statement
<!-- TBD — No description provided in spec.json. The core problem or opportunity this item addresses is the primary unknown. All downstream plan sections depend on this being answered. -->

## Scope

### In Scope
<!-- TBD — Depends on round-001 d1 (what is this?), d3 (components), and round-002 d1 (target scenario). -->

### Out of Scope
<!-- TBD -->

## Current Technical Context

### Relevant Ecosystem Integration Points
- **Agent-manager investigation scoring** (`internal/orchestration/investigation.go`): Already rates proposed fixes on a cost (1-5) scale representing effort/risk. If this item is about effort estimation, it may extend or complement this capability.
- **Swarm-manager backlog metadata**: Tracks items with priority, timestamps, and status. Natural place to attach effort estimates for scheduling and prioritization.
- **Vrooli resource stack**: PostgreSQL (storage), Ollama (local LLM inference), Redis (caching) are available as shared resources.

## Target End State
<!-- TBD -->

## Implementation Strategy
<!-- TBD — Blocked on scope and approach decisions from workshop rounds 1-3. -->

## Contract Decisions
<!-- TBD -->

## Testing Plan
<!-- TBD -->

## Rollout / Validation Checklist
<!-- TBD -->

## Risks + Mitigations
<!-- TBD -->

## Non-goals / Prohibited Patterns
<!-- TBD -->

## Definition of Done
<!-- TBD -->
