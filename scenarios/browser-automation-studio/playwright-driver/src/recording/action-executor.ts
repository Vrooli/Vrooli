/**
 * Action Executor Registry - Handler-Delegated Implementation
 *
 * Executes TimelineEntry actions by delegating to the handler infrastructure.
 * This provides a single source of truth for action execution.
 *
 * PROTO-FIRST ARCHITECTURE:
 * - Accepts TimelineEntry directly (no RecordedAction conversion)
 * - Uses proto ActionType enum (not string literals)
 * - Returns proto-aligned ActionReplayResult
 *
 * HANDLER DELEGATION:
 * All action execution is delegated to the handler infrastructure via
 * handler-adapter.ts. This:
 * - Eliminates code duplication between handlers and executors
 * - Enables all handler features in replay (tabs, frames, network, etc.)
 * - Provides a single source of truth for action execution
 * - Makes adding new actions simpler (just implement the handler)
 *
 * EXTENSION POINT: Adding a new action type
 * 1. Implement handler in handlers/*.ts
 * 2. Register handler in server.ts:registerHandlers()
 * 3. The executor will automatically delegate to the handler
 */

import type { Page } from 'playwright';
import type { TimelineEntry } from '../proto';
import { ActionType } from '../proto';

// Import shared types from outcome module (single source of truth)
import { type ActionErrorCode, type BaseExecutionResult, type SelectorError } from '../outcome/types';

// Import handler adapter for delegating execution to handlers
import { executeViaHandler, hasHandlerForActionType, type ReplayContext } from './handler-adapter';

// Re-export types for consumers of this module
export type { ActionErrorCode, SelectorError };

// =============================================================================
// Types
// =============================================================================

/** Result of validating a selector */
export interface SelectorValidation {
  valid: boolean;
  matchCount: number;
  selector: string;
  error?: string;
}

/** Context provided to action executors */
export interface ExecutorContext {
  page: Page;
  timeout: number;
  validateSelector: (selector: string) => Promise<SelectorValidation>;
}

/**
 * Replay-specific error type alias.
 * Uses SelectorError from the canonical types module.
 *
 * SelectorError includes: message, code, kind, retryable, matchCount, selector
 *
 * @see SelectorError - Canonical type in outcome/types.ts
 */
export type ActionReplayError = SelectorError;

/**
 * Result of replaying a single action.
 *
 * Extends BaseExecutionResult with entry metadata (entryId, sequenceNum, actionType)
 * needed to correlate replay results with the original recorded timeline.
 *
 * @see BaseExecutionResult - Base interface defining success/error contract
 * @see HandlerResult - Handler execution variant in outcome/outcome-builder.ts
 * @see HandlerAdapterResult - Minimal adapter variant in recording/handler-adapter.ts
 */
export interface ActionReplayResult extends Omit<BaseExecutionResult, 'error' | 'durationMs'> {
  /** ID of the timeline entry being replayed */
  entryId: string;
  /** Sequence number in the timeline */
  sequenceNum: number;
  /** Type of action being replayed */
  actionType: ActionType;
  /** Execution duration in milliseconds (always tracked for replay) */
  durationMs: number;
  /** Replay-specific error with selector context */
  error?: ActionReplayError;
  /** Screenshot captured on error for debugging */
  screenshotOnError?: string;
}

/** Function signature for timeline entry executors */
export type TimelineExecutor = (
  entry: TimelineEntry,
  context: ExecutorContext,
) => Promise<ActionReplayResult>;

// =============================================================================
// Registry
// =============================================================================

const executorRegistry = new Map<ActionType, TimelineExecutor>();

/** Register an executor for an action type */
export function registerTimelineExecutor(actionType: ActionType, executor: TimelineExecutor): void {
  if (executorRegistry.has(actionType)) {
    console.warn(`[ActionExecutor] Overwriting executor for action type: ${ActionType[actionType]}`);
  }
  executorRegistry.set(actionType, executor);
}

/** Get executor for an action type */
export function getTimelineExecutor(actionType: ActionType): TimelineExecutor | undefined {
  return executorRegistry.get(actionType);
}

/** Check if an executor is registered */
export function hasTimelineExecutor(actionType: ActionType): boolean {
  return executorRegistry.has(actionType);
}

/** Get all registered action types */
export function getRegisteredActionTypes(): ActionType[] {
  return Array.from(executorRegistry.keys());
}

// =============================================================================
// Helpers
// =============================================================================

/** Create base result for an entry */
function createBaseResult(entry: TimelineEntry): ActionReplayResult {
  return {
    entryId: entry.id,
    sequenceNum: entry.sequenceNum,
    actionType: entry.action?.type ?? ActionType.UNSPECIFIED,
    success: false,
    durationMs: 0,
  };
}

/** Create unsupported action result */
function unsupportedResult(base: ActionReplayResult): ActionReplayResult {
  return {
    ...base,
    error: {
      message: `Unsupported action type for replay: ${ActionType[base.actionType]}`,
      code: 'UNSUPPORTED_ACTION',
    },
  };
}

// =============================================================================
// Handler-Delegated Executor Factory
// =============================================================================

/**
 * Create an executor that delegates to the handler infrastructure.
 *
 * This factory creates a TimelineExecutor that:
 * 1. Checks if a handler is available for the action type
 * 2. Delegates execution to the handler via handler-adapter.ts
 * 3. Converts the HandlerAdapterResult to ActionReplayResult
 *
 * @param actionType - The ActionType this executor handles
 * @returns A TimelineExecutor that delegates to handlers
 */
function createHandlerDelegatedExecutor(actionType: ActionType): TimelineExecutor {
  return async (entry: TimelineEntry, context: ExecutorContext): Promise<ActionReplayResult> => {
    const base = createBaseResult(entry);

    // Check if handler is available
    if (!hasHandlerForActionType(actionType)) {
      return unsupportedResult(base);
    }

    // Delegate to handler
    const replayContext: ReplayContext = {
      page: context.page,
      timeout: context.timeout,
      validateSelector: context.validateSelector,
    };

    const result = await executeViaHandler(entry, replayContext);

    return {
      ...base,
      success: result.success,
      durationMs: result.durationMs,
      error: result.error ? {
        message: result.error.message,
        code: result.error.code as ActionErrorCode,
        matchCount: result.error.matchCount,
        selector: result.error.selector,
      } : undefined,
    };
  };
}

// =============================================================================
// Register All Action Types with Handler Delegation
// =============================================================================

/**
 * All action types that should be handled via handler delegation.
 *
 * Previously, many of these had inline executor implementations that duplicated
 * the handler logic. Now all actions delegate to handlers, providing:
 * - Single source of truth for action execution
 * - Automatic support for new handler features in replay
 * - ~300 fewer lines of duplicate code
 */
const allHandlerDelegatedActions: ActionType[] = [
  // Core actions (previously had inline executors)
  ActionType.NAVIGATE,
  ActionType.CLICK,
  ActionType.INPUT,
  ActionType.SCROLL,
  ActionType.SELECT,
  ActionType.KEYBOARD,
  ActionType.FOCUS,
  ActionType.HOVER,
  ActionType.BLUR,
  ActionType.WAIT,
  ActionType.SCREENSHOT,
  ActionType.ASSERT,
  ActionType.EVALUATE,

  // Extended actions (always used handler delegation)
  ActionType.SUBFLOW,
  ActionType.EXTRACT,
  ActionType.UPLOAD_FILE,
  ActionType.DOWNLOAD,
  ActionType.FRAME_SWITCH,
  ActionType.TAB_SWITCH,
  ActionType.COOKIE_STORAGE,
  ActionType.SHORTCUT,
  ActionType.DRAG_DROP,
  ActionType.GESTURE,
  ActionType.NETWORK_MOCK,
  ActionType.ROTATE,
];

// Register handler-delegated executors for all action types
for (const actionType of allHandlerDelegatedActions) {
  registerTimelineExecutor(actionType, createHandlerDelegatedExecutor(actionType));
}

// =============================================================================
// Main Executor Function
// =============================================================================

/**
 * Execute a TimelineEntry action.
 *
 * This is the main entry point for action execution. It looks up the
 * appropriate executor and runs it, handling errors and timing.
 *
 * All executors now delegate to the handler infrastructure, which provides
 * the canonical implementation for each action type.
 */
export async function executeTimelineEntry(
  entry: TimelineEntry,
  context: ExecutorContext
): Promise<ActionReplayResult> {
  const actionType = entry.action?.type ?? ActionType.UNSPECIFIED;
  const base = createBaseResult(entry);
  const startTime = Date.now();

  try {
    const executor = getTimelineExecutor(actionType);

    if (!executor) {
      return {
        ...base,
        durationMs: Date.now() - startTime,
        error: {
          message: `No executor registered for action type: ${ActionType[actionType]}`,
          code: 'UNSUPPORTED_ACTION',
        },
      };
    }

    const result = await executor(entry, context);
    result.durationMs = Date.now() - startTime;
    return result;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);

    // Classify common Playwright errors
    let code: ActionErrorCode = 'UNKNOWN';
    if (message.includes('waiting for selector') || message.includes('Timeout')) {
      code = 'TIMEOUT';
    } else if (message.includes('not visible')) {
      code = 'ELEMENT_NOT_VISIBLE';
    } else if (message.includes('not enabled') || message.includes('disabled')) {
      code = 'ELEMENT_NOT_ENABLED';
    }

    return {
      ...base,
      durationMs: Date.now() - startTime,
      error: { message, code },
    };
  }
}
