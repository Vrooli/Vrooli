import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getSelectParams } from '../types';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError } from '../utils';
import { captureElementContext } from '../telemetry';

/**
 * Select handler
 *
 * Handles dropdown selection operations: select by value, label, or index
 */
export class SelectHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['select'];
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      // Get typed params from instruction.action (required after migration)
      const typedParams = instruction.action ? getSelectParams(instruction.action) : undefined;
      const params = this.requireTypedParams(typedParams, 'select', instruction.nodeId);

      if (!params.selector) {
        return {
          success: false,
          error: {
            message: 'select instruction missing selector parameter',
            code: 'MISSING_PARAM',
            kind: 'orchestration',
            retryable: false,
          },
        };
      }

      if (!params.value && !params.label && params.index === undefined) {
        return {
          success: false,
          error: {
            message: 'select instruction requires value, label, or index parameter',
            code: 'MISSING_PARAM',
            kind: 'orchestration',
            retryable: false,
          },
        };
      }

      const timeout = params.timeoutMs || DEFAULT_TIMEOUT_MS;

      logger.debug('Selecting option', {
        selector: params.selector,
        value: params.value,
        label: params.label,
        index: params.index,
        timeout,
      });

      // Wait for select element
      await page.waitForSelector(params.selector, { timeout });

      // Capture element context BEFORE the action (recording-quality telemetry)
      const elementContext = await captureElementContext(page, params.selector, { timeout });

      let selectedValue: string | string[] | null = null;

      // Select by value, label, or index
      if (params.value !== undefined) {
        // Select by value (string or string array)
        const value = Array.isArray(params.value) ? params.value : [params.value];
        selectedValue = await page.selectOption(params.selector, value, { timeout });
        logger.info('Selected by value', {
          selector: params.selector,
          value: params.value,
          selectedValue,
        });
      } else if (params.label !== undefined) {
        // Select by label (string or use first from array)
        const label = Array.isArray(params.label) ? params.label[0] : params.label;
        selectedValue = await page.selectOption(params.selector, { label }, { timeout });
        logger.info('Selected by label', {
          selector: params.selector,
          label: params.label,
          selectedValue,
        });
      } else if (params.index !== undefined) {
        // Select by index (number or number array)
        const index = Array.isArray(params.index) ? params.index[0] : params.index;
        selectedValue = await page.selectOption(params.selector, { index }, { timeout });
        logger.info('Selected by index', {
          selector: params.selector,
          index: params.index,
          selectedValue,
        });
      }

      return {
        success: true,
        elementContext,
        focus: {
          selector: elementContext.selector,
          bounding_box: elementContext.boundingBox ? {
            x: elementContext.boundingBox.x,
            y: elementContext.boundingBox.y,
            width: elementContext.boundingBox.width,
            height: elementContext.boundingBox.height,
          } : undefined,
        },
        extracted_data: {
          selectedValue,
        },
      };
    } catch (error) {
      logger.error('Select failed', {
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
