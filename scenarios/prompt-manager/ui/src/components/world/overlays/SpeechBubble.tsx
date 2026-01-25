/**
 * SpeechBubble - Text bubble for member messages.
 * Shows text content in a chat-bubble style overlay.
 */

import { Html } from '@react-three/drei'
import { X } from 'lucide-react'
import { useOverlayStore, selectMemberSpeechBubbles } from '@/stores/overlayStore'

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
  const bubbles = useOverlayStore((state) => selectMemberSpeechBubbles(state, memberId))
  const hideSpeechBubble = useOverlayStore((state) => state.hideSpeechBubble)

  // Show only the most recent bubble (or stack them with offset)
  const latestBubble = bubbles[bubbles.length - 1]

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
            onClick={() => hideSpeechBubble(latestBubble.id)}
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
