import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { NavigateParamsSchema } from '../types/instruction';
import { DEFAULT_NAVIGATION_TIMEOUT_MS } from '../constants';
import { normalizeError } from '../utils';

// Allowed URL protocols for navigation (security hardening)
const ALLOWED_PROTOCOLS = ['http:', 'https:', 'about:', 'data:'];

/**
 * Validate and normalize URL for navigation
 *
 * Hardened assumptions:
 * - URL may be relative (needs base URL context)
 * - URL may have invalid format
 * - URL may use dangerous protocols (file:, javascript:)
 * - URL may exceed reasonable length limits
 */
function validateNavigationUrl(url: string): { valid: boolean; error?: string; normalized?: string } {
  // Check for empty or whitespace-only URL
  const trimmedUrl = url.trim();
  if (!trimmedUrl) {
    return { valid: false, error: 'URL cannot be empty' };
  }

  // Check URL length (reasonable limit to prevent abuse)
  if (trimmedUrl.length > 8192) {
    return { valid: false, error: `URL too long: ${trimmedUrl.length} chars (max: 8192)` };
  }

  try {
    // Try to parse as absolute URL first
    const parsed = new URL(trimmedUrl);

    // Check protocol is allowed (security: prevent file://, javascript:// etc.)
    if (!ALLOWED_PROTOCOLS.includes(parsed.protocol)) {
      return {
        valid: false,
        error: `Disallowed URL protocol: ${parsed.protocol}. Allowed: ${ALLOWED_PROTOCOLS.join(', ')}`,
      };
    }

    return { valid: true, normalized: parsed.href };
  } catch {
    // URL parsing failed - could be relative URL or invalid
    // For relative URLs, Playwright handles them with page context
    // But we should at least check it doesn't start with dangerous schemes
    const lowerUrl = trimmedUrl.toLowerCase();
    const dangerousSchemes = ['javascript:', 'file:', 'vbscript:', 'data:text/html'];
    for (const scheme of dangerousSchemes) {
      if (lowerUrl.startsWith(scheme)) {
        return { valid: false, error: `Disallowed URL scheme: ${scheme}` };
      }
    }

    // Allow relative URLs - Playwright will resolve them
    return { valid: true, normalized: trimmedUrl };
  }
}

/**
 * Navigation handler
 *
 * Handles page navigation operations
 */
export class NavigationHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['navigate'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      // Validate and parse parameters
      const params = NavigateParamsSchema.parse(instruction.params);

      // Get URL from params (support multiple param names for backwards compatibility)
      const url = params.url;
      if (!url) {
        return {
          success: false,
          error: {
            message: 'navigate instruction missing url parameter',
            code: 'MISSING_PARAM',
            kind: 'orchestration',
            retryable: false,
          },
        };
      }

      // Validate URL format and protocol
      const urlValidation = validateNavigationUrl(url);
      if (!urlValidation.valid) {
        logger.warn('instruction: navigate rejected - invalid URL', {
          url,
          error: urlValidation.error,
        });
        return {
          success: false,
          error: {
            message: urlValidation.error || 'Invalid URL',
            code: 'INVALID_URL',
            kind: 'orchestration',
            retryable: false,
          },
        };
      }

      const normalizedUrl = urlValidation.normalized || url;
      const timeout = params.timeoutMs || DEFAULT_NAVIGATION_TIMEOUT_MS;
      const waitUntil = params.waitUntil || 'networkidle';

      logger.debug('instruction: navigate starting', {
        targetUrl: normalizedUrl,
        timeout,
        waitUntil,
      });

      // Navigate
      await page.goto(normalizedUrl, {
        timeout,
        waitUntil,
      });

      const finalUrl = page.url();
      logger.info('instruction: navigate completed', {
        targetUrl: normalizedUrl,
        finalUrl,
        redirected: normalizedUrl !== finalUrl,
      });

      return {
        success: true,
      };
    } catch (error) {
      const driverError = normalizeError(error);
      logger.warn('instruction: navigate failed', {
        targetUrl: (instruction.params as Record<string, unknown>).url,
        errorCode: driverError.code,
        errorMessage: driverError.message,
        retryable: driverError.retryable,
      });

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
