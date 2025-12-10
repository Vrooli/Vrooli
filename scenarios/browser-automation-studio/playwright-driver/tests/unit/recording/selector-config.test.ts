/**
 * Tests for Selector Configuration
 *
 * These tests verify that the shared selector configuration is correct
 * and that the serialization for browser injection works properly.
 */
import {
  SELECTOR_STRATEGIES,
  TEST_ID_ATTRIBUTES,
  UNSTABLE_CLASS_PATTERNS,
  TEXT_CONTENT_TAGS,
  RELEVANT_ATTRIBUTES,
  SELECTOR_DEFAULTS,
  CONFIDENCE_SCORES,
  SPECIFICITY_SCORES,
  RECORDING_DEBOUNCE,
  patternsToRegExp,
  getUnstableClassPatterns,
  getDynamicIdPatterns,
  getSemanticClassPatterns,
  serializeConfigForBrowser,
} from '../../../src/recording/selector-config';

describe('Selector Configuration', () => {
  describe('SELECTOR_STRATEGIES', () => {
    it('should contain all selector strategy types in order', () => {
      expect(SELECTOR_STRATEGIES).toEqual([
        'data-testid',
        'id',
        'aria',
        'text',
        'data-attr',
        'css',
        'xpath',
      ]);
    });
  });

  describe('TEST_ID_ATTRIBUTES', () => {
    it('should contain common test ID attribute names', () => {
      expect(TEST_ID_ATTRIBUTES).toContain('data-testid');
      expect(TEST_ID_ATTRIBUTES).toContain('data-test-id');
      expect(TEST_ID_ATTRIBUTES).toContain('data-test');
      expect(TEST_ID_ATTRIBUTES).toContain('data-cy');
      expect(TEST_ID_ATTRIBUTES).toContain('data-qa');
    });
  });

  describe('UNSTABLE_CLASS_PATTERNS', () => {
    it('should have patterns for CSS-in-JS libraries', () => {
      expect(UNSTABLE_CLASS_PATTERNS.length).toBeGreaterThanOrEqual(7);

      // Verify patterns can be converted to RegExp
      const regexPatterns = patternsToRegExp(UNSTABLE_CLASS_PATTERNS);
      expect(regexPatterns.length).toBe(UNSTABLE_CLASS_PATTERNS.length);
    });

    it('should match CSS-in-JS class names when converted to RegExp', () => {
      const patterns = getUnstableClassPatterns();

      // Emotion/CSS-in-JS
      expect(patterns.some((p) => p.test('css-1abc23'))).toBe(true);

      // styled-components
      expect(patterns.some((p) => p.test('sc-bdVaJa'))).toBe(true);

      // CSS modules
      expect(patterns.some((p) => p.test('_module123'))).toBe(true);

      // Next.js styled-jsx
      expect(patterns.some((p) => p.test('jsx-123abc'))).toBe(true);

      // Should NOT match semantic classes
      expect(patterns.some((p) => p.test('btn-primary'))).toBe(false);
      expect(patterns.some((p) => p.test('nav-item'))).toBe(false);
    });
  });

  describe('DYNAMIC_ID_PATTERNS', () => {
    it('should match dynamic ID patterns when converted to RegExp', () => {
      const patterns = getDynamicIdPatterns();

      // Hex hash
      expect(patterns.some((p) => p.test('a1b2c3d4e5f6'))).toBe(true);

      // Pure numbers
      expect(patterns.some((p) => p.test('12345678'))).toBe(true);

      // React useId
      expect(patterns.some((p) => p.test(':r0:'))).toBe(true);
      expect(patterns.some((p) => p.test(':r123:'))).toBe(true);

      // Ember
      expect(patterns.some((p) => p.test('ember123'))).toBe(true);

      // Gatsby
      expect(patterns.some((p) => p.test('___gatsby'))).toBe(true);

      // Should NOT match semantic IDs
      expect(patterns.some((p) => p.test('main-content'))).toBe(false);
      expect(patterns.some((p) => p.test('user-profile'))).toBe(false);
    });
  });

  describe('SEMANTIC_CLASS_PATTERNS', () => {
    it('should match semantic class patterns when converted to RegExp', () => {
      const patterns = getSemanticClassPatterns();

      expect(patterns.some((p) => p.test('btn-primary'))).toBe(true);
      expect(patterns.some((p) => p.test('button-submit'))).toBe(true);
      expect(patterns.some((p) => p.test('nav-item'))).toBe(true);
      expect(patterns.some((p) => p.test('header-logo'))).toBe(true);
      expect(patterns.some((p) => p.test('form-group'))).toBe(true);
      expect(patterns.some((p) => p.test('input-field'))).toBe(true);

      // Should NOT match generic classes
      expect(patterns.some((p) => p.test('container'))).toBe(false);
      expect(patterns.some((p) => p.test('wrapper'))).toBe(false);
    });
  });

  describe('TEXT_CONTENT_TAGS', () => {
    it('should contain tags that commonly have meaningful text', () => {
      expect(TEXT_CONTENT_TAGS).toContain('button');
      expect(TEXT_CONTENT_TAGS).toContain('a');
      expect(TEXT_CONTENT_TAGS).toContain('label');
      expect(TEXT_CONTENT_TAGS).toContain('h1');
      expect(TEXT_CONTENT_TAGS).toContain('p');
    });

    it('should not contain non-text tags', () => {
      expect(TEXT_CONTENT_TAGS).not.toContain('input');
      expect(TEXT_CONTENT_TAGS).not.toContain('img');
      expect(TEXT_CONTENT_TAGS).not.toContain('script');
    });
  });

  describe('RELEVANT_ATTRIBUTES', () => {
    it('should contain attributes useful for element identification', () => {
      expect(RELEVANT_ATTRIBUTES).toContain('type');
      expect(RELEVANT_ATTRIBUTES).toContain('name');
      expect(RELEVANT_ATTRIBUTES).toContain('placeholder');
      expect(RELEVANT_ATTRIBUTES).toContain('href');
      expect(RELEVANT_ATTRIBUTES).toContain('role');
      expect(RELEVANT_ATTRIBUTES).toContain('aria-label');
    });
  });

  describe('SELECTOR_DEFAULTS', () => {
    it('should have reasonable default values', () => {
      expect(SELECTOR_DEFAULTS.maxCssDepth).toBeGreaterThan(0);
      expect(SELECTOR_DEFAULTS.maxCssDepth).toBeLessThanOrEqual(10);

      expect(SELECTOR_DEFAULTS.minConfidence).toBeGreaterThan(0);
      expect(SELECTOR_DEFAULTS.minConfidence).toBeLessThan(1);

      expect(SELECTOR_DEFAULTS.maxTextLength).toBeGreaterThan(50);
      expect(SELECTOR_DEFAULTS.minTextLength).toBeGreaterThan(0);
      expect(SELECTOR_DEFAULTS.maxTextLengthForSelector).toBeLessThanOrEqual(100);
    });
  });

  describe('CONFIDENCE_SCORES', () => {
    it('should have confidence scores in valid range', () => {
      Object.values(CONFIDENCE_SCORES).forEach((score) => {
        expect(score).toBeGreaterThan(0);
        expect(score).toBeLessThanOrEqual(1);
      });
    });

    it('should have data-testid as highest confidence', () => {
      expect(CONFIDENCE_SCORES.dataTestId).toBeGreaterThan(CONFIDENCE_SCORES.id);
      expect(CONFIDENCE_SCORES.dataTestId).toBeGreaterThan(CONFIDENCE_SCORES.ariaLabel);
      expect(CONFIDENCE_SCORES.dataTestId).toBeGreaterThan(CONFIDENCE_SCORES.cssPath);
    });

    it('should have xpath positional as lowest confidence', () => {
      expect(CONFIDENCE_SCORES.xpathPositional).toBeLessThan(CONFIDENCE_SCORES.dataTestId);
      expect(CONFIDENCE_SCORES.xpathPositional).toBeLessThan(CONFIDENCE_SCORES.id);
      expect(CONFIDENCE_SCORES.xpathPositional).toBeLessThan(CONFIDENCE_SCORES.cssPath);
    });
  });

  describe('SPECIFICITY_SCORES', () => {
    it('should have specificity scores in reasonable range', () => {
      Object.values(SPECIFICITY_SCORES).forEach((score) => {
        expect(score).toBeGreaterThanOrEqual(0);
        expect(score).toBeLessThanOrEqual(100);
      });
    });

    it('should have data-testid as highest specificity', () => {
      expect(SPECIFICITY_SCORES.dataTestId).toBe(100);
    });
  });

  describe('RECORDING_DEBOUNCE', () => {
    it('should have reasonable debounce timings', () => {
      expect(RECORDING_DEBOUNCE.input).toBeGreaterThan(100);
      expect(RECORDING_DEBOUNCE.input).toBeLessThanOrEqual(1000);

      expect(RECORDING_DEBOUNCE.scroll).toBeGreaterThan(50);
      expect(RECORDING_DEBOUNCE.scroll).toBeLessThanOrEqual(500);

      expect(RECORDING_DEBOUNCE.resize).toBeGreaterThan(100);
      expect(RECORDING_DEBOUNCE.resize).toBeLessThanOrEqual(500);
    });
  });

  describe('serializeConfigForBrowser', () => {
    it('should return a valid JavaScript string', () => {
      const serialized = serializeConfigForBrowser();

      // Should be a valid JavaScript object literal
      expect(serialized).toContain('TEST_ID_ATTRIBUTES');
      expect(serialized).toContain('UNSTABLE_CLASS_PATTERNS');
      expect(serialized).toContain('CONFIDENCE_SCORES');
      expect(serialized).toContain('new RegExp');
    });

    it('should produce evaluable JavaScript', () => {
      const serialized = serializeConfigForBrowser();

      // eslint-disable-next-line no-eval
      const config = eval(`(${serialized})`);

      expect(config.TEST_ID_ATTRIBUTES).toBeDefined();
      expect(config.UNSTABLE_CLASS_PATTERNS).toBeDefined();
      expect(config.CONFIDENCE_SCORES).toBeDefined();

      // Patterns should be actual RegExp objects
      expect(config.UNSTABLE_CLASS_PATTERNS[0]).toBeInstanceOf(RegExp);
      expect(config.DYNAMIC_ID_PATTERNS[0]).toBeInstanceOf(RegExp);
    });

    it('should preserve pattern matching behavior after serialization', () => {
      const serialized = serializeConfigForBrowser();

      // eslint-disable-next-line no-eval
      const config = eval(`(${serialized})`);

      // Test unstable class patterns
      const cssInJsClass = 'css-1abc23';
      expect(config.UNSTABLE_CLASS_PATTERNS.some((p: RegExp) => p.test(cssInJsClass))).toBe(true);

      // Test dynamic ID patterns
      const dynamicId = 'ember123';
      expect(config.DYNAMIC_ID_PATTERNS.some((p: RegExp) => p.test(dynamicId))).toBe(true);
    });
  });
});
