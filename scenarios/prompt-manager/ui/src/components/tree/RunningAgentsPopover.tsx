/**
 * RunningAgentsPopover - Shows a badge with count of running agents and a
 * dropdown listing them grouped by team. Follows the UnsavedChangesMenu
 * pattern (click-outside-close, escape-close, anchored dropdown).
 *
 * Accepts data as props from the parent (powered by useRunningAgentStatusSync)
 * to avoid duplicate polling. Falls back to self-polling if no data is provided.
 */

import { useState, useEffect, useRef, useLayoutEffect } from 'react'
import { Activity, Square, Loader2, Crosshair, ExternalLink } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { useRunningAgents, type TeamGroup } from '@/hooks/useRunningAgents'
import { runDetailPath, worldPath } from '@/app/routes/route-paths'

interface RunningAgentsPopoverProps {
  onNavigateToMember: (teamId: string, agentId: string) => void
  /** Pre-fetched data from useRunningAgentStatusSync (eliminates duplicate polling) */
  groupedByTeam?: TeamGroup[]
  count?: number
  stopAgent?: (teamId: string, agentId: string) => Promise<void>
  stoppingIds?: Set<string>
  className?: string
}

export function RunningAgentsPopover({
  onNavigateToMember,
  groupedByTeam: groupedByTeamProp,
  count: countProp,
  stopAgent: stopAgentProp,
  stoppingIds: stoppingIdsProp,
  className,
}: RunningAgentsPopoverProps) {
  const navigate = useNavigate()
  // Fallback to self-polling only if no data provided
  const hasExternalData = groupedByTeamProp !== undefined
  const fallback = useRunningAgents({ enabled: !hasExternalData })

  const groupedByTeam = hasExternalData ? groupedByTeamProp : fallback.groupedByTeam
  const count = hasExternalData ? (countProp ?? 0) : fallback.count
  const stopAgent = hasExternalData ? (stopAgentProp ?? fallback.stopAgent) : fallback.stopAgent
  const stoppingIds = hasExternalData ? (stoppingIdsProp ?? fallback.stoppingIds) : fallback.stoppingIds

  const [isOpen, setIsOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const [position, setPosition] = useState<{ top: number; left: number; width: number }>({ top: 0, left: 0, width: 288 })

  // Click-outside to close (delayed listener to prevent immediate close)
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
    const width = viewportWidth < 640 ? viewportWidth - 16 : Math.min(288, viewportWidth - 16)
    const estimatedHeight = Math.min(dropdownRef.current?.scrollHeight ?? 360, viewportHeight - 16)

    let left = trigger.left
    let top = trigger.bottom + 4

    if (left + width > viewportWidth - 8) {
      left = viewportWidth - width - 8
    }
    if (left < 8) {
      left = 8
    }

    if (top + estimatedHeight > viewportHeight - 8) {
      top = Math.max(8, trigger.top - estimatedHeight - 4)
    }

    setPosition({ top, left, width })
  }, [isOpen, count, groupedByTeam.length])

  // Escape to close
  useEffect(() => {
    if (!isOpen) return
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsOpen(false)
      }
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [isOpen])

  // Don't render when nothing is running
  if (count === 0) return null

  return (
    <div ref={menuRef} className={cn('relative', className)}>
      {/* Trigger badge */}
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        className="flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/25 transition-colors"
        title={`${count} running agent${count !== 1 ? 's' : ''}`}
      >
        <Activity className="h-3.5 w-3.5 animate-pulse" />
        <span>{count} running</span>
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
              Running Agents ({count})
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

                {/* Agent rows */}
                {group.agents.map((agent) => {
                  const key = `${agent.teamId}/${agent.agentId}`
                  const isStopping = stoppingIds.has(key)

                  return (
                    <div
                      key={key}
                      className="group flex items-center gap-2 px-3 py-1.5 hover:bg-muted/50 cursor-pointer transition-colors"
                      onClick={() => {
                        onNavigateToMember(agent.teamId, agent.agentId)
                        setIsOpen(false)
                      }}
                    >
                      <Activity className="h-3 w-3 text-emerald-500 flex-shrink-0 animate-pulse" />
                      <div className="flex-1 min-w-0">
                        <div className="text-xs font-medium text-foreground truncate">
                          {agent.agentName || agent.agentId}
                        </div>
                        <div className="text-[10px] text-muted-foreground">
                          {agent.duration}
                        </div>
                      </div>

                      {/* Open Run button — visible on hover */}
                      {agent.runId && (
                        <button
                          type="button"
                          className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-primary/10 hover:text-primary text-muted-foreground transition-all"
                          title="Open run detail view"
                          onClick={(e) => {
                            e.stopPropagation()
                            navigate(runDetailPath(agent.runId))
                            setIsOpen(false)
                          }}
                        >
                          <ExternalLink className="h-3.5 w-3.5" />
                        </button>
                      )}

                      {/* Focus button — visible on hover */}
                      <button
                        type="button"
                        className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-primary/10 hover:text-primary text-muted-foreground transition-all"
                        title="Focus in World View"
                        onClick={(e) => {
                          e.stopPropagation()
                          navigate(worldPath({ focus: agent.agentId }))
                          setIsOpen(false)
                        }}
                      >
                        <Crosshair className="h-3.5 w-3.5" />
                      </button>

                      {/* Stop button — visible on hover */}
                      <button
                        type="button"
                        className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-destructive/10 hover:text-destructive text-muted-foreground transition-all"
                        title="Stop agent"
                        onClick={(e) => {
                          e.stopPropagation()
                          if (!isStopping) {
                            void stopAgent(agent.teamId, agent.agentId)
                          }
                        }}
                      >
                        {isStopping ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        ) : (
                          <Square className="h-3.5 w-3.5" />
                        )}
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
