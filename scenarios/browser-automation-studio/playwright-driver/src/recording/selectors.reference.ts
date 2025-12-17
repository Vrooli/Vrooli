/* eslint-disable @typescript-eslint/ban-ts-comment, @typescript-eslint/no-unsafe-assignment, @typescript-eslint/no-unsafe-call, @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-argument, @typescript-eslint/no-redundant-type-constituents, @typescript-eslint/no-unsafe-return, @typescript-eslint/restrict-template-expressions */
// @ts-nocheck - DOM types used here are for documentation purposes only
/* eslint-enable @typescript-eslint/ban-ts-comment */

/**
 * ============================================================================
 * SELECTOR GENERATION - REFERENCE IMPLEMENTATION
 * ============================================================================
 *
 * FILE: selectors.reference.ts
 *
 * STATUS: DOCUMENTATION ONLY - NOT IMPORTED ANYWHERE
 *
 * This file contains a TypeScript reference implementation of the selector
 * generation algorithm. It is NOT executed at runtime.
 *
 * The ACTUAL runtime code lives in `injector.ts` as stringified JavaScript
 * that gets injected into browser pages via page.evaluate().
 *
 * WHY THIS DUAL-FILE APPROACH EXISTS:
 * - Browser context requires plain JavaScript (no TypeScript)
 * - injector.ts contains stringified JS for browser injection
 * - This file provides readable TypeScript with types as documentation
 * - Algorithm changes should be made in injector.ts FIRST
 *
 * KEEPING THEM IN SYNC:
 * - This file may drift from injector.ts over time
 * - injector.ts is the source of truth for runtime behavior
 * - Update this file opportunistically when making algorithm changes
 *
 * SELECTOR STRATEGY PRIORITY (highest confidence first):
 * 1. data-testid - Explicitly stable test IDs (0.98)
 * 2. Unique ID - DOM IDs if not dynamic (0.95, 0.6 if dynamic)
 * 3. ARIA attributes - Semantic accessibility (0.85)
 * 4. Tag + text - Playwright :has-text() patterns (0.55-0.6)
 * 5. Data attributes - Other data-* attributes (0.7)
 * 6. Stable CSS path - Filtered CSS classes (0.65)
 * 7. XPath fallback - Text-based or positional (0.4-0.55)
 *
 * @see injector.ts - The actual runtime implementation (SOURCE OF TRUTH)
 * @see selector-config.ts - Configuration values (shared by both files)
 */

import type {
  SelectorCandidate,
  SelectorSet,
  SelectorGeneratorOptions,
  ElementMeta,
} from './types';
import { DEFAULT_SELECTOR_OPTIONS } from './types';
import {
  TEST_ID_ATTRIBUTES,
  TEXT_CONTENT_TAGS,
  RELEVANT_ATTRIBUTES,
  CONFIDENCE_SCORES,
  SPECIFICITY_SCORES,
  SELECTOR_DEFAULTS,
  getDynamicIdPatterns,
  getSemanticClassPatterns,
} from './selector-config';

/**
 * Generate selector set for an element.
 * This function is designed to be called from browser context via page.evaluate().
 *
 * @param element - Target DOM element
 * @param options - Generation options
 * @returns SelectorSet with primary selector and ranked candidates
 */
export function generateSelectors(
  element: Element,
  options: SelectorGeneratorOptions = {}
): SelectorSet {
  const opts = { ...DEFAULT_SELECTOR_OPTIONS, ...options };
  const candidates: SelectorCandidate[] = [];

  // Strategy 1: data-testid (highest confidence)
  const testIdCandidate = generateTestIdSelector(element);
  if (testIdCandidate) {
    candidates.push(testIdCandidate);
  }

  // Strategy 2: Unique ID
  const idCandidate = generateIdSelector(element);
  if (idCandidate) {
    candidates.push(idCandidate);
  }

  // Strategy 3: ARIA attributes
  const ariaCandidate = generateAriaSelector(element);
  if (ariaCandidate) {
    candidates.push(ariaCandidate);
  }

  // Strategy 4: Tag + text (Playwright pattern)
  const textCandidate = generateTextSelector(element);
  if (textCandidate) {
    candidates.push(textCandidate);
  }

  // Strategy 5: Other data-* attributes
  const dataAttrCandidate = generateDataAttrSelector(element);
  if (dataAttrCandidate) {
    candidates.push(dataAttrCandidate);
  }

  // Strategy 6: Stable CSS path
  const cssCandidate = generateStableCssPath(element, opts);
  if (cssCandidate) {
    candidates.push(cssCandidate);
  }

  // Strategy 7: XPath fallback
  if (opts.includeXPath) {
    const xpathCandidate = generateXPathSelector(element);
    if (xpathCandidate) {
      candidates.push(xpathCandidate);
    }
  }

  // Filter by minimum confidence
  const filtered = candidates.filter((c) => c.confidence >= opts.minConfidence);

  // Sort by confidence (descending), then specificity (descending)
  filtered.sort((a, b) => {
    if (b.confidence !== a.confidence) {
      return b.confidence - a.confidence;
    }
    return b.specificity - a.specificity;
  });

  // Select primary (highest confidence that's unique)
  const primary = filtered[0]?.value || generateFallbackSelector(element);

  return {
    primary,
    candidates: filtered,
  };
}

/**
 * Generate selector from data-testid or similar test attributes.
 */
function generateTestIdSelector(element: Element): SelectorCandidate | null {
  // Check common test ID attribute names (from shared config)
  for (const attr of TEST_ID_ATTRIBUTES) {
    const value = element.getAttribute(attr);
    if (value) {
      const selector = `[${attr}="${escapeCssAttributeValue(value)}"]`;
      if (isUniqueSelector(selector)) {
        return {
          type: 'data-testid',
          value: selector,
          confidence: CONFIDENCE_SCORES.dataTestId,
          specificity: SPECIFICITY_SCORES.dataTestId,
        };
      }
    }
  }

  return null;
}

/**
 * Generate selector from element ID.
 */
function generateIdSelector(element: Element): SelectorCandidate | null {
  const id = element.id;
  if (!id) return null;

  // Check if ID looks dynamic (contains numbers that might change)
  const isDynamic = isDynamicId(id);

  const selector = `#${escapeCssSelector(id)}`;
  if (isUniqueSelector(selector)) {
    return {
      type: 'id',
      value: selector,
      confidence: isDynamic ? CONFIDENCE_SCORES.idDynamic : CONFIDENCE_SCORES.id,
      specificity: SPECIFICITY_SCORES.id,
    };
  }

  return null;
}

/**
 * Generate selector from ARIA attributes.
 */
function generateAriaSelector(element: Element): SelectorCandidate | null {
  // Try aria-label first (most common)
  const ariaLabel = element.getAttribute('aria-label');
  if (ariaLabel) {
    const selector = `[aria-label="${escapeCssAttributeValue(ariaLabel)}"]`;
    if (isUniqueSelector(selector)) {
      return {
        type: 'aria',
        value: selector,
        confidence: CONFIDENCE_SCORES.ariaLabel,
        specificity: SPECIFICITY_SCORES.ariaLabel,
      };
    }
  }

  // Try aria-labelledby
  const labelledBy = element.getAttribute('aria-labelledby');
  if (labelledBy) {
    const selector = `[aria-labelledby="${escapeCssAttributeValue(labelledBy)}"]`;
    if (isUniqueSelector(selector)) {
      return {
        type: 'aria',
        value: selector,
        confidence: CONFIDENCE_SCORES.ariaLabelledby,
        specificity: SPECIFICITY_SCORES.ariaLabelledby,
      };
    }
  }

  // Try aria-describedby
  const describedBy = element.getAttribute('aria-describedby');
  if (describedBy) {
    const selector = `[aria-describedby="${escapeCssAttributeValue(describedBy)}"]`;
    if (isUniqueSelector(selector)) {
      return {
        type: 'aria',
        value: selector,
        confidence: CONFIDENCE_SCORES.ariaDescribedby,
        specificity: SPECIFICITY_SCORES.ariaDescribedby,
      };
    }
  }

  return null;
}

/**
 * Generate Playwright-style tag + text selector.
 * Example: button:has-text("Submit")
 *
 * Note: Uses CSS tag names (a, button) not role names (link, button)
 * for compatibility with Playwright's :has-text() pseudo-selector.
 */
function generateTextSelector(element: Element): SelectorCandidate | null {
  const tagName = element.tagName.toLowerCase();

  // Only generate text selectors for elements that commonly have meaningful text
  if (!TEXT_CONTENT_TAGS.includes(tagName as typeof TEXT_CONTENT_TAGS[number])) return null;

  const text = getVisibleText(element);
  const minLen = SELECTOR_DEFAULTS.minTextLength;
  const maxLen = SELECTOR_DEFAULTS.maxTextLengthForSelector;
  const truncateLen = SELECTOR_DEFAULTS.selectorTextMaxLength;

  if (!text || text.length < minLen || text.length > maxLen) return null;

  // Truncate text for selector stability
  const selectorText = text.length > truncateLen ? text.slice(0, truncateLen) : text;
  const escapedText = selectorText.replace(/"/g, '\\"');
  const selector = `${tagName}:has-text("${escapedText}")`;

  // Can't easily validate Playwright selectors with querySelectorAll
  // Estimate uniqueness based on text specificity - lower confidence than other strategies
  const confidence = text.length > 15 ? CONFIDENCE_SCORES.textLong : CONFIDENCE_SCORES.textShort;

  return {
    type: 'text',
    value: selector,
    confidence,
    specificity: SPECIFICITY_SCORES.text,
  };
}

/**
 * Generate selector from other data-* attributes.
 */
function generateDataAttrSelector(element: Element): SelectorCandidate | null {
  // Get all data-* attributes except test IDs (already handled)
  const skipAttrs = new Set(['testid', 'test-id', 'test', 'cy', 'qa']);

  for (const attr of element.attributes) {
    if (!attr.name.startsWith('data-')) continue;

    const attrSuffix = attr.name.slice(5); // Remove 'data-' prefix
    if (skipAttrs.has(attrSuffix)) continue;

    const selector = `[${attr.name}="${escapeCssAttributeValue(attr.value)}"]`;
    if (isUniqueSelector(selector)) {
      return {
        type: 'data-attr',
        value: selector,
        confidence: CONFIDENCE_SCORES.dataAttr,
        specificity: SPECIFICITY_SCORES.dataAttr,
      };
    }
  }

  return null;
}

/**
 * Generate stable CSS path selector.
 * Filters out dynamic/generated class names.
 */
function generateStableCssPath(
  element: Element,
  options: Required<SelectorGeneratorOptions>
): SelectorCandidate | null {
  const pathParts: string[] = [];
  let current: Element | null = element;
  let depth = 0;

  while (current && current !== document.body && depth < options.maxCssDepth) {
    const tagName = current.tagName.toLowerCase();
    const stableClasses = getStableClasses(current, options.unstableClassPatterns);

    let part = tagName;

    // Add stable classes if available
    if (stableClasses.length > 0) {
      // Use at most 2 classes to avoid over-specificity
      const classSelector = stableClasses.slice(0, 2).join('.');
      part = `${tagName}.${classSelector}`;
    }

    pathParts.unshift(part);

    // Check if current path is already unique
    const selector = pathParts.join(' > ');
    if (isUniqueSelector(selector)) {
      return {
        type: 'css',
        value: selector,
        confidence: assessCssStability(selector, stableClasses),
        specificity: SPECIFICITY_SCORES.cssPath - depth * 5, // Shorter paths are better
      };
    }

    current = current.parentElement;
    depth++;
  }

  // If we couldn't find a unique path, try with nth-child
  return generateCssPathWithNthChild(element, options.maxCssDepth);
}

/**
 * Generate CSS path using nth-child for uniqueness.
 * Less stable but more likely to be unique.
 */
function generateCssPathWithNthChild(
  element: Element,
  maxDepth: number
): SelectorCandidate | null {
  const pathParts: string[] = [];
  let current: Element | null = element;
  let depth = 0;

  while (current && current !== document.body && depth < maxDepth) {
    const tagName = current.tagName.toLowerCase();
    const parent = current.parentElement;

    let part = tagName;

    if (parent) {
      // Find nth-child index
      const siblings = Array.from(parent.children);
      const sameTagSiblings = siblings.filter(
        (s) => s.tagName.toLowerCase() === tagName
      );

      if (sameTagSiblings.length > 1) {
        const index = sameTagSiblings.indexOf(current) + 1;
        part = `${tagName}:nth-of-type(${index})`;
      }
    }

    pathParts.unshift(part);

    // Check if unique
    const selector = pathParts.join(' > ');
    if (isUniqueSelector(selector)) {
      return {
        type: 'css',
        value: selector,
        confidence: CONFIDENCE_SCORES.cssNthChild,
        specificity: SPECIFICITY_SCORES.cssNthChild - depth * 5,
      };
    }

    current = parent;
    depth++;
  }

  return null;
}

/**
 * Generate XPath selector based on element attributes and text.
 */
function generateXPathSelector(element: Element): SelectorCandidate | null {
  const tagName = element.tagName.toLowerCase();
  const maxLen = SELECTOR_DEFAULTS.maxTextLengthForSelector;

  // Try text-based XPath first
  const text = getVisibleText(element);
  if (text && text.length > 0 && text.length <= maxLen) {
    const escapedText = escapeXPathString(text);
    const xpath = `//${tagName}[contains(text(), ${escapedText})]`;
    if (isUniqueXPath(xpath)) {
      return {
        type: 'xpath',
        value: xpath,
        confidence: CONFIDENCE_SCORES.xpathText,
        specificity: SPECIFICITY_SCORES.xpathText,
      };
    }
  }

  // Try attribute-based XPath
  const id = element.id;
  if (id) {
    const xpath = `//${tagName}[@id="${id}"]`;
    if (isUniqueXPath(xpath)) {
      return {
        type: 'xpath',
        value: xpath,
        confidence: isDynamicId(id) ? CONFIDENCE_SCORES.xpathPositional + 0.05 : CONFIDENCE_SCORES.idDynamic,
        specificity: SPECIFICITY_SCORES.cssNthChild, // Same as nth-child CSS
      };
    }
  }

  // Fallback to positional XPath
  const positionalXPath = generatePositionalXPath(element);
  if (positionalXPath) {
    return {
      type: 'xpath',
      value: positionalXPath,
      confidence: CONFIDENCE_SCORES.xpathPositional,
      specificity: SPECIFICITY_SCORES.xpathPositional,
    };
  }

  return null;
}

/**
 * Generate positional XPath (e.g., /html/body/div[2]/button[1])
 */
function generatePositionalXPath(element: Element): string {
  const parts: string[] = [];
  let current: Element | null = element;

  while (current && current !== document.documentElement) {
    const tagName = current.tagName.toLowerCase();
    const parent = current.parentElement;

    if (parent) {
      const siblings = Array.from(parent.children).filter(
        (s) => s.tagName.toLowerCase() === tagName
      );

      if (siblings.length > 1) {
        const index = siblings.indexOf(current) + 1;
        parts.unshift(`${tagName}[${index}]`);
      } else {
        parts.unshift(tagName);
      }
    } else {
      parts.unshift(tagName);
    }

    current = parent;
  }

  return '/' + parts.join('/');
}

/**
 * Generate a fallback selector when all strategies fail.
 * Uses the most specific path available.
 */
function generateFallbackSelector(element: Element): string {
  // Try a direct CSS path
  const tagName = element.tagName.toLowerCase();

  // If element has any distinguishing attribute
  for (const attr of element.attributes) {
    if (attr.name === 'class' || attr.name === 'style') continue;
    const selector = `${tagName}[${attr.name}="${escapeCssAttributeValue(attr.value)}"]`;
    if (isUniqueSelector(selector)) {
      return selector;
    }
  }

  // Fall back to positional XPath
  return generatePositionalXPath(element);
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Check if a CSS selector matches exactly one element.
 */
function isUniqueSelector(selector: string): boolean {
  try {
    const matches = document.querySelectorAll(selector);
    return matches.length === 1;
  } catch {
    return false;
  }
}

/**
 * Check if an XPath matches exactly one element.
 */
function isUniqueXPath(xpath: string): boolean {
  try {
    const result = document.evaluate(
      xpath,
      document,
      null,
      XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
      null
    );
    return result.snapshotLength === 1;
  } catch {
    return false;
  }
}

/**
 * Check if an ID looks dynamic (contains patterns that might change).
 * Uses patterns from selector-config.ts.
 */
function isDynamicId(id: string): boolean {
  const dynamicPatterns = getDynamicIdPatterns();
  return dynamicPatterns.some((pattern) => pattern.test(id));
}

/**
 * Get stable classes from an element, filtering out generated ones.
 */
function getStableClasses(
  element: Element,
  unstablePatterns: RegExp[]
): string[] {
  const classes = Array.from(element.classList);

  return classes.filter((className) => {
    // Skip empty or whitespace-only
    if (!className.trim()) return false;

    // Skip if matches any unstable pattern
    return !unstablePatterns.some((pattern) => pattern.test(className));
  });
}

/**
 * Assess stability of a CSS selector based on its components.
 * Uses semantic class patterns from selector-config.ts.
 */
function assessCssStability(selector: string, stableClasses: string[]): number {
  let score = CONFIDENCE_SCORES.cssPath; // Base score for CSS selectors

  // Boost for having stable semantic classes
  const semanticClassPatterns = getSemanticClassPatterns();

  for (const cls of stableClasses) {
    if (semanticClassPatterns.some((p) => p.test(cls))) {
      score += 0.1;
    }
  }

  // Penalize for positional selectors
  if (selector.includes('nth-child') || selector.includes('nth-of-type')) {
    score -= 0.15;
  }

  // Penalize for overly long selectors
  const depth = (selector.match(/>/g) || []).length;
  if (depth > 3) {
    score -= 0.05 * (depth - 3);
  }

  return Math.max(0.3, Math.min(0.9, score));
}

/**
 * Infer element role from tag name.
 */
function inferRole(element: Element): string | null {
  const tagName = element.tagName.toLowerCase();
  const roleMap: Record<string, string> = {
    button: 'button',
    a: 'link',
    input: element.getAttribute('type') === 'checkbox' ? 'checkbox' :
           element.getAttribute('type') === 'radio' ? 'radio' :
           element.getAttribute('type') === 'submit' ? 'button' : 'textbox',
    select: 'combobox',
    textarea: 'textbox',
    img: 'img',
    nav: 'navigation',
    main: 'main',
    header: 'banner',
    footer: 'contentinfo',
    aside: 'complementary',
    article: 'article',
    section: 'region',
    form: 'form',
    table: 'table',
    ul: 'list',
    ol: 'list',
    li: 'listitem',
    dialog: 'dialog',
  };

  return roleMap[tagName] || null;
}

/**
 * Get visible text content from element.
 */
function getVisibleText(element: Element): string {
  // For input elements, use value or placeholder
  if (element instanceof HTMLInputElement) {
    return element.value || element.placeholder || '';
  }

  if (element instanceof HTMLTextAreaElement) {
    return element.value || element.placeholder || '';
  }

  // For other elements, get text content
  const text = element.textContent?.trim() || '';

  // Truncate to reasonable length
  return text.slice(0, 100);
}

/**
 * Escape special characters for CSS selectors.
 */
function escapeCssSelector(str: string): string {
  return str.replace(/([!"#$%&'()*+,./:;<=>?@[\\\]^`{|}~])/g, '\\$1');
}

/**
 * Escape value for CSS attribute selectors.
 */
function escapeCssAttributeValue(str: string): string {
  return str.replace(/"/g, '\\"').replace(/\n/g, '\\n');
}

/**
 * Escape string for use in XPath.
 */
function escapeXPathString(str: string): string {
  // If string contains both quotes, use concat()
  if (str.includes("'") && str.includes('"')) {
    const parts = str.split("'");
    return `concat('${parts.join("', \"'\", '")}')`;
  }

  // If string contains single quotes, use double quotes
  if (str.includes("'")) {
    return `"${str}"`;
  }

  // Default to single quotes
  return `'${str}'`;
}

/**
 * Extract element metadata for display purposes.
 */
export function extractElementMeta(element: Element): ElementMeta {
  const computedStyle = window.getComputedStyle(element);

  return {
    tagName: element.tagName.toLowerCase(),
    id: element.id || undefined,
    className: element.className || undefined,
    innerText: getVisibleText(element).slice(0, 100) || undefined,
    attributes: getRelevantAttributes(element),
    isVisible: computedStyle.display !== 'none' &&
               computedStyle.visibility !== 'hidden' &&
               parseFloat(computedStyle.opacity) > 0,
    isEnabled: !(element as HTMLInputElement).disabled,
    role: element.getAttribute('role') || inferRole(element) || undefined,
    ariaLabel: element.getAttribute('aria-label') || undefined,
  };
}

/**
 * Get relevant attributes for element identification.
 * Uses RELEVANT_ATTRIBUTES from selector-config.ts.
 */
function getRelevantAttributes(element: Element): Record<string, string> {
  const relevant: Record<string, string> = {};
  const maxLen = SELECTOR_DEFAULTS.maxTextLength;

  for (const attr of RELEVANT_ATTRIBUTES) {
    const value = element.getAttribute(attr);
    if (value) {
      relevant[attr] = value.slice(0, maxLen); // Truncate long values
    }
  }

  // Also include data-* attributes
  for (const attr of element.attributes) {
    if (attr.name.startsWith('data-') && !relevant[attr.name]) {
      relevant[attr.name] = attr.value.slice(0, maxLen);
    }
  }

  return relevant;
}

/**
 * Validate a selector against the current page.
 */
export function validateSelector(selector: string): {
  valid: boolean;
  matchCount: number;
  error?: string;
} {
  try {
    // Try as CSS selector first
    const matches = document.querySelectorAll(selector);
    return {
      valid: matches.length === 1,
      matchCount: matches.length,
    };
  } catch (cssError) {
    // Try as XPath
    if (selector.startsWith('/') || selector.startsWith('(')) {
      try {
        const result = document.evaluate(
          selector,
          document,
          null,
          XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
          null
        );
        return {
          valid: result.snapshotLength === 1,
          matchCount: result.snapshotLength,
        };
      } catch (xpathError) {
        return {
          valid: false,
          matchCount: 0,
          error: `Invalid selector: ${xpathError}`,
        };
      }
    }

    return {
      valid: false,
      matchCount: 0,
      error: `Invalid CSS selector: ${cssError}`,
    };
  }
}
