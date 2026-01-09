/**
 * Form validation utilities for common input patterns.
 */

/**
 * Validates a proxy server URL.
 * Accepts http://, https://, socks4://, and socks5:// protocols.
 * @returns Error message or undefined if valid
 */
export function validateProxyUrl(url: string | undefined): string | undefined {
  if (!url) return undefined;

  // Basic protocol check
  const validProtocols = ['http://', 'https://', 'socks4://', 'socks5://'];
  const hasValidProtocol = validProtocols.some((p) => url.toLowerCase().startsWith(p));

  if (!hasValidProtocol) {
    return 'URL must start with http://, https://, socks4://, or socks5://';
  }

  // Try to parse as URL
  try {
    const parsed = new URL(url);
    // Must have a hostname
    if (!parsed.hostname) {
      return 'Invalid proxy URL: missing hostname';
    }
  } catch {
    return 'Invalid proxy URL format';
  }

  return undefined;
}

/**
 * Validates latitude value (-90 to 90).
 * @returns Error message or undefined if valid
 */
export function validateLatitude(lat: number | undefined): string | undefined {
  if (lat === undefined) return undefined;
  if (lat < -90 || lat > 90) {
    return 'Latitude must be between -90 and 90';
  }
  return undefined;
}

/**
 * Validates longitude value (-180 to 180).
 * @returns Error message or undefined if valid
 */
export function validateLongitude(lng: number | undefined): string | undefined {
  if (lng === undefined) return undefined;
  if (lng < -180 || lng > 180) {
    return 'Longitude must be between -180 and 180';
  }
  return undefined;
}

/**
 * Validates an HTTP header name.
 * Header names must contain only valid token characters per RFC 7230.
 * @returns Error message or undefined if valid
 */
export function validateHttpHeaderName(name: string): string | undefined {
  if (!name) {
    return 'Header name is required';
  }

  // RFC 7230 token: 1*tchar
  // tchar = "!" / "#" / "$" / "%" / "&" / "'" / "*" / "+" / "-" / "." /
  //         "^" / "_" / "`" / "|" / "~" / DIGIT / ALPHA
  const tokenRegex = /^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/;

  if (!tokenRegex.test(name)) {
    return 'Header name contains invalid characters';
  }

  return undefined;
}

/**
 * List of HTTP headers that cannot be set programmatically.
 */
export const BLOCKED_HEADERS = ['host', 'content-length', 'cookie'] as const;

/**
 * Checks if a header name is blocked.
 */
export function isBlockedHeader(name: string): boolean {
  return BLOCKED_HEADERS.includes(name.toLowerCase() as typeof BLOCKED_HEADERS[number]);
}

/**
 * Validates a header name including blocked header check.
 * @returns Error message or undefined if valid
 */
export function validateHttpHeader(name: string): string | undefined {
  const basicError = validateHttpHeaderName(name);
  if (basicError) return basicError;

  if (isBlockedHeader(name)) {
    return `"${name}" cannot be set (use storage state for cookies)`;
  }

  return undefined;
}
