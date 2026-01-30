/**
 * SettingsSelect - Reusable dropdown for enum settings.
 */

import { cn } from '@/lib/utils'

interface SettingsSelectOption<T> {
  value: T
  label: string
}

interface SettingsSelectProps<T extends string | number> {
  label: string
  value: T
  options: SettingsSelectOption<T>[]
  onChange: (value: T) => void
  disabled?: boolean
  className?: string
}

export function SettingsSelect<T extends string | number>({
  label,
  value,
  options,
  onChange,
  disabled = false,
  className,
}: SettingsSelectProps<T>) {
  return (
    <div className={cn('flex items-center justify-between', className)}>
      <span className="text-xs text-slate-300">{label}</span>
      <select
        value={value}
        onChange={(e) => {
          const newValue = typeof value === 'number'
            ? Number(e.target.value) as T
            : e.target.value as T
          onChange(newValue)
        }}
        disabled={disabled}
        className={cn(
          'px-2 py-1 text-xs rounded',
          'bg-slate-700 border border-slate-600 text-slate-300',
          'focus:outline-none focus:ring-1 focus:ring-indigo-500',
          disabled && 'opacity-50 cursor-not-allowed'
        )}
      >
        {options.map((option) => (
          <option key={String(option.value)} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  )
}
