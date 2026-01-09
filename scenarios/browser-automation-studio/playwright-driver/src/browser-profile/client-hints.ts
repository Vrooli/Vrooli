/**
 * Client Hints Module
 *
 * Generates Client Hints headers (sec-ch-ua) to match user agent strings.
 * Client Hints are sent by Chromium-based browsers and their absence or
 * mismatch with the User-Agent is a strong bot detection signal.
 */

/**
 * Client Hints headers for Chromium-based browsers.
 */
export interface ClientHintsHeaders {
  'sec-ch-ua': string;
  'sec-ch-ua-mobile': string;
  'sec-ch-ua-platform': string;
}

/**
 * Parsed user agent information.
 */
export interface ParsedUserAgent {
  browser: 'chrome' | 'edge' | 'firefox' | 'safari' | 'unknown';
  version: string;
  platform: 'Windows' | 'macOS' | 'Linux' | 'Android' | 'iOS' | 'Unknown';
  isMobile: boolean;
}

/**
 * Parse a user agent string to extract browser, version, and platform.
 */
export function parseUserAgent(userAgent: string): ParsedUserAgent {
  const ua = userAgent.toLowerCase();

  // Detect browser and version
  let browser: ParsedUserAgent['browser'] = 'unknown';
  let version = '120'; // Default version

  if (ua.includes('edg/')) {
    browser = 'edge';
    const match = userAgent.match(/Edg\/([\d.]+)/i);
    if (match) version = match[1].split('.')[0];
  } else if (ua.includes('chrome/')) {
    browser = 'chrome';
    const match = userAgent.match(/Chrome\/([\d.]+)/i);
    if (match) version = match[1].split('.')[0];
  } else if (ua.includes('firefox/')) {
    browser = 'firefox';
    const match = userAgent.match(/Firefox\/([\d.]+)/i);
    if (match) version = match[1].split('.')[0];
  } else if (ua.includes('safari/') && !ua.includes('chrome')) {
    browser = 'safari';
    const match = userAgent.match(/Version\/([\d.]+)/i);
    if (match) version = match[1].split('.')[0];
  }

  // Detect platform
  let platform: ParsedUserAgent['platform'] = 'Unknown';
  if (ua.includes('windows')) {
    platform = 'Windows';
  } else if (ua.includes('macintosh') || ua.includes('mac os x')) {
    platform = 'macOS';
  } else if (ua.includes('linux') && !ua.includes('android')) {
    platform = 'Linux';
  } else if (ua.includes('android')) {
    platform = 'Android';
  } else if (ua.includes('iphone') || ua.includes('ipad')) {
    platform = 'iOS';
  }

  // Detect mobile
  const isMobile = ua.includes('mobile') || platform === 'Android' || platform === 'iOS';

  return { browser, version, platform, isMobile };
}

/**
 * Generate GREASE placeholder brand for sec-ch-ua.
 * Chrome uses GREASE (Generate Random Extensions And Sustain Extensibility)
 * to prevent ossification of the protocol.
 */
function getGreaseBrand(): string {
  // Common GREASE value used by Chrome - use consistent value for reproducibility
  return 'Not_A Brand';
}

/**
 * Generate GREASE version number (should look plausible but be clearly fake).
 */
function getGreaseVersion(): string {
  // Common GREASE version
  return '24';
}

/**
 * Generate Client Hints headers for a Chromium-based browser.
 */
function generateChromiumClientHints(
  browser: 'chrome' | 'edge',
  version: string,
  platform: string,
  isMobile: boolean
): ClientHintsHeaders {
  const majorVersion = version.split('.')[0];
  const greaseBrand = getGreaseBrand();
  const greaseVersion = getGreaseVersion();

  // Build sec-ch-ua header
  // Format: "Brand";v="version", "Chromium";v="version", "Not_A Brand";v="version"
  let secChUa: string;

  if (browser === 'edge') {
    secChUa = `"Microsoft Edge";v="${majorVersion}", "Chromium";v="${majorVersion}", "${greaseBrand}";v="${greaseVersion}"`;
  } else {
    secChUa = `"Google Chrome";v="${majorVersion}", "Chromium";v="${majorVersion}", "${greaseBrand}";v="${greaseVersion}"`;
  }

  return {
    'sec-ch-ua': secChUa,
    'sec-ch-ua-mobile': isMobile ? '?1' : '?0',
    'sec-ch-ua-platform': `"${platform}"`,
  };
}

/**
 * Generate Client Hints headers based on a user agent string.
 * Returns null for non-Chromium browsers (Firefox, Safari) as they don't support Client Hints.
 */
export function generateClientHints(userAgent: string): ClientHintsHeaders | null {
  const parsed = parseUserAgent(userAgent);

  // Only generate Client Hints for Chromium-based browsers
  if (parsed.browser !== 'chrome' && parsed.browser !== 'edge') {
    return null;
  }

  return generateChromiumClientHints(
    parsed.browser,
    parsed.version,
    parsed.platform,
    parsed.isMobile
  );
}

/**
 * Merge Client Hints with user-provided extra headers.
 * User-provided headers take precedence over generated Client Hints.
 */
export function mergeClientHintsWithHeaders(
  clientHints: ClientHintsHeaders | null,
  extraHeaders?: Record<string, string>
): Record<string, string> {
  if (!clientHints) {
    return extraHeaders || {};
  }

  // Start with Client Hints, then override with user headers
  return {
    ...clientHints,
    ...(extraHeaders || {}),
  };
}
