# Coherence Notes

## UI Coherence Snapshot

1. UI follows domain grouping across kanban, settings, steer, and insights components.
2. Shared primitives under `components/ui` keep style and behavior consistent.
3. Data-fetching hooks are grouped by domain concern.

[CODE: ui/src/components/kanban/KanbanBoard.tsx]
[CODE: ui/src/components/modals/SettingsModal.tsx]
[CODE: ui/src/hooks/useTasks.ts]
