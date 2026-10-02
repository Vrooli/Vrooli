import { describe, it, expect } from 'vitest';
import { fromJson } from '@bufbuild/protobuf';
import { TimelineEntrySchema } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import type { TimelineEntry as RecordingTimelineEntry } from '../api/schemas';
import {
  attachTimelinePageIdentities,
  mergeTimelineItemsWithAISteps,
  timelineEntryToTimelineItem,
  recordingEntryToTimelineItem,
  workflowNodesToTimelineItems,
  updateTimelineItemStatus,
} from './timeline-unified';
import type { RecordedAction } from './types';

/**
 * Test suite for timeline unification and AI reconciliation utilities.
 *
 * These tests verify the AI correlation logic that matches AI navigation
 * decisions with recorded browser actions. This enables users to see
 * both what happened AND why the AI did it.
 */

// Helper to create test recorded actions
function createRecordedAction(
  overrides: Partial<RecordedAction> & { id: string; actionType: RecordedAction['actionType'] }
): RecordedAction {
  return {
    sessionId: 'test-session',
    sequenceNum: 1,
    timestamp: new Date().toISOString(),
    confidence: 1.0,
    url: 'https://example.com',
    ...overrides,
  };
}

describe('attachTimelinePageIdentities', () => {
  it('joins driver actions to durable logical page identities by action ID', () => {
    const actions = [
      createRecordedAction({ id: 'main-1', actionType: 'click' }),
      createRecordedAction({ id: 'popup-1', actionType: 'click' }),
      createRecordedAction({ id: 'unmatched', actionType: 'click', pageId: 'driver-fallback' }),
    ];
    const entries: RecordingTimelineEntry[] = [
      {
        type: 'action',
        pageId: 'logical-main',
        entry: fromJson(TimelineEntrySchema, {
          id: 'main-1', sequence_num: 1, timestamp: '2026-09-24T00:00:00Z',
          action: { type: 'ACTION_TYPE_CLICK', click: { selector: 'button' } },
        }, { jsonOptions: { useProtoNames: true } }),
      },
      {
        type: 'action',
        pageId: 'logical-popup',
        entry: fromJson(TimelineEntrySchema, {
          id: 'popup-1', sequence_num: 2, timestamp: '2026-09-24T00:00:01Z',
          action: { type: 'ACTION_TYPE_CLICK', click: { selector: 'button' } },
        }, { jsonOptions: { useProtoNames: true } }),
      },
    ];

    const identified = attachTimelinePageIdentities(actions, entries);

    expect(identified.map((action) => action.pageId)).toEqual([
      'logical-main',
      'logical-popup',
      'driver-fallback',
    ]);
    expect(actions[0]?.pageId).toBeUndefined();
  });
});

describe('timelineEntryToTimelineItem', () => {
  it('projects the generated proto entry into the recording view with timing and failure details', () => {
    const entry = fromJson(
      TimelineEntrySchema,
      {
        id: 'entry-9',
        sequence_num: 9,
        step_index: 4,
        timestamp: '2026-10-01T04:00:00Z',
        action: {
          type: 'ACTION_TYPE_INPUT',
          input: { selector: '#email', value: 'person@example.test' },
        },
        telemetry: { url: 'https://example.test/form' },
        context: { success: false, error: 'input timed out' },
      },
      { jsonOptions: { useProtoNames: true, ignoreUnknownFields: false } },
    );

    const item = timelineEntryToTimelineItem(entry);

    expect(item).toMatchObject({
      id: 'entry-9',
      sequenceNum: 9,
      actionType: 'input',
      selector: '#email',
      url: 'https://example.test/form',
      success: false,
      error: 'input timed out',
      payload: { text: 'person@example.test' },
    });
    expect(item.timestamp.toISOString()).toBe('2026-10-01T04:00:00.000Z');
  });
});

// Helper to create test AI steps
interface AIStepForTest {
  id: string;
  stepNumber: number;
  action: {
    type: string;
    text?: string;
    url?: string;
  };
  reasoning: string;
  currentUrl: string;
  goalAchieved: boolean;
  tokensUsed: {
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
  };
  durationMs: number;
  error?: string;
  timestamp: Date;
}

function createAIStep(overrides: Partial<AIStepForTest> & { id: string; type: string }): AIStepForTest {
  return {
    stepNumber: 1,
    action: {
      type: overrides.type,
      text: overrides.action?.text,
      url: overrides.action?.url,
    },
    reasoning: 'Test reasoning',
    currentUrl: 'https://example.com',
    goalAchieved: false,
    tokensUsed: {
      promptTokens: 100,
      completionTokens: 50,
      totalTokens: 150,
    },
    durationMs: 1000,
    timestamp: new Date(),
    ...overrides,
    // Ensure action is properly merged
    ...(overrides.action ? { action: { type: overrides.type, ...overrides.action } } : {}),
  };
}

describe('mergeTimelineItemsWithAISteps', () => {
  const makeItem = (id: string, actionType: string, timestamp: Date, entryType: 'action' | 'page_event' = 'action'): TimelineItem => ({
    id, sequenceNum: 1, actionType, timestamp, mode: 'recording', entryType,
  });

  it('attaches matching AI metadata without changing the journal projection', () => {
    const item = makeItem('entry-1', 'input', new Date('2024-01-01T12:00:00Z'));
    const aiStep = createAIStep({
      id: 'ai-1', type: 'type', reasoning: 'Enter the requested value', goalAchieved: true,
      timestamp: new Date('2024-01-01T12:00:00.500Z'),
    });

    const result = mergeTimelineItemsWithAISteps([item], [aiStep]);

    expect(result).toHaveLength(1);
    expect(result[0]).toMatchObject({ id: 'entry-1', actionType: 'input', isAI: true });
    expect(result[0]?.aiMetadata).toEqual({
      reasoning: 'Enter the requested value', tokensUsed: aiStep.tokensUsed, goalAchieved: true,
    });
  });

  it('preserves unmatched actions and page events in their original order', () => {
    const action = makeItem('entry-1', 'click', new Date('2024-01-01T12:00:00Z'));
    const pageEvent = makeItem('event-1', 'page_created', new Date('2024-01-01T12:00:01Z'), 'page_event');
    const aiStep = createAIStep({ id: 'ai-1', type: 'navigate', timestamp: new Date('2024-01-01T12:00:01Z') });

    const result = mergeTimelineItemsWithAISteps([action, pageEvent], [aiStep]);

    expect(result).toEqual([action, pageEvent]);
  });
});

describe('recordingEntryToTimelineItem', () => {
  it('converts action entry correctly', () => {
    const entry: RecordingTimelineEntry = {
      type: 'action',
      pageId: 'page-1',
      entry: fromJson(TimelineEntrySchema, {
        id: 'action-1', sequence_num: 1, timestamp: '2024-01-01T12:00:00Z',
        action: { type: 'ACTION_TYPE_CLICK', click: { selector: 'button' }, metadata: { label: 'Test Page' } },
        telemetry: { url: 'https://example.com' }, context: { session_id: 'session' },
      }, { jsonOptions: { useProtoNames: true } }),
    };

    const result = recordingEntryToTimelineItem(entry);

    expect(result.id).toBe('action-1');
    expect(result.sequenceNum).toBe(1);
    expect(result.actionType).toBe('click');
    expect(result.selector).toBe('button');
    expect(result.mode).toBe('recording');
    expect(result.entryType).toBe('action');
    expect(result.url).toBe('https://example.com');
  });

  it('converts page_created event correctly', () => {
    const entry: RecordingTimelineEntry = {
      type: 'page_event',
      pageId: 'page-2',
      pageEvent: {
        id: 'event-1',
        type: 'page_created',
        pageId: 'page-2',
        url: 'https://example.com/new',
        title: 'New Tab',
        timestamp: '2024-01-01T12:00:00Z',
      },
    };

    const result = recordingEntryToTimelineItem(entry);

    expect(result.id).toBe('event-1');
    expect(result.actionType).toBe('page_created');
    expect(result.mode).toBe('recording');
    expect(result.entryType).toBe('page_event');
    expect(result.pageEventType).toBe('page_created');
    expect(result.url).toBe('https://example.com/new');
    expect(result.pageTitle).toBe('New Tab');
  });

  it('converts page_navigated event correctly', () => {
    const entry: RecordingTimelineEntry = {
      type: 'page_event',
      pageId: 'page-1',
      pageEvent: {
        id: 'event-1',
        type: 'page_navigated',
        pageId: 'page-1',
        url: 'https://example.com/page2',
        title: 'Page 2',
        timestamp: '2024-01-01T12:00:00Z',
      },
    };

    const result = recordingEntryToTimelineItem(entry);

    expect(result.actionType).toBe('page_navigated');
    expect(result.pageEventType).toBe('page_navigated');
  });

  it('converts page_closed event correctly', () => {
    const entry: RecordingTimelineEntry = {
      type: 'page_event',
      pageId: 'page-1',
      pageEvent: {
        id: 'event-1',
        type: 'page_closed',
        pageId: 'page-1',
        timestamp: '2024-01-01T12:00:00Z',
      },
    };

    const result = recordingEntryToTimelineItem(entry);

    expect(result.actionType).toBe('page_closed');
    expect(result.pageEventType).toBe('page_closed');
  });

  it('handles a proto entry without an action', () => {
    const entry: RecordingTimelineEntry = {
      type: 'action',
      pageId: 'page-1',
      entry: fromJson(TimelineEntrySchema, { id: 'entry-1', sequence_num: 1 }),
    };

    const result = recordingEntryToTimelineItem(entry);

    expect(result.id).toBe('entry-1');
    expect(result.actionType).toBe('unknown');
    expect(result.mode).toBe('recording');
  });
});

describe('workflowNodesToTimelineItems', () => {
  it('converts action nodes to timeline items', () => {
    const nodes = [
      {
        id: 'node-1',
        action: { type: 'ACTION_TYPE_NAVIGATE', metadata: { label: 'Go to homepage' }, navigate: { url: 'https://example.com' } },
      },
      {
        id: 'node-2',
        action: { type: 'ACTION_TYPE_CLICK', metadata: { label: 'Click login' }, click: { selector: 'button#login' } },
      },
    ];

    const result = workflowNodesToTimelineItems(nodes, []);

    expect(result).toHaveLength(2);
    expect(result[0].nodeId).toBe('node-1');
    expect(result[0].actionType).toBe('navigate');
    expect(result[0].url).toBe('https://example.com');
    expect(result[0].executionStatus).toBe('pending');
    expect(result[0].mode).toBe('execution');

    expect(result[1].nodeId).toBe('node-2');
    expect(result[1].actionType).toBe('click');
    expect(result[1].selector).toBe('button#login');
  });

  it('filters out non-action nodes (start, end, etc.)', () => {
    const nodes = [
      { id: 'start-1' },
      { id: 'node-1', action: { type: 'ACTION_TYPE_CLICK', click: { selector: 'button' } } },
      { id: 'end-1' },
    ];

    const result = workflowNodesToTimelineItems(nodes, []);

    expect(result).toHaveLength(1);
    expect(result[0].nodeId).toBe('node-1');
    expect(result[0].actionType).toBe('click');
  });

  it('handles V2 format nodes with action.type', () => {
    const nodes = [
      {
        id: 'node-1',
        action: {
          type: 'ACTION_TYPE_NAVIGATE',
          metadata: { label: 'Navigate' },
          navigate: { url: 'https://example.com' },
        },
      },
    ];

    const result = workflowNodesToTimelineItems(nodes, []);

    expect(result).toHaveLength(1);
    expect(result[0].actionType).toBe('navigate');
    expect(result[0].url).toBe('https://example.com');
  });

  it('handles empty nodes array', () => {
    const result = workflowNodesToTimelineItems([], []);
    expect(result).toEqual([]);
  });

  it('assigns sequential sequence numbers', () => {
    const nodes = [
      { id: 'node-1', action: { type: 'ACTION_TYPE_CLICK', click: { selector: '.one' } } },
      { id: 'node-2', action: { type: 'ACTION_TYPE_INPUT', input: { selector: '.two', value: 'text' } } },
      { id: 'node-3', action: { type: 'ACTION_TYPE_CLICK', click: { selector: '.three' } } },
    ];

    const result = workflowNodesToTimelineItems(nodes, []);

    expect(result[0].sequenceNum).toBe(1);
    expect(result[1].sequenceNum).toBe(2);
    expect(result[2].sequenceNum).toBe(3);
  });
});

describe('updateTimelineItemStatus', () => {
  const createExecutionItem = (nodeId: string) => ({
    id: `item-${nodeId}`,
    nodeId,
    sequenceNum: 1,
    timestamp: new Date(),
    actionType: 'click',
    mode: 'execution' as const,
    executionStatus: 'pending' as const,
    entryType: 'action' as const,
  });

  it('updates status of matching node', () => {
    const items = [
      createExecutionItem('node-1'),
      createExecutionItem('node-2'),
    ];

    const result = updateTimelineItemStatus(items, 'node-1', 'running');

    expect(result[0].executionStatus).toBe('running');
    expect(result[1].executionStatus).toBe('pending');
  });

  it('sets success=true when status is completed', () => {
    const items = [createExecutionItem('node-1')];

    const result = updateTimelineItemStatus(items, 'node-1', 'completed');

    expect(result[0].executionStatus).toBe('completed');
    expect(result[0].success).toBe(true);
  });

  it('sets success=false and error when status is failed', () => {
    const items = [createExecutionItem('node-1')];

    const result = updateTimelineItemStatus(items, 'node-1', 'failed', 'Element not found');

    expect(result[0].executionStatus).toBe('failed');
    expect(result[0].success).toBe(false);
    expect(result[0].error).toBe('Element not found');
  });

  it('updates duration when provided', () => {
    const items = [createExecutionItem('node-1')];

    const result = updateTimelineItemStatus(items, 'node-1', 'completed', undefined, 250);

    expect(result[0].durationMs).toBe(250);
  });

  it('updates timestamp when status changes to running', () => {
    const originalTime = new Date('2024-01-01T12:00:00Z');
    const items = [{
      ...createExecutionItem('node-1'),
      timestamp: originalTime,
    }];

    const beforeUpdate = Date.now();
    const result = updateTimelineItemStatus(items, 'node-1', 'running');
    const afterUpdate = Date.now();

    expect(result[0].timestamp.getTime()).toBeGreaterThanOrEqual(beforeUpdate);
    expect(result[0].timestamp.getTime()).toBeLessThanOrEqual(afterUpdate);
  });

  it('preserves original timestamp for non-running status', () => {
    const originalTime = new Date('2024-01-01T12:00:00Z');
    const items = [{
      ...createExecutionItem('node-1'),
      timestamp: originalTime,
    }];

    const result = updateTimelineItemStatus(items, 'node-1', 'completed');

    expect(result[0].timestamp).toEqual(originalTime);
  });

  it('returns new array without mutating original', () => {
    const items = [createExecutionItem('node-1')];
    const originalStatus = items[0].executionStatus;

    const result = updateTimelineItemStatus(items, 'node-1', 'running');

    expect(items[0].executionStatus).toBe(originalStatus);
    expect(result).not.toBe(items);
    expect(result[0]).not.toBe(items[0]);
  });
});
