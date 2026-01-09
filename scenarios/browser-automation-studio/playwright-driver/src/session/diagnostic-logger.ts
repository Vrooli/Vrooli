/**
 * Diagnostic Logger for Redirect Loop Debugging
 *
 * This module provides comprehensive logging to help diagnose redirect loops
 * and other navigation issues. Enable by setting DIAGNOSTIC_LOGGING=true env var.
 *
 * Logs:
 * - All navigation events (framenavigated, load, domcontentloaded)
 * - Network requests/responses to specified domains (e.g., google.com)
 * - Service worker registrations and lifecycle events
 * - Anti-detection script injections
 * - Client Hints headers being sent
 * - Response headers from target domains
 */

import type { Page, BrowserContext, Frame, Request, Response } from 'rebrowser-playwright';
import { logger, LogContext, scopedLog } from '../utils';

// Enable diagnostic logging via environment variable
const DIAGNOSTIC_ENABLED = process.env.DIAGNOSTIC_LOGGING === 'true';

// Domains to monitor for network activity (case-insensitive)
const MONITORED_DOMAINS = [
  'google.com',
  'www.google.com',
  'accounts.google.com',
  'gstatic.com',
  'googleapis.com',
];

/**
 * Check if a URL belongs to a monitored domain
 */
function isMonitoredDomain(url: string): boolean {
  try {
    const hostname = new URL(url).hostname.toLowerCase();
    return MONITORED_DOMAINS.some(
      (domain) => hostname === domain || hostname.endsWith('.' + domain)
    );
  } catch {
    return false;
  }
}

/**
 * Extract key URL parameters for debugging
 */
function extractUrlParams(url: string): Record<string, string> {
  try {
    const urlObj = new URL(url);
    const params: Record<string, string> = {};
    // Only extract params that might indicate redirect issues
    const interestingParams = ['zx', 'no_sw_cr', 'sw', 'sca_esv', 'source', 'ei', 'redirect'];
    for (const key of interestingParams) {
      const value = urlObj.searchParams.get(key);
      if (value !== null) {
        params[key] = value;
      }
    }
    return params;
  } catch {
    return {};
  }
}

/**
 * Extract relevant response headers for debugging
 */
function extractResponseHeaders(response: Response): Record<string, string> {
  const headers: Record<string, string> = {};
  const allHeaders = response.headers();

  // Headers that might indicate redirect or SW issues
  const relevantHeaders = [
    'location',
    'refresh',
    'service-worker-allowed',
    'x-frame-options',
    'content-security-policy',
    'cache-control',
    'set-cookie',
  ];

  for (const header of relevantHeaders) {
    if (allHeaders[header]) {
      headers[header] = allHeaders[header];
    }
  }

  return headers;
}

/**
 * Setup diagnostic logging on a browser context
 */
export function setupDiagnosticLogging(
  context: BrowserContext,
  sessionId: string
): void {
  if (!DIAGNOSTIC_ENABLED) {
    logger.debug(scopedLog(LogContext.SESSION, 'diagnostic logging disabled'), {
      sessionId,
      hint: 'Set DIAGNOSTIC_LOGGING=true to enable',
    });
    return;
  }

  logger.info(scopedLog(LogContext.SESSION, '🔍 DIAGNOSTIC LOGGING ENABLED'), {
    sessionId,
    monitoredDomains: MONITORED_DOMAINS,
  });

  // Track navigation count for loop detection visibility
  let navigationCount = 0;
  const navigationTimestamps: { url: string; timestamp: number }[] = [];

  // Log all new pages
  context.on('page', (page: Page) => {
    const pageUrl = page.url();
    logger.info(scopedLog(LogContext.SESSION, '🔍 [DIAG] new page created'), {
      sessionId,
      url: pageUrl,
    });

    setupPageDiagnostics(page, sessionId, navigationTimestamps, () => navigationCount++);
  });

  // Log existing pages
  for (const page of context.pages()) {
    setupPageDiagnostics(page, sessionId, navigationTimestamps, () => navigationCount++);
  }
}

/**
 * Setup diagnostic logging on a single page
 */
function setupPageDiagnostics(
  page: Page,
  sessionId: string,
  navigationTimestamps: { url: string; timestamp: number }[],
  incrementNavCount: () => number
): void {
  // Log all frame navigations (including redirects)
  page.on('framenavigated', (frame: Frame) => {
    if (frame !== page.mainFrame()) {
      return; // Only log main frame navigations
    }

    const url = frame.url();
    const navCount = incrementNavCount();
    const now = Date.now();

    // Track for loop detection
    navigationTimestamps.push({ url, timestamp: now });
    // Keep only last 20 navigations
    if (navigationTimestamps.length > 20) {
      navigationTimestamps.shift();
    }

    // Calculate navigation frequency
    const recentNavs = navigationTimestamps.filter((n) => now - n.timestamp < 5000);
    const navRate = recentNavs.length;

    const params = extractUrlParams(url);
    const hasSwParam = 'no_sw_cr' in params || 'sw' in params;

    logger.info(scopedLog(LogContext.SESSION, `🔍 [DIAG] navigation #${navCount}`), {
      sessionId,
      url: url.slice(0, 200),
      params: Object.keys(params).length > 0 ? params : undefined,
      hasSwParam,
      navRate: `${navRate}/5s`,
      isMonitored: isMonitoredDomain(url),
    });

    // Warn if navigation rate is high (potential loop)
    if (navRate >= 4) {
      logger.warn(scopedLog(LogContext.SESSION, '🔍 [DIAG] HIGH NAVIGATION RATE - potential loop'), {
        sessionId,
        navRate,
        recentUrls: recentNavs.map((n) => n.url.slice(0, 100)),
      });
    }
  });

  // Log load events
  page.on('load', () => {
    const url = page.url();
    logger.debug(scopedLog(LogContext.SESSION, '🔍 [DIAG] page load'), {
      sessionId,
      url: url.slice(0, 200),
    });
  });

  // Log domcontentloaded events
  page.on('domcontentloaded', () => {
    const url = page.url();
    logger.debug(scopedLog(LogContext.SESSION, '🔍 [DIAG] DOMContentLoaded'), {
      sessionId,
      url: url.slice(0, 200),
    });
  });

  // Log network requests to monitored domains
  page.on('request', (request: Request) => {
    const url = request.url();
    if (!isMonitoredDomain(url)) {
      return;
    }

    const resourceType = request.resourceType();
    const headers = request.headers();

    // Log service worker related requests
    const isSwRelated =
      resourceType === 'serviceworker' ||
      url.includes('sw.js') ||
      url.includes('service-worker') ||
      url.includes('workbox');

    logger.info(scopedLog(LogContext.SESSION, '🔍 [DIAG] request'), {
      sessionId,
      method: request.method(),
      url: url.slice(0, 200),
      resourceType,
      isSwRelated,
      // Log Client Hints headers being sent
      clientHints: {
        'sec-ch-ua': headers['sec-ch-ua'],
        'sec-ch-ua-mobile': headers['sec-ch-ua-mobile'],
        'sec-ch-ua-platform': headers['sec-ch-ua-platform'],
      },
      referer: headers['referer']?.slice(0, 100),
    });
  });

  // Log network responses from monitored domains
  page.on('response', (response: Response) => {
    const url = response.url();
    if (!isMonitoredDomain(url)) {
      return;
    }

    const status = response.status();
    const relevantHeaders = extractResponseHeaders(response);

    // Check for redirects
    const isRedirect = status >= 300 && status < 400;
    const hasLocationHeader = 'location' in relevantHeaders;

    logger.info(scopedLog(LogContext.SESSION, '🔍 [DIAG] response'), {
      sessionId,
      url: url.slice(0, 200),
      status,
      isRedirect,
      redirectLocation: relevantHeaders['location']?.slice(0, 200),
      relevantHeaders:
        Object.keys(relevantHeaders).length > 0 ? relevantHeaders : undefined,
    });

    // Extra warning for redirect responses
    if (isRedirect && hasLocationHeader) {
      logger.warn(scopedLog(LogContext.SESSION, '🔍 [DIAG] HTTP REDIRECT detected'), {
        sessionId,
        from: url.slice(0, 200),
        to: relevantHeaders['location']?.slice(0, 200),
        status,
      });
    }
  });

  // Log request failures
  page.on('requestfailed', (request: Request) => {
    const url = request.url();
    if (!isMonitoredDomain(url)) {
      return;
    }

    const failure = request.failure();
    logger.warn(scopedLog(LogContext.SESSION, '🔍 [DIAG] request FAILED'), {
      sessionId,
      url: url.slice(0, 200),
      resourceType: request.resourceType(),
      error: failure?.errorText || 'unknown',
    });
  });

  // Log console messages that might indicate SW issues
  page.on('console', (msg) => {
    const text = msg.text().toLowerCase();
    if (
      text.includes('service worker') ||
      text.includes('serviceworker') ||
      text.includes('sw.js') ||
      text.includes('workbox') ||
      text.includes('redirect') ||
      text.includes('no_sw_cr')
    ) {
      logger.info(scopedLog(LogContext.SESSION, '🔍 [DIAG] console (SW/redirect related)'), {
        sessionId,
        type: msg.type(),
        text: msg.text().slice(0, 500),
      });
    }
  });
}

/**
 * Log anti-detection script application
 */
export function logAntiDetectionApplied(
  sessionId: string,
  enabledPatches: string[]
): void {
  if (!DIAGNOSTIC_ENABLED) return;

  logger.info(scopedLog(LogContext.SESSION, '🔍 [DIAG] anti-detection patches applied'), {
    sessionId,
    patchCount: enabledPatches.length,
    patches: enabledPatches,
  });
}

/**
 * Log Client Hints configuration
 */
export function logClientHints(
  sessionId: string,
  userAgent: string,
  clientHints: Record<string, string> | null
): void {
  if (!DIAGNOSTIC_ENABLED) return;

  logger.info(scopedLog(LogContext.SESSION, '🔍 [DIAG] client hints configured'), {
    sessionId,
    userAgent: userAgent.slice(0, 100),
    clientHints: clientHints || 'none (non-Chromium UA)',
  });
}

/**
 * Log ad blocker configuration
 */
export function logAdBlockerConfig(
  sessionId: string,
  mode: string,
  whitelist: string[]
): void {
  if (!DIAGNOSTIC_ENABLED) return;

  logger.info(scopedLog(LogContext.SESSION, '🔍 [DIAG] ad blocker configured'), {
    sessionId,
    mode,
    whitelist: whitelist.length > 0 ? whitelist : 'none',
  });
}

/**
 * Log context options being used
 */
export function logContextOptions(
  sessionId: string,
  options: Record<string, unknown>
): void {
  if (!DIAGNOSTIC_ENABLED) return;

  // Sanitize options to remove sensitive data
  const sanitized = { ...options };
  if (sanitized.proxy && typeof sanitized.proxy === 'object') {
    sanitized.proxy = { ...sanitized.proxy as Record<string, unknown>, password: '[REDACTED]' };
  }
  if (sanitized.storageState) {
    sanitized.storageState = '[PRESENT]';
  }

  logger.info(scopedLog(LogContext.SESSION, '🔍 [DIAG] context options'), {
    sessionId,
    options: sanitized,
  });
}
