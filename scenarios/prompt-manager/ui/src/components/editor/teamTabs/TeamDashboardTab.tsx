/**
 * TeamDashboardTab - Consolidated team dashboard with identity, schedule, and activity.
 *
 * Three-section layout:
 * 1. "What" - Team Identity (mission, member roster, runtime + coordination policy)
 * 2. "When" - Schedule (next up, upcoming heartbeats)
 * 3. "What happened" - Activity Feed (stats, filterable log entries)
 *
 * Reports health and last-active timestamps to the parent via callbacks.
 */

import { useEffect, useMemo, useState, useCallback } from 'react'
import { Clock, Cpu, Target, ExternalLink, ChevronDown } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import type {
  TeamDetails,
  TeamRole,
  UpdateTeamRequest,
  Coordination,
  CoordinationCapabilities,
  CoordinationPattern,
  Execution,
  MessagingMode,
  QueuePolicy,
  ReportingMode,
  RuntimeMode,
} from '@/types/team'
import type { Agent } from '@/types/agent'
import { cn } from '@/lib/utils'
import { selectors } from '@/constants/selectors'
import * as heartbeatService from '@/services/heartbeatService'
import type { HeartbeatConfig, TeamLogEntry } from '@/services/heartbeatService'
import { ExpandableDescription } from '@/components/shared/ExpandableDescription'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { runDetailPath } from '@/app/routes/route-paths'
import { formatRelativeTime, formatRelativePastTime, formatDate, formatDuration } from '@/lib/timeUtils'
import { formatScheduleSummary } from '@/lib/scheduleUtils'
import {
  buildBoundedParallelExecution,
  buildIndependentCoordination,
  buildLeaderLedCoordination,
  buildPeerCoordination,
  buildSerializedExecution,
} from '@/lib/schemas'

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
  const navigate = useNavigate()
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

  const resolvedLeadAgentId = useMemo(() => {
    return team.coordination.leadAgentId || team.members[0]?.agentId || ''
  }, [team.coordination.leadAgentId, team.members])

  const buildCoordinationPreset = useCallback(
    (pattern: CoordinationPattern, runtimeMode: RuntimeMode): Coordination => {
      if (pattern === 'leader-led') {
        return buildLeaderLedCoordination(resolvedLeadAgentId, runtimeMode)
      }
      if (pattern === 'peer') {
        return buildPeerCoordination()
      }
      return buildIndependentCoordination()
    },
    [resolvedLeadAgentId],
  )

  const handleRuntimeModeChange = useCallback(
    (mode: RuntimeMode) => {
      if (mode === team.runtime.mode) return

      if (mode === 'single-process') {
        if (!resolvedLeadAgentId) return
        void onUpdate({
          runtime: { mode },
          coordination: buildLeaderLedCoordination(resolvedLeadAgentId, 'single-process'),
          execution: buildSerializedExecution(),
        })
        return
      }

      const nextPattern = team.coordination.pattern
      const nextCoordination = buildCoordinationPreset(nextPattern, 'multi-process')
      const nextExecution =
        team.execution.queuePolicy === 'serialized'
          ? buildBoundedParallelExecution(Math.max(2, team.execution.maxConcurrentRuns))
          : team.execution

      void onUpdate({
        runtime: { mode },
        coordination: nextCoordination,
        execution: nextExecution,
      })
    },
    [buildCoordinationPreset, onUpdate, resolvedLeadAgentId, team.coordination.pattern, team.execution, team.runtime.mode],
  )

  const handleCoordinationPatternChange = useCallback(
    (pattern: CoordinationPattern) => {
      if (pattern === team.coordination.pattern) return
      if (team.runtime.mode === 'single-process' && pattern !== 'leader-led') return

      if (pattern === 'leader-led' && !resolvedLeadAgentId) return

      const nextCoordination =
        pattern === 'leader-led'
          ? buildLeaderLedCoordination(resolvedLeadAgentId, team.runtime.mode)
          : buildCoordinationPreset(pattern, team.runtime.mode)

      const nextExecution =
        team.runtime.mode === 'single-process'
          ? buildSerializedExecution()
          : team.execution

      void onUpdate({
        coordination: nextCoordination,
        execution: nextExecution,
      })
    },
    [buildCoordinationPreset, onUpdate, resolvedLeadAgentId, team.coordination.pattern, team.execution, team.runtime.mode],
  )

  const handleLeadAgentChange = useCallback(
    (leadAgentId: string) => {
      if (!leadAgentId) return
      void onUpdate({
        coordination: {
          ...team.coordination,
          leadAgentId,
        },
      })
    },
    [onUpdate, team.coordination],
  )

  const handleReportingModeChange = useCallback(
    (reportingMode: ReportingMode) => {
      void onUpdate({
        coordination: {
          ...team.coordination,
          reportingMode,
        },
      })
    },
    [onUpdate, team.coordination],
  )

  const handleMessagingModeChange = useCallback(
    (messagingMode: MessagingMode) => {
      const nextCapabilities: CoordinationCapabilities = {
        ...team.coordination.capabilities,
      }
      if (messagingMode !== 'async-inbox') {
        nextCapabilities.injectInbox = false
      }

      void onUpdate({
        coordination: {
          ...team.coordination,
          messagingMode,
          capabilities: nextCapabilities,
        },
      })
    },
    [onUpdate, team.coordination],
  )

  const handleCapabilityToggle = useCallback(
    (key: keyof CoordinationCapabilities, checked: boolean) => {
      const nextCapabilities: CoordinationCapabilities = {
        ...team.coordination.capabilities,
        [key]: checked,
      }
      if (key === 'injectInbox' && checked) {
        nextCapabilities.injectInbox = true
      }
      if (key === 'allowPeerTriggers' && checked && team.runtime.mode !== 'multi-process') {
        return
      }
      if (key === 'injectInbox' && checked && team.coordination.messagingMode !== 'async-inbox') {
        return
      }
      void onUpdate({
        coordination: {
          ...team.coordination,
          capabilities: nextCapabilities,
        },
      })
    },
    [onUpdate, team.coordination, team.runtime.mode],
  )

  const handleQueuePolicyChange = useCallback(
    (queuePolicy: QueuePolicy) => {
      if (team.runtime.mode === 'single-process') return

      const nextExecution: Execution =
        queuePolicy === 'serialized'
          ? buildSerializedExecution()
          : buildBoundedParallelExecution(Math.max(2, team.execution.maxConcurrentRuns))

      void onUpdate({ execution: nextExecution })
    },
    [onUpdate, team.execution.maxConcurrentRuns, team.runtime.mode],
  )

  const handleMaxConcurrentRunsChange = useCallback(
    (value: string) => {
      const parsed = Number.parseInt(value, 10)
      if (!Number.isFinite(parsed) || parsed < 2 || team.execution.queuePolicy !== 'bounded-parallel') {
        return
      }
      void onUpdate({
        execution: {
          queuePolicy: 'bounded-parallel',
          maxConcurrentRuns: parsed,
        },
      })
    },
    [onUpdate, team.execution.queuePolicy],
  )

  const runtimeSummary =
    team.runtime.mode === 'single-process'
      ? 'One leader session coordinates the team through Claude Code interop.'
      : 'Each team member runs in its own process with an explicit queue policy.'

  const coordinationSummary = (() => {
    if (team.coordination.pattern === 'leader-led') {
      return `Leader-led coordination with ${team.coordination.messagingMode} messaging and ${team.coordination.reportingMode} reporting.`
    }
    if (team.coordination.pattern === 'peer') {
      return `Peer coordination with ${team.coordination.messagingMode} messaging and ${team.coordination.reportingMode} reporting.`
    }
    return 'Independent specialists operate without a coordinator by default.'
  })()

  const executionSummary =
    team.execution.queuePolicy === 'serialized'
      ? 'Serialized execution keeps one active heartbeat run at a time.'
      : `Bounded parallel execution allows up to ${team.execution.maxConcurrentRuns} concurrent runs.`

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

      <section data-testid={selectors.teamEditor.runtimeMode}>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Runtime</h3>
        <div className="space-y-3">
          <div className="flex gap-2">
            {(['multi-process', 'single-process'] as const).map((mode) => {
              const selected = team.runtime.mode === mode
              const disabled = mode === 'single-process' && team.members.length === 0
              return (
                <button
                  key={mode}
                  type="button"
                  disabled={disabled}
                  onClick={() => handleRuntimeModeChange(mode)}
                  className={cn(
                    'flex-1 px-3 py-2 text-xs font-medium rounded-lg border transition-colors',
                    selected
                      ? 'bg-primary/15 border-primary/40 text-primary'
                      : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-foreground/20',
                    disabled && 'cursor-not-allowed opacity-50',
                  )}
                >
                  <Cpu className="h-3.5 w-3.5 mx-auto mb-1" />
                  {mode === 'multi-process' ? 'Multi-Process' : 'Single-Process'}
                </button>
              )
            })}
          </div>
          <p className="text-xs text-muted-foreground">{runtimeSummary}</p>
          {team.runtime.mode === 'single-process' && team.members.length === 0 && (
            <p className="text-xs text-amber-600">Add at least one member before enabling single-process runtime.</p>
          )}
        </div>
      </section>

      <section data-testid={selectors.teamEditor.coordinationPattern}>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Coordination</h3>
        <div className="space-y-3">
          <div className="flex gap-2">
            {(['independent', 'peer', 'leader-led'] as const).map((pattern) => {
              const selected = team.coordination.pattern === pattern
              const disabled =
                (pattern === 'leader-led' && team.members.length === 0) ||
                (team.runtime.mode === 'single-process' && pattern !== 'leader-led')
              return (
                <button
                  key={pattern}
                  type="button"
                  disabled={disabled}
                  onClick={() => handleCoordinationPatternChange(pattern)}
                  className={cn(
                    'flex-1 px-3 py-2 text-xs font-medium rounded-lg border transition-colors',
                    selected
                      ? 'bg-primary/15 border-primary/40 text-primary'
                      : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-foreground/20',
                    disabled && 'cursor-not-allowed opacity-50',
                  )}
                >
                  {pattern === 'leader-led' ? 'Leader-Led' : pattern === 'peer' ? 'Peer' : 'Independent'}
                </button>
              )
            })}
          </div>
          <p className="text-xs text-muted-foreground">{coordinationSummary}</p>

          <div className="grid gap-3 md:grid-cols-2">
            {team.coordination.pattern === 'leader-led' && (
              <label className="space-y-1 text-xs text-muted-foreground">
                <span className="font-medium text-foreground">Lead Agent</span>
                <select
                  aria-label="Lead Agent"
                  value={team.coordination.leadAgentId ?? ''}
                  onChange={(event) => handleLeadAgentChange(event.target.value)}
                  className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
                >
                  <option value="" disabled>
                    Select lead
                  </option>
                  {team.members.map((member) => (
                    <option key={member.agentId} value={member.agentId}>
                      {member.displayName}
                    </option>
                  ))}
                </select>
              </label>
            )}

            <label className="space-y-1 text-xs text-muted-foreground">
              <span className="font-medium text-foreground">Reporting Mode</span>
              <select
                aria-label="Reporting Mode"
                value={team.coordination.reportingMode}
                onChange={(event) => handleReportingModeChange(event.target.value as ReportingMode)}
                disabled={team.coordination.pattern === 'independent'}
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground disabled:cursor-not-allowed disabled:opacity-50"
              >
                {(['none', 'org-chart', 'leader'] as const).map((mode) => (
                  <option
                    key={mode}
                    value={mode}
                    disabled={
                      (team.coordination.pattern === 'independent' && mode !== 'none') ||
                      (team.coordination.pattern === 'peer' && mode === 'leader') ||
                      (team.coordination.pattern !== 'leader-led' && mode === 'leader')
                    }
                  >
                    {mode}
                  </option>
                ))}
              </select>
            </label>

            <label className="space-y-1 text-xs text-muted-foreground">
              <span className="font-medium text-foreground">Messaging Mode</span>
              <select
                aria-label="Messaging Mode"
                value={team.coordination.messagingMode}
                onChange={(event) => handleMessagingModeChange(event.target.value as MessagingMode)}
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
              >
                {(['disabled', 'async-inbox', 'in-session'] as const).map((mode) => (
                  <option
                    key={mode}
                    value={mode}
                    disabled={
                      (mode === 'in-session' && team.runtime.mode !== 'single-process') ||
                      (team.runtime.mode === 'single-process' && mode !== 'in-session')
                    }
                  >
                    {mode}
                  </option>
                ))}
              </select>
            </label>
          </div>

          <div className="grid gap-2 md:grid-cols-2">
            {(
              [
                ['showOrgContext', 'Show org context'],
                ['injectInbox', 'Inject inbox into prompt'],
                ['allowPeerTriggers', 'Allow peer triggers'],
                ['showTaskBoardGuidance', 'Show task board guidance'],
                ['showDecisionLogGuidance', 'Show decision log guidance'],
                ['showKnowledgeLogGuidance', 'Show knowledge log guidance'],
                ['requireHandoff', 'Require handoff'],
              ] as const
            ).map(([key, label]) => {
              const disabled =
                (key === 'injectInbox' && team.coordination.messagingMode !== 'async-inbox') ||
                (key === 'allowPeerTriggers' && team.runtime.mode !== 'multi-process')
              return (
                <label key={key} className={cn('flex items-center gap-2 text-xs', disabled && 'opacity-50')}>
                  <input
                    type="checkbox"
                    checked={team.coordination.capabilities[key]}
                    disabled={disabled}
                    onChange={(event) => handleCapabilityToggle(key, event.target.checked)}
                    className="h-4 w-4 rounded border-border"
                  />
                  <span>{label}</span>
                </label>
              )
            })}
          </div>
        </div>
      </section>

      <section data-testid={selectors.teamEditor.executionPolicy}>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Execution</h3>
        <div className="space-y-3">
          <div className="flex gap-2">
            {(['serialized', 'bounded-parallel'] as const).map((policy) => {
              const selected = team.execution.queuePolicy === policy
              const disabled = team.runtime.mode === 'single-process'
              return (
                <button
                  key={policy}
                  type="button"
                  disabled={disabled}
                  onClick={() => handleQueuePolicyChange(policy)}
                  className={cn(
                    'flex-1 px-3 py-2 text-xs font-medium rounded-lg border transition-colors',
                    selected
                      ? 'bg-primary/15 border-primary/40 text-primary'
                      : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-foreground/20',
                    disabled && 'cursor-not-allowed opacity-50',
                  )}
                >
                  {policy === 'serialized' ? 'Serialized' : 'Bounded Parallel'}
                </button>
              )
            })}
          </div>
          <p className="text-xs text-muted-foreground">{executionSummary}</p>

          <label className="space-y-1 text-xs text-muted-foreground">
            <span className="font-medium text-foreground">Max Concurrent Runs</span>
            <input
              aria-label="Max Concurrent Runs"
              type="number"
              min={team.execution.queuePolicy === 'serialized' ? 1 : 2}
              value={team.execution.maxConcurrentRuns}
              disabled={team.execution.queuePolicy !== 'bounded-parallel' || team.runtime.mode === 'single-process'}
              onChange={(event) => handleMaxConcurrentRunsChange(event.target.value)}
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground disabled:cursor-not-allowed disabled:opacity-50"
            />
          </label>
        </div>
      </section>

      {/* Decision Mode */}
      <section data-testid={selectors.teamEditor.decisionMode}>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Decision Approval</h3>
        <div className="space-y-3">
          <div className="flex gap-2">
            {(['yolo', 'approval'] as const).map((mode) => {
              const selected = team.decisionMode === mode
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
                        navigate(runDetailPath(stem))
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
