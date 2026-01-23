/**
 * TreeNode - Recursive tree node component.
 *
 * Renders a single node in the skill tree, handling:
 * - Category nodes (expandable)
 * - Leaf nodes (selectable skills)
 * - Dirty indicators
 * - Checkbox selection for skill selection mode
 */

import { type ReactNode } from 'react'
import { ChevronRight, ChevronDown, FolderOpen, Check, Minus } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TreeNode as TreeNodeType } from '@/types/editor'
import type { Skill } from '@/types'
import { countDirtyInSubtree } from '@/services/treeService'
import { useEditorStore } from '@/stores/editorStore'

type SelectionState = 'none' | 'partial' | 'all'

interface TreeNodeProps {
  node: TreeNodeType
  skills: Skill[]
  selectedItemId: string | null
  onSelectItem: (id: string) => void
  dirtyItemIds: Set<string>
  expandedNodes: Set<string>
  onToggleNode: (nodeId: string) => void
  renderItemIcon?: (skill: Skill) => ReactNode
  // Skill selection mode props
  showCheckbox?: boolean
  selectionState?: SelectionState
  onCheckboxChange?: (node: TreeNodeType) => void
  getSelectionState?: (node: TreeNodeType) => SelectionState
  // Context menu props
  onCategoryContextMenu?: (node: TreeNodeType, x: number, y: number) => void
  onSkillContextMenu?: (skillId: string, skillName: string, x: number, y: number) => void
}

/**
 * Checkbox component for skill selection.
 */
function SelectionCheckbox({
  state,
  onClick,
  className,
}: {
  state: SelectionState
  onClick: (e: React.MouseEvent) => void
  className?: string
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'w-4 h-4 rounded border flex items-center justify-center flex-shrink-0 transition-colors',
        state === 'all'
          ? 'bg-primary border-primary'
          : state === 'partial'
            ? 'bg-primary/50 border-primary'
            : 'border-muted-foreground/30 hover:border-primary/50',
        className
      )}
    >
      {state === 'all' && <Check className="h-3 w-3 text-primary-foreground" />}
      {state === 'partial' && <Minus className="h-3 w-3 text-primary-foreground" />}
    </button>
  )
}

/**
 * Recursive tree node component.
 */
export function TreeNodeComponent({
  node,
  skills,
  selectedItemId,
  onSelectItem,
  dirtyItemIds,
  expandedNodes,
  onToggleNode,
  renderItemIcon,
  showCheckbox = false,
  onCheckboxChange,
  getSelectionState,
  onCategoryContextMenu,
  onSkillContextMenu,
}: TreeNodeProps) {
  const isExpanded = expandedNodes.has(node.id)
  const paddingLeft = `${node.depth * 12 + 8}px`
  const selectionState = getSelectionState?.(node) ?? 'none'

  const handleCheckboxClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    onCheckboxChange?.(node)
  }

  if (node.isCategory) {
    // Count dirty children for this category
    const dirtyCount = countDirtyInSubtree(node, dirtyItemIds)

    const handleContextMenu = (e: React.MouseEvent) => {
      if (onCategoryContextMenu && !showCheckbox) {
        e.preventDefault()
        onCategoryContextMenu(node, e.clientX, e.clientY)
      }
    }

    return (
      <div>
        <button
          type="button"
          onClick={() => onToggleNode(node.id)}
          onContextMenu={handleContextMenu}
          className="w-full flex items-center gap-2 py-1.5 px-2 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors text-xs"
          style={{ paddingLeft }}
        >
          {showCheckbox && (
            <SelectionCheckbox
              state={selectionState}
              onClick={handleCheckboxClick}
            />
          )}
          {isExpanded ? (
            <ChevronDown className="h-3.5 w-3.5 flex-shrink-0" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 flex-shrink-0" />
          )}
          <FolderOpen className="h-3.5 w-3.5 flex-shrink-0 text-muted-foreground" />
          <span className="truncate flex-1 text-left">{node.label}</span>
          {dirtyCount > 0 && !showCheckbox && (
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
                skills={skills}
                selectedItemId={selectedItemId}
                onSelectItem={onSelectItem}
                dirtyItemIds={dirtyItemIds}
                expandedNodes={expandedNodes}
                onToggleNode={onToggleNode}
                renderItemIcon={renderItemIcon}
                showCheckbox={showCheckbox}
                onCheckboxChange={onCheckboxChange}
                getSelectionState={getSelectionState}
                onCategoryContextMenu={onCategoryContextMenu}
                onSkillContextMenu={onSkillContextMenu}
              />
            ))}
          </div>
        )}
      </div>
    )
  }

  // Leaf node (skill)
  const skill = skills.find((p) => p.id === node.itemId)
  const isSelected = selectedItemId === node.itemId
  const isDirty = node.itemId ? dirtyItemIds.has(node.itemId) : false

  // Get live form state for display name override
  const formState = useEditorStore((state) =>
    node.itemId ? state.getFormState(node.itemId) : null
  )
  const displayLabel = formState?.name || node.label

  // In checkbox mode, clicking the row toggles the checkbox
  const handleRowClick = () => {
    if (showCheckbox) {
      onCheckboxChange?.(node)
    } else if (node.itemId) {
      onSelectItem(node.itemId)
    }
  }

  const handleSkillContextMenu = (e: React.MouseEvent) => {
    if (onSkillContextMenu && !showCheckbox && node.itemId) {
      e.preventDefault()
      onSkillContextMenu(node.itemId, node.label, e.clientX, e.clientY)
    }
  }

  return (
    <button
      type="button"
      onClick={handleRowClick}
      onContextMenu={handleSkillContextMenu}
      className={cn(
        'w-full flex items-center gap-2 py-1.5 px-2 text-left transition-colors text-xs relative',
        showCheckbox
          ? selectionState === 'all'
            ? 'bg-primary/10 text-foreground'
            : 'text-muted-foreground hover:bg-muted hover:text-foreground'
          : isSelected
            ? 'bg-primary/30 text-foreground'
            : 'text-muted-foreground hover:bg-muted hover:text-foreground'
      )}
      style={{ paddingLeft }}
    >
      {showCheckbox && (
        <SelectionCheckbox
          state={selectionState}
          onClick={handleCheckboxClick}
        />
      )}
      {!showCheckbox && renderItemIcon && skill ? (
        renderItemIcon(skill)
      ) : !showCheckbox ? (
        <div className="w-3.5 h-3.5 flex-shrink-0" /> // Spacer when no icon
      ) : null}
      <span className="truncate flex-1">{displayLabel}</span>
      {isDirty && !showCheckbox && (
        <span
          className="w-2 h-2 bg-amber-500 rounded-full flex-shrink-0"
          title="Unsaved changes"
        />
      )}
    </button>
  )
}
