/**
 * TreeNode - Recursive tree node component.
 *
 * Renders a single node in the prompt tree, handling:
 * - Category nodes (expandable)
 * - Leaf nodes (selectable prompts)
 * - Dirty indicators
 */

import { type ReactNode } from 'react'
import { ChevronRight, ChevronDown, FolderOpen } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TreeNode as TreeNodeType } from '@/types/editor'
import type { Prompt } from '@/types'
import { countDirtyInSubtree } from '@/services/treeService'

interface TreeNodeProps {
  node: TreeNodeType
  prompts: Prompt[]
  selectedItemId: string | null
  onSelectItem: (id: string) => void
  dirtyItemIds: Set<string>
  expandedNodes: Set<string>
  onToggleNode: (nodeId: string) => void
  renderItemIcon?: (prompt: Prompt) => ReactNode
}

/**
 * Recursive tree node component.
 */
export function TreeNodeComponent({
  node,
  prompts,
  selectedItemId,
  onSelectItem,
  dirtyItemIds,
  expandedNodes,
  onToggleNode,
  renderItemIcon,
}: TreeNodeProps) {
  const isExpanded = expandedNodes.has(node.id)
  const paddingLeft = `${node.depth * 12 + 8}px`

  if (node.isCategory) {
    // Count dirty children for this category
    const dirtyCount = countDirtyInSubtree(node, dirtyItemIds)

    return (
      <div>
        <button
          type="button"
          onClick={() => onToggleNode(node.id)}
          className="w-full flex items-center gap-2 py-1.5 px-2 text-slate-400 hover:text-slate-200 hover:bg-white/5 transition-colors text-xs"
          style={{ paddingLeft }}
        >
          {isExpanded ? (
            <ChevronDown className="h-3.5 w-3.5 flex-shrink-0" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 flex-shrink-0" />
          )}
          <FolderOpen className="h-3.5 w-3.5 flex-shrink-0 text-slate-500" />
          <span className="truncate flex-1 text-left">{node.label}</span>
          {dirtyCount > 0 && (
            <span
              className="w-2 h-2 bg-amber-500 rounded-full flex-shrink-0"
              title={`${dirtyCount} unsaved`}
            />
          )}
        </button>
        {isExpanded && (
          <div>
            {node.children.map((child) => (
              <TreeNodeComponent
                key={child.id}
                node={child}
                prompts={prompts}
                selectedItemId={selectedItemId}
                onSelectItem={onSelectItem}
                dirtyItemIds={dirtyItemIds}
                expandedNodes={expandedNodes}
                onToggleNode={onToggleNode}
                renderItemIcon={renderItemIcon}
              />
            ))}
          </div>
        )}
      </div>
    )
  }

  // Leaf node (prompt)
  const prompt = prompts.find((p) => p.id === node.itemId)
  const isSelected = selectedItemId === node.itemId
  const isDirty = node.itemId ? dirtyItemIds.has(node.itemId) : false

  return (
    <button
      type="button"
      onClick={() => node.itemId && onSelectItem(node.itemId)}
      className={cn(
        'w-full flex items-center gap-2 py-1.5 px-2 text-left transition-colors text-xs relative',
        isSelected
          ? 'bg-indigo-600/30 text-white'
          : 'text-slate-300 hover:bg-white/5 hover:text-white'
      )}
      style={{ paddingLeft }}
    >
      {renderItemIcon && prompt ? (
        renderItemIcon(prompt)
      ) : (
        <div className="w-3.5 h-3.5 flex-shrink-0" /> // Spacer when no icon
      )}
      <span className="truncate flex-1">{node.label}</span>
      {isDirty && (
        <span
          className="w-2 h-2 bg-amber-500 rounded-full flex-shrink-0"
          title="Unsaved changes"
        />
      )}
    </button>
  )
}
