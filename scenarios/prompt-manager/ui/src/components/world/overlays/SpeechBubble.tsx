/**
 * SpeechBubble - Text bubble for member messages.
 * Shows text content in a chat-bubble style overlay.
 */

import { useCallback, useMemo } from 'react'
import { Html } from '@react-three/drei'
import { X } from 'lucide-react'
import { useOverlayStore } from '@/stores/overlayStore'
import type { SpeechBubble as SpeechBubbleType } from '@/stores/overlayStore'

interface SpeechBubbleProps {
  /** Member ID for state lookup */
  memberId: string
  /** Position offset from member center */
  position: [number, number, number]
  /** Custom Y offset above the member */
  yOffset?: number
  /** Maximum width of the bubble */
  maxWidth?: number
}

/**
 * Renders speech bubbles for a member.
 * Multiple bubbles can stack vertically.
 */
export function SpeechBubble({
  memberId,
  position,
  yOffset = 1.4,
  maxWidth = 200,
}: SpeechBubbleProps) {
  // Get all speech bubbles from store (stable reference)
  const allBubbles = useOverlayStore((state) => state.speechBubbles)
  const hideSpeechBubble = useOverlayStore((state) => state.hideSpeechBubble)

  // Filter bubbles for this member - memoized to avoid recreating array
  const memberBubbles = useMemo(() => {
    if (!memberId || !Array.isArray(allBubbles)) return []
    return allBubbles.filter((b: SpeechBubbleType) => b.memberId === memberId)
  }, [allBubbles, memberId])

  // Show only the most recent bubble
  const latestBubble = memberBubbles.length > 0 ? memberBubbles[memberBubbles.length - 1] : null

  // Memoize the close handler
  const handleClose = useCallback(() => {
    if (latestBubble) {
      hideSpeechBubble(latestBubble.id)
    }
  }, [hideSpeechBubble, latestBubble])

  // Early return for empty bubbles
  if (!latestBubble) {
    return null
  }

  return (
    <Html
      position={[position[0], position[1] + yOffset, position[2]]}
      center
      style={{
        pointerEvents: 'auto',
        userSelect: 'none',
      }}
    >
      <div
        className="
          relative
          px-3 py-2
          bg-card/95 backdrop-blur-sm
          rounded-lg
          shadow-lg
          border border-border/50
          text-sm
        "
        style={{ maxWidth }}
      >
        {/* Close button for non-temporary bubbles */}
        {!latestBubble.temporary && (
          <button
            onClick={handleClose}
            className="
              absolute -top-2 -right-2
              w-5 h-5
              flex items-center justify-center
              bg-muted rounded-full
              hover:bg-muted-foreground/20
              transition-colors
            "
          >
            <X size={12} />
          </button>
        )}

        {/* Message text */}
        <p className="text-foreground">{latestBubble.text}</p>

        {/* Bubble tail */}
        <div
          className="
            absolute -bottom-2 left-1/2 -translate-x-1/2
            w-0 h-0
            border-l-[8px] border-l-transparent
            border-r-[8px] border-r-transparent
            border-t-[8px] border-t-card/95
          "
        />
      </div>
    </Html>
  )
}
