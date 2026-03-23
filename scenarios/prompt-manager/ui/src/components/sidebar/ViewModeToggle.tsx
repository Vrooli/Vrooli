/**
 * ViewModeToggle — Three icon buttons for switching between tree, list, and card views.
 */

import { TreePine, List, LayoutGrid } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ViewMode } from '@/types/filterSort'

interface ViewModeToggleProps {
  viewMode: ViewMode
  onViewModeChange: (mode: ViewMode) => void
}

const modes: { value: ViewMode; icon: typeof TreePine; label: string }[] = [
  { value: 'tree', icon: TreePine, label: 'Tree view' },
  { value: 'list', icon: List, label: 'List view' },
  { value: 'card', icon: LayoutGrid, label: 'Card view' },
]

export function ViewModeToggle({ viewMode, onViewModeChange }: ViewModeToggleProps) {
  return (
    <div className="flex items-center gap-0.5" role="radiogroup" aria-label="View mode">
      {modes.map(({ value, icon: Icon, label }) => (
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
  )
}
