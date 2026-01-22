/**
 * PromptManagerLayout - Main two-panel layout for the prompt manager.
 *
 * Brings together all the components:
 * - PromptTreeSidebar (left)
 * - PromptEditorPanel (right)
 * - Confirmation dialogs
 * - New prompt creation
 *
 * Also handles:
 * - Responsive behavior (drawer on mobile)
 * - Storing changes when switching prompts
 * - Unsaved changes confirmation
 */

import { useState, useCallback, useEffect } from 'react'
import { Menu, X } from 'lucide-react'
import { getIcon } from '../shared/IconSelector'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import { PromptTreeSidebar } from '../tree/PromptTreeSidebar'
import { PromptEditorPanel } from '../editor/PromptEditorPanel'
import { usePromptsData } from '@/hooks/usePromptsData'
import { usePromptTree } from '@/hooks/usePromptTree'
import { usePromptEditor } from '@/hooks/usePromptEditor'
import { useModeSuggestions } from '@/hooks/useModeSuggestions'
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
  const [pendingSelection, setPendingSelection] = useState<string | null>(null)

  // Data fetching
  const {
    prompts,
    isLoading,
    createPrompt,
    updatePrompts,
    deletePrompt: deletePromptApi,
  } = usePromptsData()

  // Tree state
  const {
    filteredTreeNodes,
    selectedItemId,
    setSelectedItemId,
    expandedNodes,
    toggleNode,
    expandAll,
    collapseAll,
    searchQuery,
    setSearchQuery,
    isCollapsed,
    toggleCollapse,
  } = usePromptTree({ prompts })

  // Editor state
  const {
    currentPrompt,
    formState,
    isReadonly,
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
    selectedItemId,
    onSave: updatePrompts,
    onDelete: deletePromptApi,
  })

  // Mode suggestions
  const { getSuggestionsAtLevel } = useModeSuggestions({ prompts })

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
      if (isDirty && id !== selectedItemId) {
        setPendingSelection(id)
        setShowDiscardDialog(true)
        return
      }

      // Store any changes before switching
      storeCurrentChanges()
      setSelectedItemId(id)

      // Close mobile sidebar after selection
      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    },
    [isDirty, selectedItemId, storeCurrentChanges, setSelectedItemId, isMobile]
  )

  // Handle discard confirmation
  const handleConfirmDiscard = useCallback(() => {
    discardCurrentChanges()
    if (pendingSelection) {
      setSelectedItemId(pendingSelection)
      setPendingSelection(null)
    }
    setShowDiscardDialog(false)

    if (isMobile) {
      setIsMobileSidebarOpen(false)
    }
  }, [discardCurrentChanges, pendingSelection, setSelectedItemId, isMobile])

  // Handle delete confirmation
  const handleConfirmDelete = useCallback(async () => {
    await deleteCurrentPrompt()
    setShowDeleteDialog(false)
    setSelectedItemId(null)
  }, [deleteCurrentPrompt, setSelectedItemId])

  // Handle new prompt creation
  const handleCreateNew = useCallback(async () => {
    const newPrompt: CreatePromptRequest = {
      name: 'New Prompt',
      description: '',
      content: '# New Prompt\n\nEnter your prompt content here...',
      modes: [],
      tags: [],
      folder: 'local',
      draft: true,
    }

    try {
      const created = await createPrompt(newPrompt)
      setSelectedItemId(created.id)

      if (isMobile) {
        setIsMobileSidebarOpen(false)
      }
    } catch (error) {
      console.error('Failed to create prompt:', error)
    }
  }, [createPrompt, setSelectedItemId, isMobile])

  // Render item icon in tree
  const renderItemIcon = useCallback((prompt: Prompt) => {
    const Icon = getIcon(prompt.icon || '')
    return <Icon className="h-3.5 w-3.5 flex-shrink-0 text-slate-400" />
  }, [])

  // Sidebar component (reused for desktop and mobile)
  const sidebar = (
    <PromptTreeSidebar
      treeNodes={filteredTreeNodes}
      prompts={prompts}
      selectedItemId={selectedItemId}
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
      onCreateNew={handleCreateNew}
    />
  )

  return (
    <div className="flex h-screen bg-gradient-to-br from-slate-950 to-slate-900">
      {/* Desktop sidebar */}
      {!isMobile && sidebar}

      {/* Main content area */}
      <div className="flex-1 flex flex-col min-w-0">
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
            <h1 className="text-lg font-semibold text-white">Prompt Manager</h1>
            {dirtyCount > 0 && (
              <span className="px-2 py-0.5 text-xs font-medium bg-amber-500/20 text-amber-400 rounded-full">
                {dirtyCount} unsaved
              </span>
            )}
          </header>
        )}

        {/* Editor panel */}
        <main className="flex-1 p-4 overflow-hidden">
          <PromptEditorPanel
            currentPrompt={currentPrompt}
            formState={formState}
            validation={validation}
            isReadonly={isReadonly}
            isDirty={isDirty}
            dirtyCount={dirtyCount}
            onFieldChange={updateField}
            onModesChange={setModes}
            getSuggestionsAtLevel={getSuggestionsAtLevel}
            onSave={saveCurrentPrompt}
            onSaveAll={saveAllChanges}
            onDiscard={discardCurrentChanges}
            onDelete={() => setShowDeleteDialog(true)}
            isSaving={isSaving}
            isDeleting={isDeleting}
            className="h-full"
          />
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
                selectedItemId={selectedItemId}
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
                onCreateNew={handleCreateNew}
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
        onConfirm={handleConfirmDelete}
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
    </div>
  )
}
