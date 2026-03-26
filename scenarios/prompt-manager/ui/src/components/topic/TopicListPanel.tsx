/**
 * TopicListPanel - Panel for listing and managing topics.
 *
 * Displays topics in a list view with:
 * - Name and description
 * - Parent topic indicator
 * - Skill count badge
 * - Search/filter support
 * - Create and delete actions
 * - Selection mode with checkboxes
 */

import { useState, useMemo, useCallback } from 'react'
import { Plus, Layers, Trash2, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useTopics } from '@/hooks/useTopicData'

interface TopicListPanelProps {
  selectedTopicId: string | null
  onSelectTopic: (id: string) => void
  /** Filter topics by name */
  searchQuery?: string
  className?: string
  /** Selection mode: show checkboxes and toggle instead of navigate */
  isSelectMode?: boolean
  /** IDs currently selected (for checkbox state) */
  selectedIds?: Set<string>
  /** Called when an item is toggled in selection mode */
  onToggleSelection?: (id: string) => void
}

/**
 * Topic list panel for the sidebar.
 */
export function TopicListPanel({
  selectedTopicId,
  onSelectTopic,
  searchQuery,
  className,
  isSelectMode,
  selectedIds,
  onToggleSelection,
}: TopicListPanelProps) {
  const { topics, isLoading, isError, createTopic, deleteTopic } = useTopics()
  const [isCreating, setIsCreating] = useState(false)

  const filteredTopics = useMemo(() => {
    if (!searchQuery) return topics
    const lower = searchQuery.toLowerCase()
    return topics.filter(
      (t) =>
        t.name.toLowerCase().includes(lower) ||
        t.description.toLowerCase().includes(lower)
    )
  }, [topics, searchQuery])

  // Build a lookup for parent topic names
  const topicNameMap = useMemo(() => {
    const map = new Map<string, string>()
    for (const t of topics) {
      map.set(t.id, t.name)
    }
    return map
  }, [topics])

  const handleCreateTopic = async () => {
    setIsCreating(true)
    try {
      const name = `Topic ${topics.length + 1}`
      const newTopic = await createTopic({ name })
      onSelectTopic(newTopic.id)
    } finally {
      setIsCreating(false)
    }
  }

  const handleDeleteTopic = async (e: React.MouseEvent, id: string) => {
    e.stopPropagation()
    await deleteTopic(id)
  }

  const handleItemClick = useCallback((id: string) => {
    if (isSelectMode && onToggleSelection) {
      onToggleSelection(id)
    } else {
      onSelectTopic(id)
    }
  }, [isSelectMode, onToggleSelection, onSelectTopic])

  if (isLoading) {
    return (
      <div className={cn('flex items-center justify-center py-8', className)}>
        <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className={cn('px-3 py-8 text-center', className)}>
        <p className="text-sm text-destructive">Failed to load topics</p>
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col h-full', className)}>
      {/* Topic list */}
      <div className="flex-1 overflow-y-auto py-1">
        {topics.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <Layers className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
            <p className="text-xs text-muted-foreground mb-4">No topics yet</p>
            <button
              type="button"
              onClick={() => void handleCreateTopic()}
              className="text-xs text-primary hover:underline"
              disabled={isCreating}
            >
              Create your first topic
            </button>
          </div>
        ) : filteredTopics.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <Layers className="h-8 w-8 mx-auto mb-2 text-muted-foreground opacity-60" />
            <p className="text-xs text-muted-foreground">No matching topics</p>
          </div>
        ) : (
          filteredTopics.map((topic) => (
            <button
              key={topic.id}
              type="button"
              onClick={() => handleItemClick(topic.id)}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2 text-left group',
                'hover:bg-muted/50 transition-colors',
                !isSelectMode && selectedTopicId === topic.id && 'bg-primary/10',
                isSelectMode && selectedIds?.has(topic.id) && 'bg-primary/10'
              )}
              data-topic-id={topic.id}
            >
              {/* Selection checkbox */}
              {isSelectMode && (
                <div className="flex-shrink-0">
                  <div
                    className={cn(
                      'h-4 w-4 rounded border transition-colors',
                      selectedIds?.has(topic.id)
                        ? 'bg-primary border-primary'
                        : 'border-border bg-background'
                    )}
                  >
                    {selectedIds?.has(topic.id) && (
                      <svg viewBox="0 0 16 16" className="h-4 w-4 text-primary-foreground" fill="currentColor">
                        <path d="M12.207 4.793a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0l-2-2a1 1 0 011.414-1.414L6.5 9.086l4.293-4.293a1 1 0 011.414 0z" />
                      </svg>
                    )}
                  </div>
                </div>
              )}

              {/* Topic icon */}
              <div className="flex-shrink-0 w-7 h-7 rounded-md bg-muted flex items-center justify-center">
                {topic.icon ? (
                  <span className="text-sm">{topic.icon}</span>
                ) : (
                  <Layers className="h-3.5 w-3.5 text-muted-foreground" />
                )}
              </div>

              {/* Topic info */}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground truncate">
                  {topic.name}
                </p>
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  {topic.parentTopicId && topicNameMap.has(topic.parentTopicId) && (
                    <span className="flex items-center gap-0.5 truncate">
                      <ChevronRight className="h-3 w-3 flex-shrink-0" />
                      <span className="truncate">{topicNameMap.get(topic.parentTopicId)}</span>
                    </span>
                  )}
                  {topic.skills.length > 0 && (
                    <span className="flex-shrink-0 px-1.5 py-0.5 rounded-full bg-primary/10 text-primary text-[10px] font-medium">
                      {topic.skills.length} skill{topic.skills.length !== 1 ? 's' : ''}
                    </span>
                  )}
                </div>
              </div>

              {/* Actions (hidden in select mode) */}
              {!isSelectMode && (
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    type="button"
                    onClick={(e) => void handleDeleteTopic(e, topic.id)}
                    className="p-1 rounded hover:bg-destructive/20 text-muted-foreground hover:text-destructive transition-colors"
                    title="Delete topic"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              )}
            </button>
          ))
        )}
      </div>

      {/* Footer - New topic button (hidden in select mode) */}
      {!isSelectMode && (
        <div className="flex-shrink-0 px-3 py-3 border-t border-border">
          <button
            type="button"
            onClick={() => void handleCreateTopic()}
            disabled={isCreating}
            className={cn(
              'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm',
              'bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors',
              isCreating && 'opacity-50 cursor-not-allowed'
            )}
          >
            <Plus className="h-4 w-4" />
            New Topic
          </button>
        </div>
      )}
    </div>
  )
}
