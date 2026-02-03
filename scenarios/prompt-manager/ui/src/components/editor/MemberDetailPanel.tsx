/**
 * MemberDetailPanel - Right panel for editing selected team member.
 *
 * Features:
 * - Header with avatar, name, status dropdown
 * - Roles section with chip toggles
 * - Responsibilities.md markdown editor
 * - Heartbeat section with schedule + instructions.md
 * - Remove member button
 */

import { useState, useEffect, useCallback } from 'react'
import { X, Trash2, Clock, Play, Pause, Save, FileText, AlertCircle, ArrowUpRight, ArrowDownRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamDetails, TeamMember, UpdateMemberRequest } from '@/types/team'
import type { AgentAppearance } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import * as heartbeatService from '@/services/heartbeatService'
import type { HeartbeatConfig } from '@/services/heartbeatService'

// ============================================================================
// Types
// ============================================================================

interface MemberDetailPanelProps {
  team: TeamDetails
  member: TeamMember
  appearance?: AgentAppearance
  manager?: TeamMember | null
  directReports?: TeamMember[]
  onUpdateMember: (agentId: string, request: UpdateMemberRequest) => Promise<TeamMember>
  onRemoveMember: (agentId: string) => Promise<void>
  onClose: () => void
  className?: string
}

// ============================================================================
// Status Styles
// ============================================================================

const statusStyles: Record<string, string> = {
  active: 'bg-green-500/20 text-green-400 border-green-500/30',
  inactive: 'bg-slate-500/20 text-slate-400 border-slate-500/30',
  pending: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
}

// ============================================================================
// Component
// ============================================================================

export function MemberDetailPanel({
  team,
  member,
  appearance,
  manager = null,
  directReports = [],
  onUpdateMember,
  onRemoveMember,
  onClose,
  className,
}: MemberDetailPanelProps) {
  // Local state
  const [activeSection, setActiveSection] = useState<'roles' | 'responsibilities' | 'heartbeat'>('roles')
  const [responsibilities, setResponsibilities] = useState('')
  const [heartbeatInstructions, setHeartbeatInstructions] = useState('')
  const [heartbeatConfig, setHeartbeatConfig] = useState<HeartbeatConfig | null>(null)
  const [schedule, setSchedule] = useState('0 */6 * * *')
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Dirty tracking
  const [isResponsibilitiesDirty, setIsResponsibilitiesDirty] = useState(false)
  const [isInstructionsDirty, setIsInstructionsDirty] = useState(false)

  const reportNames = directReports.map((report) => report.displayName)
  const displayReports = reportNames.slice(0, 3)
  const remainingReports = reportNames.length - displayReports.length

  // Load member documents and heartbeat config
  useEffect(() => {
    const loadData = async () => {
      setIsLoading(true)
      setError(null)
      try {
        const [resp, instr, config] = await Promise.all([
          heartbeatService.getResponsibilities(team.id, member.agentId),
          heartbeatService.getHeartbeatInstructions(team.id, member.agentId),
          heartbeatService.getHeartbeat(team.id, member.agentId),
        ])
        setResponsibilities(resp)
        setHeartbeatInstructions(instr)
        setHeartbeatConfig(config)
        if (config?.schedule) {
          setSchedule(config.schedule)
        }
        setIsResponsibilitiesDirty(false)
        setIsInstructionsDirty(false)
      } catch (err) {
        console.warn('Failed to load member data:', err)
        setError('Failed to load member data')
      } finally {
        setIsLoading(false)
      }
    }
    void loadData()
  }, [team.id, member.agentId])

  // Handle role toggle
  const handleToggleRole = useCallback(
    async (roleId: string) => {
      const newRoles = member.roles.includes(roleId)
        ? member.roles.filter((r) => r !== roleId)
        : [...member.roles, roleId]
      try {
        await onUpdateMember(member.agentId, { roles: newRoles })
      } catch (err) {
        console.error('Failed to update roles:', err)
      }
    },
    [member.agentId, member.roles, onUpdateMember]
  )

  // Handle status change
  const handleStatusChange = useCallback(
    async (newStatus: string) => {
      try {
        await onUpdateMember(member.agentId, { status: newStatus })
      } catch (err) {
        console.error('Failed to update status:', err)
      }
    },
    [member.agentId, onUpdateMember]
  )

  // Save responsibilities
  const handleSaveResponsibilities = useCallback(async () => {
    setIsSaving(true)
    try {
      await heartbeatService.setResponsibilities(team.id, member.agentId, responsibilities)
      setIsResponsibilitiesDirty(false)
    } catch (err) {
      console.error('Failed to save responsibilities:', err)
      setError('Failed to save responsibilities')
    } finally {
      setIsSaving(false)
    }
  }, [team.id, member.agentId, responsibilities])

  // Save heartbeat instructions
  const handleSaveInstructions = useCallback(async () => {
    setIsSaving(true)
    try {
      await heartbeatService.setHeartbeatInstructions(team.id, member.agentId, heartbeatInstructions)
      setIsInstructionsDirty(false)
    } catch (err) {
      console.error('Failed to save instructions:', err)
      setError('Failed to save instructions')
    } finally {
      setIsSaving(false)
    }
  }, [team.id, member.agentId, heartbeatInstructions])

  // Save schedule
  const handleSaveSchedule = useCallback(async () => {
    setIsSaving(true)
    try {
      let updated: HeartbeatConfig
      if (heartbeatConfig) {
        updated = await heartbeatService.updateHeartbeat(team.id, member.agentId, { schedule })
      } else {
        updated = await heartbeatService.createHeartbeat(team.id, member.agentId, {
          schedule,
          enabled: false,
        })
      }
      setHeartbeatConfig(updated)
    } catch (err) {
      console.error('Failed to save schedule:', err)
      setError('Failed to save schedule')
    } finally {
      setIsSaving(false)
    }
  }, [team.id, member.agentId, schedule, heartbeatConfig])

  // Toggle heartbeat enabled
  const handleToggleHeartbeat = useCallback(async () => {
    if (!heartbeatConfig) return
    try {
      const updated = await heartbeatService.updateHeartbeat(team.id, member.agentId, {
        enabled: !heartbeatConfig.enabled,
      })
      setHeartbeatConfig(updated)
    } catch (err) {
      console.error('Failed to toggle heartbeat:', err)
    }
  }, [team.id, member.agentId, heartbeatConfig])

  // Trigger heartbeat now
  const handleTriggerHeartbeat = useCallback(async () => {
    try {
      await heartbeatService.triggerHeartbeat(team.id, member.agentId)
      const updated = await heartbeatService.getHeartbeat(team.id, member.agentId)
      if (updated) setHeartbeatConfig(updated)
    } catch (err) {
      console.error('Failed to trigger heartbeat:', err)
    }
  }, [team.id, member.agentId])

  // Handle remove member
  const handleRemove = useCallback(async () => {
    if (!confirm(`Remove ${member.displayName} from the team?`)) return
    try {
      await onRemoveMember(member.agentId)
      onClose()
    } catch (err) {
      console.error('Failed to remove member:', err)
    }
  }, [member.agentId, member.displayName, onRemoveMember, onClose])

  return (
    <div className={cn('h-full flex flex-col bg-card/50', className)}>
      {/* Header */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-border">
        <div className="flex items-center gap-3">
          {/* Close button */}
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
            aria-label="Close detail panel"
          >
            <X className="h-5 w-5" />
          </button>

          {/* Agent color badge */}
          <AgentColorBadge appearance={appearance} size="lg" />

          {/* Name and status */}
          <div className="flex-1 min-w-0">
            <h3 className="text-lg font-semibold truncate">{member.displayName}</h3>
            <select
              value={member.status}
              onChange={(e) => void handleStatusChange(e.target.value)}
              className={cn(
                'px-2 py-0.5 text-xs font-medium rounded-full border cursor-pointer',
                'focus:outline-none focus:ring-2 focus:ring-primary',
                statusStyles[member.status] ?? statusStyles.inactive
              )}
            >
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
              <option value="pending">Pending</option>
            </select>
          </div>
        </div>
      </div>

      {/* Relationships */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-border bg-muted/40">
        <h4 className="text-xs font-semibold text-muted-foreground mb-2">Relationships</h4>
        <div className="grid gap-2">
          <div className="flex items-center gap-2 text-sm">
            <ArrowUpRight className="h-4 w-4 text-muted-foreground" />
            <span className="text-muted-foreground">Reports to:</span>
            <span className="text-foreground">
              {manager ? manager.displayName : 'None'}
            </span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <ArrowDownRight className="h-4 w-4 text-muted-foreground" />
            <span className="text-muted-foreground">Direct reports:</span>
            {directReports.length === 0 ? (
              <span className="text-foreground">None</span>
            ) : (
              <span className="text-foreground">
                {displayReports.join(', ')}
                {remainingReports > 0 ? ` +${remainingReports} more` : ''}
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Section tabs */}
      <div className="flex-shrink-0 flex border-b border-border">
        <button
          type="button"
          onClick={() => setActiveSection('roles')}
          className={cn(
            'flex-1 px-4 py-2 text-sm font-medium transition-colors',
            activeSection === 'roles'
              ? 'text-primary border-b-2 border-primary'
              : 'text-muted-foreground hover:text-foreground'
          )}
        >
          Roles
        </button>
        <button
          type="button"
          onClick={() => setActiveSection('responsibilities')}
          className={cn(
            'flex-1 px-4 py-2 text-sm font-medium transition-colors relative',
            activeSection === 'responsibilities'
              ? 'text-primary border-b-2 border-primary'
              : 'text-muted-foreground hover:text-foreground'
          )}
        >
          Responsibilities
          {isResponsibilitiesDirty && (
            <span className="absolute top-1 right-2 w-2 h-2 bg-amber-500 rounded-full" />
          )}
        </button>
        <button
          type="button"
          onClick={() => setActiveSection('heartbeat')}
          className={cn(
            'flex-1 px-4 py-2 text-sm font-medium transition-colors relative',
            activeSection === 'heartbeat'
              ? 'text-primary border-b-2 border-primary'
              : 'text-muted-foreground hover:text-foreground'
          )}
        >
          Heartbeat
          {isInstructionsDirty && (
            <span className="absolute top-1 right-2 w-2 h-2 bg-amber-500 rounded-full" />
          )}
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4">
        {/* Error banner */}
        {error && (
          <div className="mb-4 px-3 py-2 bg-destructive/10 border border-destructive/30 rounded-lg flex items-center gap-2">
            <AlertCircle className="h-4 w-4 text-destructive" />
            <span className="text-sm text-destructive">{error}</span>
            <button
              type="button"
              onClick={() => setError(null)}
              className="ml-auto p-1 hover:bg-destructive/20 rounded"
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        )}

        {/* Loading skeleton */}
        {isLoading && activeSection !== 'roles' && (
          <div className="space-y-4 animate-pulse">
            <div className="h-4 bg-muted rounded w-1/3" />
            <div className="h-32 bg-muted rounded" />
          </div>
        )}

        {/* Roles section */}
        {activeSection === 'roles' && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Toggle roles to assign or remove from this member.
            </p>
            {team.roles.length === 0 ? (
              <p className="text-sm text-muted-foreground italic">
                No roles defined. Add roles in the Info tab.
              </p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {team.roles.map((role) => (
                  <button
                    key={role.id}
                    type="button"
                    onClick={() => void handleToggleRole(role.id)}
                    className={cn(
                      'px-3 py-1.5 text-sm font-medium rounded-full transition-colors',
                      member.roles.includes(role.id)
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted text-muted-foreground hover:bg-primary/20 hover:text-primary'
                    )}
                  >
                    {role.name}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Responsibilities section */}
        {activeSection === 'responsibilities' && !isLoading && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <FileText className="h-4 w-4 text-muted-foreground" />
                <label className="text-sm font-medium">RESPONSIBILITIES.md</label>
              </div>
              <button
                type="button"
                onClick={() => void handleSaveResponsibilities()}
                disabled={!isResponsibilitiesDirty || isSaving}
                className={cn(
                  'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg transition-colors',
                  isResponsibilitiesDirty
                    ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                    : 'bg-muted text-muted-foreground cursor-not-allowed'
                )}
              >
                <Save className="h-3.5 w-3.5" />
                {isSaving ? 'Saving...' : 'Save'}
              </button>
            </div>
            <textarea
              value={responsibilities}
              onChange={(e) => {
                setResponsibilities(e.target.value)
                setIsResponsibilitiesDirty(true)
              }}
              className={cn(
                'w-full h-64 px-3 py-2 text-sm font-mono',
                'bg-muted border border-border rounded-lg',
                'text-foreground placeholder:text-muted-foreground',
                'focus:outline-none focus:ring-2 focus:ring-primary',
                'resize-none'
              )}
              placeholder="# Responsibilities

Describe what this agent is responsible for in this team..."
            />
          </div>
        )}

        {/* Heartbeat section */}
        {activeSection === 'heartbeat' && !isLoading && (
          <div className="space-y-6">
            {/* Schedule */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Clock className="h-4 w-4 text-muted-foreground" />
                  <label className="text-sm font-medium">Schedule (Cron)</label>
                </div>
                <div className="flex items-center gap-2">
                  {heartbeatConfig && (
                    <>
                      <button
                        type="button"
                        onClick={() => void handleToggleHeartbeat()}
                        className={cn(
                          'p-1.5 rounded-lg transition-colors',
                          heartbeatConfig.enabled
                            ? 'bg-green-500/20 text-green-400 hover:bg-green-500/30'
                            : 'bg-muted text-muted-foreground hover:bg-muted/80'
                        )}
                        title={heartbeatConfig.enabled ? 'Disable heartbeat' : 'Enable heartbeat'}
                      >
                        {heartbeatConfig.enabled ? (
                          <Pause className="h-4 w-4" />
                        ) : (
                          <Play className="h-4 w-4" />
                        )}
                      </button>
                      <button
                        type="button"
                        onClick={() => void handleTriggerHeartbeat()}
                        className="p-1.5 rounded-lg bg-muted text-muted-foreground hover:bg-muted/80 transition-colors"
                        title="Trigger heartbeat now"
                      >
                        <Play className="h-4 w-4" />
                      </button>
                    </>
                  )}
                </div>
              </div>

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

              {/* Presets */}
              <div className="flex flex-wrap gap-1">
                {heartbeatService.SCHEDULE_PRESETS.map((preset) => (
                  <button
                    key={preset.value}
                    type="button"
                    onClick={() => setSchedule(preset.value)}
                    className={cn(
                      'px-2 py-1 text-xs rounded transition-colors',
                      schedule === preset.value
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted text-muted-foreground hover:bg-muted/80'
                    )}
                  >
                    {preset.label}
                  </button>
                ))}
              </div>

              <button
                type="button"
                onClick={() => void handleSaveSchedule()}
                disabled={isSaving}
                className={cn(
                  'w-full px-4 py-2 text-sm font-medium rounded-lg',
                  'bg-primary text-primary-foreground hover:bg-primary/90',
                  'transition-colors disabled:opacity-50'
                )}
              >
                {isSaving ? 'Saving...' : 'Save Schedule'}
              </button>

              {/* Last execution info */}
              {heartbeatConfig?.lastExecution && (
                <div className="p-3 bg-muted rounded-lg">
                  <p className="text-xs font-medium mb-1">Last Execution</p>
                  <p className="text-xs text-muted-foreground">
                    Status: {heartbeatConfig.lastExecution.status}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Started: {heartbeatConfig.lastExecution.startedAt}
                  </p>
                  {heartbeatConfig.lastExecution.error && (
                    <p className="text-xs text-destructive mt-1">
                      Error: {heartbeatConfig.lastExecution.error}
                    </p>
                  )}
                </div>
              )}
            </div>

            {/* Heartbeat instructions */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <FileText className="h-4 w-4 text-muted-foreground" />
                  <label className="text-sm font-medium">HEARTBEAT.md</label>
                </div>
                <button
                  type="button"
                  onClick={() => void handleSaveInstructions()}
                  disabled={!isInstructionsDirty || isSaving}
                  className={cn(
                    'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg transition-colors',
                    isInstructionsDirty
                      ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                      : 'bg-muted text-muted-foreground cursor-not-allowed'
                  )}
                >
                  <Save className="h-3.5 w-3.5" />
                  {isSaving ? 'Saving...' : 'Save'}
                </button>
              </div>

              <textarea
                value={heartbeatInstructions}
                onChange={(e) => {
                  setHeartbeatInstructions(e.target.value)
                  setIsInstructionsDirty(true)
                }}
                className={cn(
                  'w-full h-48 px-3 py-2 text-sm font-mono',
                  'bg-muted border border-border rounded-lg',
                  'text-foreground placeholder:text-muted-foreground',
                  'focus:outline-none focus:ring-2 focus:ring-primary',
                  'resize-none'
                )}
                placeholder="# Heartbeat Task

Describe what this agent should do on each heartbeat..."
              />
            </div>
          </div>
        )}
      </div>

      {/* Footer with remove button */}
      <div className="flex-shrink-0 px-4 py-3 border-t border-border">
        <button
          type="button"
          onClick={() => void handleRemove()}
          className={cn(
            'flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg w-full justify-center',
            'text-destructive hover:bg-destructive/10 transition-colors'
          )}
        >
          <Trash2 className="h-4 w-4" />
          Remove from Team
        </button>
      </div>
    </div>
  )
}
