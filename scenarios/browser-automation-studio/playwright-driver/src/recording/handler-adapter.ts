/**
 * Handler Adapter for Replay Execution
 *
 * Bridges the action-executor replay system to the handler infrastructure.
 * This eliminates duplicate action implementations by delegating replay
 * execution to the canonical handlers.
 *
 * ARCHITECTURE:
 *   TimelineEntry  ──▶  HandlerInstruction  ──▶  Handler  ──▶  HandlerResult
 *        │                                                           │
 *        └───────────────▶  ActionReplayResult  ◀────────────────────┘
 *
 * WHY THIS EXISTS:
 *   - Single source of truth for action execution (handlers)
 *   - Enables all handler features in replay (tabs, frames, network mocking, etc.)
 *   - Eliminates code duplication between handlers and action-executor
 *   - Previously "unsupported" actions now work through handlers
 */

import type { Page, BrowserContext } from 'playwright';
import type { TimelineEntry } from '../proto';
import type { HandlerInstruction } from '../proto';
import type { HandlerContext } from '../handlers/base';
import type { Config } from '../config';
import type { Metrics } from '../utils/metrics';
import { ActionType } from '../proto';
import { actionTypeToString, stringToActionType } from '../proto/action-type-utils';
import { handlerRegistry } from '../handlers/registry';
import { loadConfig } from '../config';
import { createNoOpLogger } from '../utils/logger';
import { createNoOpMetrics, type NoOpMetrics } from '../utils/metrics';

// =============================================================================
// Types
// =============================================================================

/**
 * Minimal context required for replay execution.
 * This is a subset of the full HandlerContext.
 */
export interface ReplayContext {
  page: Page;
  timeout: number;
  validateSelector?: (selector: string) => Promise<{ valid: boolean; matchCount: number; selector: string; error?: string }>;
}

/**
 * Result of replay execution via handler.
 *
 * This is a minimal result type for the handler adapter bridge.
 * It follows the BaseExecutionResult contract without additional fields.
 *
 * @see BaseExecutionResult - Base interface defining success/error contract
 * @see HandlerResult - Full handler execution variant in outcome/outcome-builder.ts
 * @see ActionReplayResult - Recording replay variant in recording/action-executor.ts
 */
export interface HandlerAdapterResult {
  success: boolean;
  durationMs: number;
  error?: {
    message: string;
    code: string;
    matchCount?: number;
    selector?: string;
  };
}

// =============================================================================
// Conversion Functions
// =============================================================================

/**
 * Convert TimelineEntry to HandlerInstruction.
 *
 * TimelineEntry is the recorded action format, HandlerInstruction is what
 * handlers expect. The key field is `action` which contains typed params.
 */
export function timelineEntryToHandlerInstruction(entry: TimelineEntry): HandlerInstruction {
  const actionType = entry.action?.type ?? ActionType.UNSPECIFIED;

  return {
    index: entry.sequenceNum,
    nodeId: entry.id,
    type: actionTypeToString(actionType),
    params: {}, // Legacy field, handlers use action.params
    action: entry.action,
  };
}

// NOTE: actionTypeToString is now imported from ../proto/action-type-utils (single source of truth)

// =============================================================================
// Context Creation
// =============================================================================

// Cached minimal config and context creators
let cachedConfig: Config | undefined;
let cachedLogger: ReturnType<typeof createNoOpLogger> | undefined;
let cachedMetrics: NoOpMetrics | undefined;

/**
 * Create a minimal HandlerContext suitable for replay execution.
 *
 * Replay doesn't need full telemetry, metrics, or logging - we create
 * lightweight versions to satisfy the HandlerContext interface.
 */
export function createReplayHandlerContext(
  replayContext: ReplayContext,
  sessionId: string = 'replay-session'
): HandlerContext {
  // Lazy-init cached objects
  if (!cachedConfig) {
    cachedConfig = loadConfig();
    // Override timeout from replay context
    cachedConfig.execution.defaultTimeoutMs = replayContext.timeout;
    cachedConfig.execution.navigationTimeoutMs = replayContext.timeout;
    cachedConfig.execution.waitTimeoutMs = replayContext.timeout;
  }

  if (!cachedLogger) {
    cachedLogger = createNoOpLogger();
  }

  if (!cachedMetrics) {
    cachedMetrics = createNoOpMetrics();
  }

  // Get browser context from page
  const browserContext: BrowserContext = replayContext.page.context();

  return {
    page: replayContext.page,
    context: browserContext,
    config: cachedConfig,
    logger: cachedLogger,
    // Cast to Metrics - NoOpMetrics implements the same interface
    metrics: cachedMetrics as unknown as Metrics,
    sessionId,
  };
}

// =============================================================================
// Handler Execution
// =============================================================================

/**
 * Execute a TimelineEntry via the handler infrastructure.
 *
 * This is the main entry point for replay execution. It:
 * 1. Converts TimelineEntry to HandlerInstruction
 * 2. Looks up the appropriate handler
 * 3. Creates replay context
 * 4. Executes and returns result
 *
 * @returns HandlerAdapterResult with success/error info
 */
export async function executeViaHandler(
  entry: TimelineEntry,
  replayContext: ReplayContext,
  sessionId?: string
): Promise<HandlerAdapterResult> {
  const startTime = Date.now();
  const instruction = timelineEntryToHandlerInstruction(entry);

  try {
    // Check if handler exists for this type
    if (!handlerRegistry.isSupported(instruction.type)) {
      return {
        success: false,
        durationMs: Date.now() - startTime,
        error: {
          message: `No handler registered for action type: ${instruction.type}`,
          code: 'UNSUPPORTED_ACTION',
        },
      };
    }

    // Get handler and create context
    const handler = handlerRegistry.getHandler(instruction);
    const handlerContext = createReplayHandlerContext(replayContext, sessionId);

    // Execute via handler
    const result = await handler.execute(instruction, handlerContext);

    return {
      success: result.success,
      durationMs: Date.now() - startTime,
      error: result.error ? {
        message: result.error.message,
        code: result.error.code || 'UNKNOWN',
      } : undefined,
    };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);

    // Classify common errors
    let code = 'UNKNOWN';
    if (message.includes('waiting for selector') || message.includes('Timeout')) {
      code = 'TIMEOUT';
    } else if (message.includes('not visible')) {
      code = 'ELEMENT_NOT_VISIBLE';
    } else if (message.includes('not enabled') || message.includes('disabled')) {
      code = 'ELEMENT_NOT_ENABLED';
    } else if (message.includes('UnsupportedInstructionError')) {
      code = 'UNSUPPORTED_ACTION';
    }

    return {
      success: false,
      durationMs: Date.now() - startTime,
      error: { message, code },
    };
  }
}

/**
 * Check if a handler is available for an action type.
 */
export function hasHandlerForActionType(actionType: ActionType): boolean {
  const typeString = actionTypeToString(actionType);
  return handlerRegistry.isSupported(typeString);
}

/**
 * Get all action types that have handlers registered.
 */
export function getHandlerSupportedActionTypes(): ActionType[] {
  const supportedTypes = handlerRegistry.getSupportedTypes();
  const actionTypes: ActionType[] = [];

  // Map string types back to ActionType enum
  for (const typeString of supportedTypes) {
    const actionType = stringToActionType(typeString);
    if (actionType !== ActionType.UNSPECIFIED && !actionTypes.includes(actionType)) {
      actionTypes.push(actionType);
    }
  }

  return actionTypes;
}

// NOTE: stringToActionType is now imported from ../proto/action-type-utils (single source of truth)
