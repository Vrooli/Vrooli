/**
 * MembersTab - Team member management tab.
 *
 * Features:
 * - List current members with roles
 * - Add member (select from agents)
 * - Assign/update roles per member
 * - Remove member button
 * - Status indicator (active/inactive/pending)
 */

import { useState, useCallback } from 'react'
import { X, UserPlus, Users } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamDetails, TeamMember, AddMemberRequest, UpdateMemberRequest } from '@/types/team'
import type { Agent } from '@/types/agent'

interface MembersTabProps {
  team: TeamDetails
  /** All available agents for adding as members */
  allAgents?: Agent[]
  onAddMember: (request: AddMemberRequest) => Promise<TeamMember>
  onUpdateMember: (agentId: string, request: UpdateMemberRequest) => Promise<TeamMember>
  onRemoveMember: (agentId: string) => Promise<void>
}

/**
 * Members management tab component.
 */
export function MembersTab({
  team,
  allAgents = [],
  onAddMember,
  onUpdateMember,
  onRemoveMember,
}: MembersTabProps) {
  const [showPicker, setShowPicker] = useState(false)

  // Get available agents (not already members)
  const memberAgentIds = new Set(team.members.map((m) => m.agentId))
  const availableAgents = allAgents.filter((a) => !memberAgentIds.has(a.id))

  // Handle member addition
  const handleAddMember = useCallback(
    async (agentId: string) => {
      await onAddMember({ agentId, roles: [] })
      setShowPicker(false)
    },
    [onAddMember]
  )

  // Handle role toggle
  const handleToggleRole = useCallback(
    async (member: TeamMember, roleId: string) => {
      const newRoles = member.roles.includes(roleId)
        ? member.roles.filter((r) => r !== roleId)
        : [...member.roles, roleId]
      await onUpdateMember(member.agentId, { roles: newRoles })
    },
    [onUpdateMember]
  )

  // Handle status change
  const handleStatusChange = useCallback(
    async (member: TeamMember, newStatus: string) => {
      await onUpdateMember(member.agentId, { status: newStatus })
    },
    [onUpdateMember]
  )

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">
          Team Members ({team.members.length})
        </h3>
        <button
          type="button"
          onClick={() => setShowPicker(true)}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg',
            'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors'
          )}
        >
          <UserPlus className="h-3.5 w-3.5" />
          Add Member
        </button>
      </div>

      {/* Members List */}
      {team.members.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-8 text-center">
          <Users className="h-10 w-10 text-muted-foreground/50 mb-3" />
          <p className="text-sm text-muted-foreground">No members yet</p>
          <p className="text-xs text-muted-foreground/70 mt-1">
            Click &quot;Add Member&quot; to add agents to this team
          </p>
        </div>
      ) : (
        <ul className="space-y-2">
          {team.members.map((member) => (
            <li
              key={member.agentId}
              className={cn(
                'flex items-start gap-3 px-3 py-3',
                'bg-muted rounded-lg group',
                'hover:bg-muted/80 transition-colors'
              )}
            >
              {/* Member info */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium truncate">{member.displayName}</p>
                  <StatusBadge
                    status={member.status}
                    onChange={(status) => void handleStatusChange(member, status)}
                  />
                </div>

                {/* Roles */}
                {team.roles.length > 0 && (
                  <div className="flex flex-wrap gap-1 mt-2">
                    {team.roles.map((role) => (
                      <button
                        key={role.id}
                        type="button"
                        onClick={() => void handleToggleRole(member, role.id)}
                        className={cn(
                          'px-2 py-0.5 text-xs rounded-full transition-colors',
                          member.roles.includes(role.id)
                            ? 'bg-primary/20 text-primary'
                            : 'bg-background text-muted-foreground hover:bg-primary/10'
                        )}
                      >
                        {role.name}
                      </button>
                    ))}
                  </div>
                )}
              </div>

              {/* Remove button */}
              <button
                type="button"
                onClick={() => void onRemoveMember(member.agentId)}
                className={cn(
                  'p-1 rounded opacity-0 group-hover:opacity-100',
                  'text-muted-foreground hover:text-destructive hover:bg-destructive/10',
                  'transition-all'
                )}
                title="Remove member"
              >
                <X className="h-4 w-4" />
              </button>
            </li>
          ))}
        </ul>
      )}

      {/* Member Picker Modal */}
      {showPicker && (
        <MemberPickerModal
          availableAgents={availableAgents}
          onSelect={handleAddMember}
          onClose={() => setShowPicker(false)}
        />
      )}
    </div>
  )
}

/**
 * Status badge with dropdown.
 */
interface StatusBadgeProps {
  status: string
  onChange: (status: string) => void
}

function StatusBadge({ status, onChange }: StatusBadgeProps) {
  const statusStyles: Record<string, string> = {
    active: 'bg-green-500/20 text-green-500 border-green-500/30',
    inactive: 'bg-slate-500/20 text-slate-400 border-slate-500/30',
    pending: 'bg-yellow-500/20 text-yellow-500 border-yellow-500/30',
  }

  return (
    <select
      value={status}
      onChange={(e) => onChange(e.target.value)}
      className={cn(
        'px-2 py-0.5 text-[10px] font-medium rounded-full border cursor-pointer',
        'focus:outline-none focus:ring-2 focus:ring-primary',
        statusStyles[status] ?? statusStyles.inactive
      )}
    >
      <option value="active">Active</option>
      <option value="inactive">Inactive</option>
      <option value="pending">Pending</option>
    </select>
  )
}

/**
 * Modal for selecting agents to add as members.
 */
interface MemberPickerModalProps {
  availableAgents: Agent[]
  onSelect: (agentId: string) => void
  onClose: () => void
}

function MemberPickerModal({ availableAgents, onSelect, onClose }: MemberPickerModalProps) {
  const [search, setSearch] = useState('')

  // Filter agents by search query
  const filteredAgents = availableAgents.filter(
    (agent) =>
      agent.displayName.toLowerCase().includes(search.toLowerCase()) ||
      agent.description?.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative w-full max-w-md mx-4 bg-card border border-border rounded-xl shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h3 className="font-medium">Add Team Member</h3>
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Search */}
        <div className="px-4 py-3 border-b border-border">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search agents..."
            className={cn(
              'w-full px-3 py-2 text-sm',
              'bg-muted border border-border rounded-lg',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-2 focus:ring-primary'
            )}
            autoFocus
          />
        </div>

        {/* Agents list */}
        <div className="max-h-64 overflow-y-auto">
          {filteredAgents.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-muted-foreground">
              {availableAgents.length === 0
                ? 'All agents are already team members'
                : 'No agents match your search'}
            </div>
          ) : (
            <ul className="p-2 space-y-1">
              {filteredAgents.map((agent) => (
                <li key={agent.id}>
                  <button
                    type="button"
                    onClick={() => onSelect(agent.id)}
                    className={cn(
                      'w-full flex items-center gap-3 px-3 py-2',
                      'rounded-lg text-left',
                      'hover:bg-muted transition-colors'
                    )}
                  >
                    {/* Agent avatar */}
                    <div
                      className="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0"
                      style={{ backgroundColor: agent.appearance?.body ?? '#6366f1' }}
                    >
                      <div
                        className="w-4 h-4 rounded-full"
                        style={{ backgroundColor: agent.appearance?.head ?? '#818cf8' }}
                      />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium">{agent.displayName}</p>
                      {agent.description && (
                        <p className="text-xs text-muted-foreground line-clamp-1">
                          {agent.description}
                        </p>
                      )}
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
