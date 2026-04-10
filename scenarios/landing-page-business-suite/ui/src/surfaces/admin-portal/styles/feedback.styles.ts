/**
 * Semantic variant to Tailwind class mappings for feedback display.
 * This separates UI styling concerns from service layer logic.
 */

export type FeedbackVariant = 'warning' | 'danger' | 'info' | 'primary' | 'neutral' | 'success';

/**
 * Maps semantic variants to Tailwind CSS classes.
 */
export const VARIANT_STYLES: Record<FeedbackVariant, string> = {
  warning: 'text-amber-400 bg-amber-500/10 border-amber-500/20',
  danger: 'text-red-400 bg-red-500/10 border-red-500/20',
  info: 'text-blue-400 bg-blue-500/10 border-blue-500/20',
  primary: 'text-purple-400 bg-purple-500/10 border-purple-500/20',
  neutral: 'text-slate-400 bg-slate-500/10 border-slate-500/20',
  success: 'text-green-400 bg-green-500/10 border-green-500/20',
};

/**
 * Get Tailwind classes for a semantic variant.
 */
export function getVariantStyles(variant: FeedbackVariant): string {
  return VARIANT_STYLES[variant];
}

/**
 * Feedback type to semantic variant mapping.
 */
export const TYPE_VARIANTS: Record<string, FeedbackVariant> = {
  refund: 'warning',
  bug: 'danger',
  feature: 'info',
  general: 'primary',
};

/**
 * Feedback status to semantic variant mapping.
 */
export const STATUS_VARIANTS: Record<string, FeedbackVariant> = {
  pending: 'neutral',
  in_progress: 'info',
  resolved: 'success',
  rejected: 'danger',
};

/**
 * Get Tailwind classes for a feedback type.
 */
export function getTypeStyles(type: string): string {
  const variant = TYPE_VARIANTS[type] ?? 'primary';
  return VARIANT_STYLES[variant];
}

/**
 * Get Tailwind classes for a feedback status.
 */
export function getStatusStyles(status: string): string {
  const variant = STATUS_VARIANTS[status] ?? 'neutral';
  return VARIANT_STYLES[variant];
}
