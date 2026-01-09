/**
 * Element Hierarchy Utilities
 *
 * Shared helper functions for working with element hierarchy data
 * from the element picker and coordinate-based selection.
 */

import type { ElementHierarchyEntry, ElementInfo } from '@/types/elements';

/**
 * Normalize raw hierarchy data into a validated array of ElementHierarchyEntry.
 */
export const normalizeHierarchy = (value: unknown): ElementHierarchyEntry[] => {
  if (!Array.isArray(value)) {
    return [];
  }

  const results: ElementHierarchyEntry[] = [];

  for (const entry of value) {
    if (!entry || typeof entry !== 'object') {
      continue;
    }

    const candidate = entry as Partial<ElementHierarchyEntry> & { element?: ElementInfo };
    if (!candidate.element) {
      continue;
    }

    const selector =
      typeof candidate.selector === 'string' && candidate.selector.trim().length > 0
        ? candidate.selector.trim()
        : Array.isArray(candidate.element.selectors) && candidate.element.selectors.length > 0
          ? candidate.element.selectors[0].selector
          : '';

    const depth = Number.isFinite(candidate.depth as number) ? Number(candidate.depth) : 0;

    const path = Array.isArray(candidate.path)
      ? candidate.path.filter((segment): segment is string => typeof segment === 'string')
      : [];

    const summary =
      typeof candidate.pathSummary === 'string' && candidate.pathSummary.trim().length > 0
        ? candidate.pathSummary.trim()
        : path.join(' > ');

    results.push({
      element: candidate.element,
      selector,
      depth,
      path,
      pathSummary: summary || undefined,
    });
  }

  return results;
};

/**
 * Extract the best available selector from a hierarchy entry.
 */
export const deriveSelector = (entry: ElementHierarchyEntry | null | undefined): string => {
  if (!entry) {
    return '';
  }
  if (typeof entry.selector === 'string' && entry.selector.trim().length > 0) {
    return entry.selector.trim();
  }
  const selectors = Array.isArray(entry.element?.selectors) ? entry.element.selectors : [];
  return selectors.length > 0 ? selectors[0].selector : '';
};

/**
 * Create a human-readable summary of an element (tag, id, text snippet).
 */
export const summarizeElement = (entry: ElementHierarchyEntry | null | undefined): string => {
  if (!entry || !entry.element) {
    return '';
  }
  const tagName =
    typeof entry.element.tagName === 'string' ? entry.element.tagName.toLowerCase() : '';
  const id = entry.element.attributes?.id ? `#${entry.element.attributes.id}` : '';
  const text = typeof entry.element.text === 'string' ? entry.element.text.trim() : '';
  const textSnippet =
    text.length > 0 ? ` • ${text.length > 40 ? `${text.slice(0, 40)}…` : text}` : '';
  const base = `${tagName}${id}`.trim();
  return (base + textSnippet).trim() || deriveSelector(entry) || 'element';
};

/**
 * Convert a hierarchy entry's path to a readable string.
 */
export const stringifyPath = (entry: ElementHierarchyEntry | null | undefined): string => {
  if (!entry) {
    return '';
  }
  if (typeof entry.pathSummary === 'string' && entry.pathSummary.trim().length > 0) {
    return entry.pathSummary;
  }
  return Array.isArray(entry.path) ? entry.path.filter(Boolean).join(' > ') : '';
};
