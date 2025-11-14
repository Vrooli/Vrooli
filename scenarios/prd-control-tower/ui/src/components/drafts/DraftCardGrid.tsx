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
            className={cn('cursor-pointer transition hover:-translate-y-0.5', isSelected && 'border-violet-500 shadow-lg')}
          >
            <CardHeader className="flex flex-row items-start justify-between gap-4">
              <CardTitle className="text-xl">{draft.entity_name}</CardTitle>
              <Badge variant="secondary" className="capitalize">
                {draft.entity_type}
              </Badge>
            </CardHeader>
            <CardContent className="space-y-3 text-sm text-muted-foreground">
              {draft.owner && (
                <div className="flex items-center gap-2">
                  <User size={14} />
                  <span>{draft.owner}</span>
                </div>
              )}
              <div className="flex items-center gap-2">
                <Clock size={14} />
                <span>Updated {formatDate(draft.updated_at)}</span>
              </div>
              <div className="flex items-center gap-2">
                <FileEdit size={14} />
                <span>{formatFileSize(draft.content.length)} KB</span>
              </div>
            </CardContent>
            <CardFooter className="flex items-center justify-between border-t border-dashed pt-4">
              <Badge variant={statusVariant} className="gap-1 capitalize">
                <FileEdit size={14} /> {draft.status}
              </Badge>
              <div className="flex items-center gap-3" onClick={(event) => event.stopPropagation()}>
                <Button variant="link" className="px-0" asChild>
                  <Link to={draftPath}>Edit</Link>
                </Button>
                <Button
                  variant="link"
                  className="px-0 text-destructive"
                  onClick={() => onDeleteDraft(draft.id)}
                  title="Delete draft"
                >
                  <Trash2 size={16} />
                </Button>
              </div>
            </CardFooter>
          </Card>
        )
      })}
    </div>
  )
}
