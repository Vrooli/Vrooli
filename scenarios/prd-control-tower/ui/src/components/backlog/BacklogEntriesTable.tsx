import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Loader2, CheckCircle2, Trash2, Inbox, ArrowUpDown, NotebookPen } from 'lucide-react'
import { formatDate, formatCompactDate } from '../../utils/formatters'
import type { BacklogEntry } from '../../types'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '../ui/card'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Badge } from '../ui/badge'
import { Textarea } from '../ui/textarea'
import { cn } from '../../lib/utils'

interface BacklogEntriesTableProps {
  entries: BacklogEntry[]
  selection: Set<string>
  busy: boolean
  busyId: string | null
  filter: string
  loading: boolean
  totalCount: number
  onFilterChange: (value: string) => void
  onToggleSelection: (id: string) => void
  onToggleSelectAll: () => void
  onConvert: (ids?: string[]) => void
  onDelete: (ids?: string[]) => void
  onRefresh: () => void
  onUpdateNotes: (id: string, notes: string) => Promise<void>
}

type SortKey = 'idea' | 'type' | 'status' | 'created' | 'updated'

export function BacklogEntriesTable({
  entries,
  selection,
  busy,
  busyId,
  filter,
  loading,
  totalCount,
  onFilterChange,
  onToggleSelection,
  onToggleSelectAll,
  onConvert,
  onDelete,
  onRefresh,
  onUpdateNotes,
}: BacklogEntriesTableProps) {
  const selectedCount = selection.size
  const [notesDrafts, setNotesDrafts] = useState<Record<string, string>>({})
  const [savingNotesId, setSavingNotesId] = useState<string | null>(null)
  const [activeNotesId, setActiveNotesId] = useState<string | null>(null)
  const [sortConfig, setSortConfig] = useState<{ key: SortKey; direction: 'asc' | 'desc' }>({
    key: 'created',
    direction: 'desc',
  })

  useEffect(() => {
    const next: Record<string, string> = {}
    entries.forEach((entry) => {
      next[entry.id] = entry.notes ?? ''
    })
    setNotesDrafts(next)
  }, [entries])

  const sortedEntries = useMemo(() => {
    if (!sortConfig) {
      return entries
    }
    const sorted = [...entries]
    sorted.sort((a, b) => {
      const { key, direction } = sortConfig
      const multiplier = direction === 'asc' ? 1 : -1
      let result = 0
      if (key === 'idea') {
        result = a.idea_text.localeCompare(b.idea_text)
      } else if (key === 'type') {
        result = a.entity_type.localeCompare(b.entity_type)
      } else if (key === 'status') {
        result = a.status.localeCompare(b.status)
      } else if (key === 'created') {
        result = new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
      } else if (key === 'updated') {
        result = new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime()
      }
      return result * multiplier
    })
    return sorted
  }, [entries, sortConfig])

  const activeNotesEntry = useMemo(() => {
    if (!activeNotesId) return null
    return entries.find((entry) => entry.id === activeNotesId) ?? null
  }, [activeNotesId, entries])

  const handleSort = (key: SortKey) => {
    setSortConfig((current) => {
      if (current?.key === key) {
        return {
          key,
          direction: current.direction === 'asc' ? 'desc' : 'asc',
        }
      }
      return {
        key,
        direction: key === 'idea' ? 'asc' : 'desc',
      }
    })
  }

  const renderSortLabel = (label: string, key: SortKey) => {
    const isActive = sortConfig?.key === key
    const direction = sortConfig?.direction
    return (
      <button
        type="button"
        className="flex items-center gap-1 font-semibold text-slate-600"
        onClick={() => handleSort(key)}
      >
        {label}
        {isActive ? (
          <span aria-hidden="true">{direction === 'asc' ? '↑' : '↓'}</span>
        ) : (
          <ArrowUpDown size={14} />
        )}
      </button>
    )
  }

  const handleNotesSave = async (id: string) => {
    const nextNotes = notesDrafts[id] ?? ''
    setSavingNotesId(id)
    try {
      await onUpdateNotes(id, nextNotes)
      setActiveNotesId((current) => (current === id ? null : current))
    } finally {
      setSavingNotesId((current) => (current === id ? null : current))
    }
  }

  const allDisplayedSelected = sortedEntries.length > 0 && sortedEntries.every((entry) => selection.has(entry.id))

  return (
    <Card>
      <CardHeader className="gap-6 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <CardTitle>Backlog entries</CardTitle>
          <CardDescription>
            {totalCount} captured idea{totalCount === 1 ? '' : 's'} ready for triage.
          </CardDescription>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input
            type="text"
            placeholder="Filter backlog..."
            value={filter}
            onChange={(event) => onFilterChange(event.target.value)}
          />
          <Button variant="ghost" disabled={busy} onClick={onRefresh}>
            Refresh
          </Button>
          <Button variant="secondary" disabled={busy} onClick={() => onConvert()}>
            {busy && busyId === null ? <Loader2 size={16} className="mr-2 animate-spin" /> : <CheckCircle2 size={16} className="mr-2" />}
            Convert selected
          </Button>
          <Button variant="outline" className="border-destructive text-destructive" disabled={busy} onClick={() => onDelete()}>
            <Trash2 size={16} className="mr-2" /> Delete selected
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex items-center justify-center gap-3 rounded-2xl border border-dashed py-12 text-muted-foreground">
            <Loader2 size={20} className="animate-spin" /> Loading backlog...
          </div>
        ) : entries.length === 0 ? (
          <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed bg-slate-50 py-12 text-center text-muted-foreground">
            <Inbox size={36} />
            <p>No backlog ideas yet. Save them from the intake panel to keep track.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200 text-left text-sm">
              <thead>
                <tr className="text-xs uppercase tracking-wide text-slate-500">
                  <th className="px-3 py-2">
                    <input
                      type="checkbox"
                      aria-label="Select all backlog entries"
                      className="h-4 w-4 accent-violet-500"
                      checked={allDisplayedSelected}
                      onChange={onToggleSelectAll}
                    />
                  </th>
                  <th className="px-3 py-2">{renderSortLabel('Idea', 'idea')}</th>
                  <th className="px-3 py-2">{renderSortLabel('Type', 'type')}</th>
                  <th className="px-3 py-2">{renderSortLabel('Status', 'status')}</th>
                  <th className="px-3 py-2">{renderSortLabel('Created', 'created')}</th>
                  <th className="px-3 py-2">{renderSortLabel('Updated', 'updated')}</th>
                  <th className="px-3 py-2">Notes</th>
                  <th className="px-3 py-2 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {sortedEntries.map((entry) => {
                  const isSelected = selection.has(entry.id)
                  const entryBusy = busy && busyId === entry.id
                  const encodedSlug = encodeURIComponent(entry.suggested_name)
                  const draftLink = entry.status === 'converted' ? `/draft/${entry.entity_type}/${encodedSlug}` : null
                  const notesValue = notesDrafts[entry.id] ?? ''
                  const initialNotes = entry.notes ?? ''
                  const isDirty = notesValue !== initialNotes
                  const createdLabel = formatCompactDate(entry.created_at)
                  const updatedLabel = formatCompactDate(entry.updated_at)
                  const hasNotes = Boolean(entry.notes)

                  return (
                    <tr key={entry.id} className={cn(isSelected && 'bg-violet-50/60')}>
                      <td className="px-3 py-3">
                        <input
                          type="checkbox"
                          className="h-4 w-4 accent-violet-500"
                          checked={isSelected}
                          onChange={() => onToggleSelection(entry.id)}
                        />
                      </td>
                      <td className="px-3 py-3">
                        <div className="space-y-1">
                          <p className="font-medium text-slate-900">{entry.idea_text}</p>
                          <div className="flex flex-wrap items-center gap-2 text-xs text-slate-600">
                            <span className="rounded bg-slate-100 px-2 py-0.5 font-mono">{entry.suggested_name}</span>
                            {draftLink && (
                              <Link to={draftLink} className="text-primary hover:underline">
                                Open draft
                              </Link>
                            )}
                          </div>
                        </div>
                      </td>
                      <td className="px-3 py-3">
                        <Badge variant="secondary" className="capitalize">
                          {entry.entity_type}
                        </Badge>
                      </td>
                      <td className="px-3 py-3">
                        {entry.status === 'converted' ? (
                          <Badge variant="success" className="gap-1">
                            <CheckCircle2 size={14} /> Converted
                          </Badge>
                        ) : entry.status === 'archived' ? (
                          <Badge variant="outline">Archived</Badge>
                        ) : (
                          <Badge variant="warning">Pending</Badge>
                        )}
                      </td>
                      <td className="px-3 py-3 text-slate-600" title={formatDate(entry.created_at)}>
                        {createdLabel}
                      </td>
                      <td className="px-3 py-3 text-slate-600" title={formatDate(entry.updated_at)}>
                        {updatedLabel}
                      </td>
                      <td className="px-3 py-3">
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className={cn('px-2 text-slate-600', hasNotes && 'text-violet-600')}
                          onClick={() => setActiveNotesId(entry.id)}
                          aria-label={hasNotes ? 'View or edit notes' : 'Add notes'}
                        >
                          <NotebookPen size={16} className="mr-2" />
                          {hasNotes ? 'Edit' : 'Add'}
                        </Button>
                        {isDirty && (
                          <span className="ml-2 text-xs text-amber-600">Unsaved</span>
                        )}
                      </td>
                      <td className="px-3 py-3 text-right">
                        <div className="flex justify-end gap-3">
                          <Button
                            type="button"
                            variant="link"
                            className="px-0"
                            disabled={entryBusy}
                            onClick={() => onConvert([entry.id])}
                          >
                            {entryBusy ? <Loader2 size={16} className="mr-2 animate-spin" /> : <CheckCircle2 size={16} className="mr-2" />}
                            Convert
                          </Button>
                          <Button
                            type="button"
                            variant="link"
                            className="px-0 text-destructive"
                            disabled={entryBusy}
                            onClick={() => onDelete([entry.id])}
                          >
                            <Trash2 size={16} className="mr-2" /> Delete
                          </Button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
      <CardFooter className="border-t border-dashed pt-4 text-sm text-muted-foreground">
        {selectedCount} selected
      </CardFooter>
      {activeNotesEntry && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/70 px-4"
          role="dialog"
          aria-modal="true"
          aria-labelledby="notes-dialog-title"
          onClick={() => setActiveNotesId(null)}
        >
          <div className="w-full max-w-xl rounded-2xl border bg-white shadow-2xl" onClick={(event) => event.stopPropagation()}>
            <div className="flex items-start justify-between gap-4 border-b p-6">
              <div>
                <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Backlog notes</p>
                <h2 id="notes-dialog-title" className="text-xl font-semibold text-slate-900">
                  {activeNotesEntry.idea_text}
                </h2>
                <p className="mt-1 text-sm text-muted-foreground">{activeNotesEntry.suggested_name}</p>
              </div>
              <Button variant="ghost" size="sm" onClick={() => setActiveNotesId(null)}>
                Close
              </Button>
            </div>
            <div className="space-y-4 p-6">
              <Textarea
                rows={6}
                value={notesDrafts[activeNotesEntry.id] ?? ''}
                placeholder="Add context or next steps..."
                onChange={(event) =>
                  setNotesDrafts((prev) => ({
                    ...prev,
                    [activeNotesEntry.id]: event.target.value,
                  }))
                }
              />
              <div className="flex items-center justify-end gap-2">
                <Button variant="ghost" onClick={() => setActiveNotesId(null)}>
                  Cancel
                </Button>
                <Button
                  onClick={() => handleNotesSave(activeNotesEntry.id)}
                  disabled={savingNotesId === activeNotesEntry.id ||
                    (notesDrafts[activeNotesEntry.id] ?? '') === (activeNotesEntry.notes ?? '')}
                >
                  {savingNotesId === activeNotesEntry.id ? 'Saving…' : 'Save notes'}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </Card>
  )
}
