/**
 * SkillManagerLayout - Main two-panel layout for the skill manager.
 *
 * Brings together all the components:
 * - SkillTreeSidebar (left, resizable)
 * - SkillEditorPanel (right)
 * - Confirmation dialogs
 * - New skill creation
 *
 * Also handles:
 * - Responsive behavior (drawer on mobile)
 * - Storing changes when switching skills
 * - Unsaved changes confirmation
 * - Resizable sidebar with localStorage persistence
 */

import { useState, useCallback, useEffect, useRef, useMemo } from 'react'
import { useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { toast } from '@/hooks/use-toast'
import { Home, X, GripVertical, Settings } from 'lucide-react'
import { getIcon } from '@/lib/icons'
import { copyToClipboard } from '@/lib/clipboard'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import { DeleteTeamDialog } from '../shared/DeleteTeamDialog'
import { PanelErrorBoundary } from '../PanelErrorBoundary'
import { SkillTreeSidebar } from '../tree/SkillTreeSidebar'
import { SkillEditorPanel } from '../editor/SkillEditorPanel'
import { AgentEditorPanel } from '../editor/AgentEditorPanel'
import { TeamEditorPanel } from '../editor/TeamEditorPanel'
import { RunEditorPanel } from '../editor/RunEditorPanel'
import { TopicEditorPanel } from '../topic/TopicEditorPanel'
import { ActionEditorPanel } from '../action/ActionEditorPanel'
import { TopicSelectionWizard } from '../topic/TopicSelectionWizard'
import { useSkillsData } from '@/hooks/useSkillsData'
import { useAgentData } from '@/hooks/useAgentData'
import { useTeamData, useTeamDetails } from '@/hooks/useTeamData'
import { useSkillTree } from '@/hooks/useSkillTree'
import { usePromptEditor, type SaveResult } from '@/hooks/usePromptEditor'
import { useAgentEditor } from '@/hooks/useAgentEditor'
import { useTeamEditorStore } from '@/hooks/useTeamEditorStore'
import { useModeSuggestions } from '@/hooks/useModeSuggestions'
import { useResizableSidebar } from '@/hooks/useResizableSidebar'
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
import { useSidebarPersistence, loadSidebarState } from '@/hooks/useSidebarPersistence'
import { useRunningAgentStatusSync } from '@/hooks/useRunningAgentStatusSync'
import { usePendingDecisionSync } from '@/hooks/usePendingDecisionSync'
import { RunningAgentsPopover } from '@/components/tree/RunningAgentsPopover'
import { PendingDecisionsPopover } from '@/components/tree/PendingDecisionsPopover'
import { useGraphStore, selectEffectiveHealthScores } from '@/stores/graphStore'
import { useEditorStore } from '@/stores/editorStore'
import { useAgentEditorStore } from '@/stores/agentEditorStore'
import { useCombineStore } from '@/stores/combineStore'
import { formatActions, formatAgents, formatTeams, formatTopics } from '@/lib/formatEntities'
import { recordCopySet } from '@/lib/copySetStorage'
import { api } from '@/lib/api'
import { SettingsDialog } from '../shared/SettingsDialog'
import { getAllItemIdsInSubtree } from '@/services/treeService'
import { NewFolderDialog } from '../tree/NewFolderDialog'
import { getSkill } from '@/services/skillService'
import type { TreeNode } from '@/types/editor'
import type { Skill, CreateSkillRequest, UpdateSkillRequest, ContentSearchOptions, SkillSearchMode } from '@/types'
import type { ContentSearchMatch, Reference } from '@/lib/schemas'
import type { HighlightRequest } from '@/lib/highlight'
import { createHighlightMatch } from '@/lib/highlight'
import { DEFAULT_AGENT_COLORS } from '@/types/agent'
import { useShallow } from 'zustand/react/shallow'
import { useAppBack } from '@/app/routes/useAppBack'
import {
  actionDetailPath,
  agentDetailPath,
  graphPath,
  runDetailPath,
  skillDetailPath,
  teamDetailPath,
  topicDetailPath,
  topicWizardPath,
  worldPath,
  routeForEntity,
} from '@/app/routes/route-paths'

const COLLAPSED_SIDEBAR_WIDTH = 60

function getRouteHome(pathname: string): 'world' | 'graph' {
  return pathname.startsWith('/graph') ? 'graph' : 'world'
}

function highlightFromSearchParams(searchParams: URLSearchParams): HighlightRequest | null {
  const file = searchParams.get('hlFile') ?? undefined
  const lineValue = searchParams.get('hlLine')
  const text = searchParams.get('hlText') ?? ''
  if (!file && !lineValue && !text) return null

  const parsedLine = lineValue ? Number.parseInt(lineValue, 10) : 1
  return {
    file,
    line: Number.isFinite(parsedLine) && parsedLine > 0 ? parsedLine : 1,
    text,
  }
}

/**
 * Main layout component for the skill manager.
 */
export function SkillManagerLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const params = useParams<{
    skillId?: string
    agentId?: string
    teamId?: string
    runId?: string
    topicId?: string
    actionId?: string
  }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const goBack = useAppBack(worldPath())
  const routeHome = getRouteHome(location.pathname)
  const selectedSkillId = params.skillId ?? null
  const selectedAgentId = params.agentId ?? null
  const selectedTeamId = params.teamId ?? null
  const selectedRunId = params.runId ?? null
  const selectedTopicId = params.topicId ?? null
  const selectedActionId = params.actionId ?? null
  const topicWizardActive = location.pathname === topicWizardPath()
  const pendingTab = searchParams.get('tab')
  const pendingSubTab = searchParams.get('subTab')
  const highlightRequest = useMemo(
    () => highlightFromSearchParams(searchParams),
    [searchParams]
  )

  // Running agent sync (single polling instance, feeds 3D world + stores)
  const runningAgentsData = useRunningAgentStatusSync()
  // Pending decision sync (single polling instance, feeds sidebar + world view)
  const pendingDecisionsData = usePendingDecisionSync()

  // Mobile state
  const [isMobile, setIsMobile] = useState(
    typeof window !== 'undefined' ? window.innerWidth < 1024 : false
  )
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false)

  // Dialog states
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [showDeleteAgentDialog, setShowDeleteAgentDialog] = useState(false)
  const [showDeleteTeamDialog, setShowDeleteTeamDialog] = useState(false)
  const [isTeamDeleting, setIsTeamDeleting] = useState(false)
  const [showSettingsDialog, setShowSettingsDialog] = useState(false)

  // Delete folder dialog state
  const [deleteFolderDialog, setDeleteFolderDialog] = useState<{
    skillIds: string[]
    folderLabel: string
  } | null>(null)

  // Search input ref for keyboard shortcut
  const searchInputRef = useRef<HTMLInputElement>(null)

  // Data fetching
  const {
    skills,
    isLoading: isLoadingSkills,
    createSkill,
    updateSkills,
    deleteSkill: deleteSkillApi,
  } = useSkillsData()

  const {
    agents,
    isLoading: isLoadingAgents,
    createAgent,
    updateAgent,
    deleteAgent,
  } = useAgentData()

  const {
    teams,
    updateTeam,
    deleteTeam,
    addMember,
    updateMember,
    removeMember,
    setRoles,
  } = useTeamData()

  const isLoading = isLoadingSkills || isLoadingAgents

  // Get the current team details for editing
  const { team: currentTeam } = useTeamDetails(selectedTeamId)

  // Health scores from graph store.
  // Use useShallow to avoid infinite re-renders from new [] references when graph is null.
  const graphStoreHealthScores = useGraphStore(useShallow(selectEffectiveHealthScores))

  // Eagerly fetch lightweight health scores so sidebar badges appear without visiting graph view.
  const fetchHealthScores = useGraphStore((s) => s.fetchHealthScores)
  useEffect(() => {
    void fetchHealthScores()
  }, [fetchHealthScores])

  // Load initial sidebar state from localStorage (only once on mount)
  const initialSidebarState = useMemo(() => loadSidebarState(), [])

  // Active tab state (managed here for persistence)
  const [activeTab, setActiveTab] = useState(initialSidebarState.activeTab)

  const [searchMode, setSearchMode] = useState<SkillSearchMode>(
    initialSidebarState.searchMode
  )
  const [contentSearchOptions, setContentSearchOptions] = useState<ContentSearchOptions>(
    initialSidebarState.contentSearchOptions
  )

  // Content search matches (for editor highlighting)
  const [contentMatches, setContentMatches] = useState<ContentSearchMatch[]>([])

  // Line number to scroll to in the editor (set when clicking a content search result)
  const [scrollToLine, setScrollToLine] = useState<number | null>(null)

  // Filter matches to only those for the currently selected skill
  const currentSkillMatches = useMemo(() => {
    if (!selectedSkillId || searchMode !== 'content') {
      return []
    }
    return contentMatches.filter((match) => match.skillId === selectedSkillId)
  }, [contentMatches, selectedSkillId, searchMode])

  // Tree state (expansion, filtering, collapse - but NOT selection)
  const {
    filteredTreeNodes,
    expandedNodes,
    toggleNode,
    expandAll,
    collapseAll,
    searchQuery,
    setSearchQuery,
    isCollapsed,
    toggleCollapse,
    expandToItem,
    filterState,
    setFilterState,
    sortConfig,
    setSortConfig,
    viewMode,
    setViewMode,
    detailMode,
    setDetailMode,
    filteredSortedSkills,
    availableTags,
    availableFolders,
  } = useSkillTree({
    skills,
    initialIsCollapsed: initialSidebarState.isCollapsed,
    initialExpandedNodes: initialSidebarState.expandedNodes,
    initialFilterState: initialSidebarState.filterState,
    initialSortConfig: initialSidebarState.sortConfig,
    initialViewMode: initialSidebarState.viewMode,
    initialDetailMode: initialSidebarState.detailMode,
    initialSearchQuery: initialSidebarState.searchQuery,
  })

  // Read health scores from graph store (populated when user visits graph view)
  const healthScoreMap = useMemo(() => {
    if (graphStoreHealthScores.length === 0) return undefined
    const map = new Map<string, number>()
    for (const hs of graphStoreHealthScores) {
      map.set(hs.nodeId, hs.score)
    }
    return map
  }, [graphStoreHealthScores])

  // Persist sidebar state to localStorage
  useSidebarPersistence({
    isCollapsed,
    expandedNodes,
    filterState,
    sortConfig,
    viewMode,
    detailMode,
    activeTab,
    searchQuery,
    searchMode,
    contentSearchOptions,
  })

  // Combine store
  const {
    combineMode,
    combineEntityType,
    combineSelectedIds,
    combineFormat,
    isCombineCopying,
    enterCombineMode,
    exitCombineMode,
    enterAISelectMode,
    toggleCombineSkillSelection,
    toggleCombineMultipleSkills,
    setCombineFormat,
    setIsCombineCopying,
  } = useCombineStore(useShallow((state) => ({
    combineMode: state.isActive,
    combineEntityType: state.entityType,
    combineSelectedIds: state.selectedIds,
    combineFormat: state.format,
    isCombineCopying: state.isCopying,
    enterCombineMode: state.enterCombineMode,
    exitCombineMode: state.exitCombineMode,
    enterAISelectMode: state.enterAISelectMode,
    toggleCombineSkillSelection: state.toggleSelection,
    toggleCombineMultipleSkills: state.toggleMultiple,
    setCombineFormat: state.setFormat,
    setIsCombineCopying: state.setIsCopying,
  })))

  // Combine copy success state (local since it's UI feedback)
  const [combineCopySuccess, setCombineCopySuccess] = useState(false)

  // Pre-fetched combined content — ready before the user taps "Copy" so the
  // clipboard write happens synchronously within the user-activation window.
  const [prefetchedContent, setPrefetchedContent] = useState<string | null>(null)
  const prefetchAbort = useRef<AbortController | null>(null)

  // Re-fetch whenever selection, format, or entity type changes.
  useEffect(() => {
    // Cancel any in-flight fetch.
    prefetchAbort.current?.abort()
    setPrefetchedContent(null)

    if (!combineMode || combineSelectedIds.size === 0) return

    const identifiers = Array.from(combineSelectedIds)
    const abort = new AbortController()
    prefetchAbort.current = abort

    let stale = false
    abort.signal.addEventListener('abort', () => { stale = true })

    const run = async () => {
      try {
        let text: string
        if (combineEntityType === 'skills') {
          const response = await api.displaySkills(identifiers, combineFormat)
          text = response.combined
        } else if (combineEntityType === 'agents') {
          const results = identifiers.map((id) => {
            const agent = agents.find((a) => a.id === id)
            return agent ? {
              id: agent.id,
              displayName: agent.displayName,
              description: agent.description ?? '',
              status: agent.status,
              tags: agent.tags,
              score: 0,
              scorePercent: 0,
            } : null
          }).filter((r): r is NonNullable<typeof r> => r !== null)
          text = formatAgents(results, combineFormat)
        } else if (combineEntityType === 'teams') {
          const teams = await api.getTeams()
          const selected = teams.filter((t) => combineSelectedIds.has(t.id))
          const results = selected.map((t) => ({
            id: t.id,
            displayName: t.displayName,
            mission: t.mission ?? '',
            enabled: t.enabled,
            memberCount: t.memberCount,
            score: 0,
            scorePercent: 0,
          }))
          text = formatTeams(results, combineFormat)
        } else if (combineEntityType === 'topics') {
          const topics = await api.getTopics()
          const selected = topics.filter((t) => combineSelectedIds.has(t.id))
          const results = selected.map((t) => ({
            topic: t,
            score: 0,
          }))
          text = formatTopics(results, combineFormat)
        } else {
          const actions = await api.getActions()
          const selected = actions.filter((action) => combineSelectedIds.has(action.id))
          text = formatActions(selected, combineFormat)
        }
        if (!stale) setPrefetchedContent(text)
      } catch {
        // Prefetch failed — handleCombineCopy will show an error when clicked.
      }
    }
    void run()

    return () => { abort.abort() }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- agents array identity changes; content depends on IDs not refs
  }, [combineMode, combineSelectedIds, combineFormat, combineEntityType])

  // Combine helper functions
  const handleCombineCheckboxChange = useCallback(
    (node: TreeNode) => {
      if (node.isCategory) {
        // Toggle all items in the folder
        const allIds = getAllItemIdsInSubtree(node)
        const allSelected = allIds.every((id) => combineSelectedIds.has(id))
        toggleCombineMultipleSkills(allIds, !allSelected)
      } else if (node.itemId) {
        toggleCombineSkillSelection(node.itemId)
      }
    },
    [combineSelectedIds, toggleCombineSkillSelection, toggleCombineMultipleSkills]
  )

  const handleCombineCopy = useCallback(() => {
    if (combineSelectedIds.size === 0) return

    const identifiers = Array.from(combineSelectedIds)
    const entityLabel = combineEntityType === 'skills' ? 'skill'
      : combineEntityType === 'agents' ? 'agent'
      : combineEntityType === 'teams' ? 'team'
      : combineEntityType === 'topics' ? 'topic'
      : 'action'

    if (prefetchedContent === null) {
      toast({
        title: 'Still loading',
        description: 'Content is being prepared — please try again in a moment.',
      })
      return
    }

    // Content is already resolved — copy happens synchronously within the
    // user-activation window, so clipboard APIs won't hang or be denied.
    setIsCombineCopying(true)
    setCombineCopySuccess(false)

    copyToClipboard(prefetchedContent)
      .then(() => {
        setCombineCopySuccess(true)
        recordCopySet(combineEntityType, identifiers, combineFormat)
        toast({
          title: 'Copied to clipboard',
          description: `${identifiers.length} ${entityLabel}${identifiers.length !== 1 ? 's' : ''} combined as ${combineFormat.toUpperCase()}`,
        })
        setTimeout(() => setCombineCopySuccess(false), 2000)
      })
      .catch((error: unknown) => {
        const msg = error instanceof Error ? error.message : String(error)
        console.error(`Failed to copy combined ${combineEntityType}:`, error)
        toast({
          title: 'Copy failed',
          description: msg,
          variant: 'destructive',
        })
      })
      .finally(() => {
        setIsCombineCopying(false)
      })
  }, [combineSelectedIds, combineFormat, combineEntityType, setIsCombineCopying, prefetchedContent])

  // Skill editor state
  const {
    currentSkill,
    formState,
    originalContent,
    updateField,
    validation,
    isDirty: isSkillDirty,
    dirtyItemIds,
    dirtyCount: skillDirtyCount,
    storeCurrentChanges,
    saveCurrentSkill,
    saveAllChanges,
    discardCurrentChanges,
    deleteCurrentSkill,
    undo,
    redo,
    canUndo,
    canRedo,
    isSaving,
    isDeleting,
    isLoadingContent,
  } = usePromptEditor({
    skills,
    selectedItemId: selectedSkillId,
    onSave: updateSkills,
    onDelete: deleteSkillApi,
  })

  // Agent editor state
  const {
    currentAgent: agentFromEditor,
    formState: agentFormState,
    updateField: updateAgentField,
    updateFields: updateAgentFields,
    renameFileOrderPath: renameAgentFileOrderPath,
    validation: agentValidation,
    isDirty: isAgentDirty,
    dirtyAgentIds,
    dirtyCount: agentDirtyCount,
    undo: undoAgent,
    redo: redoAgent,
    canUndo: canUndoAgent,
    canRedo: canRedoAgent,
    saveCurrentAgent,
    saveAllChanges: saveAllAgentChanges,
    discardCurrentChanges: discardAgentChanges,
    deleteCurrentAgent,
    isSaving: isAgentSaving,
    isDeleting: isAgentDeleting,
  } = useAgentEditor({
    agents,
    selectedAgentId,
    onSave: updateAgent,
    onDelete: deleteAgent,
  })

  // Team editor dirty state - compute count directly (stable primitive)
  const teamDirtyCount = useTeamEditorStore((state) => {
    let count = 0
    for (const dirty of state.dirtyState.values()) {
      if (dirty.responsibilities || dirty.heartbeatInstructions || dirty.schedule) {
        count++
      }
    }
    return count
  })

  // Cache the previous team dirty IDs to avoid creating new Set references
  const prevTeamDirtyIdsRef = useRef<Set<string>>(new Set())

  const teamDirtyMemberIds = useMemo(() => {
    if (teamDirtyCount === 0) {
      if (prevTeamDirtyIdsRef.current.size === 0) {
        return prevTeamDirtyIdsRef.current
      }
      const emptySet = new Set<string>()
      prevTeamDirtyIdsRef.current = emptySet
      return emptySet
    }

    const newDirtyIds = useTeamEditorStore.getState().getDirtyMemberIds()
    const prevIds = prevTeamDirtyIdsRef.current
    if (
      newDirtyIds.size === prevIds.size &&
      [...newDirtyIds].every((id) => prevIds.has(id))
    ) {
      return prevIds
    }
    prevTeamDirtyIdsRef.current = newDirtyIds
    return newDirtyIds
  }, [teamDirtyCount])

  // Combined dirty state for UI
  const isDirty = isSkillDirty || isAgentDirty || teamDirtyCount > 0
  const dirtyCount = skillDirtyCount + agentDirtyCount + teamDirtyCount

  // Combined dirty IDs for sidebar display (skills + agents + team members)
  const combinedDirtyIds = useMemo(() => {
    const combined = new Set(dirtyItemIds)
    for (const id of dirtyAgentIds) {
      combined.add(id)
    }
    for (const id of teamDirtyMemberIds) {
      combined.add(id)
    }
    return combined
  }, [dirtyItemIds, dirtyAgentIds, teamDirtyMemberIds])

  // Toast helper for save results
  const showSaveResultToast = useCallback((result: SaveResult, isSaveAll: boolean) => {
    if (result.success && result.savedCount > 0) {
      toast({
        title: isSaveAll ? 'All changes saved' : 'Skill saved',
        description: result.savedCount === 1
          ? 'Your changes have been saved.'
          : `${result.savedCount} skills saved successfully.`,
      })
    } else if (!result.success) {
      const errorMessage = result.errors.length > 0
        ? result.errors.map((e) => e.message).join('; ')
        : 'An unknown error occurred'
      toast({
        title: result.savedCount > 0
          ? `Partial save: ${result.savedCount} saved, ${result.failedCount} failed`
          : 'Save failed',
        description: errorMessage,
        variant: 'destructive',
      })
    }
  }, [])

  // Wrapped save functions with toast notifications
  const handleSaveCurrentSkill = useCallback(async () => {
    const result = await saveCurrentSkill()
    showSaveResultToast(result, false)
    if (result.newId) {
      navigate(skillDetailPath(result.newId), { replace: true })
    }
  }, [navigate, saveCurrentSkill, showSaveResultToast])

  const handleSaveAllChanges = useCallback(async () => {
    const result = await saveAllChanges()
    showSaveResultToast(result, true)
  }, [saveAllChanges, showSaveResultToast])

  // Auto-expand tree to show selected item
  useEffect(() => {
    if (selectedSkillId) {
      expandToItem(selectedSkillId)
    }
  }, [selectedSkillId, expandToItem])

  // Mode suggestions
  const { getSuggestionsAtLevel } = useModeSuggestions({ skills })

  // Resizable sidebar
  const {
    width: sidebarWidth,
    isResizing,
    containerRef,
    handleResizeStart,
  } = useResizableSidebar({
    defaultWidth: 280,
    minWidth: 200,
    maxWidthRatio: 0.5,
  })

  // Handle window resize
  useEffect(() => {
    const handleResize = () => {
      if (typeof window !== 'undefined') {
        setIsMobile(window.innerWidth < 1024)
      }
    }

    handleResize()
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  // Close mobile sidebar when switching to desktop
  useEffect(() => {
    if (!isMobile) {
      setIsMobileSidebarOpen(false)
    }
  }, [isMobile])

  // Handle item selection - new architecture supports multi-prompt editing
  const handleSelectItem = useCallback(
    (id: string, lineNumber?: number) => {
      if (isDirty && id !== selectedSkillId) {
        storeCurrentChanges()
      }
      const query = lineNumber ? { hlLine: lineNumber } : undefined
      navigate(skillDetailPath(id, query))

      // Store line number so the editor scrolls to it
      setScrollToLine(lineNumber ?? null)

      // Close mobile sidebar after selection
      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    },
    [isDirty, isMobile, navigate, selectedSkillId, storeCurrentChanges]
  )

  // Handle delete confirmation
  const handleConfirmDelete = useCallback(async () => {
    await deleteCurrentSkill()
    setShowDeleteDialog(false)
    navigate(worldPath(), { replace: true })
  }, [deleteCurrentSkill, navigate])

  // Handle new skill creation
  const handleCreateNew = useCallback(async (modes: string[] = []) => {
    const newSkill: CreateSkillRequest = {
      name: 'New Skill',
      description: '',
      content: '# New Skill\n\nEnter your skill content here...',
      modes,
      tags: [],
      folder: 'local',
      draft: true,
    }

    try {
      const created = await createSkill(newSkill)
      navigate(skillDetailPath(created.id))

      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    } catch (error) {
      console.error('Failed to create skill:', error)
    }
  }, [createSkill, isMobile, navigate])

  // Handle delete folder request (shows confirmation dialog)
  const handleDeleteFolderRequest = useCallback((skillIds: string[], folderLabel: string) => {
    setDeleteFolderDialog({ skillIds, folderLabel })
  }, [])

  // Handle delete folder confirmation
  const handleConfirmDeleteFolder = useCallback(async () => {
    if (!deleteFolderDialog) return

    try {
      // Delete all skills in the folder
      for (const skillId of deleteFolderDialog.skillIds) {
        await deleteSkillApi(skillId)
      }
      // Clear selection if the selected skill was in the deleted folder
      if (selectedSkillId && deleteFolderDialog.skillIds.includes(selectedSkillId)) {
        navigate(worldPath(), { replace: true })
      }
    } catch (error) {
      console.error('Failed to delete folder:', error)
    } finally {
      setDeleteFolderDialog(null)
    }
  }, [deleteFolderDialog, deleteSkillApi, navigate, selectedSkillId])

  // Handle copy skill
  const handleCopySkill = useCallback(async (skillId: string) => {
    try {
      // Fetch full skill data including content (list doesn't include content)
      const skill = await getSkill(skillId)
      if (!skill) {
        console.error('Failed to copy skill: skill not found')
        return
      }

      const newSkill: CreateSkillRequest = {
        name: `${skill.name} (Copy)`,
        description: skill.description,
        content: skill.content,
        modes: [...skill.modes],
        tags: [...skill.tags],
        icon: skill.icon ?? undefined,
        folder: skill.folder,
        draft: true,
      }

      const created = await createSkill(newSkill)
      navigate(skillDetailPath(created.id))

      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    } catch (error) {
      console.error('Failed to copy skill:', error)
    }
  }, [createSkill, isMobile, navigate])

  const handleDuplicateAgent = useCallback(async () => {
    if (!agentFromEditor) return

    try {
      const created = await createAgent({
        displayName: `${agentFromEditor.displayName} (Copy)`,
        description: agentFromEditor.description,
        appearance: agentFromEditor.appearance ?? {
          body: DEFAULT_AGENT_COLORS.body,
          head: DEFAULT_AGENT_COLORS.head,
          accent: DEFAULT_AGENT_COLORS.accent,
        },
        capabilities: agentFromEditor.capabilities ?? undefined,
        connectors: agentFromEditor.connectors,
        defaultProfileRef: agentFromEditor.defaultProfileRef ?? undefined,
        heartbeat: agentFromEditor.heartbeat ?? undefined,
        tags: [...agentFromEditor.tags],
        fileOrder: [...agentFromEditor.fileOrder],
      })
      navigate(agentDetailPath(created.id))

      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    } catch (error) {
      console.error('Failed to duplicate agent:', error)
      toast({
        title: 'Duplicate failed',
        description: 'Unable to duplicate agent. Try again.',
      })
    }
  }, [agentFromEditor, createAgent, isMobile, navigate])

  // Context menu: duplicate any agent by ID (not just the currently selected one)
  const handleDuplicateAgentById = useCallback(async (agentId: string) => {
    const agent = agents.find((a) => a.id === agentId)
    if (!agent) return

    try {
      const created = await createAgent({
        displayName: `${agent.displayName} (Copy)`,
        description: agent.description,
        appearance: agent.appearance ?? {
          body: DEFAULT_AGENT_COLORS.body,
          head: DEFAULT_AGENT_COLORS.head,
          accent: DEFAULT_AGENT_COLORS.accent,
        },
        capabilities: agent.capabilities ?? undefined,
        connectors: agent.connectors,
        defaultProfileRef: agent.defaultProfileRef ?? undefined,
        heartbeat: agent.heartbeat ?? undefined,
        tags: [...agent.tags],
        fileOrder: [...agent.fileOrder],
      })
      navigate(agentDetailPath(created.id))

      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    } catch (error) {
      console.error('Failed to duplicate agent:', error)
      toast({
        title: 'Duplicate failed',
        description: 'Unable to duplicate agent. Try again.',
      })
    }
  }, [agents, createAgent, isMobile, navigate])

  // Context menu: open customize modal for a specific agent (selects and opens editor)
  const handleCustomizeAgentById = useCallback((agentId: string) => {
    navigate(agentDetailPath(agentId))
  }, [navigate])

  // Context menu: open prompt preview for a specific agent (selects + switches to prompt tab)
  const [agentEditorInitialTab, setAgentEditorInitialTab] = useState<string | undefined>(undefined)
  const handlePreviewPromptById = useCallback((agentId: string) => {
    navigate(agentDetailPath(agentId, { tab: 'prompt' }))
    setAgentEditorInitialTab('prompt')
  }, [navigate])

  // Clear the one-shot initial tab after it's been consumed by the editor
  useEffect(() => {
    if (agentEditorInitialTab) {
      const timer = setTimeout(() => setAgentEditorInitialTab(undefined), 0)
      return () => clearTimeout(timer)
    }
  }, [agentEditorInitialTab])

  // Context menu: toggle team enabled/disabled
  const handleToggleTeamEnabled = useCallback(async (teamId: string) => {
    const team = teams.find((t) => t.id === teamId)
    if (!team) return

    try {
      await updateTeam(teamId, { enabled: !team.enabled })
    } catch (error) {
      console.error('Failed to toggle team enabled:', error)
      toast({
        title: 'Update failed',
        description: 'Unable to toggle team status. Try again.',
        variant: 'destructive',
      })
    }
  }, [teams, updateTeam])

  const handleConfirmDeleteAgent = useCallback(async () => {
    if (!agentFromEditor) {
      setShowDeleteAgentDialog(false)
      return
    }

    try {
      await deleteCurrentAgent()
      navigate(worldPath(), { replace: true })
    } catch (error) {
      console.error('Failed to delete agent:', error)
      toast({
        title: 'Delete failed',
        description: 'Unable to delete agent. Try again.',
      })
    } finally {
      setShowDeleteAgentDialog(false)
    }
  }, [agentFromEditor, deleteCurrentAgent, navigate])

  const handleConfirmDeleteTeam = useCallback(async (agentIdsToDelete: string[]) => {
    if (!selectedTeamId) {
      setShowDeleteTeamDialog(false)
      return
    }

    setIsTeamDeleting(true)
    try {
      // Delete selected exclusive agents first
      for (const agentId of agentIdsToDelete) {
        try {
          await deleteAgent(agentId)
        } catch (error) {
          console.error(`Failed to delete agent ${agentId}:`, error)
        }
      }

      // Delete the team
      await deleteTeam(selectedTeamId)
      navigate(worldPath(), { replace: true })

      toast({
        title: 'Team deleted',
        description: agentIdsToDelete.length > 0
          ? `Team and ${agentIdsToDelete.length} agent${agentIdsToDelete.length !== 1 ? 's' : ''} deleted.`
          : 'Team has been deleted.',
      })
    } catch (error) {
      console.error('Failed to delete team:', error)
      toast({
        title: 'Delete failed',
        description: 'Unable to delete team. Try again.',
        variant: 'destructive',
      })
    } finally {
      setIsTeamDeleting(false)
      setShowDeleteTeamDialog(false)
    }
  }, [selectedTeamId, deleteTeam, deleteAgent, navigate])

  // Handle move to folder (update skill modes)
  const handleMoveToFolder = useCallback(async (skillId: string, path: string[]) => {
    try {
      const updates = new Map<string, { modes: string[] }>()
      updates.set(skillId, { modes: path })
      await updateSkills(updates)
    } catch (error) {
      console.error('Failed to move skill to folder:', error)
    }
  }, [updateSkills])

  // Handle change storage location (update skill folder)
  const handleChangeStorage = useCallback(async (skillId: string, folder: 'local' | 'core' | 'drafts') => {
    try {
      const updates = new Map<string, { folder: 'local' | 'core' | 'drafts' }>()
      updates.set(skillId, { folder })
      await updateSkills(updates)
    } catch (error) {
      console.error('Failed to change storage location:', error)
    }
  }, [updateSkills])

  // New folder dialog state
  const [newFolderDialog, setNewFolderDialog] = useState<{
    skillId: string
  } | null>(null)

  // Handle create new folder request (opens dialog)
  const handleCreateNewFolderRequest = useCallback((skillId: string) => {
    setNewFolderDialog({ skillId })
  }, [])

  // Handle new folder dialog confirm
  const handleNewFolderConfirm = useCallback(async (path: string[]) => {
    if (newFolderDialog) {
      try {
        const updates = new Map<string, { modes: string[] }>()
        updates.set(newFolderDialog.skillId, { modes: path })
        await updateSkills(updates)
      } catch (error) {
        console.error('Failed to move skill to new folder:', error)
      }
      setNewFolderDialog(null)
    }
  }, [newFolderDialog, updateSkills])

  // Render item icon in tree (uses live icon from editorStore if available)
  const renderItemIcon = useCallback((skill: Skill) => {
    const formState = useEditorStore.getState().getFormState(skill.id)
    const iconName = formState?.icon ?? skill.icon ?? ''
    const Icon = getIcon(iconName)
    return <Icon className="h-3.5 w-3.5 flex-shrink-0 text-muted-foreground" />
  }, [])

  // Individual save/discard callbacks for unsaved changes menu
  const handleSaveSkillById = useCallback(async (skillId: string) => {
    const skill = skills.find((s) => s.id === skillId)
    if (!skill) return

    const state = useEditorStore.getState().getFormState(skillId)
    if (!state) return

    // Build update payload from form state
    const updates = new Map<string, UpdateSkillRequest>()
    const updatePayload: UpdateSkillRequest = {
      name: state.name,
      description: state.description,
      content: state.content,
      modes: state.modes,
      tags: state.tags,
      draft: state.draft,
      folder: state.folder,
      file: state.file, // Include file for rename detection
    }
    // Only include icon if it has a value
    if (state.icon) {
      updatePayload.icon = state.icon
    }
    updates.set(skillId, updatePayload)

    const results = await updateSkills(updates)
    const result = results.get(skillId)
    if (result && !(result instanceof Error)) {
      // Check if ID changed (rename operation)
      if (result.id !== skillId) {
        useEditorStore.getState().movePromptState(skillId, result.id)
        useEditorStore.getState().markAsSaved(result.id, result)
        if (selectedSkillId === skillId) {
          navigate(skillDetailPath(result.id), { replace: true })
        }
      } else {
        useEditorStore.getState().markAsSaved(skillId, result)
      }
      toast({
        title: 'Skill saved',
        description: `"${state.name}" has been saved.`,
      })
    }
  }, [skills, updateSkills, selectedSkillId, navigate])

  const handleDiscardSkillById = useCallback((skillId: string) => {
    useEditorStore.getState().discardChanges(skillId)
  }, [])

  const handleSaveAgentById = useCallback(async (agentId: string) => {
    const agent = agents.find((a) => a.id === agentId)
    if (!agent) return

    const store = useAgentEditorStore.getState()
    const state = store.getFormState(agentId)
    if (!state) return

    // Build update payload
    await updateAgent(agentId, {
      displayName: state.displayName,
      description: state.description,
      status: state.status,
      appearance: state.appearance,
      tags: state.tags,
    })

    // Re-fetch the agent to get the updated version
    const updatedAgent = agents.find((a) => a.id === agentId)
    if (updatedAgent) {
      useAgentEditorStore.getState().markAsSaved(agentId, updatedAgent)
    }

    toast({
      title: 'Agent saved',
      description: `"${state.displayName}" has been saved.`,
    })
  }, [agents, updateAgent])

  const handleDiscardAgentById = useCallback((agentId: string) => {
    useAgentEditorStore.getState().discardChanges(agentId)
  }, [])

  // Combined save all / discard all for the menu
  const handleSaveAllFromMenu = useCallback(async () => {
    // Save all skills
    if (skillDirtyCount > 0) {
      await saveAllChanges()
    }
    // Save all agents
    if (agentDirtyCount > 0) {
      await saveAllAgentChanges()
    }
    toast({
      title: 'All changes saved',
      description: `Saved ${skillDirtyCount + agentDirtyCount} item${skillDirtyCount + agentDirtyCount !== 1 ? 's' : ''}.`,
    })
  }, [skillDirtyCount, agentDirtyCount, saveAllChanges, saveAllAgentChanges])

  const handleDiscardAllFromMenu = useCallback(() => {
    useEditorStore.getState().discardAllChanges()
    useAgentEditorStore.getState().discardAllChanges()
  }, [])

  // Keyboard shortcuts (defined after callbacks so they're available)
  useKeyboardShortcuts({
    onSave: () => {
      if (isDirty && validation.valid) {
        void handleSaveCurrentSkill()
      }
    },
    onSaveAll: () => {
      if (dirtyCount > 0) {
        void handleSaveAllChanges()
      }
    },
    onUndo: () => {
      if (canUndo) {
        undo()
      }
    },
    onRedo: () => {
      if (canRedo) {
        redo()
      }
    },
    onNew: () => void handleCreateNew(),
    onFocusSearch: () => {
      const searchInput = searchInputRef.current
      if (!searchInput) {
        return 'unhandled'
      }
      if (document.activeElement === searchInput) {
        return 'noop'
      }
      searchInput.focus()
      return 'handled'
    },
    onEscape: () => {
      // Close any open dialogs first
      if (showDeleteDialog) {
        setShowDeleteDialog(false)
        return
      }
      if (showDeleteTeamDialog) {
        setShowDeleteTeamDialog(false)
        return
      }
      if (showSettingsDialog) {
        setShowSettingsDialog(false)
        return
      }
      if (deleteFolderDialog) {
        setDeleteFolderDialog(null)
        return
      }
      if (newFolderDialog) {
        setNewFolderDialog(null)
        return
      }
      // Exit combine mode if active
      if (combineMode) {
        exitCombineMode()
        return
      }
      // If no dialogs open and on mobile, close the sidebar
      if (isMobile && isMobileSidebarOpen) {
        setIsMobileSidebarOpen(false)
        return
      }
      if (!isDirty && routeHome !== 'world') {
        goBack()
        return
      }
    },
    onOpenSettings: () => {
      setShowSettingsDialog(true)
    },
  })

  // Navigate to an agent's files in the Agent Editor
  const handleNavigateToAgentFiles = useCallback(
    (agentId: string, filePath?: string) => {
      navigate(agentDetailPath(agentId, filePath ? { hlFile: filePath, hlLine: 1 } : undefined))
    },
    [navigate]
  )

  // Navigate to a running agent's team member view
  const handleNavigateToRunningAgent = useCallback(
    (teamId: string, agentId: string) => {
      setActiveTab('teams')
      navigate(teamDetailPath(teamId, { tab: 'members', member: agentId }))
      // Delay member selection to let the team load first
      requestAnimationFrame(() => {
        useTeamEditorStore.getState().setSelectedMemberId(agentId)
      })
      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    },
    [navigate, setActiveTab, isMobile]
  )

  // Navigate to a team's decision log
  const handleNavigateToDecision = useCallback(
    (teamId: string) => {
      setActiveTab('teams')
      navigate(teamDetailPath(teamId, { tab: 'activity', subTab: 'decisions' }))
      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    },
    [navigate, setActiveTab, isMobile]
  )

  // Handle cross-reference navigation with highlight
  const handleNavigateToXRef = useCallback(
    (ref: Reference) => {
      const { entityType, entityId } = ref.source
      const hlRequest: HighlightRequest = {
        file: ref.source.filePath || undefined,
        line: ref.source.lineNumber,
        text: ref.skillId,
      }

      navigate(routeForEntity(entityType, entityId, {
        hlFile: hlRequest.file ?? null,
        hlLine: hlRequest.line,
        hlText: hlRequest.text,
      }))
    },
    [navigate]
  )

  // Clear highlight URL params after highlight is applied visually
  const handleHighlightHandled = useCallback(() => {
    const next = new URLSearchParams(searchParams)
    next.delete('hlFile')
    next.delete('hlLine')
    next.delete('hlText')
    setSearchParams(next, { replace: true })
  }, [searchParams, setSearchParams])

  const updateRouteSearchParam = useCallback((key: string, value: string | null) => {
    const next = new URLSearchParams(searchParams)
    if (value) {
      next.set(key, value)
    } else {
      next.delete(key)
    }
    setSearchParams(next, { replace: true })
  }, [searchParams, setSearchParams])

  // Compute highlight-based search matches for skill editor
  const highlightSkillMatches = useMemo(() => {
    if (!highlightRequest || !selectedSkillId || selectedAgentId || selectedTeamId) return []
    if (!formState.content) return []
    const match = createHighlightMatch(formState.content, highlightRequest)
    return match ? [match] : []
  }, [highlightRequest, selectedSkillId, selectedAgentId, selectedTeamId, formState.content])

  const effectiveSkillSearchMatches = highlightSkillMatches.length > 0
    ? highlightSkillMatches
    : currentSkillMatches

  const effectiveScrollToLine = highlightRequest && !selectedAgentId && !selectedTeamId
    ? highlightRequest.line
    : scrollToLine

  const handleGoToHomeView = useCallback(() => {
    navigate(routeHome === 'graph' ? graphPath() : worldPath())
  }, [navigate, routeHome])

  const navigateToAgent = useCallback((id: string) => {
    navigate(agentDetailPath(id))
    if (isMobile) setIsMobileSidebarOpen(false)
  }, [isMobile, navigate])

  const navigateToTeam = useCallback((id: string) => {
    navigate(teamDetailPath(id))
    if (isMobile) setIsMobileSidebarOpen(false)
  }, [isMobile, navigate])

  const navigateToRun = useCallback((id: string) => {
    navigate(runDetailPath(id))
    if (isMobile) setIsMobileSidebarOpen(false)
  }, [isMobile, navigate])

  const navigateToTopic = useCallback((id: string) => {
    navigate(topicDetailPath(id))
    if (isMobile) setIsMobileSidebarOpen(false)
  }, [isMobile, navigate])

  const navigateToAction = useCallback((id: string) => {
    navigate(actionDetailPath(id))
    if (isMobile) setIsMobileSidebarOpen(false)
  }, [isMobile, navigate])

  // Sidebar component (reused for desktop and mobile)
  const sidebar = (
    <PanelErrorBoundary panelName="Skill Tree" className="h-full">
      <SkillTreeSidebar
        treeNodes={filteredTreeNodes}
        skills={skills}
        agents={agents}
        selectedItemId={selectedSkillId}
        onSelectItem={handleSelectItem}
        dirtyItemIds={combinedDirtyIds}
        dirtySkillIds={dirtyItemIds}
        dirtyAgentIds={dirtyAgentIds}
        dirtyTeamMemberIds={teamDirtyMemberIds}
        expandedNodes={expandedNodes}
        onToggleNode={toggleNode}
        renderItemIcon={renderItemIcon}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        searchMode={searchMode}
        onSearchModeChange={setSearchMode}
        contentSearchOptions={contentSearchOptions}
        onContentSearchOptionsChange={setContentSearchOptions}
        isCollapsed={isCollapsed}
        onToggleCollapse={toggleCollapse}
        onExpandAll={expandAll}
        onCollapseAll={collapseAll}
        onCreateNew={(modes) => void handleCreateNew(modes)}
        searchInputRef={searchInputRef}
        onOpenSettings={() => setShowSettingsDialog(true)}
        filterState={filterState}
        onFilterStateChange={setFilterState}
        sortConfig={sortConfig}
        onSortConfigChange={setSortConfig}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        detailMode={detailMode}
        onDetailModeChange={setDetailMode}
        healthScoreMap={healthScoreMap}
        filteredSortedSkills={filteredSortedSkills}
        availableTags={availableTags}
        availableFolders={availableFolders}
        onDeleteFolder={handleDeleteFolderRequest}
        onCopySkill={(skillId) => void handleCopySkill(skillId)}
        onMoveToFolder={(skillId, path) => void handleMoveToFolder(skillId, path)}
        onChangeStorage={(skillId, folder) => void handleChangeStorage(skillId, folder)}
        onCreateNewFolder={handleCreateNewFolderRequest}
        combineMode={combineMode}
        combineSelectedIds={combineSelectedIds}
        combineFormat={combineFormat}
        onCombineFormatChange={setCombineFormat}
        onCombineToggle={handleCombineCheckboxChange}
        onEnterCombineMode={enterCombineMode}
        onExitCombineMode={exitCombineMode}
        onEnterSelectMode={enterAISelectMode}
        combineEntityType={combineEntityType}
        onCombineCopy={() => { handleCombineCopy() }}
        isCombineCopying={isCombineCopying}
        combineCopySuccess={combineCopySuccess}
        initialActiveTab={activeTab}
        onActiveTabChange={setActiveTab}
        onSelectSkillFromMenu={(id) => handleSelectItem(id)}
        onSelectAgentFromMenu={navigateToAgent}
        selectedAgentId={selectedAgentId}
        onSelectTeamFromMenu={navigateToTeam}
        selectedTeamId={selectedTeamId}
        onSelectRunFromMenu={navigateToRun}
        selectedRunId={selectedRunId}
        onSelectTopicFromMenu={navigateToTopic}
        selectedTopicId={selectedTopicId}
        onSaveSkill={handleSaveSkillById}
        onDiscardSkill={handleDiscardSkillById}
        onSaveAgent={handleSaveAgentById}
        onDiscardAgent={handleDiscardAgentById}
        onSaveAll={handleSaveAllFromMenu}
        onDiscardAll={handleDiscardAllFromMenu}
        isSaving={isSaving || isAgentSaving}
        onContentMatchesChange={setContentMatches}
        onNavigateToRunningAgent={handleNavigateToRunningAgent}
        runningAgentsData={runningAgentsData}
        pendingDecisionsData={pendingDecisionsData}
        onNavigateToDecision={handleNavigateToDecision}
        onOpenTopicWizard={() => navigate(topicWizardPath())}
        onDuplicateAgent={(id) => void handleDuplicateAgentById(id)}
        onCustomizeAgent={handleCustomizeAgentById}
        onPreviewPrompt={handlePreviewPromptById}
        onToggleTeamEnabled={(id) => void handleToggleTeamEnabled(id)}
        onSelectActionFromMenu={navigateToAction}
        selectedActionId={selectedActionId}
      />
    </PanelErrorBoundary>
  )

  return (
    <div ref={containerRef} className="flex h-screen bg-gradient-to-br from-background to-background dark:from-slate-950 dark:to-slate-900">
      {/* Desktop sidebar with resize handle */}
      {!isMobile && (
        <div
          className="relative flex-shrink-0 transition-[width] duration-200"
          style={{ width: isCollapsed ? COLLAPSED_SIDEBAR_WIDTH : sidebarWidth }}
        >
          {sidebar}
          {/* Resize handle - wider hit area (12px) with narrow visual indicator */}
          {!isCollapsed && (
            <div
              role="separator"
              aria-orientation="vertical"
              aria-label="Resize sidebar"
              tabIndex={0}
              onMouseDown={handleResizeStart}
              className={`
                absolute top-0 right-0 h-full w-3 cursor-col-resize
                flex items-center justify-center group
                ${isResizing ? '' : ''}
              `}
            >
              {/* Visual indicator - narrow line with subtle visibility */}
              <div
                className={`
                  absolute right-0 top-0 h-full w-0.5 transition-colors
                  ${isResizing ? 'bg-primary' : 'bg-border group-hover:bg-primary/50'}
                `}
              />
              <GripVertical
                className={`
                  h-6 w-3 text-muted-foreground opacity-30 group-hover:opacity-100 transition-opacity z-10
                  ${isResizing ? 'opacity-100 text-primary' : ''}
                `}
              />
            </div>
          )}
        </div>
      )}

      {/* Main content area */}
      <div className={`flex-1 flex flex-col min-w-0 ${isResizing ? 'select-none' : ''}`}>
        {/* Editor panel */}
        <main className="flex-1 overflow-hidden">
          <PanelErrorBoundary panelName="Editor" className="h-full">
            {topicWizardActive ? (
              <TopicSelectionWizard
                onClose={goBack}
                className="h-full"
              />
            ) : selectedRunId ? (
              <RunEditorPanel
                runId={selectedRunId}
                onClose={goBack}
                onOpenSidebar={isMobile ? () => setIsMobileSidebarOpen(true) : undefined}
                className="h-full"
              />
            ) : selectedTopicId ? (
              <TopicEditorPanel
                topicId={selectedTopicId}
                onClose={goBack}
                onOpenSidebar={isMobile ? () => setIsMobileSidebarOpen(true) : undefined}
                className="h-full"
              />
            ) : selectedActionId ? (
              <ActionEditorPanel
                actionId={selectedActionId}
                onClose={goBack}
                onOpenSidebar={isMobile ? () => setIsMobileSidebarOpen(true) : undefined}
                className="h-full"
              />
            ) : selectedTeamId ? (
              <TeamEditorPanel
                team={currentTeam ?? null}
                allAgents={agents}
                initialTab={pendingTab}
                initialSubTab={pendingSubTab}
                onTabChange={(tab) => updateRouteSearchParam('tab', tab)}
                onSubTabChange={(subTab) => updateRouteSearchParam('subTab', subTab)}
                onNavigateToAgentFiles={handleNavigateToAgentFiles}
                onUpdate={async (updates) => {
                  if (selectedTeamId) {
                    await updateTeam(selectedTeamId, updates)
                  }
                }}
                onAddMember={async (request) => {
                  if (selectedTeamId) {
                    return addMember(selectedTeamId, request)
                  }
                  throw new Error('No team selected')
                }}
                onUpdateMember={async (agentId, request) => {
                  if (selectedTeamId) {
                    return updateMember(selectedTeamId, agentId, request)
                  }
                  throw new Error('No team selected')
                }}
                onRemoveMember={async (agentId) => {
                  if (selectedTeamId) {
                    await removeMember(selectedTeamId, agentId)
                  }
                }}
                onSetRoles={async (roles) => {
                  if (selectedTeamId) {
                    return setRoles(selectedTeamId, roles)
                  }
                  throw new Error('No team selected')
                }}
                onClose={goBack}
                onOpenSidebar={isMobile ? () => setIsMobileSidebarOpen(true) : undefined}
                onDelete={() => setShowDeleteTeamDialog(true)}
                isDeleting={isTeamDeleting}
                highlightRequest={highlightRequest}
                onHighlightHandled={handleHighlightHandled}
                className="h-full"
              />
            ) : selectedAgentId ? (
              <AgentEditorPanel
                agent={agentFromEditor}
                formState={agentFormState}
                updateField={updateAgentField}
                updateFields={updateAgentFields}
                renameFileOrderPath={renameAgentFileOrderPath}
                validation={agentValidation}
                isDirty={isAgentDirty}
                dirtyCount={agentDirtyCount}
                onUndo={undoAgent}
                onRedo={redoAgent}
                canUndo={canUndoAgent}
                canRedo={canRedoAgent}
                onSave={() => void saveCurrentAgent()}
                onSaveAll={() => void saveAllAgentChanges()}
                onDiscard={discardAgentChanges}
                onDelete={() => setShowDeleteAgentDialog(true)}
                onDuplicate={() => void handleDuplicateAgent()}
                onClose={goBack}
                onOpenSidebar={isMobile ? () => setIsMobileSidebarOpen(true) : undefined}
                isSaving={isAgentSaving}
                isDeleting={isAgentDeleting}
                highlightRequest={highlightRequest}
                onHighlightHandled={handleHighlightHandled}
                initialTab={agentEditorInitialTab}
                className="h-full"
              />
            ) : (
              <SkillEditorPanel
                currentSkill={currentSkill}
                formState={formState}
                originalContent={originalContent}
                validation={validation}
                allSkills={skills}
                isDirty={isDirty}
                dirtyCount={dirtyCount}
                onFieldChange={updateField}
                availableTags={availableTags}
                onUndo={undo}
                onRedo={redo}
                canUndo={canUndo}
                canRedo={canRedo}
                onSave={() => void handleSaveCurrentSkill()}
                onSaveAll={() => void handleSaveAllChanges()}
                onDiscard={discardCurrentChanges}
                onDelete={() => setShowDeleteDialog(true)}
                onSelectSkill={handleSelectItem}
                onSelectTeam={navigateToTeam}
                isSaving={isSaving}
                isDeleting={isDeleting}
                isLoadingContent={isLoadingContent}
                searchMatches={effectiveSkillSearchMatches}
                scrollToLine={effectiveScrollToLine}
                onScrollToLineHandled={() => {
                  setScrollToLine(null)
                  if (highlightRequest) {
                    handleHighlightHandled()
                  }
                }}
                onNavigateToXRef={handleNavigateToXRef}
                onClose={goBack}
                onOpenSidebar={isMobile ? () => setIsMobileSidebarOpen(true) : undefined}
                onOpenMobileSidebar={isMobile ? () => setIsMobileSidebarOpen(true) : undefined}
                pendingDecisionCount={pendingDecisionsData.count}
                runningAgentCount={runningAgentsData.count}
                homeView={routeHome}
                onHomeViewChange={(view) => navigate(view === 'graph' ? graphPath() : worldPath())}
                className="h-full"
              />
            )}
          </PanelErrorBoundary>
        </main>
      </div>

      {/* Mobile sidebar overlay */}
      {isMobile && isMobileSidebarOpen && (
        <div className="fixed inset-0 z-50">
          {/* Backdrop */}
          <div
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
            onClick={() => setIsMobileSidebarOpen(false)}
          />

          {/* Sidebar drawer */}
          <div className="absolute left-0 top-0 h-full w-full bg-card shadow-2xl animate-in slide-in-from-left duration-200">
            <div className="flex items-center justify-between px-3 py-3 border-b border-border">
              <div className="flex items-center gap-2 min-w-0">
                <button
                  type="button"
                  onClick={() => {
                    handleGoToHomeView()
                    setIsMobileSidebarOpen(false)
                  }}
                  className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                  aria-label="Go to world or grid view"
                >
                  <Home className="h-5 w-5" />
                </button>
                <h2 className="text-sm font-semibold text-foreground truncate">Skills</h2>
              </div>
              <div className="flex items-center gap-1">
                {runningAgentsData.count > 0 && (
                  <RunningAgentsPopover
                    onNavigateToMember={(teamId, agentId) => {
                      handleNavigateToRunningAgent(teamId, agentId)
                      setIsMobileSidebarOpen(false)
                    }}
                    groupedByTeam={runningAgentsData.groupedByTeam}
                    count={runningAgentsData.count}
                    stopAgent={runningAgentsData.stopAgent}
                    stoppingIds={runningAgentsData.stoppingIds}
                  />
                )}
                {pendingDecisionsData.count > 0 && (
                  <PendingDecisionsPopover
                    onNavigateToDecision={(teamId) => {
                      handleNavigateToDecision(teamId)
                      setIsMobileSidebarOpen(false)
                    }}
                    groupedByTeam={pendingDecisionsData.groupedByTeam}
                    count={pendingDecisionsData.count}
                    acceptDecision={pendingDecisionsData.acceptDecision}
                    rejectDecision={pendingDecisionsData.rejectDecision}
                    processingIds={pendingDecisionsData.processingIds}
                  />
                )}
                <button
                  type="button"
                  onClick={() => setShowSettingsDialog(true)}
                  className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                  aria-label="Settings"
                >
                  <Settings className="h-5 w-5" />
                </button>
                <button
                  type="button"
                  onClick={() => setIsMobileSidebarOpen(false)}
                  className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                  aria-label="Close menu"
                >
                  <X className="h-5 w-5" />
                </button>
              </div>
            </div>
            <div className="h-[calc(100%-53px)]">
              <SkillTreeSidebar
                treeNodes={filteredTreeNodes}
                skills={skills}
                agents={agents}
                selectedItemId={selectedSkillId}
                onSelectItem={(id, lineNumber) => {
                  handleSelectItem(id, lineNumber)
                  setIsMobileSidebarOpen(false)
                }}
                dirtyItemIds={combinedDirtyIds}
                dirtySkillIds={dirtyItemIds}
                dirtyAgentIds={dirtyAgentIds}
                dirtyTeamMemberIds={teamDirtyMemberIds}
                expandedNodes={expandedNodes}
                onToggleNode={toggleNode}
                renderItemIcon={renderItemIcon}
                searchQuery={searchQuery}
                onSearchChange={setSearchQuery}
                searchMode={searchMode}
                onSearchModeChange={setSearchMode}
                contentSearchOptions={contentSearchOptions}
                onContentSearchOptionsChange={setContentSearchOptions}
                isCollapsed={false}
                onToggleCollapse={() => {}}
                onExpandAll={expandAll}
                onCollapseAll={collapseAll}
                onCreateNew={(modes) => void handleCreateNew(modes)}
                filterState={filterState}
                onFilterStateChange={setFilterState}
                sortConfig={sortConfig}
                onSortConfigChange={setSortConfig}
                viewMode={viewMode}
                onViewModeChange={setViewMode}
        detailMode={detailMode}
        onDetailModeChange={setDetailMode}
        healthScoreMap={healthScoreMap}
                filteredSortedSkills={filteredSortedSkills}
                availableTags={availableTags}
                availableFolders={availableFolders}
                onDeleteFolder={handleDeleteFolderRequest}
                onCopySkill={(skillId) => void handleCopySkill(skillId)}
                onMoveToFolder={(skillId, path) => void handleMoveToFolder(skillId, path)}
                onChangeStorage={(skillId, folder) => void handleChangeStorage(skillId, folder)}
                onCreateNewFolder={handleCreateNewFolderRequest}
                combineMode={combineMode}
                combineSelectedIds={combineSelectedIds}
                combineFormat={combineFormat}
                onCombineFormatChange={setCombineFormat}
                onCombineToggle={handleCombineCheckboxChange}
                onEnterCombineMode={enterCombineMode}
                onExitCombineMode={exitCombineMode}
                onEnterSelectMode={enterAISelectMode}
                combineEntityType={combineEntityType}
                onCombineCopy={() => { handleCombineCopy() }}
                isCombineCopying={isCombineCopying}
                combineCopySuccess={combineCopySuccess}
                initialActiveTab={activeTab}
                onActiveTabChange={setActiveTab}
                onSelectSkillFromMenu={(id) => {
                  handleSelectItem(id)
                  setIsMobileSidebarOpen(false)
                }}
                onSelectAgentFromMenu={(id) => {
                  navigate(agentDetailPath(id))
                  setIsMobileSidebarOpen(false)
                }}
                selectedAgentId={selectedAgentId}
                onSelectTeamFromMenu={(id) => {
                  navigate(teamDetailPath(id))
                  setIsMobileSidebarOpen(false)
                }}
                selectedTeamId={selectedTeamId}
                onSelectRunFromMenu={(id) => {
                  navigate(runDetailPath(id))
                  setIsMobileSidebarOpen(false)
                }}
                selectedRunId={selectedRunId}
                onSelectTopicFromMenu={(id) => {
                  navigate(topicDetailPath(id))
                  setIsMobileSidebarOpen(false)
                }}
                selectedTopicId={selectedTopicId}
                onSelectActionFromMenu={(id) => {
                  navigate(actionDetailPath(id))
                  setIsMobileSidebarOpen(false)
                }}
                selectedActionId={selectedActionId}
                onSaveSkill={handleSaveSkillById}
                onDiscardSkill={handleDiscardSkillById}
                onSaveAgent={handleSaveAgentById}
                onDiscardAgent={handleDiscardAgentById}
                onSaveAll={handleSaveAllFromMenu}
                onDiscardAll={handleDiscardAllFromMenu}
                isSaving={isSaving || isAgentSaving}
                onContentMatchesChange={setContentMatches}
                onNavigateToRunningAgent={handleNavigateToRunningAgent}
                runningAgentsData={runningAgentsData}
        pendingDecisionsData={pendingDecisionsData}
        onNavigateToDecision={handleNavigateToDecision}
                onOpenTopicWizard={() => navigate(topicWizardPath())}
                onDuplicateAgent={(id) => void handleDuplicateAgentById(id)}
                onCustomizeAgent={handleCustomizeAgentById}
                onPreviewPrompt={handlePreviewPromptById}
                onToggleTeamEnabled={(id) => void handleToggleTeamEnabled(id)}
                hideTopControlsRow={true}
                className="border-r-0"
              />
            </div>
          </div>
        </div>
      )}

      {/* New folder dialog */}
      <NewFolderDialog
        isOpen={newFolderDialog !== null}
        onClose={() => setNewFolderDialog(null)}
        onConfirm={(path) => void handleNewFolderConfirm(path)}
        getSuggestionsAtLevel={getSuggestionsAtLevel}
      />

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        isOpen={showDeleteDialog}
        onClose={() => setShowDeleteDialog(false)}
        onConfirm={() => void handleConfirmDelete()}
        title="Delete Skill"
        message={`Are you sure you want to delete "${currentSkill?.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        variant="danger"
        isLoading={isDeleting}
      />

      {/* Delete agent confirmation dialog */}
      <ConfirmDialog
        isOpen={showDeleteAgentDialog}
        onClose={() => setShowDeleteAgentDialog(false)}
        onConfirm={() => void handleConfirmDeleteAgent()}
        title="Delete Agent"
        message={`Are you sure you want to delete "${agentFromEditor?.displayName}"? This action cannot be undone.`}
        confirmLabel="Delete"
        variant="danger"
        isLoading={isAgentDeleting}
      />

      {/* Delete team dialog with exclusive member cleanup */}
      <DeleteTeamDialog
        isOpen={showDeleteTeamDialog}
        teamId={selectedTeamId}
        teamName={currentTeam?.displayName ?? ''}
        onClose={() => setShowDeleteTeamDialog(false)}
        onConfirm={(agentIds) => void handleConfirmDeleteTeam(agentIds)}
        isLoading={isTeamDeleting}
      />

      {/* Delete folder confirmation dialog */}
      <ConfirmDialog
        isOpen={deleteFolderDialog !== null}
        onClose={() => setDeleteFolderDialog(null)}
        onConfirm={() => void handleConfirmDeleteFolder()}
        title="Delete Folder?"
        message={deleteFolderDialog
          ? `Are you sure you want to delete the folder "${deleteFolderDialog.folderLabel}" and all ${deleteFolderDialog.skillIds.length} skill${deleteFolderDialog.skillIds.length !== 1 ? 's' : ''} inside it? This action cannot be undone.`
          : ''
        }
        confirmLabel="Delete All"
        cancelLabel="Cancel"
        variant="danger"
      />

      {/* Loading overlay */}
      {isLoading && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-background/80 backdrop-blur-sm">
          <div className="text-center">
            <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-3" />
            <p className="text-sm text-muted-foreground">Loading skills...</p>
          </div>
        </div>
      )}

      {/* Settings dialog */}
      <SettingsDialog
        isOpen={showSettingsDialog}
        onClose={() => setShowSettingsDialog(false)}
      />
    </div>
  )
}
