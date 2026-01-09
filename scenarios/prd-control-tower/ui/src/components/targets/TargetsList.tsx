import { CheckCircle2, Circle, Target } from 'lucide-react'
import type { OperationalTarget } from '../../types'
import { Badge } from '../ui/badge'
import { cn } from '../../lib/utils'
import { selectors } from '../../consts/selectors'

interface TargetsListProps {
  targets: OperationalTarget[]
  selectedId: string | null
  onSelect: (target: OperationalTarget) => void
}

export function TargetsList({ targets, selectedId, onSelect }: TargetsListProps) {
  if (targets.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center p-12 text-center text-muted-foreground">
        <Target size={48} className="mb-4 text-slate-300" />
        <p>No operational targets found</p>
      </div>
    )
  }

  // Group targets by category
  const groupedTargets = targets.reduce(
    (acc, target) => {
      const category = target.category || 'Uncategorized'
      if (!acc[category]) {
        acc[category] = []
      }
      acc[category].push(target)
      return acc
    },
    {} as Record<string, OperationalTarget[]>,
  )

  const categoryOrder = ['Must Have', 'Should Have', 'Nice to Have', 'Uncategorized']
  const sortedCategories = Object.keys(groupedTargets).sort((a, b) => {
    const aIndex = categoryOrder.indexOf(a)
    const bIndex = categoryOrder.indexOf(b)
    if (aIndex === -1 && bIndex === -1) return a.localeCompare(b)
    if (aIndex === -1) return 1
    if (bIndex === -1) return -1
    return aIndex - bIndex
  })

  return (
    <div data-testid={selectors.requirements.targetsList} className="space-y-6 overflow-y-auto max-h-[calc(100vh-250px)]">
      {sortedCategories.map((category) => (
        <div key={category} className="space-y-2">
          <h3 className="text-sm font-semibold text-slate-700 px-2">{category}</h3>
          <div className="space-y-1">
            {groupedTargets[category].map((target) => (
              <div
                key={target.id}
                className={cn(
                  'flex items-start gap-3 rounded-lg px-3 py-3 cursor-pointer transition-colors',
                  selectedId === target.id
                    ? 'bg-blue-50 border-l-4 border-blue-500 shadow-sm'
                    : 'hover:bg-slate-50 border-l-4 border-transparent',
                )}
                onClick={() => onSelect(target)}
              >
                {target.status === 'complete' ? (
                  <CheckCircle2 size={18} className="text-green-600 mt-0.5 flex-shrink-0" />
                ) : (
                  <Circle size={18} className="text-slate-400 mt-0.5 flex-shrink-0" />
                )}
                <div className="flex-1 min-w-0">
                  <p className={cn('text-sm font-medium', selectedId === target.id ? 'text-blue-900' : 'text-slate-700')}>
                    {target.title}
                  </p>
                  {target.notes && <p className="text-xs text-muted-foreground mt-1 line-clamp-2">{target.notes}</p>}
                  <div className="flex flex-wrap gap-1 mt-2">
                    {target.criticality && (
                      <Badge
                        variant={target.criticality === 'P0' ? 'destructive' : target.criticality === 'P1' ? 'warning' : 'secondary'}
                        className="text-xs"
                      >
                        {target.criticality}
                      </Badge>
                    )}
                    {target.linked_requirement_ids && target.linked_requirement_ids.length > 0 && (
                      <Badge variant="outline" className="text-xs">
                        {target.linked_requirement_ids.length} req{target.linked_requirement_ids.length !== 1 ? 's' : ''}
                      </Badge>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
