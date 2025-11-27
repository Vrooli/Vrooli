import { BaseHandler, HandlerContext, HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { NetworkMockParamsSchema, type NetworkMockParams } from '../types/instruction';
import { normalizeError } from '../utils/errors';

/**
 * NetworkHandler implements request interception and response mocking
 *
 * Supported instruction types:
 * - network-mock: Intercept and modify network requests/responses
 *
 * Operations:
 * - mock: Mock response for matching requests
 * - block: Block requests matching pattern
 * - modifyRequest: Modify request headers/body
 * - modifyResponse: Modify response headers/body
 * - clear: Clear all active mocks
 *
 * State Management:
 * - Active routes are stored in session context
 * - Routes persist across instructions until cleared
 * - Multiple routes can be active simultaneously
 *
 * Phase 3 handler - Network mocking and interception
 */
export class NetworkHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['network-mock', 'network', 'mock', 'intercept'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { logger } = context;

    try {
      const validated = NetworkMockParamsSchema.parse(instruction.params);

      logger.debug('Executing network operation', {
        operation: validated.operation,
        urlPattern: validated.urlPattern,
        method: validated.method,
      });

      switch (validated.operation) {
        case 'mock':
          return this.handleMockResponse(validated, context, logger);
        case 'block':
          return this.handleBlockRequest(validated, context, logger);
        case 'modifyRequest':
          return this.handleModifyRequest(validated, context, logger);
        case 'modifyResponse':
          return this.handleModifyResponse(validated, context, logger);
        case 'clear':
          return this.handleClearMocks(context, logger);
        default:
          return {
            success: false,
            error: {
              message: `Unknown network operation: ${validated.operation}`,
              code: 'INVALID_OPERATION',
              kind: 'user',
              retryable: false,
            },
          };
      }
    } catch (error) {
      logger.error('Network operation failed', {
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
   * Mock response for matching requests
   */
  private async handleMockResponse(
    params: NetworkMockParams,
    context: HandlerContext,
    logger: any
  ): Promise<HandlerResult> {
    const page = context.page;
    const urlPattern = this.compileUrlPattern(params.urlPattern);

    logger.debug('Setting up mock response', {
      urlPattern: params.urlPattern,
      statusCode: params.statusCode,
      hasBody: !!params.body,
    });

    await page.route(urlPattern, async (route) => {
      // Check if method matches (if specified)
      if (params.method && route.request().method() !== params.method.toUpperCase()) {
        await route.continue();
        return;
      }

      // Simulate delay if specified
      if (params.delayMs && params.delayMs > 0) {
        await new Promise((resolve) => setTimeout(resolve, params.delayMs));
      }

      // Fulfill with mock response
      await route.fulfill({
        status: params.statusCode || 200,
        headers: params.headers || {},
        body: this.serializeBody(params.body),
        contentType: this.getContentType(params.body, params.headers),
      });

      logger.debug('Mock response fulfilled', {
        url: route.request().url(),
        status: params.statusCode,
      });
    });

    return {
      success: true,
      extracted_data: {
        network: {
          operation: 'mock',
          urlPattern: params.urlPattern.toString(),
          method: params.method,
          statusCode: params.statusCode,
        },
      },
    };
  }

  /**
   * Block requests matching pattern
   */
  private async handleBlockRequest(
    params: NetworkMockParams,
    context: HandlerContext,
    logger: any
  ): Promise<HandlerResult> {
    const page = context.page;
    const urlPattern = this.compileUrlPattern(params.urlPattern);

    logger.debug('Setting up request blocking', {
      urlPattern: params.urlPattern,
      method: params.method,
    });

    await page.route(urlPattern, async (route) => {
      // Check if method matches (if specified)
      if (params.method && route.request().method() !== params.method.toUpperCase()) {
        await route.continue();
        return;
      }

      // Abort the request
      await route.abort('blockedbyclient');

      logger.debug('Request blocked', {
        url: route.request().url(),
        method: route.request().method(),
      });
    });

    return {
      success: true,
      extracted_data: {
        network: {
          operation: 'block',
          urlPattern: params.urlPattern.toString(),
          method: params.method,
        },
      },
    };
  }

  /**
   * Modify request headers/body before sending
   */
  private async handleModifyRequest(
    params: NetworkMockParams,
    context: HandlerContext,
    logger: any
  ): Promise<HandlerResult> {
    const page = context.page;
    const urlPattern = this.compileUrlPattern(params.urlPattern);

    logger.debug('Setting up request modification', {
      urlPattern: params.urlPattern,
      hasHeaders: !!params.headers,
      hasBody: !!params.body,
    });

    await page.route(urlPattern, async (route) => {
      // Check if method matches (if specified)
      if (params.method && route.request().method() !== params.method.toUpperCase()) {
        await route.continue();
        return;
      }

      // Continue with modified request
      await route.continue({
        headers: params.headers ? { ...route.request().headers(), ...params.headers } : undefined,
        postData: params.body ? this.serializeBody(params.body) : undefined,
      });

      logger.debug('Request modified', {
        url: route.request().url(),
      });
    });

    return {
      success: true,
      extracted_data: {
        network: {
          operation: 'modifyRequest',
          urlPattern: params.urlPattern.toString(),
          method: params.method,
        },
      },
    };
  }

  /**
   * Modify response headers/body before returning
   *
   * Note: Playwright's route.fulfill() with modified response requires
   * fetching the original response first
   */
  private async handleModifyResponse(
    params: NetworkMockParams,
    context: HandlerContext,
    logger: any
  ): Promise<HandlerResult> {
    const page = context.page;
    const urlPattern = this.compileUrlPattern(params.urlPattern);

    logger.debug('Setting up response modification', {
      urlPattern: params.urlPattern,
      hasHeaders: !!params.headers,
      hasBody: !!params.body,
    });

    await page.route(urlPattern, async (route) => {
      // Check if method matches (if specified)
      if (params.method && route.request().method() !== params.method.toUpperCase()) {
        await route.continue();
        return;
      }

      // Fetch original response
      const response = await route.fetch();

      // Prepare modified response
      const modifiedHeaders = params.headers
        ? { ...response.headers(), ...params.headers }
        : response.headers();

      const modifiedBody = params.body ? this.serializeBody(params.body) : await response.body();

      // Fulfill with modified response
      await route.fulfill({
        status: params.statusCode || response.status(),
        headers: modifiedHeaders,
        body: modifiedBody,
      });

      logger.debug('Response modified', {
        url: route.request().url(),
        status: params.statusCode || response.status(),
      });
    });

    return {
      success: true,
      extracted_data: {
        network: {
          operation: 'modifyResponse',
          urlPattern: params.urlPattern.toString(),
          method: params.method,
          statusCode: params.statusCode,
        },
      },
    };
  }

  /**
   * Clear all active network mocks/routes
   */
  private async handleClearMocks(context: HandlerContext, logger: any): Promise<HandlerResult> {
    const page = context.page;

    logger.debug('Clearing all network mocks');

    // Unroute all handlers
    await page.unroute('**/*');

    logger.info('Network mocks cleared');

    return {
      success: true,
      extracted_data: {
        network: {
          operation: 'clear',
        },
      },
    };
  }

  /**
   * Compile URL pattern to Playwright-compatible format
   */
  private compileUrlPattern(pattern: string | RegExp): string | RegExp {
    if (pattern instanceof RegExp) {
      return pattern;
    }

    // If pattern is already a glob/regex pattern, return as-is
    if (pattern.includes('*') || pattern.includes('?')) {
      return pattern;
    }

    // Otherwise, treat as substring match
    return new RegExp(pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));
  }

  /**
   * Serialize body for network response
   */
  private serializeBody(body: unknown): string | Buffer {
    if (!body) {
      return '';
    }

    if (typeof body === 'string') {
      return body;
    }

    if (Buffer.isBuffer(body)) {
      return body;
    }

    // Assume JSON-serializable object
    return JSON.stringify(body);
  }

  /**
   * Determine content type from body and headers
   */
  private getContentType(body: unknown, headers?: Record<string, string>): string {
    if (headers?.['content-type'] || headers?.['Content-Type']) {
      return headers['content-type'] || headers['Content-Type'];
    }

    if (typeof body === 'string') {
      // Try to detect JSON
      if (body.trim().startsWith('{') || body.trim().startsWith('[')) {
        return 'application/json';
      }
      return 'text/plain';
    }

    if (typeof body === 'object') {
      return 'application/json';
    }

    return 'application/octet-stream';
  }
}
