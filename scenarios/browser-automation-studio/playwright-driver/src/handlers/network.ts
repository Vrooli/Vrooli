import { BaseHandler, HandlerContext, HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getNetworkMockParams } from '../types';
import { normalizeError } from '../utils/errors';
import { logger, scopedLog, LogContext } from '../utils';

/** Internal params type returned by getNetworkMockParams */
interface NetworkMockParams {
  operation: string;
  urlPattern: string;
  method?: string;
  statusCode?: number;
  headers?: Record<string, string>;
  body?: string;
  delayMs?: number;
}

/**
 * Canonical network operations supported by this handler.
 */
type NetworkOperation = 'mock' | 'block' | 'modifyRequest' | 'modifyResponse' | 'clear';

/**
 * DECISION BOUNDARY: Network Operations
 *
 * Each operation has distinct behavior and side effects:
 * - mock: Intercept requests and return synthetic responses
 * - block: Abort matching requests entirely
 * - modifyRequest: Alter request headers/body before sending
 * - modifyResponse: Alter response headers/body before returning
 * - clear: Remove all active route handlers for the session
 */
const NETWORK_OPERATIONS: Record<NetworkOperation, { description: string }> = {
  mock: { description: 'Mock response for matching requests' },
  block: { description: 'Block requests matching pattern' },
  modifyRequest: { description: 'Modify request headers/body before sending' },
  modifyResponse: { description: 'Modify response headers/body before returning' },
  clear: { description: 'Clear all mocks for session' },
} as const;

/** Result when an idempotent route is found */
interface IdempotentResult {
  isIdempotent: true;
  result: HandlerResult;
}

/** Result when route should be registered */
interface ProceedResult {
  isIdempotent: false;
  routeKey: string;
  routes: Map<string, RouteMetadata>;
}

/**
 * Track registered routes per session to enable idempotency.
 * Key: sessionId
 * Value: Map of routeKey -> route metadata
 *
 * Idempotency guarantee:
 * - Re-registering the same URL pattern + method is a no-op
 * - Prevents duplicate route handlers that would confuse Playwright
 * - Routes are cleared when 'clear' operation is called
 */
interface RouteMetadata {
  urlPattern: string;
  method?: string;
  operation: string;
  registeredAt: number;
}
const sessionRoutes: Map<string, Map<string, RouteMetadata>> = new Map();

/**
 * Generate a stable key for route idempotency.
 */
function generateRouteKey(urlPattern: string, method: string | undefined, operation: string): string {
  return `${operation}:${method || '*'}:${urlPattern}`;
}

/**
 * Get or create route map for a session.
 */
function getSessionRouteMap(sessionId: string): Map<string, RouteMetadata> {
  let routes = sessionRoutes.get(sessionId);
  if (!routes) {
    routes = new Map();
    sessionRoutes.set(sessionId, routes);
  }
  return routes;
}

/**
 * Clear all tracked routes for a session.
 * Exported for use by session reset operations.
 */
export function clearSessionRoutes(sessionId: string): void {
  sessionRoutes.delete(sessionId);
}

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
 * Idempotency behavior:
 * - Re-registering the same URL pattern + method + operation is a no-op
 * - Returns success without re-adding the route handler
 * - Prevents duplicate handlers that could cause unexpected behavior
 * - 'clear' operation removes all routes for the session
 *
 * Phase 3 handler - Network mocking and interception
 */
export class NetworkHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['network-mock', 'network', 'mock', 'intercept'];
  }

  /**
   * Check if a route is already registered (idempotency guard).
   *
   * This helper centralizes the idempotency check pattern used by all
   * route-registering operations (mock, block, modifyRequest, modifyResponse).
   *
   * @returns IdempotentResult if route exists (caller should return immediately),
   *          ProceedResult if route should be registered (caller should proceed)
   */
  private checkIdempotency(
    sessionId: string,
    urlPattern: string,
    method: string | undefined,
    operation: NetworkOperation,
    extraData?: Record<string, unknown>
  ): IdempotentResult | ProceedResult {
    const routeKey = generateRouteKey(urlPattern, method, operation);
    const routes = getSessionRouteMap(sessionId);

    if (routes.has(routeKey)) {
      const existing = routes.get(routeKey)!;
      logger.debug(scopedLog(LogContext.INSTRUCTION, `${operation} route already registered (idempotent)`), {
        sessionId,
        urlPattern,
        method,
        registeredAt: new Date(existing.registeredAt).toISOString(),
      });

      return {
        isIdempotent: true,
        result: {
          success: true,
          extracted_data: {
            network: {
              operation,
              urlPattern,
              method,
              idempotent: true,
              ...extraData,
            },
          },
        },
      };
    }

    return { isIdempotent: false, routeKey, routes };
  }

  /**
   * Register a route after successful setup.
   */
  private registerRoute(
    routes: Map<string, RouteMetadata>,
    routeKey: string,
    urlPattern: string,
    method: string | undefined,
    operation: string
  ): void {
    routes.set(routeKey, {
      urlPattern,
      method,
      operation,
      registeredAt: Date.now(),
    });
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    try {
      // Get typed params from instruction.action (required after migration)
      const typedParams = instruction.action ? getNetworkMockParams(instruction.action) : undefined;
      const validated = this.requireTypedParams(typedParams, 'network-mock', instruction.nodeId);

      logger.debug('Executing network operation', {
        operation: validated.operation,
        urlPattern: validated.urlPattern,
        method: validated.method,
      });

      // Validate operation against known operations
      const operation = validated.operation as NetworkOperation;
      if (!(operation in NETWORK_OPERATIONS)) {
        return {
          success: false,
          error: {
            message: `Unknown network operation: ${validated.operation}. Valid operations: ${Object.keys(NETWORK_OPERATIONS).join(', ')}`,
            code: 'INVALID_OPERATION',
            kind: 'user',
            retryable: false,
          },
        };
      }

      // Dispatch to appropriate handler
      switch (operation) {
        case 'mock':
          return this.handleMockResponse(validated, context);
        case 'block':
          return this.handleBlockRequest(validated, context);
        case 'modifyRequest':
          return this.handleModifyRequest(validated, context);
        case 'modifyResponse':
          return this.handleModifyResponse(validated, context);
        case 'clear':
          return this.handleClearMocks(context);
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
   *
   * Idempotency: If this exact mock is already registered, returns success
   * without re-registering. This prevents duplicate handlers.
   */
  private async handleMockResponse(
    params: NetworkMockParams,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const page = context.page;
    const sessionId = context.sessionId;
    const urlPattern = this.compileUrlPattern(params.urlPattern);
    const urlPatternStr = params.urlPattern.toString();

    // Idempotency check
    const idempotencyCheck = this.checkIdempotency(
      sessionId, urlPatternStr, params.method, 'mock',
      { statusCode: params.statusCode }
    );
    if (idempotencyCheck.isIdempotent) {
      return idempotencyCheck.result;
    }
    const { routeKey, routes } = idempotencyCheck;

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
      // Temporal hardening: If the page closes during the delay, route.fulfill() will
      // throw an error. We wrap the delay in a try-catch to handle page-closed gracefully.
      if (params.delayMs && params.delayMs > 0) {
        await new Promise((resolve) => setTimeout(resolve, params.delayMs));
      }

      // Fulfill with mock response
      // Temporal hardening: Page may have closed during delay - handle gracefully
      try {
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
      } catch (err) {
        // Page may have closed - this is expected during shutdown
        const message = err instanceof Error ? err.message : String(err);
        if (message.includes('closed') || message.includes('Target closed')) {
          logger.debug('Mock response skipped - page closed', {
            url: route.request().url(),
          });
        } else {
          // Re-throw unexpected errors
          throw err;
        }
      }
    });

    // Track this route registration for idempotency
    this.registerRoute(routes, routeKey, urlPatternStr, params.method, 'mock');

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
   *
   * Idempotency: If this exact block is already registered, returns success
   * without re-registering. This prevents duplicate handlers.
   */
  private async handleBlockRequest(
    params: NetworkMockParams,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const page = context.page;
    const sessionId = context.sessionId;
    const urlPattern = this.compileUrlPattern(params.urlPattern);
    const urlPatternStr = params.urlPattern.toString();

    // Idempotency check
    const idempotencyCheck = this.checkIdempotency(
      sessionId, urlPatternStr, params.method, 'block'
    );
    if (idempotencyCheck.isIdempotent) {
      return idempotencyCheck.result;
    }
    const { routeKey, routes } = idempotencyCheck;

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

    // Track this route registration for idempotency
    this.registerRoute(routes, routeKey, urlPatternStr, params.method, 'block');

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
   *
   * Idempotency: If this exact modifier is already registered, returns success
   * without re-registering. This prevents duplicate handlers.
   */
  private async handleModifyRequest(
    params: NetworkMockParams,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const page = context.page;
    const sessionId = context.sessionId;
    const urlPattern = this.compileUrlPattern(params.urlPattern);
    const urlPatternStr = params.urlPattern.toString();

    // Idempotency check
    const idempotencyCheck = this.checkIdempotency(
      sessionId, urlPatternStr, params.method, 'modifyRequest'
    );
    if (idempotencyCheck.isIdempotent) {
      return idempotencyCheck.result;
    }
    const { routeKey, routes } = idempotencyCheck;

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

    // Track this route registration for idempotency
    this.registerRoute(routes, routeKey, urlPatternStr, params.method, 'modifyRequest');

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
   *
   * Idempotency: If this exact modifier is already registered, returns success
   * without re-registering. This prevents duplicate handlers.
   */
  private async handleModifyResponse(
    params: NetworkMockParams,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const page = context.page;
    const sessionId = context.sessionId;
    const urlPattern = this.compileUrlPattern(params.urlPattern);
    const urlPatternStr = params.urlPattern.toString();

    // Idempotency check
    const idempotencyCheck = this.checkIdempotency(
      sessionId, urlPatternStr, params.method, 'modifyResponse',
      { statusCode: params.statusCode }
    );
    if (idempotencyCheck.isIdempotent) {
      return idempotencyCheck.result;
    }
    const { routeKey, routes } = idempotencyCheck;

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

    // Track this route registration for idempotency
    this.registerRoute(routes, routeKey, urlPatternStr, params.method, 'modifyResponse');

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
   *
   * Idempotency: This operation is inherently idempotent - clearing an already
   * cleared state is a no-op and returns success.
   */
  private async handleClearMocks(context: HandlerContext): Promise<HandlerResult> {
    const page = context.page;
    const sessionId = context.sessionId;

    // Get the route count before clearing for logging
    const routes = getSessionRouteMap(sessionId);
    const routeCount = routes.size;

    logger.debug('Clearing all network mocks', {
      sessionId,
      trackedRouteCount: routeCount,
    });

    // Unroute all handlers
    await page.unroute('**/*');

    // Clear tracked routes for this session
    clearSessionRoutes(sessionId);

    logger.info('Network mocks cleared', {
      sessionId,
      clearedRouteCount: routeCount,
    });

    return {
      success: true,
      extracted_data: {
        network: {
          operation: 'clear',
          clearedRouteCount: routeCount,
        },
      },
    };
  }

  /**
   * Compile URL pattern to Playwright-compatible format.
   *
   * Hardened: Validates regex patterns and catches invalid patterns.
   */
  private compileUrlPattern(pattern: string | RegExp): string | RegExp {
    if (pattern instanceof RegExp) {
      return pattern;
    }

    // If pattern is already a glob/regex pattern, return as-is
    if (pattern.includes('*') || pattern.includes('?')) {
      return pattern;
    }

    // Otherwise, treat as substring match - escape and wrap in regex
    try {
      const escaped = pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
      const regex = new RegExp(escaped);
      // Validate the regex is usable
      regex.test('');
      return regex;
    } catch (error) {
      logger.warn('Invalid URL pattern, using as literal string', {
        pattern,
        error: error instanceof Error ? error.message : String(error),
      });
      // Fall back to the escaped string pattern for glob matching
      return `*${pattern}*`;
    }
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
