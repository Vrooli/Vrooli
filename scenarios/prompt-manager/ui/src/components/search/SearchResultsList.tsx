/**
 * SearchResultsList - Renders search results for any entity type.
 *
 * Supports optional selection mode with checkboxes and discover source badges.
 * Used inline in the sidebar for AI search results and in select/copy flows.
 */

import { ExternalLink } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AISearchResult, AIAgentSearchResult, AITeamSearchResult, TopicMatchResult, DiscoverResult } from '@/lib/schemas'

type EntityType = 'skills' | 'agents' | 'teams' | 'topics'

interface SearchResultsListProps {
  entityType: EntityType
  // Results per entity type (only the active one is populated)
  skillResults?: AISearchResult[]
  agentResults?: AIAgentSearchResult[]
  teamResults?: AITeamSearchResult[]
  topicResults?: TopicMatchResult[]
  discoverResults?: DiscoverResult[]
  // Selection mode
  isSelectMode: boolean
  selectedIds: Set<string>
  onToggleSelection: (id: string, contentChars?: number) => void
  onNavigate: (id: string) => void
  // Discover mode (skills only)
  discoverMode?: boolean
  // Budget: ids that exceed budget
  overBudgetIds?: Set<string>
  /** Use compact padding for sidebar display */
  compact?: boolean
}

export function SearchResultsList({
  entityType,
  skillResults,
  agentResults,
  teamResults,
  topicResults,
  discoverResults,
  isSelectMode,
  selectedIds,
  onToggleSelection,
  onNavigate,
  discoverMode,
  overBudgetIds,
  compact,
}: SearchResultsListProps) {
  return (
    <ul className="divide-y divide-border">
      {entityType === 'skills' && discoverMode && discoverResults?.map((result) => (
        <ResultRow
          key={result.id}
          id={result.id}
          isSelectMode={isSelectMode}
          isSelected={selectedIds.has(result.id)}
          isOverBudget={overBudgetIds?.has(result.id)}
          compact={compact}
          onToggle={() => onToggleSelection(result.id, result.contentChars)}
          onNavigate={() => onNavigate(result.id)}
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="font-medium text-foreground truncate">{result.name}</span>
            </div>
            {result.description && (
              <p className="mt-1 text-xs text-muted-foreground line-clamp-2">{result.description}</p>
            )}
            <div className="mt-1.5 flex flex-wrap items-center gap-1">
              <SourceBadge source={result.source} topicId={result.topicId} topicDepth={result.topicDepth} />
              {result.tags.slice(0, 5).map((tag) => (
                <TagPill key={tag} tag={tag} />
              ))}
              {result.tags.length > 5 && (
                <span className="text-[10px] text-muted-foreground">+{result.tags.length - 5}</span>
              )}
            </div>
          </div>
          <ScoreBadge scorePercent={result.scorePercent} />
        </ResultRow>
      ))}

      {entityType === 'skills' && !discoverMode && skillResults?.map((result) => (
        <ResultRow
          key={result.id}
          id={result.id}
          isSelectMode={isSelectMode}
          isSelected={selectedIds.has(result.id)}
          compact={compact}
          onToggle={() => onToggleSelection(result.id)}
          onNavigate={() => onNavigate(result.id)}
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="font-medium text-foreground truncate">{result.name}</span>
              <span className="text-xs text-muted-foreground flex-shrink-0">{result.folder}/</span>
            </div>
            {result.description && (
              <p className="mt-1 text-xs text-muted-foreground line-clamp-2">{result.description}</p>
            )}
            {result.tags.length > 0 && (
              <div className="mt-1.5 flex flex-wrap gap-1">
                {result.tags.slice(0, 5).map((tag) => (
                  <TagPill key={tag} tag={tag} />
                ))}
                {result.tags.length > 5 && (
                  <span className="text-[10px] text-muted-foreground">+{result.tags.length - 5}</span>
                )}
              </div>
            )}
          </div>
          <ScoreBadge scorePercent={result.scorePercent} />
        </ResultRow>
      ))}

      {entityType === 'agents' && agentResults?.map((result) => (
        <ResultRow
          key={result.id}
          id={result.id}
          isSelectMode={isSelectMode}
          isSelected={selectedIds.has(result.id)}
          compact={compact}
          onToggle={() => onToggleSelection(result.id)}
          onNavigate={() => onNavigate(result.id)}
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="font-medium text-foreground truncate">{result.displayName}</span>
              <span className="text-xs text-muted-foreground flex-shrink-0">{result.status}</span>
            </div>
            {result.description && (
              <p className="mt-1 text-xs text-muted-foreground line-clamp-2">{result.description}</p>
            )}
            {result.tags.length > 0 && (
              <div className="mt-1.5 flex flex-wrap gap-1">
                {result.tags.slice(0, 5).map((tag) => (
                  <TagPill key={tag} tag={tag} />
                ))}
              </div>
            )}
          </div>
          <ScoreBadge scorePercent={result.scorePercent} />
        </ResultRow>
      ))}

      {entityType === 'teams' && teamResults?.map((result) => (
        <ResultRow
          key={result.id}
          id={result.id}
          isSelectMode={isSelectMode}
          isSelected={selectedIds.has(result.id)}
          compact={compact}
          onToggle={() => onToggleSelection(result.id)}
          onNavigate={() => onNavigate(result.id)}
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="font-medium text-foreground truncate">{result.displayName}</span>
              <span className="text-xs text-muted-foreground flex-shrink-0">
                {result.enabled ? 'enabled' : 'disabled'} · {result.memberCount} member{result.memberCount !== 1 ? 's' : ''}
              </span>
            </div>
            {result.mission && (
              <p className="mt-1 text-xs text-muted-foreground line-clamp-2">{result.mission}</p>
            )}
          </div>
          <ScoreBadge scorePercent={result.scorePercent} />
        </ResultRow>
      ))}

      {entityType === 'topics' && topicResults?.map((result) => (
        <ResultRow
          key={result.topic.id}
          id={result.topic.id}
          isSelectMode={isSelectMode}
          isSelected={selectedIds.has(result.topic.id)}
          compact={compact}
          onToggle={() => onToggleSelection(result.topic.id)}
          onNavigate={() => onNavigate(result.topic.id)}
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              {result.topic.icon && (
                <span className="text-sm flex-shrink-0">{result.topic.icon}</span>
              )}
              <span className="font-medium text-foreground truncate">{result.topic.name}</span>
              {result.topic.parentTopicId && (
                <span className="text-xs text-muted-foreground flex-shrink-0">child topic</span>
              )}
              {result.topic.skills.length > 0 && (
                <span className="text-xs text-muted-foreground flex-shrink-0">
                  {result.topic.skills.length} skill{result.topic.skills.length !== 1 ? 's' : ''}
                </span>
              )}
            </div>
            {result.topic.description && (
              <p className="mt-1 text-xs text-muted-foreground line-clamp-2">{result.topic.description}</p>
            )}
          </div>
          <ScoreBadge scorePercent={result.score * 100} />
        </ResultRow>
      ))}
    </ul>
  )
}

// --- Sub-components ---

interface ResultRowProps {
  id: string
  isSelectMode: boolean
  isSelected: boolean
  isOverBudget?: boolean
  compact?: boolean
  onToggle: () => void
  onNavigate: () => void
  children: React.ReactNode
}

function ResultRow({
  isSelectMode,
  isSelected,
  isOverBudget,
  compact,
  onToggle,
  onNavigate,
  children,
}: ResultRowProps) {
  const handleClick = () => {
    if (isSelectMode) {
      onToggle()
    } else {
      onNavigate()
    }
  }

  return (
    <li>
      <button
        type="button"
        onClick={handleClick}
        className={cn(
          'w-full text-left',
          compact ? 'px-3 py-2' : 'px-4 py-3',
          'hover:bg-muted/50 transition-colors',
          'focus:outline-none focus:bg-muted/50',
          isOverBudget && 'opacity-50',
        )}
      >
        <div className="flex items-start justify-between gap-3">
          {isSelectMode && (
            <div className="flex-shrink-0 pt-0.5">
              <div
                className={cn(
                  'h-4 w-4 rounded border transition-colors',
                  isSelected
                    ? 'bg-primary border-primary'
                    : 'border-border bg-background'
                )}
              >
                {isSelected && (
                  <svg viewBox="0 0 16 16" className="h-4 w-4 text-primary-foreground" fill="currentColor">
                    <path d="M12.207 4.793a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0l-2-2a1 1 0 011.414-1.414L6.5 9.086l4.293-4.293a1 1 0 011.414 0z" />
                  </svg>
                )}
              </div>
            </div>
          )}
          {children}
          {isSelectMode && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                onNavigate()
              }}
              className="flex-shrink-0 p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
              title="Go to entity"
            >
              <ExternalLink className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </button>
    </li>
  )
}

function ScoreBadge({ scorePercent }: { scorePercent: number }) {
  const rounded = Math.round(scorePercent)
  return (
    <div className="flex-shrink-0">
      <span
        className={cn(
          'inline-flex items-center px-2 py-1 text-xs font-medium rounded-full',
          rounded >= 70
            ? 'bg-green-500/20 text-green-400'
            : rounded >= 50
              ? 'bg-yellow-500/20 text-yellow-400'
              : 'bg-muted text-muted-foreground'
        )}
      >
        {rounded}%
      </span>
    </div>
  )
}

function SourceBadge({ source, topicId, topicDepth }: { source: string; topicId?: string; topicDepth?: number | null }) {
  if (source === 'topic' && topicId) {
    return (
      <span className="px-1.5 py-0.5 text-[10px] bg-primary/10 text-primary rounded">
        via topic: {topicId}{topicDepth != null ? ` (depth ${topicDepth})` : ''}
      </span>
    )
  }
  return (
    <span className="px-1.5 py-0.5 text-[10px] bg-muted text-muted-foreground rounded">
      direct match
    </span>
  )
}

function TagPill({ tag }: { tag: string }) {
  return (
    <span className="px-1.5 py-0.5 text-[10px] bg-muted text-muted-foreground rounded">
      {tag}
    </span>
  )
}
