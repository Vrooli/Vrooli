import { describe, test, expect } from '@jest/globals';
import {
  parseUserAgent,
  generateClientHints,
  mergeClientHintsWithHeaders,
} from '@/browser-profile/client-hints';

// User agent strings from presets.ts
const USER_AGENTS = {
  'chrome-win':
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
  'chrome-mac':
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
  'chrome-linux':
    'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
  'firefox-win':
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0',
  'firefox-mac':
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0',
  'firefox-linux': 'Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0',
  'safari-mac':
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15',
  'edge-win':
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0',
};

describe('Client Hints - User Agent Parsing', () => {
  test('parses Chrome on Windows', () => {
    const parsed = parseUserAgent(USER_AGENTS['chrome-win']);

    expect(parsed.browser).toBe('chrome');
    expect(parsed.version).toBe('120');
    expect(parsed.platform).toBe('Windows');
    expect(parsed.isMobile).toBe(false);
  });

  test('parses Chrome on macOS', () => {
    const parsed = parseUserAgent(USER_AGENTS['chrome-mac']);

    expect(parsed.browser).toBe('chrome');
    expect(parsed.version).toBe('120');
    expect(parsed.platform).toBe('macOS');
    expect(parsed.isMobile).toBe(false);
  });

  test('parses Chrome on Linux', () => {
    const parsed = parseUserAgent(USER_AGENTS['chrome-linux']);

    expect(parsed.browser).toBe('chrome');
    expect(parsed.version).toBe('120');
    expect(parsed.platform).toBe('Linux');
    expect(parsed.isMobile).toBe(false);
  });

  test('parses Edge on Windows', () => {
    const parsed = parseUserAgent(USER_AGENTS['edge-win']);

    expect(parsed.browser).toBe('edge');
    expect(parsed.version).toBe('120');
    expect(parsed.platform).toBe('Windows');
    expect(parsed.isMobile).toBe(false);
  });

  test('parses Firefox on Windows', () => {
    const parsed = parseUserAgent(USER_AGENTS['firefox-win']);

    expect(parsed.browser).toBe('firefox');
    expect(parsed.version).toBe('121');
    expect(parsed.platform).toBe('Windows');
  });

  test('parses Firefox on macOS', () => {
    const parsed = parseUserAgent(USER_AGENTS['firefox-mac']);

    expect(parsed.browser).toBe('firefox');
    expect(parsed.platform).toBe('macOS');
  });

  test('parses Firefox on Linux', () => {
    const parsed = parseUserAgent(USER_AGENTS['firefox-linux']);

    expect(parsed.browser).toBe('firefox');
    expect(parsed.platform).toBe('Linux');
  });

  test('parses Safari on macOS', () => {
    const parsed = parseUserAgent(USER_AGENTS['safari-mac']);

    expect(parsed.browser).toBe('safari');
    expect(parsed.version).toBe('17');
    expect(parsed.platform).toBe('macOS');
  });

  test('detects mobile user agents', () => {
    const mobileUA =
      'Mozilla/5.0 (Linux; Android 10; SM-G973F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36';
    const parsed = parseUserAgent(mobileUA);

    expect(parsed.browser).toBe('chrome');
    expect(parsed.platform).toBe('Android');
    expect(parsed.isMobile).toBe(true);
  });

  test('handles unknown user agents gracefully', () => {
    const parsed = parseUserAgent('SomeUnknownBot/1.0');

    expect(parsed.browser).toBe('unknown');
    expect(parsed.version).toBe('120'); // Default
    expect(parsed.platform).toBe('Unknown');
    expect(parsed.isMobile).toBe(false);
  });
});

describe('Client Hints - Header Generation', () => {
  test('generates Client Hints for Chrome on Windows', () => {
    const hints = generateClientHints(USER_AGENTS['chrome-win']);

    expect(hints).not.toBeNull();
    expect(hints!['sec-ch-ua']).toContain('Google Chrome');
    expect(hints!['sec-ch-ua']).toContain('v="120"');
    expect(hints!['sec-ch-ua']).toContain('Chromium');
    expect(hints!['sec-ch-ua']).toContain('Not_A Brand');
    expect(hints!['sec-ch-ua-mobile']).toBe('?0');
    expect(hints!['sec-ch-ua-platform']).toBe('"Windows"');
  });

  test('generates Client Hints for Chrome on macOS', () => {
    const hints = generateClientHints(USER_AGENTS['chrome-mac']);

    expect(hints).not.toBeNull();
    expect(hints!['sec-ch-ua']).toContain('Google Chrome');
    expect(hints!['sec-ch-ua-platform']).toBe('"macOS"');
  });

  test('generates Client Hints for Chrome on Linux', () => {
    const hints = generateClientHints(USER_AGENTS['chrome-linux']);

    expect(hints).not.toBeNull();
    expect(hints!['sec-ch-ua-platform']).toBe('"Linux"');
  });

  test('generates Client Hints for Edge on Windows', () => {
    const hints = generateClientHints(USER_AGENTS['edge-win']);

    expect(hints).not.toBeNull();
    expect(hints!['sec-ch-ua']).toContain('Microsoft Edge');
    expect(hints!['sec-ch-ua']).toContain('v="120"');
    expect(hints!['sec-ch-ua']).toContain('Chromium');
    expect(hints!['sec-ch-ua-platform']).toBe('"Windows"');
  });

  test('returns null for Firefox (no Client Hints)', () => {
    const hints = generateClientHints(USER_AGENTS['firefox-win']);
    expect(hints).toBeNull();
  });

  test('returns null for Safari (no Client Hints)', () => {
    const hints = generateClientHints(USER_AGENTS['safari-mac']);
    expect(hints).toBeNull();
  });

  test('returns null for unknown browsers', () => {
    const hints = generateClientHints('SomeUnknownBot/1.0');
    expect(hints).toBeNull();
  });

  test('generates mobile hints correctly', () => {
    const mobileUA =
      'Mozilla/5.0 (Linux; Android 10; SM-G973F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36';
    const hints = generateClientHints(mobileUA);

    expect(hints).not.toBeNull();
    expect(hints!['sec-ch-ua-mobile']).toBe('?1');
    expect(hints!['sec-ch-ua-platform']).toBe('"Android"');
  });
});

describe('Client Hints - Header Merging', () => {
  test('merges Client Hints with empty extra headers', () => {
    const hints = generateClientHints(USER_AGENTS['chrome-win']);
    const merged = mergeClientHintsWithHeaders(hints, {});

    expect(merged['sec-ch-ua']).toContain('Google Chrome');
    expect(merged['sec-ch-ua-mobile']).toBe('?0');
    expect(merged['sec-ch-ua-platform']).toBe('"Windows"');
  });

  test('user headers override Client Hints', () => {
    const hints = generateClientHints(USER_AGENTS['chrome-win']);
    const userHeaders = {
      'sec-ch-ua': '"CustomBrowser";v="99"',
      'x-custom-header': 'custom-value',
    };
    const merged = mergeClientHintsWithHeaders(hints, userHeaders);

    // User's sec-ch-ua should override
    expect(merged['sec-ch-ua']).toBe('"CustomBrowser";v="99"');
    // Client Hints that weren't overridden should still be there
    expect(merged['sec-ch-ua-mobile']).toBe('?0');
    // User's custom header should be included
    expect(merged['x-custom-header']).toBe('custom-value');
  });

  test('preserves user headers when no Client Hints', () => {
    const userHeaders = {
      authorization: 'Bearer token123',
      'x-api-key': 'key456',
    };
    const merged = mergeClientHintsWithHeaders(null, userHeaders);

    expect(merged).toEqual(userHeaders);
  });

  test('handles undefined extra headers', () => {
    const hints = generateClientHints(USER_AGENTS['chrome-win']);
    const merged = mergeClientHintsWithHeaders(hints, undefined);

    expect(merged['sec-ch-ua']).toContain('Google Chrome');
    expect(merged['sec-ch-ua-mobile']).toBe('?0');
  });

  test('returns empty object when no hints and no headers', () => {
    const merged = mergeClientHintsWithHeaders(null, undefined);
    expect(merged).toEqual({});
  });

  test('returns empty object when no hints and empty headers', () => {
    const merged = mergeClientHintsWithHeaders(null, {});
    expect(merged).toEqual({});
  });
});

describe('Client Hints - All Preset User Agents', () => {
  test('chrome-win generates Windows Client Hints', () => {
    const hints = generateClientHints(USER_AGENTS['chrome-win']);
    expect(hints).not.toBeNull();
    expect(hints!['sec-ch-ua']).toContain('Google Chrome');
    expect(hints!['sec-ch-ua-platform']).toBe('"Windows"');
  });

  test('chrome-mac generates macOS Client Hints', () => {
    const hints = generateClientHints(USER_AGENTS['chrome-mac']);
    expect(hints).not.toBeNull();
    expect(hints!['sec-ch-ua-platform']).toBe('"macOS"');
  });

  test('chrome-linux generates Linux Client Hints', () => {
    const hints = generateClientHints(USER_AGENTS['chrome-linux']);
    expect(hints).not.toBeNull();
    expect(hints!['sec-ch-ua-platform']).toBe('"Linux"');
  });

  test('edge-win generates Edge Client Hints', () => {
    const hints = generateClientHints(USER_AGENTS['edge-win']);
    expect(hints).not.toBeNull();
    expect(hints!['sec-ch-ua']).toContain('Microsoft Edge');
  });

  test('firefox-win returns null (no Client Hints)', () => {
    expect(generateClientHints(USER_AGENTS['firefox-win'])).toBeNull();
  });

  test('firefox-mac returns null (no Client Hints)', () => {
    expect(generateClientHints(USER_AGENTS['firefox-mac'])).toBeNull();
  });

  test('firefox-linux returns null (no Client Hints)', () => {
    expect(generateClientHints(USER_AGENTS['firefox-linux'])).toBeNull();
  });

  test('safari-mac returns null (no Client Hints)', () => {
    expect(generateClientHints(USER_AGENTS['safari-mac'])).toBeNull();
  });
});
