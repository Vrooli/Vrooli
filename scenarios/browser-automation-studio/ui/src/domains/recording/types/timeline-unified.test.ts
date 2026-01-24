import { describe, it, expect } from 'vitest';
import {
  mergeActionsWithAISteps,
  recordedActionToTimelineItem,
  useTimelineEntryToTimelineItem,
  workflowNodesToTimelineItems,
  updateTimelineItemStatus,
  type UseTimelineEntry,
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

describe('mergeActionsWithAISteps', () => {
  describe('basic functionality', () => {
    it('returns timeline items without AI metadata when no AI steps provided', () => {
      const actions: RecordedAction[] = [
        createRecordedAction({ id: '1', actionType: 'click' }),
        createRecordedAction({ id: '2', actionType: 'input' }),
      ];

      const result = mergeActionsWithAISteps(actions, []);

      expect(result).toHaveLength(2);
      expect(result[0].isAI).toBeFalsy();
      expect(result[0].aiMetadata).toBeUndefined();
      expect(result[1].isAI).toBeFalsy();
      expect(result[1].aiMetadata).toBeUndefined();
    });

    it('handles empty actions array', () => {
      const result = mergeActionsWithAISteps([], []);
      expect(result).toEqual([]);
    });
  });

  describe('timestamp-based matching', () => {
    it('matches AI steps to actions by timestamp proximity within 5s window', () => {
      const baseTime = new Date('2024-01-01T12:00:00Z');
      const actions: RecordedAction[] = [
        createRecordedAction({
          id: '1',
          actionType: 'click',
          timestamp: baseTime.toISOString(),
        }),
      ];
      const aiSteps: AIStepForTest[] = [
        createAIStep({
          id: 'ai-1',
          type: 'click',
          reasoning: 'Clicking the submit button',
          timestamp: new Date(baseTime.getTime() + 1000), // 1 second later
        }),
      ];

      const result = mergeActionsWithAISteps(actions, aiSteps);

      expect(result).toHaveLength(1);
      expect(result[0].isAI).toBe(true);
      expect(result[0].aiMetadata?.reasoning).toBe('Clicking the submit button');
    });

    it('does not match when outside 5-second window', () => {
      const baseTime = new Date('2024-01-01T12:00:00Z');
      const actions: RecordedAction[] = [
        createRecordedAction({
          id: '1',
          actionType: 'click',
          timestamp: baseTime.toISOString(),
        }),
      ];
      const aiSteps: AIStepForTest[] = [
        createAIStep({
          id: 'ai-1',
          type: 'click',
          reasoning: 'Too far away',
          timestamp: new Date(baseTime.getTime() + 6000), // 6 seconds later (outside window)
        }),
      ];

      const result = mergeActionsWithAISteps(actions, aiSteps);

      expect(result).toHaveLength(1);
      expect(result[0].isAI).toBeFalsy();
      expect(result[0].aiMetadata).toBeUndefined();
    });

    it('selects closest match when multiple candidates exist', () => {
      const baseTime = new Date('2024-01-01T12:00:00Z');
      const actions: RecordedAction[] = [
        createRecordedAction({
          id: '1',
          actionType: 'click',
          timestamp: baseTime.toISOString(),
        }),
      ];
      const aiSteps: AIStepForTest[] = [
        createAIStep({
          id: 'ai-1',
          type: 'click',
          reasoning: 'Farther match',
          timestamp: new Date(baseTime.getTime() + 3000), // 3 seconds
        }),
        createAIStep({
          id: 'ai-2',
          type: 'click',
          reasoning: 'Closest match',
          timestamp: new Date(baseTime.getTime() + 500), // 0.5 seconds
        }),
      ];

      const result = mergeActionsWithAISteps(actions, aiSteps);

      expect(result).toHaveLength(1);
      expect(result[0].aiMetadata?.reasoning).toBe('Closest match');
    });
  });

  describe('action type matching', () => {
    it('only matches when action types match', () => {
      const baseTime = new Date('2024-01-01T12:00:00Z');
      const actions: RecordedAction[] = [
        createRecordedAction({
          id: '1',
          actionType: 'click',
          timestamp: baseTime.toISOString(),
        }),
      ];
      const aiSteps: AIStepForTest[] = [
        createAIStep({
          id: 'ai-1',
          type: 'input', // Different type
          reasoning: 'Wrong type',
          timestamp: new Date(baseTime.getTime() + 500),
        }),
      ];

      const result = mergeActionsWithAISteps(actions, aiSteps);

      expect(result).toHaveLength(1);
      expect(result[0].isAI).toBeFalsy();
    });

    it('normalizes "type" AI action to "input" for matching', () => {
      const baseTime = new Date('2024-01-01T12:00:00Z');
      const actions: RecordedAction[] = [
        createRecordedAction({
          id: '1',
          actionType: 'input',
          timestamp: baseTime.toISOString(),
        }),
      ];
      const aiSteps: AIStepForTest[] = [
        createAIStep({
          id: 'ai-1',
          type: 'type', // AI uses "type" but recorder uses "input"
          reasoning: 'Typing text',
          timestamp: new Date(baseTime.getTime() + 500),
        }),
      ];

      const result = mergeActionsWithAISteps(actions, aiSteps);

      expect(result).toHaveLength(1);
      expect(result[0].isAI).toBe(true);
      expect(result[0].aiMetadata?.reasoning).toBe('Typing text');
    });

    it('normalizes "keypress" AI action to "keyboard" for matching', () => {
      const baseTime = new Date('2024-01-01T12:00:00Z');
      const actions: RecordedAction[] = [
        createRecordedAction({
          id: '1',
          actionType: 'keyboard',
          timestamp: baseTime.toISOString(),
        }),
      ];
      const aiSteps: AIStepForTest[] = [
        createAIStep({
          id: 'ai-1',
          type: 'keypress', // AI uses "keypress" but recorder uses "keyboard"
          reasoning: 'Pressing Enter',
          timestamp: new Date(baseTime.getTime() + 500),
        }),
      ];

      const result = mergeActionsWithAISteps(actions, aiSteps);

      expect(result).toHaveLength(1);
      expect(result[0].isAI).toBe(true);
    });
  });

  describe('consumption behavior (preventing duplicates)', () => {
    it('consumes matched AI steps to prevent duplicate attribution', () => {
      const baseTime = new Date('2024-01-01T12:00:00Z');
      const actions: RecordedAction[] = [
        createRecordedAction({
          id: '1',
          actionType: 'click',
          timestamp: baseTime.toISOString(),
        }),
        createRecordedAction({
          id: '2',
          actionType: 'click',
          timestamp: new Date(baseTime.getTime() + 1000).toISOString(),
        }),
      ];
      // Single AI step should only match one action
      const aiSteps: AIStepForTest[] = [
        createAIStep({
          id: 'ai-1',
          type: 'click',
          reasoning: 'Single AI reasoning',
          timestamp: new Date(baseTime.getTime() + 500),
        }),
      ];

      const result = mergeActionsWithAISteps(actions, aiSteps);

      expect(result).toHaveLength(2);
      // First action gets the AI match (closest)
      expect(result[0].isAI).toBe(true);
      expect(result[0].aiMetadata?.reasoning).toBe('Single AI reasoning');
      // Second action should NOT get the same AI step
      expect(result[1].isAI).toBeFalsy();
    });

    it('leaves unmatched actions without AI metadata', () => {
      const baseTime = new Date('2024-01-01T12:00:00Z');
      const actions: RecordedAction[] = [
        createRecordedAction({
          id: '1',
          actionType: 'click',
          timestamp: baseTime.toISOString(),
        }),
        createRecordedAction({
          id: '2',
          actionType: 'scroll', // No matching AI step for scroll
          timestamp: new Date(baseTime.getTime() + 2000).toISOString(),
        }),
      ];
      const aiSteps: AIStepForTest[] = [
        createAIStep({
          id: 'ai-1',
          type: 'click',
          reasoning: 'Click reasoning',
          timestamp: new Date(baseTime.getTime() + 500),
        }),
      ];

      const result = mergeActionsWithAISteps(actions, aiSteps);

      expect(result).toHaveLength(2);
      expect(result[0].isAI).toBe(true);
      expect(result[1].isAI).toBeFalsy();
    });
  });

  describe('AI metadata attachment', () => {
    it('attaches AI reasoning to matching actions', () => {
      const baseTime = new Date('2024-01-01T12:00:00Z');
      const actions: RecordedAction[] = [
        createRecordedAction({
          id: '1',
          actionType: 'click',
          timestamp: baseTime.toISOString(),
        }),
      ];
      const aiSteps: AIStepForTest[] = [
        createAIStep({
          id: 'ai-1',
          type: 'click',
          reasoning: 'Detailed reasoning about why this click was performed',
          timestamp: new Date(baseTime.getTime() + 500),
        }),
      ];

      const result = mergeActionsWithAISteps(actions, aiSteps);

      expect(result[0].aiMetadata?.reasoning).toBe('Detailed reasoning about why this click was performed');
    });

    it('attaches token usage to matching actions', () => {
      const baseTime = new Date('2024-01-01T12:00:00Z');
      const actions: RecordedAction[] = [
        createRecordedAction({
          id: '1',
          actionType: 'click',
          timestamp: baseTime.toISOString(),
        }),
      ];
      const aiSteps: AIStepForTest[] = [
        createAIStep({
          id: 'ai-1',
          type: 'click',
          reasoning: 'Test',
          tokensUsed: {
            promptTokens: 200,
            completionTokens: 100,
            totalTokens: 300,
          },
          timestamp: new Date(baseTime.getTime() + 500),
        }),
      ];

      const result = mergeActionsWithAISteps(actions, aiSteps);

      expect(result[0].aiMetadata?.tokensUsed).toEqual({
        promptTokens: 200,
        completionTokens: 100,
        totalTokens: 300,
      });
    });

    it('attaches goalAchieved status to matching actions', () => {
      const baseTime = new Date('2024-01-01T12:00:00Z');
      const actions: RecordedAction[] = [
        createRecordedAction({
          id: '1',
          actionType: 'click',
          timestamp: baseTime.toISOString(),
        }),
      ];
      const aiSteps: AIStepForTest[] = [
        createAIStep({
          id: 'ai-1',
          type: 'click',
          reasoning: 'Final action',
          goalAchieved: true,
          timestamp: new Date(baseTime.getTime() + 500),
        }),
      ];

      const result = mergeActionsWithAISteps(actions, aiSteps);

      expect(result[0].aiMetadata?.goalAchieved).toBe(true);
    });
  });
});

describe('recordedActionToTimelineItem', () => {
  it('converts basic recorded action to timeline item', () => {
    const action = createRecordedAction({
      id: 'action-1',
      actionType: 'click',
      sequenceNum: 5,
      timestamp: '2024-01-01T12:00:00Z',
      durationMs: 150,
      selector: { primary: 'button#submit', candidates: [] },
      url: 'https://example.com/page',
      pageId: 'page-1',
      pageTitle: 'Example Page',
    });

    const result = recordedActionToTimelineItem(action);

    expect(result.id).toBe('action-1');
    expect(result.sequenceNum).toBe(5);
    expect(result.timestamp).toEqual(new Date('2024-01-01T12:00:00Z'));
    expect(result.durationMs).toBe(150);
    expect(result.actionType).toBe('click');
    expect(result.selector).toBe('button#submit');
    expect(result.url).toBe('https://example.com/page');
    expect(result.success).toBe(true);
    expect(result.mode).toBe('recording');
    expect(result.pageId).toBe('page-1');
    expect(result.entryType).toBe('action');
    expect(result.pageTitle).toBe('Example Page');
  });

  it('converts action without AI metadata', () => {
    const action = createRecordedAction({ id: '1', actionType: 'click' });
    const result = recordedActionToTimelineItem(action);

    expect(result.isAI).toBeFalsy();
    expect(result.aiMetadata).toBeUndefined();
  });

  it('converts action with AI metadata', () => {
    const action = createRecordedAction({ id: '1', actionType: 'click' });
    const aiMetadata = {
      reasoning: 'AI reasoning',
      tokensUsed: { promptTokens: 100, completionTokens: 50, totalTokens: 150 },
      goalAchieved: false,
    };

    const result = recordedActionToTimelineItem(action, aiMetadata);

    expect(result.isAI).toBe(true);
    expect(result.aiMetadata).toEqual(aiMetadata);
  });

  it('handles action without selector', () => {
    const action = createRecordedAction({ id: '1', actionType: 'navigate' });
    const result = recordedActionToTimelineItem(action);

    expect(result.selector).toBeUndefined();
  });
});

describe('useTimelineEntryToTimelineItem', () => {
  it('converts action entry correctly', () => {
    const entry: UseTimelineEntry = {
      id: 'entry-1',
      type: 'action',
      timestamp: '2024-01-01T12:00:00Z',
      pageId: 'page-1',
      action: {
        id: 'action-1',
        actionType: 'click',
        url: 'https://example.com',
        sequenceNum: 1,
        timestamp: '2024-01-01T12:00:00Z',
        selector: { primary: 'button' },
        confidence: 0.95,
        pageTitle: 'Test Page',
      },
    };

    const result = useTimelineEntryToTimelineItem(entry);

    expect(result.id).toBe('action-1');
    expect(result.sequenceNum).toBe(1);
    expect(result.actionType).toBe('click');
    expect(result.selector).toBe('button');
    expect(result.mode).toBe('recording');
    expect(result.entryType).toBe('action');
    expect(result.pageTitle).toBe('Test Page');
  });

  it('converts page_created event correctly', () => {
    const entry: UseTimelineEntry = {
      id: 'entry-1',
      type: 'page_event',
      timestamp: '2024-01-01T12:00:00Z',
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

    const result = useTimelineEntryToTimelineItem(entry);

    expect(result.id).toBe('entry-1');
    expect(result.actionType).toBe('page_created');
    expect(result.mode).toBe('recording');
    expect(result.entryType).toBe('page_event');
    expect(result.pageEventType).toBe('page_created');
    expect(result.url).toBe('https://example.com/new');
    expect(result.pageTitle).toBe('New Tab');
  });

  it('converts page_navigated event correctly', () => {
    const entry: UseTimelineEntry = {
      id: 'entry-1',
      type: 'page_event',
      timestamp: '2024-01-01T12:00:00Z',
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

    const result = useTimelineEntryToTimelineItem(entry);

    expect(result.actionType).toBe('page_navigated');
    expect(result.pageEventType).toBe('page_navigated');
  });

  it('converts page_closed event correctly', () => {
    const entry: UseTimelineEntry = {
      id: 'entry-1',
      type: 'page_event',
      timestamp: '2024-01-01T12:00:00Z',
      pageId: 'page-1',
      pageEvent: {
        id: 'event-1',
        type: 'page_closed',
        pageId: 'page-1',
        timestamp: '2024-01-01T12:00:00Z',
      },
    };

    const result = useTimelineEntryToTimelineItem(entry);

    expect(result.actionType).toBe('page_closed');
    expect(result.pageEventType).toBe('page_closed');
  });

  it('handles malformed entry gracefully', () => {
    const entry: UseTimelineEntry = {
      id: 'entry-1',
      type: 'action',
      timestamp: '2024-01-01T12:00:00Z',
      pageId: 'page-1',
      // Missing action property
    };

    const result = useTimelineEntryToTimelineItem(entry);

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
        type: 'navigate',
        data: { label: 'Go to homepage', url: 'https://example.com' },
      },
      {
        id: 'node-2',
        type: 'click',
        data: { label: 'Click login', selector: 'button#login' },
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
      { id: 'start-1', type: 'start', data: {} },
      { id: 'node-1', type: 'click', data: { selector: 'button' } },
      { id: 'end-1', type: 'end', data: {} },
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
      { id: 'node-1', type: 'click', data: {} },
      { id: 'node-2', type: 'input', data: {} },
      { id: 'node-3', type: 'click', data: {} },
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
