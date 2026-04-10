/**
 * AgentColorBadge - Abstract visual representation of agent colors.
 *
 * Design: Two rectangles stacked vertically on the left (body bottom, head top)
 * with a column on the right (accent color), forming a square.
 *
 * This component replaces the previous concentric circles pattern used
 * throughout the application for displaying agent colors.
 */

import { cn } from '@/lib/utils'
import { DEFAULT_AGENT_COLORS, type AgentAppearance } from '@/types/agent'

interface AgentColorBadgeProps {
  /** Agent appearance colors */
  appearance?: AgentAppearance | null
  /** Size variant */
  size?: 'xs' | 'sm' | 'md' | 'lg'
  /** Additional class names */
  className?: string
}

const sizeStyles = {
  xs: 'w-5 h-5',
  sm: 'w-8 h-8',
  md: 'w-10 h-10',
  lg: 'w-12 h-12',
}

/**
 * Abstract badge showing agent colors in a professional layout.
 * Layout: [head][accent]
 *         [body][accent]
 */
export function AgentColorBadge({
  appearance,
  size = 'md',
  className,
}: AgentColorBadgeProps) {
  const bodyColor = appearance?.body ?? DEFAULT_AGENT_COLORS.body
  const headColor = appearance?.head ?? DEFAULT_AGENT_COLORS.head
  const accentColor = appearance?.accent ?? DEFAULT_AGENT_COLORS.accent

  return (
    <div
      className={cn(
        'flex flex-shrink-0 rounded overflow-hidden',
        sizeStyles[size],
        className
      )}
    >
      {/* Left column - stacked head (top) and body (bottom) */}
      <div className="flex-1 flex flex-col">
        <div
          className="flex-1"
          style={{ backgroundColor: headColor }}
        />
        <div
          className="flex-1"
          style={{ backgroundColor: bodyColor }}
        />
      </div>
      {/* Right column - accent */}
      <div
        className="w-1/3"
        style={{ backgroundColor: accentColor }}
      />
    </div>
  )
}
