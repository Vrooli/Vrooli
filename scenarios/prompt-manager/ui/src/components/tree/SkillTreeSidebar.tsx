/**
 * PromptTreeSidebar - Full tree sidebar for prompt navigation.
 *
 * Adapted from agent-inbox ItemTreeSidebar for full-page experience.
 * Features:
 * - Mode-based tree navigation
 * - Search filtering
 * - Tag filtering
 * - Skill selection mode for avatars
 * - Dirty indicators
 * - Collapse/expand controls
 * - New prompt button
 */

import { type ReactNode, type RefObject, useState, useRef, useCallback } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { PanelLeftClose, PanelLeftOpen, Search, Plus, ChevronDown, ChevronUp, Settings, User, Check, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TreeNode } from '@/types/editor'
import type { Prompt } from '@/types'
import type { Avatar } from '@/types/avatar'
import { TreeNodeComponent } from './TreeNode'
import { TagFilterChips } from './TagFilterChips'
import { TagFilterPopover } from './TagFilterPopover'
import { AvatarListPanel } from '../avatar/AvatarListPanel'
import { FolderContextMenu } from './FolderContextMenu'
import { PromptContextMenu } from './PromptContextMenu'
import { getModesPathFromNode, getAllItemIdsInSubtree } from '@/services/treeService'

interface PromptTreeSidebarProps {
  treeNodes: TreeNode[]
  prompts: Prompt[]
  selectedItemId: string | null
  onSelectItem: (id: string) => void
  dirtyItemIds: Set<string>
  expandedNodes: Set<string>
  onToggleNode: (nodeId: string) => void
  renderItemIcon?: (prompt: Prompt) => ReactNode
  searchQuery: string
  onSearchChange: (query: string) => void
  isCollapsed: boolean
  onToggleCollapse: () => void
  onExpandAll: () => void
  onCollapseAll: () => void
  onCreateNew: (modes?: string[]) => void
  /** Ref for the search input (for keyboard shortcuts) */
  searchInputRef?: RefObject<HTMLInputElement>
  /** Callback to open settings modal */
  onOpenSettings?: () => void
  // Tag filter props
  selectedTags: string[]
  onSelectedTagsChange: (tags: string[]) => void
  availableTags: string[]
  // Skill selection mode props
  skillSelectionMode: boolean
  skillSelectedIds: Set<string>
  currentAvatar: Avatar | null
  onSkillSelectionSave: () => void
  onSkillSelectionCancel: () => void
  getSkillSelectionState: (node: TreeNode) => 'none' | 'partial' | 'all'
  onSkillCheckboxChange: (node: TreeNode) => void
  // Context menu callbacks
  onDeleteFolder: (promptIds: string[], folderLabel: string) => void
  onCopyPrompt: (promptId: string) => void
  className?: string
}

/**
 * Full tree sidebar component.
 */
export function PromptTreeSidebar({
  treeNodes,
  prompts,
  selectedItemId,
  onSelectItem,
  dirtyItemIds,
  expandedNodes,
  onToggleNode,
  renderItemIcon,
  searchQuery,
  onSearchChange,
  isCollapsed,
  onToggleCollapse,
  onExpandAll,
  onCollapseAll,
  onCreateNew,
  searchInputRef,
  onOpenSettings,
  selectedTags,
  onSelectedTagsChange,
  availableTags,
  skillSelectionMode,
  skillSelectedIds,
  currentAvatar,
  onSkillSelectionSave,
  onSkillSelectionCancel,
  getSkillSelectionState,
  onSkillCheckboxChange,
  onDeleteFolder,
  onCopyPrompt,
  className = '',
}: PromptTreeSidebarProps) {
  // Count total dirty items
  const dirtyCount = dirtyItemIds.size

  // Tag filter popover state
  const [isTagPopoverOpen, setIsTagPopoverOpen] = useState(false)
  const tagFilterRef = useRef<HTMLDivElement>(null)

  // Avatar state
  const [selectedAvatarId, setSelectedAvatarId] = useState<string | null>(null)

  // Active tab state - locked to prompts when in skill selection mode
  const [activeTab, setActiveTab] = useState('prompts')
  const effectiveTab = skillSelectionMode ? 'prompts' : activeTab

  // Folder context menu state
  const [folderContextMenu, setFolderContextMenu] = useState<{
    node: TreeNode
    x: number
    y: number
  } | null>(null)

  // Prompt context menu state
  const [promptContextMenu, setPromptContextMenu] = useState<{
    promptId: string
    promptName: string
    x: number
    y: number
  } | null>(null)

  const handleCategoryContextMenu = useCallback((node: TreeNode, x: number, y: number) => {
    setPromptContextMenu(null) // Close any open prompt menu
    setFolderContextMenu({ node, x, y })
  }, [])

  const handlePromptContextMenu = useCallback((promptId: string, promptName: string, x: number, y: number) => {
    setFolderContextMenu(null) // Close any open folder menu
    setPromptContextMenu({ promptId, promptName, x, y })
  }, [])

  const handleCloseFolderContextMenu = useCallback(() => {
    setFolderContextMenu(null)
  }, [])

  const handleClosePromptContextMenu = useCallback(() => {
    setPromptContextMenu(null)
  }, [])

  const handleAddPromptInFolder = useCallback(() => {
    if (folderContextMenu) {
      const modes = getModesPathFromNode(folderContextMenu.node)
      onCreateNew(modes)
      setFolderContextMenu(null)
    }
  }, [folderContextMenu, onCreateNew])

  const handleDeleteFolder = useCallback(() => {
    if (folderContextMenu) {
      const promptIds = getAllItemIdsInSubtree(folderContextMenu.node)
      onDeleteFolder(promptIds, folderContextMenu.node.label)
      setFolderContextMenu(null)
    }
  }, [folderContextMenu, onDeleteFolder])

  const handleCopyPrompt = useCallback(() => {
    if (promptContextMenu) {
      onCopyPrompt(promptContextMenu.promptId)
      setPromptContextMenu(null)
    }
  }, [promptContextMenu, onCopyPrompt])

  // Collapsed state - show narrow strip with expand button
  if (isCollapsed) {
    return (
      <div
        className={cn(
          'flex flex-col h-full border-r border-border w-full bg-card/50',
          className
        )}
      >
        <div className="flex flex-col items-center py-3 gap-3">
          <button
            type="button"
            onClick={onToggleCollapse}
            className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            title="Expand sidebar"
          >
            <PanelLeftOpen className="h-4 w-4" />
          </button>
          {dirtyCount > 0 && (
            <span
              className="w-6 h-6 flex items-center justify-center text-[10px] font-medium bg-amber-500/20 text-amber-400 rounded-full"
              title={`${dirtyCount} unsaved changes`}
            >
              {dirtyCount}
            </span>
          )}
          {onOpenSettings && (
            <button
              type="button"
              onClick={onOpenSettings}
              className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
              title="Settings (,)"
            >
              <Settings className="h-4 w-4" />
            </button>
          )}
          <button
            type="button"
            onClick={() => onCreateNew()}
            className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            title="New prompt (Ctrl+N)"
          >
            <Plus className="h-4 w-4" />
          </button>
        </div>
      </div>
    )
  }

  // Expanded state - full sidebar with tabs
  return (
    <div
      className={cn(
        'flex flex-col h-full border-r border-border w-full bg-card/50',
        className
      )}
    >
      {/* Header with tabs */}
      <div className="flex-shrink-0 border-b border-border">
        {/* Top bar with settings and collapse */}
        <div className="flex items-center justify-between px-3 py-2">
          <div className="flex items-center gap-1">
            {skillSelectionMode && currentAvatar ? (
              <div className="flex items-center gap-2">
                <div
                  className="w-6 h-6 rounded-full flex items-center justify-center"
                  style={{ backgroundColor: currentAvatar.bodyColor }}
                >
                  <div
                    className="w-3 h-3 rounded-full"
                    style={{ backgroundColor: currentAvatar.headColor }}
                  />
                </div>
                <span className="text-xs font-medium text-foreground">
                  {skillSelectedIds.size} skill{skillSelectedIds.size !== 1 ? 's' : ''} selected
                </span>
              </div>
            ) : dirtyCount > 0 ? (
              <span className="px-1.5 py-0.5 text-[10px] font-medium bg-amber-500/20 text-amber-400 rounded">
                {dirtyCount} unsaved
              </span>
            ) : null}
          </div>
          <div className="flex items-center gap-1">
            {onOpenSettings && !skillSelectionMode && (
              <button
                type="button"
                onClick={onOpenSettings}
                className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                title="Settings (,)"
              >
                <Settings className="h-4 w-4" />
              </button>
            )}
            {!skillSelectionMode && (
              <button
                type="button"
                onClick={onToggleCollapse}
                className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                title="Collapse sidebar"
              >
                <PanelLeftClose className="h-4 w-4" />
              </button>
            )}
          </div>
        </div>
      </div>

      <Tabs.Root
        value={effectiveTab}
        onValueChange={skillSelectionMode ? undefined : setActiveTab}
        className="flex flex-col flex-1 min-h-0"
      >
        {/* Tab triggers */}
        <Tabs.List className="flex-shrink-0 flex border-b border-border">
          <Tabs.Trigger
            value="prompts"
            disabled={skillSelectionMode}
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium',
              'border-b-2 border-transparent',
              'text-muted-foreground hover:text-foreground',
              'data-[state=active]:text-foreground data-[state=active]:border-primary',
              'transition-colors',
              skillSelectionMode && 'cursor-default'
            )}
          >
            <Search className="h-3.5 w-3.5" />
            Prompts
          </Tabs.Trigger>
          <Tabs.Trigger
            value="avatars"
            disabled={skillSelectionMode}
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium',
              'border-b-2 border-transparent',
              'text-muted-foreground hover:text-foreground',
              'data-[state=active]:text-foreground data-[state=active]:border-primary',
              'transition-colors',
              skillSelectionMode && 'opacity-50 cursor-not-allowed'
            )}
          >
            <User className="h-3.5 w-3.5" />
            Avatars
          </Tabs.Trigger>
        </Tabs.List>

        {/* Prompts Tab */}
        <Tabs.Content value="prompts" className="flex-1 flex flex-col min-h-0 data-[state=inactive]:hidden">
          {/* Search */}
          <div className="flex-shrink-0 px-3 py-2">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => onSearchChange(e.target.value)}
                placeholder={skillSelectionMode ? 'Search skills...' : 'Search prompts... (Ctrl+K)'}
                className={cn(
                  'w-full pl-8 pr-3 py-1.5 text-xs',
                  'bg-muted border border-border rounded-md',
                  'text-foreground placeholder:text-muted-foreground',
                  'focus:outline-none focus:ring-2 focus:ring-primary'
                )}
              />
            </div>

            {/* Tag filter + Expand/Collapse controls */}
            <div className="flex items-center justify-between mt-2 gap-2" ref={tagFilterRef}>
              <div className="relative flex-1 min-w-0">
                <TagFilterChips
                  selectedTags={selectedTags}
                  onRemoveTag={(tag) => onSelectedTagsChange(selectedTags.filter((t) => t !== tag))}
                  onAddFilter={() => setIsTagPopoverOpen(true)}
                  onClearAll={() => onSelectedTagsChange([])}
                />
                <TagFilterPopover
                  availableTags={availableTags}
                  selectedTags={selectedTags}
                  isOpen={isTagPopoverOpen}
                  onClose={() => setIsTagPopoverOpen(false)}
                  onApply={onSelectedTagsChange}
                  className="left-0 top-full"
                />
              </div>
              <div className="flex items-center gap-1 flex-shrink-0">
                <button
                  type="button"
                  onClick={onExpandAll}
                  className="flex items-center gap-1 px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded transition-colors"
                  title="Expand all"
                >
                  <ChevronDown className="h-3 w-3" />
                </button>
                <button
                  type="button"
                  onClick={onCollapseAll}
                  className="flex items-center gap-1 px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded transition-colors"
                  title="Collapse all"
                >
                  <ChevronUp className="h-3 w-3" />
                </button>
              </div>
            </div>
          </div>

          {/* Tree */}
          <div className="flex-1 overflow-y-auto py-1">
            {treeNodes.length === 0 ? (
              <div className="px-3 py-8 text-center">
                <p className="text-xs text-muted-foreground">
                  {searchQuery || selectedTags.length > 0 ? 'No prompts match your filters' : 'No prompts yet'}
                </p>
              </div>
            ) : (
              treeNodes.map((node) => (
                <TreeNodeComponent
                  key={node.id}
                  node={node}
                  prompts={prompts}
                  selectedItemId={selectedItemId}
                  onSelectItem={onSelectItem}
                  dirtyItemIds={dirtyItemIds}
                  expandedNodes={expandedNodes}
                  onToggleNode={onToggleNode}
                  renderItemIcon={renderItemIcon}
                  showCheckbox={skillSelectionMode}
                  onCheckboxChange={onSkillCheckboxChange}
                  getSelectionState={getSkillSelectionState}
                  onCategoryContextMenu={handleCategoryContextMenu}
                  onPromptContextMenu={handlePromptContextMenu}
                />
              ))
            )}

            {/* Folder context menu */}
            {folderContextMenu && (
              <FolderContextMenu
                x={folderContextMenu.x}
                y={folderContextMenu.y}
                folderLabel={folderContextMenu.node.label}
                promptCount={getAllItemIdsInSubtree(folderContextMenu.node).length}
                onClose={handleCloseFolderContextMenu}
                onAddPrompt={handleAddPromptInFolder}
                onDeleteFolder={handleDeleteFolder}
              />
            )}

            {/* Prompt context menu */}
            {promptContextMenu && (
              <PromptContextMenu
                x={promptContextMenu.x}
                y={promptContextMenu.y}
                promptName={promptContextMenu.promptName}
                onClose={handleClosePromptContextMenu}
                onCopyPrompt={handleCopyPrompt}
              />
            )}
          </div>

          {/* Footer - Context dependent */}
          <div className="flex-shrink-0 px-3 py-3 border-t border-border">
            {skillSelectionMode ? (
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={onSkillSelectionCancel}
                  className={cn(
                    'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm',
                    'bg-muted hover:bg-muted/80 text-foreground rounded-lg transition-colors'
                  )}
                >
                  <X className="h-4 w-4" />
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={onSkillSelectionSave}
                  className={cn(
                    'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm',
                    'bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors'
                  )}
                >
                  <Check className="h-4 w-4" />
                  Save Skills
                </button>
              </div>
            ) : (
              <button
                type="button"
                onClick={() => onCreateNew()}
                title="Create new prompt (Ctrl+N)"
                className={cn(
                  'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm',
                  'bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors'
                )}
              >
                <Plus className="h-4 w-4" />
                New Prompt
              </button>
            )}
          </div>
        </Tabs.Content>

        {/* Avatars Tab */}
        <Tabs.Content value="avatars" className="flex-1 flex flex-col min-h-0 data-[state=inactive]:hidden">
          <AvatarListPanel
            selectedAvatarId={selectedAvatarId}
            onSelectAvatar={setSelectedAvatarId}
            onCreateAvatar={() => {}}
            onDeleteAvatar={() => {}}
            className="flex-1"
          />
        </Tabs.Content>
      </Tabs.Root>
    </div>
  )
}
