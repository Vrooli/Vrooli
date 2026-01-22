/**
 * PromptTreeSidebar - Full tree sidebar for prompt navigation.
 *
 * Adapted from agent-inbox ItemTreeSidebar for full-page experience.
 * Features:
 * - Mode-based tree navigation
 * - Search filtering
 * - Dirty indicators
 * - Collapse/expand controls
 * - New prompt button
 */

import { type ReactNode, type RefObject, useState } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { PanelLeftClose, PanelLeftOpen, Search, Plus, ChevronDown, ChevronUp, Settings, User } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TreeNode } from '@/types/editor'
import type { Prompt } from '@/types'
import { TreeNodeComponent } from './TreeNode'
import { AvatarListPanel } from '../avatar/AvatarListPanel'

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
  onCreateNew: () => void
  /** Ref for the search input (for keyboard shortcuts) */
  searchInputRef?: RefObject<HTMLInputElement>
  /** Callback to open settings modal */
  onOpenSettings?: () => void
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
  className = '',
}: PromptTreeSidebarProps) {
  // Count total dirty items
  const dirtyCount = dirtyItemIds.size

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
            onClick={onCreateNew}
            className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            title="New prompt (Ctrl+N)"
          >
            <Plus className="h-4 w-4" />
          </button>
        </div>
      </div>
    )
  }

  // Avatar state
  const [selectedAvatarId, setSelectedAvatarId] = useState<string | null>(null)

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
            {dirtyCount > 0 && (
              <span className="px-1.5 py-0.5 text-[10px] font-medium bg-amber-500/20 text-amber-400 rounded">
                {dirtyCount} unsaved
              </span>
            )}
          </div>
          <div className="flex items-center gap-1">
            {onOpenSettings && (
              <button
                type="button"
                onClick={onOpenSettings}
                className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                title="Settings (,)"
              >
                <Settings className="h-4 w-4" />
              </button>
            )}
            <button
              type="button"
              onClick={onToggleCollapse}
              className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
              title="Collapse sidebar"
            >
              <PanelLeftClose className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>

      <Tabs.Root defaultValue="prompts" className="flex flex-col flex-1 min-h-0">
        {/* Tab triggers */}
        <Tabs.List className="flex-shrink-0 flex border-b border-border">
          <Tabs.Trigger
            value="prompts"
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium',
              'border-b-2 border-transparent',
              'text-muted-foreground hover:text-foreground',
              'data-[state=active]:text-foreground data-[state=active]:border-primary',
              'transition-colors'
            )}
          >
            <Search className="h-3.5 w-3.5" />
            Prompts
          </Tabs.Trigger>
          <Tabs.Trigger
            value="avatars"
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium',
              'border-b-2 border-transparent',
              'text-muted-foreground hover:text-foreground',
              'data-[state=active]:text-foreground data-[state=active]:border-primary',
              'transition-colors'
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
                placeholder="Search prompts... (Ctrl+K)"
                className={cn(
                  'w-full pl-8 pr-3 py-1.5 text-xs',
                  'bg-muted border border-border rounded-md',
                  'text-foreground placeholder:text-muted-foreground',
                  'focus:outline-none focus:ring-2 focus:ring-primary'
                )}
              />
            </div>

            {/* Expand/Collapse all */}
            <div className="flex items-center gap-1 mt-2">
              <button
                type="button"
                onClick={onExpandAll}
                className="flex items-center gap-1 px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded transition-colors"
                title="Expand all"
              >
                <ChevronDown className="h-3 w-3" />
                Expand
              </button>
              <button
                type="button"
                onClick={onCollapseAll}
                className="flex items-center gap-1 px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded transition-colors"
                title="Collapse all"
              >
                <ChevronUp className="h-3 w-3" />
                Collapse
              </button>
            </div>
          </div>

          {/* Tree */}
          <div className="flex-1 overflow-y-auto py-1">
            {treeNodes.length === 0 ? (
              <div className="px-3 py-8 text-center">
                <p className="text-xs text-muted-foreground">
                  {searchQuery ? 'No prompts match your search' : 'No prompts yet'}
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
                />
              ))
            )}
          </div>

          {/* Footer - New prompt button */}
          <div className="flex-shrink-0 px-3 py-3 border-t border-border">
            <button
              type="button"
              onClick={onCreateNew}
              title="Create new prompt (Ctrl+N)"
              className={cn(
                'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm',
                'bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors'
              )}
            >
              <Plus className="h-4 w-4" />
              New Prompt
            </button>
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
