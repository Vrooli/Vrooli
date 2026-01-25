/**
 * NameTag - Billboard name label for members.
 * Floats above the member and always faces the camera.
 */

import { Html } from '@react-three/drei'
import { useOverlayStore } from '@/stores/overlayStore'

interface NameTagProps {
  /** Member ID for visibility checks */
  memberId: string
  /** Display name */
  name: string
  /** Position offset from member center */
  position: [number, number, number]
  /** Whether the member is currently hovered */
  isHovered?: boolean
  /** Custom Y offset above the member */
  yOffset?: number
}

/**
 * Renders a name tag that floats above a member.
 * Visibility is controlled by the overlay store settings.
 */
export function NameTag({
  memberId,
  name,
  position,
  isHovered = false,
  yOffset = 1.2,
}: NameTagProps) {
  const shouldShow = useOverlayStore((state) => state.shouldShowNameTag(memberId, isHovered))
  const overlaysVisible = useOverlayStore((state) => state.overlaysVisible)

  if (!overlaysVisible || !shouldShow) {
    return null
  }

  return (
    <Html
      position={[position[0], position[1] + yOffset, position[2]]}
      center
      distanceFactor={10}
      occlude
      style={{
        pointerEvents: 'none',
        userSelect: 'none',
      }}
    >
      <div
        className={`
          px-2 py-1
          bg-card/90 backdrop-blur-sm
          rounded-full
          text-sm font-medium
          shadow-lg
          border border-border/50
          transition-opacity duration-200
          ${isHovered ? 'opacity-100' : 'opacity-80'}
        `}
      >
        {name}
      </div>
    </Html>
  )
}
