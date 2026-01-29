import {
  inputBaseClassName as sharedInputBaseClassName,
  textareaBaseClassName as sharedTextareaBaseClassName,
} from '../../../shared/ui/input';

/**
 * Base input className without spacing.
 * Useful for layouts that manage vertical rhythm separately.
 */
export const inputBaseClassName = sharedInputBaseClassName;

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
