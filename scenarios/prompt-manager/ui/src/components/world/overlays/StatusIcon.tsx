/**
 * StatusIcon - Visual indicator for agent status.
 * Shows warning, error, info, or other status icons.
 * Hover to reveal an optional message tooltip.
 */

import { useState } from 'react'
import { Html } from '@react-three/drei'
import { AlertTriangle, XCircle, Info, Loader2, MessageCircle, Scale } from 'lucide-react'
import type { AgentStatusType } from '@/types/accessory'

/** Icon mapping for status types */
const STATUS_ICONS: Record<AgentStatusType, typeof AlertTriangle | null> = {
  normal: null,
  warning: AlertTriangle,
  error: XCircle,
  info: Info,
  thinking: Loader2,
  speaking: MessageCircle,
  'pending-work': Scale,
}

/** Color mapping for status types */
const STATUS_COLORS: Record<AgentStatusType, string> = {
  normal: 'text-muted-foreground',
  warning: 'text-yellow-500',
  error: 'text-red-500',
  info: 'text-blue-500',
  thinking: 'text-purple-500',
  speaking: 'text-green-500',
  'pending-work': 'text-amber-500',
}

/** Background color mapping for tooltips */
const STATUS_BG_COLORS: Record<AgentStatusType, string> = {
  normal: 'bg-muted',
  warning: 'bg-yellow-500/10 border-yellow-500/30',
  error: 'bg-red-500/10 border-red-500/30',
  info: 'bg-blue-500/10 border-blue-500/30',
  thinking: 'bg-purple-500/10 border-purple-500/30',
  speaking: 'bg-green-500/10 border-green-500/30',
  'pending-work': 'bg-amber-500/10 border-amber-500/30',
}

/** Animation classes for status types */
const STATUS_ANIMATIONS: Record<AgentStatusType, string> = {
  normal: '',
  warning: 'animate-bounce',
  error: 'animate-pulse',
  info: '',
  thinking: 'animate-spin',
  speaking: 'animate-pulse',
  'pending-work': '',
}

interface StatusIconProps {
  /** Status type to display */
  status: AgentStatusType
  /** Position offset from agent center */
  position: [number, number, number]
  /** Custom Y offset above the agent */
  yOffset?: number
  /** Icon size */
  size?: number
  /** Optional message to show on hover */
  message?: string
  /** Click handler (e.g. focus camera on agent) */
  onClick?: () => void
}

/**
 * Renders a status icon that floats above an agent.
 * Different status types have different icons and animations.
 * Hover over the icon to see the optional message tooltip.
 */
export function StatusIcon({
  status,
  position,
  yOffset = 1.5,
  size = 20,
  message,
  onClick,
}: StatusIconProps) {
  const [showTooltip, setShowTooltip] = useState(false)
  const Icon = STATUS_ICONS[status]

  if (!Icon || status === 'normal') {
    return null
  }

  const isInteractive = !!message || !!onClick

  return (
    <Html
      position={[position[0], position[1] + yOffset, position[2]]}
      center
      zIndexRange={[10, 0]}
      style={{
        pointerEvents: isInteractive ? 'auto' : 'none',
        userSelect: 'none',
      }}
    >
      <div
        className={`
          relative flex flex-col items-center
          ${STATUS_COLORS[status]} ${STATUS_ANIMATIONS[status]}
          ${onClick ? 'cursor-pointer' : 'cursor-default'}
        `}
        onClick={onClick ? (e) => { e.stopPropagation(); onClick() } : undefined}
        onPointerEnter={() => setShowTooltip(true)}
        onPointerLeave={() => setShowTooltip(false)}
      >
        <Icon size={size} strokeWidth={2.5} />

        {/* Tooltip shown on hover */}
        {message && showTooltip && (
          <div
            className={`
              absolute top-full mt-1
              px-2 py-1
              text-xs whitespace-nowrap
              rounded-md border
              ${STATUS_BG_COLORS[status]}
              ${STATUS_COLORS[status]}
              animate-in fade-in duration-200
              shadow-lg
            `}
          >
            {message}
          </div>
        )}
      </div>
    </Html>
  )
}
