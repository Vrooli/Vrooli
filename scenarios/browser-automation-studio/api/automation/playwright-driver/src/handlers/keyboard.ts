import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { KeyboardParamsSchema, ShortcutParamsSchema } from '../types/instruction';
import { normalizeError } from '../utils';

/**
 * Keyboard handler
 *
 * Handles keyboard operations: press, down, up, and shortcuts
 */
export class KeyboardHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['keyboard', 'shortcut'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { logger } = context;

    try {
      switch (instruction.type.toLowerCase()) {
        case 'keyboard':
          return await this.handleKeyboard(instruction, context);

        case 'shortcut':
          return await this.handleShortcut(instruction, context);

        default:
          return {
            success: false,
            error: {
              message: `Unsupported keyboard type: ${instruction.type}`,
              code: 'UNSUPPORTED_TYPE',
              kind: 'orchestration',
              retryable: false,
            },
          };
      }
    } catch (error) {
      logger.error('Keyboard operation failed', {
        type: instruction.type,
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

  private async handleKeyboard(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Validate parameters
    const params = KeyboardParamsSchema.parse(instruction.params);

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

    // Handle each key
    for (const key of keys) {
      switch (action) {
        case 'press':
          // Press and release
          await page.keyboard.press(key);
          break;

        case 'down':
          // Press down only
          await page.keyboard.down(key);
          break;

        case 'up':
          // Release only
          await page.keyboard.up(key);
          break;
      }
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
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Validate parameters
    const params = ShortcutParamsSchema.parse(instruction.params);

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
