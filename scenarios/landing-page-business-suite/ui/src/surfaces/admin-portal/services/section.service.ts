import type { ContentSection, LandingSection } from '../../../shared/api';
import { getStylingConfig, getVariantStylingGuidance } from '../../../shared/lib/stylingConfig';

/**
 * Styling configuration singleton for the section editor
 */
export const STYLING_CONFIG = getStylingConfig();

/**
 * LocalStorage key for comparison variant preferences
 */
export const COMPARE_STORAGE_KEY = 'landing-manager-section-editor-compare';

/**
 * Get variant styling guidance for a specific variant
 */
export function getVariantGuidance(variantSlug?: string) {
  return getVariantStylingGuidance(variantSlug);
}

/**
 * Load saved comparison variant preference from localStorage
 */
export function loadComparePreference(variantSlug: string): string | null {
  if (typeof window === 'undefined') {
    return null;
  }
  try {
    const raw = window.localStorage.getItem(COMPARE_STORAGE_KEY);
    if (!raw) {
      return null;
    }
    const stored = JSON.parse(raw);
    return stored?.[variantSlug] ?? null;
  } catch {
    return null;
  }
}

/**
 * Save comparison variant preference to localStorage
 */
export function saveComparePreference(variantSlug: string, compareSlug: string | null): void {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    const raw = window.localStorage.getItem(COMPARE_STORAGE_KEY);
    const stored = raw ? JSON.parse(raw) : {};
    if (compareSlug) {
      stored[variantSlug] = compareSlug;
    } else {
      delete stored[variantSlug];
    }
    window.localStorage.setItem(COMPARE_STORAGE_KEY, JSON.stringify(stored));
  } catch {
    // localStorage not available or full - silently fail
  }
}

/**
 * Sort sections by order
 */
export function sortSectionsByOrder(sections: LandingSection[]): LandingSection[] {
  return [...sections].sort((a, b) => (a.order ?? 0) - (b.order ?? 0));
}

/**
 * Find a section by type and order in a list
 */
export function findSectionByType(
  sections: LandingSection[],
  sectionType: ContentSection['section_type']
): LandingSection | null {
  const matching = sections
    .filter((section) => section.section_type === sectionType)
    .sort((a, b) => (a.order ?? 0) - (b.order ?? 0));
  return matching[0] ?? null;
}

/**
 * Build default content fields for a section
 */
export function buildDefaultSectionContent(): Record<string, unknown> {
  return {
    title: '',
    subtitle: '',
    cta_text: '',
    cta_url: '',
  };
}
