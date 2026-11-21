import { describe, expect, it } from 'vitest';
import type { RecordedAction } from './types/types';
import { selectRecordedActions, selectedActionIndicesForIds } from './selectedActions';

const actions: RecordedAction[] = [
  { id: 'action-a', actionType: 'click', timestamp: '2026-01-01T00:00:00.000Z' },
  { id: 'action-b', actionType: 'type', timestamp: '2026-01-01T00:00:01.000Z' },
];

describe('selectRecordedActions', () => {
  it('resolves selected action-list indices in the requested order', () => {
    const selected = selectRecordedActions([1, 0], actions);

    expect(selected).toEqual([actions[1], actions[0]]);
  });

  it('removes duplicate action indices', () => {
    const selected = selectRecordedActions([1, 0, 1], actions);

    expect(selected).toEqual([actions[1], actions[0]]);
  });

  it('ignores invalid or stale indices instead of widening to every action', () => {
    expect(selectRecordedActions([-1, 2, 1.5], actions)).toEqual([]);
  });
});

describe('selectedActionIndicesForIds', () => {
  it('maps selected timeline IDs to indices in the filtered action list', () => {
    expect(selectedActionIndicesForIds(['action-b', 'page-event', 'action-a'], actions)).toEqual([0, 1]);
  });

  it('ignores stale and duplicate IDs while preserving action-list order', () => {
    expect(selectedActionIndicesForIds(['missing', 'action-b', 'action-b'], actions)).toEqual([1]);
  });
});
