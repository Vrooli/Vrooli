/**
 * TeamEditorPanel - Full-panel editor for teams with org chart visualization.
 *
 * Features:
 * - Header with close button, editable name, member count badge
 * - Editable mission statement
 * - Split-panel Members tab: Org chart (left) + Member detail (right)
 * - Toggle between Graph and Code views in Members tab
 * - Tabs: Info, Members, Files
 * - Keyboard shortcuts: Escape (close), Ctrl+S (save)
 */

import { useState, useCallback, useEffect, useMemo } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { Menu, X, Users, Info, ChevronDown, ChevronUp, GripVertical, Folder, Power, MoreHorizontal, Trash2, PanelRightOpen, Eye } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamDetails, UpdateTeamRequest, TeamRole, TeamMember, AddMemberRequest, UpdateMemberRequest } from '@/types/team'
import type { Agent } from '@/types/agent'
import type { HighlightRequest } from '@/lib/highlight'
import { InlineEditableText } from '../shared/InlineEditableText'
import { ExpandableDescription } from '../shared/ExpandableDescription'
import { selectors } from '@/constants/selectors'
import { useResizableSplitPanel } from '@/hooks/useResizableSplitPanel'
import { useIsCompactHeader, useIsMobile } from '@/hooks/useMediaQuery'
import { useTeamEditorStore } from '@/hooks/useTeamEditorStore'
import * as orgChartService from '@/services/orgChartService'

import { ToolbarDropdown, DropdownItem } from './ToolbarDropdown'
import { OrgChartPanel } from './OrgChartPanel'
import { MemberDetailPanel } from './MemberDetailPanel'
import type { MemberDetailSection } from './MemberDetailPanel'
import { TeamCodeView } from './TeamCodeView'
import { MemberPickerModal } from './teamTabs/MembersTab'
import { TeamInfoTab, TeamFilesTab, TeamPromptMatrixTab } from './teamTabs'

// ============================================================================
// Types
// ============================================================================

export type MembersViewMode = 'graph' | 'code'

const MEMBERS_VIEW_STORAGE_KEY = 'pm.teamMembersViewMode'

interface TeamEditorPanelProps {
  /** Current team being edited */
  team: TeamDetails | null
  /** All available agents for the member picker */
  allAgents?: Agent[]
  /** Callback when team data changes */
  onUpdate: (updates: UpdateTeamRequest) => Promise<void>
  /** Callback to add a member */
  onAddMember: (request: AddMemberRequest) => Promise<TeamMember>
  /** Callback to update a member */
  onUpdateMember: (agentId: string, request: UpdateMemberRequest) => Promise<TeamMember>
  /** Callback to remove a member */
  onRemoveMember: (agentId: string) => Promise<void>
  /** Callback to set roles */
  onSetRoles: (roles: TeamRole[]) => Promise<TeamRole[]>
  /** Callback to close the editor */
  onClose: () => void
  /** Optional callback to open sidebar (used on mobile) */
  onOpenSidebar?: () => void
  /** Callback to trigger team deletion */
  onDelete: () => void
  /** Whether a delete operation is in progress */
  isDeleting?: boolean
  /** Navigate to an agent's files in the Agent Editor */
  onNavigateToAgentFiles?: (agentId: string, filePath?: string) => void
  /** Cross-reference highlight request */
  highlightRequest?: HighlightRequest | null
  /** Called after highlight is applied (clears URL params) */
  onHighlightHandled?: () => void
  /** Additional class names */
  className?: string
}

/**
 * Team editor panel component.
 */
export function TeamEditorPanel({
  team,
  allAgents = [],
  onUpdate,
  onAddMember,
  onUpdateMember,
  onRemoveMember,
  onSetRoles,
  onClose,
  onOpenSidebar,
  onDelete,
  isDeleting = false,
  onNavigateToAgentFiles,
  highlightRequest,
  onHighlightHandled,
  className,
}: TeamEditorPanelProps) {
  // Active tab state
  const [activeTab, setActiveTab] = useState('info')

  const isMobile = useIsMobile()
  const isCompactHeader = useIsCompactHeader()
  const isMobileSidebarToggle = Boolean(onOpenSidebar)

  // Mission expanded state
  const [isMissionExpanded, setIsMissionExpanded] = useState(false)

  // Member picker modal state
  const [showMemberPicker, setShowMemberPicker] = useState(false)

  // Navigation request for MemberDetailPanel (set via Info tab heartbeat click).
  // The nonce ensures repeated clicks always trigger the section switch.
  const [memberSectionNav, setMemberSectionNav] = useState<{ section: MemberDetailSection; nonce: number } | null>(null)

  // Members view mode (graph vs code)
  const [membersViewMode, setMembersViewMode] = useState<MembersViewMode>(() => {
    if (typeof window === 'undefined') return 'graph'
    const stored = localStorage.getItem(MEMBERS_VIEW_STORAGE_KEY)
    return stored === 'code' ? 'code' : 'graph'
  })

  // Persist view mode preference
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(MEMBERS_VIEW_STORAGE_KEY, membersViewMode)
    }
  }, [membersViewMode])

  // Team editor store
  const selectedMemberId = useTeamEditorStore((state) => state.selectedMemberId)
  const setSelectedMemberId = useTeamEditorStore((state) => state.setSelectedMemberId)
  const edges = useTeamEditorStore((state) => state.edges)
  const setEdges = useTeamEditorStore((state) => state.setEdges)
  const updateEdge = useTeamEditorStore((state) => state.updateEdge)
  const reset = useTeamEditorStore((state) => state.reset)
  const memberCount = team?.memberCount ?? team?.members.length ?? 0

  // Split panel for members tab
  const { width, isResizing, isCollapsed, containerRef, handleResizeStart, expand, collapse } = useResizableSplitPanel({
    defaultWidth: 400,
    minWidth: 280,
    maxWidthRatio: 0.6,
    anchor: 'right',
    storageKey: 'pm.teamEditorSplitWidth',
    snapCloseThreshold: 200,
  })

  // Load org chart edges when team changes
  useEffect(() => {
    if (!team) {
      reset()
      return
    }

    const loadEdges = async () => {
      try {
        const orgEdges = await orgChartService.getEdges(team.id)
        setEdges(orgEdges)
      } catch (error) {
        console.error('[TeamEditorPanel] Failed to load org chart edges:', error)
      }
    }
    void loadEdges()
    // eslint-disable-next-line react-hooks/exhaustive-deps -- Only re-run when team ID changes
  }, [team?.id, setEdges, reset])

  // Get selected member
  const selectedMember = useMemo(() => {
    if (!team || !selectedMemberId) return null
    return team.members.find((m) => m.agentId === selectedMemberId) ?? null
  }, [team, selectedMemberId])

  const selectedMemberManagerId = useMemo(() => {
    if (!selectedMemberId) return null
    return edges.find((edge) => edge.reportId === selectedMemberId)?.managerId ?? null
  }, [edges, selectedMemberId])

  const selectedMemberManager = useMemo(() => {
    if (!team || !selectedMemberManagerId) return null
    return team.members.find((member) => member.agentId === selectedMemberManagerId) ?? null
  }, [team, selectedMemberManagerId])

  const selectedMemberReports = useMemo(() => {
    if (!team || !selectedMemberId) return []
    const reportIds = new Set(
      edges.filter((edge) => edge.managerId === selectedMemberId).map((edge) => edge.reportId)
    )
    return team.members.filter((member) => reportIds.has(member.agentId))
  }, [team, edges, selectedMemberId])

  // Get agent appearance for selected member
  const selectedMemberAppearance = useMemo(() => {
    if (!selectedMemberId) return undefined
    const agent = allAgents.find((a) => a.id === selectedMemberId)
    return agent?.appearance ?? undefined
  }, [selectedMemberId, allAgents])

  const showDetailOnly = Boolean(isMobile && selectedMember)
  const detailPanelWidth: number | string = showDetailOnly ? '100%' : width

  useEffect(() => {
    if (showDetailOnly && activeTab !== 'members') {
      setActiveTab('members')
    }
  }, [showDetailOnly, activeTab])

  // Auto-switch to files tab when a highlight request targets a file
  useEffect(() => {
    if (highlightRequest?.file && activeTab !== 'files') {
      setActiveTab('files')
    }
  }, [highlightRequest, activeTab])

  const handleSwitchToGraph = useCallback(() => {
    setMembersViewMode('graph')
  }, [])

  const handleSwitchToCode = useCallback(() => {
    setMembersViewMode('code')
  }, [])

  // Available agents (not already members)
  const availableAgents = useMemo(() => {
    if (!team) return allAgents
    const memberIds = new Set(team.members.map((m) => m.agentId))
    return allAgents.filter((a) => !memberIds.has(a.id))
  }, [team, allAgents])

  // Handle name change
  const handleNameChange = useCallback(
    async (newName: string) => {
      if (team && newName !== team.displayName) {
        await onUpdate({ displayName: newName })
      }
    },
    [team, onUpdate]
  )

  // Handle mission change
  const handleMissionChange = useCallback(
    async (newMission: string) => {
      if (team) {
        await onUpdate({ mission: newMission })
      }
    },
    [team, onUpdate]
  )

  const handleToggleTeam = useCallback(async () => {
    if (!team) return
    await onUpdate({ enabled: !team.enabled })
  }, [team, onUpdate])

  // Handle edge update
  const handleEdgeUpdate = useCallback(
    async (agentId: string, managerId: string | null) => {
      if (!team) return
      if (managerId === agentId) return

      const previousEdges = edges

      // Update local state first
      updateEdge(agentId, managerId)

      try {
        // Then sync to API
        await orgChartService.updateEdge(team.id, agentId, { managerId })
      } catch (error) {
        console.error('[TeamEditorPanel] Failed to update org chart edge:', error)
        setEdges(previousEdges)
      }
    },
    [team, edges, updateEdge, setEdges]
  )

  // Navigate to a member's heartbeat tab (from Info tab upcoming heartbeats)
  const handleNavigateToMemberHeartbeat = useCallback(
    (agentId: string) => {
      setMemberSectionNav((prev) => ({ section: 'heartbeat', nonce: (prev?.nonce ?? 0) + 1 }))
      setSelectedMemberId(agentId)
      setActiveTab('members')
    },
    [setSelectedMemberId]
  )

  // Navigate to a member in the Members tab (from role member chips)
  const handleNavigateToMember = useCallback(
    (agentId: string) => {
      setSelectedMemberId(agentId)
      setActiveTab('members')
    },
    [setSelectedMemberId]
  )

  // Handle add member
  const handleAddMember = useCallback(
    async (agentId: string) => {
      await onAddMember({ agentId, roles: [] })
      setShowMemberPicker(false)
    },
    [onAddMember]
  )

  // Handle remove member
  const handleRemoveMember = useCallback(
    async (agentId: string) => {
      await onRemoveMember(agentId)
      if (selectedMemberId === agentId) {
        setSelectedMemberId(null)
      }
    },
    [onRemoveMember, selectedMemberId, setSelectedMemberId]
  )

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Escape: deselect member or close editor
      if (e.key === 'Escape') {
        if (selectedMemberId) {
          setSelectedMemberId(null)
        } else {
          onClose()
        }
        return
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [selectedMemberId, setSelectedMemberId, onClose])

  // Empty state when no team selected
  if (!team) {
    return (
      <div className={cn('h-full flex items-center justify-center', className)}>
        <div className="text-center">
          <Users className="h-16 w-16 mx-auto mb-4 text-muted-foreground/50" />
          <h3 className="text-lg font-medium text-muted-foreground">No Team Selected</h3>
          <p className="text-sm text-muted-foreground/70 max-w-xs mx-auto mt-2">
            Select a team from the list to view and edit its details.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className={cn('h-full flex flex-col bg-card/50', className)}>
      {/* Header */}
      {!showDetailOnly && (
        <div
          className="flex-shrink-0 px-4 py-3 border-b border-border space-y-2"
          data-testid={selectors.teamEditor.header}
        >
          {/* Row 1: Close, Icon, Name */}
          <div className="flex items-center gap-2 min-w-0">
            {/* Close button */}
            <button
              type="button"
              onClick={onOpenSidebar ?? onClose}
              className="h-9 w-9 flex items-center justify-center rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
              aria-label={isMobileSidebarToggle ? 'Open sidebar' : 'Close editor'}
              title={isMobileSidebarToggle ? 'Open sidebar' : 'Close (Esc)'}
            >
              {isMobileSidebarToggle ? <Menu className="h-5 w-5" /> : <X className="h-5 w-5" />}
            </button>

            {/* Team icon */}
            <div className="flex-shrink-0">
              <div className="w-10 h-10 rounded-full flex items-center justify-center bg-primary/20">
                <Users className="h-5 w-5 text-primary" />
              </div>
            </div>

            {/* Editable name */}
            <div className="flex-1 min-w-0">
              <InlineEditableText
                value={team.displayName}
                onChange={(value) => void handleNameChange(value)}
                placeholder="Team name"
                className="text-lg font-semibold"
              />
            </div>

            <div className="flex items-center gap-1.5 flex-shrink-0">
              <button
                type="button"
                onClick={() => void handleToggleTeam()}
                className={cn(
                  'flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-full border transition-colors max-[389px]:hidden',
                  team.enabled
                    ? 'bg-emerald-500/15 text-emerald-500 border-emerald-500/30 hover:bg-emerald-500/25'
                    : 'bg-muted text-muted-foreground border-border hover:bg-muted/80'
                )}
                title={team.enabled ? 'Turn team off' : 'Turn team on'}
                aria-label={team.enabled ? 'Turn team off' : 'Turn team on'}
                aria-pressed={team.enabled}
              >
                <Power className="h-3.5 w-3.5" />
                {team.enabled ? 'Team On' : 'Team Off'}
              </button>
              <ToolbarDropdown
                icon={<MoreHorizontal className="h-4 w-4" />}
                label="More actions"
                showChevron={false}
                align="right"
                className="h-9 w-9 p-0 rounded-lg"
              >
                {isCompactHeader && (
                  <DropdownItem
                    onClick={() => void handleToggleTeam()}
                    icon={<Power className="h-4 w-4" />}
                    label={team.enabled ? 'Turn team off' : 'Turn team on'}
                  />
                )}
                <DropdownItem
                  onClick={onDelete}
                  disabled={isDeleting}
                  icon={<Trash2 className="h-4 w-4 text-destructive" />}
                  label={isDeleting ? 'Deleting...' : 'Delete team'}
                />
              </ToolbarDropdown>
            </div>
          </div>

          {/* Row 2: Expandable mission */}
          <div className="flex items-start gap-2">
            <button
              type="button"
              onClick={() => setIsMissionExpanded(!isMissionExpanded)}
              className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            >
              {isMissionExpanded ? (
                <ChevronUp className="h-4 w-4" />
              ) : (
                <ChevronDown className="h-4 w-4" />
              )}
            </button>
            {isMissionExpanded ? (
              <ExpandableDescription
                value={team.mission ?? ''}
                onChange={(value) => void handleMissionChange(value)}
                placeholder="Add a mission statement..."
                className="flex-1"
              />
            ) : (
              <p className="flex-1 text-sm text-muted-foreground truncate">
                {team.mission || 'No mission statement'}
              </p>
            )}
          </div>
        </div>
      )}

      {/* Tabs */}
      <Tabs.Root
        value={activeTab}
        onValueChange={setActiveTab}
        className="flex-1 flex flex-col min-h-0 overflow-hidden"
      >
        {/* Tab List */}
        {!showDetailOnly && (
          <Tabs.List className="flex-shrink-0 flex border-b border-border px-4">
            <TabTrigger value="info" icon={<Info className="h-4 w-4" />} label="Info" />
            <TabTrigger
              value="members"
              icon={<Users className="h-4 w-4" />}
              label={`Members (${memberCount})`}
            />
            <TabTrigger value="files" icon={<Folder className="h-4 w-4" />} label="Files" />
            <TabTrigger value="prompts" icon={<Eye className="h-4 w-4" />} label="Prompts" />
          </Tabs.List>
        )}

        {/* Tab Content */}
        <div className="flex-1 min-h-0 flex flex-col">
          {/* Members tab - Split panel layout with view mode toggle */}
          <Tabs.Content
            value="members"
            className="flex-1 min-h-0 flex flex-col data-[state=inactive]:hidden"
          >
            {/* Content area */}
            <div className="flex-1 min-h-0">
              {membersViewMode === 'graph' ? (
                <div
                  ref={containerRef}
                  className={cn('h-full flex relative', isResizing && 'select-none')}
                >
                  {/* Left panel: Org Chart */}
                  {!showDetailOnly && (
                    <div className="flex-1 min-w-0 h-full">
                      <OrgChartPanel
                        team={team}
                        edges={edges}
                        allAgents={allAgents}
                        selectedMemberId={selectedMemberId}
                      onSelectMember={setSelectedMemberId}
                      onEdgeUpdate={(agentId, managerId) => void handleEdgeUpdate(agentId, managerId)}
                      onAddMember={() => setShowMemberPicker(true)}
                      onSwitchToCode={handleSwitchToCode}
                      className="h-full"
                    />
                  </div>
                )}

                  {/* Expand button when panel is collapsed */}
                  {isCollapsed && selectedMember && !showDetailOnly && (
                    <button
                      type="button"
                      onClick={expand}
                      className="absolute top-2 right-2 z-10 p-1.5 rounded-lg bg-card border border-border text-muted-foreground hover:text-foreground hover:bg-muted transition-colors shadow-sm"
                      title="Show member details"
                      aria-label="Show member details"
                    >
                      <PanelRightOpen className="h-4 w-4" />
                    </button>
                  )}

                  {/* Resize handle + Right panel: Member detail (conditional) */}
                  {selectedMember && !showDetailOnly && !isCollapsed && (
                    <>
                      {/* Resize handle */}
                      <div
                        onMouseDown={handleResizeStart}
                        className={cn(
                          'flex-shrink-0 w-1.5 cursor-col-resize relative group',
                          'hover:bg-primary/30 transition-colors',
                          isResizing && 'bg-primary/50'
                        )}
                      >
                        <div className="absolute inset-y-0 left-1/2 -translate-x-1/2 w-4 flex items-center justify-center">
                          <GripVertical className="h-4 w-4 text-muted-foreground/50 opacity-0 group-hover:opacity-100 transition-opacity" />
                        </div>
                      </div>
                    </>
                  )}

                  {/* Right panel: Member detail */}
                  {selectedMember && !isCollapsed && (
                    <div
                      style={{ width: detailPanelWidth }}
                      className={cn(
                        'flex-shrink-0 h-full overflow-hidden',
                        !showDetailOnly && 'border-l border-border'
                      )}
                    >
                      <MemberDetailPanel
                        team={team}
                        member={selectedMember}
                        appearance={selectedMemberAppearance}
                        manager={selectedMemberManager}
                        directReports={selectedMemberReports}
                        initialSection={memberSectionNav?.section}
                        initialSectionNonce={memberSectionNav?.nonce}
                        onUpdateMember={onUpdateMember}
                        onRemoveMember={handleRemoveMember}
                        onClose={() => setSelectedMemberId(null)}
                        onCollapse={collapse}
                        onNavigateToAgentFiles={onNavigateToAgentFiles}
                        className="h-full"
                      />
                    </div>
                  )}
                </div>
              ) : (
                <TeamCodeView
                  team={team}
                  edges={edges}
                  readOnly
                  onSwitchToGraph={handleSwitchToGraph}
                  className="h-full"
                />
              )}
            </div>
          </Tabs.Content>

          <Tabs.Content
            value="info"
            className="flex-1 min-h-0 overflow-y-auto p-4 data-[state=inactive]:hidden"
          >
            <TeamInfoTab team={team} onSetRoles={onSetRoles} onUpdate={onUpdate} allAgents={allAgents} onNavigateToMemberHeartbeat={handleNavigateToMemberHeartbeat} onNavigateToMember={handleNavigateToMember} />
          </Tabs.Content>

          <Tabs.Content
            value="files"
            className="flex-1 min-h-0 data-[state=inactive]:hidden"
          >
            <TeamFilesTab
              teamId={team.id}
              highlightRequest={highlightRequest}
              onHighlightHandled={onHighlightHandled}
              className="h-full min-h-0"
            />
          </Tabs.Content>

          <Tabs.Content
            value="prompts"
            className="flex-1 min-h-0 overflow-y-auto p-4 data-[state=inactive]:hidden"
          >
            <TeamPromptMatrixTab
              teamId={team.id}
              onNavigateToMember={handleNavigateToMember}
            />
          </Tabs.Content>
        </div>
      </Tabs.Root>

      {/* Member picker modal */}
      {showMemberPicker && (
        <MemberPickerModal
          availableAgents={availableAgents}
          onSelect={(agentId) => void handleAddMember(agentId)}
          onClose={() => setShowMemberPicker(false)}
        />
      )}
    </div>
  )
}

/**
 * Individual tab trigger button.
 */
interface TabTriggerProps {
  value: string
  icon: React.ReactNode
  label: string
}

function TabTrigger({ value, icon, label }: TabTriggerProps) {
  return (
    <Tabs.Trigger
      value={value}
      className={cn(
        'flex items-center gap-1.5 px-3 py-2 text-sm font-medium',
        'border-b-2 transition-colors',
        'data-[state=active]:border-primary data-[state=active]:text-primary',
        'data-[state=inactive]:border-transparent data-[state=inactive]:text-muted-foreground',
        'hover:text-foreground'
      )}
    >
      {icon}
      {label}
    </Tabs.Trigger>
  )
}
