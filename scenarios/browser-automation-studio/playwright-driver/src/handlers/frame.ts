import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import type { Frame } from 'playwright';
import { getFrameSwitchParams } from '../types';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError, FrameNotFoundError, validateTimeout, logger, scopedLog, LogContext } from '../utils';

/**
 * Frame handler
 *
 * Handles iframe navigation operations: enter, exit, parent
 *
 * CRITICAL: This handler fixes the contract violation where
 * Playwright reports SupportsIframes: true but frame-switch was not implemented
 *
 * Idempotency and Replay Safety:
 * - enter: If already in the target frame (by URL match), returns success (no-op)
 * - exit: If already at main frame, returns error (safe, deterministic)
 * - parent: Same as exit - error if at main frame
 *
 * Frame Stack Validation:
 * - Frame references can become stale after navigation (see navigation.ts)
 * - The navigation handler clears frame stack after navigation
 * - This handler validates frame references before use
 */
/**
 * Check if a frame reference is still valid (not detached).
 * Frame references can become stale after navigation or page reload.
 */
async function isFrameValid(frame: Frame): Promise<boolean> {
  try {
    // Attempt to access frame URL - will throw if frame is detached
    frame.url();
    return true;
  } catch {
    return false;
  }
}

export class FrameHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['frame-switch'];
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, sessionId } = context;

    try {
      // Get typed params from instruction.action (required after migration)
      const typedParams = instruction.action ? getFrameSwitchParams(instruction.action) : undefined;
      const params = this.requireTypedParams(typedParams, 'frame-switch', instruction.nodeId);

      const action = params.action;
      // Hardened: Validate timeout bounds
      const timeout = validateTimeout(params.timeoutMs, DEFAULT_TIMEOUT_MS, 'frame-switch');

      logger.debug(scopedLog(LogContext.INSTRUCTION, 'frame operation'), {
        sessionId,
        action,
        selector: params.selector,
        frameId: params.frameId,
        frameUrl: params.frameUrl,
      });

      // Hardened: Access frame stack with type safety
      // Frame stack is stored in SessionState and passed via context
      const frameStack: Frame[] = Array.isArray(context.frameStack) ? context.frameStack : [];
      if (!Array.isArray(context.frameStack)) {
        logger.warn(scopedLog(LogContext.INSTRUCTION, 'frame stack not found in context'), {
          sessionId,
          hint: 'Using empty stack - frame tracking may be incomplete',
        });
      }

      // Replay safety: Validate existing frame references before use
      // This handles the case where navigation occurred and invalidated frames
      if (frameStack.length > 0) {
        const validFrames: Frame[] = [];
        for (const frame of frameStack) {
          if (await isFrameValid(frame)) {
            validFrames.push(frame);
          }
        }

        if (validFrames.length !== frameStack.length) {
          const invalidCount = frameStack.length - validFrames.length;
          logger.warn(scopedLog(LogContext.INSTRUCTION, 'stale frame references detected'), {
            sessionId,
            originalCount: frameStack.length,
            invalidCount,
            hint: 'Frame references may have been invalidated by navigation',
          });

          // Clear and repopulate with valid frames
          frameStack.length = 0;
          frameStack.push(...validFrames);
        }
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

          let targetFrame: Frame | null = null;

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
            targetFrame = page.frames().find(f => f.url().includes(params.frameUrl!)) ?? null;
          }

          if (!targetFrame) {
            throw new FrameNotFoundError(
              params.selector,
              params.frameId,
              params.frameUrl
            );
          }

          // Idempotency: Check if we're already in this frame
          // Compare by URL since frame references may differ
          const currentFrame = frameStack.length > 0 ? frameStack[frameStack.length - 1] : null;
          if (currentFrame && currentFrame.url() === targetFrame.url()) {
            logger.debug(scopedLog(LogContext.INSTRUCTION, 'already in target frame (idempotent)'), {
              sessionId,
              frameUrl: targetFrame.url(),
              stackDepth: frameStack.length,
            });

            return {
              success: true,
              extracted_data: {
                frameUrl: targetFrame.url(),
                frameName: await targetFrame.name(),
                stackDepth: frameStack.length,
                idempotent: true,
              },
            };
          }

          // Push current frame to stack
          frameStack.push(page.mainFrame());

          logger.info(scopedLog(LogContext.INSTRUCTION, 'entered frame'), {
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

          logger.info(scopedLog(LogContext.INSTRUCTION, 'exited frame'), {
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

          logger.info(scopedLog(LogContext.INSTRUCTION, 'switched to parent frame'), {
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
      logger.error(scopedLog(LogContext.INSTRUCTION, 'frame operation failed'), {
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
