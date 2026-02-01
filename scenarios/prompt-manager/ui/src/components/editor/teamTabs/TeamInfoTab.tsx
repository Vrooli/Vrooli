/**
 * TeamInfoTab - Team information display tab.
 *
 * Features:
 * - Team ID (read-only)
 * - Created/Updated timestamps
 * - Member count
 * - Role count
 */

import { Clock, Hash, Users, Shield, Target } from 'lucide-react'
import type { TeamDetails } from '@/types/team'

interface TeamInfoTabProps {
  team: TeamDetails
}

/**
 * Info display tab component for teams.
 */
export function TeamInfoTab({ team }: TeamInfoTabProps) {
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
      {team.mission && (
        <section>
          <h3 className="text-sm font-medium text-foreground mb-3">Mission</h3>
          <div className="flex items-start gap-3 p-3 bg-muted rounded-lg">
            <Target className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
            <p className="text-sm text-foreground">{team.mission}</p>
          </div>
        </section>
      )}

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

      {/* Roles Summary */}
      {team.roles.length > 0 && (
        <section>
          <h3 className="text-sm font-medium text-foreground mb-3">Defined Roles</h3>
          <ul className="space-y-2">
            {team.roles.map((role) => (
              <li
                key={role.id}
                className="flex items-start gap-2 px-3 py-2 bg-muted rounded-lg"
              >
                <Shield className="h-4 w-4 text-primary mt-0.5 flex-shrink-0" />
                <div>
                  <p className="text-sm font-medium">{role.name}</p>
                  {role.description && (
                    <p className="text-xs text-muted-foreground mt-0.5">{role.description}</p>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Skill Grants Summary */}
      {team.defaults?.skillGrantsByRole && Object.keys(team.defaults.skillGrantsByRole).length > 0 && (
        <section>
          <h3 className="text-sm font-medium text-foreground mb-3">Skill Grants Configuration</h3>
          <dl className="space-y-2">
            {Object.entries(team.defaults.skillGrantsByRole).map(([roleId, skills]) => {
              const role = team.roles.find((r) => r.id === roleId)
              return (
                <div key={roleId} className="px-3 py-2 bg-muted rounded-lg">
                  <dt className="text-sm font-medium">{role?.name ?? roleId}</dt>
                  <dd className="text-xs text-muted-foreground mt-1">
                    {skills.length} skill{skills.length !== 1 ? 's' : ''} granted
                  </dd>
                </div>
              )
            })}
          </dl>
        </section>
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
