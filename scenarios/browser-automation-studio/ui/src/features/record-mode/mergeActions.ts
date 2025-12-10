/**
 * Action Merging Utilities
 *
 * Client-side implementation that mirrors the backend mergeConsecutiveActions logic
 * from api/handlers/record_mode.go. This ensures the timeline preview matches
 * what users will get in their generated workflow.
 *
 * Merging rules:
 * 1. Consecutive type actions on the same selector → text concatenated
 * 2. Consecutive scroll actions → single action with final position
 * 3. Focus events before type on same element → removed (implicit in typing)
 */

import type { RecordedAction, SelectorSet } from './types';

/**
 * Metadata about merged actions for UI display.
 */
export interface MergedActionMeta {
  /** Number of original actions merged into this one */
  mergedCount: number;
  /** Original action IDs that were merged */
  mergedIds: string[];
  /** Type of merge applied */
  mergeType: 'type' | 'scroll' | 'navigate' | 'focus-removed' | null;
}

/**
 * RecordedAction extended with merge metadata.
 */
export type MergedAction = RecordedAction & {
  _merged?: MergedActionMeta;
};

/**
 * Check if two SelectorSets refer to the same element.
 */
function selectorsMatch(a?: SelectorSet, b?: SelectorSet): boolean {
  if (!a || !b) return false;
  return a.primary === b.primary;
}

/**
 * Merge consecutive actions to create a cleaner timeline.
 *
 * This mirrors the backend logic in api/handlers/record_mode.go:mergeConsecutiveActions
 * to ensure "what you see = what you get" in the workflow.
 *
 * @param actions - Raw recorded actions
 * @returns Merged actions with metadata about merges applied
 */
export function mergeConsecutiveActions(actions: RecordedAction[]): MergedAction[] {
  if (actions.length <= 1) {
    return actions.map((a) => ({ ...a }));
  }

  const merged: MergedAction[] = [];

  for (let i = 0; i < actions.length; i++) {
    const action = { ...actions[i] } as MergedAction;
    const mergedIds: string[] = [action.id];

    // Skip focus events that are immediately followed by type on the same element
    if (action.actionType === 'focus' && i + 1 < actions.length) {
      const next = actions[i + 1];
      if (next.actionType === 'type' && selectorsMatch(action.selector, next.selector)) {
        // Mark the focus as removed but don't add it to merged output
        // The next action will include info about the removed focus
        continue;
      }
    }

    // Merge consecutive type actions on same selector
    if (action.actionType === 'type' && action.selector) {
      let mergedText = '';
      if (action.payload?.text) {
        mergedText = String(action.payload.text);
      }

      // Look ahead for more type actions on same element
      while (i + 1 < actions.length) {
        const next = actions[i + 1];
        if (next.actionType !== 'type' || !selectorsMatch(action.selector, next.selector)) {
          break;
        }
        // Merge the text
        if (next.payload?.text) {
          mergedText += String(next.payload.text);
        }
        mergedIds.push(next.id);
        i++; // Skip this action, we've merged it
      }

      // Update the action with merged text
      if (mergedIds.length > 1 || mergedText !== String(action.payload?.text || '')) {
        action.payload = { ...action.payload, text: mergedText };
        action._merged = {
          mergedCount: mergedIds.length,
          mergedIds,
          mergeType: 'type',
        };
      }
    }

    // Merge consecutive scroll actions
    if (action.actionType === 'scroll') {
      let finalScrollX = action.payload?.scrollX as number | undefined;
      let finalScrollY = action.payload?.scrollY as number | undefined;
      let totalDeltaX = (action.payload?.deltaX as number) || 0;
      let totalDeltaY = (action.payload?.deltaY as number) || 0;

      // Look ahead for more scroll actions
      while (i + 1 < actions.length) {
        const next = actions[i + 1];
        if (next.actionType !== 'scroll') {
          break;
        }
        // Accumulate deltas and use final position
        if (next.payload?.scrollX !== undefined) {
          finalScrollX = next.payload.scrollX as number;
        }
        if (next.payload?.scrollY !== undefined) {
          finalScrollY = next.payload.scrollY as number;
        }
        totalDeltaX += (next.payload?.deltaX as number) || 0;
        totalDeltaY += (next.payload?.deltaY as number) || 0;
        mergedIds.push(next.id);
        i++; // Skip this action, we've merged it
      }

      // Update the action with final scroll position and total delta
      if (mergedIds.length > 1) {
        action.payload = {
          ...action.payload,
          scrollX: finalScrollX,
          scrollY: finalScrollY,
          deltaX: totalDeltaX,
          deltaY: totalDeltaY,
        };
        action._merged = {
          mergedCount: mergedIds.length,
          mergedIds,
          mergeType: 'scroll',
        };
      }
    }

    // Merge consecutive navigate actions (e.g., redirect chains, rapid navigations)
    // Keep only the final URL, similar to scroll merging
    if (action.actionType === 'navigate') {
      let finalUrl = action.payload?.targetUrl as string | undefined;

      // Look ahead for more navigate actions
      while (i + 1 < actions.length) {
        const next = actions[i + 1];
        if (next.actionType !== 'navigate') {
          break;
        }
        // Use the final URL from the chain
        if (next.payload?.targetUrl) {
          finalUrl = next.payload.targetUrl as string;
        }
        mergedIds.push(next.id);
        i++; // Skip this action, we've merged it
      }

      // Update the action with final URL
      if (mergedIds.length > 1) {
        action.payload = {
          ...action.payload,
          targetUrl: finalUrl,
        };
        action.url = finalUrl || action.url;
        action._merged = {
          mergedCount: mergedIds.length,
          mergedIds,
          mergeType: 'navigate',
        };
      }
    }

    // Check if the previous action was a focus that should be noted
    if (
      merged.length === 0 &&
      i > 0 &&
      actions[i - 1].actionType === 'focus' &&
      action.actionType === 'type' &&
      selectorsMatch(actions[i - 1].selector, action.selector)
    ) {
      // This type action had a focus removed before it
      if (!action._merged) {
        action._merged = {
          mergedCount: 1,
          mergedIds: [action.id],
          mergeType: 'focus-removed',
        };
      }
    }

    merged.push(action);
  }

  return merged;
}

/**
 * Get a human-readable description of the merge applied.
 */
export function getMergeDescription(meta?: MergedActionMeta): string | null {
  if (!meta || meta.mergedCount <= 1) return null;

  switch (meta.mergeType) {
    case 'type':
      return `Merged ${meta.mergedCount} keystrokes`;
    case 'scroll':
      return `Merged ${meta.mergedCount} scroll events`;
    case 'navigate':
      return `Merged ${meta.mergedCount} navigation events`;
    case 'focus-removed':
      return 'Focus event removed (implicit)';
    default:
      return `Merged ${meta.mergedCount} actions`;
  }
}
