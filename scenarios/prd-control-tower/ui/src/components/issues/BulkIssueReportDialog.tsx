import { useCallback, useEffect, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import toast from 'react-hot-toast'
import { Loader2 } from 'lucide-react'
import type { IssueReportSeed, ScenarioIssueReportRequest } from '../../types'
import { bulkSubmitIssueReports } from '../../services/issues'
import { scenarioIssuesStore } from '../../state/scenarioIssuesStore'
import { Button } from '../ui/button'
import { Separator } from '../ui/separator'
import { Input } from '../ui/input'
import { Textarea } from '../ui/textarea'
import { Badge } from '../ui/badge'

interface BulkIssueReportDialogProps {
  open: boolean
  seeds: IssueReportSeed[]
  onOpenChange?: (open: boolean) => void
}

interface BatchResult {
  id: string
  seeds: IssueReportSeed[]
  status: 'pending' | 'running' | 'completed'
  results: Array<{ seed: IssueReportSeed; status: 'success' | 'error'; message: string; issueId?: string }>
}

const MAX_BATCH_SIZE = 20

function buildRequestFromSeed(seed: IssueReportSeed): ScenarioIssueReportRequest {
  const selections = seed.categories.flatMap((category) => category.items)
  return {
    entity_type: seed.entity_type,
    entity_name: seed.entity_name,
    source: seed.source,
    title: seed.title,
    description: seed.description,
    summary: seed.summary,
    tags: seed.tags,
    labels: seed.labels,
    metadata: seed.metadata,
    attachments: seed.attachments,
    selections,
  }
}

export function BulkIssueReportDialog({ open, seeds, onOpenChange }: BulkIssueReportDialogProps) {
  const [batchSize, setBatchSize] = useState(5)
  const [submitting, setSubmitting] = useState(false)
  const [batches, setBatches] = useState<BatchResult[]>([])
  const [draftSeeds, setDraftSeeds] = useState<IssueReportSeed[]>([])
  const [activeSeedIndex, setActiveSeedIndex] = useState(0)

  const resetDialogState = useCallback(
    (seedList: IssueReportSeed[]) => {
      setBatches([])
      setSubmitting(false)
      setDraftSeeds(seedList.map((seed) => ({ ...seed })))
      setActiveSeedIndex(0)
    },
    [],
  )

  useEffect(() => {
    if (!open) {
      return
    }
    // We intentionally reset internal dialog state whenever the modal is re-opened
    // so that the batch planner reflects the seeds provided by the caller.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    resetDialogState(seeds)
  }, [open, resetDialogState, seeds])

  const resolvedBatchSize = Math.min(
    Math.max(batchSize, 1),
    Math.min(MAX_BATCH_SIZE, Math.max(1, draftSeeds.length || seeds.length || 1)),
  )
  const plannedBatches = useMemo(() => {
    if (!draftSeeds || draftSeeds.length === 0) return []
    const chunks: IssueReportSeed[][] = []
    for (let i = 0; i < draftSeeds.length; i += resolvedBatchSize) {
      chunks.push(draftSeeds.slice(i, i + resolvedBatchSize))
    }
    return chunks
  }, [draftSeeds, resolvedBatchSize])

  const close = () => onOpenChange?.(false)

  const activeSeed = draftSeeds[activeSeedIndex]

  const updateActiveSeed = (updates: Partial<IssueReportSeed>) => {
    setDraftSeeds((prev) =>
      prev.map((seed, index) => (index === activeSeedIndex ? { ...seed, ...updates } : seed)),
    )
  }

  const handleSubmit = async () => {
    if (!plannedBatches.length) {
      return
    }
    setSubmitting(true)
    const pendingBatches: BatchResult[] = plannedBatches.map((chunk, idx) => ({
      id: `batch-${idx + 1}`,
      seeds: chunk,
      status: 'pending',
      results: [],
    }))
    setBatches(pendingBatches)

    for (let index = 0; index < plannedBatches.length; index += 1) {
      const chunk = plannedBatches[index]
      setBatches((prev) => prev.map((entry, idx) => (idx === index ? { ...entry, status: 'running' } : entry)))

      const requests = chunk.map((seed) => buildRequestFromSeed(seed))
      const results = await bulkSubmitIssueReports(requests)

      setBatches((prev) =>
        prev.map((entry, idx) => {
          if (idx !== index) {
            return entry
          }
          return {
            ...entry,
            status: 'completed',
            results: results.map((result, resultIdx) => {
              const seed = chunk[resultIdx]
              if (result.response) {
                scenarioIssuesStore.flagIssueReported(seed.entity_type, seed.entity_name)
              }
              return {
                seed,
                status: result.response ? 'success' : 'error',
                message: result.response ? 'Reported successfully' : result.error || 'Failed',
                issueId: result.response?.issue_id,
              }
            }),
          }
        }),
      )
    }

    toast.success('Bulk issue reporting complete')
    setSubmitting(false)
  }

  if (!open) {
    return null
  }

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/40 p-6">
      <div className="relative mt-10 w-full max-w-4xl rounded-2xl bg-white shadow-2xl">
        <div className="flex items-center justify-between border-b px-6 py-4">
          <div>
            <p className="text-xs uppercase tracking-wide text-slate-500">Bulk issue reports</p>
            <h2 className="text-2xl font-semibold text-slate-900">{draftSeeds.length} scenario(s) selected</h2>
            <p className="text-sm text-muted-foreground">Chunked into {plannedBatches.length || 1} batch(es)</p>
          </div>
          <button type="button" className="text-slate-500 hover:text-slate-900" onClick={close}>
            ×
          </button>
        </div>

        <div className="space-y-6 px-6 py-4">
          <div className="rounded-lg border border-slate-200 p-4">
            <label className="text-xs font-semibold text-slate-600">Scenarios per batch</label>
            <div className="mt-2 flex items-center gap-4">
              <input
                type="range"
                min={1}
                max={Math.max(1, Math.min(MAX_BATCH_SIZE, draftSeeds.length || seeds.length || 1))}
                value={resolvedBatchSize}
                onChange={(event) => setBatchSize(Number(event.target.value))}
                className="flex-1"
              />
              <span className="w-12 text-sm text-slate-700">{resolvedBatchSize}</span>
            </div>
            <p className="mt-2 text-xs text-muted-foreground">
              {plannedBatches.length} batch{plannedBatches.length === 1 ? '' : 'es'} will be submitted sequentially.
            </p>
          </div>

          <div className="rounded-lg border border-dashed p-4">
            <p className="text-sm font-semibold text-slate-700">Included scenarios</p>
            <ul className="mt-2 grid gap-2 text-sm text-slate-600 md:grid-cols-2">
              {draftSeeds.map((seed) => (
                <li key={`${seed.entity_type}:${seed.entity_name}`}>{seed.display_name || seed.entity_name}</li>
              ))}
            </ul>
          </div>

          {draftSeeds.length > 0 && activeSeed && (
            <div className="grid gap-4 lg:grid-cols-[260px,1fr]">
              <div className="space-y-2">
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Review scenarios</p>
                <div className="rounded-lg border">
                  {draftSeeds.map((seed, index) => {
                    const key = `${seed.entity_type}:${seed.entity_name}`
                    const isActive = index === activeSeedIndex
                    return (
                      <button
                        type="button"
                        key={key}
                        className={`block w-full border-b px-3 py-2 text-left text-sm ${isActive ? 'bg-violet-50 font-semibold text-violet-900' : 'hover:bg-slate-50'}`}
                        onClick={() => setActiveSeedIndex(index)}
                      >
                        {seed.display_name || seed.entity_name}
                      </button>
                    )
                  })}
                </div>
              </div>

              <div className="rounded-xl border p-4 space-y-4">
                <div>
                  <p className="text-xs font-semibold text-slate-500">Title</p>
                  <Input value={activeSeed.title} onChange={(event) => updateActiveSeed({ title: event.target.value })} className="mt-1" />
                </div>
                <div>
                  <p className="text-xs font-semibold text-slate-500">Description</p>
                  <Textarea
                    value={activeSeed.description}
                    onChange={(event) => updateActiveSeed({ description: event.target.value })}
                    rows={6}
                    className="mt-1"
                  />
                </div>
                <div className="grid gap-3 md:grid-cols-2">
                  <div>
                    <p className="text-xs font-semibold text-slate-500">Summary</p>
                    <p className="mt-1 text-sm text-slate-700">{activeSeed.summary || 'No summary provided'}</p>
                  </div>
                  <div>
                    <p className="text-xs font-semibold text-slate-500">Tags</p>
                    <div className="mt-1 flex flex-wrap gap-2 text-xs">
                      {(activeSeed.tags || []).length === 0 && <span className="text-muted-foreground">None</span>}
                      {activeSeed.tags?.map((tag) => (
                        <Badge key={tag} variant="secondary">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  </div>
                </div>
                <div>
                  <p className="text-xs font-semibold text-slate-500">Issue categories</p>
                  <div className="mt-2 max-h-48 space-y-2 overflow-y-auto">
                    {activeSeed.categories.map((category, index) => (
                      <div
                        key={category.id ?? `${category.title}-${index}`}
                        className="rounded border border-slate-200 p-2 text-xs text-slate-700"
                      >
                        <div className="flex items-center justify-between text-sm font-semibold text-slate-900">
                          <span>{category.title}</span>
                          <Badge variant="outline">{category.items.length}</Badge>
                        </div>
                        {category.description && <p className="text-[11px] text-muted-foreground">{category.description}</p>}
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          )}

          {batches.length > 0 && (
            <div className="space-y-3">
              {batches.map((batch) => (
                <div key={batch.id} className="rounded-lg border border-slate-200">
                  <div className="flex items-center justify-between border-b bg-slate-50 px-4 py-2 text-sm font-semibold text-slate-700">
                    <span>{batch.id.toUpperCase()}</span>
                    <span className="text-xs text-muted-foreground">{batch.status === 'completed' ? 'Done' : batch.status === 'running' ? 'In progress…' : 'Pending'}</span>
                  </div>
                  <ul className="divide-y text-sm">
                    {batch.results.map((result) => (
                      <li key={`${batch.id}-${result.seed.entity_name}`} className="flex items-center justify-between px-4 py-2">
                        <div>
                          <p className="font-medium text-slate-900">{result.seed.display_name || result.seed.entity_name}</p>
                          <p className="text-xs text-muted-foreground">{result.message}</p>
                        </div>
                        {result.status === 'success' ? (
                          <span className="text-xs text-emerald-600">{result.issueId}</span>
                        ) : (
                          <span className="text-xs text-rose-600">Error</span>
                        )}
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          )}
        </div>

        <Separator />
        <div className="flex items-center justify-end gap-3 px-6 py-4">
          <Button variant="ghost" onClick={close}>
            Close
          </Button>
          <Button onClick={handleSubmit} disabled={submitting || draftSeeds.length === 0} className="gap-2">
            {submitting ? <Loader2 size={16} className="animate-spin" /> : 'Submit batches'}
          </Button>
        </div>
      </div>
    </div>,
    document.body,
  )
}
