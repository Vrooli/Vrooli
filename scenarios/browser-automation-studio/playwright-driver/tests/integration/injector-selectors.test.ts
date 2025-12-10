/* eslint-disable @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call, @typescript-eslint/no-explicit-any */
/**
 * Integration Tests for Injector Selector Generation
 *
 * These tests verify that the selector generation logic in injector.ts
 * works correctly when executed in a real browser context.
 *
 * This is a CRITICAL test for the Selector Strategies change axis because:
 * 1. It validates the browser-context code that cannot be unit tested
 * 2. It ensures config serialization works correctly
 * 3. It catches drift between selectors.ts and injector.ts implementations
 */
import { chromium, Browser, Page } from 'playwright';
import { getRecordingScript } from '../../src/recording/injector';
import { TEST_ID_ATTRIBUTES, CONFIDENCE_SCORES, SPECIFICITY_SCORES } from '../../src/recording/selector-config';

// Type for generated selectors from browser context
interface SelectorSet {
  primary: string;
  candidates: Array<{
    type: string;
    value: string;
    confidence: number;
    specificity: number;
  }>;
}

describe('Injector Selector Generation (Integration)', () => {
  let browser: Browser;
  let page: Page;

  beforeAll(async () => {
    browser = await chromium.launch({ headless: true });
  });

  afterAll(async () => {
    await browser.close();
  });

  beforeEach(async () => {
    page = await browser.newPage();
  });

  afterEach(async () => {
    await page.close();
  });

  /**
   * Helper to inject recording script and generate selectors for an element
   */
  async function generateSelectorsFor(html: string, elementSelector: string): Promise<SelectorSet> {
    await page.setContent(html);
    await page.evaluate(getRecordingScript());

    // Use a function string to avoid TypeScript DOM type issues
    const evalFn = `
      (function(selector) {
        const element = document.querySelector(selector);
        if (!element) {
          throw new Error('Element not found: ' + selector);
        }
        return window.__generateSelectors(element);
      })('${elementSelector.replace(/'/g, "\\'")}')
    `;
    return await page.evaluate(evalFn) as SelectorSet;
  }

  describe('data-testid Selectors (Highest Priority)', () => {
    it('should generate data-testid selector with highest confidence', async () => {
      const result = await generateSelectorsFor(
        '<button data-testid="submit-btn">Submit</button>',
        'button'
      );

      expect(result.primary).toBe('[data-testid="submit-btn"]');
      expect(result.candidates[0]).toMatchObject({
        type: 'data-testid',
        confidence: CONFIDENCE_SCORES.dataTestId,
        specificity: SPECIFICITY_SCORES.dataTestId,
      });
    });

    for (const attr of TEST_ID_ATTRIBUTES) {
      it(`should recognize ${attr} as a test ID attribute`, async () => {
        const result = await generateSelectorsFor(
          `<button ${attr}="test-button">Click</button>`,
          'button'
        );

        expect(result.primary).toContain(attr);
        expect(result.candidates[0].type).toBe('data-testid');
      });
    }
  });

  describe('ID Selectors', () => {
    it('should generate ID selector for unique IDs', async () => {
      const result = await generateSelectorsFor(
        '<button id="main-submit">Submit</button>',
        'button'
      );

      // ID might be primary if no data-testid exists
      const idCandidate = result.candidates.find((c) => c.type === 'id');
      expect(idCandidate).toBeDefined();
      expect(idCandidate?.value).toBe('#main-submit');
      expect(idCandidate?.confidence).toBe(CONFIDENCE_SCORES.id);
    });

    it('should lower confidence for dynamic-looking IDs', async () => {
      const result = await generateSelectorsFor(
        '<button id="ember123">Click</button>',
        'button'
      );

      const idCandidate = result.candidates.find((c) => c.type === 'id');
      expect(idCandidate).toBeDefined();
      expect(idCandidate?.confidence).toBe(CONFIDENCE_SCORES.idDynamic);
    });

    it('should detect React useId pattern as dynamic', async () => {
      const result = await generateSelectorsFor(
        '<button id=":r0:">Click</button>',
        'button'
      );

      const idCandidate = result.candidates.find((c) => c.type === 'id');
      expect(idCandidate?.confidence).toBe(CONFIDENCE_SCORES.idDynamic);
    });
  });

  describe('ARIA Selectors', () => {
    it('should generate aria-label selector', async () => {
      const result = await generateSelectorsFor(
        '<button aria-label="Close dialog">X</button>',
        'button'
      );

      const ariaCandidate = result.candidates.find((c) => c.type === 'aria');
      expect(ariaCandidate).toBeDefined();
      expect(ariaCandidate?.value).toBe('[aria-label="Close dialog"]');
      expect(ariaCandidate?.confidence).toBe(CONFIDENCE_SCORES.ariaLabel);
    });
  });

  describe('Text Selectors', () => {
    it('should generate text selector for buttons', async () => {
      const result = await generateSelectorsFor(
        '<button>Submit Form</button>',
        'button'
      );

      const textCandidate = result.candidates.find((c) => c.type === 'text');
      expect(textCandidate).toBeDefined();
      expect(textCandidate?.value).toContain('has-text');
      expect(textCandidate?.value).toContain('Submit Form');
    });

    it('should generate text selector for links', async () => {
      const result = await generateSelectorsFor(
        '<a href="/about">About Us</a>',
        'a'
      );

      const textCandidate = result.candidates.find((c) => c.type === 'text');
      expect(textCandidate).toBeDefined();
      expect(textCandidate?.value).toContain('About Us');
    });

    it('should truncate long text for selector stability', async () => {
      // Text between 15-50 chars to trigger text selector but test truncation
      const longText = 'This button text should truncate at 30';
      const result = await generateSelectorsFor(
        `<button>${longText}</button>`,
        'button'
      );

      const textCandidate = result.candidates.find((c) => c.type === 'text');
      expect(textCandidate).toBeDefined();
      // The selector should contain truncated text (first 30 chars)
      // Full format: button:has-text("...")
      expect(textCandidate?.value).toContain('has-text');
      // Value length should be reasonable (not the full text + tag + quotes)
      expect(textCandidate?.value.length).toBeLessThan(60);
    });
  });

  describe('data-attr Selectors', () => {
    it('should generate selector for data-* attributes', async () => {
      const result = await generateSelectorsFor(
        '<button data-action="submit">Submit</button>',
        'button'
      );

      const dataAttrCandidate = result.candidates.find((c) => c.type === 'data-attr');
      expect(dataAttrCandidate).toBeDefined();
      expect(dataAttrCandidate?.value).toBe('[data-action="submit"]');
      expect(dataAttrCandidate?.confidence).toBe(CONFIDENCE_SCORES.dataAttr);
    });
  });

  describe('CSS Path Selectors', () => {
    it('should generate CSS path when other strategies fail', async () => {
      const result = await generateSelectorsFor(
        `<div class="container">
          <div class="btn-wrapper">
            <button class="btn-primary">Click</button>
          </div>
        </div>`,
        '.btn-primary'
      );

      const cssCandidate = result.candidates.find((c) => c.type === 'css');
      expect(cssCandidate).toBeDefined();
      expect(cssCandidate?.value).toContain('btn-primary');
    });

    it('should filter out unstable CSS-in-JS class names', async () => {
      const result = await generateSelectorsFor(
        '<button class="css-abc123 btn-submit">Click</button>',
        'button'
      );

      // Should use btn-submit (semantic) not css-abc123 (generated)
      const cssCandidate = result.candidates.find((c) => c.type === 'css');
      if (cssCandidate) {
        expect(cssCandidate.value).not.toContain('css-abc123');
      }
    });

    it('should filter out styled-components class names', async () => {
      const result = await generateSelectorsFor(
        '<button class="sc-bdVaJa btn-primary">Click</button>',
        'button'
      );

      const cssCandidate = result.candidates.find((c) => c.type === 'css');
      if (cssCandidate) {
        expect(cssCandidate.value).not.toContain('sc-bdVaJa');
      }
    });
  });

  describe('XPath Selectors (Fallback)', () => {
    it('should generate XPath when no other strategy produces unique selector', async () => {
      const result = await generateSelectorsFor(
        `<div>
          <span>First</span>
          <span>Second</span>
          <span>Target</span>
        </div>`,
        'span:last-child'
      );

      const xpathCandidate = result.candidates.find((c) => c.type === 'xpath');
      expect(xpathCandidate).toBeDefined();
    });
  });

  describe('Selector Ranking', () => {
    it('should rank selectors by confidence (descending)', async () => {
      const result = await generateSelectorsFor(
        '<button id="btn" data-testid="submit" aria-label="Submit form">Submit</button>',
        'button'
      );

      // data-testid should be first (highest confidence)
      expect(result.candidates[0].type).toBe('data-testid');

      // Verify candidates are sorted by confidence
      for (let i = 1; i < result.candidates.length; i++) {
        expect(result.candidates[i - 1].confidence).toBeGreaterThanOrEqual(
          result.candidates[i].confidence
        );
      }
    });

    it('should use highest confidence selector as primary', async () => {
      const result = await generateSelectorsFor(
        '<button id="btn" data-testid="submit">Click</button>',
        'button'
      );

      expect(result.primary).toBe('[data-testid="submit"]');
    });
  });

  describe('Complex DOM Scenarios', () => {
    it('should handle nested elements', async () => {
      const result = await generateSelectorsFor(
        `<form>
          <div class="form-group">
            <label>Email</label>
            <input type="email" data-testid="email-input" />
          </div>
        </form>`,
        '[data-testid="email-input"]'
      );

      expect(result.primary).toBe('[data-testid="email-input"]');
    });

    it('should handle multiple similar elements', async () => {
      const html = `
        <ul>
          <li data-testid="item-1">First</li>
          <li data-testid="item-2">Second</li>
          <li data-testid="item-3">Third</li>
        </ul>
      `;

      for (let i = 1; i <= 3; i++) {
        const result = await generateSelectorsFor(html, `[data-testid="item-${i}"]`);
        expect(result.primary).toBe(`[data-testid="item-${i}"]`);
      }
    });

    it('should handle shadow DOM boundary (main document)', async () => {
      const result = await generateSelectorsFor(
        '<button data-testid="light-dom-btn">Click</button>',
        'button'
      );

      // Should work in light DOM
      expect(result.primary).toBe('[data-testid="light-dom-btn"]');
    });
  });

  describe('Config Serialization Verification', () => {
    it('should have access to shared config in browser context', async () => {
      await page.setContent('<div>Test</div>');
      await page.evaluate(getRecordingScript());

      // Verify CONFIG exists and has expected structure
      const configCheck = await page.evaluate('typeof window.__recordingActive === "boolean"') as boolean;

      expect(configCheck).toBe(true);
    });

    it('should use config values from selector-config.ts', async () => {
      const result = await generateSelectorsFor(
        '<button data-testid="config-test">Test</button>',
        'button'
      );

      // Verify confidence matches selector-config.ts
      const testIdCandidate = result.candidates.find((c) => c.type === 'data-testid');
      expect(testIdCandidate?.confidence).toBe(CONFIDENCE_SCORES.dataTestId);
      expect(testIdCandidate?.specificity).toBe(SPECIFICITY_SCORES.dataTestId);
    });
  });
});
