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
import { Clock, Hash, Users, Shield, Target } from 'lucide-react'
import type { TeamDetails, TeamRole, UpdateTeamRequest } from '@/types/team'
import * as heartbeatService from '@/services/heartbeatService'
import type { HeartbeatConfig } from '@/services/heartbeatService'
import { ExpandableDescription } from '@/components/shared/ExpandableDescription'
import { RolesTab } from './RolesTab'

interface TeamInfoTabProps {
  team: TeamDetails
  onSetRoles: (roles: TeamRole[]) => Promise<TeamRole[]>
  onUpdate: (updates: UpdateTeamRequest) => Promise<void>
}

/**
 * Info display tab component for teams.
 */
export function TeamInfoTab({ team, onSetRoles, onUpdate }: TeamInfoTabProps) {
  const [heartbeatConfigs, setHeartbeatConfigs] = useState<HeartbeatConfig[]>([])
  const [isLoadingHeartbeats, setIsLoadingHeartbeats] = useState(false)
  const [heartbeatError, setHeartbeatError] = useState<string | null>(null)

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

  const upcomingHeartbeats = useMemo(() => {
    const membersById = new Map(team.members.map((member) => [member.agentId, member]))

    return heartbeatConfigs
      .filter((config) => config.enabled && config.nextExecution)
      .map((config) => {
        const nextRun = config.nextExecution ? new Date(config.nextExecution) : null
        return {
          config,
          memberName: membersById.get(config.agentId)?.displayName ?? config.agentId,
          nextRun,
        }
      })
      .filter(
        (entry): entry is { config: HeartbeatConfig; memberName: string; nextRun: Date } =>
          !!entry.nextRun && !Number.isNaN(entry.nextRun.getTime())
      )
      .sort((a, b) => a.nextRun.getTime() - b.nextRun.getTime())
      .slice(0, 5)
  }, [heartbeatConfigs, team.members])

  const enabledHeartbeatCount = useMemo(() => {
    return heartbeatConfigs.filter((config) => config.enabled).length
  }, [heartbeatConfigs])

  useEffect(() => {
    let isActive = true
    const loadHeartbeats = async () => {
      setIsLoadingHeartbeats(true)
      setHeartbeatError(null)
      try {
        const configs = await heartbeatService.listHeartbeats(team.id)
        if (!isActive) return
        setHeartbeatConfigs(configs)
      } catch (error) {
        if (!isActive) return
        console.warn('Failed to load heartbeat schedule:', error)
        setHeartbeatConfigs([])
        setHeartbeatError('Unable to load heartbeat schedule.')
      } finally {
        if (isActive) setIsLoadingHeartbeats(false)
      }
    }

    void loadHeartbeats()
    return () => {
      isActive = false
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
        {!team.enabled && (
          <div className="flex items-start gap-3 p-3 mb-3 bg-amber-500/10 border border-amber-500/30 rounded-lg">
            <Clock className="h-4 w-4 text-amber-500 mt-0.5 flex-shrink-0" />
            <div>
              <p className="text-sm text-amber-500 font-medium">Team is turned off</p>
              <p className="text-xs text-muted-foreground mt-0.5">
                Heartbeats are paused
                {enabledHeartbeatCount > 0 ? ` (${enabledHeartbeatCount} configured)` : ''}.
              </p>
            </div>
          </div>
        )}
        {isLoadingHeartbeats ? (
          <p className="text-sm text-muted-foreground">Loading heartbeat schedule...</p>
        ) : heartbeatError ? (
          <p className="text-sm text-destructive">{heartbeatError}</p>
        ) : upcomingHeartbeats.length === 0 ? (
          <div className="flex items-start gap-3 p-3 bg-muted rounded-lg">
            <Clock className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
            <div>
              <p className="text-sm text-muted-foreground">
                {team.enabled ? 'No upcoming heartbeats scheduled.' : 'Team is off. Heartbeats are paused.'}
              </p>
              <p className="text-xs text-muted-foreground/70 mt-0.5">
                {team.enabled
                  ? 'Enable heartbeats on team members to populate this list.'
                  : 'Turn the team on to resume scheduled heartbeats.'}
              </p>
            </div>
          </div>
        ) : (
          <ul className="space-y-2">
            {upcomingHeartbeats.map((entry) => (
              <li
                key={`${entry.config.agentId}-${entry.nextRun.toISOString()}`}
                className="flex items-start justify-between gap-4 px-3 py-2 bg-muted rounded-lg"
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
        <RolesTab team={team} onSetRoles={onSetRoles} />
      </section>
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
