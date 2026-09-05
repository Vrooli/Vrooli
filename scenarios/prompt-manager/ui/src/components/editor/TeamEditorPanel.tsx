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

import { useState, useCallback, useEffect, useMemo, useRef } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { Menu, X, Users, ChevronDown, ChevronUp, GripVertical, Folder, Power, MoreHorizontal, Trash2, PanelRightOpen, Eye, LayoutDashboard, Activity, UserPlus, LayoutGrid, Code } from 'lucide-react'
import { TabList, TabTrigger } from '../shared/TabTrigger'
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
import { useGlobalKeydown } from '@/hooks/useGlobalKeydown'
import { useTopicsGraph } from '@/hooks/useTopicsGraph'
import * as orgChartService from '@/services/orgChartService'

import { ToolbarDropdown, DropdownItem } from './ToolbarDropdown'
import { OrgChartPanel } from './OrgChartPanel'
import { TopicsGraphPanel } from './TopicsGraphPanel'
import { MemberDetailPanel } from './MemberDetailPanel'
import type { MemberDetailSection } from './MemberDetailPanel'
import { TeamCodeView } from './TeamCodeView'
import { MemberPickerModal } from './teamTabs/MembersTab'
import { TeamDashboardTab, TeamFilesTab, TeamPromptMatrixTab } from './teamTabs'
import { TeamActivityTab } from './teamTabs/TeamActivityTab'
import { formatRelativePastTime } from '@/lib/timeUtils'

// ============================================================================
// Types
// ============================================================================

export type MembersViewMode = 'graph' | 'code'

/**
 * Graph sub-mode: hierarchy (managerId edges) vs topics (topics.json edges).
 * Auto-defaults from team.coordination.reportingMode:
 *   - 'none' → topics
 *   - anything else → hierarchy
 * Operator override (any explicit click) wins and persists per team.
 */
export type GraphMode = 'hierarchy' | 'topics'

export type LayoutDirection = 'TB' | 'LR'

const MEMBERS_VIEW_STORAGE_KEY = 'pm.teamMembersViewMode'
const GRAPH_MODE_STORAGE_PREFIX = 'pm.teamGraphMode.'
const LAYOUT_DIRECTION_STORAGE_PREFIX = 'pm.teamLayoutDirection.'

function readGraphMode(teamId: string): GraphMode | null {
  if (typeof window === 'undefined') return null
  const stored = localStorage.getItem(GRAPH_MODE_STORAGE_PREFIX + teamId)
  return stored === 'hierarchy' || stored === 'topics' ? stored : null
}

function writeGraphMode(teamId: string, mode: GraphMode): void {
  if (typeof window === 'undefined') return
  localStorage.setItem(GRAPH_MODE_STORAGE_PREFIX + teamId, mode)
}

function readLayoutDirection(teamId: string): LayoutDirection {
  if (typeof window === 'undefined') return 'TB'
  const stored = localStorage.getItem(LAYOUT_DIRECTION_STORAGE_PREFIX + teamId)
  return stored === 'LR' ? 'LR' : 'TB'
}

function writeLayoutDirection(teamId: string, dir: LayoutDirection): void {
  if (typeof window === 'undefined') return
  localStorage.setItem(LAYOUT_DIRECTION_STORAGE_PREFIX + teamId, dir)
}

function autoDefaultGraphMode(reportingMode: string | undefined): GraphMode {
  return reportingMode === 'none' ? 'topics' : 'hierarchy'
}

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
  /** Externally-requested tab (e.g. from URL deep-link) */
  initialTab?: string | null
  /** Externally-requested sub-tab (e.g. from URL deep-link) */
  initialSubTab?: string | null
  /** Externally-requested member id (e.g. from URL deep-link) */
  initialMemberId?: string | null
  /** Externally-requested member section (e.g. from URL deep-link) */
  initialMemberSection?: MemberDetailSection | null
  /** Called when the active tab changes */
  onTabChange?: (tab: string) => void
  /** Called when the active sub-tab changes */
  onSubTabChange?: (subTab: string | null) => void
  /** Called when the selected member changes (for URL persistence) */
  onMemberSelect?: (agentId: string | null) => void
  /** Called when the active member-detail section changes (for URL persistence) */
  onMemberSectionChange?: (section: MemberDetailSection | null) => void
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
  onSetRoles: _onSetRoles,
  onClose,
  onOpenSidebar,
  onDelete,
  isDeleting = false,
  onNavigateToAgentFiles,
  initialTab,
  initialSubTab,
  initialMemberId,
  initialMemberSection,
  onTabChange,
  onSubTabChange,
  onMemberSelect,
  onMemberSectionChange,
  highlightRequest,
  onHighlightHandled,
  className,
}: TeamEditorPanelProps) {
  const teamId = team?.id
  const teamReportingMode = team?.coordination.reportingMode

  // Active tab state
  const [activeTab, setActiveTab] = useState('info')

  // Respond to external tab navigation requests (e.g. from URL deep-link)
  useEffect(() => {
    if (initialTab) {
      setActiveTab(initialTab)
    }
  }, [initialTab])

  const handleTabChange = useCallback((value: string) => {
    setActiveTab(value)
    onTabChange?.(value)
  }, [onTabChange])

  // Dashboard health/activity state
  const [teamHealth, setTeamHealth] = useState<'green' | 'yellow' | 'red' | 'gray'>('gray')
  const [lastActiveAt, setLastActiveAt] = useState<string | null>(null)

  const isMobile = useIsMobile()
  const isCompactHeader = useIsCompactHeader()
  const isMobileSidebarToggle = Boolean(onOpenSidebar)

  // Mission expanded state
  const [isMissionExpanded, setIsMissionExpanded] = useState(false)

  // Member picker modal state
  const [showMemberPicker, setShowMemberPicker] = useState(false)

  // Active member-detail section (controlled via URL or programmatic nav).
  const [memberSection, setMemberSection] = useState<MemberDetailSection>(
    initialMemberSection ?? 'overview',
  )

  // Validation sidebar visibility (lifted from TopicsGraphPanel so it can be
  // auto-collapsed when MemberDetailPanel opens).
  const [showValidation, setShowValidation] = useState(true)
  const userValidationPrefRef = useRef(true)
  const handleValidationToggle = useCallback(() => {
    setShowValidation((v) => {
      const next = !v
      userValidationPrefRef.current = next
      return next
    })
  }, [])

  // Topics graph (single source of truth — also feeds the Members-tab pill).
  const { graph: topicsGraph } = useTopicsGraph(teamId)
  const validationCount = useMemo(() => {
    if (!topicsGraph) return 0
    return topicsGraph.validation.errors + topicsGraph.validation.warnings
  }, [topicsGraph])

  // Members view mode (graph vs code)
  const [membersViewMode, setMembersViewMode] = useState<MembersViewMode>(() => {
    if (typeof window === 'undefined') return 'graph'
    const stored = localStorage.getItem(MEMBERS_VIEW_STORAGE_KEY)
    return stored === 'code' ? 'code' : 'graph'
  })

  // Graph sub-mode (hierarchy vs topics) — per-team; auto-default derived
  // from team.coordination.reportingMode unless an explicit pick is stored.
  // Initialized to 'hierarchy' as a safe placeholder; the team-change effect
  // immediately recomputes once `team` is known.
  const [graphMode, setGraphMode] = useState<GraphMode>('hierarchy')

  // Recompute graph-mode whenever the active team changes. Stored override
  // (per-team) wins; otherwise auto-default from reportingMode.
  useEffect(() => {
    if (!teamId || !teamReportingMode) return
    const stored = readGraphMode(teamId)
    setGraphMode(stored ?? autoDefaultGraphMode(teamReportingMode))
  }, [teamId, teamReportingMode])

  // Hierarchy layout direction — per-team; default 'TB' (vertical).
  const [layoutDirection, setLayoutDirection] = useState<LayoutDirection>('TB')

  // Recompute layout direction on team change.
  useEffect(() => {
    if (!teamId) return
    setLayoutDirection(readLayoutDirection(teamId))
  }, [teamId])

  // Persist view mode preference
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(MEMBERS_VIEW_STORAGE_KEY, membersViewMode)
    }
  }, [membersViewMode])

  // Operator override: persists per-team and updates state.
  const handleSetGraphMode = useCallback((mode: GraphMode) => {
    setGraphMode(mode)
    if (teamId) writeGraphMode(teamId, mode)
  }, [teamId])

  const handleSetLayoutDirection = useCallback((dir: LayoutDirection) => {
    setLayoutDirection(dir)
    if (teamId) writeLayoutDirection(teamId, dir)
  }, [teamId])

  // Team editor store
  const selectedMemberId = useTeamEditorStore((state) => state.selectedMemberId)
  const setSelectedMemberId = useTeamEditorStore((state) => state.setSelectedMemberId)
  const edges = useTeamEditorStore((state) => state.edges)
  const setEdges = useTeamEditorStore((state) => state.setEdges)
  const updateEdge = useTeamEditorStore((state) => state.updateEdge)
  const reset = useTeamEditorStore((state) => state.reset)
  const memberCount = team?.memberCount ?? team?.members.length ?? 0

  // URL → store: sync only when the URL value itself changes. We compare
  // against the last value we applied (a ref) rather than against the store,
  // so an in-component selection (which is mirrored TO the URL by the effect
  // below) does not boomerang back and overwrite itself.
  const lastAppliedUrlMemberIdRef = useRef<string | null | undefined>(undefined)
  useEffect(() => {
    if (initialMemberId === undefined) return
    if (lastAppliedUrlMemberIdRef.current === initialMemberId) return
    lastAppliedUrlMemberIdRef.current = initialMemberId
    setSelectedMemberId(initialMemberId)
  }, [initialMemberId, setSelectedMemberId])

  // Store → URL: mirror selection out to the parent so it can update the URL.
  const lastReportedMemberIdRef = useRef<string | null | undefined>(undefined)
  useEffect(() => {
    if (lastReportedMemberIdRef.current === selectedMemberId) return
    lastReportedMemberIdRef.current = selectedMemberId
    // Treat URL writes as "already applied" so the URL→store effect above
    // skips the immediate echo back.
    lastAppliedUrlMemberIdRef.current = selectedMemberId
    onMemberSelect?.(selectedMemberId)
  }, [selectedMemberId, onMemberSelect])

  // URL → state: sync section only when URL value changes. Same ref-based
  // pattern as memberId to avoid bounce-back loops.
  const lastAppliedUrlSectionRef = useRef<MemberDetailSection | null | undefined>(undefined)
  useEffect(() => {
    if (initialMemberSection === undefined) return
    if (lastAppliedUrlSectionRef.current === initialMemberSection) return
    lastAppliedUrlSectionRef.current = initialMemberSection
    if (initialMemberSection !== null) {
      setMemberSection(initialMemberSection)
    }
  }, [initialMemberSection])

  // State → URL: mirror section out (cleared when no member is selected).
  const lastReportedSectionRef = useRef<MemberDetailSection | null | undefined>(undefined)
  useEffect(() => {
    const next = selectedMemberId ? memberSection : null
    if (lastReportedSectionRef.current === next) return
    lastReportedSectionRef.current = next
    lastAppliedUrlSectionRef.current = next
    onMemberSectionChange?.(next)
  }, [memberSection, selectedMemberId, onMemberSectionChange])

  // Auto-collapse validation sidebar when MemberDetailPanel opens; restore the
  // user's previous preference when it closes. This avoids the right-side
  // collision identified in the workshop without going to a 3-column layout.
  useEffect(() => {
    if (selectedMemberId) {
      setShowValidation(false)
    } else {
      setShowValidation(userValidationPrefRef.current)
    }
  }, [selectedMemberId])

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
    if (!teamId) {
      reset()
      return
    }

    const loadEdges = async () => {
      try {
        const orgEdges = await orgChartService.getEdges(teamId)
        setEdges(orgEdges)
      } catch (error) {
        console.error('[TeamEditorPanel] Failed to load org chart edges:', error)
      }
    }
    void loadEdges()
  }, [teamId, setEdges, reset])

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
      setMemberSection('heartbeat')
      setSelectedMemberId(agentId)
      setActiveTab('members')
    },
    [setSelectedMemberId]
  )

  // Navigate to a member in the Members tab (from role member chips)
  const handleNavigateToMember = useCallback(
    (agentId: string) => {
      setMemberSection('overview')
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
  useGlobalKeydown((e) => {
    if (e.key !== 'Escape') return
    if (selectedMemberId) {
      setSelectedMemberId(null)
      return
    }
    onClose()
  })

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

            {/* Team icon with health dot */}
            <div className="flex-shrink-0 relative">
              <div className="w-10 h-10 rounded-full flex items-center justify-center bg-primary/20">
                <Users className="h-5 w-5 text-primary" />
              </div>
              <span
                className={cn(
                  'absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-card',
                  teamHealth === 'green' && 'bg-emerald-500',
                  teamHealth === 'yellow' && 'bg-amber-500',
                  teamHealth === 'red' && 'bg-red-500',
                  teamHealth === 'gray' && 'bg-slate-500',
                )}
                title={teamHealth === 'green' ? 'Healthy' : teamHealth === 'yellow' ? 'Some failures' : teamHealth === 'red' ? 'Failing' : 'Disabled'}
              />
            </div>

            {/* Editable name + last active */}
            <div className="flex-1 min-w-0">
              <InlineEditableText
                value={team.displayName}
                onChange={(value) => void handleNameChange(value)}
                placeholder="Team name"
                className="text-lg font-semibold"
              />
              {lastActiveAt && (
                <p className="text-[11px] text-muted-foreground -mt-0.5">
                  Last active: {formatRelativePastTime(new Date(lastActiveAt))}
                </p>
              )}
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
        onValueChange={handleTabChange}
        className="flex-1 flex flex-col min-h-0 overflow-hidden"
      >
        {/* Tab List */}
        {!showDetailOnly && (
          <TabList>
            <TabTrigger value="info" icon={<LayoutDashboard className="h-4 w-4" />} label="Dashboard" />
            <TabTrigger
              value="members"
              icon={<Users className="h-4 w-4" />}
              label={
                validationCount > 0
                  ? `Members (${memberCount}) • ${validationCount}`
                  : `Members (${memberCount})`
              }
            />
            <TabTrigger value="files" icon={<Folder className="h-4 w-4" />} label="Files" />
            <TabTrigger value="prompts" icon={<Eye className="h-4 w-4" />} label="Prompts" />
            <TabTrigger value="activity" icon={<Activity className="h-4 w-4" />} label="Activity" />
          </TabList>
        )}

        {/* Tab Content */}
        <div className="flex-1 min-h-0 flex flex-col">
          {/* Members tab - Split panel layout with view mode toggle */}
          <Tabs.Content
            value="members"
            className="flex-1 min-h-0 flex flex-col data-[state=inactive]:hidden"
          >
            {/* Content area */}
            <div className="flex-1 min-h-0 flex flex-col">
              {membersViewMode === 'graph' ? (
                <>
                  {/* Members-tab toolbar (graph mode toggle, layout direction, code view, add member) */}
                  {!showDetailOnly && (
                    <div className="flex-shrink-0 flex items-center gap-2 px-2 py-1 border-b border-border bg-card/50">
                      <div className="flex items-center gap-1">
                        <span
                          className="text-[10px] uppercase tracking-wide text-muted-foreground mr-1 select-none"
                          aria-hidden="true"
                        >
                          Graph
                        </span>
                        <span
                          className="h-3 w-px bg-border mr-1"
                          aria-hidden="true"
                        />
                        {(['hierarchy', 'topics'] as const).map((mode) => (
                          <button
                            key={mode}
                            type="button"
                            onClick={() => handleSetGraphMode(mode)}
                            className={cn(
                              'px-2 py-0.5 text-xs rounded transition-colors',
                              graphMode === mode
                                ? 'bg-primary text-primary-foreground'
                                : 'text-muted-foreground hover:bg-muted',
                            )}
                            data-testid={`graph-mode-${mode}`}
                            aria-pressed={graphMode === mode}
                          >
                            {mode === 'hierarchy' ? 'Hierarchy' : 'Topics'}
                          </button>
                        ))}
                      </div>

                      <div className="ml-auto flex items-center gap-2">
                        {graphMode === 'hierarchy' && edges.length > 0 && (
                          <button
                            type="button"
                            onClick={() => handleSetLayoutDirection(layoutDirection === 'TB' ? 'LR' : 'TB')}
                            className={cn(
                              'flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg',
                              'bg-card border border-border text-foreground hover:bg-muted transition-colors',
                            )}
                            title={`Layout: ${layoutDirection === 'TB' ? 'Vertical' : 'Horizontal'}`}
                            aria-label="Toggle layout direction"
                            data-testid="hierarchy-layout-toggle"
                          >
                            <LayoutGrid className="h-3.5 w-3.5" />
                            {!isMobile && (
                              <span>{layoutDirection === 'TB' ? 'Layout: Vertical' : 'Layout: Horizontal'}</span>
                            )}
                          </button>
                        )}
                        <button
                          type="button"
                          onClick={handleSwitchToCode}
                          className={cn(
                            'flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg',
                            'bg-card border border-border text-foreground hover:bg-muted transition-colors',
                          )}
                          title="Switch to code view"
                          aria-label="Code View"
                          data-testid="members-code-view"
                        >
                          <Code className="h-3.5 w-3.5" />
                          {!isMobile && <span>Code View</span>}
                        </button>
                        <button
                          type="button"
                          onClick={() => setShowMemberPicker(true)}
                          className={cn(
                            'flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg',
                            'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors',
                          )}
                          title="Add member"
                          aria-label="Add Member"
                          data-testid="members-add-member"
                        >
                          <UserPlus className="h-3.5 w-3.5" />
                          {!isMobile && <span>Add Member</span>}
                        </button>
                      </div>
                    </div>
                  )}

                  <div
                    ref={containerRef}
                    className={cn('flex-1 min-h-0 flex relative', isResizing && 'select-none')}
                  >
                    {/* Left panel: graph (Hierarchy | Topics) */}
                    {!showDetailOnly && (
                      <div className="flex-1 min-w-0 h-full">
                        {graphMode === 'topics' ? (
                          <TopicsGraphPanel
                            teamId={team.id}
                            onSelectMember={(agentId) => {
                              setMemberSection('overview')
                              setSelectedMemberId(agentId)
                            }}
                            showValidation={showValidation}
                            onValidationToggle={handleValidationToggle}
                            onOpenMemberFile={(_team, _member, fileName) => {
                              if (onNavigateToAgentFiles) {
                                onNavigateToAgentFiles(_member, fileName)
                              }
                            }}
                            className="h-full"
                          />
                        ) : (
                          <OrgChartPanel
                            team={team}
                            edges={edges}
                            allAgents={allAgents}
                            selectedMemberId={selectedMemberId}
                            onSelectMember={setSelectedMemberId}
                            onEdgeUpdate={(agentId, managerId) => void handleEdgeUpdate(agentId, managerId)}
                            onAddMember={() => setShowMemberPicker(true)}
                            layoutDirection={layoutDirection}
                            className="h-full"
                          />
                        )}
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
                      <div
                        onMouseDown={handleResizeStart}
                        className={cn(
                          'flex-shrink-0 w-1.5 cursor-col-resize relative group',
                          'hover:bg-primary/30 transition-colors',
                          isResizing && 'bg-primary/50',
                        )}
                      >
                        <div className="absolute inset-y-0 left-1/2 -translate-x-1/2 w-4 flex items-center justify-center">
                          <GripVertical className="h-4 w-4 text-muted-foreground/50 opacity-0 group-hover:opacity-100 transition-opacity" />
                        </div>
                      </div>
                    )}

                    {/* Right panel: Member detail */}
                    {selectedMember && !isCollapsed && (
                      <div
                        style={{ width: detailPanelWidth }}
                        className={cn(
                          'flex-shrink-0 h-full overflow-hidden',
                          !showDetailOnly && 'border-l border-border',
                        )}
                      >
                        <MemberDetailPanel
                          team={team}
                          member={selectedMember}
                          appearance={selectedMemberAppearance}
                          manager={selectedMemberManager}
                          directReports={selectedMemberReports}
                          section={memberSection}
                          onSectionChange={setMemberSection}
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
                </>
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
            <TeamDashboardTab
              team={team}
              onUpdate={onUpdate}
              allAgents={allAgents}
              onNavigateToMemberHeartbeat={handleNavigateToMemberHeartbeat}
              onNavigateToMember={handleNavigateToMember}
              onHealthChange={setTeamHealth}
              onLastActiveChange={setLastActiveAt}
            />
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

          <Tabs.Content
            value="activity"
            className="flex-1 min-h-0 data-[state=inactive]:hidden"
          >
            <TeamActivityTab
              teamId={team.id}
              members={team.members}
              allAgents={allAgents}
              initialSubTab={initialSubTab}
              onSubTabChange={(subTab) => onSubTabChange?.(subTab)}
              className="h-full min-h-0"
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
