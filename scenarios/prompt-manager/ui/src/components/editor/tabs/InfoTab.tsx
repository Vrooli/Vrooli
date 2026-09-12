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
import { Clock, Hash, Activity, Folder, Tag, Server, Users } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'
import type { Agent } from '@/types/agent'
import type { AgentTeamMembership } from '@/lib/schemas'
import { getAgentTeams } from '@/services/agentService'
import * as heartbeatService from '@/services/heartbeatService'
import type { HeartbeatConfig } from '@/services/heartbeatService'
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

  useEffect(() => {
    let cancelled = false
    void getAgentTeams(agent.id).then(async (result) => {
      if (cancelled) return
      setMemberships(result)
      const entries = await Promise.all(
        result.map(async (membership) => {
          try {
            const config = await heartbeatService.getHeartbeat(membership.teamId, agent.id)
            return [membership.teamId, config] as const
          } catch {
            return [membership.teamId, null] as const
          }
        })
      )
      setHeartbeatByTeam(new Map(entries))
    })
    return () => { cancelled = true }
  }, [agent.id])

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Unknown'
    return new Date(dateString).toLocaleString()
  }

  return (
    <div className="space-y-6">
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
                      className="truncate text-sm font-medium text-primary hover:underline cursor-pointer"
                      onClick={() => navigate(teamDetailPath(membership.teamId))}
                    >
                      {membership.teamDisplayName}
                    </button>
                    {membership.roles.length > 0 && (
                      <span className="truncate text-xs text-muted-foreground">
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
      <span className="truncate text-muted-foreground">
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
    <div className="flex items-center gap-3">
      <div className="text-muted-foreground">{icon}</div>
      <dt className="text-sm text-muted-foreground min-w-[100px]">{label}</dt>
      <dd className={cn('text-sm flex-1', mono && 'font-mono text-xs')}>{value}</dd>
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
