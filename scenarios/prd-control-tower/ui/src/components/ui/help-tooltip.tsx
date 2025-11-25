import { HelpCircle } from 'lucide-react'
import { Tooltip } from './tooltip'
import { cn } from '../../lib/utils'

interface HelpTooltipProps {
  content: string
  side?: 'top' | 'bottom' | 'left' | 'right'
  className?: string
  iconSize?: number
}

/**
 * Help icon with tooltip for contextual assistance.
 * Provides subtle, non-intrusive help without cluttering the UI.
 */
export function HelpTooltip({
  content,
  side = 'top',
  className,
  iconSize = 14
}: HelpTooltipProps) {
  return (
    <Tooltip content={content} side={side}>
      <button
        type="button"
        className={cn(
          'inline-flex items-center justify-center text-slate-400 hover:text-slate-600 transition-colors cursor-help focus:outline-none focus:ring-2 focus:ring-violet-300 focus:ring-offset-1 rounded-full',
          className
        )}
        aria-label="Help"
        onClick={(e) => e.stopPropagation()}
      >
        <HelpCircle size={iconSize} />
      </button>
    </Tooltip>
  )
}
