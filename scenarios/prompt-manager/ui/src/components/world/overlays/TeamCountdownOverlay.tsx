/**
 * TeamCountdownOverlay — shows a countdown (upcoming) or elapsed timer (running)
 * above a furniture piece. Clicking navigates to the team editor.
 */

import { useState, useEffect } from 'react'
import { Html } from '@react-three/drei'
import type { TeamActivity } from '@/stores/teamActivityStore'

interface TeamCountdownOverlayProps {
  activity: TeamActivity
  position: [number, number, number]
  yOffset: number
  onClick: () => void
}

function formatTimer(ms: number): string {
  const totalSeconds = Math.max(0, Math.floor(Math.abs(ms) / 1000))
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

export function TeamCountdownOverlay({
  activity,
  position,
  yOffset,
  onClick,
}: TeamCountdownOverlayProps) {
  const [display, setDisplay] = useState('')
  const [showName, setShowName] = useState(false)

  useEffect(() => {
    function update() {
      const now = Date.now()
      const ref = new Date(activity.referenceTime).getTime()
      if (activity.status === 'upcoming') {
        const remaining = ref - now
        setDisplay(`T-${formatTimer(remaining)}`)
      } else {
        const elapsed = now - ref
        setDisplay(`T+${formatTimer(elapsed)}`)
      }
    }
    update()
    const id = setInterval(update, 1000)
    return () => clearInterval(id)
  }, [activity.referenceTime, activity.status])

  const isUpcoming = activity.status === 'upcoming'

  return (
    <Html
      position={[position[0], position[1] + yOffset, position[2]]}
      center
      zIndexRange={[10, 0]}
      style={{ pointerEvents: 'auto', userSelect: 'none' }}
    >
      <div
        className={`
          flex items-center gap-2
          px-3 py-1.5
          bg-card/95 backdrop-blur-sm
          rounded-2xl
          shadow-lg
          border border-border/50
          cursor-pointer
          transition-all duration-200
          hover:scale-105
          whitespace-nowrap
        `}
        onClick={(e) => {
          e.stopPropagation()
          onClick()
        }}
        onPointerEnter={() => setShowName(true)}
        onPointerLeave={() => setShowName(false)}
      >
        {/* Status dot */}
        <span
          className={`w-2 h-2 rounded-full flex-shrink-0 ${
            isUpcoming ? 'bg-amber-400' : 'bg-emerald-400'
          }`}
        />

        {/* Timer */}
        <span
          className={`text-sm font-mono font-medium ${
            isUpcoming ? 'text-amber-300' : 'text-emerald-300'
          }`}
        >
          {display}
        </span>

        {/* Team name on hover */}
        {showName && (
          <span className="ml-1 text-xs text-muted-foreground whitespace-nowrap animate-in fade-in duration-200">
            {activity.teamName}
          </span>
        )}
      </div>
    </Html>
  )
}
