/**
 * TopicTreeView - Hierarchical tree visualization for topics.
 *
 * Displays topics in an indented tree showing parent-child relationships.
 * Clicking a topic selects it for editing.
 */

import { useMemo, useState, useCallback } from 'react'
import { ChevronRight, ChevronDown, Layers } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Topic } from '@/lib/schemas'

interface TopicTreeNode {
  topic: Topic
  children: TopicTreeNode[]
}

interface TopicTreeViewProps {
  topics: Topic[]
  selectedTopicId: string | null
  onSelectTopic: (id: string) => void
  className?: string
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
    const node = topicMap.get(topic.id)!
    if (topic.parentTopicId && topicMap.has(topic.parentTopicId)) {
      topicMap.get(topic.parentTopicId)!.children.push(node)
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
}

function TreeNodeRow({
  node,
  depth,
  selectedTopicId,
  expandedIds,
  onToggleExpand,
  onSelectTopic,
}: TreeNodeRowProps) {
  const hasChildren = node.children.length > 0
  const isExpanded = expandedIds.has(node.topic.id)
  const isSelected = selectedTopicId === node.topic.id

  return (
    <>
      <button
        type="button"
        onClick={() => onSelectTopic(node.topic.id)}
        className={cn(
          'w-full flex items-center gap-1.5 py-1.5 pr-3 text-left text-sm',
          'hover:bg-muted/50 transition-colors',
          isSelected && 'bg-primary/10 text-primary'
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

        {/* Icon */}
        {node.topic.icon ? (
          <span className="text-sm flex-shrink-0">{node.topic.icon}</span>
        ) : (
          <Layers className="h-3.5 w-3.5 text-muted-foreground flex-shrink-0" />
        )}

        {/* Name */}
        <span className="truncate">{node.topic.name}</span>

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
        />
      ))}
    </div>
  )
}
