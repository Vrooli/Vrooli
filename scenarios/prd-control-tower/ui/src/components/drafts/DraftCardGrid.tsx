import { Link } from 'react-router-dom'
import { FileEdit, User, Clock, Trash2 } from 'lucide-react'
import type { Draft } from '../../types'
import { formatDate, formatFileSize } from '../../utils/formatters'
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '../ui/card'
import { Badge } from '../ui/badge'
import { Button } from '../ui/button'
import { cn } from '../../lib/utils'

export interface DraftCardGridProps {
  drafts: Draft[]
  selectedDraftId?: string | null
  onSelectDraft: (draft: Draft) => void
  onDeleteDraft: (draftId: string) => void
}

export function DraftCardGrid({ drafts, selectedDraftId, onSelectDraft, onDeleteDraft }: DraftCardGridProps) {
  if (drafts.length === 0) {
    return null
  }

  return (
    <div className="grid gap-4 md:grid-cols-2">
      {drafts.map((draft) => {
        const encodedName = encodeURIComponent(draft.entity_name)
        const draftPath = `/draft/${draft.entity_type}/${encodedName}`
        const isSelected = selectedDraftId === draft.id
        const statusVariant = draft.status === 'draft' ? 'warning' : 'success'

        return (
          <Card
            key={draft.id}
            role="button"
            tabIndex={0}
            aria-pressed={isSelected}
            onClick={() => onSelectDraft(draft)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                onSelectDraft(draft)
              }
            }}
            className={cn(
              'group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl hover:border-violet-400 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-violet-200 focus-visible:ring-offset-2',
              isSelected && 'border-violet-500 shadow-xl ring-2 ring-violet-300 ring-offset-2'
            )}
          >
            <CardHeader className="flex flex-row items-start justify-between gap-4 pb-3">
              <CardTitle className="text-lg sm:text-xl leading-tight break-words group-hover:text-violet-600 transition-colors">{draft.entity_name}</CardTitle>
              <Badge variant="secondary" className="capitalize shrink-0 text-xs group-hover:bg-violet-100 group-hover:text-violet-700 transition-colors">
                {draft.entity_type}
              </Badge>
            </CardHeader>
            <CardContent className="space-y-2.5 text-sm text-muted-foreground pb-3">
              {draft.owner && (
                <div className="flex items-center gap-2">
                  <User size={15} className="shrink-0 text-slate-400" />
                  <span className="truncate">{draft.owner}</span>
                </div>
              )}
              <div className="flex items-center gap-2">
                <Clock size={15} className="shrink-0 text-slate-400" />
                <span>Updated {formatDate(draft.updated_at)}</span>
              </div>
              <div className="flex items-center gap-2">
                <FileEdit size={15} className="shrink-0 text-slate-400" />
                <span>{formatFileSize(draft.content.length)} KB</span>
              </div>
            </CardContent>
            <CardFooter className="flex items-center justify-between gap-3 border-t border-dashed pt-3.5 bg-gradient-to-r from-transparent to-slate-50/50">
              <Badge variant={statusVariant} className="gap-1.5 capitalize text-xs font-medium px-2.5 py-1.5 shadow-sm">
                <FileEdit size={13} /> {draft.status}
              </Badge>
              <div className="flex items-center gap-2.5" onClick={(event) => event.stopPropagation()}>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 gap-1.5 font-semibold hover:bg-violet-100 hover:text-violet-700 transition-colors"
                  asChild
                >
                  <Link to={draftPath}>
                    <FileEdit size={14} />
                    <span>Edit</span>
                  </Link>
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 w-8 p-0 text-destructive hover:bg-rose-100 hover:text-rose-700 hover:scale-110 transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-rose-400 focus-visible:ring-offset-1"
                  onClick={() => onDeleteDraft(draft.id)}
                  title="Delete draft"
                  aria-label={`Delete draft for ${draft.entity_name}`}
                >
                  <Trash2 size={15} />
                </Button>
              </div>
            </CardFooter>
          </Card>
        )
      })}
    </div>
  )
}
