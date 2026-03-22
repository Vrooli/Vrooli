/**
 * PendingDecisionsPopover - Shows a badge with count of pending decisions and a
 * dropdown listing them grouped by team. Follows RunningAgentsPopover pattern.
 */
import { useState, useEffect, useRef, useLayoutEffect } from 'react'
import { Scale, Check, X, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { usePendingDecisions } from '@/hooks/usePendingDecisions'
import type { PendingDecisionTeamGroup } from '@/services/heartbeatService'
import { formatRelativePastTime } from '@/lib/timeUtils'

interface PendingDecisionsPopoverProps {
  onNavigateToDecision: (teamId: string) => void
  /** Pre-fetched data from usePendingDecisionSync (eliminates duplicate polling) */
  groupedByTeam?: PendingDecisionTeamGroup[]
  count?: number
  acceptDecision?: (teamId: string, decisionId: string) => Promise<void>
  rejectDecision?: (teamId: string, decisionId: string) => Promise<void>
  processingIds?: Set<string>
  className?: string
}

export function PendingDecisionsPopover({
  onNavigateToDecision,
  groupedByTeam: groupedByTeamProp,
  count: countProp,
  acceptDecision: acceptProp,
  rejectDecision: rejectProp,
  processingIds: processingIdsProp,
  className,
}: PendingDecisionsPopoverProps) {
  const hasExternalData = groupedByTeamProp !== undefined
  const fallback = usePendingDecisions()

  const groupedByTeam = hasExternalData ? groupedByTeamProp : fallback.groupedByTeam
  const count = hasExternalData ? (countProp ?? 0) : fallback.count
  const acceptDecision = hasExternalData ? (acceptProp ?? fallback.acceptDecision) : fallback.acceptDecision
  const rejectDecision = hasExternalData ? (rejectProp ?? fallback.rejectDecision) : fallback.rejectDecision
  const processingIds = hasExternalData ? (processingIdsProp ?? fallback.processingIds) : fallback.processingIds

  const [isOpen, setIsOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const [position, setPosition] = useState<{ top: number; left: number; width: number }>({ top: 0, left: 0, width: 320 })

  // Click-outside to close
  useEffect(() => {
    if (!isOpen) return
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    const timeoutId = setTimeout(() => {
      document.addEventListener('mousedown', handleClickOutside)
    }, 0)
    return () => {
      clearTimeout(timeoutId)
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  useLayoutEffect(() => {
    if (!isOpen || !triggerRef.current) return

    const trigger = triggerRef.current.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight
    const width = viewportWidth < 640 ? viewportWidth - 16 : Math.min(320, viewportWidth - 16)
    const estimatedHeight = Math.min(dropdownRef.current?.scrollHeight ?? 400, viewportHeight - 16)

    let left = trigger.left
    let top = trigger.bottom + 4

    if (left + width > viewportWidth - 8) {
      left = viewportWidth - width - 8
    }
    if (left < 8) left = 8

    if (top + estimatedHeight > viewportHeight - 8) {
      top = Math.max(8, trigger.top - estimatedHeight - 4)
    }

    setPosition({ top, left, width })
  }, [isOpen, count, groupedByTeam.length])

  // Escape to close
  useEffect(() => {
    if (!isOpen) return
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setIsOpen(false)
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [isOpen])

  if (count === 0) return null

  return (
    <div ref={menuRef} className={cn('relative', className)}>
      {/* Trigger badge */}
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        className="flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium bg-amber-500/15 text-amber-600 dark:text-amber-400 hover:bg-amber-500/25 transition-colors"
        title={`${count} pending decision${count !== 1 ? 's' : ''}`}
      >
        <Scale className="h-3.5 w-3.5" />
        <span>{count} pending</span>
      </button>

      {/* Dropdown */}
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
          {/* Header */}
          <div className="flex items-center justify-between px-3 py-2 border-b border-border">
            <span className="text-xs font-semibold text-foreground">
              Pending Decisions ({count})
            </span>
          </div>

          {/* Grouped list */}
          <div className="max-h-64 overflow-y-auto py-1">
            {groupedByTeam.map((group) => (
              <div key={group.teamId}>
                {/* Team header */}
                <div className="px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {group.teamName}
                </div>

                {/* Decision rows */}
                {group.entries.map((entry) => {
                  const isProcessing = processingIds.has(entry.id)

                  return (
                    <div
                      key={entry.id}
                      className="group flex items-start gap-2 px-3 py-1.5 hover:bg-muted/50 cursor-pointer transition-colors"
                      onClick={() => {
                        onNavigateToDecision(group.teamId)
                        setIsOpen(false)
                      }}
                    >
                      <Scale className="h-3 w-3 text-amber-500 flex-shrink-0 mt-0.5" />
                      <div className="flex-1 min-w-0">
                        <div className="text-xs font-medium text-foreground line-clamp-2">
                          {entry.decision}
                        </div>
                        <div className="text-[10px] text-muted-foreground">
                          {entry.by} · {formatRelativePastTime(new Date(entry.at))}
                        </div>
                      </div>

                      {/* Accept button */}
                      <button
                        type="button"
                        className="opacity-0 group-hover:opacity-100 p-1 rounded bg-emerald-500/10 text-emerald-500 hover:bg-emerald-500/20 transition-all"
                        title="Accept"
                        onClick={(e) => {
                          e.stopPropagation()
                          if (!isProcessing) void acceptDecision(group.teamId, entry.id)
                        }}
                      >
                        {isProcessing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Check className="h-3.5 w-3.5" />}
                      </button>

                      {/* Reject button */}
                      <button
                        type="button"
                        className="opacity-0 group-hover:opacity-100 p-1 rounded bg-red-500/10 text-red-500 hover:bg-red-500/20 transition-all"
                        title="Reject"
                        onClick={(e) => {
                          e.stopPropagation()
                          if (!isProcessing) void rejectDecision(group.teamId, entry.id)
                        }}
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  )
                })}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
