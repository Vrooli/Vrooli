/**
 * SavedSetEntry - A single row in the saved sets list.
 *
 * Shows set name (or item count), copy frequency, recency,
 * and action buttons for apply, edit, and delete.
 */

import { useState } from 'react'
import { Play, Pencil, X, Bookmark, Check } from 'lucide-react'
import type { CopySetEntry } from '@/lib/copySetStorage'

interface SavedSetEntryProps {
  entry: CopySetEntry
  entityLookup: Map<string, string>
  onApply: () => void
  onEdit: () => void
  onDelete: () => void
  onRename: (name: string | null) => void
}

function formatRelativeTime(isoDate: string): string {
  const diff = Date.now() - new Date(isoDate).getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  return `${Math.floor(days / 30)}mo ago`
}

export function SavedSetEntry({
  entry,
  entityLookup,
  onApply,
  onEdit,
  onDelete,
  onRename,
}: SavedSetEntryProps) {
  const [isRenaming, setIsRenaming] = useState(false)
  const [nameInput, setNameInput] = useState(entry.name ?? '')

  const handleRenameSubmit = () => {
    const trimmed = nameInput.trim()
    onRename(trimmed || null)
    setIsRenaming(false)
  }

  const handleRenameKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleRenameSubmit()
    } else if (e.key === 'Escape') {
      setIsRenaming(false)
      setNameInput(entry.name ?? '')
    }
  }

  // Build a preview of entity names (show first 3, then "+N more")
  const entityNames = entry.ids
    .map((id) => entityLookup.get(id))
    .filter((name): name is string => name !== undefined)
  const previewNames = entityNames.slice(0, 3)
  const remaining = entry.ids.length - previewNames.length

  return (
    <div className="group flex flex-col gap-1 px-3 py-2 border-b border-border/50 hover:bg-muted/50 transition-colors">
      {/* Top row: name/label + actions */}
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-1.5 min-w-0 flex-1">
          {entry.name !== null && (
            <Bookmark className="h-3 w-3 text-primary shrink-0" />
          )}
          {isRenaming ? (
            <div className="flex items-center gap-1 flex-1">
              <input
                type="text"
                value={nameInput}
                onChange={(e) => setNameInput(e.target.value)}
                onKeyDown={handleRenameKeyDown}
                onBlur={handleRenameSubmit}
                className="h-5 flex-1 px-1 text-xs rounded border border-primary/50 bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                placeholder="Name this set"
                autoFocus
              />
              <button
                type="button"
                onClick={handleRenameSubmit}
                className="p-0.5 text-primary hover:text-primary/80"
                title="Save name"
              >
                <Check className="h-3 w-3" />
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => {
                setNameInput(entry.name ?? '')
                setIsRenaming(true)
              }}
              className="text-xs font-medium text-foreground truncate hover:text-primary transition-colors text-left"
              title={entry.name ?? `${entry.ids.length} items`}
            >
              {entry.name ?? `${entry.ids.length} items`}
            </button>
          )}
        </div>

        {/* Action buttons - visible on hover */}
        <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
          <button
            type="button"
            onClick={onApply}
            className="p-1 rounded text-muted-foreground hover:text-green-400 hover:bg-green-500/10 transition-colors"
            title="Apply this selection"
          >
            <Play className="h-3 w-3" />
          </button>
          <button
            type="button"
            onClick={onEdit}
            className="p-1 rounded text-muted-foreground hover:text-primary hover:bg-primary/10 transition-colors"
            title="Edit set"
          >
            <Pencil className="h-3 w-3" />
          </button>
          <button
            type="button"
            onClick={onDelete}
            className="p-1 rounded text-muted-foreground hover:text-red-400 hover:bg-red-500/10 transition-colors"
            title="Delete set"
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      </div>

      {/* Bottom row: preview names + stats */}
      <div className="flex items-center justify-between gap-2">
        <span className="text-[10px] text-muted-foreground truncate">
          {previewNames.length > 0
            ? previewNames.join(', ') + (remaining > 0 ? ` +${remaining} more` : '')
            : `${entry.ids.length} items`}
        </span>
        <span className="text-[10px] text-muted-foreground whitespace-nowrap">
          {entry.copyCount}x &middot; {formatRelativeTime(entry.lastCopiedAt)}
        </span>
      </div>
    </div>
  )
}
