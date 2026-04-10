/**
 * TopicTreeView - Hierarchical tree visualization for topics.
 *
 * Displays topics in an indented tree showing parent-child relationships.
 * Clicking a topic selects it for editing. Supports selection mode with checkboxes.
 */

import { useMemo, useState, useCallback } from 'react'
import { ChevronRight, ChevronDown, Layers, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Topic } from '@/lib/schemas'
import type { DetailMode } from '@/types/filterSort'

interface TopicTreeNode {
  topic: Topic
  children: TopicTreeNode[]
}

interface TopicTreeViewProps {
  topics: Topic[]
  selectedTopicId: string | null
  onSelectTopic: (id: string) => void
  className?: string
  detailMode?: DetailMode
  /** Selection mode: show checkboxes and toggle instead of navigate */
  isSelectMode?: boolean
  /** IDs currently selected (for checkbox state) */
  selectedIds?: Set<string>
  /** Called when an item is toggled in selection mode */
  onToggleSelection?: (id: string) => void
}

/**
 * Build a tree structure from flat topic list.
 */
function buildTree(topics: Topic[]): TopicTreeNode[] {
  const topicMap = new Map<string, TopicTreeNode>()
  const roots: TopicTreeNode[] = []

  // Create nodes
  for (const topic of topics) {
    topicMap.set(topic.id, { topic, children: [] })
  }

  // Build parent-child relationships
  for (const topic of topics) {
    const node = topicMap.get(topic.id)
    if (!node) continue
    if (topic.parentTopicId && topicMap.has(topic.parentTopicId)) {
      const parentNode = topicMap.get(topic.parentTopicId)
      if (parentNode) parentNode.children.push(node)
    } else {
      roots.push(node)
    }
  }

  // Sort children alphabetically
  const sortNodes = (nodes: TopicTreeNode[]) => {
    nodes.sort((a, b) => a.topic.name.localeCompare(b.topic.name))
    for (const node of nodes) {
      sortNodes(node.children)
    }
  }
  sortNodes(roots)

  return roots
}

interface TreeNodeRowProps {
  node: TopicTreeNode
  depth: number
  selectedTopicId: string | null
  expandedIds: Set<string>
  onToggleExpand: (id: string) => void
  onSelectTopic: (id: string) => void
  detailMode: DetailMode
  isSelectMode: boolean
  selectedIds?: Set<string>
  onToggleSelection?: (id: string) => void
}

function TreeNodeRow({
  node,
  depth,
  selectedTopicId,
  expandedIds,
  onToggleExpand,
  onSelectTopic,
  detailMode,
  isSelectMode,
  selectedIds,
  onToggleSelection,
}: TreeNodeRowProps) {
  const hasChildren = node.children.length > 0
  const isExpanded = expandedIds.has(node.topic.id)
  const isSelected = selectedTopicId === node.topic.id
  const isCombineSelected = isSelectMode && selectedIds?.has(node.topic.id)

  const handleClick = () => {
    if (isSelectMode && onToggleSelection) {
      onToggleSelection(node.topic.id)
    } else {
      onSelectTopic(node.topic.id)
    }
  }

  return (
    <>
      <button
        type="button"
        onClick={handleClick}
        className={cn(
          'w-full flex items-center gap-1.5 py-1.5 pr-3 text-left text-sm',
          'hover:bg-muted/50 transition-colors',
          !isSelectMode && isSelected && 'bg-primary/10 text-primary',
          isCombineSelected && 'bg-primary/10'
        )}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        data-topic-id={node.topic.id}
      >
        {/* Expand/collapse toggle */}
        {hasChildren ? (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation()
              onToggleExpand(node.topic.id)
            }}
            className="p-0.5 rounded hover:bg-muted transition-colors flex-shrink-0"
          >
            {isExpanded ? (
              <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
            )}
          </button>
        ) : (
          <span className="w-[18px] flex-shrink-0" />
        )}

        {/* Selection checkbox */}
        {isSelectMode && (
          <span className={cn(
            'flex-shrink-0 w-4 h-4 rounded border flex items-center justify-center transition-colors',
            isCombineSelected
              ? 'bg-primary border-primary'
              : 'border-muted-foreground/40'
          )}>
            {isCombineSelected && <Check className="h-3 w-3 text-primary-foreground" />}
          </span>
        )}

        {/* Icon */}
        {node.topic.icon ? (
          <span className="text-sm flex-shrink-0">{node.topic.icon}</span>
        ) : (
          <Layers className="h-3.5 w-3.5 text-muted-foreground flex-shrink-0" />
        )}

        {/* Name + optional description */}
        <div className="flex-1 min-w-0">
          <span className="truncate block">{node.topic.name}</span>
          {detailMode === 'full' && node.topic.description && (
            <p className="text-[10px] text-muted-foreground truncate">{node.topic.description}</p>
          )}
        </div>

        {/* Skill count */}
        {node.topic.skills.length > 0 && (
          <span className="ml-auto flex-shrink-0 text-[10px] text-muted-foreground">
            {node.topic.skills.length}
          </span>
        )}
      </button>

      {/* Children */}
      {hasChildren && isExpanded && (
        node.children.map((child) => (
          <TreeNodeRow
            key={child.topic.id}
            node={child}
            depth={depth + 1}
            selectedTopicId={selectedTopicId}
            expandedIds={expandedIds}
            onToggleExpand={onToggleExpand}
            onSelectTopic={onSelectTopic}
            detailMode={detailMode}
            isSelectMode={isSelectMode}
            selectedIds={selectedIds}
            onToggleSelection={onToggleSelection}
          />
        ))
      )}
    </>
  )
}

/**
 * Hierarchical tree view of topics.
 */
export function TopicTreeView({
  topics,
  selectedTopicId,
  onSelectTopic,
  className,
  detailMode = 'compact',
  isSelectMode = false,
  selectedIds,
  onToggleSelection,
}: TopicTreeViewProps) {
  const tree = useMemo(() => buildTree(topics), [topics])

  // Start with all nodes expanded
  const [expandedIds, setExpandedIds] = useState<Set<string>>(() => {
    const ids = new Set<string>()
    for (const t of topics) {
      ids.add(t.id)
    }
    return ids
  })

  const handleToggleExpand = useCallback((id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  if (topics.length === 0) {
    return (
      <div className={cn('px-3 py-8 text-center', className)}>
        <Layers className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
        <p className="text-xs text-muted-foreground">No topics</p>
      </div>
    )
  }

  return (
    <div className={cn('py-1', className)}>
      {tree.map((node) => (
        <TreeNodeRow
          key={node.topic.id}
          node={node}
          depth={0}
          selectedTopicId={selectedTopicId}
          expandedIds={expandedIds}
          onToggleExpand={handleToggleExpand}
          onSelectTopic={onSelectTopic}
          detailMode={detailMode}
          isSelectMode={isSelectMode}
          selectedIds={selectedIds}
          onToggleSelection={onToggleSelection}
        />
      ))}
    </div>
  )
}
