/**
 * Element Annotator
 *
 * STABILITY: STABLE CORE
 *
 * This module extracts interactive elements from a page and assigns them
 * numbered labels for AI navigation. The labels help vision models
 * reference specific elements by number (e.g., "click element [5]").
 *
 * DESIGN NOTES:
 * - Currently provides metadata-only annotation (no visual overlay)
 * - Visual annotation (drawing labels on screenshot) can be added later
 *   using sharp or canvas libraries
 * - Elements are prioritized by visibility and interactivity
 */

import type { Page } from 'rebrowser-playwright';
import type { ElementAnnotatorInterface, AnnotatedScreenshot } from '../vision-agent/types';
import type { ElementLabel } from '../vision-client/types';

/**
 * Configuration for element annotation.
 */
export interface AnnotatorConfig {
  /** Maximum number of elements to label (default: 50) */
  maxElements?: number;
  /** Include only visible elements (default: true) */
  visibleOnly?: boolean;
  /** Maximum text length to capture per element */
  maxTextLength?: number;
}

/**
 * Default maximum elements to annotate.
 */
const DEFAULT_MAX_ELEMENTS = 50;

/**
 * Default max text length per element.
 */
const DEFAULT_MAX_TEXT_LENGTH = 100;

/**
 * Selectors for interactive elements.
 */
const INTERACTIVE_SELECTORS = [
  'a[href]',
  'button',
  'input:not([type="hidden"])',
  'select',
  'textarea',
  '[role="button"]',
  '[role="link"]',
  '[role="menuitem"]',
  '[role="tab"]',
  '[role="checkbox"]',
  '[role="radio"]',
  '[role="switch"]',
  '[role="combobox"]',
  '[role="option"]',
  '[role="textbox"]',
  '[role="searchbox"]',
  '[onclick]',
  '[tabindex]:not([tabindex="-1"])',
].join(', ');

/**
 * Create an element annotator.
 *
 * TESTING SEAM: This returns an interface that can be mocked.
 */
export function createElementAnnotator(
  _config: AnnotatorConfig = {}
): ElementAnnotatorInterface {
  // Config will be used in Phase 6 for visual annotation options

  return {
    async annotate(
      screenshot: Buffer,
      _elements: ElementLabel[]
    ): Promise<AnnotatedScreenshot> {
      // For now, return the screenshot as-is with the provided labels
      // Visual annotation (drawing labels) will be added in Phase 6
      return {
        image: screenshot,
        labels: _elements,
      };
    },
  };
}

/**
 * Arguments passed to the browser-context extraction function.
 */
interface ExtractArgs {
  selectors: string;
  maxElements: number;
  visibleOnly: boolean;
  maxTextLength: number;
  viewportWidth: number;
  viewportHeight: number;
}

/**
 * Raw element data returned from browser context.
 */
interface RawElement {
  selector: string;
  tagName: string;
  bounds: { x: number; y: number; width: number; height: number };
  text?: string;
  role?: string;
  placeholder?: string;
  ariaLabel?: string;
}

/**
 * Extract interactive elements from a page and assign numbered labels.
 *
 * This is the main function for discovering actionable elements.
 * It runs in the browser context to extract element metadata.
 *
 * @param page - Playwright page
 * @param config - Annotation configuration
 * @returns Array of labeled elements
 */
export async function extractInteractiveElements(
  page: Page,
  config: AnnotatorConfig = {}
): Promise<ElementLabel[]> {
  const maxElements = config.maxElements ?? DEFAULT_MAX_ELEMENTS;
  const visibleOnly = config.visibleOnly ?? true;
  const maxTextLength = config.maxTextLength ?? DEFAULT_MAX_TEXT_LENGTH;

  // Get viewport for visibility check
  const viewport = page.viewportSize();
  if (!viewport) {
    return [];
  }

  // Extract elements in browser context
  // Note: The callback runs in browser context where DOM types are available
  const rawElements = await page.evaluate<RawElement[], ExtractArgs>(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (args: any) => {
      const {
        selectors,
        maxElements,
        visibleOnly,
        maxTextLength,
        viewportWidth,
        viewportHeight,
      } = args;

      // Helper: Check if element is visible in viewport
      function isInViewport(rect: { top: number; bottom: number; left: number; right: number; width: number; height: number }): boolean {
        return (
          rect.top < viewportHeight &&
          rect.bottom > 0 &&
          rect.left < viewportWidth &&
          rect.right > 0 &&
          rect.width > 0 &&
          rect.height > 0
        );
      }

      // Helper: Check if element is visually visible
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      function isVisuallyVisible(el: any): boolean {
        const style = (window as any).getComputedStyle(el);
        return (
          style.display !== 'none' &&
          style.visibility !== 'hidden' &&
          parseFloat(style.opacity) > 0
        );
      }

      // Helper: Generate a unique CSS selector for an element
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      function generateSelector(el: any): string {
        // Try ID first
        if (el.id) {
          return `#${(CSS as any).escape(el.id)}`;
        }

        // Try data-testid
        const testId = el.getAttribute('data-testid');
        if (testId) {
          return `[data-testid="${(CSS as any).escape(testId)}"]`;
        }

        // Try aria-label
        const ariaLabel = el.getAttribute('aria-label');
        if (ariaLabel) {
          const escaped = ariaLabel.replace(/"/g, '\\"');
          return `[aria-label="${escaped}"]`;
        }

        // Try name attribute for form elements
        const name = el.getAttribute('name');
        if (name) {
          return `${el.tagName.toLowerCase()}[name="${(CSS as any).escape(name)}"]`;
        }

        // Fall back to tag + nth-of-type
        const parent = el.parentElement;
        if (parent) {
          const siblings = Array.from(parent.children).filter(
            (s: any) => s.tagName === el.tagName
          );
          const index = siblings.indexOf(el) + 1;
          const parentSelector = parent === (document as any).body
            ? 'body'
            : generateSelector(parent);
          return `${parentSelector} > ${el.tagName.toLowerCase()}:nth-of-type(${index})`;
        }

        return el.tagName.toLowerCase();
      }

      // Helper: Get element text content (truncated)
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      function getText(el: any): string | undefined {
        // For inputs, use placeholder
        if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
          return el.placeholder || undefined;
        }

        // For other elements, use text content
        const text = el.textContent?.trim();
        if (!text) return undefined;

        return text.length > maxTextLength
          ? text.slice(0, maxTextLength) + '...'
          : text;
      }

      // Helper: Get element role
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      function getRole(el: any): string | undefined {
        const explicit = el.getAttribute('role');
        if (explicit) return explicit;

        // Infer from tag
        const tag = el.tagName.toLowerCase();
        const type = el.getAttribute('type')?.toLowerCase();

        const roleMap: Record<string, string> = {
          a: 'link',
          button: 'button',
          select: 'combobox',
          textarea: 'textbox',
        };

        if (tag === 'input') {
          if (type === 'checkbox') return 'checkbox';
          if (type === 'radio') return 'radio';
          if (type === 'submit' || type === 'button') return 'button';
          return 'textbox';
        }

        return roleMap[tag];
      }

      // Query all interactive elements
      const elements = Array.from((document as any).querySelectorAll(selectors));

      // Filter and map elements
      const results: Array<{
        selector: string;
        tagName: string;
        bounds: { x: number; y: number; width: number; height: number };
        text?: string;
        role?: string;
        placeholder?: string;
        ariaLabel?: string;
      }> = [];

      for (const el of elements) {
        if (results.length >= maxElements) break;

        const rect = (el as any).getBoundingClientRect();

        // Skip if not in viewport (when visibleOnly is true)
        if (visibleOnly && !isInViewport(rect)) continue;

        // Skip if not visually visible
        if (visibleOnly && !isVisuallyVisible(el)) continue;

        // Skip very small elements
        if (rect.width < 10 || rect.height < 10) continue;

        results.push({
          selector: generateSelector(el),
          tagName: (el as any).tagName.toLowerCase(),
          bounds: {
            x: Math.round(rect.x),
            y: Math.round(rect.y),
            width: Math.round(rect.width),
            height: Math.round(rect.height),
          },
          text: getText(el),
          role: getRole(el),
          placeholder: (el as any).getAttribute('placeholder') || undefined,
          ariaLabel: (el as any).getAttribute('aria-label') || undefined,
        });
      }

      return results;
    },
    {
      selectors: INTERACTIVE_SELECTORS,
      maxElements,
      visibleOnly,
      maxTextLength,
      viewportWidth: viewport.width,
      viewportHeight: viewport.height,
    }
  );

  // Assign sequential IDs
  return rawElements.map((el, index) => ({
    id: index + 1, // 1-indexed for human readability
    selector: el.selector,
    tagName: el.tagName,
    bounds: el.bounds,
    text: el.text,
    role: el.role,
    placeholder: el.placeholder,
    ariaLabel: el.ariaLabel,
  }));
}

/**
 * Format element labels for inclusion in LLM prompt.
 *
 * Creates a text description of available elements that helps
 * the vision model reference elements by number.
 */
export function formatElementLabelsForPrompt(labels: ElementLabel[]): string {
  if (labels.length === 0) {
    return 'No interactive elements detected on this page.';
  }

  const lines = ['Interactive elements on this page:', ''];

  for (const label of labels) {
    const parts: string[] = [];

    // Primary descriptor
    parts.push(`[${label.id}]`);

    // Role/type
    if (label.role) {
      parts.push(label.role);
    } else {
      parts.push(label.tagName);
    }

    // Text content or placeholder
    if (label.text) {
      parts.push(`"${label.text}"`);
    } else if (label.placeholder) {
      parts.push(`(placeholder: "${label.placeholder}")`);
    } else if (label.ariaLabel) {
      parts.push(`(aria-label: "${label.ariaLabel}")`);
    }

    lines.push(parts.join(' '));
  }

  return lines.join('\n');
}

/**
 * Create a mock annotator for testing.
 */
export function createMockAnnotator(
  mockLabels?: ElementLabel[]
): ElementAnnotatorInterface & {
  getCalls(): number;
  reset(): void;
} {
  let callCount = 0;

  return {
    async annotate(
      screenshot: Buffer,
      _elements: ElementLabel[]
    ): Promise<AnnotatedScreenshot> {
      callCount++;
      return {
        image: screenshot,
        labels: mockLabels ?? _elements,
      };
    },

    getCalls() {
      return callCount;
    },

    reset() {
      callCount = 0;
    },
  };
}
