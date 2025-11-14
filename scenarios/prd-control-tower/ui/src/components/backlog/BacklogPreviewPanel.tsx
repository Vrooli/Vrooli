import { Loader2, CheckCircle2, Inbox } from 'lucide-react'
import type { EntityType } from '../../types'
import type { PendingIdea } from '../../utils/backlog'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '../ui/card'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { cn } from '../../lib/utils'

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
}: BacklogPreviewPanelProps) {
  const selectedCount = selection.size

  return (
    <Card className="h-full">
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div className="space-y-1">
          <CardTitle className="text-2xl font-semibold">Live preview</CardTitle>
          <CardDescription>Select entries to save in backlog or instantly create drafts.</CardDescription>
        </div>
        <div className="flex flex-col items-end gap-2 text-sm text-muted-foreground">
          <Button variant="link" className="h-auto px-0" onClick={onToggleSelectAll}>
            {selection.size === ideas.length ? 'Clear selection' : 'Select all'}
          </Button>
          <Badge variant="secondary" className="rounded-lg px-2 py-1 text-xs">
            {selectedCount} selected
          </Badge>
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
      <CardFooter className="flex flex-col gap-3 border-t border-dashed pt-4 text-sm sm:flex-row sm:items-center sm:justify-between">
        <span className="text-muted-foreground">
          {selectedCount} of {ideas.length} selected
        </span>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Button variant="secondary" disabled={ideas.length === 0 || busy} onClick={() => onRunAction('save')}>
            {busy && busyId === null && <Loader2 size={16} className="mr-2 animate-spin" />}
            Save to backlog
          </Button>
          <Button disabled={ideas.length === 0 || busy} onClick={() => onRunAction('convert')}>
            {busy && busyId === null ? <Loader2 size={16} className="mr-2 animate-spin" /> : <CheckCircle2 size={16} className="mr-2" />}
            Convert to drafts
          </Button>
        </div>
      </CardFooter>
    </Card>
  )
}
