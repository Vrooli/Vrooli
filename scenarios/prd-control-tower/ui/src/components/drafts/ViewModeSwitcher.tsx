import type { ViewMode } from '../../types'
import { ViewModes } from '../../types'
import { Button } from '../ui/button'

interface ViewModeSwitcherProps {
  viewMode: ViewMode
  onViewModeChange: (mode: ViewMode) => void
}

/**
 * View mode switcher component for draft editor.
 * Allows switching between Edit, Split, and Preview modes.
 */
export function ViewModeSwitcher({ viewMode, onViewModeChange }: ViewModeSwitcherProps) {
  const viewOptions: Array<{ label: string; mode: ViewMode }> = [
    { label: 'Edit', mode: ViewModes.EDIT },
    { label: 'Split', mode: ViewModes.SPLIT },
    { label: 'Preview', mode: ViewModes.PREVIEW },
  ]

  return (
    <div className="flex flex-wrap gap-2" role="group" aria-label="Editor view mode">
      {viewOptions.map((option) => (
        <Button key={option.mode} type="button" variant={viewMode === option.mode ? 'default' : 'outline'} onClick={() => onViewModeChange(option.mode)}>
          {option.label}
        </Button>
      ))}
    </div>
  )
}
