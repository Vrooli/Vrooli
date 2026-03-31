/**
 * SavedSetsPanel - List view of saved copy sets for an entity type.
 *
 * Shows previously copied selections sorted by frequency or recency.
 * Allows applying, editing, renaming, and deleting sets.
 */

import { useState, useMemo, useCallback } from 'react'
import { Clock, TrendingUp } from 'lucide-react'
import { cn } from '@/lib/utils'
import { loadCopySets, deleteCopySet, renameCopySet, DISPLAY_LIMIT } from '@/lib/copySetStorage'
import type { CopySetEntry } from '@/lib/copySetStorage'
import type { CombineEntityType } from '@/stores/combineStore'
import { SavedSetEntry } from './SavedSetEntry'

type SortMode = 'frequency' | 'recency'

interface SavedSetsPanelProps {
  entityType: CombineEntityType
  onApplySet: (ids: string[]) => void
  onEditSet: (entry: CopySetEntry) => void
  entityLookup: Map<string, string>
  /** Incremented externally to trigger a re-read from localStorage. */
  refreshKey?: number
}

export function SavedSetsPanel({
  entityType,
  onApplySet,
  onEditSet,
  entityLookup,
  refreshKey = 0,
}: SavedSetsPanelProps) {
  const [sortMode, setSortMode] = useState<SortMode>('frequency')
  const [localRefresh, setLocalRefresh] = useState(0)

  // Load and sort entries
  const entries = useMemo(() => {
    // Dependencies: entityType, sortMode, refreshKey, localRefresh
    void refreshKey
    void localRefresh
    const all = loadCopySets(entityType)
    const sorted = [...all].sort((a, b) => {
      if (sortMode === 'frequency') {
        // Higher count first, tie-break by recency
        if (b.copyCount !== a.copyCount) return b.copyCount - a.copyCount
        return b.lastCopiedAt.localeCompare(a.lastCopiedAt)
      }
      // Recency: most recent first
      return b.lastCopiedAt.localeCompare(a.lastCopiedAt)
    })
    return sorted.slice(0, DISPLAY_LIMIT)
  }, [entityType, sortMode, refreshKey, localRefresh])

  const handleDelete = useCallback((entryId: string) => {
    deleteCopySet(entityType, entryId)
    setLocalRefresh((n) => n + 1)
  }, [entityType])

  const handleRename = useCallback((entryId: string, name: string | null) => {
    renameCopySet(entityType, entryId, name)
    setLocalRefresh((n) => n + 1)
  }, [entityType])

  if (entries.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 px-4 py-8 text-center">
        <span className="text-sm text-muted-foreground">No saved sets yet</span>
        <span className="text-xs text-muted-foreground/70">
          Copy a selection to start building your library
        </span>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Sort toggle */}
      <div className="flex items-center gap-1 px-3 py-2 border-b border-border">
        <span className="text-[10px] text-muted-foreground mr-1">Sort:</span>
        <button
          type="button"
          onClick={() => setSortMode('frequency')}
          className={cn(
            'flex items-center gap-1 px-2 py-0.5 text-[10px] rounded transition-colors',
            sortMode === 'frequency'
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-muted-foreground hover:text-foreground'
          )}
          title="Sort by most used"
        >
          <TrendingUp className="h-3 w-3" />
          Most used
        </button>
        <button
          type="button"
          onClick={() => setSortMode('recency')}
          className={cn(
            'flex items-center gap-1 px-2 py-0.5 text-[10px] rounded transition-colors',
            sortMode === 'recency'
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-muted-foreground hover:text-foreground'
          )}
          title="Sort by most recent"
        >
          <Clock className="h-3 w-3" />
          Recent
        </button>
      </div>

      {/* Entries list */}
      <div className="flex-1 overflow-y-auto">
        {entries.map((entry) => (
          <SavedSetEntry
            key={entry.id}
            entry={entry}
            entityLookup={entityLookup}
            onApply={() => onApplySet(entry.ids)}
            onEdit={() => onEditSet(entry)}
            onDelete={() => handleDelete(entry.id)}
            onRename={(name) => handleRename(entry.id, name)}
          />
        ))}
      </div>
    </div>
  )
}
