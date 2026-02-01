/**
 * ThinkingBubble - Animated thinking indicator.
 * Shows animated dots when a agent is "thinking".
 * Hover to reveal the full label text.
 */

import { useState } from 'react'
import { Html } from '@react-three/drei'
import { useOverlayStore } from '@/stores/overlayStore'

interface ThinkingBubbleProps {
  /** Member ID for state lookup */
  agentId: string
  /** Position offset from agent center */
  position: [number, number, number]
  /** Custom Y offset above the agent */
  yOffset?: number
}

/**
 * Renders an animated thinking bubble with bouncing dots.
 * Visibility is controlled by the overlay store thinking state.
 * Hover over the bubble to see the full label text if available.
 */
export function ThinkingBubble({
  agentId,
  position,
  yOffset = 1.3,
}: ThinkingBubbleProps) {
  const [showLabel, setShowLabel] = useState(false)

  // Always call hook with a stable key (empty string for invalid agentId)
  const thinkingState = useOverlayStore((state) =>
    agentId ? state.thinkingStates[agentId] : undefined
  )

  // Early return for invalid agentId or not thinking
  if (!agentId || !thinkingState?.isThinking) {
    return null
  }

  const hasLabel = !!thinkingState.label

  return (
    <Html
      position={[position[0], position[1] + yOffset, position[2]]}
      center
      style={{
        pointerEvents: hasLabel ? 'auto' : 'none',
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
          cursor-default
          transition-all duration-200
        "
        onPointerEnter={() => setShowLabel(true)}
        onPointerLeave={() => setShowLabel(false)}
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

        {/* Label shown on hover (or always if no hover mechanism needed) */}
        {thinkingState.label && showLabel && (
          <span className="ml-2 text-xs text-muted-foreground whitespace-nowrap animate-in fade-in duration-200">
            {thinkingState.label}
          </span>
        )}
      </div>
    </Html>
  )
}
