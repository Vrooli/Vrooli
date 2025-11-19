import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import toast from 'react-hot-toast'
import { ClipboardCopy, Loader2 } from 'lucide-react'
import type { EntityType, IssueReportCategorySeed, IssueReportSeed, ScenarioIssueReportRequest } from '../../types'
import { bulkSubmitIssueReports } from '../../services/issues'
import { scenarioIssuesStore } from '../../state/scenarioIssuesStore'
import { Button } from '../ui/button'
import { Separator } from '../ui/separator'
import { Badge } from '../ui/badge'
import { Textarea } from '../ui/textarea'
import { ISSUE_DOCUMENTATION_LIBRARY, type DocumentationLink } from './issueDocumentation'

interface BulkIssueReportDialogProps {
  open: boolean
  seeds: IssueReportSeed[]
  onOpenChange?: (open: boolean) => void
}

interface BatchResult {
  id: string
  scenarioNames: string[]
  status: 'pending' | 'running' | 'completed'
  result?: { status: 'success' | 'error'; message: string; issueId?: string }
}

interface BatchPreview {
  description: string
  scenarioCount: number
  categoryCount: number
  docLinks: DocumentationLink[]
}

interface BatchCategoryScenarioEntry {
  key: string
  displayName: string
  entityName: string
  entityType: EntityType
  issueCount: number
  detailSummary?: string
}

interface BatchCategorySummary {
  id: string
  title: string
  description?: string
  severity?: IssueReportCategorySeed['severity']
  scenarios: BatchCategoryScenarioEntry[]
}

interface BatchRequestPayload {
  id: string
  request: ScenarioIssueReportRequest
  scenarios: BatchCategoryScenarioEntry[]
}

const MAX_BATCH_SIZE = 20

function makeScenarioKey(seed: IssueReportSeed) {
  return `${seed.entity_type}:${seed.entity_name}`
}

function slugify(value: string) {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

function resolveCategoryKey(category: IssueReportCategorySeed, index: number) {
  if (category.id) return category.id
  const label = typeof category.title === 'string' && category.title ? category.title : `category-${index}`
  const slug = slugify(label || `category-${index}`)
  return slug || `category-${index}`
}

function summarizeList(items: string[], limit = 3) {
  if (!items.length) return ''
  const picked = items.slice(0, limit)
  const remainder = items.length - picked.length
  return `${picked.join(', ')}${remainder > 0 ? ` +${remainder} more` : ''}`
}

function buildBatchCategorySummaries(batchSeeds: IssueReportSeed[]): BatchCategorySummary[] {
  const map = new Map<string, BatchCategorySummary>()
  batchSeeds.forEach((seed) => {
    const scenarioKey = makeScenarioKey(seed)
    const displayName = seed.display_name || seed.entity_name
    seed.categories.forEach((category, index) => {
      const categoryId = resolveCategoryKey(category, index)
      const current = map.get(categoryId)
      if (!current) {
        map.set(categoryId, {
          id: categoryId,
          title: typeof category.title === 'string' ? category.title : category.title ?? `Category ${index + 1}`,
          description: category.description,
          severity: category.severity,
          scenarios: [
            {
              key: scenarioKey,
              displayName,
              entityName: seed.entity_name,
              entityType: seed.entity_type,
              issueCount: category.items.length,
              detailSummary: summarizeList(category.items.map((item) => item.title || item.detail), 3),
            },
          ],
        })
      } else {
        current.scenarios.push({
          key: scenarioKey,
          displayName,
          entityName: seed.entity_name,
          entityType: seed.entity_type,
          issueCount: category.items.length,
          detailSummary: summarizeList(category.items.map((item) => item.title || item.detail), 3),
        })
      }
    })
  })
  return Array.from(map.values())
}

function collectDocumentationLinks(categoryIds: string[]) {
  const seen = new Set<string>()
  const docs: DocumentationLink[] = []
  categoryIds.forEach((categoryId) => {
    const links = ISSUE_DOCUMENTATION_LIBRARY[categoryId] ?? []
    links.forEach((link) => {
      const key = `${link.title}-${link.path}`
      if (!seen.has(key)) {
        docs.push(link)
        seen.add(key)
      }
    })
  })
  return docs
}

function getSelectedCategories(
  categories: BatchCategorySummary[],
  selection: Record<string, Set<string>>,
): BatchCategorySummary[] {
  return categories
    .map((category) => {
      const selectedScenarios = category.scenarios.filter((scenario) => selection?.[category.id]?.has(scenario.key))
      if (selectedScenarios.length === 0) {
        return null
      }
      return { ...category, scenarios: selectedScenarios }
    })
    .filter((entry): entry is BatchCategorySummary => Boolean(entry))
}

function buildBatchPreview(
  batchIndex: number,
  categories: BatchCategorySummary[],
  selection: Record<string, Set<string>>,
): BatchPreview {
  const included = getSelectedCategories(categories, selection)

  if (included.length === 0) {
    return {
      description: 'No scenarios selected for this batch. Enable at least one scenario within an issue category to generate a report.',
      scenarioCount: 0,
      categoryCount: 0,
      docLinks: [],
    }
  }

  const scenarioKeys = new Set<string>()
  included.forEach((category) => category.scenarios.forEach((scenario) => scenarioKeys.add(scenario.key)))
  const docLinks = collectDocumentationLinks(included.map((category) => category.id))
  const sections: string[] = []
  sections.push(
    `Batch ${batchIndex + 1}: ${scenarioKeys.size} scenario${scenarioKeys.size === 1 ? '' : 's'} flagged across ${included.length} issue categor${
      included.length === 1 ? 'y' : 'ies'
    }.`,
  )
  included.forEach((category) => {
    sections.push(`### ${category.title}`)
    if (category.description) {
      sections.push(category.description)
    }
    category.scenarios.forEach((scenario) => {
      const detail = scenario.detailSummary ? ` – ${scenario.detailSummary}` : ''
      sections.push(`- ${scenario.displayName} (${scenario.issueCount} issue${scenario.issueCount === 1 ? '' : 's'})${detail}`)
    })
  })
  if (docLinks.length > 0) {
    sections.push('### Documentation kit', ...docLinks.map((link) => `- ${link.title}: ${link.summary} (${link.path})`))
  }
  sections.push(
    '### Next steps',
    '1. Update the PRD.md / requirements for every listed scenario per the category guidance above.',
    '2. Rerun PRD Control Tower → Quality Scanner after edits to confirm the counters drop to zero.',
  )

  return {
    description: sections.join('\n\n'),
    scenarioCount: scenarioKeys.size,
    categoryCount: included.length,
    docLinks,
  }
}

function buildBatchRequest(
  batchIndex: number,
  categories: BatchCategorySummary[],
  selection: Record<string, Set<string>>,
  preview: BatchPreview,
  seeds: IssueReportSeed[],
): BatchRequestPayload | null {
  const included = getSelectedCategories(categories, selection)
  if (included.length === 0) {
    return null
  }

  const scenarioMap = new Map<string, BatchCategoryScenarioEntry>()
  included.forEach((category) => {
    category.scenarios.forEach((scenario) => {
      if (!scenarioMap.has(scenario.key)) {
        scenarioMap.set(scenario.key, scenario)
      }
    })
  })

  const entityTypes = new Set<EntityType>(Array.from(scenarioMap.values()).map((scenario) => scenario.entityType))
  const entityType: EntityType = entityTypes.size === 1 ? Array.from(entityTypes)[0] : 'scenario'
  const entityName = `bulk-batch-${batchIndex + 1}-${Date.now()}`
  const baseSeed = seeds[0]
  const tags = new Set<string>(['quality_scan', 'bulk_batch'])
  seeds.forEach((seed) => seed.tags?.forEach((tag) => tags.add(tag)))

  const selections = included.map((category) => ({
    id: `${category.id}-batch-${batchIndex + 1}`,
    title: category.title,
    detail: category.scenarios.map((scenario) => scenario.displayName).join(', '),
    category: category.id,
    severity: category.severity || 'medium',
    reference: '',
    notes: category.description || '',
  }))

  const metadata: Record<string, string> = {
    batch_index: String(batchIndex + 1),
    scenario_count: String(scenarioMap.size),
    category_count: String(included.length),
    scenarios: Array.from(scenarioMap.values())
      .map((scenario) => scenario.displayName)
      .join(', '),
  }

  const request: ScenarioIssueReportRequest = {
    entity_type: entityType,
    entity_name: entityName,
    source: baseSeed?.source ? `${baseSeed.source}_bulk` : 'quality_scanner_bulk',
    title: `Quality remediation batch ${batchIndex + 1}`,
    description: preview.description,
    summary: `Bulk batch ${batchIndex + 1}: ${scenarioMap.size} scenario${scenarioMap.size === 1 ? '' : 's'}`,
    tags: Array.from(tags),
    labels: baseSeed?.labels,
    metadata: { ...(baseSeed?.metadata ?? {}), ...metadata },
    selections,
    attachments: baseSeed?.attachments,
  }

  return {
    id: `batch-${batchIndex + 1}`,
    request,
    scenarios: Array.from(scenarioMap.values()),
  }
}

export function BulkIssueReportDialog({ open, seeds, onOpenChange }: BulkIssueReportDialogProps) {
  const [batchSize, setBatchSize] = useState(5)
  const [submitting, setSubmitting] = useState(false)
  const [batches, setBatches] = useState<BatchResult[]>([])
  const [draftSeeds, setDraftSeeds] = useState<IssueReportSeed[]>([])
  const [activeBatchIndex, setActiveBatchIndex] = useState(0)
  const [batchSelections, setBatchSelections] = useState<Record<string, Set<string>>[]>([])
  const [collapsedCategories, setCollapsedCategories] = useState<Record<number, Set<string>>>({})
  const structureSignatureRef = useRef('')
  const categoryCheckboxRefs = useRef<Record<string, HTMLInputElement | null>>({})

  const resetDialogState = useCallback(
    (seedList: IssueReportSeed[]) => {
      setBatches([])
      setSubmitting(false)
      setDraftSeeds(seedList.map((seed) => ({ ...seed })))
      setActiveBatchIndex(0)
    },
    [],
  )

  useEffect(() => {
    if (!open) {
      return
    }
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

  const batchCategorySummaries = useMemo(
    () => plannedBatches.map((batch) => buildBatchCategorySummaries(batch)),
    [plannedBatches],
  )

  const structureSignature = useMemo(() => {
    return batchCategorySummaries
      .map((categories) =>
        categories
          .map((category) => `${category.id}:${category.scenarios.map((scenario) => scenario.key).join(',')}`)
          .join('|'),
      )
      .join('||')
  }, [batchCategorySummaries])

  useEffect(() => {
    if (structureSignature === structureSignatureRef.current) {
      return
    }
    structureSignatureRef.current = structureSignature
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setBatchSelections(
      batchCategorySummaries.map((categories) => {
        const selection: Record<string, Set<string>> = {}
        categories.forEach((category) => {
          selection[category.id] = new Set(category.scenarios.map((scenario) => scenario.key))
        })
        return selection
      }),
    )
  }, [batchCategorySummaries, structureSignature])

  useEffect(() => {
    if (activeBatchIndex >= plannedBatches.length) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setActiveBatchIndex(plannedBatches.length > 0 ? plannedBatches.length - 1 : 0)
    }
  }, [activeBatchIndex, plannedBatches.length])

  useEffect(() => {
    categoryCheckboxRefs.current = {}
  }, [activeBatchIndex])

  const activeBatchCategories = useMemo(
    () => batchCategorySummaries[activeBatchIndex] ?? [],
    [batchCategorySummaries, activeBatchIndex],
  )
  const activeSelection = useMemo(
    () => batchSelections[activeBatchIndex] ?? {},
    [batchSelections, activeBatchIndex],
  )
  const activeBatchSeeds = useMemo(
    () => plannedBatches[activeBatchIndex] ?? [],
    [plannedBatches, activeBatchIndex],
  )

  const getSelectedScenarioKeys = useCallback(
    (batchIdx: number) => {
      const selection = batchSelections[batchIdx]
      if (!selection) return new Set<string>()
      const scenarioKeys = new Set<string>()
      Object.values(selection).forEach((set) => set.forEach((key) => scenarioKeys.add(key)))
      return scenarioKeys
    },
    [batchSelections],
  )

  const activeSelectedScenarioKeys = useMemo(
    () => getSelectedScenarioKeys(activeBatchIndex),
    [activeBatchIndex, getSelectedScenarioKeys],
  )

  const totalSelectedScenarioCount = useMemo(() => {
    return plannedBatches.reduce((count, _, idx) => count + getSelectedScenarioKeys(idx).size, 0)
  }, [getSelectedScenarioKeys, plannedBatches])

  const activeBatchPreview = useMemo(
    () => buildBatchPreview(activeBatchIndex, activeBatchCategories, activeSelection),
    [activeBatchCategories, activeBatchIndex, activeSelection],
  )

  useEffect(() => {
    const selection = batchSelections[activeBatchIndex]
    if (!selection) {
      return
    }
    activeBatchCategories.forEach((category) => {
      const ref = categoryCheckboxRefs.current[category.id]
      if (!ref) return
      const selectedSet = selection[category.id]
      const selectedCount = selectedSet ? category.scenarios.filter((scenario) => selectedSet.has(scenario.key)).length : 0
      ref.indeterminate = selectedCount > 0 && selectedCount < category.scenarios.length
    })
  }, [activeBatchCategories, activeBatchIndex, batchSelections])

  const close = () => onOpenChange?.(false)

  const modifyBatchSelection = (batchIdx: number, updater: (current: Record<string, Set<string>>) => Record<string, Set<string>>) => {
    setBatchSelections((prev) => {
      const next = [...prev]
      const current = next[batchIdx] ?? {}
      next[batchIdx] = updater(current)
      return next
    })
  }

  const setCategorySelection = (categoryId: string, shouldSelect: boolean) => {
    const category = activeBatchCategories.find((entry) => entry.id === categoryId)
    if (!category) return
    modifyBatchSelection(activeBatchIndex, (current) => {
      const next = { ...current }
      next[categoryId] = shouldSelect ? new Set(category.scenarios.map((scenario) => scenario.key)) : new Set()
      return next
    })
  }

  const toggleScenarioSelection = (categoryId: string, scenarioKey: string) => {
    modifyBatchSelection(activeBatchIndex, (current) => {
      const next = { ...current }
      const existing = new Set(next[categoryId] ?? [])
      if (existing.has(scenarioKey)) {
        existing.delete(scenarioKey)
      } else {
        existing.add(scenarioKey)
      }
      next[categoryId] = existing
      return next
    })
  }

  const toggleCategoryCollapsed = (categoryId: string) => {
    setCollapsedCategories((prev) => {
      const current = new Set(prev[activeBatchIndex] ?? [])
      if (current.has(categoryId)) {
        current.delete(categoryId)
      } else {
        current.add(categoryId)
      }
      return { ...prev, [activeBatchIndex]: current }
    })
  }

  const copyPreview = async () => {
    if (!activeBatchPreview.description.trim()) {
      toast.error('Nothing to copy yet')
      return
    }
    try {
      await navigator.clipboard.writeText(activeBatchPreview.description)
      toast.success('Copied batch preview')
    } catch (error) {
      console.error(error)
      toast.error('Unable to copy preview')
    }
  }

  const handleSubmit = async () => {
    if (totalSelectedScenarioCount === 0) {
      toast.error('Select at least one scenario to report')
      return
    }

    setSubmitting(true)

    const payloads = plannedBatches
      .map((batch, idx) => {
        const selection = batchSelections[idx] ?? {}
        const preview = buildBatchPreview(idx, batchCategorySummaries[idx] ?? [], selection)
        const payload = buildBatchRequest(idx, batchCategorySummaries[idx] ?? [], selection, preview, batch)
        return payload ? { payload, preview } : null
      })
      .filter((entry): entry is { payload: BatchRequestPayload; preview: BatchPreview } => Boolean(entry))

    if (payloads.length === 0) {
      toast.error('No batches have scenarios selected')
      setSubmitting(false)
      return
    }

    setBatches(
      payloads.map((entry) => ({
        id: entry.payload.id,
        scenarioNames: entry.payload.scenarios.map((scenario) => scenario.displayName),
        status: 'pending',
      })),
    )

    for (let index = 0; index < payloads.length; index += 1) {
      const { payload } = payloads[index]
      setBatches((prev) => prev.map((entry, idx) => (idx === index ? { ...entry, status: 'running' } : entry)))

      const results = await bulkSubmitIssueReports([payload.request])
      const [result] = results

      if (result?.response) {
        payload.scenarios.forEach((scenario) => {
          scenarioIssuesStore.flagIssueReported(scenario.entityType, scenario.entityName)
        })
      }

      setBatches((prev) =>
        prev.map((entry, idx) => {
          if (idx !== index) {
            return entry
          }
          return {
            ...entry,
            status: 'completed',
            result: {
              status: result?.response ? 'success' : 'error',
              message: result?.response ? 'Reported successfully' : result?.error || 'Failed',
              issueId: result?.response?.issue_id,
            },
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
      <div className="relative mt-10 w-full max-w-5xl rounded-2xl bg-white shadow-2xl">
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

          {plannedBatches.length > 0 && (
            <div className="rounded-lg border border-slate-200 p-4">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Batches</p>
              <div className="mt-3 flex flex-wrap gap-2">
                {plannedBatches.map((batch, idx) => {
                  const selectedCount = getSelectedScenarioKeys(idx).size
                  const isActive = idx === activeBatchIndex
                  return (
                    <button
                      type="button"
                      key={`batch-nav-${idx}`}
                      onClick={() => setActiveBatchIndex(idx)}
                      className={`rounded-lg border px-3 py-2 text-left text-xs ${
                        isActive ? 'border-violet-500 bg-violet-50 text-violet-900' : 'hover:bg-slate-50'
                      }`}
                    >
                      <p className="font-semibold">Batch {idx + 1}</p>
                      <p className="text-[11px] text-muted-foreground">
                        {selectedCount}/{batch.length} scenarios selected
                      </p>
                    </button>
                  )
                })}
              </div>
            </div>
          )}

          {plannedBatches.length > 0 && (
            <div className="grid gap-6 lg:grid-cols-[360px,1fr]">
              <div className="space-y-4 border-r pr-4">
                <div className="rounded-lg border border-slate-200 p-3 text-sm">
                  <p className="font-semibold text-slate-900">Batch {activeBatchIndex + 1}</p>
                  <p className="text-xs text-muted-foreground">
                    {activeSelectedScenarioKeys.size}/{activeBatchSeeds.length} scenarios selected
                  </p>
                </div>
                <div className="space-y-3 overflow-y-auto pr-2" style={{ maxHeight: '60vh' }}>
                  {activeBatchCategories.length === 0 && (
                    <p className="text-xs text-muted-foreground">No issue categories detected for this batch.</p>
                  )}
                  {activeBatchCategories.map((category) => {
                    const categorySelection = activeSelection?.[category.id] ?? new Set<string>()
                    const selectedCount = category.scenarios.filter((scenario) => categorySelection.has(scenario.key)).length
                    const isCollapsed = collapsedCategories[activeBatchIndex]?.has(category.id) ?? false
                    const severityLabel = category.severity?.toUpperCase()
                    return (
                      <div key={category.id} className="rounded-lg border p-3">
                        <div className="flex items-start justify-between gap-2">
                          <label className="flex flex-1 items-start gap-3 text-sm font-semibold text-slate-900">
                            <input
                              ref={(element) => {
                                categoryCheckboxRefs.current[category.id] = element
                              }}
                              type="checkbox"
                              className="mt-1 h-4 w-4 rounded border-slate-300"
                              checked={selectedCount === category.scenarios.length && category.scenarios.length > 0}
                              disabled={category.scenarios.length === 0}
                              onChange={(event) => setCategorySelection(category.id, event.target.checked)}
                            />
                            <span>
                              {category.title}
                              {category.description && (
                                <span className="mt-1 block text-xs font-normal text-muted-foreground">{category.description}</span>
                              )}
                              <span className="mt-1 block text-[11px] font-normal text-muted-foreground">
                                {selectedCount}/{category.scenarios.length} scenarios selected
                              </span>
                            </span>
                          </label>
                          <div className="flex flex-col items-end gap-2 text-xs">
                            {severityLabel && (
                              <Badge variant="outline" className="text-[11px]">
                                {severityLabel}
                              </Badge>
                            )}
                            <button
                              type="button"
                              className="text-primary underline-offset-2 hover:underline"
                              onClick={() => toggleCategoryCollapsed(category.id)}
                            >
                              {isCollapsed ? 'Expand' : 'Collapse'}
                            </button>
                          </div>
                        </div>
                        {!isCollapsed && (
                          <div className="mt-3 space-y-2">
                            {category.scenarios.map((scenario) => (
                              <label key={scenario.key} className="flex items-start gap-2 rounded border border-slate-200 p-3 text-xs">
                                <input
                                  type="checkbox"
                                  className="mt-1"
                                  checked={categorySelection.has(scenario.key)}
                                  onChange={() => toggleScenarioSelection(category.id, scenario.key)}
                                />
                                <div className="space-y-1">
                                  <p className="font-semibold text-slate-900">{scenario.displayName}</p>
                                  <p className="text-[11px] text-muted-foreground">
                                    {scenario.issueCount} issue{scenario.issueCount === 1 ? '' : 's'}
                                    {scenario.detailSummary && ` · ${scenario.detailSummary}`}
                                  </p>
                                </div>
                              </label>
                            ))}
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>

              <div className="space-y-4">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="text-xs font-semibold text-slate-500">Batch preview</p>
                    <p className="text-xs text-muted-foreground">
                      {activeBatchPreview.scenarioCount} scenario{activeBatchPreview.scenarioCount === 1 ? '' : 's'} ·{' '}
                      {activeBatchPreview.categoryCount} categor{activeBatchPreview.categoryCount === 1 ? 'y' : 'ies'} selected
                    </p>
                  </div>
                  <Button variant="outline" size="sm" onClick={copyPreview} disabled={!activeBatchPreview.description.trim()} className="gap-2">
                    <ClipboardCopy size={14} /> Copy preview
                  </Button>
                </div>
                <Textarea value={activeBatchPreview.description} readOnly rows={14} className="bg-slate-50" />
                {activeBatchPreview.docLinks.length > 0 && (
                  <div className="rounded-lg border border-dashed p-3 text-xs">
                    <p className="font-semibold text-slate-700">Documentation kit</p>
                    <ul className="mt-2 list-disc space-y-1 pl-4 text-slate-600">
                      {activeBatchPreview.docLinks.map((link) => (
                        <li key={link.path}>
                          <span className="font-semibold text-slate-900">{link.title}</span>: {link.summary} ({link.path})
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            </div>
          )}

          {batches.length > 0 && (
            <div className="space-y-3">
              {batches.map((batch) => (
                <div key={batch.id} className="rounded-lg border border-slate-200">
                  <div className="flex items-center justify-between border-b bg-slate-50 px-4 py-2 text-sm font-semibold text-slate-700">
                    <span>{batch.id.toUpperCase()}</span>
                    <span className="text-xs text-muted-foreground">
                      {batch.status === 'completed' ? 'Done' : batch.status === 'running' ? 'In progress…' : 'Pending'}
                    </span>
                  </div>
                  <div className="space-y-2 px-4 py-3 text-sm">
                    <p className="font-medium text-slate-900">Scenarios</p>
                    <p className="text-xs text-muted-foreground">{batch.scenarioNames.join(', ')}</p>
                    {batch.result && (
                      <div className="flex items-center justify-between text-xs">
                        <span className={batch.result.status === 'success' ? 'text-emerald-600' : 'text-rose-600'}>
                          {batch.result.message}
                        </span>
                        {batch.result.issueId && (
                          <span className="text-emerald-600">{batch.result.issueId}</span>
                        )}
                      </div>
                    )}
                  </div>
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
          <Button onClick={handleSubmit} disabled={submitting || draftSeeds.length === 0 || totalSelectedScenarioCount === 0} className="gap-2">
            {submitting ? <Loader2 size={16} className="animate-spin" /> : 'Submit batches'}
          </Button>
        </div>
      </div>
    </div>,
    document.body,
  )
}
