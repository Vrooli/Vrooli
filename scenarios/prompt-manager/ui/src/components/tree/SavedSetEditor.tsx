/**
 * SavedSetEditor - Edit a saved copy set's name and contents.
 *
 * Shown in-place in the sidebar, replacing the saved sets list.
 * Allows renaming, adding/removing entities via search, and saving.
 */

import { useState, useMemo } from 'react'
import { X, Plus, Search } from 'lucide-react'
import { cn } from '@/lib/utils'
import { updateCopySetIds, renameCopySet } from '@/lib/copySetStorage'
import type { CopySetEntry } from '@/lib/copySetStorage'
import type { CombineEntityType } from '@/stores/combineStore'

interface SavedSetEditorProps {
  entry: CopySetEntry
  entityType: CombineEntityType
  allEntities: Array<{ id: string; name: string }>
  onSave: () => void
  onCancel: () => void
}

export function SavedSetEditor({
  entry,
  entityType,
  allEntities,
  onSave,
  onCancel,
}: SavedSetEditorProps) {
  const [name, setName] = useState(entry.name ?? '')
  const [selectedIds, setSelectedIds] = useState<string[]>([...entry.ids])
  const [searchQuery, setSearchQuery] = useState('')
  const [error, setError] = useState<string | null>(null)

  // Entities not currently in the set, filtered by search
  const availableEntities = useMemo(() => {
    const selected = new Set(selectedIds)
    const available = allEntities.filter((e) => !selected.has(e.id))
    if (!searchQuery.trim()) return available.slice(0, 20)
    const query = searchQuery.toLowerCase()
    return available
      .filter((e) => e.name.toLowerCase().includes(query) || e.id.toLowerCase().includes(query))
      .slice(0, 20)
  }, [allEntities, selectedIds, searchQuery])

  // Entity name lookup for chips
  const entityMap = useMemo(() => {
    const map = new Map<string, string>()
    for (const e of allEntities) {
      map.set(e.id, e.name)
    }
    return map
  }, [allEntities])

  const handleRemove = (id: string) => {
    setSelectedIds((prev) => prev.filter((i) => i !== id))
    setError(null)
  }

  const handleAdd = (id: string) => {
    setSelectedIds((prev) => [...prev, id])
    setSearchQuery('')
    setError(null)
  }

  const handleSave = () => {
    if (selectedIds.length === 0) {
      setError('Set must contain at least one item')
      return
    }

    // Update name
    const trimmedName = name.trim() || null
    if (trimmedName !== entry.name) {
      renameCopySet(entityType, entry.id, trimmedName)
    }

    // Update IDs
    const ok = updateCopySetIds(entityType, entry.id, selectedIds)
    if (!ok) {
      setError('Another set with these exact items already exists')
      return
    }

    onSave()
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="px-3 py-2 border-b border-border">
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          Edit Set
        </span>
      </div>

      <div className="flex-1 overflow-y-auto px-3 py-3 flex flex-col gap-3">
        {/* Name input */}
        <div className="flex flex-col gap-1">
          <label className="text-[10px] text-muted-foreground uppercase tracking-wider">Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="h-7 px-2 text-xs rounded border border-border bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
            placeholder="Name this set (optional)"
          />
        </div>

        {/* Current items */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[10px] text-muted-foreground uppercase tracking-wider">
            Items ({selectedIds.length})
          </label>
          <div className="flex flex-wrap gap-1.5">
            {selectedIds.map((id) => (
              <span
                key={id}
                className="inline-flex items-center gap-1 px-2 py-1 bg-muted text-foreground text-xs rounded-md"
              >
                {entityMap.get(id) ?? id}
                <button
                  type="button"
                  onClick={() => handleRemove(id)}
                  className="p-0.5 rounded hover:bg-destructive/20 hover:text-destructive transition-colors"
                  title="Remove"
                >
                  <X className="h-2.5 w-2.5" />
                </button>
              </span>
            ))}
          </div>
        </div>

        {/* Add entities */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[10px] text-muted-foreground uppercase tracking-wider">Add items</label>
          <div className="relative">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="h-7 w-full pl-7 pr-2 text-xs rounded border border-border bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
              placeholder="Search to add..."
            />
          </div>
          {(searchQuery.trim() || availableEntities.length <= 10) && availableEntities.length > 0 && (
            <div className="flex flex-col max-h-32 overflow-y-auto rounded border border-border bg-background">
              {availableEntities.map((entity) => (
                <button
                  key={entity.id}
                  type="button"
                  onClick={() => handleAdd(entity.id)}
                  className="flex items-center gap-2 px-2 py-1.5 text-xs text-foreground hover:bg-muted transition-colors text-left"
                >
                  <Plus className="h-3 w-3 text-muted-foreground shrink-0" />
                  <span className="truncate">{entity.name}</span>
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Error */}
        {error && (
          <span className="text-xs text-red-400">{error}</span>
        )}
      </div>

      {/* Footer actions */}
      <div className="flex items-center gap-2 px-3 py-2 border-t border-border">
        <button
          type="button"
          onClick={handleSave}
          className={cn(
            'flex-1 px-3 py-1.5 text-xs rounded transition-colors',
            'bg-primary text-primary-foreground hover:bg-primary/90'
          )}
        >
          Save
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="flex-1 px-3 py-1.5 text-xs rounded bg-muted text-muted-foreground hover:text-foreground transition-colors"
        >
          Cancel
        </button>
      </div>
    </div>
  )
}
