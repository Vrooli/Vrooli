import type { ReactNode } from "react";
import {
  inputBaseClassName as sharedInputBaseClassName,
  textareaBaseClassName as sharedTextareaBaseClassName,
} from "../../../shared/ui/input";

export interface FormFieldProps {
  /** Label text displayed above the input */
  label: string;
  /** Optional help text displayed below the input */
  helpText?: string;
  /** Optional character count display (current/max) */
  charCount?: { current: number; max: number };
  /** Optional error message */
  error?: string;
  /** The input element(s) to render */
  children: ReactNode;
  /** Test ID for the field container */
  testId?: string;
  /** Additional className for the container */
  className?: string;
  /** HTML for attribute to link label to input */
  htmlFor?: string;
}

/**
 * FormField - A standardized wrapper for form inputs.
 *
 * Replaces the repeated label + input + help text pattern used throughout admin pages.
 *
 * @example
 * ```tsx
 * <FormField label="Site Name" helpText="Your primary brand name">
 *   <input
 *     type="text"
 *     value={form.site_name}
 *     onChange={handleInput('site_name')}
 *     className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
 *   />
 * </FormField>
 *
 * <FormField
 *   label="Description"
 *   charCount={{ current: form.description.length, max: 160 }}
 * >
 *   <Textarea ... />
 * </FormField>
 * ```
 */
export function FormField({
  label,
  helpText,
  charCount,
  error,
  children,
  testId,
  className,
  htmlFor,
}: FormFieldProps) {
  return (
    <div className={className} data-testid={testId}>
      <label
        htmlFor={htmlFor}
        className="block text-xs font-semibold uppercase tracking-wide text-slate-400"
      >
        {label}
      </label>
      {children}
      {charCount && (
        <p className="mt-1 text-xs text-slate-500">
          {charCount.current}/{charCount.max} characters (recommended)
        </p>
      )}
      {helpText && !charCount && <p className="mt-1 text-xs text-slate-500">{helpText}</p>}
      {error && <p className="mt-1 text-xs text-rose-400">{error}</p>}
    </div>
  );
}

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
