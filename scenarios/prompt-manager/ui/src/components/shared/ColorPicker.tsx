/**
 * ColorPicker - Shared color picker component with presets and custom input.
 *
 * Features:
 * - Preset color swatches for quick selection
 * - Native color picker input
 * - Text input for hex values
 */

import { cn } from '@/lib/utils'

/**
 * Default color preset palette.
 */
const COLOR_PRESETS = [
  '#6366f1', // indigo
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#ef4444', // red
  '#f97316', // orange
  '#eab308', // yellow
  '#22c55e', // green
  '#06b6d4', // cyan
  '#3b82f6', // blue
  '#64748b', // slate
]

interface ColorPickerProps {
  /** Label displayed above the picker */
  label: string
  /** Current color value (hex) */
  value: string
  /** Callback when color changes */
  onChange: (color: string) => void
  /** Custom color presets (defaults to COLOR_PRESETS) */
  presets?: string[]
  /** Whether the picker is disabled */
  disabled?: boolean
  /** Additional class name */
  className?: string
}

/**
 * Color picker component with presets and custom input.
 */
export function ColorPicker({
  label,
  value,
  onChange,
  presets = COLOR_PRESETS,
  disabled = false,
  className,
}: ColorPickerProps) {
  return (
    <div className={className}>
      <label className="block text-sm font-medium text-foreground mb-2">
        {label}
      </label>
      <div className="flex items-center gap-3">
        {/* Color presets */}
        <div className="flex gap-1 flex-wrap">
          {presets.map((color) => (
            <button
              key={color}
              type="button"
              onClick={() => !disabled && onChange(color)}
              disabled={disabled}
              className={cn(
                'w-6 h-6 rounded-full border-2 transition-all',
                value === color
                  ? 'border-foreground scale-110'
                  : 'border-transparent hover:scale-105',
                disabled && 'opacity-50 cursor-not-allowed hover:scale-100'
              )}
              style={{ backgroundColor: color }}
              title={color}
            />
          ))}
        </div>

        {/* Custom color inputs */}
        <div className="flex items-center gap-2 ml-auto">
          <input
            type="color"
            value={value}
            onChange={(e) => !disabled && onChange(e.target.value)}
            disabled={disabled}
            className={cn(
              'w-8 h-8 rounded cursor-pointer border border-border',
              disabled && 'opacity-50 cursor-not-allowed'
            )}
          />
          <input
            type="text"
            value={value}
            onChange={(e) => !disabled && onChange(e.target.value)}
            disabled={disabled}
            className={cn(
              'w-20 px-2 py-1 text-xs font-mono',
              'bg-muted border border-border rounded',
              'text-foreground',
              'focus:outline-none focus:ring-1 focus:ring-primary',
              disabled && 'opacity-50 cursor-not-allowed'
            )}
          />
        </div>
      </div>
    </div>
  )
}

/**
 * Compact color picker for use in tighter spaces.
 * Shows only the current color and a picker, no presets.
 */
interface CompactColorPickerProps {
  /** Current color value (hex) */
  value: string
  /** Callback when color changes */
  onChange: (color: string) => void
  /** Whether the picker is disabled */
  disabled?: boolean
  /** Size of the color swatch */
  size?: 'sm' | 'md' | 'lg'
  /** Additional class name */
  className?: string
}

export function CompactColorPicker({
  value,
  onChange,
  disabled = false,
  size = 'md',
  className,
}: CompactColorPickerProps) {
  const sizeClasses = {
    sm: 'w-6 h-6',
    md: 'w-8 h-8',
    lg: 'w-10 h-10',
  }

  return (
    <div className={cn('relative inline-block', className)}>
      <input
        type="color"
        value={value}
        onChange={(e) => !disabled && onChange(e.target.value)}
        disabled={disabled}
        className={cn(
          sizeClasses[size],
          'rounded cursor-pointer border border-border',
          disabled && 'opacity-50 cursor-not-allowed'
        )}
      />
    </div>
  )
}
