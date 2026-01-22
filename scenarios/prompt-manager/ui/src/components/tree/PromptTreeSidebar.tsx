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

import { type ReactNode } from 'react'
import { PanelLeftClose, PanelLeftOpen, Search, Plus, ChevronDown, ChevronUp } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TreeNode } from '@/types/editor'
import type { Prompt } from '@/types'
import { TreeNodeComponent } from './TreeNode'

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
  className = '',
}: PromptTreeSidebarProps) {
  // Count total dirty items
  const dirtyCount = dirtyItemIds.size

  // Collapsed state - show narrow strip with expand button
  if (isCollapsed) {
    return (
      <div
        className={cn(
          'flex flex-col h-full border-r border-white/10 w-12 flex-shrink-0 bg-slate-900/50',
          className
        )}
      >
        <div className="flex flex-col items-center py-3 gap-3">
          <button
            type="button"
            onClick={onToggleCollapse}
            className="p-2 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
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
          <button
            type="button"
            onClick={onCreateNew}
            className="p-2 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
            title="New prompt"
          >
            <Plus className="h-4 w-4" />
          </button>
        </div>
      </div>
    )
  }

  // Expanded state - full sidebar
  return (
    <div
      className={cn(
        'flex flex-col h-full border-r border-white/10 w-60 flex-shrink-0 bg-slate-900/50',
        className
      )}
    >
      {/* Header */}
      <div className="flex-shrink-0 px-3 py-3 border-b border-white/10">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-sm font-semibold text-slate-200">Prompts</h3>
          <div className="flex items-center gap-1">
            {dirtyCount > 0 && (
              <span className="px-1.5 py-0.5 text-[10px] font-medium bg-amber-500/20 text-amber-400 rounded">
                {dirtyCount} unsaved
              </span>
            )}
            <button
              type="button"
              onClick={onToggleCollapse}
              className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
              title="Collapse sidebar"
            >
              <PanelLeftClose className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-slate-500" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search prompts..."
            className={cn(
              'w-full pl-8 pr-3 py-1.5 text-xs',
              'bg-slate-800 border border-white/10 rounded-md',
              'text-white placeholder:text-slate-500',
              'focus:outline-none focus:ring-2 focus:ring-indigo-500'
            )}
          />
        </div>

        {/* Expand/Collapse all */}
        <div className="flex items-center gap-1 mt-2">
          <button
            type="button"
            onClick={onExpandAll}
            className="flex items-center gap-1 px-2 py-1 text-[10px] text-slate-400 hover:text-white hover:bg-white/5 rounded transition-colors"
            title="Expand all"
          >
            <ChevronDown className="h-3 w-3" />
            Expand
          </button>
          <button
            type="button"
            onClick={onCollapseAll}
            className="flex items-center gap-1 px-2 py-1 text-[10px] text-slate-400 hover:text-white hover:bg-white/5 rounded transition-colors"
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
            <p className="text-xs text-slate-500">
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
      <div className="flex-shrink-0 px-3 py-3 border-t border-white/10">
        <button
          type="button"
          onClick={onCreateNew}
          className={cn(
            'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm',
            'bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg transition-colors'
          )}
        >
          <Plus className="h-4 w-4" />
          New Prompt
        </button>
      </div>
    </div>
  )
}
