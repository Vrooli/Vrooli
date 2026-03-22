/**
 * TeamDashboardTab - Consolidated team dashboard with identity, schedule, and activity.
 *
 * Three-section layout:
 * 1. "What" - Team Identity (mission, member roster, spawn mode)
 * 2. "When" - Schedule (next up, upcoming heartbeats)
 * 3. "What happened" - Activity Feed (stats, filterable log entries)
 *
 * Reports health and last-active timestamps to the parent via callbacks.
 */

import { useEffect, useMemo, useState, useCallback } from 'react'
import { Clock, Cpu, Target, ExternalLink, ChevronDown } from 'lucide-react'
import type { TeamDetails, TeamRole, UpdateTeamRequest } from '@/types/team'
import type { Agent } from '@/types/agent'
import { cn } from '@/lib/utils'
import { selectors } from '@/constants/selectors'
import * as heartbeatService from '@/services/heartbeatService'
import type { HeartbeatConfig, TeamLogEntry } from '@/services/heartbeatService'
import { ExpandableDescription } from '@/components/shared/ExpandableDescription'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { useSelectionStore } from '@/stores/selectionStore'
import { formatRelativeTime, formatRelativePastTime, formatDate, formatDuration } from '@/lib/timeUtils'
import { formatScheduleSummary } from '@/lib/scheduleUtils'

interface TeamDashboardTabProps {
  team: TeamDetails
  onSetRoles?: (roles: TeamRole[]) => Promise<TeamRole[]>
  onUpdate: (updates: UpdateTeamRequest) => Promise<void>
  /** All agents for resolving appearance colors in member badges */
  allAgents?: Agent[]
  /** Called when the user clicks an upcoming heartbeat entry */
  onNavigateToMemberHeartbeat?: (agentId: string) => void
  /** Called when the user clicks a member in the roster */
  onNavigateToMember?: (agentId: string) => void
  /** Report computed health to parent */
  onHealthChange?: (health: 'green' | 'yellow' | 'red' | 'gray') => void
  /** Report most-recent activity timestamp to parent */
  onLastActiveChange?: (lastActive: string | null) => void
}

const LOGS_PAGE_SIZE = 25

/**
 * Team dashboard tab - identity, schedule, and activity in one view.
 */
export function TeamDashboardTab({
  team,
  onUpdate,
  allAgents,
  onNavigateToMemberHeartbeat,
  onNavigateToMember,
  onHealthChange,
  onLastActiveChange,
}: TeamDashboardTabProps) {
  // --- Heartbeat polling state ---
  const [heartbeatConfigs, setHeartbeatConfigs] = useState<HeartbeatConfig[]>([])
  const [isLoadingHeartbeats, setIsLoadingHeartbeats] = useState(false)
  const [heartbeatError, setHeartbeatError] = useState<string | null>(null)

  // --- Activity feed state ---
  const [teamLogs, setTeamLogs] = useState<TeamLogEntry[]>([])
  const [logsOffset, setLogsOffset] = useState(0)
  const [hasMoreLogs, setHasMoreLogs] = useState(false)
  const [isLoadingLogs, setIsLoadingLogs] = useState(false)
  const [memberFilter, setMemberFilter] = useState('')

  // --- Agents lookup ---
  const agentsById = useMemo(() => {
    const map = new Map<string, Agent>()
    for (const agent of allAgents ?? []) {
      map.set(agent.id, agent)
    }
    return map
  }, [allAgents])

  // --- Handlers ---
  const handleMissionChange = useCallback(
    (value: string) => {
      void onUpdate({ mission: value })
    },
    [onUpdate],
  )

  // --- Upcoming heartbeats ---
  const upcomingHeartbeats = useMemo(() => {
    const membersById = new Map(team.members.map((m) => [m.agentId, m]))
    const entries: { config: HeartbeatConfig; memberName: string; nextRun: Date }[] = []
    for (const config of heartbeatConfigs) {
      if (!config.enabled) continue
      const memberName = membersById.get(config.agentId)?.displayName ?? config.agentId
      const times = config.nextExecutions ?? (config.nextExecution ? [config.nextExecution] : [])
      for (const iso of times) {
        const nextRun = new Date(iso)
        if (!Number.isNaN(nextRun.getTime())) {
          entries.push({ config, memberName, nextRun })
        }
      }
    }
    return entries.sort((a, b) => a.nextRun.getTime() - b.nextRun.getTime())
  }, [heartbeatConfigs, team.members])

  // Filter upcoming to next 24 hours
  const upcoming24h = useMemo(() => {
    const cutoff = Date.now() + 86_400_000
    return upcomingHeartbeats.filter((e) => e.nextRun.getTime() <= cutoff)
  }, [upcomingHeartbeats])

  const enabledHeartbeatCount = useMemo(() => {
    return heartbeatConfigs.filter((c) => c.enabled).length
  }, [heartbeatConfigs])

  // --- Summary stats ---
  const summaryStats = useMemo(() => {
    // Success rate from heartbeat configs with last executions
    const withExec = heartbeatConfigs.filter(
      (c): c is HeartbeatConfig & { lastExecution: NonNullable<HeartbeatConfig['lastExecution']> } =>
        !!c.lastExecution,
    )
    const total = withExec.length
    const completed = withExec.filter((c) => c.lastExecution.status === 'completed').length
    const successRate = total > 0 ? Math.round((completed / total) * 100) : -1

    // Run count in 24h from teamLogs
    const oneDayAgo = Date.now() - 86_400_000
    const recentLogs = teamLogs.filter((l) => new Date(l.timestamp).getTime() >= oneDayAgo)
    const runCount24h = recentLogs.length

    // Average duration from heartbeat configs that have both startedAt and endedAt
    const durations: number[] = []
    for (const c of withExec) {
      if (c.lastExecution.startedAt && c.lastExecution.endedAt) {
        const d = new Date(c.lastExecution.endedAt).getTime() - new Date(c.lastExecution.startedAt).getTime()
        if (d > 0) durations.push(d)
      }
    }
    const avgDuration = durations.length > 0 ? Math.round(durations.reduce((a, b) => a + b, 0) / durations.length) : -1

    return { successRate, runCount24h, avgDuration }
  }, [heartbeatConfigs, teamLogs])

  // --- Heartbeat polling (10s) ---
  useEffect(() => {
    let isActive = true
    let isFirstLoad = true
    const loadHeartbeats = async () => {
      if (isFirstLoad) {
        setIsLoadingHeartbeats(true)
        setHeartbeatError(null)
      }
      try {
        const configs = await heartbeatService.listHeartbeats(team.id)
        if (!isActive) return
        setHeartbeatConfigs(configs)
        if (isFirstLoad) setHeartbeatError(null)
      } catch (error) {
        if (!isActive) return
        if (isFirstLoad) {
          console.warn('Failed to load heartbeat schedule:', error)
          setHeartbeatConfigs([])
          setHeartbeatError('Unable to load heartbeat schedule.')
        }
      } finally {
        if (isActive && isFirstLoad) setIsLoadingHeartbeats(false)
        isFirstLoad = false
      }
    }

    void loadHeartbeats()
    const interval = setInterval(() => void loadHeartbeats(), 10_000)
    return () => {
      isActive = false
      clearInterval(interval)
    }
  }, [team.id, team.enabled])

  // --- Team logs loading ---
  useEffect(() => {
    let isActive = true
    const loadLogs = async () => {
      setIsLoadingLogs(true)
      try {
        const response = await heartbeatService.listTeamLogs(team.id, {
          limit: LOGS_PAGE_SIZE,
          offset: logsOffset,
          agentId: memberFilter || undefined,
        })
        if (!isActive) return
        if (logsOffset === 0) {
          setTeamLogs(response.logs)
        } else {
          setTeamLogs((prev) => [...prev, ...response.logs])
        }
        setHasMoreLogs(response.hasMore)
      } catch (error) {
        if (!isActive) return
        console.warn('Failed to load team logs:', error)
        if (logsOffset === 0) setTeamLogs([])
      } finally {
        if (isActive) setIsLoadingLogs(false)
      }
    }

    void loadLogs()
    return () => {
      isActive = false
    }
  }, [team.id, logsOffset, memberFilter])

  // Reset pagination when filter changes
  useEffect(() => {
    setLogsOffset(0)
    setTeamLogs([])
  }, [memberFilter])

  // --- Health computation ---
  useEffect(() => {
    if (!team.enabled) {
      onHealthChange?.('gray')
    } else {
      const withExec = heartbeatConfigs.filter(
        (c): c is HeartbeatConfig & { lastExecution: NonNullable<HeartbeatConfig['lastExecution']> } =>
          !!c.lastExecution,
      )
      if (withExec.length === 0) {
        onHealthChange?.('green')
      } else {
        const failed = withExec.filter((c) => c.lastExecution.status === 'failed')
        // Most recent execution across all configs
        const sorted = [...withExec].sort(
          (a, b) => new Date(b.lastExecution.startedAt).getTime() - new Date(a.lastExecution.startedAt).getTime(),
        )
        const mostRecent = sorted[0]
        if (mostRecent && mostRecent.lastExecution.status === 'failed') {
          onHealthChange?.('red')
        } else if (failed.length > 0) {
          onHealthChange?.('yellow')
        } else {
          onHealthChange?.('green')
        }
      }
    }

    // Last active
    const allExecs = heartbeatConfigs
      .filter((c) => c.lastExecution?.startedAt)
      .map((c) => c.lastExecution?.startedAt as string)
    if (allExecs.length > 0) {
      const sorted = allExecs.sort((a, b) => new Date(b).getTime() - new Date(a).getTime())
      onLastActiveChange?.(sorted[0] ?? null)
    } else {
      onLastActiveChange?.(null)
    }
  }, [heartbeatConfigs, team.enabled, onHealthChange, onLastActiveChange])

  // --- Load more handler ---
  const handleLoadMore = useCallback(() => {
    setLogsOffset((prev) => prev + LOGS_PAGE_SIZE)
  }, [])

  // Resolve member name from agentId
  const resolveMemberName = useCallback(
    (agentId: string): string => {
      const member = team.members.find((m) => m.agentId === agentId)
      return member?.displayName ?? agentId
    },
    [team.members],
  )

  // Resolve role names from role IDs
  const roleNameMap = useMemo(() => {
    const map = new Map<string, string>()
    for (const role of team.roles) {
      map.set(role.id, role.name)
    }
    return map
  }, [team.roles])

  return (
    <div className="space-y-6">
      {/* ================================================================ */}
      {/* Section 1: "What" - Team Identity                                */}
      {/* ================================================================ */}
      <section>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Mission</h3>
        <div className="flex items-start gap-3 p-3 bg-muted rounded-lg border border-border">
          <Target className="h-4 w-4 text-primary mt-0.5 flex-shrink-0" />
          <ExpandableDescription
            value={team.mission ?? ''}
            onChange={handleMissionChange}
            placeholder="Add a mission statement..."
            className="flex-1"
            maxLines={3}
          />
        </div>
      </section>

      {/* Member Roster */}
      <section>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">
          Members ({team.members.length})
        </h3>
        {team.members.length === 0 ? (
          <p className="text-sm text-muted-foreground">No members yet.</p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {team.members.map((member) => {
              const agent = agentsById.get(member.agentId)
              const isActive = member.status === 'active'
              return (
                <button
                  key={member.agentId}
                  type="button"
                  onClick={() => onNavigateToMember?.(member.agentId)}
                  className={cn(
                    'flex items-center gap-2 px-2.5 py-1.5 rounded-lg border transition-colors text-left',
                    'bg-muted border-border hover:bg-muted/70 hover:border-foreground/20',
                    onNavigateToMember && 'cursor-pointer',
                  )}
                >
                  <AgentColorBadge appearance={agent?.appearance} size="xs" />
                  <span className="text-sm font-medium text-foreground truncate max-w-[120px]">
                    {member.displayName}
                  </span>
                  {member.roles.length > 0 && (
                    <span className="flex gap-1">
                      {member.roles.slice(0, 2).map((roleId) => (
                        <span
                          key={roleId}
                          className="text-[10px] px-1.5 py-0.5 rounded-full bg-primary/10 text-primary font-medium truncate max-w-[60px]"
                        >
                          {roleNameMap.get(roleId) ?? roleId}
                        </span>
                      ))}
                      {member.roles.length > 2 && (
                        <span className="text-[10px] px-1 text-muted-foreground">
                          +{member.roles.length - 2}
                        </span>
                      )}
                    </span>
                  )}
                  <span
                    className={cn(
                      'inline-block h-2 w-2 rounded-full flex-shrink-0',
                      isActive ? 'bg-emerald-500' : 'bg-slate-400',
                    )}
                    title={member.status}
                  />
                </button>
              )
            })}
          </div>
        )}
      </section>

      {/* Spawn Mode */}
      <section data-testid={selectors.teamEditor.spawnMode}>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Execution Mode</h3>
        <div className="space-y-3">
          <div className="flex gap-2">
            {(['multi-process', 'single-process'] as const).map((mode) => {
              const selected = team.spawnMode === mode
              return (
                <button
                  key={mode}
                  type="button"
                  onClick={() => void onUpdate({ spawnMode: mode })}
                  className={cn(
                    'flex-1 px-3 py-2 text-xs font-medium rounded-lg border transition-colors',
                    selected
                      ? 'bg-primary/15 border-primary/40 text-primary'
                      : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-foreground/20',
                  )}
                >
                  <Cpu className="h-3.5 w-3.5 mx-auto mb-1" />
                  {mode === 'multi-process' ? 'Multi-Process' : 'Single-Process'}
                </button>
              )
            })}
          </div>
          <p className="text-xs text-muted-foreground">
            {team.spawnMode === 'multi-process'
              ? 'Each member runs as an independent agent process with its own heartbeat.'
              : 'One team lead agent coordinates all members via Claude Code Teams.'}
          </p>
        </div>
      </section>

      {/* Decision Mode */}
      <section data-testid={selectors.teamEditor.decisionMode}>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Decision Approval</h3>
        <div className="space-y-3">
          <div className="flex gap-2">
            {(['yolo', 'approval'] as const).map((mode) => {
              const selected = team.decisionMode === mode || (!team.decisionMode && mode === 'yolo')
              return (
                <button
                  key={mode}
                  type="button"
                  onClick={() => void onUpdate({ decisionMode: mode })}
                  className={cn(
                    'flex-1 px-3 py-2 text-xs font-medium rounded-lg border transition-colors',
                    selected
                      ? 'bg-primary/15 border-primary/40 text-primary'
                      : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-foreground/20',
                  )}
                >
                  {mode === 'yolo' ? 'Auto-approve' : 'Require Approval'}
                </button>
              )
            })}
          </div>
          <p className="text-xs text-muted-foreground">
            {(team.decisionMode === 'approval')
              ? 'Decisions require human approval before agents can act on them.'
              : 'Agents can freely approve and act on their own decisions.'}
          </p>
        </div>
      </section>

      {/* ================================================================ */}
      {/* Section 2: "When" - Schedule                                     */}
      {/* ================================================================ */}
      <section>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Schedule</h3>

        {/* Team-off warning */}
        {!team.enabled ? (
          <div className="flex items-start gap-3 p-3 bg-amber-500/10 border border-amber-500/30 rounded-lg">
            <Clock className="h-4 w-4 text-amber-500 mt-0.5 flex-shrink-0" />
            <div>
              <p className="text-sm text-amber-500 font-medium">Team is turned off</p>
              <p className="text-xs text-muted-foreground mt-0.5">
                Heartbeats are paused
                {enabledHeartbeatCount > 0 ? ` (${enabledHeartbeatCount} configured)` : ''}. Turn the team on to resume.
              </p>
            </div>
          </div>
        ) : isLoadingHeartbeats ? (
          <p className="text-sm text-muted-foreground">Loading heartbeat schedule...</p>
        ) : heartbeatError ? (
          <p className="text-sm text-destructive">{heartbeatError}</p>
        ) : upcoming24h.length === 0 ? (
          <div className="flex items-start gap-3 p-3 bg-muted rounded-lg">
            <Clock className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
            <div>
              <p className="text-sm text-muted-foreground">No upcoming heartbeats in the next 24 hours.</p>
              <p className="text-xs text-muted-foreground/70 mt-0.5">
                Enable heartbeats on team members to populate this list.
              </p>
            </div>
          </div>
        ) : (
          <div className="space-y-2">
            {/* "Next up" callout - first entry */}
            {upcoming24h[0] && (
              <button
                type="button"
                onClick={() => onNavigateToMemberHeartbeat?.(upcoming24h[0]?.config.agentId ?? '')}
                className={cn(
                  'w-full flex items-center gap-3 p-3 bg-primary/5 border border-primary/20 rounded-lg text-left',
                  onNavigateToMemberHeartbeat && 'cursor-pointer hover:bg-primary/10 transition-colors',
                )}
              >
                <AgentColorBadge
                  appearance={agentsById.get(upcoming24h[0].config.agentId)?.appearance}
                  size="sm"
                />
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-foreground truncate">
                    {upcoming24h[0].memberName}
                  </p>
                  <p className="text-xs text-muted-foreground truncate">
                    {formatScheduleSummary(upcoming24h[0].config.schedule)}
                  </p>
                </div>
                <div className="text-right flex-shrink-0">
                  <p className="text-sm font-medium text-primary">
                    {formatRelativeTime(upcoming24h[0].nextRun)}
                  </p>
                </div>
              </button>
            )}

            {/* Remaining schedule entries */}
            {upcoming24h.length > 1 && (
              <ul className="space-y-1">
                {upcoming24h.slice(1).map((entry) => (
                  <li key={`${entry.config.agentId}-${entry.nextRun.toISOString()}`}>
                    <button
                      type="button"
                      onClick={() => onNavigateToMemberHeartbeat?.(entry.config.agentId)}
                      className={cn(
                        'w-full flex items-center gap-2.5 px-3 py-2 bg-muted rounded-lg text-left',
                        onNavigateToMemberHeartbeat && 'cursor-pointer hover:bg-muted/70 transition-colors',
                      )}
                    >
                      <AgentColorBadge
                        appearance={agentsById.get(entry.config.agentId)?.appearance}
                        size="xs"
                      />
                      <span className="text-sm text-foreground truncate min-w-0 flex-1">
                        {entry.memberName}
                      </span>
                      <span className="text-xs text-muted-foreground truncate max-w-[140px]">
                        {formatScheduleSummary(entry.config.schedule)}
                      </span>
                      <span className="text-xs text-muted-foreground flex-shrink-0">
                        {formatRelativeTime(entry.nextRun)}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </section>

      {/* ================================================================ */}
      {/* Section 3: "What happened" - Activity Feed                       */}
      {/* ================================================================ */}
      <section>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Activity</h3>

        {/* Summary stats bar */}
        <div className="flex gap-4 mb-3 text-sm">
          {summaryStats.successRate >= 0 && (
            <span
              className={cn(
                'font-medium',
                summaryStats.successRate > 80 && 'text-emerald-500',
                summaryStats.successRate >= 50 && summaryStats.successRate <= 80 && 'text-amber-500',
                summaryStats.successRate < 50 && 'text-red-500',
              )}
            >
              {summaryStats.successRate}% success
            </span>
          )}
          <span className="text-muted-foreground">
            {summaryStats.runCount24h} run{summaryStats.runCount24h !== 1 ? 's' : ''} in 24h
          </span>
          {summaryStats.avgDuration >= 0 && (
            <span className="text-muted-foreground">
              avg {formatDuration(summaryStats.avgDuration)}
            </span>
          )}
        </div>

        {/* Member filter */}
        {team.members.length > 1 && (
          <div className="relative mb-3">
            <select
              value={memberFilter}
              onChange={(e) => setMemberFilter(e.target.value)}
              className="w-full appearance-none px-3 py-1.5 pr-8 text-sm bg-muted border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            >
              <option value="">All members</option>
              {team.members.map((member) => (
                <option key={member.agentId} value={member.agentId}>
                  {member.displayName}
                </option>
              ))}
            </select>
            <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
          </div>
        )}

        {/* Activity entries */}
        {teamLogs.length === 0 && !isLoadingLogs ? (
          <div className="flex items-start gap-3 p-3 bg-muted rounded-lg">
            <Clock className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
            <p className="text-sm text-muted-foreground">No recent activity.</p>
          </div>
        ) : (
          <ul className="space-y-1.5">
            {teamLogs.map((entry, idx) => {
              const entryDate = new Date(entry.timestamp)
              const statusNormalized = (entry.status ?? '').toLowerCase()
              return (
                <li
                  key={`${entry.agentId}-${entry.filename}-${idx}`}
                  className="flex items-center gap-2.5 px-3 py-2 bg-muted rounded-lg"
                >
                  <span
                    className={cn(
                      'inline-block h-2 w-2 rounded-full flex-shrink-0',
                      statusNormalized === 'completed' && 'bg-emerald-500',
                      statusNormalized === 'failed' && 'bg-red-500',
                      statusNormalized === 'cancelled' && 'bg-slate-400',
                      statusNormalized === 'running' && 'bg-amber-500 animate-pulse',
                      !['completed', 'failed', 'cancelled', 'running'].includes(statusNormalized) && 'bg-slate-400',
                    )}
                  />
                  <span className="text-sm text-foreground truncate min-w-0 flex-1">
                    {resolveMemberName(entry.agentId)}
                  </span>
                  <span className="text-xs text-muted-foreground flex-shrink-0">
                    {!Number.isNaN(entryDate.getTime()) ? formatRelativePastTime(entryDate) : ''}
                  </span>
                  {entry.status && (
                    <span className="text-xs text-muted-foreground capitalize flex-shrink-0">
                      {entry.status}
                    </span>
                  )}
                  <button
                    type="button"
                    className="p-1 rounded hover:bg-background text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
                    title="Open run view"
                    onClick={() => {
                      const stem = entry.filename.replace(/\.[^.]+$/, '')
                      if (stem) {
                        useSelectionStore.getState().setSelectedRunId(stem)
                      }
                    }}
                  >
                    <ExternalLink className="h-3 w-3" />
                  </button>
                </li>
              )
            })}
          </ul>
        )}

        {isLoadingLogs && (
          <p className="text-sm text-muted-foreground mt-2">Loading...</p>
        )}

        {hasMoreLogs && !isLoadingLogs && (
          <button
            type="button"
            onClick={handleLoadMore}
            className="mt-2 w-full py-1.5 text-xs font-medium text-primary hover:text-primary/80 transition-colors"
          >
            Load more
          </button>
        )}
      </section>

      {/* ================================================================ */}
      {/* Compact Footer                                                   */}
      {/* ================================================================ */}
      <footer className="text-xs text-muted-foreground pt-2 border-t border-border">
        <span className="font-mono">{team.id}</span>
        {' \u00B7 '}
        Created {formatDate(team.createdAt)}
        {' \u00B7 '}
        Updated {formatDate(team.updatedAt)}
      </footer>
    </div>
  )
}
