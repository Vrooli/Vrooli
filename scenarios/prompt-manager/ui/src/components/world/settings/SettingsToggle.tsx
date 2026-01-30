/**
 * SettingsToggle - Reusable toggle switch for boolean settings.
 */

import { cn } from '@/lib/utils'

interface SettingsToggleProps {
  label: string
  value: boolean
  onChange: (value: boolean) => void
  disabled?: boolean
  className?: string
}

export function SettingsToggle({
  label,
  value,
  onChange,
  disabled = false,
  className,
}: SettingsToggleProps) {
  return (
    <div className={cn('flex items-center justify-between', className)}>
      <span className="text-xs text-slate-300">{label}</span>
      <button
        type="button"
        role="switch"
        aria-checked={value}
        disabled={disabled}
        onClick={() => onChange(!value)}
        className={cn(
          'relative h-5 w-9 rounded-full transition-colors',
          value ? 'bg-indigo-500' : 'bg-slate-600',
          disabled && 'opacity-50 cursor-not-allowed'
        )}
      >
        <span
          className={cn(
            'absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white transition-transform',
            value && 'translate-x-4'
          )}
        />
      </button>
    </div>
  )
}
