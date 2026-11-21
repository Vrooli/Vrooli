import type { RecordedAction } from './types/types';

/** Convert canonical selected action IDs to the action-list indices expected by the form. */
export function selectedActionIndicesForIds(
  selectedIds: string[],
  actions: RecordedAction[],
): number[] {
  const selected = new Set(selectedIds);
  return actions.flatMap((action, index) => selected.has(action.id) ? [index] : []);
}

/**
 * Resolve the index contract used by WorkflowCreationForm against its
 * `actions` prop. Timeline selections are translated from stable IDs into
 * these action-list indices by RecordModeWorkspace before reaching the form.
 */
export function selectRecordedActions(
  actionIndices: number[],
  actions: RecordedAction[],
): RecordedAction[] {
  const selected: RecordedAction[] = [];
  const included = new Set<number>();

  for (const index of actionIndices) {
    if (!Number.isInteger(index) || index < 0 || index >= actions.length || included.has(index)) continue;

    const action = actions[index];
    if (!action) continue;

    selected.push(action);
    included.add(index);
  }

  return selected;
}
