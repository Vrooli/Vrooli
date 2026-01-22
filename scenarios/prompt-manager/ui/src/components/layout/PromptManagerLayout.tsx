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
import { SettingsDialog } from '../shared/SettingsDialog'
import type { Prompt, CreatePromptRequest } from '@/types'

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
  } = usePromptTree({ prompts })

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
  const handleCreateNew = useCallback(async () => {
    const newPrompt: CreatePromptRequest = {
      name: 'New Prompt',
      description: '',
      content: '# New Prompt\n\nEnter your prompt content here...',
      modes: [],
      tags: [],
      folder: 'internal',
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

  // Render item icon in tree
  const renderItemIcon = useCallback((prompt: Prompt) => {
    const Icon = getIcon(prompt.icon || '')
    return <Icon className="h-3.5 w-3.5 flex-shrink-0 text-slate-400" />
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
        onCreateNew={() => void handleCreateNew()}
        searchInputRef={searchInputRef}
        onOpenSettings={() => setShowSettingsDialog(true)}
      />
    </PanelErrorBoundary>
  )

  return (
    <div ref={containerRef} className="flex h-screen bg-gradient-to-br from-slate-950 to-slate-900">
      {/* Desktop sidebar with resize handle */}
      {!isMobile && (
        <div
          className="relative flex-shrink-0"
          style={{ width: sidebarWidth }}
        >
          {sidebar}
          {/* Resize handle - wider hit area (12px) with narrow visual indicator */}
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
                ${isResizing ? 'bg-indigo-500' : 'bg-slate-700 group-hover:bg-indigo-500/50'}
              `}
            />
            <GripVertical
              className={`
                h-6 w-3 text-slate-600 opacity-30 group-hover:opacity-100 transition-opacity z-10
                ${isResizing ? 'opacity-100 text-indigo-400' : ''}
              `}
            />
          </div>
        </div>
      )}

      {/* Main content area */}
      <div className={`flex-1 flex flex-col min-w-0 ${isResizing ? 'select-none' : ''}`}>
        {/* Mobile header with menu button */}
        {isMobile && (
          <header className="flex-shrink-0 flex items-center gap-3 px-4 py-3 border-b border-white/10 bg-slate-900/50">
            <button
              type="button"
              onClick={() => setIsMobileSidebarOpen(true)}
              className="p-2 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
              aria-label="Open menu"
            >
              <Menu className="h-5 w-5" />
            </button>
            <h1 className="flex-1 text-lg font-semibold text-white">Prompt Manager</h1>
            {dirtyCount > 0 && (
              <span className="px-2 py-0.5 text-xs font-medium bg-amber-500/20 text-amber-400 rounded-full">
                {dirtyCount} unsaved
              </span>
            )}
            <button
              type="button"
              onClick={() => setShowSettingsDialog(true)}
              className="p-2 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
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
          <div className="absolute left-0 top-0 h-full w-72 max-w-[85vw] bg-slate-900 shadow-2xl animate-in slide-in-from-left duration-200">
            <div className="flex items-center justify-between px-3 py-3 border-b border-white/10">
              <h2 className="text-sm font-semibold text-white">Prompts</h2>
              <button
                type="button"
                onClick={() => setIsMobileSidebarOpen(false)}
                className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
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
                onCreateNew={() => void handleCreateNew()}
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

      {/* Loading overlay */}
      {isLoading && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/80 backdrop-blur-sm">
          <div className="text-center">
            <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
            <p className="text-sm text-slate-400">Loading prompts...</p>
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
