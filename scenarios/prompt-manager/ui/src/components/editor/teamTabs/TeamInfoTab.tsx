/**
 * TeamInfoTab - Team information display tab.
 *
 * Features:
 * - Team ID (read-only)
 * - Created/Updated timestamps
 * - Member count
 * - Role count
 * - Upcoming heartbeat schedule
 * - Editable roles list
 */

import { useEffect, useMemo, useState, useCallback } from 'react'
import { Clock, Hash, Users, Shield, Target, Cpu, FileText } from 'lucide-react'
import type { TeamDetails, TeamRole, UpdateTeamRequest } from '@/types/team'
import type { Agent } from '@/types/agent'
import { cn } from '@/lib/utils'
import { selectors } from '@/constants/selectors'
import * as heartbeatService from '@/services/heartbeatService'
import type { HeartbeatConfig } from '@/services/heartbeatService'
import { ExpandableDescription } from '@/components/shared/ExpandableDescription'
import { EventsModal } from '@/components/shared/EventsModal'
import { RolesTab } from './RolesTab'

interface TeamInfoTabProps {
  team: TeamDetails
  onSetRoles: (roles: TeamRole[]) => Promise<TeamRole[]>
  onUpdate: (updates: UpdateTeamRequest) => Promise<void>
  /** All agents for resolving appearance colors in role member chips */
  allAgents?: Agent[]
  /** Called when the user clicks an upcoming heartbeat entry */
  onNavigateToMemberHeartbeat?: (agentId: string) => void
  /** Called when the user clicks a member chip in the roles section */
  onNavigateToMember?: (agentId: string) => void
}

/**
 * Info display tab component for teams.
 */
export function TeamInfoTab({ team, onSetRoles, onUpdate, allAgents, onNavigateToMemberHeartbeat, onNavigateToMember }: TeamInfoTabProps) {
  const [heartbeatConfigs, setHeartbeatConfigs] = useState<HeartbeatConfig[]>([])
  const [isLoadingHeartbeats, setIsLoadingHeartbeats] = useState(false)
  const [heartbeatError, setHeartbeatError] = useState<string | null>(null)
  const [eventsRunId, setEventsRunId] = useState<string | null>(null)
  const [eventsAgentName, setEventsAgentName] = useState<string | undefined>()
  const [eventsLive, setEventsLive] = useState(false)
  const [eventsError, setEventsError] = useState<string | undefined>()

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Unknown'
    return new Date(dateString).toLocaleString()
  }

  const handleMissionChange = useCallback(
    (value: string) => {
      void onUpdate({ mission: value })
    },
    [onUpdate]
  )

  const formatRelativeTime = (date: Date) => {
    const diffMs = date.getTime() - Date.now()
    if (Number.isNaN(diffMs)) return 'Unknown'
    if (diffMs < 0) return 'Overdue'
    if (diffMs < 60000) return 'In less than a minute'
    if (diffMs < 3600000) return `In ${Math.round(diffMs / 60000)} minutes`
    if (diffMs < 86400000) return `In ${Math.round(diffMs / 3600000)} hours`
    return `In ${Math.round(diffMs / 86400000)} days`
  }

  const formatRelativePastTime = (date: Date) => {
    const diffMs = Date.now() - date.getTime()
    if (Number.isNaN(diffMs) || diffMs < 0) return 'Just now'
    if (diffMs < 60000) return 'Just now'
    if (diffMs < 3600000) {
      const mins = Math.round(diffMs / 60000)
      return `${mins} min${mins !== 1 ? 's' : ''} ago`
    }
    if (diffMs < 86400000) {
      const hrs = Math.round(diffMs / 3600000)
      return `${hrs} hour${hrs !== 1 ? 's' : ''} ago`
    }
    const days = Math.round(diffMs / 86400000)
    return `${days} day${days !== 1 ? 's' : ''} ago`
  }

  const upcomingHeartbeats = useMemo(() => {
    const membersById = new Map(team.members.map((member) => [member.agentId, member]))

    // Expand each config's nextExecutions into individual entries so that
    // a single daily cron can produce multiple upcoming rows.
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

    return entries.sort((a, b) => a.nextRun.getTime() - b.nextRun.getTime()).slice(0, 5)
  }, [heartbeatConfigs, team.members])

  const enabledHeartbeatCount = useMemo(() => {
    return heartbeatConfigs.filter((config) => config.enabled).length
  }, [heartbeatConfigs])

  const recentHeartbeats = useMemo(() => {
    const membersById = new Map(team.members.map((member) => [member.agentId, member]))

    return heartbeatConfigs
      .filter((config): config is HeartbeatConfig & { lastExecution: NonNullable<HeartbeatConfig['lastExecution']> } => !!config.lastExecution)
      .map((config) => {
        const memberName = membersById.get(config.agentId)?.displayName ?? config.agentId
        const startedAt = new Date(config.lastExecution.startedAt)
        return { config, memberName, startedAt, status: config.lastExecution.status }
      })
      .filter((entry) => !Number.isNaN(entry.startedAt.getTime()))
      .sort((a, b) => b.startedAt.getTime() - a.startedAt.getTime())
      .slice(0, 5)
  }, [heartbeatConfigs, team.members])

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
    // Poll every 10s so Recent Heartbeats / upcoming schedules stay current
    const interval = setInterval(() => void loadHeartbeats(), 10_000)
    return () => {
      isActive = false
      clearInterval(interval)
    }
  }, [team.id, team.enabled])

  return (
    <div className="space-y-6">
      {/* Basic Info */}
      <section>
        <h3 className="text-sm font-medium text-foreground mb-3">Basic Information</h3>
        <dl className="grid gap-3">
          <InfoRow
            icon={<Hash className="h-4 w-4" />}
            label="ID"
            value={team.id}
            mono
          />
          <InfoRow
            icon={<Users className="h-4 w-4" />}
            label="Members"
            value={`${team.memberCount} member${team.memberCount !== 1 ? 's' : ''}`}
          />
          <InfoRow
            icon={<Shield className="h-4 w-4" />}
            label="Roles"
            value={`${team.roles.length} role${team.roles.length !== 1 ? 's' : ''} defined`}
          />
        </dl>
      </section>

      {/* Execution Mode */}
      <section data-testid={selectors.teamEditor.spawnMode}>
        <h3 className="text-sm font-medium text-foreground mb-3">Execution Mode</h3>
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
                      : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-foreground/20'
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

      {/* Mission */}
      <section>
        <h3 className="text-sm font-medium text-foreground mb-3">Mission</h3>
        <div className="flex items-start gap-3 p-3 bg-muted rounded-lg">
          <Target className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
          <ExpandableDescription
            value={team.mission ?? ''}
            onChange={handleMissionChange}
            placeholder="Add a mission statement..."
            className="flex-1"
          />
        </div>
      </section>

      {/* Upcoming Heartbeats */}
      <section>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-foreground">Upcoming Heartbeats</h3>
          <span className="text-xs text-muted-foreground">Next 5</span>
        </div>
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
        ) : upcomingHeartbeats.length === 0 ? (
          <div className="flex items-start gap-3 p-3 bg-muted rounded-lg">
            <Clock className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
            <div>
              <p className="text-sm text-muted-foreground">No upcoming heartbeats scheduled.</p>
              <p className="text-xs text-muted-foreground/70 mt-0.5">
                Enable heartbeats on team members to populate this list.
              </p>
            </div>
          </div>
        ) : (
          <ul className="space-y-2">
            {upcomingHeartbeats.map((entry) => (
              <li
                key={`${entry.config.agentId}-${entry.nextRun.toISOString()}`}
              >
                <button
                  type="button"
                  onClick={() => onNavigateToMemberHeartbeat?.(entry.config.agentId)}
                  className={cn(
                    'w-full flex items-start justify-between gap-4 px-3 py-2 bg-muted rounded-lg text-left',
                    onNavigateToMemberHeartbeat && 'cursor-pointer hover:bg-muted/70 transition-colors'
                  )}
                >
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-foreground truncate">{entry.memberName}</p>
                    <p className="text-xs text-muted-foreground mt-0.5 truncate">
                      Schedule: {entry.config.schedule}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm text-foreground">{entry.nextRun.toLocaleString()}</p>
                    <p className="text-xs text-muted-foreground">{formatRelativeTime(entry.nextRun)}</p>
                  </div>
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Recent Heartbeats */}
      <section>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-foreground">Recent Heartbeats</h3>
          <span className="text-xs text-muted-foreground">Last 5</span>
        </div>
        {recentHeartbeats.length === 0 ? (
          <div className="flex items-start gap-3 p-3 bg-muted rounded-lg">
            <Clock className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
            <div>
              <p className="text-sm text-muted-foreground">No recent heartbeat executions.</p>
              <p className="text-xs text-muted-foreground/70 mt-0.5">
                Heartbeat history will appear here after members run their first heartbeat.
              </p>
            </div>
          </div>
        ) : (
          <ul className="space-y-2">
            {recentHeartbeats.map((entry) => (
              <li key={`${entry.config.agentId}-${entry.startedAt.toISOString()}`}>
                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    onClick={() => onNavigateToMemberHeartbeat?.(entry.config.agentId)}
                    className={cn(
                      'flex-1 flex items-start justify-between gap-4 px-3 py-2 bg-muted rounded-lg text-left',
                      onNavigateToMemberHeartbeat && 'cursor-pointer hover:bg-muted/70 transition-colors'
                    )}
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <span
                        className={cn(
                          'inline-block h-2 w-2 rounded-full flex-shrink-0',
                          entry.status === 'completed' && 'bg-emerald-500',
                          entry.status === 'failed' && 'bg-red-500',
                          entry.status === 'cancelled' && 'bg-slate-400',
                          entry.status === 'running' && 'bg-amber-500 animate-pulse'
                        )}
                      />
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-foreground truncate">{entry.memberName}</p>
                        <p className="text-xs text-muted-foreground mt-0.5 capitalize">{entry.status}</p>
                      </div>
                    </div>
                    <div className="text-right flex-shrink-0">
                      <p className="text-sm text-foreground">{entry.startedAt.toLocaleString()}</p>
                      <p className="text-xs text-muted-foreground">{formatRelativePastTime(entry.startedAt)}</p>
                    </div>
                  </button>
                  {(() => {
                    const rid = entry.config.lastExecution.runId
                    if (!rid) return null
                    return (
                      <button
                        type="button"
                        className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
                        title="View events"
                        onClick={() => {
                          setEventsRunId(rid)
                          setEventsAgentName(entry.memberName)
                          setEventsLive(entry.status === 'running')
                          setEventsError(entry.config.lastExecution.error || undefined)
                        }}
                      >
                        <FileText className="h-3.5 w-3.5" />
                      </button>
                    )
                  })()}
                </div>
                {/* Inline error message for failed/cancelled runs */}
                {(entry.status === 'failed' || entry.status === 'cancelled') && entry.config.lastExecution.error && (
                  <p className="text-xs text-red-400 mt-1 px-3 truncate" title={entry.config.lastExecution.error}>
                    {entry.config.lastExecution.error}
                  </p>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Timestamps */}
      <section>
        <h3 className="text-sm font-medium text-foreground mb-3">Timestamps</h3>
        <dl className="grid gap-3">
          <InfoRow
            icon={<Clock className="h-4 w-4" />}
            label="Created"
            value={formatDate(team.createdAt)}
          />
          <InfoRow
            icon={<Clock className="h-4 w-4" />}
            label="Updated"
            value={formatDate(team.updatedAt)}
          />
        </dl>
      </section>

      {/* Roles */}
      <section>
        <RolesTab team={team} onSetRoles={onSetRoles} allAgents={allAgents} onNavigateToMember={onNavigateToMember} />
      </section>

      {eventsRunId && (
        <EventsModal
          isOpen
          onClose={() => { setEventsRunId(null); setEventsError(undefined) }}
          runId={eventsRunId}
          agentName={eventsAgentName}
          live={eventsLive}
          errorMessage={eventsError}
        />
      )}
    </div>
  )
}

/**
 * Individual info row component.
 */
interface InfoRowProps {
  icon: React.ReactNode
  label: string
  value: React.ReactNode
  mono?: boolean
}

function InfoRow({ icon, label, value, mono }: InfoRowProps) {
  return (
    <div className="flex items-center gap-3">
      <div className="text-muted-foreground">{icon}</div>
      <dt className="text-sm text-muted-foreground min-w-[100px]">{label}</dt>
      <dd className={`text-sm flex-1 ${mono ? 'font-mono text-xs' : ''}`}>{value}</dd>
    </div>
  )
}
