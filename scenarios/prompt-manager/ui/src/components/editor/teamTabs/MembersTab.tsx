/**
 * MembersTab - Team member management tab.
 *
 * Features:
 * - List current members with roles
 * - Add member (select from agents)
 * - Assign/update roles per member
 * - Remove member button
 * - Status indicator (active/inactive/pending)
 * - Heartbeat configuration per member
 */

import { useState, useCallback, useEffect } from 'react'
import { X, UserPlus, Users, Clock, Play, Pause, Settings } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamDetails, TeamMember, AddMemberRequest, UpdateMemberRequest } from '@/types/team'
import type { Agent } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import * as heartbeatService from '@/services/heartbeatService'
import type { HeartbeatConfig } from '@/services/heartbeatService'

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
  const [heartbeatConfigs, setHeartbeatConfigs] = useState<Record<string, HeartbeatConfig>>({})
  const [selectedMember, setSelectedMember] = useState<string | null>(null)
  const [showHeartbeatModal, setShowHeartbeatModal] = useState(false)

  // Load heartbeat configs
  useEffect(() => {
    const loadHeartbeats = async () => {
      try {
        const configs = await heartbeatService.listHeartbeats(team.id)
        const configMap: Record<string, HeartbeatConfig> = {}
        for (const config of configs) {
          configMap[config.agentId] = config
        }
        setHeartbeatConfigs(configMap)
      } catch (error) {
        console.warn('Failed to load heartbeat configs:', error)
      }
    }
    void loadHeartbeats()
  }, [team.id])

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

  // Handle heartbeat toggle
  const handleHeartbeatToggle = useCallback(
    async (agentId: string) => {
      const config = heartbeatConfigs[agentId]
      if (config) {
        // Toggle enabled state
        const updated = await heartbeatService.updateHeartbeat(team.id, agentId, {
          enabled: !config.enabled,
        })
        setHeartbeatConfigs((prev) => ({ ...prev, [agentId]: updated }))
      } else {
        // Open modal to create config
        setSelectedMember(agentId)
        setShowHeartbeatModal(true)
      }
    },
    [team.id, heartbeatConfigs]
  )

  // Handle heartbeat config click
  const handleHeartbeatConfig = useCallback((agentId: string) => {
    setSelectedMember(agentId)
    setShowHeartbeatModal(true)
  }, [])

  // Handle heartbeat trigger
  const handleHeartbeatTrigger = useCallback(
    async (agentId: string) => {
      try {
        await heartbeatService.triggerHeartbeat(team.id, agentId)
        // Refresh config to show updated lastExecution
        const updated = await heartbeatService.getHeartbeat(team.id, agentId)
        if (updated) {
          setHeartbeatConfigs((prev) => ({ ...prev, [agentId]: updated }))
        }
      } catch (error) {
        console.error('Failed to trigger heartbeat:', error)
      }
    },
    [team.id]
  )

  // Handle heartbeat save
  const handleHeartbeatSave = useCallback(
    async (agentId: string, schedule: string, profileKey?: string) => {
      try {
        const config = heartbeatConfigs[agentId]
        let updated: HeartbeatConfig
        if (config) {
          updated = await heartbeatService.updateHeartbeat(team.id, agentId, {
            schedule,
            profileKey,
          })
        } else {
          updated = await heartbeatService.createHeartbeat(team.id, agentId, {
            schedule,
            profileKey,
            enabled: false,
          })
        }
        setHeartbeatConfigs((prev) => ({ ...prev, [agentId]: updated }))
        setShowHeartbeatModal(false)
        setSelectedMember(null)
      } catch (error) {
        console.error('Failed to save heartbeat config:', error)
      }
    },
    [team.id, heartbeatConfigs]
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
          {team.members.map((member) => {
            const heartbeat = heartbeatConfigs[member.agentId]
            return (
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

                  {/* Heartbeat status */}
                  <div className="flex items-center gap-2 mt-2">
                    <HeartbeatIndicator
                      config={heartbeat}
                      onToggle={() => void handleHeartbeatToggle(member.agentId)}
                      onConfigure={() => handleHeartbeatConfig(member.agentId)}
                      onTrigger={() => void handleHeartbeatTrigger(member.agentId)}
                    />
                  </div>
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
            )
          })}
        </ul>
      )}

      {/* Member Picker Modal */}
      {showPicker && (
        <MemberPickerModal
          availableAgents={availableAgents}
          onSelect={(agentId) => void handleAddMember(agentId)}
          onClose={() => setShowPicker(false)}
        />
      )}

      {/* Heartbeat Config Modal */}
      {showHeartbeatModal && selectedMember && (
        <HeartbeatConfigModal
          teamId={team.id}
          agentId={selectedMember}
          config={heartbeatConfigs[selectedMember]}
          onSave={(schedule, profileKey) =>
            void handleHeartbeatSave(selectedMember, schedule, profileKey)
          }
          onClose={() => {
            setShowHeartbeatModal(false)
            setSelectedMember(null)
          }}
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
export interface MemberPickerModalProps {
  availableAgents: Agent[]
  onSelect: (agentId: string) => void
  onClose: () => void
}

export function MemberPickerModal({ availableAgents, onSelect, onClose }: MemberPickerModalProps) {
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
                    {/* Agent color badge */}
                    <AgentColorBadge appearance={agent.appearance} size="sm" />
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

/**
 * Heartbeat status indicator with controls.
 */
interface HeartbeatIndicatorProps {
  config?: HeartbeatConfig
  onToggle: () => void
  onConfigure: () => void
  onTrigger: () => void
}

function HeartbeatIndicator({ config, onToggle, onConfigure, onTrigger }: HeartbeatIndicatorProps) {
  if (!config) {
    return (
      <button
        type="button"
        onClick={onConfigure}
        className={cn(
          'flex items-center gap-1.5 px-2 py-1 text-xs rounded-md',
          'text-muted-foreground hover:text-foreground hover:bg-muted/50',
          'transition-colors'
        )}
        title="Configure heartbeat"
      >
        <Clock className="h-3.5 w-3.5" />
        <span>No heartbeat</span>
      </button>
    )
  }

  const statusColor = config.enabled
    ? config.lastExecution?.status === 'failed'
      ? 'text-destructive'
      : 'text-green-500'
    : 'text-muted-foreground'

  return (
    <div className="flex items-center gap-1">
      {/* Status indicator */}
      <div
        className={cn(
          'flex items-center gap-1.5 px-2 py-1 text-xs rounded-md',
          statusColor
        )}
      >
        <Clock className="h-3.5 w-3.5" />
        <span>{config.schedule}</span>
        {config.enabled && (
          <span className="text-[10px] px-1 py-0.5 bg-green-500/20 rounded">ON</span>
        )}
      </div>

      {/* Toggle button */}
      <button
        type="button"
        onClick={onToggle}
        className={cn(
          'p-1 rounded hover:bg-muted transition-colors',
          config.enabled ? 'text-green-500' : 'text-muted-foreground'
        )}
        title={config.enabled ? 'Disable heartbeat' : 'Enable heartbeat'}
      >
        {config.enabled ? (
          <Pause className="h-3.5 w-3.5" />
        ) : (
          <Play className="h-3.5 w-3.5" />
        )}
      </button>

      {/* Trigger button */}
      <button
        type="button"
        onClick={onTrigger}
        className="p-1 rounded text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
        title="Trigger heartbeat now"
      >
        <Play className="h-3.5 w-3.5" />
      </button>

      {/* Configure button */}
      <button
        type="button"
        onClick={onConfigure}
        className="p-1 rounded text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
        title="Configure heartbeat"
      >
        <Settings className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}

/**
 * Modal for configuring heartbeat settings.
 */
interface HeartbeatConfigModalProps {
  teamId: string
  agentId: string
  config?: HeartbeatConfig
  onSave: (schedule: string, profileKey?: string) => void
  onClose: () => void
}

function HeartbeatConfigModal({
  teamId,
  agentId,
  config,
  onSave,
  onClose,
}: HeartbeatConfigModalProps) {
  const [schedule, setSchedule] = useState(config?.schedule || '0 */6 * * *')
  const [profileKey, setProfileKey] = useState(config?.profileKey || '')
  const [responsibilities, setResponsibilities] = useState('')
  const [heartbeatInstructions, setHeartbeatInstructions] = useState('')
  const [activeTab, setActiveTab] = useState<'schedule' | 'responsibilities' | 'instructions'>('schedule')
  const [loading, setLoading] = useState(true)

  // Load documents
  useEffect(() => {
    const loadDocs = async () => {
      try {
        const [resp, instr] = await Promise.all([
          heartbeatService.getResponsibilities(teamId, agentId),
          heartbeatService.getHeartbeatInstructions(teamId, agentId),
        ])
        setResponsibilities(resp)
        setHeartbeatInstructions(instr)
      } catch (error) {
        console.warn('Failed to load member documents:', error)
      } finally {
        setLoading(false)
      }
    }
    void loadDocs()
  }, [teamId, agentId])

  const handleSaveSchedule = () => {
    onSave(schedule, profileKey || undefined)
  }

  const handleSaveResponsibilities = async () => {
    try {
      await heartbeatService.setResponsibilities(teamId, agentId, responsibilities)
    } catch (error) {
      console.error('Failed to save responsibilities:', error)
    }
  }

  const handleSaveInstructions = async () => {
    try {
      await heartbeatService.setHeartbeatInstructions(teamId, agentId, heartbeatInstructions)
    } catch (error) {
      console.error('Failed to save instructions:', error)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onClose} />

      {/* Modal */}
      <div className="relative w-full max-w-lg mx-4 bg-card border border-border rounded-xl shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h3 className="font-medium">Heartbeat Configuration</h3>
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-border">
          <button
            type="button"
            onClick={() => setActiveTab('schedule')}
            className={cn(
              'flex-1 px-4 py-2 text-sm font-medium',
              activeTab === 'schedule'
                ? 'text-primary border-b-2 border-primary'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            Schedule
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('responsibilities')}
            className={cn(
              'flex-1 px-4 py-2 text-sm font-medium',
              activeTab === 'responsibilities'
                ? 'text-primary border-b-2 border-primary'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            Responsibilities
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('instructions')}
            className={cn(
              'flex-1 px-4 py-2 text-sm font-medium',
              activeTab === 'instructions'
                ? 'text-primary border-b-2 border-primary'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            Instructions
          </button>
        </div>

        {/* Content */}
        <div className="p-4">
          {activeTab === 'schedule' && (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">Schedule (Cron)</label>
                <input
                  type="text"
                  value={schedule}
                  onChange={(e) => setSchedule(e.target.value)}
                  className={cn(
                    'w-full px-3 py-2 text-sm',
                    'bg-muted border border-border rounded-lg',
                    'text-foreground placeholder:text-muted-foreground',
                    'focus:outline-none focus:ring-2 focus:ring-primary'
                  )}
                  placeholder="0 */6 * * *"
                />
                <div className="mt-2 flex flex-wrap gap-1">
                  {heartbeatService.SCHEDULE_PRESETS.map((preset) => (
                    <button
                      key={preset.value}
                      type="button"
                      onClick={() => setSchedule(preset.value)}
                      className={cn(
                        'px-2 py-1 text-xs rounded',
                        schedule === preset.value
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-muted text-muted-foreground hover:bg-muted/80'
                      )}
                    >
                      {preset.label}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">Profile Key (optional)</label>
                <input
                  type="text"
                  value={profileKey}
                  onChange={(e) => setProfileKey(e.target.value)}
                  className={cn(
                    'w-full px-3 py-2 text-sm',
                    'bg-muted border border-border rounded-lg',
                    'text-foreground placeholder:text-muted-foreground',
                    'focus:outline-none focus:ring-2 focus:ring-primary'
                  )}
                  placeholder="prompt-manager-heartbeat"
                />
              </div>

              {config?.lastExecution && (
                <div className="p-3 bg-muted rounded-lg">
                  <p className="text-xs font-medium mb-1">Last Execution</p>
                  <p className="text-xs text-muted-foreground">
                    Status: {config.lastExecution.status}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Started: {config.lastExecution.startedAt}
                  </p>
                  {config.lastExecution.error && (
                    <p className="text-xs text-destructive mt-1">
                      Error: {config.lastExecution.error}
                    </p>
                  )}
                </div>
              )}

              <button
                type="button"
                onClick={handleSaveSchedule}
                className={cn(
                  'w-full px-4 py-2 text-sm font-medium rounded-lg',
                  'bg-primary text-primary-foreground hover:bg-primary/90',
                  'transition-colors'
                )}
              >
                Save Schedule
              </button>
            </div>
          )}

          {activeTab === 'responsibilities' && (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">
                  RESPONSIBILITIES.md
                </label>
                <textarea
                  value={responsibilities}
                  onChange={(e) => setResponsibilities(e.target.value)}
                  className={cn(
                    'w-full h-64 px-3 py-2 text-sm font-mono',
                    'bg-muted border border-border rounded-lg',
                    'text-foreground placeholder:text-muted-foreground',
                    'focus:outline-none focus:ring-2 focus:ring-primary',
                    'resize-none'
                  )}
                  placeholder="# Responsibilities&#10;&#10;Describe what this agent is responsible for in this team..."
                  disabled={loading}
                />
              </div>
              <button
                type="button"
                onClick={() => void handleSaveResponsibilities()}
                className={cn(
                  'w-full px-4 py-2 text-sm font-medium rounded-lg',
                  'bg-primary text-primary-foreground hover:bg-primary/90',
                  'transition-colors'
                )}
              >
                Save Responsibilities
              </button>
            </div>
          )}

          {activeTab === 'instructions' && (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">
                  HEARTBEAT.md
                </label>
                <textarea
                  value={heartbeatInstructions}
                  onChange={(e) => setHeartbeatInstructions(e.target.value)}
                  className={cn(
                    'w-full h-64 px-3 py-2 text-sm font-mono',
                    'bg-muted border border-border rounded-lg',
                    'text-foreground placeholder:text-muted-foreground',
                    'focus:outline-none focus:ring-2 focus:ring-primary',
                    'resize-none'
                  )}
                  placeholder="# Heartbeat Task&#10;&#10;Describe what this agent should do on each heartbeat..."
                  disabled={loading}
                />
              </div>
              <button
                type="button"
                onClick={() => void handleSaveInstructions()}
                className={cn(
                  'w-full px-4 py-2 text-sm font-medium rounded-lg',
                  'bg-primary text-primary-foreground hover:bg-primary/90',
                  'transition-colors'
                )}
              >
                Save Instructions
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
