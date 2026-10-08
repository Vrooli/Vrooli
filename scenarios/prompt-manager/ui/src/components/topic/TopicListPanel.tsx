/**
 * TopicListPanel - Panel for listing and managing topics.
 *
 * Displays topics in a list view with:
 * - Name and description
 * - Parent topic indicator
 * - Skill count badge
 * - Search/filter support
 * - Create action
 * - Selection mode with checkboxes
 */

import { useState, useMemo } from 'react'
import { Plus, Layers, ChevronRight } from 'lucide-react'
import { CollectionList } from '@vrooli/react-component-library/CollectionList/1.0.0'
import type { RowAction } from '@vrooli/react-component-library/useCollection/1'
import { cn } from '@/lib/utils'
import { useTopics } from '@/hooks/useTopicData'
import type { DetailMode } from '@/types/filterSort'
import type { Topic } from '@/lib/schemas'

interface TopicListPanelProps {
  selectedTopicId: string | null
  onSelectTopic: (id: string) => void
  /** Filter topics by name */
  searchQuery?: string
  className?: string
  /** Detail level for metadata display */
  detailMode?: DetailMode
  /** Selection mode: show checkboxes and toggle instead of navigate */
  isSelectMode?: boolean
  /** IDs currently selected (for checkbox state) */
  selectedIds?: Set<string>
  /** Called when an item is toggled in selection mode */
  onToggleSelection?: (id: string) => void
}

function topicActions(onOpen: (topic: Topic) => void): RowAction<Topic>[] {
  return [{ id: 'open', label: 'Open', shortcut: 'Enter', onSelect: (rows) => { const topic = rows[0]; if (topic) onOpen(topic) } }]
}

/**
 * Topic list panel for the sidebar.
 */
export function TopicListPanel({
  selectedTopicId,
  onSelectTopic,
  searchQuery,
  className,
  detailMode = 'full',
  isSelectMode,
  selectedIds,
  onToggleSelection,
}: TopicListPanelProps) {
  const { topics, isLoading, isError, createTopic } = useTopics()
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

  const actions = useMemo(() => topicActions((topic) => onSelectTopic(topic.id)), [onSelectTopic])
  const syncSelection = (keys: string[]) => {
    if (!onToggleSelection) return
    const next = new Set(keys)
    selectedIds?.forEach((id) => { if (!next.has(id)) onToggleSelection(id) })
    next.forEach((id) => { if (!selectedIds?.has(id)) onToggleSelection(id) })
  }

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
    <div className={cn('flex flex-col min-h-0', className)}>
      <div className="min-h-0 flex-1 overflow-y-auto py-1">
        <CollectionList
          items={filteredTopics}
          getKey={(topic) => topic.id}
          label="Topics"
          virtualize
          height="100%"
          selection={{ mode: isSelectMode ? 'multi' : 'none', selected: selectedIds ? [...selectedIds] : undefined, onChange: syncSelection }}
          onOpen={(topic) => onSelectTopic(topic.id)}
          actions={actions}
          empty={topics.length === 0 ? (
            <div className="px-3 py-8 text-center"><Layers className="mx-auto mb-2 h-8 w-8 text-muted-foreground" /><p className="mb-4 text-xs text-muted-foreground">No topics yet</p><button type="button" onClick={() => void handleCreateTopic()} className="text-xs text-primary hover:underline" disabled={isCreating}>Create your first topic</button></div>
          ) : (
            <div className="px-3 py-8 text-center"><Layers className="mx-auto mb-2 h-8 w-8 text-muted-foreground opacity-60" /><p className="text-xs text-muted-foreground">No matching topics</p></div>
          )}
          renderItem={(topic) => <div className={cn('flex w-full items-center gap-3 px-3 py-2 text-left', !isSelectMode && selectedTopicId === topic.id && 'bg-primary/10')} data-topic-id={topic.id}>
            <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-muted">{topic.icon ? <span className="text-sm">{topic.icon}</span> : <Layers className="h-3.5 w-3.5 text-muted-foreground" />}</div>
            <div className="min-w-0 flex-1"><p className="truncate text-sm font-medium text-foreground">{topic.name}</p>{detailMode === 'full' && <div className="flex items-center gap-1.5 text-xs text-muted-foreground">{topic.parentTopicId && topicNameMap.has(topic.parentTopicId) && <span className="flex items-center gap-0.5 truncate"><ChevronRight className="h-3 w-3 shrink-0" /><span className="truncate">{topicNameMap.get(topic.parentTopicId)}</span></span>}{topic.skills.length > 0 && <span className="shrink-0 rounded-full bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium text-primary">{topic.skills.length} skill{topic.skills.length !== 1 ? 's' : ''}</span>}</div>}</div>
          </div>}
          className="h-full w-full"
        />
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
