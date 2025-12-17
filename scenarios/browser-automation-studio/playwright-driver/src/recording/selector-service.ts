/**
 * Selector Service
 *
 * Consolidated service for selector validation, scoring, and utilities.
 *
 * DESIGN:
 * Previously, selector logic was scattered across:
 * - controller.ts (validateSelector method)
 * - selector-config.ts (configuration constants)
 * - action-types.ts (calculateActionConfidence)
 * - action-executor.ts (inline validation in each executor)
 *
 * This service consolidates selector operations into a single, reusable module:
 * - Validation: Check if selector matches exactly one element
 * - Scoring: Calculate confidence and specificity scores
 * - Configuration: Access to selector strategy settings
 * - Utilities: XPath detection, pattern matching
 *
 * USAGE:
 * ```typescript
 * const selectorService = new SelectorService(page, config);
 *
 * // Validate a selector
 * const validation = await selectorService.validate(selector);
 * if (!validation.valid) {
 *   // Handle error: matchCount === 0 (not found) or > 1 (ambiguous)
 * }
 *
 * // Calculate confidence
 * const confidence = selectorService.calculateConfidence(selector, candidates);
 * ```
 *
 * @module recording/selector-service
 */

import type { Page } from 'playwright';
import type { Config } from '../config';
import {
  CONFIDENCE_SCORES,
  SPECIFICITY_SCORES,
  getDynamicIdPatterns,
  getUnstableClassPatterns,
  getSemanticClassPatterns,
  TEST_ID_ATTRIBUTES,
  SELECTOR_DEFAULTS,
  adjustConfidenceForStrongType,
} from './selector-config';

// =============================================================================
// Types
// =============================================================================

/**
 * Result of selector validation.
 */
export interface SelectorValidation {
  /** Whether selector is valid (matches exactly 1 element) */
  valid: boolean;
  /** Number of elements matching the selector */
  matchCount: number;
  /** The selector that was validated */
  selector: string;
  /** Error message if validation failed */
  error?: string;
}

/**
 * Selector candidate with confidence metadata.
 */
export interface SelectorCandidate {
  type: string;
  value: string;
  confidence?: number;
  specificity?: number;
}

/**
 * Options for selector validation.
 */
export interface ValidationOptions {
  /** Timeout for selector evaluation (ms) */
  timeout?: number;
  /** Whether to wait for selector before validating */
  waitForSelector?: boolean;
}

// =============================================================================
// SelectorService
// =============================================================================

/**
 * Service for selector validation and scoring.
 *
 * Provides a unified interface for all selector-related operations,
 * consolidating logic that was previously scattered across multiple modules.
 */
export class SelectorService {
  private readonly page: Page;
  private readonly config: Config;
  private readonly dynamicIdPatterns: RegExp[];
  private readonly unstableClassPatterns: RegExp[];
  private readonly semanticClassPatterns: RegExp[];

  constructor(page: Page, config: Config) {
    this.page = page;
    this.config = config;
    // Pre-compile patterns for performance
    this.dynamicIdPatterns = getDynamicIdPatterns();
    this.unstableClassPatterns = getUnstableClassPatterns();
    this.semanticClassPatterns = getSemanticClassPatterns();
  }

  // ===========================================================================
  // Validation
  // ===========================================================================

  /**
   * Validate a selector on the current page.
   *
   * Checks if the selector matches exactly one element. Returns validation
   * result with match count and any errors.
   *
   * @param selector - CSS selector or XPath expression
   * @param options - Validation options
   * @returns Validation result with valid flag, matchCount, and error
   */
  async validate(selector: string, options?: ValidationOptions): Promise<SelectorValidation> {
    try {
      if (this.isXPath(selector)) {
        return await this.validateXPath(selector);
      } else {
        return await this.validateCSS(selector, options);
      }
    } catch (error) {
      return {
        valid: false,
        matchCount: 0,
        selector,
        error: error instanceof Error ? error.message : String(error),
      };
    }
  }

  /**
   * Validate an XPath selector.
   */
  private async validateXPath(selector: string): Promise<SelectorValidation> {
    const count = await this.page.evaluate<number>(`
      (function() {
        try {
          const result = document.evaluate(
            ${JSON.stringify(selector)},
            document,
            null,
            XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
          );
          return result.snapshotLength;
        } catch {
          return -1;
        }
      })()
    `);

    if (count === -1) {
      return { valid: false, matchCount: 0, selector, error: 'Invalid XPath expression' };
    }

    return {
      valid: count === 1,
      matchCount: count,
      selector,
    };
  }

  /**
   * Validate a CSS selector.
   */
  private async validateCSS(selector: string, options?: ValidationOptions): Promise<SelectorValidation> {
    const locator = this.page.locator(selector);

    if (options?.waitForSelector) {
      try {
        await locator.first().waitFor({
          timeout: options.timeout ?? this.config.execution.defaultTimeoutMs,
          state: 'attached',
        });
      } catch {
        // Element didn't appear - will be reported as matchCount: 0
      }
    }

    const count = await locator.count();
    return {
      valid: count === 1,
      matchCount: count,
      selector,
    };
  }

  /**
   * Check if a string is a valid selector that can be evaluated.
   * Does not check if it matches any elements - just syntax validity.
   */
  async isValidSyntax(selector: string): Promise<boolean> {
    try {
      if (this.isXPath(selector)) {
        const result = await this.page.evaluate<boolean>(`
          (function() {
            try {
              document.evaluate(${JSON.stringify(selector)}, document, null, XPathResult.ANY_TYPE, null);
              return true;
            } catch {
              return false;
            }
          })()
        `);
        return result;
      } else {
        // For CSS, try to create a locator - throws if invalid
        await this.page.locator(selector).count();
        return true;
      }
    } catch {
      return false;
    }
  }

  // ===========================================================================
  // Scoring
  // ===========================================================================

  /**
   * Calculate confidence score for a selector.
   *
   * Confidence indicates how reliable a selector is likely to be for
   * replay. Higher scores mean more stable selectors.
   *
   * @param selector - The selector string
   * @param candidates - Optional array of selector candidates with metadata
   * @returns Confidence score 0-1
   */
  calculateConfidence(
    selector: string,
    candidates?: SelectorCandidate[]
  ): number {
    // If we have candidates with confidence, use the primary's confidence
    if (candidates && candidates.length > 0) {
      const primary = candidates.find(c => c.value === selector);
      if (primary?.confidence !== undefined) {
        return this.adjustConfidence(primary.type, primary.confidence);
      }
    }

    // Otherwise calculate based on selector pattern
    return this.calculateConfidenceFromPattern(selector);
  }

  /**
   * Calculate confidence based on selector pattern analysis.
   */
  private calculateConfidenceFromPattern(selector: string): number {
    // Check for test ID attributes
    for (const attr of TEST_ID_ATTRIBUTES) {
      if (selector.includes(`[${attr}=`) || selector.includes(`[${attr}]`)) {
        return CONFIDENCE_SCORES.dataTestId;
      }
    }

    // Check for ID selector
    if (selector.match(/^#[a-zA-Z][a-zA-Z0-9_-]*$/)) {
      const id = selector.slice(1);
      // Check if ID appears dynamic
      if (this.dynamicIdPatterns.some(p => p.test(id))) {
        return CONFIDENCE_SCORES.idDynamic;
      }
      return CONFIDENCE_SCORES.id;
    }

    // Check for ARIA selectors
    if (selector.includes('[aria-label=') || selector.includes('[role=')) {
      return CONFIDENCE_SCORES.ariaLabel;
    }

    // Check for data attributes
    if (selector.includes('[data-')) {
      return CONFIDENCE_SCORES.dataAttr;
    }

    // Check for text-based selectors
    if (selector.includes(':has-text(') || selector.includes('text=')) {
      const textMatch = selector.match(/:has-text\("([^"]+)"\)/) ||
                        selector.match(/text="([^"]+)"/);
      if (textMatch && textMatch[1].length > 20) {
        return CONFIDENCE_SCORES.textLong;
      }
      return CONFIDENCE_SCORES.textShort;
    }

    // Check for nth-child (less stable)
    if (selector.includes(':nth-child(') || selector.includes(':nth-of-type(')) {
      return CONFIDENCE_SCORES.cssNthChild;
    }

    // XPath
    if (this.isXPath(selector)) {
      if (selector.includes('text()') || selector.includes('contains(')) {
        return CONFIDENCE_SCORES.xpathText;
      }
      return CONFIDENCE_SCORES.xpathPositional;
    }

    // Default CSS path confidence
    return CONFIDENCE_SCORES.cssPath;
  }

  /**
   * Adjust confidence based on selector type and known patterns.
   * Delegates to the shared adjustConfidenceForStrongType function
   * from selector-config.ts (single source of truth).
   */
  private adjustConfidence(type: string, baseConfidence: number): number {
    return adjustConfidenceForStrongType(type, baseConfidence);
  }

  /**
   * Calculate specificity score for a selector.
   *
   * Specificity indicates how uniquely a selector identifies an element.
   * Used for ranking selector candidates.
   */
  calculateSpecificity(selector: string, type?: string): number {
    if (type) {
      const typeKey = type as keyof typeof SPECIFICITY_SCORES;
      if (typeKey in SPECIFICITY_SCORES) {
        return SPECIFICITY_SCORES[typeKey];
      }
    }

    // Infer from selector pattern
    for (const attr of TEST_ID_ATTRIBUTES) {
      if (selector.includes(`[${attr}=`)) {
        return SPECIFICITY_SCORES.dataTestId;
      }
    }

    if (selector.startsWith('#')) return SPECIFICITY_SCORES.id;
    if (selector.includes('[aria-label=')) return SPECIFICITY_SCORES.ariaLabel;
    if (selector.includes('[data-')) return SPECIFICITY_SCORES.dataAttr;
    if (this.isXPath(selector)) return SPECIFICITY_SCORES.xpathPositional;

    return SPECIFICITY_SCORES.cssPath;
  }

  // ===========================================================================
  // Utilities
  // ===========================================================================

  /**
   * Check if a selector is an XPath expression.
   */
  isXPath(selector: string): boolean {
    return selector.startsWith('/') || selector.startsWith('(');
  }

  /**
   * Check if a selector requires an element (vs page-level).
   *
   * Selectors for actions like scroll, navigate don't always need selectors.
   */
  requiresElement(actionType: string): boolean {
    const noSelectorActions = ['navigate', 'scroll', 'wait', 'keyboard', 'screenshot', 'evaluate'];
    return !noSelectorActions.includes(actionType.toLowerCase());
  }

  /**
   * Check if an ID appears to be dynamically generated.
   */
  isDynamicId(id: string): boolean {
    return this.dynamicIdPatterns.some(p => p.test(id));
  }

  /**
   * Check if a class name appears unstable (generated by CSS-in-JS, etc).
   */
  isUnstableClass(className: string): boolean {
    return this.unstableClassPatterns.some(p => p.test(className));
  }

  /**
   * Check if a class name appears semantic/stable.
   */
  isSemanticClass(className: string): boolean {
    return this.semanticClassPatterns.some(p => p.test(className));
  }

  /**
   * Get the minimum confidence threshold from config.
   */
  getMinConfidence(): number {
    return this.config.recording?.minSelectorConfidence ?? SELECTOR_DEFAULTS.minConfidence;
  }

  /**
   * Get maximum CSS depth from config.
   */
  getMaxCssDepth(): number {
    return this.config.recording?.selector?.maxCssDepth ?? SELECTOR_DEFAULTS.maxCssDepth;
  }

  /**
   * Check if XPath fallback is enabled.
   */
  isXPathEnabled(): boolean {
    return this.config.recording?.selector?.includeXPath ?? SELECTOR_DEFAULTS.includeXPath;
  }
}

/**
 * Create a SelectorService instance.
 *
 * @param page - Playwright page to validate selectors against
 * @param config - Application config with selector settings
 * @returns SelectorService instance
 */
export function createSelectorService(page: Page, config: Config): SelectorService {
  return new SelectorService(page, config);
}

// =============================================================================
// Standalone Validation Function
// =============================================================================

/**
 * Check if a selector string is an XPath expression.
 * XPath expressions start with '/' or '(' (for grouped expressions).
 */
export function isXPathSelector(selector: string): boolean {
  return selector.startsWith('/') || selector.startsWith('(');
}

/**
 * Validate a selector on a page without requiring a full SelectorService.
 *
 * This is a convenience function for contexts (like RecordModeController) that
 * need selector validation but don't have access to the full Config object.
 *
 * @param page - Playwright page to validate against
 * @param selector - CSS selector or XPath expression to validate
 * @returns Validation result with valid flag, matchCount, and error
 */
export async function validateSelectorOnPage(
  page: Page,
  selector: string
): Promise<SelectorValidation> {
  try {
    if (isXPathSelector(selector)) {
      // XPath validation via page.evaluate
      const count = await page.evaluate<number>(`
        (function() {
          try {
            const result = document.evaluate(
              ${JSON.stringify(selector)},
              document,
              null,
              XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
              null
            );
            return result.snapshotLength;
          } catch {
            return -1;
          }
        })()
      `);

      if (count === -1) {
        return { valid: false, matchCount: 0, selector, error: 'Invalid XPath expression' };
      }

      return {
        valid: count === 1,
        matchCount: count,
        selector,
      };
    } else {
      // CSS selector validation via locator
      const count = await page.locator(selector).count();
      return {
        valid: count === 1,
        matchCount: count,
        selector,
      };
    }
  } catch (error) {
    return {
      valid: false,
      matchCount: 0,
      selector,
      error: error instanceof Error ? error.message : String(error),
    };
  }
}
