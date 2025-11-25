import { type ReactNode } from 'react'
import { AlertCircle, CheckCircle2, Info } from 'lucide-react'
import { cn } from '../../lib/utils'

export interface FormFieldProps {
  label: string
  id?: string
  children: ReactNode
  description?: string
  error?: string
  success?: string
  hint?: string
  required?: boolean
  className?: string
}

/**
 * FormField component - Provides inline validation feedback and contextual help
 *
 * Design principles:
 * - Clear label with optional required indicator
 * - Helpful description text below label
 * - Inline error/success messages with appropriate icons
 * - Hint text for additional guidance
 * - Proper accessibility with htmlFor and aria-describedby
 */
export function FormField({
  label,
  id,
  children,
  description,
  error,
  success,
  hint,
  required,
  className,
}: FormFieldProps) {
  const errorId = error && id ? `${id}-error` : undefined
  const descriptionId = description && id ? `${id}-description` : undefined
  const hintId = hint && id ? `${id}-hint` : undefined

  return (
    <div className={cn('space-y-2', className)}>
      <div className="space-y-1">
        <label
          htmlFor={id}
          className="text-sm font-medium text-slate-900 flex items-center gap-1.5"
        >
          {label}
          {required && (
            <span className="text-red-500" aria-label="required">
              *
            </span>
          )}
        </label>

        {description && (
          <p
            id={descriptionId}
            className="text-xs text-slate-600 leading-relaxed"
          >
            {description}
          </p>
        )}
      </div>

      {children}

      {hint && !error && !success && (
        <div
          id={hintId}
          className="flex items-start gap-2 text-xs text-slate-600 leading-relaxed"
        >
          <Info size={14} className="shrink-0 mt-0.5 text-slate-400" />
          <span>{hint}</span>
        </div>
      )}

      {error && (
        <div
          id={errorId}
          className="flex items-start gap-2 rounded-lg bg-red-50 border border-red-200 px-3 py-2 text-xs text-red-700"
          role="alert"
        >
          <AlertCircle size={14} className="shrink-0 mt-0.5" />
          <span>{error}</span>
        </div>
      )}

      {success && !error && (
        <div className="flex items-start gap-2 rounded-lg bg-emerald-50 border border-emerald-200 px-3 py-2 text-xs text-emerald-700">
          <CheckCircle2 size={14} className="shrink-0 mt-0.5" />
          <span>{success}</span>
        </div>
      )}
    </div>
  )
}
