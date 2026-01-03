/**
 * CAPTCHA Detection Module
 *
 * STABILITY: STABLE CORE
 *
 * Programmatic detection of common CAPTCHA and human verification systems.
 * Runs BEFORE the AI decides, providing early detection of intervention needs.
 *
 * Detected systems:
 * - reCAPTCHA (Google)
 * - hCaptcha
 * - Cloudflare Turnstile
 * - Generic verification text patterns
 */

import type { Page } from 'rebrowser-playwright';

/**
 * Result of CAPTCHA detection.
 */
export interface CaptchaDetectionResult {
  /** Whether a CAPTCHA or verification was detected */
  detected: boolean;
  /** Type of CAPTCHA detected */
  type: 'recaptcha' | 'hcaptcha' | 'cloudflare_turnstile' | 'generic_verification' | null;
  /** Detection confidence level */
  confidence: 'high' | 'medium' | 'low';
  /** Human-readable reason for detection */
  reason: string;
  /** Instructions for the user */
  instructions: string;
  /** Element selector if found */
  selector?: string;
}

/**
 * Internal detection pattern configuration.
 */
interface DetectionPattern {
  type: NonNullable<CaptchaDetectionResult['type']>;
  selectors: string[];
  iframeSrcPatterns: RegExp[];
  textPatterns: RegExp[];
  instructions: string;
}

/**
 * Detection patterns for known CAPTCHA systems.
 */
const DETECTION_PATTERNS: DetectionPattern[] = [
  {
    type: 'recaptcha',
    selectors: [
      'iframe[src*="recaptcha"]',
      '.g-recaptcha',
      '[data-sitekey]',
      '#recaptcha',
      '[class*="recaptcha"]',
    ],
    iframeSrcPatterns: [
      /google\.com\/recaptcha/i,
      /recaptcha\.net/i,
    ],
    textPatterns: [],
    instructions: 'Please complete the reCAPTCHA verification, then click "I\'m Done".',
  },
  {
    type: 'hcaptcha',
    selectors: [
      'iframe[src*="hcaptcha"]',
      '.h-captcha',
      '[data-hcaptcha-sitekey]',
      '[class*="hcaptcha"]',
    ],
    iframeSrcPatterns: [
      /hcaptcha\.com/i,
    ],
    textPatterns: [],
    instructions: 'Please complete the hCaptcha verification, then click "I\'m Done".',
  },
  {
    type: 'cloudflare_turnstile',
    selectors: [
      'iframe[src*="challenges.cloudflare.com"]',
      '.cf-turnstile',
      '[data-cf-turnstile-sitekey]',
      '[data-cf-beacon]',
      '[class*="turnstile"]',
    ],
    iframeSrcPatterns: [
      /challenges\.cloudflare\.com/i,
      /turnstile/i,
    ],
    textPatterns: [
      /checking your browser/i,
      /verifying you are human/i,
      /just a moment/i,
    ],
    instructions: 'Please complete the Cloudflare verification, then click "I\'m Done".',
  },
  {
    type: 'generic_verification',
    selectors: [],
    iframeSrcPatterns: [],
    textPatterns: [
      /verify you're human/i,
      /are you a robot/i,
      /prove you're not a robot/i,
      /human verification/i,
      /security check/i,
      /please verify/i,
      /i'm not a robot/i,
      /confirm you're human/i,
      /complete the security check/i,
      /access denied.*bot/i,
    ],
    instructions: 'Please complete the verification challenge, then click "I\'m Done".',
  },
];

/**
 * Internal detection result from browser context.
 */
interface BrowserDetectionResult {
  type: string;
  confidence: 'high' | 'medium' | 'low';
  selector?: string;
  matchedPattern: string;
}

/**
 * Detect CAPTCHAs and human verification challenges on the page.
 *
 * @param page - Playwright Page to analyze
 * @returns Detection result with type, confidence, and instructions
 */
export async function detectCaptcha(page: Page): Promise<CaptchaDetectionResult> {
  try {
    // Run detection in browser context
    const results = await page.evaluate((patterns: DetectionPattern[]) => {
      /**
       * Check if an element is visible to users.
       *
       * Catches common hiding techniques:
       * - display: none (zero bounding rect)
       * - width/height: 0 (zero bounding rect)
       * - visibility: hidden (invisible but occupies space)
       * - opacity: 0 (fully transparent)
       * - Off-screen positioning (left: -9999px pattern)
       * - Hidden parent elements (child rect becomes zero)
       */
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      function isElementVisible(element: any): boolean {
        // Bounding rect catches: display:none, zero size, hidden parents
        const rect = element.getBoundingClientRect();
        if (rect.width <= 0 || rect.height <= 0) {
          return false;
        }

        // Computed style catches: visibility:hidden, opacity:0
        const style = window.getComputedStyle(element);
        if (style.visibility === 'hidden') {
          return false;
        }
        if (style.opacity === '0') {
          return false;
        }

        // Off-screen check catches: position:absolute; left:-9999px
        // Note: Don't check right/bottom edges - element could legitimately be below fold
        if (rect.right <= 0 || rect.bottom <= 0) {
          return false;
        }

        return true;
      }

      const detections: BrowserDetectionResult[] = [];

      for (const pattern of patterns) {
        // Check selectors (highest confidence)
        for (const selector of pattern.selectors) {
          try {
            const element = document.querySelector(selector);
            if (element && isElementVisible(element)) {
              detections.push({
                type: pattern.type,
                confidence: 'high',
                selector,
                matchedPattern: `selector: ${selector}`,
              });
            }
          } catch {
            // Invalid selector, skip
          }
        }

        // Check iframe sources (high confidence)
        const iframes = document.querySelectorAll('iframe');
        for (const iframe of iframes) {
          // Skip hidden iframes (e.g., preloaded captchas)
          if (!isElementVisible(iframe)) continue;

          const src = iframe.getAttribute('src') || '';
          for (const srcPattern of pattern.iframeSrcPatterns) {
            if (srcPattern.test(src)) {
              detections.push({
                type: pattern.type,
                confidence: 'high',
                selector: `iframe[src="${src}"]`,
                matchedPattern: `iframe src: ${srcPattern.source}`,
              });
            }
          }
        }

        // Check text patterns (lower confidence - could be false positive)
        const bodyText = document.body?.innerText || '';
        for (const textPattern of pattern.textPatterns) {
          if (textPattern.test(bodyText)) {
            detections.push({
              type: pattern.type,
              confidence: pattern.type === 'generic_verification' ? 'low' : 'medium',
              matchedPattern: `text: ${textPattern.source}`,
            });
          }
        }
      }

      return detections;
    }, DETECTION_PATTERNS);

    if (results.length === 0) {
      return {
        detected: false,
        type: null,
        confidence: 'low',
        reason: 'No CAPTCHA detected',
        instructions: '',
      };
    }

    // Sort by confidence and pick the best match
    const confidenceOrder = { high: 0, medium: 1, low: 2 };
    const sorted = results.sort((a, b) =>
      confidenceOrder[a.confidence] - confidenceOrder[b.confidence]
    );

    const best = sorted[0];
    const pattern = DETECTION_PATTERNS.find(p => p.type === best.type)!;

    return {
      detected: true,
      type: best.type as CaptchaDetectionResult['type'],
      confidence: best.confidence,
      reason: `${formatCaptchaType(best.type)} detected (${best.matchedPattern})`,
      instructions: pattern.instructions,
      selector: best.selector,
    };
  } catch (error) {
    // If detection fails, assume no CAPTCHA (don't block navigation)
    return {
      detected: false,
      type: null,
      confidence: 'low',
      reason: `Detection error: ${error instanceof Error ? error.message : 'Unknown error'}`,
      instructions: '',
    };
  }
}

/**
 * Format CAPTCHA type for display.
 */
function formatCaptchaType(type: string): string {
  switch (type) {
    case 'recaptcha':
      return 'reCAPTCHA';
    case 'hcaptcha':
      return 'hCaptcha';
    case 'cloudflare_turnstile':
      return 'Cloudflare Turnstile';
    case 'generic_verification':
      return 'Human verification';
    default:
      return type;
  }
}

/**
 * Create a mock detector for testing.
 */
export function createMockCaptchaDetector(
  mockResult: CaptchaDetectionResult
): { detect: (page: Page) => Promise<CaptchaDetectionResult> } {
  return {
    detect: async () => mockResult,
  };
}

/**
 * No CAPTCHA detected result (for reuse).
 */
export const NO_CAPTCHA_RESULT: CaptchaDetectionResult = {
  detected: false,
  type: null,
  confidence: 'low',
  reason: 'No CAPTCHA detected',
  instructions: '',
};
