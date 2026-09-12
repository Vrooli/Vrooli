import {
  textareaBaseClassName as sharedTextareaBaseClassName,
} from '../../../shared/ui/textarea';

/**
 * Base input className without spacing.
 * Useful for layouts that manage vertical rhythm separately.
 */
export const inputBaseClassName =
  'min-h-0 w-full rounded-lg border border-white/10 bg-surface-primary/70 px-3 py-2 text-base text-white placeholder:text-slate-500 focus:border-white/20 focus:outline-none focus:ring-1 focus:ring-white/20 disabled:cursor-not-allowed disabled:opacity-50 md:text-sm';

/**
 * Common input className for consistency across admin forms.
 * Use this when creating inputs to maintain visual consistency.
 */
export const inputClassName = `mt-1 ${inputBaseClassName}`;

export const textareaBaseClassName = sharedTextareaBaseClassName;

/**
 * Common textarea className for consistency across admin forms.
 */
export const textareaClassName = "mt-1";
