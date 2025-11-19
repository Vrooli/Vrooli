import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import toast from 'react-hot-toast'
import { AlertTriangle, ChevronDown, ChevronUp, ClipboardCopy, Eraser, Loader2 } from 'lucide-react'
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
import { ISSUE_DOCUMENTATION_LIBRARY, type DocumentationLink } from './issueDocumentation'

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
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [notes, setNotes] = useState<Record<string, string>>({})
  const [collapsedCategories, setCollapsedCategories] = useState<Set<string>>(new Set())
  const [expandedNotes, setExpandedNotes] = useState<Set<string>>(new Set())
  const [acknowledged, setAcknowledged] = useState(false)
  const [busy, setBusy] = useState(false)
  const [isCustomizingDescription, setIsCustomizingDescription] = useState(false)
  const [manualDescription, setManualDescription] = useState('')
  const [customizationBaseline, setCustomizationBaseline] = useState<{ selected: string[]; notes: Record<string, string> } | null>(null)
  const overallCheckboxRef = useRef<HTMLInputElement | null>(null)
  const categoryCheckboxRefs = useRef<Record<string, HTMLInputElement | null>>({})

  useEffect(() => {
    if (!seed || !open) {
      return
    }
    setTitle(seed.title)
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
    setCollapsedCategories(new Set())
    setExpandedNotes(new Set(Object.keys(nextNotes)))
    setAcknowledged(false)
    setIsCustomizingDescription(false)
    setManualDescription('')
    setCustomizationBaseline(null)
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
  const totalSelectable = selections.length
  const overallSelectionState: 'none' | 'partial' | 'all' = useMemo(() => {
    if (totalSelectable === 0 || selectedEntries.length === 0) {
      return 'none'
    }
    if (selectedEntries.length === totalSelectable) {
      return 'all'
    }
    return 'partial'
  }, [selectedEntries.length, totalSelectable])

  const normalizedNotes = useMemo(() => {
    const snapshot: Record<string, string> = {}
    Object.entries(notes).forEach(([key, value]) => {
      const trimmed = value?.trim()
      if (trimmed) {
        snapshot[key] = trimmed
      }
    })
    return snapshot
  }, [notes])

  const selectionSignature = useMemo(() => Array.from(selectedIds).sort(), [selectedIds])

  const selectedDocs = useMemo(() => {
    const seen = new Set<string>()
    const docs: DocumentationLink[] = []
    selectedEntries.forEach((entry) => {
      const entries = ISSUE_DOCUMENTATION_LIBRARY[entry.categoryId] ?? []
      entries.forEach((doc) => {
        const key = `${doc.title}-${doc.path}`
        if (!seen.has(key)) {
          docs.push(doc)
          seen.add(key)
        }
      })
    })
    return docs
  }, [selectedEntries])

  const generatedDescription = useMemo(() => {
    if (!seed) return ''
    const sections: string[] = []
    const scanDate = seed.metadata?.last_scanned
    const intro = `${seed.display_name || seed.entity_name}: ${selectedEntries.length} actionable issue${selectedEntries.length === 1 ? '' : 's'} selected for ${seed.source?.replace(/_/g, ' ') || 'quality scan'} triage${scanDate ? ` (last scan ${scanDate})` : ''}.`
    sections.push(intro)
    if (seed.description?.trim()) {
      sections.push(seed.description.trim())
    }

    const issueLines = selectedEntries.map((entry, index) => {
      const severityLabel = entry.severity ? entry.severity.toUpperCase() : 'INFO'
      const reference = entry.reference ? ` Reference: ${entry.reference}.` : ''
      const note = (normalizedNotes[entry.id] ?? entry.notes)?.trim()
      const noteText = note ? ` Notes: ${note}` : ''
      return `${index + 1}. [${entry.categoryTitle}] ${entry.title} — ${entry.detail.trim()} (Severity: ${severityLabel}).${reference}${noteText}`
    })

    if (issueLines.length > 0) {
      sections.push(['### Selected issues', '', ...issueLines].join('\n'))
    }

    if (selectedDocs.length > 0) {
      sections.push([
        '### Documentation kit',
        '',
        ...selectedDocs.map((doc) => `- ${doc.title}: ${doc.summary} (${doc.path})`),
      ].join('\n'))
    }

    return sections.filter(Boolean).join('\n\n')
  }, [seed, selectedEntries, normalizedNotes, selectedDocs])

  const descriptionToSubmit = isCustomizingDescription ? manualDescription : generatedDescription

  const { summary, status, error: statusError, refresh } = useScenarioIssues({
    entityType: seed?.entity_type,
    entityName: seed?.entity_name,
    autoFetch: open,
  })

  const hasOpenIssues = Boolean(summary && (summary.open_count > 0 || summary.active_count > 0))
  const hasDescription = descriptionToSubmit.trim().length > 0
  const canSubmit = Boolean(seed && selectedEntries.length > 0 && (!hasOpenIssues || acknowledged) && !busy && hasDescription)

  useEffect(() => {
    if (overallCheckboxRef.current) {
      overallCheckboxRef.current.indeterminate = overallSelectionState === 'partial'
    }
  }, [overallSelectionState])

  useEffect(() => {
    Object.entries(categoryCheckboxRefs.current).forEach(([categoryId, element]) => {
      if (!element) return
      const items = selections.filter((entry) => entry.categoryId === categoryId)
      if (items.length === 0) {
        element.indeterminate = false
        return
      }
      const selectedCount = items.filter((entry) => selectedIds.has(entry.id)).length
      element.indeterminate = selectedCount > 0 && selectedCount < items.length
    })
  }, [selections, selectedIds])

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

  const setCategorySelection = (categoryId: string, shouldSelect: boolean) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      const items = selections.filter((entry) => entry.categoryId === categoryId)
      items.forEach((item) => {
        if (shouldSelect) {
          next.add(item.id)
        } else {
          next.delete(item.id)
        }
      })
      return next
    })
  }

  const setAllSelected = (shouldSelect: boolean) => {
    setSelectedIds(() => {
      if (!shouldSelect) {
        return new Set()
      }
      return new Set(selections.map((entry) => entry.id))
    })
  }

  const toggleCategoryCollapsed = (categoryId: string) => {
    setCollapsedCategories((prev) => {
      const next = new Set(prev)
      if (next.has(categoryId)) {
        next.delete(categoryId)
      } else {
        next.add(categoryId)
      }
      return next
    })
  }

  const toggleNoteVisibility = (id: string) => {
    setExpandedNotes((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const copySummary = async () => {
    if (!seed) return
    const payload = descriptionToSubmit.trim() || generatedDescription
    if (!payload) {
      toast.error('Nothing to copy yet')
      return
    }
    try {
      await navigator.clipboard.writeText(payload)
      toast.success('Copied issue summary')
    } catch (err) {
      console.error(err)
      toast.error('Unable to copy summary')
    }
  }

  const beginCustomizing = () => {
    setIsCustomizingDescription(true)
    setManualDescription((prev) => (prev ? prev : generatedDescription))
    setCustomizationBaseline({ selected: [...selectionSignature], notes: { ...normalizedNotes } })
  }

  const exitCustomizing = () => {
    setIsCustomizingDescription(false)
    setCustomizationBaseline(null)
    setManualDescription('')
  }

  const selectionDrift = useMemo(() => {
    if (!customizationBaseline) return false
    const { selected } = customizationBaseline
    if (selected.length !== selectionSignature.length) return true
    return selected.some((value, index) => selectionSignature[index] !== value)
  }, [customizationBaseline, selectionSignature])

  const notesDrift = useMemo(() => {
    if (!customizationBaseline) return false
    const keys = new Set([...Object.keys(customizationBaseline.notes), ...Object.keys(normalizedNotes)])
    for (const key of keys) {
      if ((customizationBaseline.notes[key] ?? '') !== (normalizedNotes[key] ?? '')) {
        return true
      }
    }
    return false
  }, [customizationBaseline, normalizedNotes])

  const hasManualDrift = Boolean(isCustomizingDescription && customizationBaseline && (selectionDrift || notesDrift))

  const handleSubmit = async () => {
    if (!seed || !canSubmit) {
      return
    }

    const finalDescription = descriptionToSubmit.trim() || generatedDescription

    const metadata = {
      ...(seed.metadata ?? {}),
      selection_count: String(selectedEntries.length),
    }

    const payload: ScenarioIssueReportRequest = {
      entity_type: seed.entity_type,
      entity_name: seed.entity_name,
      source: seed.source,
      title: title.trim(),
      description: finalDescription.trim(),
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
              <label className="flex flex-1 items-center gap-3">
                <input
                  ref={overallCheckboxRef}
                  type="checkbox"
                  className="h-4 w-4 rounded border-slate-300"
                  disabled={totalSelectable === 0}
                  checked={overallSelectionState === 'all' && totalSelectable > 0}
                  onChange={(event) => setAllSelected(event.target.checked)}
                />
                <div>
                  <span>Detected issues</span>
                  <p className="text-[11px] font-normal text-muted-foreground">
                    {selectedEntries.length}/{totalSelectable} selected
                  </p>
                </div>
              </label>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                title="Clear selection"
                onClick={() => setSelectedIds(new Set())}
                disabled={selectedEntries.length === 0}
              >
                <Eraser size={16} />
                <span className="sr-only">Clear selection</span>
              </Button>
            </div>

            <div className="space-y-3 overflow-y-auto pr-2" style={{ maxHeight: '60vh' }}>
              {seed.categories.map((category, index) => {
                const categoryKey = category.id ?? `category-${index}`
                const categoryItems = selections.filter((entry) => entry.categoryId === categoryKey)
                const isFullySelected = categoryItems.every((entry) => selectedIds.has(entry.id))
                const isCollapsed = collapsedCategories.has(categoryKey)
                const selectedCount = categoryItems.filter((entry) => selectedIds.has(entry.id)).length
                return (
                  <div key={categoryKey} className="rounded-lg border p-3">
                    <div className="flex items-center justify-between">
                      <label className="flex flex-1 items-start gap-3">
                        <input
                          ref={(element) => {
                            categoryCheckboxRefs.current[categoryKey] = element
                          }}
                          type="checkbox"
                          className="mt-1 h-4 w-4 rounded border-slate-300"
                          checked={isFullySelected && categoryItems.length > 0}
                          disabled={categoryItems.length === 0}
                          onChange={(event) => setCategorySelection(categoryKey, event.target.checked)}
                        />
                        <div>
                          <p className="font-semibold text-slate-900">{category.title}</p>
                          {category.description && <p className="text-xs text-muted-foreground">{category.description}</p>}
                          <p className="text-[11px] font-normal text-muted-foreground">
                            {selectedCount}/{categoryItems.length} selected
                          </p>
                        </div>
                      </label>
                      <div className="flex items-center gap-1 text-xs pl-3">
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          title={isCollapsed ? 'Expand category' : 'Collapse category'}
                          onClick={() => toggleCategoryCollapsed(categoryKey)}
                        >
                          {isCollapsed ? <ChevronDown size={16} /> : <ChevronUp size={16} />}
                          <span className="sr-only">{isCollapsed ? 'Expand category' : 'Collapse category'}</span>
                        </Button>
                      </div>
                    </div>
                    {!isCollapsed && (
                      <div className="mt-3 space-y-2">
                        {categoryItems.map((entry) => {
                          const severityClass = getSeverityBadge(entry.severity)
                          const existingNote = notes[entry.id]?.trim()
                          const showNotes = expandedNotes.has(entry.id) || Boolean(existingNote)
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
                                  <div className="flex flex-wrap items-center gap-2 text-slate-900">
                                    <span className="font-semibold">{entry.title}</span>
                                    <Badge className={severityClass}>{entry.severity?.toUpperCase() || 'INFO'}</Badge>
                                  </div>
                                  <p className="text-slate-600">{entry.detail}</p>
                                  <div className="flex items-center gap-2 text-[11px] text-slate-500">
                                    {existingNote && !showNotes && <span className="truncate">{existingNote}</span>}
                                    <button
                                      type="button"
                                      className="font-semibold text-primary underline-offset-2 hover:underline"
                                      onClick={() => toggleNoteVisibility(entry.id)}
                                    >
                                      {showNotes ? 'Hide notes' : existingNote ? 'Edit notes' : 'Add notes'}
                                    </button>
                                  </div>
                                  {showNotes && (
                                    <Textarea
                                      value={notes[entry.id] ?? ''}
                                      onChange={(event) =>
                                        setNotes((prev) => ({ ...prev, [entry.id]: event.target.value }))
                                      }
                                      placeholder="Add context so fixes are deterministic"
                                      className="mt-1 h-16"
                                    />
                                  )}
                                </div>
                              </label>
                            </div>
                          )
                        })}
                      </div>
                    )}
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
              <div className="flex items-start justify-between gap-4">
                <div>
                  <label className="text-xs font-semibold text-slate-500">Issue report preview</label>
                  <p className="text-xs text-muted-foreground">Auto-updates with selections until you customize it.</p>
                </div>
                {isCustomizingDescription ? (
                  <Button variant="outline" size="sm" onClick={exitCustomizing}>
                    Return to preview
                  </Button>
                ) : (
                  <Button variant="ghost" size="sm" onClick={beginCustomizing}>
                    Customize description
                  </Button>
                )}
              </div>
              <Textarea
                value={isCustomizingDescription ? manualDescription : generatedDescription}
                onChange={(event) => setManualDescription(event.target.value)}
                disabled={!isCustomizingDescription}
                rows={12}
              />
              {isCustomizingDescription && hasManualDrift && (
                <div className="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-900">
                  <AlertTriangle size={16} className="mt-0.5 text-amber-600" />
                  <div>
                    <p className="font-semibold">Selections have drifted</p>
                    <p>
                      The chosen issues or their notes changed after you started editing. Refresh your custom description or return to the preview to keep the report aligned.
                    </p>
                  </div>
                </div>
              )}
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
