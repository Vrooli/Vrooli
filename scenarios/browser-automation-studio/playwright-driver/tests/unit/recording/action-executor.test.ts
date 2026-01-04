/**
 * Tests for Action Executor (Proto-Native)
 *
 * These tests verify replay execution against TimelineEntry + typed action params.
 */

import type { Page } from 'playwright';
import { create } from '@bufbuild/protobuf';
import {
  ActionDefinitionSchema,
  ActionType,
  ClickParamsSchema,
  MouseButton,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { TimelineEntrySchema, type TimelineEntry } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';

// Mock handler-adapter before importing action-executor
jest.mock('../../../src/recording/handler-adapter', () => {
  const originalModule = jest.requireActual('../../../src/recording/handler-adapter');
  return {
    ...originalModule,
    hasHandlerForActionType: jest.fn().mockReturnValue(true),
    executeViaHandler: jest.fn(),
  };
});

import { hasTimelineExecutor, getTimelineExecutor, getRegisteredActionTypes } from '../../../src/recording/action-executor';
import type { ExecutorContext, SelectorValidation } from '../../../src/recording/action-executor';
import { executeViaHandler } from '../../../src/recording/handler-adapter';

const createMockPage = (): jest.Mocked<Page> =>
  ({
    click: jest.fn().mockResolvedValue(undefined),
  }) as unknown as jest.Mocked<Page>;

function createBaseEntry(overrides: Partial<TimelineEntry> = {}): TimelineEntry {
  return create(TimelineEntrySchema, {
    id: 'entry-1',
    sequenceNum: 0,
    action: create(ActionDefinitionSchema, {
      type: ActionType.CLICK,
      params: {
        case: 'click',
        value: create(ClickParamsSchema, {
          selector: '[data-testid="test-button"]',
          button: MouseButton.LEFT,
          clickCount: 1,
        }),
      },
    }),
    ...overrides,
  });
}

function createContext(
  page: Page,
  validationResult: SelectorValidation = { valid: true, matchCount: 1, selector: '[data-testid="test-button"]' }
): ExecutorContext {
  return {
    page,
    timeout: 5000,
    validateSelector: jest.fn().mockResolvedValue(validationResult),
  };
}

describe('Action Executor Registry', () => {
  it('registers standard action executors', () => {
    expect(hasTimelineExecutor(ActionType.CLICK)).toBe(true);
    expect(hasTimelineExecutor(ActionType.INPUT)).toBe(true);
    expect(hasTimelineExecutor(ActionType.NAVIGATE)).toBe(true);
  });

  it('returns registered action types', () => {
    const types = getRegisteredActionTypes();
    expect(types).toContain(ActionType.CLICK);
  });
});

describe('Click Executor', () => {
  const mockExecuteViaHandler = executeViaHandler as jest.MockedFunction<typeof executeViaHandler>;

  beforeEach(() => {
    mockExecuteViaHandler.mockReset();
  });

  it('executes click successfully', async () => {
    const page = createMockPage();
    const entry = createBaseEntry();
    const context = createContext(page);

    // Mock successful execution
    mockExecuteViaHandler.mockResolvedValue({
      success: true,
      durationMs: 100,
    });

    const executor = getTimelineExecutor(ActionType.CLICK);
    expect(executor).toBeDefined();

    const result = await executor!(entry, context);
    expect(result.success).toBe(true);
    expect(mockExecuteViaHandler).toHaveBeenCalledWith(entry, expect.objectContaining({
      page,
      timeout: 5000,
    }));
  });

  it('returns MISSING_PARAMS when click params are missing', async () => {
    const page = createMockPage();
    const entry = createBaseEntry({
      action: create(ActionDefinitionSchema, { type: ActionType.CLICK }),
    });
    const context = createContext(page);

    // Mock error response for missing params
    mockExecuteViaHandler.mockResolvedValue({
      success: false,
      durationMs: 10,
      error: {
        message: 'Missing required click params',
        code: 'MISSING_PARAMS',
      },
    });

    const executor = getTimelineExecutor(ActionType.CLICK)!;
    const result = await executor(entry, context);
    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('MISSING_PARAMS');
  });

  it('returns SELECTOR_NOT_FOUND when selector validation fails', async () => {
    const page = createMockPage();
    const entry = createBaseEntry();
    const context = createContext(page, { valid: false, matchCount: 0, selector: '[data-testid="test-button"]' });

    // Mock error response for selector not found
    mockExecuteViaHandler.mockResolvedValue({
      success: false,
      durationMs: 10,
      error: {
        message: 'Selector not found',
        code: 'SELECTOR_NOT_FOUND',
        matchCount: 0,
        selector: '[data-testid="test-button"]',
      },
    });

    const executor = getTimelineExecutor(ActionType.CLICK)!;
    const result = await executor(entry, context);
    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('SELECTOR_NOT_FOUND');
    expect(result.error?.matchCount).toBe(0);
  });
});

