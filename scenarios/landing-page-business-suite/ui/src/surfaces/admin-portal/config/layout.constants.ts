/**
 * Standardized layout constants for admin portal pages.
 *
 * Use these constants to maintain consistent spacing and layout
 * across all admin pages.
 */
export const LAYOUT = {
  /**
   * Max-width presets for page containers.
   * - narrow: Simple pages like Profile, Coming Soon stubs
   * - default: Standard settings pages
   * - wide: Dashboards and data-heavy pages
   * - extraWide: Split-panel editors (e.g., SectionEditor)
   */
  maxWidth: {
    narrow: 'max-w-4xl',
    default: 'max-w-5xl',
    wide: 'max-w-6xl',
    extraWide: 'max-w-[2000px]',
  },

  /**
   * Vertical spacing between major sections on a page.
   * Use for spacing between PageHeader and content, between cards, etc.
   */
  pageSpacing: 'space-y-8',

  /**
   * Vertical spacing between form sections within a card.
   */
  sectionSpacing: 'space-y-6',

  /**
   * Vertical spacing between form fields within a section.
   */
  contentSpacing: 'space-y-6',

  /**
   * Grid gaps for multi-column layouts.
   */
  gridGap: {
    default: 'gap-6',
    tight: 'gap-4',
    wide: 'gap-8',
  },

  /**
   * Common card styling for admin forms.
   * - base: Standard border and background
   * - hover: Interactive hover state
   * - elevated: Prominent cards with shadow
   * - interactive: Cards that respond to hover
   * - section: Large section containers
   */
  card: {
    base: 'border-white/10 bg-slate-900/60',
    hover: 'hover:border-white/20',
    elevated: 'rounded-2xl border border-white/10 bg-white/5 shadow-lg',
    interactive: 'rounded-2xl border border-white/10 bg-white/5 hover:border-white/20 transition-colors',
    section: 'rounded-3xl border border-white/5 bg-white/5',
  },

  /**
   * Border radius presets for consistency.
   */
  borderRadius: {
    default: 'rounded-xl',
    large: 'rounded-2xl',
    extraLarge: 'rounded-3xl',
  },

  /**
   * Border opacity presets.
   */
  borderOpacity: {
    subtle: 'border-white/5',
    default: 'border-white/10',
    prominent: 'border-white/20',
  },
} as const;

/**
 * Type-safe helper to combine layout classes.
 */
export function layoutClasses(...classes: (string | undefined | false)[]): string {
  return classes.filter(Boolean).join(' ');
}
