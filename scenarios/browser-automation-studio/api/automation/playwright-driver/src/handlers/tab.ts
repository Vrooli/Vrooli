import { Page } from 'playwright';
import { BaseHandler, HandlerContext, HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { TabSwitchParamsSchema, type TabSwitchParams } from '../types/instruction';
import { normalizeError } from '../utils/errors';

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
 * Phase 3 handler - Multi-tab support
 */
export class TabHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['tab-switch', 'tab', 'tabs'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      const validated = TabSwitchParamsSchema.parse(instruction.params);

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
   */
  private async handleOpenTab(
    params: TabSwitchParams,
    context: HandlerContext,
    logger: any
  ): Promise<HandlerResult> {
    const browserContext = context.context;
    const url = params.url || 'about:blank';

    logger.debug('Opening new tab', { url });

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
        timeout: 30000,
      });
    }

    // Switch to new tab
    context.page = newPage;

    logger.info('New tab opened', {
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
   */
  private async handleSwitchTab(
    params: TabSwitchParams,
    context: HandlerContext,
    logger: any
  ): Promise<HandlerResult> {
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

    // Bring target page to front
    await targetPage.bringToFront();

    // Update current page in session
    context.page = targetPage;

    logger.info('Switched to tab', {
      index: targetIndex,
      url: targetPage.url(),
      title: await targetPage.title(),
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
        },
      },
    };
  }

  /**
   * Close a tab
   */
  private async handleCloseTab(
    params: TabSwitchParams,
    context: HandlerContext,
    logger: any
  ): Promise<HandlerResult> {
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

    logger.debug('Closing tab', {
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

    logger.info('Tab closed', {
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
   */
  private async handleListTabs(context: HandlerContext, logger: any): Promise<HandlerResult> {
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

    logger.info('Listed tabs', {
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
