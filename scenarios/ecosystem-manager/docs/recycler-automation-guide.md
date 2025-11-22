# Ecosystem Manager Recycler Automation Guide

## Overview

This document captures the design and implementation plan for adding automatic task recycling, note updates, and terminal columns to the ecosystem-manager scenario. It includes architecture notes, UX requirements, API contracts, and a delivery checklist to keep execution on track.

## Goals

- Automate post-run processing so completed and failed tasks refresh their notes and re-enter the queue without manual work.
- Introduce guardrails that stop recycled tasks once they are repeatedly reported as fully complete or repeatedly fail.
- Provide clear UI indicators and configuration controls for the new automation without breaking current workflows.

## Key Concepts

- **Recycler Daemon**: Background loop that periodically processes one eligible task from `completed/` or `failed/` queues, updates notes via LLM analysis, and either requeues or finalizes the task.
- **Terminal Columns**: New board lanes (`completed-finalized/`, `failed-blocked/`) for tasks that should stop recycling. Cards show distinctive borders/icons.
- **Streak Counters**: Two YAML fields per task track consecutive success claims and consecutive failures. Thresholds (defaults 3 and 5) control when tasks move into terminal columns.
- **LLM Summarizer**: Configurable model (Ollama or Resource OpenRouter) that converts task output into the canonical note styles and returns a completion classification flag.

## Architecture Plan

### Data Model Updates

Add the following fields to each task YAML:

- `consecutive_completion_claims` (int, default 0)
- `consecutive_failures` (int, default 0)
- `processor_auto_requeue` (bool, default true)
- Optional `classification` metadata inside `results` to log the last recycler verdict (for debugging).

Provide a migration utility to seed defaults on existing tasks.

### Settings & Configuration

Create a new Recycler tab in the settings modal and extend the API schema:

```json
{
  "recycler": {
    "enabled_for": "off | resources | scenarios | both",
    "interval_seconds": 60,
    "model_provider": "ollama | openrouter",
    "model_name": "llama3.1:8b",
    "completion_threshold": 3,
    "failure_threshold": 5
  }
}
```

- Maintain backward compatibility by giving sensible defaults in server startup.
- Store settings in existing settings package; expose getters for recycler config.
- UI changes: new tab, interval slider (30s–10m), dropdowns for mode/provider/model, numeric inputs for thresholds.

### Recycler Loop Logic

1. On interval tick, load settings and exit early if automation disabled.
2. Enumerate `completed/` then `failed/` directories to find the first eligible task matching enabled types.
3. Verify `processor_auto_requeue` is true and task not already in terminal status.
4. Summarize output:
   - If `results.output` present, call LLM with prompt returning plain text with classification and multi-section note (see Prompt Contract below).
   - On failure or missing output, fallback to conservative note (`Not sure current status`) and classification `uncertain`.
5. Update note and streak counters:
   - `full_complete`: increment completion streak, reset failure streak.
   - `significant_progress` / `some_progress`: reset completion streak, reset failure streak.
   - `uncertain`: reset completion streak, reset failure streak only if there was no error.
   - Failed tasks without output: increment failure streak, reset completion streak.
6. Compare streaks to thresholds:
   - If completion streak >= threshold → move file to `completed-finalized/`, set `processor_auto_requeue=false`, broadcast UI update with green styling data.
   - Else if failure streak >= threshold → move file to `failed-blocked/`, set `processor_auto_requeue=false`, broadcast with red styling.
   - Otherwise → move file back to `pending/` and persist updates.
7. Only process one task per interval; exit loop until next tick.

### Execution Processor Adjustments

- After each run, ensure task counters are updated but do not auto-move back to pending.
- Add helper to persist `results.output` and timestamps if not already stored.
- Ensure processor does not touch tasks marked with `processor_auto_requeue=false` or located in terminal directories.

### UI Enhancements

- Add new columns to board layout: `Completed – Finalized`, `Failed – Blocked`.
- Card styling: green border + check icon for finalized, red border + error icon for blocked.
- Task detail drawer displays streak counters and thresholds.
- Settings modal gains Recycler tab as described.

## Prompt Contract

LLM prompt requests a plain text response with this exact format:

```
classification: <full_complete|significant_progress|some_progress|uncertain>
note:
<multi-line note content with 5 required sections>
```

The note must contain these sections in order:
- **What was accomplished:** specific deliverables, files created/edited, operational targets completed
- **Current status:** detailed status including what works, validation results
- **Remaining issues or limitations:** specific blockers, failing tests, regressions, open TODOs
- **Suggested next actions:** specific commands to run, priorities for next iteration
- **Validation evidence:** specific commands run, test results, health check outputs

Character limit: 4000 chars (prioritizing validation commands and specific blockers).
Input is truncated to 12000 chars before being sent to the LLM.

## Failure Handling

- If LLM call fails, fall back to conservative note, do not increment completion streak.
- If YAML save/move fails, log via `systemlog` and leave task in current folder.
- Consider circuit breaker if repeated LLM errors occur (future enhancement).

## Delivery Checklist

1. **Design Validation**
   - [ ] Confirm settings schema changes with stakeholders.
   - [ ] Finalize LLM prompt and response validation logic.

2. **Data Layer**
   - [ ] Update `TaskItem` struct with new fields.
   - [ ] Extend storage read/write to handle defaults.
   - [ ] Implement migration script to seed existing tasks.

3. **Settings API**
   - [ ] Extend `Settings` struct & handlers with recycler config.
   - [ ] Add validation for interval and thresholds.
   - [ ] Update WebSocket broadcasts to include recycler changes.

4. **Recycler Implementation**
   - [ ] Build recycler service/loop module.
   - [ ] Wire into main server startup/shutdown lifecycle.
   - [ ] Add LLM client abstraction (provider + model selection).
   - [ ] Implement single-task-per-interval processing logic.
   - [ ] Write unit tests covering streak logic and folder transitions.

5. **Execution Processor Updates**
   - [ ] Ensure post-run counters persist without requeueing.
   - [ ] Respect `processor_auto_requeue` when scheduling tasks.

6. **UI Work**
   - [ ] Add Recycler tab in settings modal.
   - [ ] Insert new board columns and styles.
   - [ ] Display streak counters in task details.
   - [ ] Update WebSocket handlers to handle new status events.

7. **Testing & QA**
   - [ ] Manual test: completed task recycled back to pending with updated note.
   - [ ] Manual test: consecutive completions hit threshold → task moves to finalized column.
   - [ ] Manual test: repeated failures hit threshold → task moves to blocked column.
   - [ ] Failure path: missing output produces conservative note.
   - [ ] Model switch (Ollama vs OpenRouter) works end-to-end.

8. **Documentation & Handoff**
   - [ ] Update README or user docs with automation instructions.
   - [ ] Record default settings and how to override via CLI.
   - [ ] Capture known limitations and TODOs.

## Open Questions & Future Enhancements

- File watching instead of polling for faster response (later).
- Batch processing or priority heuristics if queue grows large.
- Historical note archive for audit trail.
- Automated alerts when tasks reach blocked state.
