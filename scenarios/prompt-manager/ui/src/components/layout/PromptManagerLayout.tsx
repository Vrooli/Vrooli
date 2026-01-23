/**
 * PromptManagerLayout - Main two-panel layout for the prompt manager.
 *
 * Brings together all the components:
 * - PromptTreeSidebar (left, resizable)
 * - PromptEditorPanel (right)
 * - Confirmation dialogs
 * - New prompt creation
 *
 * Also handles:
 * - Responsive behavior (drawer on mobile)
 * - Storing changes when switching prompts
 * - Unsaved changes confirmation
 * - Resizable sidebar with localStorage persistence
 */

import { useState, useCallback, useEffect, useRef } from 'react'
import { Menu, X, GripVertical, Settings } from 'lucide-react'
import { getIcon } from '@/lib/icons'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import { PanelErrorBoundary } from '../PanelErrorBoundary'
import { PromptTreeSidebar } from '../tree/PromptTreeSidebar'
import { PromptEditorPanel } from '../editor/PromptEditorPanel'
import { usePromptsData } from '@/hooks/usePromptsData'
import { usePromptTree } from '@/hooks/usePromptTree'
import { usePromptEditor } from '@/hooks/usePromptEditor'
import { useModeSuggestions } from '@/hooks/useModeSuggestions'
import { useResizableSidebar } from '@/hooks/useResizableSidebar'
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
import { useSelectionStore } from '@/stores/selectionStore'
import { useSkillSelectionStore } from '@/stores/skillSelectionStore'
import { SettingsDialog } from '../shared/SettingsDialog'
import { getAllItemIdsInSubtree, countSelectedInSubtree } from '@/services/treeService'
import type { TreeNode } from '@/types/editor'
import type { Prompt, CreatePromptRequest } from '@/types'

const COLLAPSED_SIDEBAR_WIDTH = 60

/**
 * Main layout component for the prompt manager.
 */
export function PromptManagerLayout() {
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
    promptIds: string[]
    folderLabel: string
  } | null>(null)

  // Search input ref for keyboard shortcut
  const searchInputRef = useRef<HTMLInputElement>(null)

  // Data fetching
  const {
    prompts,
    isLoading,
    createPrompt,
    updatePrompts,
    deletePrompt: deletePromptApi,
  } = usePromptsData()

  // Centralized selection state from Zustand store
  const selectedPromptId = useSelectionStore((state) => state.selectedPromptId)
  const setSelectedPromptId = useSelectionStore((state) => state.setSelectedPromptId)

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
  } = usePromptTree({ prompts })

  // Skill selection store
  const skillSelectionMode = useSkillSelectionStore((state) => state.isActive)
  const skillSelectedIds = useSkillSelectionStore((state) => state.selectedSkillIds)
  const currentAvatar = useSkillSelectionStore((state) => state.currentAvatar)
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
    currentPrompt,
    formState,
    updateField,
    setModes,
    validation,
    isDirty,
    dirtyItemIds,
    dirtyCount,
    storeCurrentChanges,
    saveCurrentPrompt,
    saveAllChanges,
    discardCurrentChanges,
    deleteCurrentPrompt,
    isSaving,
    isDeleting,
  } = usePromptEditor({
    prompts,
    selectedItemId: selectedPromptId,
    onSave: updatePrompts,
    onDelete: deletePromptApi,
  })

  // Auto-expand tree to show selected item
  useEffect(() => {
    if (selectedPromptId) {
      expandToItem(selectedPromptId)
    }
  }, [selectedPromptId, expandToItem])

  // Mode suggestions
  const { getSuggestionsAtLevel } = useModeSuggestions({ prompts })

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
      if (isDirty && id !== selectedPromptId) {
        setPendingSelection(id)
        setShowDiscardDialog(true)
        return
      }

      // Store any changes before switching
      storeCurrentChanges()
      setSelectedPromptId(id)

      // Close mobile sidebar after selection
      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    },
    [isDirty, selectedPromptId, storeCurrentChanges, setSelectedPromptId, isMobile]
  )

  // Handle discard confirmation
  const handleConfirmDiscard = useCallback(() => {
    discardCurrentChanges()
    if (pendingSelection) {
      setSelectedPromptId(pendingSelection)
      setPendingSelection(null)
    }
    setShowDiscardDialog(false)

    if (isMobile) {
      setIsMobileSidebarOpen(false)
    }
  }, [discardCurrentChanges, pendingSelection, setSelectedPromptId, isMobile])

  // Handle delete confirmation
  const handleConfirmDelete = useCallback(async () => {
    await deleteCurrentPrompt()
    setShowDeleteDialog(false)
    setSelectedPromptId(null)
  }, [deleteCurrentPrompt, setSelectedPromptId])

  // Handle new prompt creation
  const handleCreateNew = useCallback(async (modes: string[] = []) => {
    const newPrompt: CreatePromptRequest = {
      name: 'New Prompt',
      description: '',
      content: '# New Prompt\n\nEnter your prompt content here...',
      modes,
      tags: [],
      folder: 'local',
      draft: true,
    }

    try {
      const created = await createPrompt(newPrompt)
      setSelectedPromptId(created.id)

      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    } catch (error) {
      console.error('Failed to create prompt:', error)
    }
  }, [createPrompt, setSelectedPromptId, isMobile])

  // Handle delete folder request (shows confirmation dialog)
  const handleDeleteFolderRequest = useCallback((promptIds: string[], folderLabel: string) => {
    setDeleteFolderDialog({ promptIds, folderLabel })
  }, [])

  // Handle delete folder confirmation
  const handleConfirmDeleteFolder = useCallback(async () => {
    if (!deleteFolderDialog) return

    try {
      // Delete all prompts in the folder
      for (const promptId of deleteFolderDialog.promptIds) {
        await deletePromptApi(promptId)
      }
      // Clear selection if the selected prompt was in the deleted folder
      if (selectedPromptId && deleteFolderDialog.promptIds.includes(selectedPromptId)) {
        setSelectedPromptId(null)
      }
    } catch (error) {
      console.error('Failed to delete folder:', error)
    } finally {
      setDeleteFolderDialog(null)
    }
  }, [deleteFolderDialog, deletePromptApi, selectedPromptId, setSelectedPromptId])

  // Handle copy prompt
  const handleCopyPrompt = useCallback(async (promptId: string) => {
    const prompt = prompts.find(p => p.id === promptId)
    if (!prompt) return

    const newPrompt: CreatePromptRequest = {
      name: `${prompt.name} (Copy)`,
      description: prompt.description,
      content: prompt.content,
      modes: [...prompt.modes],
      tags: [...prompt.tags],
      icon: prompt.icon,
      folder: prompt.folder,
      draft: true,
    }

    try {
      const created = await createPrompt(newPrompt)
      setSelectedPromptId(created.id)

      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    } catch (error) {
      console.error('Failed to copy prompt:', error)
    }
  }, [prompts, createPrompt, setSelectedPromptId, isMobile])

  // Render item icon in tree
  const renderItemIcon = useCallback((prompt: Prompt) => {
    const Icon = getIcon(prompt.icon || '')
    return <Icon className="h-3.5 w-3.5 flex-shrink-0 text-muted-foreground" />
  }, [])

  // Keyboard shortcuts (defined after callbacks so they're available)
  useKeyboardShortcuts({
    onSave: () => {
      if (isDirty && validation.valid) {
        void saveCurrentPrompt()
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
      // If editing a prompt and not dirty, close the editor and return to skill tree
      if (selectedPromptId && !isDirty) {
        setSelectedPromptId(null)
        return
      }
    },
    onOpenSettings: () => {
      setShowSettingsDialog(true)
    },
  })

  // Sidebar component (reused for desktop and mobile)
  const sidebar = (
    <PanelErrorBoundary panelName="Prompt Tree" className="h-full">
      <PromptTreeSidebar
        treeNodes={filteredTreeNodes}
        prompts={prompts}
        selectedItemId={selectedPromptId}
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
        currentAvatar={currentAvatar}
        onSkillSelectionSave={() => void saveAndExitSkillSelection()}
        onSkillSelectionCancel={exitSkillSelectionMode}
        getSkillSelectionState={getSkillSelectionState}
        onSkillCheckboxChange={handleSkillCheckboxChange}
        onDeleteFolder={handleDeleteFolderRequest}
        onCopyPrompt={(promptId) => void handleCopyPrompt(promptId)}
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
            <h1 className="flex-1 text-lg font-semibold text-foreground">Prompt Manager</h1>
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
            <PromptEditorPanel
              currentPrompt={currentPrompt}
              formState={formState}
              validation={validation}
              allPrompts={prompts}
              isDirty={isDirty}
              dirtyCount={dirtyCount}
              onFieldChange={updateField}
              onModesChange={setModes}
              getSuggestionsAtLevel={getSuggestionsAtLevel}
              onSave={() => void saveCurrentPrompt()}
              onSaveAll={() => void saveAllChanges()}
              onDiscard={discardCurrentChanges}
              onDelete={() => setShowDeleteDialog(true)}
              onSelectPrompt={handleSelectItem}
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
              <h2 className="text-sm font-semibold text-foreground">Prompts</h2>
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
              <PromptTreeSidebar
                treeNodes={filteredTreeNodes}
                prompts={prompts}
                selectedItemId={selectedPromptId}
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
                currentAvatar={currentAvatar}
                onSkillSelectionSave={() => void saveAndExitSkillSelection()}
                onSkillSelectionCancel={exitSkillSelectionMode}
                getSkillSelectionState={getSkillSelectionState}
                onSkillCheckboxChange={handleSkillCheckboxChange}
                onDeleteFolder={handleDeleteFolderRequest}
                onCopyPrompt={(promptId) => void handleCopyPrompt(promptId)}
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
        title="Delete Prompt"
        message={`Are you sure you want to delete "${currentPrompt?.name}"? This action cannot be undone.`}
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
        message="You have unsaved changes. Do you want to discard them and switch to a different prompt?"
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
          ? `Are you sure you want to delete the folder "${deleteFolderDialog.folderLabel}" and all ${deleteFolderDialog.promptIds.length} prompt${deleteFolderDialog.promptIds.length !== 1 ? 's' : ''} inside it? This action cannot be undone.`
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
            <p className="text-sm text-muted-foreground">Loading prompts...</p>
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
