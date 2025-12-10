import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import type { Frame } from 'playwright';
import { FrameSwitchParamsSchema } from '../types/instruction';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError, FrameNotFoundError, validateTimeout, validateParams } from '../utils';

/**
 * Frame handler
 *
 * Handles iframe navigation operations: enter, exit, parent
 *
 * CRITICAL: This handler fixes the contract violation where
 * Playwright reports SupportsIframes: true but frame-switch was not implemented
 */
export class FrameHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['frame-switch'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger, sessionId } = context;

    try {
      // Hardened: Validate params object exists
      const rawParams = validateParams(instruction.params, 'frame-switch');
      const params = FrameSwitchParamsSchema.parse(rawParams);

      const action = params.action;
      // Hardened: Validate timeout bounds
      const timeout = validateTimeout(params.timeoutMs, DEFAULT_TIMEOUT_MS, 'frame-switch');

      logger.debug('Frame operation', {
        sessionId,
        action,
        selector: params.selector,
        frameId: params.frameId,
      });

      // Hardened: Access frame stack with type safety
      // Frame stack is stored in SessionState and passed via context
      const frameStack: Frame[] = Array.isArray(context.frameStack) ? context.frameStack : [];
      if (!Array.isArray(context.frameStack)) {
        logger.warn('Frame stack not found in context, using empty stack', { sessionId });
      }

      switch (action) {
        case 'enter': {
          // Enter iframe
          if (!params.selector && !params.frameId && !params.frameUrl) {
            return {
              success: false,
              error: {
                message: 'frame-switch enter requires selector, frameId, or frameUrl',
                code: 'MISSING_PARAM',
                kind: 'orchestration',
                retryable: false,
              },
            };
          }

          let targetFrame = null;

          if (params.selector) {
            // Find frame by selector
            // Wait for frame to be available
            await page.locator(params.selector).first().waitFor({ timeout });

            // Get the content frame
            const frames = page.frames();
            for (const frame of frames) {
              const frameElement = await frame.frameElement().catch(() => null);
              if (frameElement) {
                const matches = await page.locator(params.selector).first().evaluate(
                  (el, fe) => el === fe,
                  frameElement
                );
                if (matches) {
                  targetFrame = frame;
                  break;
                }
              }
            }
          } else if (params.frameId) {
            // Find frame by ID
            const frames = page.frames();
            for (const frame of frames) {
              const frameElement = await frame.frameElement().catch(() => null);
              if (frameElement) {
                const id = await frameElement.getAttribute('id');
                if (id === params.frameId) {
                  targetFrame = frame;
                  break;
                }
              }
            }
          } else if (params.frameUrl) {
            // Find frame by URL (partial match)
            targetFrame = page.frames().find(f => f.url().includes(params.frameUrl!));
          }

          if (!targetFrame) {
            throw new FrameNotFoundError(
              `Frame not found: ${params.selector || params.frameId || params.frameUrl}`,
              'FRAME_NOT_FOUND'
            );
          }

          // Push current frame to stack
          frameStack.push(page.mainFrame());

          logger.info('Entered frame', {
            sessionId,
            frameUrl: targetFrame.url(),
            stackDepth: frameStack.length,
          });

          // Note: The session manager should handle updating the frame stack
          // For now, we return the frame info
          return {
            success: true,
            extracted_data: {
              frameUrl: targetFrame.url(),
              frameName: await targetFrame.name(),
              stackDepth: frameStack.length,
            },
          };
        }

        case 'exit': {
          // Exit to parent frame
          if (frameStack.length === 0) {
            return {
              success: false,
              error: {
                message: 'Cannot exit frame: already at main frame',
                code: 'NOT_IN_FRAME',
                kind: 'orchestration',
                retryable: false,
              },
            };
          }

          // Pop from frame stack
          frameStack.pop();

          logger.info('Exited frame', {
            sessionId,
            stackDepth: frameStack.length,
          });

          return {
            success: true,
            extracted_data: {
              stackDepth: frameStack.length,
            },
          };
        }

        case 'parent': {
          // Same as exit - go to parent frame
          if (frameStack.length === 0) {
            return {
              success: false,
              error: {
                message: 'Cannot go to parent: already at main frame',
                code: 'NOT_IN_FRAME',
                kind: 'orchestration',
                retryable: false,
              },
            };
          }

          frameStack.pop();

          logger.info('Switched to parent frame', {
            sessionId,
            stackDepth: frameStack.length,
          });

          return {
            success: true,
            extracted_data: {
              stackDepth: frameStack.length,
            },
          };
        }

        default:
          return {
            success: false,
            error: {
              message: `Unknown frame action: ${action}`,
              code: 'INVALID_ACTION',
              kind: 'orchestration',
              retryable: false,
            },
          };
      }
    } catch (error) {
      logger.error('Frame operation failed', {
        sessionId,
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
}
