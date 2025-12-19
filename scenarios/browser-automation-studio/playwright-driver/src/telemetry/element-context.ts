/**
 * Element Context Capture
 *
 * Captures element context (metadata, bounding box, selector confidence) BEFORE
 * executing an action. This enriches execution outcomes to have the same quality
 * telemetry as recording.
 *
 * DESIGN RATIONALE:
 * Recording captures rich "before action" data (selector candidates, element metadata,
 * confidence scores), but execution historically only captured "after action" results.
 * This module bridges that gap by capturing element context before actions execute.
 *
 * USAGE:
 * Handlers that target elements should call captureElementContext() before executing
 * the action, then include the result in their HandlerResult.
 *
 * @see docs/plans/enrich-execution-with-element-context.md
 */

import type { Page, Locator } from 'playwright';
import { create } from '@bufbuild/protobuf';
import {
  ElementMetaSchema,
  type ElementMeta,
} from '@vrooli/proto-types/browser-automation-studio/v1/domain/selectors_pb';
import {
  BoundingBoxSchema,
  type BoundingBox,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/geometry_pb';
import { logger } from '../utils';

// =============================================================================
// TYPES
// =============================================================================

/**
 * Element context captured before action execution.
 * This matches the richness of recording's captured data.
 */
export interface ElementContext {
  /** The selector that was used to find the element */
  selector: string;
  /** Bounding box of the element when found (viewport coordinates) */
  boundingBox?: BoundingBox;
  /** Full element metadata (tag, id, class, aria-label, etc.) */
  elementMeta?: ElementMeta;
  /** Match confidence: 1.0 = unique match, lower = ambiguous */
  confidence: number;
  /** Number of elements that matched the selector */
  matchCount: number;
}

/**
 * Raw element metadata extracted from the page.
 * This is the browser-side format before proto conversion.
 */
interface RawElementMeta {
  tagName: string;
  id?: string;
  className?: string;
  innerText?: string;
  attributes?: Record<string, string>;
  isVisible: boolean;
  isEnabled: boolean;
  role?: string;
  ariaLabel?: string;
}

// =============================================================================
// MAIN CAPTURE FUNCTION
// =============================================================================

/**
 * Capture element context before executing an action.
 *
 * This function locates the target element and captures:
 * - Bounding box (position/size)
 * - Element metadata (tag, id, class, aria attributes)
 * - Match confidence (based on uniqueness)
 *
 * Call this BEFORE executing the action to get "before" state.
 *
 * @param page - Playwright page
 * @param selector - The selector to locate the element
 * @param options - Optional configuration
 * @returns Element context, or minimal context if element not found
 */
export async function captureElementContext(
  page: Page,
  selector: string,
  options: {
    /** Timeout for locating element (ms). Default: 1000 */
    timeout?: number;
    /** Whether to capture innerText (can be expensive for large elements) */
    captureText?: boolean;
    /** Max length of innerText to capture */
    maxTextLength?: number;
  } = {}
): Promise<ElementContext> {
  const {
    timeout = 1000,
    captureText = true,
    maxTextLength = 100,
  } = options;

  // Handle empty/missing selector
  if (!selector || selector.trim() === '') {
    return {
      selector: '',
      confidence: 0,
      matchCount: 0,
    };
  }

  try {
    const locator = page.locator(selector);

    // Get match count first (quick check)
    const matchCount = await locator.count();

    if (matchCount === 0) {
      logger.debug('element-context: no elements matched selector', { selector });
      return {
        selector,
        confidence: 0,
        matchCount: 0,
      };
    }

    // Use first match for context capture
    const first = locator.first();

    // Capture bounding box (may be null if element is hidden)
    const rawBoundingBox = await first.boundingBox({ timeout });
    const boundingBox = rawBoundingBox
      ? create(BoundingBoxSchema, {
          x: rawBoundingBox.x,
          y: rawBoundingBox.y,
          width: rawBoundingBox.width,
          height: rawBoundingBox.height,
        })
      : undefined;

    // Capture element metadata via page.evaluate
    const rawMeta = await captureRawElementMeta(first, { captureText, maxTextLength });
    const elementMeta = rawMeta ? rawElementMetaToProto(rawMeta) : undefined;

    // Calculate confidence based on match uniqueness
    const confidence = calculateMatchConfidence(matchCount);

    logger.debug('element-context: captured', {
      selector,
      matchCount,
      confidence,
      tagName: rawMeta?.tagName,
      hasBoundingBox: !!boundingBox,
    });

    return {
      selector,
      boundingBox,
      elementMeta,
      confidence,
      matchCount,
    };
  } catch (error) {
    // Element context capture is best-effort - don't fail the action
    logger.warn('element-context: capture failed', {
      selector,
      error: error instanceof Error ? error.message : String(error),
    });

    return {
      selector,
      confidence: 0,
      matchCount: 0,
    };
  }
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Capture raw element metadata from a locator.
 * Runs in page context via evaluate().
 */
async function captureRawElementMeta(
  locator: Locator,
  options: { captureText: boolean; maxTextLength: number }
): Promise<RawElementMeta | undefined> {
  try {
    return await locator.evaluate(
      (el, opts) => {
        const style = window.getComputedStyle(el);

        // Get relevant attributes
        const attrs: Record<string, string> = {};
        const interestingAttrs = [
          'type',
          'name',
          'placeholder',
          'title',
          'alt',
          'href',
          'value',
          'role',
          'data-testid',
          'data-test-id',
          'data-cy',
        ];
        for (const attr of interestingAttrs) {
          const val = el.getAttribute(attr);
          if (val) {
            attrs[attr] = val;
          }
        }

        // Get visible text (truncated)
        let innerText: string | undefined;
        if (opts.captureText) {
          const text = el.textContent?.trim() || '';
          innerText = text.length > opts.maxTextLength
            ? text.slice(0, opts.maxTextLength) + '...'
            : text || undefined;
        }

        return {
          tagName: el.tagName.toLowerCase(),
          id: el.id || undefined,
          className: el.className || undefined,
          innerText,
          attributes: Object.keys(attrs).length > 0 ? attrs : undefined,
          isVisible:
            style.display !== 'none' &&
            style.visibility !== 'hidden' &&
            parseFloat(style.opacity) > 0 &&
            (el.offsetWidth > 0 || el.offsetHeight > 0),
          isEnabled: !(el as HTMLInputElement).disabled,
          role: el.getAttribute('role') || inferRole(el) || undefined,
          ariaLabel: el.getAttribute('aria-label') || undefined,
        };

        // Helper to infer ARIA role from element type
        function inferRole(element: Element): string | undefined {
          const tag = element.tagName.toLowerCase();
          const type = element.getAttribute('type')?.toLowerCase();

          const roleMap: Record<string, string> = {
            button: 'button',
            a: 'link',
            input: type === 'checkbox' ? 'checkbox' : type === 'radio' ? 'radio' : 'textbox',
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
          };

          return roleMap[tag];
        }
      },
      options
    );
  } catch {
    return undefined;
  }
}

/**
 * Convert raw element metadata to proto ElementMeta.
 */
function rawElementMetaToProto(raw: RawElementMeta): ElementMeta {
  return create(ElementMetaSchema, {
    tagName: raw.tagName,
    id: raw.id ?? '',
    className: raw.className ?? '',
    innerText: raw.innerText ?? '',
    isVisible: raw.isVisible,
    isEnabled: raw.isEnabled,
    role: raw.role ?? '',
    ariaLabel: raw.ariaLabel ?? '',
    attributes: raw.attributes ?? {},
  });
}

/**
 * Calculate match confidence based on selector uniqueness.
 *
 * - 1 match = 1.0 (unique, high confidence)
 * - 2 matches = 0.5
 * - 3+ matches = decreasing confidence
 * - 0 matches = 0.0 (element not found)
 */
function calculateMatchConfidence(matchCount: number): number {
  if (matchCount === 0) return 0;
  if (matchCount === 1) return 1.0;
  if (matchCount === 2) return 0.5;
  // For 3+ matches, use inverse proportion with floor
  return Math.max(0.2, 1 / matchCount);
}
