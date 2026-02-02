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
import { toast } from '@/hooks/use-toast'
import { Menu, X, GripVertical, Settings } from 'lucide-react'
import { getIcon } from '@/lib/icons'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import { PanelErrorBoundary } from '../PanelErrorBoundary'
import { SkillTreeSidebar } from '../tree/SkillTreeSidebar'
import { SkillEditorPanel } from '../editor/SkillEditorPanel'
import { AgentEditorPanel } from '../editor/AgentEditorPanel'
import { TeamEditorPanel } from '../editor/TeamEditorPanel'
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
import { useUrlState } from '@/hooks/useUrlState'
import { useSidebarPersistence, loadSidebarState } from '@/hooks/useSidebarPersistence'
import { useSelectionStore } from '@/stores/selectionStore'
import { useEditorStore } from '@/stores/editorStore'
import { useAgentEditorStore } from '@/stores/agentEditorStore'
import { useSkillSelectionStore } from '@/stores/skillSelectionStore'
import { useCombineStore } from '@/stores/combineStore'
import { api } from '@/lib/api'
import * as agentService from '@/services/agentService'
import { SettingsDialog } from '../shared/SettingsDialog'
import { getAllItemIdsInSubtree, countSelectedInSubtree } from '@/services/treeService'
import { NewFolderDialog } from '../tree/NewFolderDialog'
import { getSkill } from '@/services/skillService'
import type { TreeNode } from '@/types/editor'
import type { Skill, CreateSkillRequest, UpdateSkillRequest } from '@/types'

const COLLAPSED_SIDEBAR_WIDTH = 60

/**
 * Main layout component for the skill manager.
 */
export function SkillManagerLayout() {
  // Mobile state
  const [isMobile, setIsMobile] = useState(
    typeof window !== 'undefined' ? window.innerWidth < 1024 : false
  )
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false)

  // Dialog states
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
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
    updateAgent,
  } = useAgentData()

  const {
    updateTeam,
    addMember,
    updateMember,
    removeMember,
    setRoles,
  } = useTeamData()

  const isLoading = isLoadingSkills || isLoadingAgents

  // Centralized selection state from Zustand store
  const selectedSkillId = useSelectionStore((state) => state.selectedSkillId)
  const setSelectedSkillId = useSelectionStore((state) => state.setSelectedSkillId)
  const selectedAgentId = useSelectionStore((state) => state.selectedAgentId)
  const setSelectedAgentId = useSelectionStore((state) => state.setSelectedAgentId)
  const selectedTeamId = useSelectionStore((state) => state.selectedTeamId)
  const setSelectedTeamId = useSelectionStore((state) => state.setSelectedTeamId)

  // Get the current team details for editing
  const { team: currentTeam } = useTeamDetails(selectedTeamId)

  // Load initial sidebar state from localStorage (only once on mount)
  const initialSidebarState = useMemo(() => loadSidebarState(), [])

  // Active tab state (managed here for persistence)
  const [activeTab, setActiveTab] = useState(initialSidebarState.activeTab)

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
    selectedTags,
    setSelectedTags,
    availableTags,
    selectedFolders,
    setSelectedFolders,
    availableFolders,
  } = useSkillTree({
    skills,
    initialIsCollapsed: initialSidebarState.isCollapsed,
    initialExpandedNodes: initialSidebarState.expandedNodes,
    initialSelectedTags: initialSidebarState.selectedTags,
    initialSelectedFolders: initialSidebarState.selectedFolders,
    initialSearchQuery: initialSidebarState.searchQuery,
  })

  // Persist sidebar state to localStorage
  useSidebarPersistence({
    isCollapsed,
    expandedNodes,
    selectedTags,
    selectedFolders,
    activeTab,
    searchQuery,
  })

  // Skill selection store
  const skillSelectionMode = useSkillSelectionStore((state) => state.isActive)
  const skillSelectedIds = useSkillSelectionStore((state) => state.selectedSkillIds)
  const skillSelectionTargetAgent = useSkillSelectionStore((state) => state.currentAgent)
  const exitSkillSelectionMode = useSkillSelectionStore((state) => state.exitSkillSelectionMode)
  const toggleSkillSelection = useSkillSelectionStore((state) => state.toggleSkillSelection)
  const toggleMultipleSkills = useSkillSelectionStore((state) => state.toggleMultipleSkills)
  const saveAndExitSkillSelection = useSkillSelectionStore((state) => state.saveAndExit)

  // Combine store
  const combineMode = useCombineStore((state) => state.isActive)
  const combineSelectedIds = useCombineStore((state) => state.selectedSkillIds)
  const combineFormat = useCombineStore((state) => state.format)
  const isCombineCopying = useCombineStore((state) => state.isCopying)
  const enterCombineMode = useCombineStore((state) => state.enterCombineMode)
  const exitCombineMode = useCombineStore((state) => state.exitCombineMode)
  const toggleCombineSkillSelection = useCombineStore((state) => state.toggleSkillSelection)
  const toggleCombineMultipleSkills = useCombineStore((state) => state.toggleMultipleSkills)
  const setCombineFormat = useCombineStore((state) => state.setFormat)
  const setIsCombineCopying = useCombineStore((state) => state.setIsCopying)

  // Combine copy success state (local since it's UI feedback)
  const [combineCopySuccess, setCombineCopySuccess] = useState(false)

  // Skill selection helper functions
  const handleSkillCheckboxChange = useCallback(
    (node: TreeNode) => {
      if (node.isCategory) {
        // Toggle all items in the folder
        const allIds = getAllItemIdsInSubtree(node)
        const allSelected = allIds.every((id) => skillSelectedIds.has(id))
        toggleMultipleSkills(allIds, !allSelected)
      } else if (node.itemId) {
        toggleSkillSelection(node.itemId)
      }
    },
    [skillSelectedIds, toggleSkillSelection, toggleMultipleSkills]
  )

  const getSkillSelectionState = useCallback(
    (node: TreeNode): 'none' | 'partial' | 'all' => {
      if (!node.isCategory && node.itemId) {
        return skillSelectedIds.has(node.itemId) ? 'all' : 'none'
      }

      const allIds = getAllItemIdsInSubtree(node)
      if (allIds.length === 0) return 'none'

      const selectedCount = countSelectedInSubtree(node, skillSelectedIds)
      if (selectedCount === 0) return 'none'
      if (selectedCount === allIds.length) return 'all'
      return 'partial'
    },
    [skillSelectedIds]
  )

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

  const getCombineSelectionState = useCallback(
    (node: TreeNode): 'none' | 'partial' | 'all' => {
      if (!node.isCategory && node.itemId) {
        return combineSelectedIds.has(node.itemId) ? 'all' : 'none'
      }

      const allIds = getAllItemIdsInSubtree(node)
      if (allIds.length === 0) return 'none'

      const selectedCount = countSelectedInSubtree(node, combineSelectedIds)
      if (selectedCount === 0) return 'none'
      if (selectedCount === allIds.length) return 'all'
      return 'partial'
    },
    [combineSelectedIds]
  )

  const handleCombineCopy = useCallback(async () => {
    if (combineSelectedIds.size === 0) return

    setIsCombineCopying(true)
    setCombineCopySuccess(false)

    try {
      const identifiers = Array.from(combineSelectedIds)
      const response = await api.displaySkills(identifiers, combineFormat)

      await navigator.clipboard.writeText(response.combined)
      setCombineCopySuccess(true)

      toast({
        title: 'Copied to clipboard',
        description: `${combineSelectedIds.size} skill${combineSelectedIds.size !== 1 ? 's' : ''} combined as ${combineFormat.toUpperCase()}`,
      })

      // Reset success state after delay
      setTimeout(() => setCombineCopySuccess(false), 2000)
    } catch (error) {
      console.error('Failed to copy combined skills:', error)
      toast({
        title: 'Copy failed',
        description: 'Failed to combine and copy skills',
        variant: 'destructive',
      })
    } finally {
      setIsCombineCopying(false)
    }
  }, [combineSelectedIds, combineFormat, setIsCombineCopying])

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
    originalState: agentOriginalState,
    updateField: updateAgentField,
    updateFields: updateAgentFields,
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
    onDelete: (id: string) => {
      // Agent deletion would go here if implemented
      console.log('Delete agent:', id)
      return Promise.resolve()
    },
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
    // If skill was renamed, update selection to new ID
    if (result.newId) {
      setSelectedSkillId(result.newId)
    }
  }, [saveCurrentSkill, showSaveResultToast, setSelectedSkillId])

  const handleSaveAllChanges = useCallback(async () => {
    const result = await saveAllChanges()
    showSaveResultToast(result, true)
  }, [saveAllChanges, showSaveResultToast])

  // URL state synchronization
  const { updateUrl } = useUrlState({
    onSkillIdChange: useCallback((id: string | null) => {
      // If navigating via browser back/forward with dirty state,
      // the discard dialog is not shown - changes are stored instead
      if (isDirty && id !== selectedSkillId) {
        storeCurrentChanges()
      }
      setSelectedSkillId(id)
    }, [isDirty, selectedSkillId, storeCurrentChanges, setSelectedSkillId]),
    onAgentIdChange: useCallback((id: string | null) => {
      if (isDirty && id !== selectedAgentId) {
        storeCurrentChanges()
      }
      setSelectedAgentId(id)
    }, [isDirty, selectedAgentId, storeCurrentChanges, setSelectedAgentId]),
    onTeamIdChange: useCallback((id: string | null) => {
      if (isDirty && id !== selectedTeamId) {
        storeCurrentChanges()
      }
      setSelectedTeamId(id)
    }, [isDirty, selectedTeamId, storeCurrentChanges, setSelectedTeamId]),
    onSettingsOpenChange: useCallback((open: boolean) => {
      setShowSettingsDialog(open)
    }, []),
    isDirty,
    storeCurrentChanges,
  })

  // Sync URL when selected skill changes
  useEffect(() => {
    updateUrl({ skillId: selectedSkillId })
  }, [selectedSkillId, updateUrl])

  // Sync URL when selected agent changes
  useEffect(() => {
    updateUrl({ agentId: selectedAgentId })
  }, [selectedAgentId, updateUrl])

  // Sync URL when selected team changes
  useEffect(() => {
    updateUrl({ teamId: selectedTeamId })
  }, [selectedTeamId, updateUrl])

  // Sync URL when settings dialog state changes
  useEffect(() => {
    updateUrl({ settingsOpen: showSettingsDialog })
  }, [showSettingsDialog, updateUrl])

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
    (id: string) => {
      // Changes are auto-saved to store, just switch
      setSelectedSkillId(id)

      // Close mobile sidebar after selection
      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    },
    [setSelectedSkillId, isMobile]
  )

  // Handle delete confirmation
  const handleConfirmDelete = useCallback(async () => {
    await deleteCurrentSkill()
    setShowDeleteDialog(false)
    setSelectedSkillId(null)
  }, [deleteCurrentSkill, setSelectedSkillId])

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
      setSelectedSkillId(created.id)

      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    } catch (error) {
      console.error('Failed to create skill:', error)
    }
  }, [createSkill, setSelectedSkillId, isMobile])

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
        setSelectedSkillId(null)
      }
    } catch (error) {
      console.error('Failed to delete folder:', error)
    } finally {
      setDeleteFolderDialog(null)
    }
  }, [deleteFolderDialog, deleteSkillApi, selectedSkillId, setSelectedSkillId])

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
      setSelectedSkillId(created.id)

      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    } catch (error) {
      console.error('Failed to copy skill:', error)
    }
  }, [createSkill, setSelectedSkillId, isMobile])

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
        // Update selection if this was the currently selected skill
        if (selectedSkillId === skillId) {
          setSelectedSkillId(result.id)
        }
      } else {
        useEditorStore.getState().markAsSaved(skillId, result)
      }
      toast({
        title: 'Skill saved',
        description: `"${state.name}" has been saved.`,
      })
    }
  }, [skills, updateSkills, selectedSkillId, setSelectedSkillId])

  const handleDiscardSkillById = useCallback((skillId: string) => {
    useEditorStore.getState().discardChanges(skillId)
  }, [])

  const handleSaveAgentById = useCallback(async (agentId: string) => {
    const agent = agents.find((a) => a.id === agentId)
    if (!agent) return

    const store = useAgentEditorStore.getState()
    const state = store.getFormState(agentId)
    const original = store.getOriginalState(agentId)
    if (!state) return

    // Build update payload
    await updateAgent(agentId, {
      displayName: state.displayName,
      description: state.description,
      status: state.status,
      appearance: state.appearance,
      skills: state.skills,
      tags: state.tags,
    })

    const soulChanged = state.soul !== (original?.soul ?? '')
    if (soulChanged) {
      await agentService.setAgentSoul(agentId, state.soul)
    }

    // Re-fetch the agent to get the updated version
    const updatedAgent = agents.find((a) => a.id === agentId)
    if (updatedAgent) {
      useAgentEditorStore.getState().markAsSaved(agentId, updatedAgent, state.soul)
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
      searchInputRef.current?.focus()
    },
    onEscape: () => {
      // Close any open dialogs first
      if (showDeleteDialog) {
        setShowDeleteDialog(false)
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
      // If editing a skill and not dirty, close the editor and return to skill tree
      if (selectedSkillId && !isDirty) {
        setSelectedSkillId(null)
        return
      }
    },
    onOpenSettings: () => {
      setShowSettingsDialog(true)
    },
  })

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
        isCollapsed={isCollapsed}
        onToggleCollapse={toggleCollapse}
        onExpandAll={expandAll}
        onCollapseAll={collapseAll}
        onCreateNew={(modes) => void handleCreateNew(modes)}
        searchInputRef={searchInputRef}
        onOpenSettings={() => setShowSettingsDialog(true)}
        selectedTags={selectedTags}
        onSelectedTagsChange={setSelectedTags}
        availableTags={availableTags}
        selectedFolders={selectedFolders}
        onSelectedFoldersChange={setSelectedFolders}
        availableFolders={availableFolders}
        skillSelectionMode={skillSelectionMode}
        skillSelectedIds={skillSelectedIds}
        currentAgent={skillSelectionTargetAgent}
        onSkillSelectionSave={() => void saveAndExitSkillSelection()}
        onSkillSelectionCancel={exitSkillSelectionMode}
        getSkillSelectionState={getSkillSelectionState}
        onSkillCheckboxChange={handleSkillCheckboxChange}
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
        getCombineSelectionState={getCombineSelectionState}
        onEnterCombineMode={enterCombineMode}
        onExitCombineMode={exitCombineMode}
        onCombineCopy={() => void handleCombineCopy()}
        isCombineCopying={isCombineCopying}
        combineCopySuccess={combineCopySuccess}
        initialActiveTab={activeTab}
        onActiveTabChange={setActiveTab}
        onSelectSkillFromMenu={setSelectedSkillId}
        onSelectAgentFromMenu={setSelectedAgentId}
        onSaveSkill={handleSaveSkillById}
        onDiscardSkill={handleDiscardSkillById}
        onSaveAgent={handleSaveAgentById}
        onDiscardAgent={handleDiscardAgentById}
        onSaveAll={handleSaveAllFromMenu}
        onDiscardAll={handleDiscardAllFromMenu}
        isSaving={isSaving || isAgentSaving}
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
        {/* Mobile header with menu button */}
        {isMobile && (
          <header className="flex-shrink-0 flex items-center gap-3 px-4 py-3 border-b border-border bg-card/50">
            <button
              type="button"
              onClick={() => setIsMobileSidebarOpen(true)}
              className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
              aria-label="Open menu"
            >
              <Menu className="h-5 w-5" />
            </button>
            <h1 className="flex-1 text-lg font-semibold text-foreground">Skill Manager</h1>
            {dirtyCount > 0 && (
              <span className="px-2 py-0.5 text-xs font-medium bg-amber-500/20 text-amber-400 rounded-full">
                {dirtyCount} unsaved
              </span>
            )}
            <button
              type="button"
              onClick={() => setShowSettingsDialog(true)}
              className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
              aria-label="Settings"
            >
              <Settings className="h-5 w-5" />
            </button>
          </header>
        )}

        {/* Editor panel */}
        <main className="flex-1 overflow-hidden">
          <PanelErrorBoundary panelName="Editor" className="h-full">
            {selectedTeamId ? (
              <TeamEditorPanel
                team={currentTeam ?? null}
                allSkills={skills}
                allAgents={agents}
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
                onClose={() => setSelectedTeamId(null)}
                className="h-full"
              />
            ) : selectedAgentId ? (
              <AgentEditorPanel
                agent={agentFromEditor}
                formState={agentFormState}
                originalState={agentOriginalState}
                allSkills={skills}
                updateField={updateAgentField}
                updateFields={updateAgentFields}
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
                onDelete={() => void deleteCurrentAgent()}
                onClose={() => setSelectedAgentId(null)}
                isSaving={isAgentSaving}
                isDeleting={isAgentDeleting}
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
                isSaving={isSaving}
                isDeleting={isDeleting}
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
          <div className="absolute left-0 top-0 h-full w-72 max-w-[85vw] bg-card shadow-2xl animate-in slide-in-from-left duration-200">
            <div className="flex items-center justify-between px-3 py-3 border-b border-border">
              <h2 className="text-sm font-semibold text-foreground">Skills</h2>
              <button
                type="button"
                onClick={() => setIsMobileSidebarOpen(false)}
                className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                aria-label="Close menu"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="h-[calc(100%-53px)]">
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
                isCollapsed={false}
                onToggleCollapse={() => {}}
                onExpandAll={expandAll}
                onCollapseAll={collapseAll}
                onCreateNew={(modes) => void handleCreateNew(modes)}
                selectedTags={selectedTags}
                onSelectedTagsChange={setSelectedTags}
                availableTags={availableTags}
                selectedFolders={selectedFolders}
                onSelectedFoldersChange={setSelectedFolders}
                availableFolders={availableFolders}
                skillSelectionMode={skillSelectionMode}
                skillSelectedIds={skillSelectedIds}
                currentAgent={skillSelectionTargetAgent}
                onSkillSelectionSave={() => void saveAndExitSkillSelection()}
                onSkillSelectionCancel={exitSkillSelectionMode}
                getSkillSelectionState={getSkillSelectionState}
                onSkillCheckboxChange={handleSkillCheckboxChange}
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
                getCombineSelectionState={getCombineSelectionState}
                onEnterCombineMode={enterCombineMode}
                onExitCombineMode={exitCombineMode}
                onCombineCopy={() => void handleCombineCopy()}
                isCombineCopying={isCombineCopying}
                combineCopySuccess={combineCopySuccess}
                initialActiveTab={activeTab}
                onActiveTabChange={setActiveTab}
                onSelectSkillFromMenu={(id) => {
                  setSelectedSkillId(id)
                  setIsMobileSidebarOpen(false)
                }}
                onSelectAgentFromMenu={(id) => {
                  setSelectedAgentId(id)
                  setIsMobileSidebarOpen(false)
                }}
                onSaveSkill={handleSaveSkillById}
                onDiscardSkill={handleDiscardSkillById}
                onSaveAgent={handleSaveAgentById}
                onDiscardAgent={handleDiscardAgentById}
                onSaveAll={handleSaveAllFromMenu}
                onDiscardAll={handleDiscardAllFromMenu}
                isSaving={isSaving || isAgentSaving}
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
