/**
 * SettingsSlider - Reusable slider row for numeric settings.
 * Matches SettingsToggle / SettingsSelect visual style.
 */

import { useCallback } from 'react'
import { cn } from '@/lib/utils'
import { Slider } from '@/components/ui/slider'

interface SettingsSliderProps {
  label: string
  value: number
  onChange: (value: number) => void
  min: number
  max: number
  step: number
  disabled?: boolean
  className?: string
}

export function SettingsSlider({
  label,
  value,
  onChange,
  min,
  max,
  step,
  disabled = false,
  className,
}: SettingsSliderProps) {
  const handleChange = useCallback(
    (values: number[]) => {
      const v = values[0]
      if (v !== undefined) onChange(v)
    },
    [onChange]
  )

  return (
    <div className={cn('flex items-center gap-3', className)}>
      <span className="text-xs text-slate-300 w-24 shrink-0">{label}</span>
      <Slider
        value={[value]}
        onValueChange={handleChange}
        min={min}
        max={max}
        step={step}
        disabled={disabled}
        className="flex-1"
        aria-label={label}
      />
      <span className="text-xs text-slate-400 w-10 text-right font-mono tabular-nums">
        {value.toFixed(2)}
      </span>
    </div>
  )
}
