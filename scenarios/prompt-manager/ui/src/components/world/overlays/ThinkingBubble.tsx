/**
 * ThinkingBubble - Animated thinking indicator.
 * Shows animated dots when a member is "thinking".
 */

import { Html } from '@react-three/drei'
import { useOverlayStore } from '@/stores/overlayStore'

interface ThinkingBubbleProps {
  /** Member ID for state lookup */
  memberId: string
  /** Position offset from member center */
  position: [number, number, number]
  /** Custom Y offset above the member */
  yOffset?: number
}

/**
 * Renders an animated thinking bubble with bouncing dots.
 * Visibility is controlled by the overlay store thinking state.
 */
export function ThinkingBubble({
  memberId,
  position,
  yOffset = 1.3,
}: ThinkingBubbleProps) {
  const thinkingState = useOverlayStore((state) => state.thinkingStates[memberId])

  if (!thinkingState?.isThinking) {
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
      <div
        className="
          flex items-center gap-1
          px-3 py-2
          bg-card/95 backdrop-blur-sm
          rounded-2xl
          shadow-lg
          border border-border/50
        "
      >
        {/* Animated dots */}
        <span
          className="w-2 h-2 bg-primary rounded-full animate-bounce"
          style={{ animationDelay: '0ms' }}
        />
        <span
          className="w-2 h-2 bg-primary rounded-full animate-bounce"
          style={{ animationDelay: '150ms' }}
        />
        <span
          className="w-2 h-2 bg-primary rounded-full animate-bounce"
          style={{ animationDelay: '300ms' }}
        />

        {/* Optional label */}
        {thinkingState.label && (
          <span className="ml-2 text-xs text-muted-foreground">
            {thinkingState.label}
          </span>
        )}
      </div>
    </Html>
  )
}
