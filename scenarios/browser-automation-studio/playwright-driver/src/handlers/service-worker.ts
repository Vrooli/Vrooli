/**
 * Service Worker Handler
 *
 * Instruction handler for service worker operations.
 * Follows the same pattern as cookie-storage.ts.
 *
 * Supported operations:
 * - list: Get all registered service workers
 * - unregister: Unregister a specific SW by scope URL
 * - unregister-all: Unregister all SWs
 * - stop-all: Stop all running SWs
 */

import { BaseHandler, type HandlerContext, type HandlerResult, type HandlerInstruction } from './base';
import { normalizeError } from '../utils';

/** Service worker operation params */
interface ServiceWorkerParams {
  operation: 'list' | 'unregister' | 'unregister-all' | 'stop-all';
  scopeURL?: string; // For 'unregister' operation
}

/**
 * Service Worker instruction handler.
 */
export class ServiceWorkerHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['service-worker'];
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { logger, sessionId } = context;

    try {
      // Extract params from instruction
      const params = this.extractParams(instruction);

      if (!params) {
        return this.missingParamError('service-worker', 'operation');
      }

      const operation = params.operation;

      logger.debug('Service worker operation', {
        sessionId,
        operation,
        scopeURL: params.scopeURL,
      });

      // Get SW controller from context
      // The controller is attached to the session by manager.ts
      const swController = (context as any).serviceWorkerController;

      if (!swController) {
        return {
          success: false,
          error: {
            message: 'Service worker controller not initialized',
            code: 'SW_NOT_INITIALIZED',
            kind: 'orchestration',
            retryable: false,
          },
        };
      }

      switch (operation) {
        case 'list': {
          const workers = swController.getWorkers();
          return {
            success: true,
            extracted_data: { workers },
          };
        }

        case 'unregister': {
          if (!params.scopeURL) {
            return this.missingParamError('service-worker', 'scopeURL');
          }
          const unregistered = await swController.unregister(params.scopeURL);
          return {
            success: true,
            extracted_data: { unregistered, scopeURL: params.scopeURL },
          };
        }

        case 'unregister-all': {
          const count = await swController.unregisterAll();
          logger.info('Service workers unregistered', { sessionId, count });
          return {
            success: true,
            extracted_data: { unregisteredCount: count },
          };
        }

        case 'stop-all': {
          await swController.stopAll();
          logger.info('Service workers stopped', { sessionId });
          return { success: true };
        }

        default:
          return {
            success: false,
            error: {
              message: `Unsupported service worker operation: ${operation}`,
              code: 'UNSUPPORTED_OPERATION',
              kind: 'orchestration',
              retryable: false,
            },
          };
      }
    } catch (error) {
      logger.error('Service worker operation failed', {
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

  /**
   * Extract params from instruction.
   * Since there's no proto type yet, we extract from raw params.
   */
  private extractParams(instruction: HandlerInstruction): ServiceWorkerParams | undefined {
    // Extract from action params if available
    const rawParams = instruction.action?.params as Record<string, unknown> | undefined;
    if (!rawParams || typeof rawParams !== 'object') {
      return undefined;
    }

    // Check for operation field
    const paramsValue = 'value' in rawParams ? rawParams.value : rawParams;
    if (typeof paramsValue !== 'object' || paramsValue === null) {
      return undefined;
    }

    const params = paramsValue as Record<string, unknown>;
    if (typeof params.operation !== 'string') {
      return undefined;
    }

    return {
      operation: params.operation as ServiceWorkerParams['operation'],
      scopeURL: typeof params.scopeURL === 'string' ? params.scopeURL : undefined,
    };
  }
}
