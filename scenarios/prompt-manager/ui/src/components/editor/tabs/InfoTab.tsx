/**
 * InfoTab - Agent information display tab.
 *
 * Features:
 * - Read-only metadata display
 * - ID, status, timestamps
 * - Runtime configuration
 * - Team memberships
 * - Tags display
 */

import { useState, useEffect } from 'react'
import { Clock, Hash, Activity, Folder, Tag, Server, Users, ShieldCheck, AlertTriangle } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'
import type { Agent } from '@/types/agent'
import type { AgentTeamMembership } from '@/lib/schemas'
import { getAgentTeams } from '@/services/agentService'
import * as heartbeatService from '@/services/heartbeatService'
import type { HeartbeatConfig, RunDetails } from '@/services/heartbeatService'
import { runDetailPath, teamDetailPath } from '@/app/routes/route-paths'

// Extended agent type with optional v1 fields for display purposes
interface AgentWithLegacyFields extends Agent {
  schemaVersion?: number
  revision?: number
  runtime?: {
    workspaceRef?: string
  }
}

interface InfoTabProps {
  agent: AgentWithLegacyFields
}

/**
 * Info display tab component.
 */
export function InfoTab({ agent }: InfoTabProps) {
  const navigate = useNavigate()
  const [memberships, setMemberships] = useState<AgentTeamMembership[]>([])
  const [heartbeatByTeam, setHeartbeatByTeam] = useState<Map<string, HeartbeatConfig | null>>(new Map())
  const [membershipReadState, setMembershipReadState] = useState<'loading' | 'ready' | 'unavailable'>('loading')
  const [heartbeatUnavailableCount, setHeartbeatUnavailableCount] = useState(0)
  const [agentRuns, setAgentRuns] = useState<RunDetails[]>([])
  const [runReadState, setRunReadState] = useState<'loading' | 'ready' | 'unavailable'>('loading')

  useEffect(() => {
    let cancelled = false
    setMembershipReadState('loading')
    setHeartbeatUnavailableCount(0)
    void getAgentTeams(agent.id).then(async (result) => {
      if (cancelled) return
      setMemberships(result)
      setMembershipReadState('ready')
      const entries = await Promise.all(
        result.map(async (membership) => {
          try {
            const config = await heartbeatService.getHeartbeat(membership.teamId, agent.id)
            return { teamId: membership.teamId, config, unavailable: false }
          } catch {
            return { teamId: membership.teamId, config: null, unavailable: true }
          }
        })
      )
      if (cancelled) return
      setHeartbeatByTeam(new Map(entries.map(({ teamId, config }) => [teamId, config])))
      setHeartbeatUnavailableCount(entries.filter((entry) => entry.unavailable).length)
    }).catch(() => {
      if (cancelled) return
      setMembershipReadState('unavailable')
      setMemberships([])
      setHeartbeatByTeam(new Map())
    })
    return () => { cancelled = true }
  }, [agent.id])

  useEffect(() => {
    let cancelled = false
    setRunReadState('loading')
    setAgentRuns([])
    void heartbeatService.listRuns({ agentId: agent.id, limit: 100 })
      .then((result) => {
        if (cancelled) return
        setAgentRuns(result.runs)
        setRunReadState('ready')
      })
      .catch(() => {
        if (cancelled) return
        setRunReadState('unavailable')
      })
    return () => { cancelled = true }
  }, [agent.id])

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Unknown'
    return new Date(dateString).toLocaleString()
  }

  const activeMemberships = memberships.filter((membership) => membership.status === 'active').length
  const failedMemberships = memberships.filter((membership) => heartbeatByTeam.get(membership.teamId)?.lastExecution?.status === 'failed').length
  const freshnessWindowMs = 24 * 60 * 60 * 1000
  const staleMemberships = memberships.filter((membership) => {
    const endedAt = heartbeatByTeam.get(membership.teamId)?.lastExecution?.endedAt
    return Boolean(endedAt && Date.now() - new Date(endedAt).getTime() > freshnessWindowMs)
  }).length
  const completedRuns = agentRuns.filter((run) => run.status === 'completed').length
  const failedRuns = agentRuns.filter((run) => run.status === 'failed').length
  const runSuccessRate = agentRuns.length > 0 ? Math.round((completedRuns / agentRuns.length) * 100) : null
  const staleRuns = agentRuns.filter((run) => {
    const endedAt = run.endedAt
    return Boolean(endedAt && Date.now() - new Date(endedAt).getTime() > freshnessWindowMs)
  }).length
  const runHistoryState = runReadState === 'loading'
    ? { label: 'Loading', detail: 'Reading owner run history for this agent.' }
    : runReadState === 'unavailable'
      ? { label: 'Unavailable', detail: 'The owner run source did not return an agent-scoped response.' }
      : agentRuns.length === 0
        ? { label: 'Empty history', detail: 'The owner returned no runs for this agent.' }
      : { label: 'Observed', detail: `${agentRuns.length} owner runs observed; ${runSuccessRate}% completed successfully${failedRuns > 0 ? `, ${failedRuns} failed` : ''}.` }
  const currentHealthState = membershipReadState === 'loading'
    ? { label: 'Loading', detail: 'Reading current membership signals from the ownership source.' }
    : membershipReadState === 'unavailable'
      ? { label: 'Unavailable', detail: 'The membership owner did not return a usable current-health response.' }
      : memberships.length === 0
        ? { label: 'Empty', detail: 'No team memberships were returned, so current health cannot be observed.' }
        : heartbeatUnavailableCount === memberships.length
          ? { label: 'Unavailable', detail: 'Every current membership signal is unavailable.' }
          : heartbeatUnavailableCount > 0
            ? { label: 'Partial', detail: `${heartbeatUnavailableCount} of ${memberships.length} membership signals are unavailable.` }
            : failedMemberships > 0
              ? { label: 'Attention', detail: `${failedMemberships} latest membership signal${failedMemberships === 1 ? '' : 's'} failed.` }
              : staleMemberships > 0
                ? { label: 'Stale', detail: `${staleMemberships} membership signal${staleMemberships === 1 ? '' : 's'} is older than the 24-hour freshness window.` }
                : { label: 'Observed', detail: 'All current membership signals are available and fresh; this is not an agent-wide health verdict.' }
  const activityState = runReadState === 'loading'
    ? { label: 'Loading', detail: 'Reading owner-observed activity for this agent.' }
    : runReadState === 'unavailable'
      ? { label: 'Unavailable', detail: 'The owner run source did not return agent-scoped activity.' }
      : agentRuns.length === 0
        ? { label: 'Empty history', detail: 'The owner returned no runs for this agent.' }
        : staleRuns === agentRuns.length
          ? { label: 'Stale', detail: 'Every observed run is older than the 24-hour freshness window.' }
          : { label: 'Observed', detail: `${agentRuns.length} owner run${agentRuns.length === 1 ? '' : 's'} observed; freshness is explicit.` }
const profileState = membershipReadState === 'loading'
    ? { label: 'Reading evidence', tone: 'text-muted-foreground', detail: 'Loading membership and latest-run evidence for this agent.' }
    : membershipReadState === 'unavailable'
      ? { label: 'Coverage unavailable', tone: 'text-amber-500', detail: 'The membership owner did not return a usable agent-scoped response.' }
      : memberships.length === 0
        ? { label: 'No memberships', tone: 'text-muted-foreground', detail: 'The membership owner returned no team membership evidence for this agent.' }
        : heartbeatUnavailableCount === memberships.length
          ? { label: 'Signals unavailable', tone: 'text-amber-500', detail: 'Memberships are known, but latest-run heartbeat evidence is unavailable for every membership.' }
          : heartbeatUnavailableCount > 0
            ? { label: 'Partial coverage', tone: 'text-amber-500', detail: `${heartbeatUnavailableCount} membership signal${heartbeatUnavailableCount === 1 ? '' : 's'} could not be read; displayed values are partial.` }
      : failedMemberships > 0
      ? { label: 'Needs attention', tone: 'text-amber-500', detail: `${failedMemberships} membership${failedMemberships === 1 ? '' : 's'} has a failed latest run.` }
      : staleMemberships > 0
        ? { label: 'Stale evidence', tone: 'text-amber-500', detail: `${staleMemberships} membership${staleMemberships === 1 ? '' : 's'} has evidence older than the freshness window.` }
      : { label: 'Observed', tone: 'text-emerald-500', detail: `${activeMemberships} active membership${activeMemberships === 1 ? '' : 's'} ${activeMemberships === 1 ? 'is' : 'are'} registered; latest run evidence is shown below.` }

  return (
    <div className="min-w-0 max-w-full space-y-6 overflow-x-hidden">
      <section className="rounded-2xl border border-primary/30 bg-gradient-to-br from-primary/15 via-primary/5 to-muted/30 p-4 shadow-sm sm:p-5" aria-labelledby="agent-profile-heading">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="flex min-w-0 items-start gap-3">
            <div className="grid h-10 w-10 shrink-0 place-items-center rounded-2xl border border-primary/30 bg-primary/10 text-primary sm:h-12 sm:w-12"><ShieldCheck className="h-5 w-5 sm:h-6 sm:w-6" aria-hidden="true" /></div>
            <div className="min-w-0">
              <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-primary">Operational profile</p>
              <h2 id="agent-profile-heading" className="mt-1 break-words text-lg font-semibold leading-tight text-foreground line-clamp-2 sm:text-xl">{agent.id}</h2>
              <p className="mt-1 text-xs text-muted-foreground sm:text-sm">Identity, current signals, memberships, and configuration in one place.</p>
            </div>
          </div>
          <span className={cn('inline-flex items-center gap-1 rounded-full border border-current/30 px-2 py-1 text-xs font-medium', profileState.tone)}>{failedMemberships > 0 ? <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" /> : <Activity className="h-3.5 w-3.5" aria-hidden="true" />}{profileState.label}</span>
        </div>
        <p className="mt-4 max-w-2xl text-sm leading-relaxed text-muted-foreground">{profileState.detail} This surface does not infer health from metadata alone; unavailable and stale reads remain labeled as unknown.</p>
        <div className="mt-4 grid grid-cols-2 gap-2 sm:grid-cols-4" aria-label="Agent profile summary">
          <ProfileMetric label="Status" value={agent.status} />
          <ProfileMetric label="Teams" value={membershipReadState === 'loading' ? '…' : String(memberships.length)} />
          <ProfileMetric label="Active" value={String(activeMemberships)} />
          <ProfileMetric label="Attention" value={String(failedMemberships)} />
        </div>
      </section>
      <section className="rounded-2xl border border-border bg-card p-5 shadow-sm" aria-labelledby="agent-evidence-heading">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-primary">Evidence coverage</p>
            <h3 id="agent-evidence-heading" className="mt-1 text-lg font-semibold text-foreground">What this profile can prove</h3>
          </div>
          <span className={cn('rounded-full border px-2 py-1 text-xs font-medium', runReadState === 'ready' ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-600' : 'border-amber-500/30 bg-amber-500/10 text-amber-600')}>{runReadState === 'ready' ? 'Agent-scoped run evidence observed' : 'Agent-scoped run evidence ' + runHistoryState.label.toLowerCase()}</span>
        </div>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-muted-foreground">The page now reads owner-scoped run history directly and keeps current membership health separate. Freshness, ownership, and limitations remain visible; no agent-wide health classification is inferred from historical runs.</p>
        <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <EvidenceCell label="Current health" state={currentHealthState.label} detail={currentHealthState.detail} />
          <EvidenceCell label="Historical performance" state={runHistoryState.label} detail={runHistoryState.detail} />
          <EvidenceCell label="Attention" state={failedMemberships ? 'Observed' : memberships.length ? 'Partial' : 'Unavailable'} detail={failedMemberships ? `${failedMemberships} latest membership signal${failedMemberships === 1 ? '' : 's'} failed.` : 'No agent-scoped attention feed is available.'} />
          <EvidenceCell label="Activity" state={activityState.label} detail={activityState.detail} />
          <EvidenceCell label="Memberships" state={membershipReadState === 'loading' ? 'Loading' : membershipReadState === 'unavailable' ? 'Unavailable' : memberships.length ? 'Observed' : 'Empty'} detail={membershipReadState === 'loading' ? 'Reading memberships from the ownership source.' : membershipReadState === 'unavailable' ? 'The membership owner did not return a usable response.' : memberships.length ? `${memberships.length} team membership${memberships.length === 1 ? '' : 's'} loaded from the membership owner.` : 'No team memberships returned.'} />
          <EvidenceCell label="Configuration" state="Observed" detail="Agent metadata, connectors, capabilities, and heartbeat configuration are shown below." />
        </div>
      </section>
      {runReadState === 'ready' && agentRuns.length > 0 && (
        <section className="rounded-2xl border border-border bg-card p-5 shadow-sm" aria-labelledby="agent-activity-heading">
          <div className="flex flex-wrap items-end justify-between gap-3">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-primary">Recent activity</p>
              <h3 id="agent-activity-heading" className="mt-1 text-lg font-semibold text-foreground">Owner-observed runs</h3>
            </div>
            <span className="text-xs text-muted-foreground">{agentRuns.length} observed · {completedRuns} completed</span>
          </div>
          <ul className="mt-4 space-y-2">
            {agentRuns.slice(0, 5).map((run) => (
              <li key={run.id} className="flex min-w-0 flex-wrap items-center justify-between gap-3 rounded-xl border border-border/70 bg-muted/20 px-3 py-2">
                <div className="min-w-0">
                  <button type="button" onClick={() => navigate(runDetailPath(run.id))} className="max-w-full truncate text-left text-sm font-medium text-primary hover:underline">{run.id}</button>
                  <p className="mt-0.5 text-xs text-muted-foreground">{run.startedAt ? new Date(run.startedAt).toLocaleString() : 'Time unavailable'}{run.teamId ? ` · ${run.teamId}` : ''}</p>
                </div>
                <StatusBadge status={run.status} />
              </li>
            ))}
          </ul>
        </section>
      )}
      {/* Basic Info */}
      <section>
        <h3 className="text-sm font-medium text-foreground mb-3">Basic Information</h3>
        <dl className="grid gap-3">
          <InfoRow
            icon={<Hash className="h-4 w-4" />}
            label="ID"
            value={agent.id}
            mono
          />
          <InfoRow
            icon={<Activity className="h-4 w-4" />}
            label="Status"
            value={
              <StatusBadge status={agent.status} />
            }
          />
          <InfoRow
            icon={<Folder className="h-4 w-4" />}
            label="Schema Version"
            value={`v${agent.schemaVersion ?? 1}`}
          />
        </dl>
      </section>

      {/* Timestamps */}
      <section>
        <h3 className="text-sm font-medium text-foreground mb-3">Timestamps</h3>
        <dl className="grid gap-3">
          <InfoRow
            icon={<Clock className="h-4 w-4" />}
            label="Created"
            value={formatDate(agent.createdAt)}
          />
          <InfoRow
            icon={<Clock className="h-4 w-4" />}
            label="Updated"
            value={formatDate(agent.updatedAt)}
          />
          <InfoRow
            icon={<Hash className="h-4 w-4" />}
            label="Revision"
            value={`#${agent.revision ?? 1}`}
          />
        </dl>
      </section>

      {/* Runtime */}
      {agent.runtime && (
        <section>
          <h3 className="text-sm font-medium text-foreground mb-3">Runtime</h3>
          <dl className="grid gap-3">
            <InfoRow
              icon={<Server className="h-4 w-4" />}
              label="Workspace Reference"
              value={agent.runtime.workspaceRef ?? 'Not configured'}
              mono
            />
          </dl>
        </section>
      )}

      {/* Tags */}
      {agent.tags.length > 0 && (
        <section>
          <h3 className="text-sm font-medium text-foreground mb-3">Tags</h3>
          <div className="flex flex-wrap gap-2">
            {agent.tags.map((tag) => (
              <span
                key={tag}
                className="flex items-center gap-1 px-2 py-1 text-sm bg-primary/20 text-primary rounded-full"
              >
                <Tag className="h-3 w-3" />
                {tag}
              </span>
            ))}
          </div>
        </section>
      )}

      {/* Capabilities (if defined) */}
      {agent.capabilities && (
        <section>
          <h3 className="text-sm font-medium text-foreground mb-3">Capabilities</h3>
          <div className="space-y-3">
            {agent.capabilities.provides.length > 0 && (
              <div>
                <span className="text-xs text-muted-foreground">Provides:</span>
                <ul className="mt-1 space-y-1">
                  {agent.capabilities.provides.map((cap, idx) => (
                    <li key={idx} className="text-sm font-mono">
                      {cap.capabilityId} ({cap.verbs.join(', ') || 'all'})
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {agent.capabilities.requires.length > 0 && (
              <div>
                <span className="text-xs text-muted-foreground">Requires:</span>
                <ul className="mt-1 space-y-1">
                  {agent.capabilities.requires.map((cap, idx) => (
                    <li key={idx} className="text-sm font-mono">
                      {cap.capabilityId} ({cap.verbs.join(', ') || 'all'})
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        </section>
      )}

      {/* Connectors (if defined) */}
      {agent.connectors.length > 0 && (
        <section>
          <h3 className="text-sm font-medium text-foreground mb-3">Connectors</h3>
          <ul className="space-y-2">
            {agent.connectors.map((connector, idx) => (
              <li
                key={idx}
                className="flex items-center justify-between p-2 bg-muted rounded-lg"
              >
                <div>
                  <span className="text-sm font-medium">{connector.id}</span>
                  <span className="text-xs text-muted-foreground ml-2">
                    ({connector.type})
                  </span>
                </div>
                <span
                  className={cn(
                    'px-2 py-0.5 text-xs rounded-full',
                    connector.enabled
                      ? 'bg-green-500/20 text-green-500'
                      : 'bg-slate-500/20 text-slate-400'
                  )}
                >
                  {connector.enabled ? 'Enabled' : 'Disabled'}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Heartbeat (if defined) */}
      {agent.heartbeat && (
        <section>
          <h3 className="text-sm font-medium text-foreground mb-3">Heartbeat</h3>
          <dl className="grid gap-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Interval</span>
              <span>{agent.heartbeat.intervalSeconds ?? 30}s</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Timeout</span>
              <span>{agent.heartbeat.timeoutSeconds ?? 120}s</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Max Missed Beats</span>
              <span>{agent.heartbeat.maxMissedBeats ?? 3}</span>
            </div>
          </dl>
        </section>
      )}

      {/* Team Memberships */}
      {memberships.length > 0 && (
        <section>
          <h3 className="text-sm font-medium text-foreground mb-3">Team Memberships</h3>
          <ul className="space-y-2">
            {memberships.map((membership) => (
              <li
                key={membership.teamId}
                className="min-w-0 p-2 bg-muted rounded-lg"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 items-center gap-2">
                    <Users className="h-4 w-4 shrink-0 text-muted-foreground" />
                    <button
                      type="button"
                      className="min-w-0 break-words text-left text-sm font-medium text-primary hover:underline cursor-pointer"
                      onClick={() => navigate(teamDetailPath(membership.teamId))}
                    >
                      {membership.teamDisplayName}
                    </button>
                    {membership.roles.length > 0 && (
                      <span className="break-words text-xs text-muted-foreground">
                        ({membership.roles.join(', ')})
                      </span>
                    )}
                  </div>
                  <StatusBadge status={membership.status} />
                </div>
                <MembershipRuns
                  config={heartbeatByTeam.get(membership.teamId)}
                  onOpenRun={(runId) => navigate(runDetailPath(runId))}
                />
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  )
}

function MembershipRuns({
  config,
  onOpenRun,
}: {
  config: HeartbeatConfig | null | undefined
  onOpenRun: (runId: string) => void
}) {
  const last = config?.lastExecution

  if (config === undefined) {
    return <div className="mt-2 text-xs text-muted-foreground">Loading runs...</div>
  }

  if (!last?.runId) {
    return <div className="mt-2 text-xs text-muted-foreground">No recent runs for this membership.</div>
  }
  const runId = last.runId

  return (
    <div className="mt-2 flex min-w-0 flex-wrap items-center gap-2 rounded-md bg-background/60 px-2 py-1.5 text-xs">
      <Activity className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      <button
        type="button"
        onClick={() => onOpenRun(runId)}
        className="font-medium text-primary hover:underline"
      >
        Last run
      </button>
      <StatusBadge status={last.status} />
      <span className="break-words text-muted-foreground">
        {last.startedAt ? new Date(last.startedAt).toLocaleString() : runId}
      </span>
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
    <div className="flex min-w-0 flex-wrap items-start gap-3">
      <div className="text-muted-foreground">{icon}</div>
      <dt className="w-full text-sm text-muted-foreground sm:w-auto sm:min-w-[100px]">{label}</dt>
      <dd className={cn('min-w-0 flex-1 break-words text-sm', mono && 'font-mono text-xs')}>{value}</dd>
    </div>
  )
}

function ProfileMetric({ label, value }: { label: string; value: string }) {
  return <div className="min-w-0 rounded-xl border border-border bg-muted/30 p-3"><p className="text-[11px] text-muted-foreground">{label}</p><p className="mt-1 break-words text-sm font-semibold capitalize text-foreground">{value}</p></div>
}

function EvidenceCell({ label, state, detail }: { label: string; state: string; detail: string }) {
  const tone = state === 'Observed'
    ? 'text-emerald-600'
    : state === 'Unavailable' || state === 'Empty history' || state === 'Empty'
      ? 'text-muted-foreground'
      : state === 'Partial'
        ? 'text-amber-600'
        : 'text-foreground'
  return (
    <div className="rounded-xl border border-border/70 bg-muted/30 p-3">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <span className="min-w-0 break-words text-sm font-medium text-foreground">{label}</span>
        <span className={cn('break-words text-xs font-semibold', tone)}>{state}</span>
      </div>
      <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{detail}</p>
    </div>
  )
}

/**
 * Status badge component.
 */
interface StatusBadgeProps {
  status: string
}

function StatusBadge({ status }: StatusBadgeProps) {
  const statusStyles: Record<string, string> = {
    active: 'bg-green-500/20 text-green-500',
    completed: 'bg-green-500/20 text-green-500',
    running: 'bg-amber-500/20 text-amber-500',
    pending: 'bg-blue-500/20 text-blue-500',
    failed: 'bg-red-500/20 text-red-500',
    cancelled: 'bg-slate-500/20 text-slate-400',
    inactive: 'bg-slate-500/20 text-slate-400',
    suspended: 'bg-yellow-500/20 text-yellow-500',
  }

  return (
    <span
      className={cn(
        'px-2 py-0.5 text-xs font-medium rounded-full capitalize',
        statusStyles[status] ?? statusStyles.inactive
      )}
    >
      {status}
    </span>
  )
}
