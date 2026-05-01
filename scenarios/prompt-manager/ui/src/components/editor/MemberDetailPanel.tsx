/**
 * MemberDetailPanel - Right panel for editing selected team member.
 *
 * Features:
 * - Header with avatar, name, status dropdown
 * - Overview tab with relationships, roles, schedule, heartbeat instructions, responsibilities
 * - Pipeline tab with prompt pipeline editor
 * - Remove member button
 */

import { useState, useEffect, useCallback, useMemo } from 'react'
import { X, Trash2, Save, FileText, AlertCircle, ArrowUpRight, ArrowDownRight, PanelRightClose, Clock, ExternalLink } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'
import type { TeamDetails, TeamMember, UpdateMemberRequest } from '@/types/team'
import type { AgentAppearance } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { CollapsibleSection } from '@/components/shared/CollapsibleSection'
import * as heartbeatService from '@/services/heartbeatService'
import { toast } from '@/hooks/use-toast'
import type { HeartbeatConfig } from '@/services/heartbeatService'
import { MemberScheduleSection } from './MemberScheduleSection'
import { MemberPromptPipelineSection } from './MemberPromptPipelineSection'
import { MemberPromptPreview } from './MemberPromptPreview'
import { useRunningAgentsStore } from '@/stores/runningAgentsStore'
import { ToastAction } from '@/components/ui/toast'
import { runDetailPath } from '@/app/routes/route-paths'

// ============================================================================
// Types
// ============================================================================

export type MemberDetailSection = 'overview' | 'responsibilities' | 'heartbeat' | 'pipeline' | 'prompt'

type ActiveTab = 'overview' | 'pipeline' | 'prompt'

interface MemberDetailPanelProps {
  team: TeamDetails
  member: TeamMember
  appearance?: AgentAppearance
  manager?: TeamMember | null
  directReports?: TeamMember[]
  /** Which section tab to navigate to */
  initialSection?: MemberDetailSection
  /** Nonce that changes on each navigation request to guarantee the effect fires */
  initialSectionNonce?: number
  onUpdateMember: (agentId: string, request: UpdateMemberRequest) => Promise<TeamMember>
  onRemoveMember: (agentId: string) => Promise<void>
  onClose: () => void
  onCollapse?: () => void
  onNavigateToAgentFiles?: (agentId: string, filePath?: string) => void
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

function formatRelativePastTime(date: Date) {
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

// ============================================================================
// Component
// ============================================================================

export function MemberDetailPanel({
  team,
  member,
  appearance,
  manager = null,
  directReports = [],
  initialSection,
  initialSectionNonce,
  onUpdateMember,
  onRemoveMember,
  onClose,
  onCollapse,
  onNavigateToAgentFiles,
  className,
}: MemberDetailPanelProps) {
  const navigate = useNavigate()
  // Running agent state from shared store
  const runningAgent = useRunningAgentsStore((s) => s.agentMap.get(member.agentId))

  // Local state — 3 tabs: overview, pipeline, prompt
  const [activeSection, setActiveSection] = useState<ActiveTab>(
    initialSection === 'pipeline' ? 'pipeline'
      : initialSection === 'prompt' ? 'prompt'
        : 'overview'
  )

  // Sync when a navigation request arrives (e.g. clicking a heartbeat in Info tab).
  // The nonce ensures the effect fires even for repeated navigations to the same section.
  useEffect(() => {
    if (!initialSection) return
    const tab: ActiveTab = (initialSection === 'responsibilities' || initialSection === 'heartbeat')
      ? 'overview' : initialSection
    setActiveSection(tab)
    if (initialSection === 'responsibilities' || initialSection === 'heartbeat') {
      setTimeout(() => {
        document.getElementById(`section-${initialSection}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      }, 100)
    }
  }, [initialSection, initialSectionNonce])

  const [responsibilities, setResponsibilities] = useState('')
  const [heartbeatInstructions, setHeartbeatInstructions] = useState('')
  const [heartbeatConfig, setHeartbeatConfig] = useState<HeartbeatConfig | null>(null)
  const [schedule, setSchedule] = useState('0 */6 * * *')
  const [recentHeartbeatLogs, setRecentHeartbeatLogs] = useState<heartbeatService.LogEntry[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isLoadingRecentHeartbeats, setIsLoadingRecentHeartbeats] = useState(true)
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
      setIsLoadingRecentHeartbeats(true)
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
        setSchedule(config?.schedule ?? '0 */6 * * *')
        setIsResponsibilitiesDirty(false)
        setIsInstructionsDirty(false)

        try {
          const logs = await heartbeatService.listLogs(team.id, member.agentId)
          setRecentHeartbeatLogs(logs)
        } catch (logErr) {
          console.warn('Failed to load member heartbeat logs:', logErr)
          setRecentHeartbeatLogs([])
        } finally {
          setIsLoadingRecentHeartbeats(false)
        }
      } catch (err) {
        console.warn('Failed to load member data:', err)
        setError('Failed to load member data')
        setIsLoadingRecentHeartbeats(false)
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
  const handleSaveSchedule = useCallback(async (nextSchedule?: string): Promise<boolean> => {
    const scheduleValue = (nextSchedule ?? schedule).trim()
    if (!scheduleValue) return false
    setIsSaving(true)
    try {
      let updated: HeartbeatConfig
      if (heartbeatConfig) {
        updated = await heartbeatService.updateHeartbeat(team.id, member.agentId, { schedule: scheduleValue })
      } else {
        updated = await heartbeatService.createHeartbeat(team.id, member.agentId, {
          schedule: scheduleValue,
          enabled: false,
        })
      }
      setHeartbeatConfig(updated)
      setSchedule(updated.schedule || scheduleValue)
      return true
    } catch (err) {
      console.error('Failed to save schedule:', err)
      setError('Failed to save schedule')
      return false
    } finally {
      setIsSaving(false)
    }
  }, [team.id, member.agentId, schedule, heartbeatConfig])

  const handleSetHeartbeatEnabled = useCallback(async (enabled: boolean) => {
    setIsSaving(true)
    try {
      let updated: HeartbeatConfig
      if (heartbeatConfig) {
        updated = await heartbeatService.updateHeartbeat(team.id, member.agentId, { enabled })
      } else {
        updated = await heartbeatService.createHeartbeat(team.id, member.agentId, {
          schedule,
          enabled,
        })
      }
      setHeartbeatConfig(updated)
      setSchedule(updated.schedule)
    } catch (err) {
      console.error('Failed to update heartbeat state:', err)
      setError('Failed to update heartbeat state. The schedule may be invalid.')
    } finally {
      setIsSaving(false)
    }
  }, [team.id, member.agentId, heartbeatConfig, schedule])

  // Trigger heartbeat now
  const handleTriggerHeartbeat = useCallback(async () => {
    try {
      const result = await heartbeatService.triggerHeartbeat(team.id, member.agentId)
      const runId = result.runId
      toast({
        title: 'Heartbeat triggered',
        variant: 'success',
        action: runId ? (
          <ToastAction
            altText="Open run"
            onClick={() => navigate(runDetailPath(runId))}
          >
            Open Run
          </ToastAction>
        ) : undefined,
      })
      const updated = await heartbeatService.getHeartbeat(team.id, member.agentId)
      if (updated) setHeartbeatConfig(updated)
      const logs = await heartbeatService.listLogs(team.id, member.agentId)
      setRecentHeartbeatLogs(logs)
    } catch (err) {
      console.error('Failed to trigger heartbeat:', err)
      toast({ title: 'Failed to trigger heartbeat', variant: 'destructive' })
    }
  }, [team.id, member.agentId])

  const recentHeartbeats = useMemo(() => {
    const entries: {
      startedAt: Date
      status: string
      runId?: string
      error?: string
      source: 'execution' | 'log'
    }[] = []

    const lastExecution = heartbeatConfig?.lastExecution
    if (lastExecution?.startedAt) {
      const startedAt = new Date(lastExecution.startedAt)
      if (!Number.isNaN(startedAt.getTime())) {
        entries.push({
          startedAt,
          status: lastExecution.status,
          runId: lastExecution.runId,
          error: lastExecution.error,
          source: 'execution',
        })
      }
    }

    for (const log of recentHeartbeatLogs) {
      const startedAt = new Date(log.timestamp)
      if (Number.isNaN(startedAt.getTime())) continue
      entries.push({
        startedAt,
        status: (log.status ?? 'completed').toLowerCase(),
        source: 'log',
      })
    }

    const deduped = new Map<string, (typeof entries)[number]>()
    for (const entry of entries) {
      const key = `${entry.startedAt.toISOString()}-${entry.status}`
      const existing = deduped.get(key)
      if (!existing || (existing.source === 'log' && entry.source === 'execution')) {
        deduped.set(key, entry)
      }
    }

    return Array.from(deduped.values())
      .sort((a, b) => b.startedAt.getTime() - a.startedAt.getTime())
      .slice(0, 5)
  }, [heartbeatConfig?.lastExecution, recentHeartbeatLogs])

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

  const saveButton = (onClick: () => void, isDirty: boolean) => (
    <button
      type="button"
      onClick={onClick}
      disabled={!isDirty || isSaving}
      className={cn(
        'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg transition-colors',
        isDirty
          ? 'bg-primary text-primary-foreground hover:bg-primary/90'
          : 'bg-muted text-muted-foreground cursor-not-allowed'
      )}
    >
      <Save className="h-3.5 w-3.5" />
      {isSaving ? 'Saving...' : 'Save'}
    </button>
  )

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

          {onCollapse && (
            <button
              type="button"
              onClick={onCollapse}
              className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0 ml-auto"
              title="Collapse panel"
              aria-label="Collapse panel"
            >
              <PanelRightClose className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>

      {/* Section tabs — 2 tabs */}
      <div className="flex-shrink-0 flex border-b border-border">
        <button
          type="button"
          onClick={() => setActiveSection('overview')}
          className={cn(
            'flex-1 px-4 py-2 text-sm font-medium transition-colors',
            activeSection === 'overview'
              ? 'text-primary border-b-2 border-primary'
              : 'text-muted-foreground hover:text-foreground'
          )}
        >
          Overview
        </button>
        <button
          type="button"
          onClick={() => setActiveSection('pipeline')}
          className={cn(
            'flex-1 px-4 py-2 text-sm font-medium transition-colors',
            activeSection === 'pipeline'
              ? 'text-primary border-b-2 border-primary'
              : 'text-muted-foreground hover:text-foreground'
          )}
        >
          Pipeline
        </button>
        <button
          type="button"
          onClick={() => setActiveSection('prompt')}
          className={cn(
            'flex-1 px-4 py-2 text-sm font-medium transition-colors',
            activeSection === 'prompt'
              ? 'text-primary border-b-2 border-primary'
              : 'text-muted-foreground hover:text-foreground'
          )}
        >
          Prompt
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

        {/* Overview section */}
        {activeSection === 'overview' && (
          <div className="space-y-4">
            {/* Relationships */}
            <CollapsibleSection title="Relationships" defaultExpanded>
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
            </CollapsibleSection>

            {/* Roles */}
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

            {/* Schedule */}
            <MemberScheduleSection
              schedule={schedule}
              heartbeatConfig={heartbeatConfig}
              isSaving={isSaving}
              onSaveSchedule={(nextSchedule) => handleSaveSchedule(nextSchedule)}
              onTriggerHeartbeat={() => void handleTriggerHeartbeat()}
              onSetHeartbeatEnabled={(enabled) => void handleSetHeartbeatEnabled(enabled)}
              isRunning={!!runningAgent}
              runDuration={runningAgent?.duration}
              runningRunId={runningAgent?.runId}
              onOpenRun={(runId) => navigate(runDetailPath(runId))}
            />

            <section>
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-sm font-medium text-foreground">Recent Heartbeats</h3>
                <span className="text-xs text-muted-foreground">Last 5</span>
              </div>
              {isLoadingRecentHeartbeats ? (
                <p className="text-sm text-muted-foreground">Loading recent heartbeats...</p>
              ) : recentHeartbeats.length === 0 ? (
                <div className="flex items-start gap-3 p-3 bg-muted rounded-lg">
                  <Clock className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                  <div>
                    <p className="text-sm text-muted-foreground">No recent heartbeat executions.</p>
                    <p className="text-xs text-muted-foreground/70 mt-0.5">
                      Heartbeat history will appear here after this member runs their first heartbeat.
                    </p>
                  </div>
                </div>
              ) : (
                <ul className="space-y-2">
                  {recentHeartbeats.map((entry) => (
                    <li key={`${entry.startedAt.toISOString()}-${entry.status}`}>
                      <div className="flex items-center gap-1">
                        <div className="flex-1 flex items-start justify-between gap-4 px-3 py-2 bg-muted rounded-lg text-left">
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
                              <p className="text-sm font-medium text-foreground truncate">{member.displayName}</p>
                              <p className="text-xs text-muted-foreground mt-0.5 capitalize">{entry.status}</p>
                            </div>
                          </div>
                          <div className="text-right flex-shrink-0">
                            <p className="text-sm text-foreground">{entry.startedAt.toLocaleString()}</p>
                            <p className="text-xs text-muted-foreground">{formatRelativePastTime(entry.startedAt)}</p>
                          </div>
                        </div>
                        {entry.runId && (
                          <button
                            type="button"
                            className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
                            title="Open full run view"
                            onClick={() => navigate(runDetailPath(entry.runId ?? ''))}
                          >
                            <ExternalLink className="h-3.5 w-3.5" />
                          </button>
                        )}
                      </div>
                      {(entry.status === 'failed' || entry.status === 'cancelled') && entry.error && (
                        <p className="text-xs text-red-400 mt-1 px-3 truncate" title={entry.error}>
                          {entry.error}
                        </p>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </section>

            {/* Heartbeat Instructions */}
            <CollapsibleSection
              title="Heartbeat Instructions"
              id="section-heartbeat"
              isDirty={isInstructionsDirty}
              headerRight={saveButton(() => void handleSaveInstructions(), isInstructionsDirty)}
            >
              {isLoading ? (
                <div className="space-y-2 animate-pulse">
                  <div className="h-4 bg-muted rounded w-1/3" />
                  <div className="h-32 bg-muted rounded" />
                </div>
              ) : (
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <FileText className="h-4 w-4 text-muted-foreground" />
                    <label className="text-sm font-medium">HEARTBEAT.md</label>
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
              )}
            </CollapsibleSection>

            {/* Responsibilities */}
            <CollapsibleSection
              title="Responsibilities"
              id="section-responsibilities"
              isDirty={isResponsibilitiesDirty}
              headerRight={saveButton(() => void handleSaveResponsibilities(), isResponsibilitiesDirty)}
            >
              {isLoading ? (
                <div className="space-y-2 animate-pulse">
                  <div className="h-4 bg-muted rounded w-1/3" />
                  <div className="h-32 bg-muted rounded" />
                </div>
              ) : (
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <FileText className="h-4 w-4 text-muted-foreground" />
                    <label className="text-sm font-medium">RESPONSIBILITIES.md</label>
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
            </CollapsibleSection>
          </div>
        )}

        {/* Pipeline section */}
        {activeSection === 'pipeline' && (
          <MemberPromptPipelineSection
            teamId={team.id}
            memberId={member.agentId}
            onNavigateToTab={(section) => {
              setActiveSection('overview')
              setTimeout(() => {
                document.getElementById(`section-${section}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
              }, 50)
            }}
            onNavigateToAgentFiles={onNavigateToAgentFiles ? (filePath) => onNavigateToAgentFiles(member.agentId, filePath) : undefined}
          />
        )}

        {/* Prompt preview section */}
        {activeSection === 'prompt' && (
          <MemberPromptPreview
            teamId={team.id}
            agentId={member.agentId}
            onNavigateToFile={onNavigateToAgentFiles ? (filePath) => onNavigateToAgentFiles(member.agentId, filePath) : undefined}
          />
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
