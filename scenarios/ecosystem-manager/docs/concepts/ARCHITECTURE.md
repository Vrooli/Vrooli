# Ecosystem Manager Architecture

## Mental Model
- **Tasks** are the primary unit of work (resource or scenario), each with an operation (generator or improver), targets, and steering configuration.
- **Queue processing** orchestrates task execution, prompt assembly, agent runs, and execution history.
- **Auto Steer** applies multi-phase steering profiles, storing active state in Postgres while profile definitions live on disk.
- **UI** configures tasks and visualizes state through the HTTP API.

## Core Flows
1. **Task create/update**
   - HTTP handler validates inputs, persists task state, and wakes the queue.
   - If an Auto Steer profile is attached, the handler initializes execution state as a best-effort step.

2. **Task execution**
   - Execution manager assembles prompts, initializes Auto Steer state if needed, and executes via agent-manager.
   - Auto Steer evaluation advances phases or stops based on metrics.

3. **Auto Steer state visibility**
   - UI queries `/auto-steer/execution/{taskId}` to render phase/iteration context.
   - Manual initialization uses `/auto-steer/execution/seek` when no state exists yet.

## Key Boundaries
- **Presentation (UI):** React components, task configuration, and state visualization.
- **HTTP handlers:** Input validation, task transitions, and response shaping.
- **Orchestration:** Queue processor and execution manager coordinate work.
- **Auto Steer domain:** Profile registry (filesystem) + execution state/history (Postgres).
- **Cross-cutting:** Systemlog for audit visibility and troubleshooting.

## Auto Steer Initialization Paths
- **Execution-time:** Execution manager initializes before running a task.
- **Configuration-time:** Task create/update initializes when a profile is attached.
- **Manual:** UI triggers initialization via seek when state is missing.
