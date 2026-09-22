import { BaseHandler, getDocument, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import type { Frame } from 'rebrowser-playwright';
import { getFrameSwitchParams } from '../types';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import {
  normalizeError,
  FrameNotFoundError,
  InvalidInstructionError,
  validateTimeout,
} from '../utils';

/** Select documents by frame identity. Transport receipts own replay idempotency. */
export class FrameHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['frame-switch'];
  }

  async execute(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    try {
      const params = this.requireTypedParams(
        instruction.action ? getFrameSwitchParams(instruction.action) : undefined,
        'frame-switch',
        instruction.nodeId
      );
      const stack = context.frameStack;
      if (!stack)
        throw new InvalidInstructionError('Frame navigation requires session-owned frame state');
      if (params.action === 'exit' || params.action === 'parent') {
        if (!stack.length) {
          return {
            success: false,
            error: {
              message: 'Cannot leave frame: already at main frame',
              code: 'NOT_IN_FRAME',
              kind: 'orchestration',
              retryable: false,
            },
          };
        }
        // EXIT can recover from a detached selection; PARENT retains ancestry validation.
        if (params.action === 'exit') stack.length = 0;
        else {
          getDocument({ ...context, frameStack: stack.slice(0, -1) });
          stack.pop();
        }
      } else if (params.action === 'enter') {
        const timeout = validateTimeout(params.timeoutMs, DEFAULT_TIMEOUT_MS, 'frame-switch');
        const current = getDocument(context);
        let target: Frame | null = null;
        if (params.selector) {
          const locator = current.locator(params.selector);
          await locator.waitFor({ state: 'attached', timeout });
          const element = await locator.elementHandle();
          try {
            target = (await element?.contentFrame()) ?? null;
          } finally {
            await element?.dispose();
          }
        } else if (params.frameId || params.frameUrl) {
          const frame = stack.at(-1) ?? context.page.mainFrame();
          const matches: Frame[] = [];
          for (const child of [...(stack.length ? [frame] : []), ...frame.childFrames()]) {
            if (params.frameUrl && child.url().includes(params.frameUrl)) matches.push(child);
            else if (params.frameId) {
              const element = await child.frameElement();
              try {
                if ((await element.getAttribute('id')) === params.frameId) matches.push(child);
              } finally {
                await element.dispose();
              }
            }
          }
          if (matches.length > 1) throw new InvalidInstructionError('Frame target is ambiguous');
          target = matches[0] ?? null;
        } else {
          return this.missingParamError('frame-switch enter', 'selector, frameId, or frameUrl');
        }
        if (!target || target.isDetached()) {
          throw new FrameNotFoundError(params.selector, params.frameId, params.frameUrl);
        }
        if (target === stack.at(-1)) {
          return {
            success: true,
            extracted_data: {
              frameUrl: target.url(),
              frameName: target.name(),
              stackDepth: stack.length,
              idempotent: true,
            },
          };
        }
        if (
          target.page() !== context.page ||
          target.parentFrame() !== (stack.at(-1) ?? context.page.mainFrame())
        ) {
          throw new InvalidInstructionError(
            'Frame target does not belong to the selected document'
          );
        }
        stack.push(target);
      } else {
        throw new InvalidInstructionError(`Unknown frame action: ${params.action}`);
      }
      const selected = stack.at(-1) ?? context.page.mainFrame();
      return {
        success: true,
        extracted_data: {
          frameUrl: selected.url(),
          frameName: selected.name(),
          stackDepth: stack.length,
        },
      };
    } catch (error) {
      const normalized = normalizeError(error);
      return {
        success: false,
        error: {
          message: normalized.message,
          code: normalized.code,
          kind: normalized.kind,
          retryable: normalized.retryable,
        },
      };
    }
  }
}
