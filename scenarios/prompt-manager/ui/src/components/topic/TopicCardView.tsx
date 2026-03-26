/**
 * TopicCardView — Card grid of topics with metadata and description preview.
 */

import { useMemo } from 'react'
import { Layers, Check, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Topic } from '@/lib/schemas'
import type { DetailMode } from '@/types/filterSort'

interface TopicCardViewProps {
  topics: Topic[]
  selectedTopicId: string | null
  onSelectTopic: (id: string) => void
  detailMode: DetailMode
  className?: string
  /** Selection mode: show checkboxes and toggle instead of navigate */
  isSelectMode?: boolean
  /** IDs currently selected (for checkbox state) */
  selectedIds?: Set<string>
  /** Called when an item is toggled in selection mode */
  onToggleSelection?: (id: string) => void
}

export function TopicCardView({
  topics,
  selectedTopicId,
  onSelectTopic,
  detailMode,
  className,
  isSelectMode = false,
  selectedIds,
  onToggleSelection,
}: TopicCardViewProps) {
  // Build parent name lookup
  const topicNameMap = useMemo(() => {
    const map = new Map<string, string>()
    for (const t of topics) {
      map.set(t.id, t.name)
    }
    return map
  }, [topics])

  if (topics.length === 0) {
    return (
      <div className={cn('flex items-center justify-center py-8 text-xs text-muted-foreground', className)}>
        No topics match your filters
      </div>
    )
  }

  const showDetails = detailMode === 'full'

  return (
    <div
      className={cn('grid grid-cols-2 gap-2 p-2', className)}
      role="listbox"
      data-testid="topic-card-view"
    >
      {topics.map((topic) => {
        const isSelected = topic.id === selectedTopicId
        const isCombineSelected = isSelectMode && selectedIds?.has(topic.id)

        const handleClick = () => {
          if (isSelectMode && onToggleSelection) {
            onToggleSelection(topic.id)
          } else {
            onSelectTopic(topic.id)
          }
        }

        return (
          <button
            key={topic.id}
            type="button"
            role="option"
            aria-selected={isSelectMode ? !!isCombineSelected : isSelected}
            onClick={handleClick}
            className={cn(
              'flex flex-col gap-1 p-2 rounded-lg border text-left transition-colors relative',
              isSelectMode
                ? isCombineSelected
                  ? 'bg-primary/10 border-primary/40'
                  : 'border-border hover:bg-muted hover:border-muted-foreground/20'
                : isSelected
                  ? 'bg-primary/20 border-primary/40'
                  : 'border-border hover:bg-muted hover:border-muted-foreground/20'
            )}
            data-testid="topic-card-item"
            data-topic-id={topic.id}
          >
            {/* Combine checkbox overlay */}
            {isSelectMode && (
              <span className={cn(
                'absolute top-1.5 right-1.5 w-4 h-4 rounded border flex items-center justify-center transition-colors',
                isCombineSelected
                  ? 'bg-primary border-primary'
                  : 'border-muted-foreground/40 bg-background'
              )}>
                {isCombineSelected && <Check className="h-3 w-3 text-primary-foreground" />}
              </span>
            )}

            {/* Icon + Name */}
            <div className="flex items-start gap-1.5 min-w-0">
              <span className="flex-shrink-0 mt-0.5">
                {topic.icon ? (
                  <span className="text-sm">{topic.icon}</span>
                ) : (
                  <Layers className="h-3.5 w-3.5 text-muted-foreground" />
                )}
              </span>
              <span className="text-xs font-medium truncate flex-1 text-foreground">
                {topic.name}
              </span>
            </div>

            {showDetails && (
              <>
                {/* Description preview */}
                {topic.description && (
                  <p className="text-[10px] text-muted-foreground line-clamp-2 leading-tight">
                    {topic.description}
                  </p>
                )}

                {/* Parent topic */}
                {topic.parentTopicId && topicNameMap.has(topic.parentTopicId) && (
                  <div className="flex items-center gap-0.5 text-[10px] text-muted-foreground">
                    <ChevronRight className="h-2.5 w-2.5 flex-shrink-0" />
                    <span className="truncate">{topicNameMap.get(topic.parentTopicId)}</span>
                  </div>
                )}

                {/* Skill count badge */}
                {topic.skills.length > 0 && (
                  <span className="self-start px-1.5 py-0.5 rounded-full bg-primary/10 text-primary text-[9px] font-medium">
                    {topic.skills.length} skill{topic.skills.length !== 1 ? 's' : ''}
                  </span>
                )}
              </>
            )}
          </button>
        )
      })}
    </div>
  )
}
