import { Loader2, CheckCircle2, Inbox } from 'lucide-react'
import type { EntityType } from '../../types'
import type { PendingIdea } from '../../utils/backlog'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '../ui/card'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { cn } from '../../lib/utils'
import { selectors } from '../../consts/selectors'

interface BacklogPreviewPanelProps {
  ideas: PendingIdea[]
  entityOptions: Array<{ value: EntityType; label: string }>
  selection: Set<string>
  busy: boolean
  busyId: string | null
  onToggleSelectAll: () => void
  onToggleSelection: (id: string) => void
  onChangeEntityType: (id: string, entityType: EntityType) => void
  onChangeSlug: (id: string, slug: string) => void
  onChangeNotes: (id: string, notes: string) => void
  onRunAction: (mode: 'save' | 'convert', idea?: PendingIdea) => void
}

export function BacklogPreviewPanel({
  ideas,
  entityOptions,
  selection,
  busy,
  busyId,
  onToggleSelectAll,
  onToggleSelection,
  onChangeEntityType,
  onChangeSlug,
  onRunAction,
  onChangeNotes,
}: BacklogPreviewPanelProps) {
  const selectedCount = selection.size

  return (
    <Card data-testid={selectors.backlog.previewPanel} className="h-full">
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div className="space-y-1 flex-1">
          <CardTitle className="text-2xl font-semibold">Live preview</CardTitle>
          <CardDescription>Select entries to save in backlog or instantly create drafts.</CardDescription>
        </div>
        <div className="flex flex-col items-end gap-2">
          <Button
            data-testid={selectors.backlog.selectAllButton}
            variant={selection.size > 0 ? "secondary" : "outline"}
            size="sm"
            onClick={onToggleSelectAll}
            className="gap-2 min-w-[140px] justify-between font-medium"
          >
            <span>{selection.size === ideas.length && ideas.length > 0 ? 'Clear all' : 'Select all'}</span>
            <Badge variant={selection.size > 0 ? "default" : "secondary"} className="rounded-full px-2 py-0.5 text-xs font-semibold">
              {selectedCount}/{ideas.length}
            </Badge>
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {ideas.length === 0 ? (
          <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed bg-slate-50 py-12 text-center text-muted-foreground">
            <Inbox size={42} />
            <p>Paste ideas to see them parsed here.</p>
          </div>
        ) : (
          <div className="space-y-3">
            {ideas.map((idea) => {
              const isSelected = selection.has(idea.id)
              const ideaBusy = busy && busyId === idea.id
              return (
                <div
                  key={idea.id}
                  className={cn(
                    'flex gap-3 rounded-2xl border bg-white/70 p-4 shadow-sm transition',
                    isSelected ? 'border-violet-400 bg-violet-50/80 shadow' : 'border-slate-200',
                  )}
                >
                  <label className="mt-1">
                    <input
                      type="checkbox"
                      className="h-4 w-4 accent-violet-500"
                      checked={isSelected}
                      onChange={() => onToggleSelection(idea.id)}
                    />
                  </label>
                  <div className="flex-1 space-y-3">
                    <p className="font-medium text-slate-900">{idea.ideaText}</p>
                    <div className="flex flex-col gap-3 md:flex-row md:items-center">
                      <select
                        value={idea.entityType}
                        onChange={(event) => onChangeEntityType(idea.id, event.target.value as EntityType)}
                        className="rounded-xl border border-input bg-white px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      >
                        {entityOptions.map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </select>
                      <input
                        type="text"
                        className="flex-1 rounded-xl border border-input bg-white px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        value={idea.suggestedName}
                        onChange={(event) => onChangeSlug(idea.id, event.target.value)}
                        placeholder="slug"
                      />
                    </div>
                    <textarea
                      rows={3}
                      className="w-full rounded-xl border border-input bg-white px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      placeholder="Add optional context or notes..."
                      value={idea.notes}
                      onChange={(event) => onChangeNotes(idea.id, event.target.value)}
                    />
                    <div className="flex flex-wrap items-center gap-3">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => onRunAction('convert', idea)}
                        disabled={busy && !ideaBusy}
                      >
                        {ideaBusy ? <Loader2 size={16} className="animate-spin" /> : <CheckCircle2 size={16} />}
                        <span className="ml-2">Create draft</span>
                      </Button>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </CardContent>
      <CardFooter className="flex flex-col gap-4 border-t bg-gradient-to-b from-white to-slate-50 pt-5 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-slate-700">
            {selectedCount > 0 ? (
              <>
                <span className="text-lg font-bold text-violet-600">{selectedCount}</span> of {ideas.length} selected
              </>
            ) : (
              <span className="text-muted-foreground">No items selected</span>
            )}
          </span>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Button
            data-testid={selectors.backlog.saveButton}
            variant="outline"
            disabled={selectedCount === 0 || busy}
            onClick={() => onRunAction('save')}
            className="gap-2 border-slate-300 hover:border-slate-400"
          >
            {busy && busyId === null && <Loader2 size={16} className="animate-spin" />}
            Save to backlog
          </Button>
          <Button
            data-testid={selectors.backlog.convertButton}
            disabled={selectedCount === 0 || busy}
            onClick={() => onRunAction('convert')}
            size="lg"
            className="gap-2 bg-gradient-to-r from-violet-600 to-violet-700 font-semibold text-white shadow-sm hover:from-violet-700 hover:to-violet-800 hover:shadow-md"
          >
            {busy && busyId === null ? <Loader2 size={16} className="animate-spin" /> : <CheckCircle2 size={16} />}
            Convert to drafts
          </Button>
        </div>
      </CardFooter>
    </Card>
  )
}
