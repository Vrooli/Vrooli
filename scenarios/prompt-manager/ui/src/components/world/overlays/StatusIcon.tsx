/**
 * StatusIcon - Visual indicator for member status.
 * Shows warning, error, info, or other status icons.
 */

import { Html } from '@react-three/drei'
import { AlertTriangle, XCircle, Info, Loader2, MessageCircle } from 'lucide-react'
import type { MemberStatusType } from '@/types/accessory'

/** Icon mapping for status types */
const STATUS_ICONS: Record<MemberStatusType, typeof AlertTriangle | null> = {
  normal: null,
  warning: AlertTriangle,
  error: XCircle,
  info: Info,
  thinking: Loader2,
  speaking: MessageCircle,
}

/** Color mapping for status types */
const STATUS_COLORS: Record<MemberStatusType, string> = {
  normal: 'text-muted-foreground',
  warning: 'text-yellow-500',
  error: 'text-red-500',
  info: 'text-blue-500',
  thinking: 'text-purple-500',
  speaking: 'text-green-500',
}

/** Animation classes for status types */
const STATUS_ANIMATIONS: Record<MemberStatusType, string> = {
  normal: '',
  warning: 'animate-bounce',
  error: 'animate-pulse',
  info: '',
  thinking: 'animate-spin',
  speaking: 'animate-pulse',
}

interface StatusIconProps {
  /** Status type to display */
  status: MemberStatusType
  /** Position offset from member center */
  position: [number, number, number]
  /** Custom Y offset above the member */
  yOffset?: number
  /** Icon size */
  size?: number
}

/**
 * Renders a status icon that floats above a member.
 * Different status types have different icons and animations.
 */
export function StatusIcon({
  status,
  position,
  yOffset = 1.5,
  size = 20,
}: StatusIconProps) {
  const Icon = STATUS_ICONS[status]

  if (!Icon || status === 'normal') {
    return null
  }

  return (
    <Html
      position={[position[0], position[1] + yOffset, position[2]]}
      center
      style={{
        pointerEvents: 'none',
        userSelect: 'none',
      }}
    >
      <div className={`${STATUS_COLORS[status]} ${STATUS_ANIMATIONS[status]}`}>
        <Icon size={size} strokeWidth={2.5} />
      </div>
    </Html>
  )
}
