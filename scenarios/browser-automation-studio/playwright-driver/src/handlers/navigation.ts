import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getNavigateParams } from '../types';
import { normalizeError } from '../utils';
import { resolveTimeoutFromContext } from './behavior-utils';
import type { AppTargetSpec } from '../types/session';
import path from 'node:path';
import { applyInteractionState } from '../session/interaction-state';

// =============================================================================
// URL VALIDATION - Decision Boundary for Navigation Security
// =============================================================================

/**
 * CONTROL LEVER: Allowed URL protocols for navigation.
 *
 * SECURITY DECISION: Only these protocols are permitted.
 * - http/https: Standard web navigation
 * - about: Browser internal pages (about:blank)
 * - data: Data URLs (with restrictions on dangerous types)
 */
const ALLOWED_PROTOCOLS = ['http:', 'https:', 'about:', 'data:'];

/**
 * CONTROL LEVER: Dangerous URL schemes that are always blocked.
 *
 * SECURITY DECISION: These schemes can execute code or access local files.
 * They are blocked even for relative URLs (which bypass URL() parsing).
 */
const DANGEROUS_SCHEMES = ['javascript:', 'file:', 'vbscript:', 'data:text/html'];

/**
 * CONTROL LEVER: Maximum URL length to prevent abuse.
 *
 * DECISION: 8KB is generous for legitimate URLs while preventing abuse.
 * Most browsers limit URLs to ~2KB anyway for practical purposes.
 */
const MAX_URL_LENGTH = 8192;

/**
 * URL validation result with structured error information.
 */
export interface UrlValidationResult {
  valid: boolean;
  error?: string;
  normalized?: string;
  /** Which check failed (for debugging) */
  failedCheck?: 'empty' | 'length' | 'protocol' | 'dangerous_scheme';
}

/**
 * Validate and normalize URL for navigation.
 *
 * DECISION BOUNDARY: This is the single point where URL safety is determined.
 *
 * Security checks performed (in order):
 * 1. Empty URL check - reject empty/whitespace URLs
 * 2. Length check - prevent excessively long URLs
 * 3. Protocol allowlist - only permit safe protocols
 * 4. Dangerous scheme blocklist - catch schemes that bypass URL() parsing
 *
 * @param url - The URL to validate
 * @returns Validation result with normalized URL if valid
 */
export function validateNavigationUrl(
  url: string,
  electronTarget?: AppTargetSpec
): UrlValidationResult {
  // CHECK 1: Empty URL
  const trimmedUrl = url.trim();
  if (!trimmedUrl) {
    return { valid: false, error: 'URL cannot be empty', failedCheck: 'empty' };
  }

  // CHECK 2: Length limit
  if (trimmedUrl.length > MAX_URL_LENGTH) {
    return {
      valid: false,
      error: `URL too long: ${trimmedUrl.length} chars (max: ${MAX_URL_LENGTH})`,
      failedCheck: 'length',
    };
  }

  // CHECK 3: Parse as absolute URL and check protocol
  try {
    const parsed = new URL(trimmedUrl);

    // Protocol allowlist check
    const admittedFileRenderer =
      parsed.protocol === 'file:' && isAdmittedFileRenderer(parsed.href, electronTarget);
    if (!ALLOWED_PROTOCOLS.includes(parsed.protocol) && !admittedFileRenderer) {
      return {
        valid: false,
        error: `Disallowed URL protocol: ${parsed.protocol}. Allowed: ${ALLOWED_PROTOCOLS.join(', ')}`,
        failedCheck: 'protocol',
      };
    }

    return { valid: true, normalized: parsed.href };
  } catch {
    // URL parsing failed - could be relative URL or invalid
    // Relative URLs are fine - Playwright resolves them against current page
    // But we must still check for dangerous schemes that bypass URL() parsing
  }

  // CHECK 4: Dangerous scheme blocklist for relative/malformed URLs
  const lowerUrl = trimmedUrl.toLowerCase();
  for (const scheme of DANGEROUS_SCHEMES) {
    if (lowerUrl.startsWith(scheme)) {
      return {
        valid: false,
        error: `Disallowed URL scheme: ${scheme}`,
        failedCheck: 'dangerous_scheme',
      };
    }
  }

  // DECISION: Allow relative URLs - Playwright will resolve them
  return { valid: true, normalized: trimmedUrl };
}

function isAdmittedFileRenderer(url: string, target?: AppTargetSpec): boolean {
  if (!target || !target.renderer_url.startsWith('file:')) return false;
  try {
    const admitted = new URL(target.renderer_url);
    const requested = new URL(url);
    if (admitted.protocol !== 'file:' || requested.protocol !== 'file:') return false;
    const directory = path.posix.dirname(admitted.pathname);
    return (
      requested.pathname === admitted.pathname || requested.pathname.startsWith(`${directory}/`)
    );
  } catch {
    return false;
  }
}

// =============================================================================
// NAVIGATION HANDLER
// =============================================================================

/**
 * Navigation handler
 *
 * Handles page navigation operations
 */
export class NavigationHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['navigate'];
  }

  async execute(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      // Extract typed params from action
      const typedParams = instruction.action ? getNavigateParams(instruction.action) : undefined;
      const params = this.requireTypedParams(typedParams, 'navigate', instruction.nodeId);

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
      const urlValidation = validateNavigationUrl(url, context.electronTarget);
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
      // DECISION: Use 'navigation' category - networks can be slow, need longer timeout
      const timeout = resolveTimeoutFromContext(params.timeoutMs, context, 'navigation');
      // Navigation establishes a document boundary only. Readiness is a
      // separate post-navigation operation so caller waits cannot be confused
      // with page.goto's timeout.
      const requestedWaitUntil = params.waitUntil as
        | 'load'
        | 'domcontentloaded'
        | 'networkidle'
        | 'commit'
        | undefined;
      const waitUntil = 'domcontentloaded' as const;

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

      if (requestedWaitUntil === 'load' || requestedWaitUntil === 'networkidle') {
        await page.waitForLoadState(requestedWaitUntil, { timeout });
      }

      if (params.waitForSelector) {
        await page.waitForSelector(params.waitForSelector, { timeout, state: 'visible' });
      }

      if (context.interactionState && context.interactionState !== 'rest') {
        await applyInteractionState(page, context.interactionState, timeout);
      }

      const finalUrl = page.url();

      // Frame stack invalidation: After navigation, any stored frame references
      // become stale. Clear the frame stack to prevent replayed frame-switch
      // instructions from targeting invalid frames.
      // This is an idempotency safeguard: if the same workflow is replayed after
      // a navigation failure and retry, we won't have stale frame references.
      if (context.frameStack && context.frameStack.length > 0) {
        const clearedFrames = context.frameStack.length;
        context.frameStack.length = 0; // Clear in-place
        logger.debug('instruction: navigate cleared stale frame stack', {
          clearedFrames,
          hint: 'Frame references invalidated after navigation',
        });
      }

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
        targetUrl: getNavigateParams(instruction.action)?.url,
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
