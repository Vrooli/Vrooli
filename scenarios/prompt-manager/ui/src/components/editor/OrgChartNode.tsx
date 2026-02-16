/**
 * OrgChartNode - Custom React Flow node for org chart visualization.
 *
 * Displays:
 * - Avatar (agent body/head colors)
 * - Name
 * - Role pills
 * - Status badge
 *
 * Features:
 * - Selected state with ring highlight
 * - Click to select
 * - Compact layout for tree visualization
 */

import { memo } from 'react'
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
import { Heart, HeartOff } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { OrgChartNodeData } from '@/types/orgChart'
import type { TeamRole } from '@/types/team'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'

type OrgChartNodeProps = NodeProps<Node<OrgChartNodeData, 'orgMember'>>

/**
 * Status badge color mapping.
 */
const statusStyles: Record<string, string> = {
  active: 'bg-green-500/20 text-green-400 border-green-500/30',
  inactive: 'bg-slate-500/20 text-slate-400 border-slate-500/30',
  pending: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
}

/**
 * OrgChartNode component.
 */
function OrgChartNodeComponent({ data }: OrgChartNodeProps) {
  const { member, appearance, teamRoles, isSelected, onSelect, managerName, directReportCount, heartbeatEnabled, heartbeatStatus } = data

  // Get roles that this member has
  const memberRoles: TeamRole[] = teamRoles.filter((role) => member.roles.includes(role.id))

  const handleClick = () => {
    onSelect(member.agentId)
  }

  return (
    <div
      onClick={handleClick}
      className={cn(
        'relative px-4 py-3 bg-card border rounded-lg cursor-pointer transition-all',
        'hover:bg-muted/50 hover:border-primary/50',
        'min-w-[160px] max-w-[220px]',
        isSelected
          ? 'border-primary ring-2 ring-primary/50 bg-primary/5'
          : 'border-border'
      )}
    >
      {/* Target handle (for incoming edges - from manager) */}
      <Handle
        type="target"
        position={Position.Top}
        className="!w-3 !h-3 !bg-primary/50 !border-2 !border-background"
      />

      {/* Content */}
      <div className="flex items-start gap-3">
        {/* Agent color badge */}
        <AgentColorBadge appearance={appearance} size="md" />

        {/* Info */}
        <div className="flex-1 min-w-0">
          {/* Name and status row */}
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium truncate flex-1">
              {member.displayName}
            </p>
            <span
              className={cn(
                'px-1.5 py-0.5 text-[10px] font-medium rounded border flex-shrink-0',
                statusStyles[member.status] ?? statusStyles.inactive
              )}
            >
              {member.status}
            </span>
            {heartbeatEnabled !== undefined && (
              heartbeatEnabled ? (
                <span title={heartbeatStatus === 'running' ? 'Heartbeat running' : 'Heartbeat enabled'}>
                  <Heart
                    className={cn(
                      'h-3 w-3 flex-shrink-0',
                      heartbeatStatus === 'running'
                        ? 'text-green-400 animate-pulse'
                        : 'text-primary/50'
                    )}
                  />
                </span>
              ) : (
                <span title="Heartbeat disabled">
                  <HeartOff
                    className="h-3 w-3 flex-shrink-0 text-muted-foreground/50"
                  />
                </span>
              )
            )}
          </div>

          {/* Role pills */}
          {memberRoles.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-1.5">
              {memberRoles.slice(0, 3).map((role) => (
                <span
                  key={role.id}
                  className="px-1.5 py-0.5 text-[10px] font-medium bg-primary/20 text-primary rounded-full"
                >
                  {role.name}
                </span>
              ))}
              {memberRoles.length > 3 && (
                <span className="px-1.5 py-0.5 text-[10px] font-medium bg-muted text-muted-foreground rounded-full">
                  +{memberRoles.length - 3}
                </span>
              )}
            </div>
          )}

          {/* Relationship labels */}
          {(directReportCount > 0 || managerName) && (
            <div className="mt-1.5 space-y-0.5">
              {directReportCount > 0 && (
                <p className="text-[10px] text-muted-foreground">
                  Directs {directReportCount} member{directReportCount !== 1 ? 's' : ''}
                </p>
              )}
              {managerName && (
                <p className="text-[10px] text-muted-foreground">
                  Reports to {managerName}
                </p>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Source handle (for outgoing edges - to reports) */}
      <Handle
        type="source"
        position={Position.Bottom}
        className="!w-3 !h-3 !bg-primary/50 !border-2 !border-background"
      />
    </div>
  )
}

export const OrgChartNode = memo(OrgChartNodeComponent)
