/**
 * TeamCountdownOverlay — shows a countdown (upcoming) or elapsed timer (running)
 * above a furniture piece. Clicking navigates to the team editor.
 */

import { useMemo, useState, useSyncExternalStore } from 'react'
import { Html } from '@react-three/drei'
import type { TeamActivity } from '@/stores/teamActivityStore'

interface TeamCountdownOverlayProps {
  activity: TeamActivity
  position: [number, number, number]
  yOffset: number
  onClick: () => void
  pendingDecisionCount?: number
}

let nowValue = Date.now()
let tickerId: ReturnType<typeof setInterval> | null = null
const tickerListeners = new Set<() => void>()

function emitTick() {
  nowValue = Date.now()
  for (const listener of tickerListeners) {
    listener()
  }
}

function subscribeTick(listener: () => void) {
  tickerListeners.add(listener)
  if (tickerId === null) {
    tickerId = setInterval(emitTick, 1000)
  }
  return () => {
    tickerListeners.delete(listener)
    if (tickerListeners.size === 0 && tickerId !== null) {
      clearInterval(tickerId)
      tickerId = null
    }
  }
}

function getTickSnapshot() {
  return nowValue
}

function useNowTick() {
  return useSyncExternalStore(subscribeTick, getTickSnapshot, getTickSnapshot)
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
  pendingDecisionCount,
}: TeamCountdownOverlayProps) {
  const now = useNowTick()
  const [showName, setShowName] = useState(false)
  const display = useMemo(() => {
    const referenceTimeMs = new Date(activity.referenceTime).getTime()
    if (activity.status === 'upcoming') {
      return `T-${formatTimer(referenceTimeMs - now)}`
    }
    return `T+${formatTimer(now - referenceTimeMs)}`
  }, [activity.referenceTime, activity.status, now])

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

        {/* Pending decision badge */}
        {pendingDecisionCount != null && pendingDecisionCount > 0 && (
          <span className="text-[10px] font-semibold bg-amber-500/20 text-amber-400 px-1.5 rounded-full">
            {pendingDecisionCount}
          </span>
        )}

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
