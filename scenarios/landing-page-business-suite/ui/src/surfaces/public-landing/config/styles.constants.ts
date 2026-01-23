/**
 * Standardized style constants for public landing page sections.
 *
 * Use these constants to maintain consistent styling across all
 * landing page sections and components.
 */

/**
 * Card style variants for public landing sections.
 * Unified border radius, border opacity, and backgrounds.
 */
export const PUBLIC_CARD_STYLES = {
  /**
   * Default card style for standard sections
   */
  default: 'rounded-2xl border border-white/10 bg-white/5',

  /**
   * Elevated card with shadow for prominent elements
   */
  elevated: 'rounded-2xl border border-white/10 bg-white/5 shadow-lg',

  /**
   * Interactive card that responds to hover
   */
  interactive: 'rounded-2xl border border-white/10 bg-white/5 hover:border-white/20 transition-colors',

  /**
   * Large section container with softer borders
   */
  section: 'rounded-3xl border border-white/5 bg-white/5',

  /**
   * Gradient card for hero sections
   */
  gradient: 'rounded-2xl border border-white/10 bg-gradient-to-b from-white/[0.08] to-white/[0.02]',

  /**
   * Subtle card for secondary content
   */
  subtle: 'rounded-xl border border-white/5 bg-white/[0.02]',
} as const;

/**
 * Border radius presets for consistency.
 */
export const PUBLIC_BORDER_RADIUS = {
  small: 'rounded-lg',
  default: 'rounded-xl',
  large: 'rounded-2xl',
  extraLarge: 'rounded-3xl',
  full: 'rounded-full',
} as const;

/**
 * Border opacity presets.
 */
export const PUBLIC_BORDER_OPACITY = {
  subtle: 'border-white/5',
  light: 'border-white/8',
  default: 'border-white/10',
  prominent: 'border-white/20',
} as const;

/**
 * Background opacity presets.
 */
export const PUBLIC_BACKGROUND_OPACITY = {
  subtle: 'bg-white/[0.02]',
  light: 'bg-white/5',
  default: 'bg-white/[0.08]',
  prominent: 'bg-white/10',
} as const;

/**
 * Section spacing presets for consistent vertical rhythm.
 */
export const PUBLIC_SECTION_SPACING = {
  tight: 'py-12',
  default: 'py-16',
  loose: 'py-20',
  extraLoose: 'py-24',
} as const;

/**
 * Container max-width presets.
 */
export const PUBLIC_MAX_WIDTH = {
  narrow: 'max-w-3xl',
  default: 'max-w-4xl',
  wide: 'max-w-5xl',
  extraWide: 'max-w-6xl',
  full: 'max-w-7xl',
} as const;

/**
 * Helper to combine style classes.
 */
export function publicStyles(...classes: (string | undefined | false | null)[]): string {
  return classes.filter(Boolean).join(' ');
}
