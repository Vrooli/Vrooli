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
import { Menu, X, GripVertical, Settings } from 'lucide-react'
import { getIcon } from '@/lib/icons'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import { PanelErrorBoundary } from '../PanelErrorBoundary'
import { SkillTreeSidebar } from '../tree/SkillTreeSidebar'
import { SkillEditorPanel } from '../editor/SkillEditorPanel'
import { useSkillsData } from '@/hooks/useSkillsData'
import { useSkillTree } from '@/hooks/useSkillTree'
import { useSkillEditor } from '@/hooks/useSkillEditor'
import { useModeSuggestions } from '@/hooks/useModeSuggestions'
import { useResizableSidebar } from '@/hooks/useResizableSidebar'
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
import { useUrlState } from '@/hooks/useUrlState'
import { useSidebarPersistence, loadSidebarState } from '@/hooks/useSidebarPersistence'
import { useSelectionStore } from '@/stores/selectionStore'
import { useSkillSelectionStore } from '@/stores/skillSelectionStore'
import { SettingsDialog } from '../shared/SettingsDialog'
import { getAllItemIdsInSubtree, countSelectedInSubtree } from '@/services/treeService'
import { getSkill } from '@/services/skillService'
import type { TreeNode } from '@/types/editor'
import type { Skill, CreateSkillRequest } from '@/types'

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
  const [showDiscardDialog, setShowDiscardDialog] = useState(false)
  const [showSettingsDialog, setShowSettingsDialog] = useState(false)
  const [pendingSelection, setPendingSelection] = useState<string | null>(null)

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
    isLoading,
    createSkill,
    updateSkills,
    deleteSkill: deleteSkillApi,
  } = useSkillsData()

  // Centralized selection state from Zustand store
  const selectedSkillId = useSelectionStore((state) => state.selectedSkillId)
  const setSelectedSkillId = useSelectionStore((state) => state.setSelectedSkillId)

  // Load initial sidebar state from localStorage (only once on mount)
  const initialSidebarState = useMemo(() => loadSidebarState(), [])

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
  } = useSkillTree({
    skills,
    initialIsCollapsed: initialSidebarState.isCollapsed,
    initialExpandedNodes: initialSidebarState.expandedNodes,
    initialSelectedTags: initialSidebarState.selectedTags,
  })

  // Persist sidebar state to localStorage
  useSidebarPersistence({
    isCollapsed,
    expandedNodes,
    selectedTags,
  })

  // Skill selection store
  const skillSelectionMode = useSkillSelectionStore((state) => state.isActive)
  const skillSelectedIds = useSkillSelectionStore((state) => state.selectedSkillIds)
  const currentMember = useSkillSelectionStore((state) => state.currentMember)
  const exitSkillSelectionMode = useSkillSelectionStore((state) => state.exitSkillSelectionMode)
  const toggleSkillSelection = useSkillSelectionStore((state) => state.toggleSkillSelection)
  const toggleMultipleSkills = useSkillSelectionStore((state) => state.toggleMultipleSkills)
  const saveAndExitSkillSelection = useSkillSelectionStore((state) => state.saveAndExit)

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

  // Editor state
  const {
    currentSkill,
    formState,
    updateField,
    setModes,
    validation,
    isDirty,
    dirtyItemIds,
    dirtyCount,
    storeCurrentChanges,
    saveCurrentSkill,
    saveAllChanges,
    discardCurrentChanges,
    deleteCurrentSkill,
    isSaving,
    isDeleting,
  } = useSkillEditor({
    skills,
    selectedItemId: selectedSkillId,
    onSave: updateSkills,
    onDelete: deleteSkillApi,
  })

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

  // Handle item selection with dirty check
  const handleSelectItem = useCallback(
    (id: string) => {
      // If there are unsaved changes, ask for confirmation
      if (isDirty && id !== selectedSkillId) {
        setPendingSelection(id)
        setShowDiscardDialog(true)
        return
      }

      // Store any changes before switching
      storeCurrentChanges()
      setSelectedSkillId(id)

      // Close mobile sidebar after selection
      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    },
    [isDirty, selectedSkillId, storeCurrentChanges, setSelectedSkillId, isMobile]
  )

  // Handle discard confirmation
  const handleConfirmDiscard = useCallback(() => {
    discardCurrentChanges()
    if (pendingSelection) {
      setSelectedSkillId(pendingSelection)
      setPendingSelection(null)
    }
    setShowDiscardDialog(false)

    if (isMobile) {
      setIsMobileSidebarOpen(false)
    }
  }, [discardCurrentChanges, pendingSelection, setSelectedSkillId, isMobile])

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
        icon: skill.icon,
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

  // Render item icon in tree
  const renderItemIcon = useCallback((skill: Skill) => {
    const Icon = getIcon(skill.icon || '')
    return <Icon className="h-3.5 w-3.5 flex-shrink-0 text-muted-foreground" />
  }, [])

  // Keyboard shortcuts (defined after callbacks so they're available)
  useKeyboardShortcuts({
    onSave: () => {
      if (isDirty && validation.valid) {
        void saveCurrentSkill()
      }
    },
    onSaveAll: () => {
      if (dirtyCount > 0) {
        void saveAllChanges()
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
      if (showDiscardDialog) {
        setShowDiscardDialog(false)
        setPendingSelection(null)
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
        selectedItemId={selectedSkillId}
        onSelectItem={handleSelectItem}
        dirtyItemIds={dirtyItemIds}
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
        skillSelectionMode={skillSelectionMode}
        skillSelectedIds={skillSelectedIds}
        currentMember={currentMember}
        onSkillSelectionSave={() => void saveAndExitSkillSelection()}
        onSkillSelectionCancel={exitSkillSelectionMode}
        getSkillSelectionState={getSkillSelectionState}
        onSkillCheckboxChange={handleSkillCheckboxChange}
        onDeleteFolder={handleDeleteFolderRequest}
        onCopySkill={(skillId) => void handleCopySkill(skillId)}
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
            <SkillEditorPanel
              currentSkill={currentSkill}
              formState={formState}
              validation={validation}
              allSkills={skills}
              isDirty={isDirty}
              dirtyCount={dirtyCount}
              onFieldChange={updateField}
              onModesChange={setModes}
              getSuggestionsAtLevel={getSuggestionsAtLevel}
              onSave={() => void saveCurrentSkill()}
              onSaveAll={() => void saveAllChanges()}
              onDiscard={discardCurrentChanges}
              onDelete={() => setShowDeleteDialog(true)}
              onSelectSkill={handleSelectItem}
              isSaving={isSaving}
              isDeleting={isDeleting}
              className="h-full"
            />
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
                selectedItemId={selectedSkillId}
                onSelectItem={handleSelectItem}
                dirtyItemIds={dirtyItemIds}
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
                skillSelectionMode={skillSelectionMode}
                skillSelectedIds={skillSelectedIds}
                currentMember={currentMember}
                onSkillSelectionSave={() => void saveAndExitSkillSelection()}
                onSkillSelectionCancel={exitSkillSelectionMode}
                getSkillSelectionState={getSkillSelectionState}
                onSkillCheckboxChange={handleSkillCheckboxChange}
                onDeleteFolder={handleDeleteFolderRequest}
                onCopySkill={(skillId) => void handleCopySkill(skillId)}
                className="border-r-0"
              />
            </div>
          </div>
        </div>
      )}

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

      {/* Discard changes dialog */}
      <ConfirmDialog
        isOpen={showDiscardDialog}
        onClose={() => {
          setShowDiscardDialog(false)
          setPendingSelection(null)
        }}
        onConfirm={handleConfirmDiscard}
        title="Discard Changes?"
        message="You have unsaved changes. Do you want to discard them and switch to a different skill?"
        confirmLabel="Discard"
        cancelLabel="Keep Editing"
        variant="warning"
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
