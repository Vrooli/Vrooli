import { useCallback, useEffect, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import toast from 'react-hot-toast'
import { AlertTriangle, ClipboardCopy, Loader2 } from 'lucide-react'
import type {
  IssueReportSeed,
  IssueReportSelectionInput,
  ScenarioIssueReportRequest,
  ScenarioIssueReportResponse,
} from '../../types'
import { submitIssueReport } from '../../services/issues'
import { scenarioIssuesStore } from '../../state/scenarioIssuesStore'
import { useScenarioIssues } from '../../hooks/useScenarioIssues'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Textarea } from '../ui/textarea'
import { Badge } from '../ui/badge'
import { Separator } from '../ui/separator'

interface IssueReportDialogProps {
  seed: IssueReportSeed | null
  open: boolean
  onOpenChange?: (open: boolean) => void
  onSubmitted?: (response: ScenarioIssueReportResponse) => void
}

interface SelectionEntry extends IssueReportSelectionInput {
  categoryId: string
  categoryTitle: string
}

function getSeverityBadge(severity?: string) {
  switch (severity) {
    case 'critical':
    case 'p0':
      return 'bg-rose-100 text-rose-800'
    case 'high':
    case 'p1':
      return 'bg-amber-100 text-amber-800'
    case 'low':
      return 'bg-slate-100 text-slate-600'
    default:
      return 'bg-blue-100 text-blue-800'
  }
}

export function IssueReportDialog({ seed, open, onOpenChange, onSubmitted }: IssueReportDialogProps) {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [notes, setNotes] = useState<Record<string, string>>({})
  const [acknowledged, setAcknowledged] = useState(false)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (!seed || !open) {
      return
    }
    setTitle(seed.title)
    setDescription(seed.description)
    const next = new Set<string>()
    const nextNotes: Record<string, string> = {}
    seed.categories.forEach((category) => {
      category.items.forEach((item) => {
        if (category.defaultSelected ?? true) {
          next.add(item.id)
        }
        if (item.notes) {
          nextNotes[item.id] = item.notes
        }
      })
    })
    setSelectedIds(next)
    setNotes(nextNotes)
    setAcknowledged(false)
  }, [seed, open])

  const selections = useMemo<SelectionEntry[]>(() => {
    if (!seed) return []
    return seed.categories.flatMap((category, idx) => {
      const categoryId = category.id ?? `category-${idx}`
      const categoryTitle =
        typeof category.title === 'string' ? category.title : String(category.title ?? `Category ${idx + 1}`)
      return category.items.map((item) => ({
        ...item,
        categoryId,
        categoryTitle,
      }))
    })
  }, [seed])

  const selectedEntries = useMemo(() => selections.filter((entry) => selectedIds.has(entry.id)), [selections, selectedIds])

  const { summary, status, error: statusError, refresh } = useScenarioIssues({
    entityType: seed?.entity_type,
    entityName: seed?.entity_name,
    autoFetch: open,
  })

  const hasOpenIssues = Boolean(summary && (summary.open_count > 0 || summary.active_count > 0))
  const canSubmit = Boolean(seed && selectedEntries.length > 0 && (!hasOpenIssues || acknowledged) && !busy)

  const close = useCallback(() => {
    onOpenChange?.(false)
  }, [onOpenChange])

  const handleSelectionToggle = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const toggleCategory = (categoryId: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      const items = selections.filter((entry) => entry.categoryId === categoryId)
      const isFullySelected = items.every((item) => next.has(item.id))
      items.forEach((item) => {
        if (isFullySelected) {
          next.delete(item.id)
        } else {
          next.add(item.id)
        }
      })
      return next
    })
  }

  const copySummary = async () => {
    if (!seed) return
    const lines = [`${seed.entity_name}: ${selectedEntries.length} issues`]
    selectedEntries.forEach((entry) => {
      lines.push(`- [${entry.categoryTitle}] ${entry.title} — ${entry.detail}`)
    })
    try {
      await navigator.clipboard.writeText(lines.join('\n'))
      toast.success('Copied issue summary')
    } catch (err) {
      console.error(err)
      toast.error('Unable to copy summary')
    }
  }

  const handleSubmit = async () => {
    if (!seed || !canSubmit) {
      return
    }

    const metadata = {
      ...(seed.metadata ?? {}),
      selection_count: String(selectedEntries.length),
    }

    const payload: ScenarioIssueReportRequest = {
      entity_type: seed.entity_type,
      entity_name: seed.entity_name,
      source: seed.source,
      title: title.trim(),
      description: description.trim(),
      summary: seed.summary,
      tags: seed.tags,
      labels: seed.labels,
      metadata,
      attachments: seed.attachments,
      selections: selectedEntries.map((entry) => ({
        id: entry.id,
        title: entry.title,
        detail: entry.detail,
        category: entry.category,
        severity: entry.severity,
        reference: entry.reference,
        notes: notes[entry.id] ?? entry.notes,
      })),
    }

    setBusy(true)
    try {
      const response = await submitIssueReport(payload)
      scenarioIssuesStore.flagIssueReported(seed.entity_type, seed.entity_name)
      toast.success('Issue reported to control tower')
      onSubmitted?.(response)
      close()
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to submit report'
      toast.error(message)
    } finally {
      setBusy(false)
    }
  }

  if (!open || !seed) {
    return null
  }

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/40 p-6">
      <div className="relative mt-8 w-full max-w-5xl rounded-2xl bg-white shadow-2xl">
        <div className="flex items-start justify-between border-b px-6 py-4">
          <div>
            <p className="text-xs uppercase tracking-wide text-slate-500">Issue report</p>
            <h2 className="text-2xl font-semibold text-slate-900">{seed.display_name || seed.entity_name}</h2>
            <p className="text-sm text-muted-foreground">
              {selectedEntries.length} selection{selectedEntries.length === 1 ? '' : 's'} included
            </p>
          </div>
          <button type="button" className="text-slate-500 hover:text-slate-900" onClick={close}>
            ×
          </button>
        </div>

        <div className="grid gap-6 px-6 py-4 lg:grid-cols-[360px,1fr]">
          <div className="space-y-4 border-r pr-4">
            {summary && (
              <div className="rounded-lg border border-slate-200 bg-slate-50 p-3 text-xs text-slate-700">
                <div className="flex items-center justify-between">
                  <p>
                    {summary.open_count} open · {summary.active_count} active
                  </p>
                  <button type="button" className="text-primary underline-offset-2 hover:underline" onClick={() => refresh().catch(() => undefined)}>
                    {status === 'loading' ? 'Refreshing…' : 'Refresh'}
                  </button>
                </div>
                <div className="mt-2 flex items-center gap-2">
                  <AlertTriangle size={14} className="text-amber-600" />
                  <span>
                    {hasOpenIssues ? 'Open issues detected. Acknowledge before submitting.' : 'No blocking issues in tracker.'}
                  </span>
                </div>
                {summary.tracker_url && (
                  <a href={summary.tracker_url} className="mt-2 inline-block text-primary underline-offset-2 hover:underline" target="_blank" rel="noreferrer">
                    View tracker →
                  </a>
                )}
              </div>
            )}
            {!summary && statusError && <p className="text-xs text-rose-600">{statusError}</p>}

            <div className="flex items-center justify-between text-xs font-semibold text-slate-600">
              <span>Detected issues</span>
              <div className="space-x-2">
                <button type="button" onClick={() => setSelectedIds(new Set())} className="text-primary underline-offset-2 hover:underline">
                  Clear
                </button>
                <button
                  type="button"
                  onClick={() => setSelectedIds(new Set(selections.map((entry) => entry.id)))}
                  className="text-primary underline-offset-2 hover:underline"
                >
                  Select all
                </button>
              </div>
            </div>

            <div className="space-y-3 overflow-y-auto pr-2" style={{ maxHeight: '60vh' }}>
              {seed.categories.map((category, index) => {
                const categoryKey = category.id ?? `category-${index}`
                const categoryItems = selections.filter((entry) => entry.categoryId === categoryKey)
                const isFullySelected = categoryItems.every((entry) => selectedIds.has(entry.id))
                return (
                  <div key={categoryKey} className="rounded-lg border p-3">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="font-semibold text-slate-900">{category.title}</p>
                        {category.description && <p className="text-xs text-muted-foreground">{category.description}</p>}
                      </div>
                      <button type="button" className="text-xs text-primary underline-offset-2 hover:underline" onClick={() => toggleCategory(categoryKey)}>
                        {isFullySelected ? 'Deselect' : 'Select'} all
                      </button>
                    </div>
                    <div className="mt-3 space-y-2">
                      {categoryItems.map((entry) => {
                        const severityClass = getSeverityBadge(entry.severity)
                        return (
                          <div key={entry.id} className="rounded border border-slate-200 p-3 text-xs">
                            <label className="flex items-start gap-2">
                              <input
                                type="checkbox"
                                className="mt-1"
                                checked={selectedIds.has(entry.id)}
                                onChange={() => handleSelectionToggle(entry.id)}
                              />
                              <div className="space-y-1">
                                <div className="flex items-center gap-2 text-slate-900">
                                  <span className="font-semibold">{entry.title}</span>
                                  <Badge className={severityClass}>{entry.severity?.toUpperCase() || 'INFO'}</Badge>
                                </div>
                                <p className="text-slate-600">{entry.detail}</p>
                                <Textarea
                                  value={notes[entry.id] ?? ''}
                                  onChange={(event) => setNotes((prev) => ({ ...prev, [entry.id]: event.target.value }))}
                                  placeholder="Add context..."
                                  className="mt-1 h-16"
                                />
                              </div>
                            </label>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>

          <div className="space-y-4">
            <div className="space-y-2">
              <label className="text-xs font-semibold text-slate-500">Title</label>
              <Input value={title} onChange={(event) => setTitle(event.target.value)} />
            </div>
            <div className="space-y-2">
              <label className="text-xs font-semibold text-slate-500">Description</label>
              <Textarea value={description} onChange={(event) => setDescription(event.target.value)} rows={10} />
            </div>
            {seed.metadata && (
              <div className="rounded-lg border border-dashed p-3 text-xs">
                <p className="font-semibold text-slate-700">Context</p>
                <div className="mt-2 grid gap-x-4 gap-y-1 md:grid-cols-2">
                  {Object.entries(seed.metadata).map(([key, value]) => (
                    <div key={key} className="text-slate-600">
                      <span className="font-semibold text-slate-900">{key.replace(/_/g, ' ')}:</span> {value}
                    </div>
                  ))}
                </div>
              </div>
            )}
            {seed.attachments && seed.attachments.length > 0 && (
              <div className="rounded-lg border border-slate-200 p-3 text-xs">
                <p className="font-semibold text-slate-700">Attachments</p>
                <ul className="mt-2 list-disc space-y-1 pl-4 text-slate-600">
                  {seed.attachments.map((attachment) => (
                    <li key={attachment.name}>{attachment.name}</li>
                  ))}
                </ul>
              </div>
            )}
            {hasOpenIssues && (
              <label className="flex items-center gap-2 text-xs text-slate-600">
                <input type="checkbox" checked={acknowledged} onChange={(event) => setAcknowledged(event.target.checked)} />
                I understand there are open or active tickets for this scenario.
              </label>
            )}
          </div>
        </div>

        <Separator />
        <div className="flex flex-wrap items-center justify-between gap-3 px-6 py-4">
          <div className="text-sm text-muted-foreground">
            {selectedEntries.length} issue{selectedEntries.length === 1 ? '' : 's'} selected
          </div>
          <div className="flex items-center gap-3">
            <Button variant="outline" onClick={copySummary} className="gap-2">
              <ClipboardCopy size={16} /> Copy summary
            </Button>
            <Button variant="ghost" onClick={close}>
              Cancel
            </Button>
            <Button onClick={handleSubmit} disabled={!canSubmit} className="gap-2">
              {busy ? <Loader2 size={16} className="animate-spin" /> : 'Submit report'}
            </Button>
          </div>
        </div>
      </div>
    </div>,
    document.body,
  )
}
