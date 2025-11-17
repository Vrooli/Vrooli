import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Loader2, CheckCircle2, Trash2, Inbox } from 'lucide-react'
import { formatDate } from '../../utils/formatters'
import type { BacklogEntry } from '../../types'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '../ui/card'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Badge } from '../ui/badge'
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

  useEffect(() => {
    const next: Record<string, string> = {}
    entries.forEach((entry) => {
      next[entry.id] = entry.notes ?? ''
    })
    setNotesDrafts(next)
  }, [entries])

  const handleNotesSave = async (id: string) => {
    const nextNotes = notesDrafts[id] ?? ''
    setSavingNotesId(id)
    try {
      await onUpdateNotes(id, nextNotes)
    } finally {
      setSavingNotesId((current) => (current === id ? null : current))
    }
  }

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
                      checked={selection.size === entries.length}
                      onChange={onToggleSelectAll}
                    />
                  </th>
                  <th className="px-3 py-2">Idea</th>
                  <th className="px-3 py-2">Type</th>
                  <th className="px-3 py-2">Slug / Draft</th>
                  <th className="px-3 py-2">Notes</th>
                  <th className="px-3 py-2">Status</th>
                  <th className="px-3 py-2">Created</th>
                  <th className="px-3 py-2">Updated</th>
                  <th className="px-3 py-2 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {entries.map((entry) => {
                  const isSelected = selection.has(entry.id)
                  const entryBusy = busy && busyId === entry.id
                  const encodedSlug = encodeURIComponent(entry.suggested_name)
                  const draftLink = entry.status === 'converted' ? `/draft/${entry.entity_type}/${encodedSlug}` : null
                  const notesValue = notesDrafts[entry.id] ?? ''
                  const initialNotes = entry.notes ?? ''
                  const isDirty = notesValue !== initialNotes

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
                        <p className="font-medium text-slate-900">{entry.idea_text}</p>
                      </td>
                      <td className="px-3 py-3">
                        <Badge variant="secondary" className="capitalize">
                          {entry.entity_type}
                        </Badge>
                      </td>
                      <td className="px-3 py-3">
                        <div className="flex flex-col gap-1 text-xs font-mono text-slate-600">
                          <code>{entry.suggested_name}</code>
                          {draftLink && (
                            <Link to={draftLink} className="text-xs text-primary hover:underline">
                              Open draft
                            </Link>
                          )}
                        </div>
                      </td>
                      <td className="px-3 py-3">
                        <div className="space-y-2">
                          <textarea
                            rows={3}
                            className="w-full rounded-lg border border-input bg-white px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                            placeholder="Add context or next steps..."
                            value={notesValue}
                            onChange={(event) =>
                              setNotesDrafts((prev) => ({
                                ...prev,
                                [entry.id]: event.target.value,
                              }))
                            }
                          />
                          <div className="flex justify-end">
                            <Button
                              type="button"
                              size="sm"
                              variant="secondary"
                              disabled={!isDirty || savingNotesId === entry.id}
                              onClick={() => handleNotesSave(entry.id)}
                            >
                              {savingNotesId === entry.id && <Loader2 size={14} className="mr-2 animate-spin" />}
                              Save notes
                            </Button>
                          </div>
                        </div>
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
                      <td className="px-3 py-3 text-slate-600">{formatDate(entry.created_at)}</td>
                      <td className="px-3 py-3 text-slate-600">{formatDate(entry.updated_at)}</td>
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
    </Card>
  )
}
