import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { AlertTriangle, HeartPulse, Loader2, Pause, Play } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { HeartbeatControlStatus } from '@/services/heartbeatService'

interface HeartbeatControlPopoverProps {
  status: HeartbeatControlStatus | null
  isLoading?: boolean
  onPause?: () => Promise<void>
  onResume?: () => Promise<void>
  className?: string
}

function statusLabel(status?: string) {
  switch (status) {
    case 'warning-idle-soon':
      return 'Idle soon'
    case 'paused-auto-idle':
      return 'Auto-paused'
    case 'paused-manual':
      return 'Paused'
    default:
      return 'Heartbeat'
  }
}

function statusClasses(status?: string) {
  switch (status) {
    case 'warning-idle-soon':
      return 'bg-amber-500/15 text-amber-600 dark:text-amber-400 hover:bg-amber-500/25'
    case 'paused-auto-idle':
    case 'paused-manual':
      return 'bg-red-500/15 text-red-600 dark:text-red-400 hover:bg-red-500/25'
    default:
      return 'bg-sky-500/15 text-sky-600 dark:text-sky-400 hover:bg-sky-500/25'
  }
}

function formatTime(value?: string | null) {
  if (!value) return 'Not recorded'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

export function HeartbeatControlPopover({
  status,
  isLoading = false,
  onPause,
  onResume,
  className,
}: HeartbeatControlPopoverProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [isMutating, setIsMutating] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const [position, setPosition] = useState({ top: 0, left: 0, width: 320 })
  const isPaused = status?.status === 'paused-auto-idle' || status?.status === 'paused-manual'
  const isWarning = status?.status === 'warning-idle-soon'

  useEffect(() => {
    if (!isOpen) return
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    const timeoutId = setTimeout(() => document.addEventListener('mousedown', handleClickOutside), 0)
    return () => {
      clearTimeout(timeoutId)
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  useEffect(() => {
    if (!isOpen) return
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setIsOpen(false)
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [isOpen])

  useLayoutEffect(() => {
    if (!isOpen || !triggerRef.current) return
    const trigger = triggerRef.current.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight
    const width = viewportWidth < 640 ? viewportWidth - 16 : Math.min(320, viewportWidth - 16)
    const estimatedHeight = Math.min(dropdownRef.current?.scrollHeight ?? 360, viewportHeight - 16)
    let left = trigger.left
    let top = trigger.bottom + 4
    if (left + width > viewportWidth - 8) left = viewportWidth - width - 8
    if (left < 8) left = 8
    if (top + estimatedHeight > viewportHeight - 8) top = Math.max(8, trigger.top - estimatedHeight - 4)
    setPosition({ top, left, width })
  }, [isOpen, status?.status])

  const runAction = async (action?: () => Promise<void>) => {
    if (!action) return
    setIsMutating(true)
    try {
      await action()
    } finally {
      setIsMutating(false)
    }
  }

  return (
    <div ref={menuRef} className={cn('relative', className)}>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        className={cn(
          'flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium transition-colors',
          statusClasses(status?.status)
        )}
        title={status ? `Heartbeat control: ${statusLabel(status.status)}` : 'Heartbeat control'}
      >
        {isLoading ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
        ) : isPaused || isWarning ? (
          <AlertTriangle className="h-3.5 w-3.5" />
        ) : (
          <HeartPulse className="h-3.5 w-3.5" />
        )}
        <span>{statusLabel(status?.status)}</span>
      </button>

      {isOpen && (
        <div
          ref={dropdownRef}
          style={{
            position: 'fixed',
            top: position.top,
            left: position.left,
            width: position.width,
            maxWidth: 'calc(100vw - 16px)',
            maxHeight: 'calc(100vh - 16px)',
          }}
          className="z-50 overflow-y-auto bg-popover border border-border rounded-lg shadow-lg animate-in fade-in-0 zoom-in-95 duration-100"
        >
          <div className="px-3 py-2 border-b border-border">
            <div className="flex items-center justify-between gap-2">
              <span className="text-xs font-semibold text-foreground">Heartbeat Control</span>
              <span className={cn('text-[10px] font-medium uppercase tracking-wide', isPaused ? 'text-red-500' : isWarning ? 'text-amber-500' : 'text-sky-500')}>
                {statusLabel(status?.status)}
              </span>
            </div>
          </div>
          <div className="space-y-2 px-3 py-3 text-xs">
            <div>
              <div className="text-muted-foreground">Last human engagement</div>
              <div className="font-medium text-foreground">{formatTime(status?.lastHumanEngagementAt)}</div>
              {status?.lastHumanEngagementReason && (
                <div className="text-muted-foreground">{status.lastHumanEngagementReason}</div>
              )}
            </div>
            {status?.pausedReason && (
              <div>
                <div className="text-muted-foreground">Pause reason</div>
                <div className="font-medium text-foreground">{status.pausedReason}</div>
              </div>
            )}
            <div className="grid grid-cols-2 gap-2">
              <div>
                <div className="text-muted-foreground">Warning</div>
                <div className="font-medium text-foreground">{status?.effectivePolicy.warningAfterDaysWithoutHumanEngagement ?? '-'}d</div>
              </div>
              <div>
                <div className="text-muted-foreground">Auto-pause</div>
                <div className="font-medium text-foreground">{status?.effectivePolicy.pauseAfterDaysWithoutHumanEngagement ?? '-'}d</div>
              </div>
            </div>
            {status?.autoPauseAt && (
              <div>
                <div className="text-muted-foreground">Auto-pause at</div>
                <div className="font-medium text-foreground">{formatTime(status.autoPauseAt)}</div>
              </div>
            )}
            <div className="flex items-center gap-2 pt-1">
              {isPaused ? (
                <button
                  type="button"
                  onClick={() => void runAction(onResume)}
                  disabled={isMutating}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-60"
                >
                  {isMutating ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Play className="h-3.5 w-3.5" />}
                  Resume
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => void runAction(onPause)}
                  disabled={isMutating}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-md bg-muted text-foreground hover:bg-muted/80 disabled:opacity-60"
                >
                  {isMutating ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Pause className="h-3.5 w-3.5" />}
                  Pause
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
