/**
 * ViewModeToggle — View mode (tree/list/card) + detail mode (compact/full) toggles.
 */

import { TreePine, List, LayoutGrid, AlignJustify, Rows3 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ViewMode, DetailMode } from '@/types/filterSort'

interface ViewModeToggleProps {
  viewMode: ViewMode
  onViewModeChange: (mode: ViewMode) => void
  detailMode: DetailMode
  onDetailModeChange: (mode: DetailMode) => void
}

const viewModes: { value: ViewMode; icon: typeof TreePine; label: string }[] = [
  { value: 'tree', icon: TreePine, label: 'Tree view' },
  { value: 'list', icon: List, label: 'List view' },
  { value: 'card', icon: LayoutGrid, label: 'Card view' },
]

export function ViewModeToggle({ viewMode, onViewModeChange, detailMode, onDetailModeChange }: ViewModeToggleProps) {
  return (
    <div className="flex items-center gap-1.5">
      {/* View mode */}
      <div className="flex items-center gap-0.5" role="radiogroup" aria-label="View mode">
        {viewModes.map(({ value, icon: Icon, label }) => (
          <button
            key={value}
            type="button"
            role="radio"
            aria-checked={viewMode === value}
            aria-label={label}
            onClick={() => onViewModeChange(value)}
            className={cn(
              'p-1 rounded transition-colors',
              viewMode === value
                ? 'bg-primary/20 text-primary'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted'
            )}
          >
            <Icon className="h-3.5 w-3.5" />
          </button>
        ))}
      </div>

      {/* Separator */}
      <div className="w-px h-4 bg-border" />

      {/* Detail toggle */}
      <button
        type="button"
        onClick={() => onDetailModeChange(detailMode === 'compact' ? 'full' : 'compact')}
        aria-label={detailMode === 'compact' ? 'Show details' : 'Hide details'}
        className={cn(
          'p-1 rounded transition-colors',
          detailMode === 'full'
            ? 'bg-primary/20 text-primary'
            : 'text-muted-foreground hover:text-foreground hover:bg-muted'
        )}
        data-testid="detail-mode-toggle"
      >
        {detailMode === 'full' ? (
          <Rows3 className="h-3.5 w-3.5" />
        ) : (
          <AlignJustify className="h-3.5 w-3.5" />
        )}
      </button>
    </div>
  )
}
