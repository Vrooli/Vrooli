import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { CookieStorageParamsSchema, type CookieStorageParams } from '../types/instruction';
import { normalizeError } from '../utils';

/**
 * Cookie and Storage handler
 *
 * Handles cookie, localStorage, and sessionStorage operations via a single
 * 'cookie-storage' instruction type with operation and storageType discriminators
 */
export class CookieStorageHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['cookie-storage'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { logger } = context;

    try {
      const params = CookieStorageParamsSchema.parse(instruction.params);

      const operation = params.operation;
      const storageType = params.storageType;

      logger.debug('Cookie/Storage operation', {
        operation,
        storageType,
        key: params.key,
        name: params.name,
      });

      // Dispatch based on storageType
      if (storageType === 'cookie') {
        return await this.handleCookieOperation(params, operation, context);
      } else {
        // localStorage or sessionStorage
        return await this.handleStorageOperation(params, operation, storageType, context);
      }
    } catch (error) {
      logger.error('Cookie/Storage operation failed', {
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
   * Handle cookie operations (get, set, delete, clear)
   */
  private async handleCookieOperation(
    params: CookieStorageParams,
    operation: string,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { context: browserContext, logger } = context;

    switch (operation) {
      case 'set':
        if (!params.name || params.value === undefined) {
          return {
            success: false,
            error: {
              message: 'cookie set requires name and value',
              code: 'MISSING_PARAM',
              kind: 'orchestration',
              retryable: false,
            },
          };
        }

        logger.debug('Setting cookie', { name: params.name });

        await browserContext.addCookies([
          {
            name: params.name,
            value: params.value,
            domain: params.cookieOptions?.domain,
            path: params.cookieOptions?.path || '/',
            expires: params.cookieOptions?.expires,
            httpOnly: params.cookieOptions?.httpOnly,
            secure: params.cookieOptions?.secure,
            sameSite: params.cookieOptions?.sameSite,
          },
        ]);

        logger.info('Cookie set', { name: params.name });
        return { success: true };

      case 'get':
        logger.debug('Getting cookies', { name: params.name });

        const cookies = await browserContext.cookies();

        if (params.name) {
          const cookie = cookies.find((c) => c.name === params.name);
          logger.info('Cookie retrieved', { name: params.name, found: !!cookie });
          return {
            success: true,
            extracted_data: { cookie: cookie?.value },
          };
        } else {
          const result = cookies.reduce((acc, cookie) => {
            acc[cookie.name] = cookie.value;
            return acc;
          }, {} as Record<string, string>);
          logger.info('All cookies retrieved', { count: cookies.length });
          return {
            success: true,
            extracted_data: { cookie: result },
          };
        }

      case 'delete':
        if (!params.name) {
          return {
            success: false,
            error: {
              message: 'cookie delete requires name',
              code: 'MISSING_PARAM',
              kind: 'orchestration',
              retryable: false,
            },
          };
        }

        logger.debug('Deleting cookie', { name: params.name });

        const allCookies = await browserContext.cookies();
        const remainingCookies = allCookies.filter((c) => c.name !== params.name);

        await browserContext.clearCookies();
        await browserContext.addCookies(remainingCookies);

        logger.info('Cookie deleted', { name: params.name });
        return { success: true };

      case 'clear':
        logger.debug('Clearing all cookies');
        await browserContext.clearCookies();
        logger.info('All cookies cleared');
        return { success: true };

      default:
        return {
          success: false,
          error: {
            message: `Unsupported cookie operation: ${operation}`,
            code: 'UNSUPPORTED_OPERATION',
            kind: 'orchestration',
            retryable: false,
          },
        };
    }
  }

  /**
   * Handle localStorage/sessionStorage operations (get, set, delete, clear)
   */
  private async handleStorageOperation(
    params: CookieStorageParams,
    operation: string,
    storageType: string,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    switch (operation) {
      case 'set':
        if (!params.key || params.value === undefined) {
          return {
            success: false,
            error: {
              message: 'storage set requires key and value',
              code: 'MISSING_PARAM',
              kind: 'orchestration',
              retryable: false,
            },
          };
        }

        logger.debug('Setting storage', { storage: storageType, key: params.key });

        await page.evaluate(
          ({ storage, key, value }) => {
            if (storage === 'localStorage') {
              // @ts-expect-error - window is available in browser context
              window.localStorage.setItem(key, value);
            } else {
              // @ts-expect-error - window is available in browser context
              window.sessionStorage.setItem(key, value);
            }
          },
          { storage: storageType, key: params.key, value: params.value }
        );

        logger.info('Storage value set', { storage: storageType, key: params.key });
        return { success: true };

      case 'get':
        logger.debug('Getting storage', { storage: storageType, key: params.key });

        const value = await page.evaluate(
          ({ storage, key }) => {
            if (storage === 'localStorage') {
              if (key) {
                // @ts-expect-error - window is available in browser context
                return window.localStorage.getItem(key);
              } else {
                // @ts-expect-error - window is available in browser context
                return Object.fromEntries(Object.entries(window.localStorage));
              }
            } else {
              if (key) {
                // @ts-expect-error - window is available in browser context
                return window.sessionStorage.getItem(key);
              } else {
                // @ts-expect-error - window is available in browser context
                return Object.fromEntries(Object.entries(window.sessionStorage));
              }
            }
          },
          { storage: storageType, key: params.key }
        );

        logger.info('Storage value retrieved', {
          storage: storageType,
          key: params.key,
          found: value !== null,
        });
        return {
          success: true,
          extracted_data: { value },
        };

      case 'delete':
        logger.debug('Deleting storage', { storage: storageType, key: params.key });

        await page.evaluate(
          ({ storage, key }) => {
            if (storage === 'localStorage') {
              if (key) {
                // @ts-expect-error - window is available in browser context
                window.localStorage.removeItem(key);
              } else {
                // @ts-expect-error - window is available in browser context
                window.localStorage.clear();
              }
            } else {
              if (key) {
                // @ts-expect-error - window is available in browser context
                window.sessionStorage.removeItem(key);
              } else {
                // @ts-expect-error - window is available in browser context
                window.sessionStorage.clear();
              }
            }
          },
          { storage: storageType, key: params.key }
        );

        logger.info('Storage deleted', { storage: storageType, key: params.key });
        return { success: true };

      case 'clear':
        logger.debug('Clearing storage', { storage: storageType });

        await page.evaluate(
          ({ storage }) => {
            if (storage === 'localStorage') {
              // @ts-expect-error - window is available in browser context
              window.localStorage.clear();
            } else {
              // @ts-expect-error - window is available in browser context
              window.sessionStorage.clear();
            }
          },
          { storage: storageType }
        );

        logger.info('Storage cleared', { storage: storageType });
        return { success: true };

      default:
        return {
          success: false,
          error: {
            message: `Unsupported storage operation: ${operation}`,
            code: 'UNSUPPORTED_OPERATION',
            kind: 'orchestration',
            retryable: false,
          },
        };
    }
  }
}
