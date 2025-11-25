import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import toast from 'react-hot-toast'
import { StickyNote, ListChecks, Sparkles, ClipboardList } from 'lucide-react'
import { useConfirm } from '../utils/confirmDialog'
import {
  parseBacklogInput,
  PendingIdea,
  slugifyIdea,
  sanitizeSlugInput,
  createBacklogEntriesRequest,
  convertBacklogEntriesRequest,
} from '../utils/backlog'
import type { EntityType } from '../types'
import { useBacklogData } from '../hooks/useBacklogData'
import { BacklogIntakeCard, BacklogPreviewPanel, BacklogEntriesTable } from '../components/backlog'
import { Button } from '../components/ui/button'
import { Card } from '../components/ui/card'
import { TopNav } from '../components/ui/top-nav'

const entityOptions: Array<{ value: EntityType; label: string }> = [
  { value: 'scenario', label: 'Scenario' },
  { value: 'resource', label: 'Resource' },
]

type PreviewOverride = Partial<Pick<PendingIdea, 'entityType' | 'suggestedName' | 'notes'>>

type PreviewActionMode = 'save' | 'convert'

export default function Backlog() {
  const confirm = useConfirm()
  const [rawInput, setRawInput] = useState('')
  const [previewOverrides, setPreviewOverrides] = useState<Record<string, PreviewOverride>>({})
  const [previewSelection, setPreviewSelection] = useState<Set<string>>(() => new Set())
  const [previewBusy, setPreviewBusy] = useState(false)
  const [previewBusyId, setPreviewBusyId] = useState<string | null>(null)

  const {
    backlogEntries,
    filteredBacklogEntries,
    loadingBacklog,
    listFilter,
    setListFilter,
    backlogSelection,
    toggleBacklogSelection,
    toggleBacklogSelectAll,
    backlogBusy,
    backlogBusyId,
    convertExistingEntries,
    deleteEntries,
    refreshBacklog,
    updateBacklogEntryNotes,
  } = useBacklogData({ confirm })

  const parsedIdeas = useMemo(() => parseBacklogInput(rawInput, 'scenario'), [rawInput])

  useEffect(() => {
    const validIds = new Set(parsedIdeas.map((idea) => idea.id))
    setPreviewOverrides((prev) => {
      const next: Record<string, PreviewOverride> = {}
      parsedIdeas.forEach((idea) => {
        if (prev[idea.id]) {
          next[idea.id] = prev[idea.id]
        }
      })
      return next
    })
    setPreviewSelection((prev) => {
      const next = new Set<string>()
      prev.forEach((id) => {
        if (validIds.has(id)) {
          next.add(id)
        }
      })
      return next
    })
  }, [parsedIdeas])

  const previewIdeas = useMemo(() => {
    return parsedIdeas.map((idea) => {
      const override = previewOverrides[idea.id]
      if (!override) {
        return idea
      }
      return {
        ...idea,
        ...override,
      }
    })
  }, [parsedIdeas, previewOverrides])

  const handlePreviewSelection = useCallback((id: string) => {
    setPreviewSelection((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  const toggleSelectAllPreview = useCallback(() => {
    if (previewSelection.size === previewIdeas.length) {
      setPreviewSelection(new Set())
      return
    }
    setPreviewSelection(new Set(previewIdeas.map((idea) => idea.id)))
  }, [previewSelection.size, previewIdeas])

  const updateOverride = useCallback((id: string, patch: PreviewOverride) => {
    setPreviewOverrides((prev) => ({
      ...prev,
      [id]: {
        ...prev[id],
        ...patch,
      },
    }))
  }, [])

  const handlePreviewEntityTypeChange = useCallback(
    (id: string, entityType: EntityType) => {
      updateOverride(id, { entityType })
    },
    [updateOverride],
  )

  const handlePreviewSlugChange = useCallback(
    (id: string, slug: string) => {
      updateOverride(id, { suggestedName: sanitizeSlugInput(slug) })
    },
    [updateOverride],
  )

  const handlePreviewNotesChange = useCallback(
    (id: string, notes: string) => {
      updateOverride(id, { notes })
    },
    [updateOverride],
  )

  const updateEntryNotes = useCallback(
    (id: string, notes: string) => updateBacklogEntryNotes(id, notes),
    [updateBacklogEntryNotes],
  )

  const handlePreviewAction = useCallback(
    async (mode: PreviewActionMode, explicitIdea?: PendingIdea) => {
      const ideas = explicitIdea ? [explicitIdea] : previewIdeas.filter((idea) => previewSelection.has(idea.id))
      if (ideas.length === 0) {
        toast.error('Select at least one idea to continue')
        return
      }

      setPreviewBusy(true)
      setPreviewBusyId(explicitIdea ? explicitIdea.id : null)
      try {
        const created = await createBacklogEntriesRequest(
          ideas.map((idea) => ({
            ...idea,
            suggestedName: idea.suggestedName || slugifyIdea(idea.ideaText),
          })),
        )
        if (mode === 'convert') {
          const results = await convertBacklogEntriesRequest(created.map((entry) => entry.id))
          const successes = results.filter((result) => !result.error)
          const failures = results.length - successes.length
          if (successes.length) {
            toast.success(`Created ${successes.length} draft${successes.length === 1 ? '' : 's'} from backlog ideas`)
          }
          if (failures > 0) {
            toast.error(`${failures} item${failures === 1 ? '' : 's'} failed to convert. Check logs for details.`)
          }
        } else {
          toast.success(`Added ${created.length} idea${created.length === 1 ? '' : 's'} to backlog`)
        }
        await refreshBacklog()
        if (!explicitIdea) {
          setPreviewSelection(new Set())
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Unexpected backlog error'
        toast.error(message)
      } finally {
        setPreviewBusy(false)
        setPreviewBusyId(null)
      }
    },
    [previewIdeas, previewSelection, refreshBacklog],
  )

  return (
    <div className="app-container space-y-8" data-layout="dual">
      <TopNav />
      <header className="rounded-3xl border bg-white/90 p-4 sm:p-6 shadow-soft-lg">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div className="space-y-2">
            <span className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-500">Idea backlog</span>
            <div className="flex items-center gap-3 text-2xl sm:text-3xl font-semibold text-slate-900">
              <span className="rounded-2xl bg-amber-100 p-2 sm:p-3 text-amber-600">
                <StickyNote size={24} strokeWidth={2.5} className="sm:w-7 sm:h-7" />
              </span>
              Scenario Backlog
            </div>
            <p className="max-w-3xl text-sm sm:text-base text-muted-foreground">
              Paste raw notes, triage ideas, and convert them into drafts without leaving PRD Control Tower.
            </p>
          </div>
          <div className="flex flex-col sm:flex-row gap-2 sm:gap-3 w-full sm:w-auto">
            <Button asChild variant="secondary" size="lg" className="w-full sm:w-auto">
              <Link to="/catalog" className="flex items-center justify-center gap-2">
                <ClipboardList size={18} />
                <span className="hidden sm:inline">Catalog</span>
                <span className="sm:hidden">Catalog</span>
              </Link>
            </Button>
            <Button asChild size="lg" className="w-full sm:w-auto">
              <Link to="/drafts" className="flex items-center justify-center gap-2">
                <ListChecks size={18} />
                <span className="hidden sm:inline">Drafts</span>
                <span className="sm:hidden">Drafts</span>
              </Link>
            </Button>
          </div>
        </div>
        <Card className="mt-4 sm:mt-6 border-2 border-amber-200 bg-gradient-to-br from-amber-50 to-white p-4 sm:p-5">
          <div className="flex items-start gap-3 sm:gap-4 text-sm">
            <span className="rounded-xl bg-gradient-to-br from-amber-500 to-amber-600 p-2.5 sm:p-3 text-white shadow-md shrink-0">
              <Sparkles size={18} strokeWidth={2.5} className="sm:w-5 sm:h-5" />
            </span>
            <div className="space-y-1.5">
              <p className="font-bold text-base text-slate-900">Quick Workflow Guide</p>
              <p className="text-sm text-slate-700 leading-relaxed">
                <span className="font-semibold text-amber-700">Left:</span> Paste raw ideas, bullet lists, or brainstorming notes.
                <span className="mx-2">·</span>
                <span className="font-semibold text-amber-700">Right:</span> Preview, refine, and convert to drafts.
              </p>
            </div>
          </div>
        </Card>
      </header>

      <section className="grid gap-6 xl:grid-cols-2">
        <BacklogIntakeCard
          rawInput={rawInput}
          ideaCount={previewIdeas.length}
          onChange={setRawInput}
          onClear={() => setRawInput('')}
        />
        <BacklogPreviewPanel
          ideas={previewIdeas}
          entityOptions={entityOptions}
          selection={previewSelection}
          busy={previewBusy}
          busyId={previewBusyId}
          onToggleSelectAll={toggleSelectAllPreview}
          onToggleSelection={handlePreviewSelection}
          onChangeEntityType={handlePreviewEntityTypeChange}
          onChangeSlug={handlePreviewSlugChange}
          onChangeNotes={handlePreviewNotesChange}
          onRunAction={handlePreviewAction}
        />
      </section>

      <BacklogEntriesTable
        entries={filteredBacklogEntries}
        selection={backlogSelection}
        busy={backlogBusy}
        busyId={backlogBusyId}
        filter={listFilter}
        loading={loadingBacklog}
        totalCount={backlogEntries.length}
        onFilterChange={setListFilter}
        onToggleSelection={toggleBacklogSelection}
        onToggleSelectAll={toggleBacklogSelectAll}
        onConvert={convertExistingEntries}
        onDelete={deleteEntries}
        onRefresh={refreshBacklog}
        onUpdateNotes={updateEntryNotes}
      />

      <div className="flex flex-wrap gap-3">
        <Button variant="ghost" size="sm" asChild className="gap-2">
          <Link to="/catalog">
            <ClipboardList size={16} />
            Catalog
          </Link>
        </Button>
        <Button variant="ghost" size="sm" asChild className="gap-2">
          <Link to="/drafts">
            <ListChecks size={16} />
            Drafts
          </Link>
        </Button>
      </div>
    </div>
  )
}
