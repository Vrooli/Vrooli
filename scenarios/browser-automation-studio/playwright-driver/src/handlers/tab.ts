import { Page } from 'playwright';
import { BaseHandler, HandlerContext, HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getTabSwitchParams } from '../proto';
import { normalizeError } from '../utils/errors';
import { logger, scopedLog, LogContext } from '../utils';

/** Internal params type returned by getTabSwitchParams */
interface TabSwitchParams {
  action: string;
  url?: string;
  index?: number;
  title?: string;
  urlPattern?: string;
}

/**
 * Track pending tab operations to prevent duplicate concurrent operations.
 * Key: composite of sessionId + operation + unique identifier (URL/index)
 * Value: Promise that resolves to the handler result
 *
 * Idempotency guarantee:
 * - Concurrent open requests for the same URL will await the first
 * - Concurrent switch requests to the same target will await the first
 * - Prevents duplicate tabs from retries or concurrent calls
 */
const pendingTabOperations: Map<string, Promise<HandlerResult>> = new Map();

/**
 * Generate a stable key for tab operation idempotency.
 */
function generateTabOperationKey(
  sessionId: string,
  action: string,
  identifier: string | number | undefined
): string {
  return `${sessionId}:${action}:${identifier ?? 'default'}`;
}

/**
 * TabHandler implements multi-tab/window management
 *
 * Supported instruction types:
 * - tab-switch: Manage multiple browser tabs/windows
 *
 * Operations:
 * - open: Open new tab with optional URL
 * - switch: Switch to tab by index, URL, or title
 * - close: Close tab by index
 * - list: List all open tabs
 *
 * State Management:
 * - Maintains tab stack in session.tabStack
 * - Current page is always session.page
 * - Tab indices are 0-based
 *
 * Idempotency behavior:
 * - open: Concurrent opens with same URL await the first (prevents duplicate tabs)
 * - switch: Safe to replay - switching to already-current tab is a no-op
 * - close: Closing an already-closed tab index returns error (safe)
 * - list: Read-only operation, always safe
 *
 * Phase 3 handler - Multi-tab support
 */
export class TabHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['tab-switch', 'tab', 'tabs'];
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { logger } = context;

    try {
      // Get typed params from instruction.action (required after migration)
      const typedParams = instruction.action ? getTabSwitchParams(instruction.action) : undefined;
      const validated = this.requireTypedParams(typedParams, 'tab-switch', instruction.nodeId);

      logger.debug('Executing tab operation', {
        action: validated.action,
        index: validated.index,
        url: validated.url,
        title: validated.title,
      });

      switch (validated.action) {
        case 'open':
          return this.handleOpenTab(validated, context, logger);
        case 'switch':
          return this.handleSwitchTab(validated, context, logger);
        case 'close':
          return this.handleCloseTab(validated, context, logger);
        case 'list':
          return this.handleListTabs(context, logger);
        default:
          return {
            success: false,
            error: {
              message: `Unknown tab action: ${validated.action}`,
              code: 'INVALID_ACTION',
              kind: 'user',
              retryable: false,
            },
          };
      }
    } catch (error) {
      logger.error('Tab operation failed', {
        error: error instanceof Error ? error.message : String(error),
      });

      const driverError = normalizeError(error);

      return {
        success: false,
        error: {
          message: driverError.message,
          code: driverError.code,
          kind: driverError.kind,
          retryable: driverError.retryable,
        },
      };
    }
  }

  /**
   * Open a new tab
   *
   * Idempotency behavior:
   * - Concurrent opens with the same URL will await the first operation
   * - This prevents duplicate tabs when requests are retried concurrently
   */
  private async handleOpenTab(
    params: TabSwitchParams,
    context: HandlerContext,
    _logger: unknown
  ): Promise<HandlerResult> {
    const url = params.url || 'about:blank';
    const sessionId = context.sessionId;

    // Generate idempotency key for this open operation
    const operationKey = generateTabOperationKey(sessionId, 'open', url);

    // Idempotency: Check for in-flight open with same URL
    const pendingOperation = pendingTabOperations.get(operationKey);
    if (pendingOperation) {
      logger.debug(scopedLog(LogContext.INSTRUCTION, 'tab open already in progress, waiting'), {
        sessionId,
        url,
        hint: 'Concurrent tab open request detected; waiting for in-flight operation',
      });
      return pendingOperation;
    }

    // Create the operation promise and track it
    const operationPromise = this.executeOpenTab(params, context, url);
    pendingTabOperations.set(operationKey, operationPromise);

    try {
      return await operationPromise;
    } finally {
      // Clean up in-flight tracking
      pendingTabOperations.delete(operationKey);
    }
  }

  /**
   * Internal tab open execution logic.
   * Separated from handleOpenTab to enable idempotency tracking.
   */
  private async executeOpenTab(
    _params: TabSwitchParams,
    context: HandlerContext,
    url: string
  ): Promise<HandlerResult> {
    const browserContext = context.context;
    const sessionId = context.sessionId;

    logger.debug(scopedLog(LogContext.INSTRUCTION, 'opening new tab'), {
      sessionId,
      url,
    });

    // Create new page in the same browser context
    const newPage = await browserContext.newPage();

    // Initialize tab stack if needed
    if (!context.tabStack) {
      context.tabStack = [context.page];
    }

    // Add to tab stack
    context.tabStack.push(newPage);

    // Navigate if URL provided
    if (url !== 'about:blank') {
      await newPage.goto(url, {
        waitUntil: 'domcontentloaded',
        timeout: context.config.execution.navigationTimeoutMs,
      });
    }

    // Switch to new tab
    context.page = newPage;

    logger.info(scopedLog(LogContext.INSTRUCTION, 'new tab opened'), {
      sessionId,
      url,
      tabCount: context.tabStack.length,
      currentIndex: context.tabStack.length - 1,
    });

    return {
      success: true,
      extracted_data: {
        tab: {
          action: 'open',
          url: newPage.url(),
          title: await newPage.title(),
          index: context.tabStack.length - 1,
          totalTabs: context.tabStack.length,
        },
      },
    };
  }

  /**
   * Switch to an existing tab
   *
   * Idempotency behavior:
   * - Switching to the already-current tab is a no-op (returns success)
   * - This makes replay safe - repeated switch instructions don't cause errors
   */
  private async handleSwitchTab(
    params: TabSwitchParams,
    context: HandlerContext,
    _logger: unknown
  ): Promise<HandlerResult> {
    const sessionId = context.sessionId;

    if (!context.tabStack || context.tabStack.length === 0) {
      return {
        success: false,
        error: {
          message: 'No tabs available to switch',
          code: 'NO_TABS',
          kind: 'user',
          retryable: false,
        },
      };
    }

    let targetPage: Page | null = null;
    let targetIndex = -1;

    // Switch by index
    if (params.index !== undefined) {
      targetIndex = params.index;
      if (targetIndex < 0 || targetIndex >= context.tabStack.length) {
        return {
          success: false,
          error: {
            message: `Tab index out of range: ${targetIndex} (valid: 0-${context.tabStack.length - 1})`,
            code: 'INVALID_INDEX',
            kind: 'user',
            retryable: false,
          },
        };
      }
      targetPage = context.tabStack[targetIndex];
    }
    // Switch by title pattern
    else if (params.title) {
      const titlePattern = new RegExp(params.title, 'i');
      for (let i = 0; i < context.tabStack.length; i++) {
        const page = context.tabStack[i];
        const title = await page.title();
        if (titlePattern.test(title)) {
          targetPage = page;
          targetIndex = i;
          break;
        }
      }
      if (!targetPage) {
        return {
          success: false,
          error: {
            message: `No tab found with title matching: ${params.title}`,
            code: 'TAB_NOT_FOUND',
            kind: 'user',
            retryable: false,
          },
        };
      }
    }
    // Switch by URL pattern
    else if (params.urlPattern) {
      const urlPattern = new RegExp(params.urlPattern, 'i');
      for (let i = 0; i < context.tabStack.length; i++) {
        const page = context.tabStack[i];
        if (urlPattern.test(page.url())) {
          targetPage = page;
          targetIndex = i;
          break;
        }
      }
      if (!targetPage) {
        return {
          success: false,
          error: {
            message: `No tab found with URL matching: ${params.urlPattern}`,
            code: 'TAB_NOT_FOUND',
            kind: 'user',
            retryable: false,
          },
        };
      }
    } else {
      return {
        success: false,
        error: {
          message: 'Must provide index, title, or urlPattern to switch tabs',
          code: 'MISSING_PARAMS',
          kind: 'user',
          retryable: false,
        },
      };
    }

    // Safety check (should never happen due to logic above)
    if (!targetPage) {
      return {
        success: false,
        error: {
          message: 'Tab not found',
          code: 'TAB_NOT_FOUND',
          kind: 'user',
          retryable: false,
        },
      };
    }

    // Idempotency: If already on the target tab, this is a no-op
    // This makes replay safe - repeated switch instructions succeed
    const isAlreadyCurrent = context.page === targetPage;
    if (isAlreadyCurrent) {
      logger.debug(scopedLog(LogContext.INSTRUCTION, 'already on target tab (idempotent)'), {
        sessionId,
        index: targetIndex,
        url: targetPage.url(),
      });
    } else {
      // Bring target page to front
      await targetPage.bringToFront();

      // Update current page in session
      context.page = targetPage;
    }

    logger.info(scopedLog(LogContext.INSTRUCTION, 'switched to tab'), {
      sessionId,
      index: targetIndex,
      url: targetPage.url(),
      title: await targetPage.title(),
      wasNoOp: isAlreadyCurrent,
    });

    return {
      success: true,
      extracted_data: {
        tab: {
          action: 'switch',
          index: targetIndex,
          url: targetPage.url(),
          title: await targetPage.title(),
          totalTabs: context.tabStack.length,
          idempotent: isAlreadyCurrent || undefined,
        },
      },
    };
  }

  /**
   * Close a tab
   *
   * Note: Close is intentionally NOT idempotent - closing an already-closed
   * tab returns an error. This is the correct behavior because:
   * 1. Tab indices shift after close, so "close tab 2" means different things
   * 2. The caller should know if their close succeeded or not
   * 3. Silent no-ops could hide bugs in the automation flow
   */
  private async handleCloseTab(
    params: TabSwitchParams,
    context: HandlerContext,
    _logger: unknown
  ): Promise<HandlerResult> {
    const sessionId = context.sessionId;

    if (!context.tabStack || context.tabStack.length === 0) {
      return {
        success: false,
        error: {
          message: 'No tabs available to close',
          code: 'NO_TABS',
          kind: 'user',
          retryable: false,
        },
      };
    }

    if (context.tabStack.length === 1) {
      return {
        success: false,
        error: {
          message: 'Cannot close the last remaining tab',
          code: 'LAST_TAB',
          kind: 'user',
          retryable: false,
        },
      };
    }

    const index = params.index ?? context.tabStack.indexOf(context.page);
    if (index < 0 || index >= context.tabStack.length) {
      return {
        success: false,
        error: {
          message: `Tab index out of range: ${index} (valid: 0-${context.tabStack.length - 1})`,
          code: 'INVALID_INDEX',
          kind: 'user',
          retryable: false,
        },
      };
    }

    const pageToClose = context.tabStack[index];

    logger.debug(scopedLog(LogContext.INSTRUCTION, 'closing tab'), {
      sessionId,
      index,
      url: pageToClose.url(),
    });

    // Close the page
    await pageToClose.close();

    // Remove from tab stack
    context.tabStack.splice(index, 1);

    // If we closed the current page, switch to another tab
    if (context.page === pageToClose) {
      // Switch to the previous tab, or first tab if we closed index 0
      const newIndex = Math.min(index, context.tabStack.length - 1);
      context.page = context.tabStack[newIndex];
      await context.page.bringToFront();
    }

    logger.info(scopedLog(LogContext.INSTRUCTION, 'tab closed'), {
      sessionId,
      closedIndex: index,
      currentIndex: context.tabStack.indexOf(context.page),
      remainingTabs: context.tabStack.length,
    });

    return {
      success: true,
      extracted_data: {
        tab: {
          action: 'close',
          closedIndex: index,
          currentIndex: context.tabStack.indexOf(context.page),
          totalTabs: context.tabStack.length,
        },
      },
    };
  }

  /**
   * List all open tabs
   *
   * Read-only operation - inherently idempotent and safe to replay.
   */
  private async handleListTabs(context: HandlerContext, _logger: unknown): Promise<HandlerResult> {
    const sessionId = context.sessionId;

    if (!context.tabStack || context.tabStack.length === 0) {
      return {
        success: false,
        error: {
          message: 'No tabs available',
          code: 'NO_TABS',
          kind: 'user',
          retryable: false,
        },
      };
    }

    const tabs = await Promise.all(
      context.tabStack.map(async (page, index) => ({
        index,
        url: page.url(),
        title: await page.title(),
        isCurrent: page === context.page,
      }))
    );

    logger.info(scopedLog(LogContext.INSTRUCTION, 'listed tabs'), {
      sessionId,
      count: tabs.length,
      currentIndex: context.tabStack.indexOf(context.page),
    });

    return {
      success: true,
      extracted_data: {
        tabs,
        currentIndex: context.tabStack.indexOf(context.page),
        totalTabs: tabs.length,
      },
    };
  }
}
