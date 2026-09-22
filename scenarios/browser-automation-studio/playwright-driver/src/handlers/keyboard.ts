import type { Page } from 'rebrowser-playwright';
import { BaseHandler, getDocument, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getKeyboardParams, getShortcutParams } from '../types';
import { normalizeError } from '../utils';
import { getActionType } from '../proto';

/**
 * Keyboard handler
 *
 * Handles keyboard operations: press, down, up, and shortcuts
 */
export class KeyboardHandler extends BaseHandler {
  // Persistent key-down actions belong to their page and survive temporary chords.
  private readonly heldKeys = new WeakMap<Page, Set<string>>();

  getSupportedTypes(): string[] {
    return ['keyboard', 'shortcut'];
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { logger } = context;

    try {

		const actionType = getActionType(instruction);
		switch (actionType.toLowerCase()) {
        case 'keyboard':
          return await this.handleKeyboard(instruction, context);

        case 'shortcut':
          return await this.handleShortcut(instruction, context);

        default:
          return {
            success: false,
            error: {
				message: `Unsupported keyboard type: ${actionType}`,
              code: 'UNSUPPORTED_TYPE',
              kind: 'orchestration',
              retryable: false,
            },
          };
      }
    } catch (error) {
      logger.error('Keyboard operation failed', {
			type: getActionType(instruction),
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

  private async focusDocument(context: HandlerContext): Promise<void> {
    const target = getDocument(context);
    if ('frameElement' in target && !(await target.evaluate(() => document.hasFocus()))) {
      const element = await target.frameElement();
      try { await element.focus(); }
      finally { await element.dispose(); }
    }
    // A focused descendant iframe must not receive keys for its selected parent.
    await target.evaluate(() => {
      const active = document.activeElement;
      if (active instanceof HTMLIFrameElement || active instanceof HTMLFrameElement) active.blur();
    });
  }

  private async handleKeyboard(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Get typed params from instruction.action (required after migration)
    const typedParams = instruction.action ? getKeyboardParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'keyboard', instruction.nodeId);

    const action = params.action || 'press';
    const keys = params.keys || (params.key ? [params.key] : []);

    if (keys.length === 0) {
      return {
        success: false,
        error: {
          message: 'keyboard instruction missing key/keys parameter',
          code: 'MISSING_PARAM',
          kind: 'orchestration',
          retryable: false,
        },
      };
    }

    logger.debug('Keyboard operation', {
      action,
      keys,
      modifiers: params.modifiers,
    });

    await this.focusDocument(context);
    const held = this.heldKeys.get(page) ?? new Set<string>();
    this.heldKeys.set(page, held);
    const acquired: string[] = [];
    try {
      for (const modifier of params.modifiers ?? []) {
        if (held.has(modifier) || acquired.includes(modifier)) continue;
        // Track before awaiting: even a rejected transport can have pressed it.
        acquired.push(modifier);
        await page.keyboard.down(modifier);
      }
      for (const key of keys) {
        switch (action) {
          case 'press':
            await page.keyboard.press(key);
            break;
          case 'down':
            await page.keyboard.down(key);
            held.add(key);
            break;
          case 'up':
            await page.keyboard.up(key);
            held.delete(key);
            break;
          default:
            throw new Error(`Unsupported keyboard action: ${action}`);
        }
      }
    } finally {
      const releases = await Promise.allSettled(
        acquired
          .reverse()
          .filter((modifier) => !held.has(modifier))
          .map((modifier) => page.keyboard.up(modifier))
      );
      const failed = releases.find((result) => result.status === 'rejected');
      if (failed?.status === 'rejected') throw failed.reason;
    }

    logger.info('Keyboard operation successful', {
      action,
      keys,
    });

    return {
      success: true,
    };
  }

  private async handleShortcut(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Get typed params from instruction.action (required after migration)
    const typedParams = instruction.action ? getShortcutParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'shortcut', instruction.nodeId);

    if (!params.shortcut) {
      return {
        success: false,
        error: {
          message: 'shortcut instruction missing shortcut parameter',
          code: 'MISSING_PARAM',
          kind: 'orchestration',
          retryable: false,
        },
      };
    }

    const shortcut = params.shortcut;

    logger.debug('Executing keyboard shortcut', {
      shortcut,
    });

    await this.focusDocument(context);
    // Playwright handles shortcuts like "Control+A", "Meta+Shift+K", etc.
    await page.keyboard.press(shortcut);

    logger.info('Shortcut executed', {
      shortcut,
    });

    return {
      success: true,
    };
  }
}
